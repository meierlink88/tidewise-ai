package integrations

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/internal/ingestion"
)

type RSSFeedConnector struct {
	Client *http.Client
}

func (c RSSFeedConnector) Fetch(ctx context.Context, source domain.SourceCatalog, _ ingestion.Credential) (ingestion.RawResponse, error) {
	return fetchURL(ctx, c.Client, source.SourceURL, "")
}

type EastmoneyConnector struct {
	Client *http.Client
}

func (c EastmoneyConnector) Fetch(ctx context.Context, source domain.SourceCatalog, _ ingestion.Credential) (ingestion.RawResponse, error) {
	return fetchURL(ctx, c.Client, source.SourceURL, "Mozilla/5.0 TidewiseBot/1.0")
}

type RSSHubConnector struct {
	Client  *http.Client
	BaseURL string
}

func (c RSSHubConnector) Fetch(ctx context.Context, source domain.SourceCatalog, _ ingestion.Credential) (ingestion.RawResponse, error) {
	base := strings.TrimRight(c.BaseURL, "/")
	route := "/" + strings.TrimLeft(source.RouteTemplate, "/")
	target := base + route
	if _, err := url.ParseRequestURI(target); err != nil {
		return ingestion.RawResponse{}, fmt.Errorf("invalid rsshub url: %w", err)
	}
	return fetchURL(ctx, c.Client, target, "")
}

type WebFetchConnector struct {
	Client *http.Client
}

func (c WebFetchConnector) Fetch(ctx context.Context, source domain.SourceCatalog, _ ingestion.Credential) (ingestion.RawResponse, error) {
	return fetchURL(ctx, c.Client, source.SourceURL, "Mozilla/5.0 TidewiseBot/1.0")
}

type LocalFileConnector struct{}

func (LocalFileConnector) Fetch(_ context.Context, source domain.SourceCatalog, _ ingestion.Credential) (ingestion.RawResponse, error) {
	content, err := os.ReadFile(source.SourceURL)
	if err != nil {
		return ingestion.RawResponse{}, fmt.Errorf("read local file: %w", err)
	}
	return ingestion.RawResponse{
		ContentType: "text/plain",
		Content:     content,
		CollectedAt: time.Now(),
	}, nil
}

type SDKStubConnector struct {
	Name string
}

func (c SDKStubConnector) Fetch(context.Context, domain.SourceCatalog, ingestion.Credential) (ingestion.RawResponse, error) {
	return ingestion.RawResponse{}, fmt.Errorf("%s requires a python worker, sidecar, or internal http wrapper", c.Name)
}

type RSSItemParser struct{}

func (RSSItemParser) Parse(_ context.Context, source domain.SourceCatalog, response ingestion.RawResponse) ([]ingestion.RawDocumentCandidate, error) {
	var feed rssFeed
	if err := xml.Unmarshal(response.Content, &feed); err != nil {
		return nil, fmt.Errorf("parse rss feed: %w", err)
	}

	items := feed.Channel.Items
	if len(items) == 0 {
		items = feed.Items
	}

	docs := make([]ingestion.RawDocumentCandidate, 0, len(items))
	for _, item := range items {
		publishedAt := parseTime(item.PubDate)
		title := strings.TrimSpace(item.Title)
		content := strings.TrimSpace(firstNonEmpty(item.Description, item.Summary, item.Content))
		externalID := strings.TrimSpace(firstNonEmpty(item.GUID, item.Link, title))
		docs = append(docs, candidateFromParts(source, title, content, externalID, item.Link, response.ContentType, publishedAt))
	}
	return docs, nil
}

type EastmoneyJSONParser struct{}

func (EastmoneyJSONParser) Parse(_ context.Context, source domain.SourceCatalog, response ingestion.RawResponse) ([]ingestion.RawDocumentCandidate, error) {
	var payload eastmoneyPayload
	if err := json.Unmarshal(response.Content, &payload); err != nil {
		return nil, fmt.Errorf("parse eastmoney json: %w", err)
	}

	docs := make([]ingestion.RawDocumentCandidate, 0, len(payload.Data))
	for _, item := range payload.Data {
		publishedAt := parseTime(item.PublishTime)
		title := strings.TrimSpace(item.Title)
		content := strings.TrimSpace(item.Content)
		externalID := strings.TrimSpace(firstNonEmpty(item.ID, item.URL, title))
		docs = append(docs, candidateFromParts(source, title, content, externalID, item.URL, response.ContentType, publishedAt))
	}
	return docs, nil
}

type TextParser struct{}

func (TextParser) Parse(_ context.Context, source domain.SourceCatalog, response ingestion.RawResponse) ([]ingestion.RawDocumentCandidate, error) {
	text := readableText(response.Content)
	title, body := splitTitleAndBody(text)
	return []ingestion.RawDocumentCandidate{
		candidateFromParts(source, title, body, source.SourceURL, source.SourceURL, response.ContentType, nil),
	}, nil
}

func fetchURL(ctx context.Context, client *http.Client, target string, userAgent string) (ingestion.RawResponse, error) {
	if client == nil {
		client = http.DefaultClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return ingestion.RawResponse{}, fmt.Errorf("build request: %w", err)
	}
	if userAgent != "" {
		request.Header.Set("User-Agent", userAgent)
	}

	response, err := client.Do(request)
	if err != nil {
		return ingestion.RawResponse{}, fmt.Errorf("fetch url: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ingestion.RawResponse{}, fmt.Errorf("fetch url status %d", response.StatusCode)
	}
	content, err := io.ReadAll(response.Body)
	if err != nil {
		return ingestion.RawResponse{}, fmt.Errorf("read response body: %w", err)
	}

	return ingestion.RawResponse{
		ContentType: response.Header.Get("Content-Type"),
		Content:     content,
		CollectedAt: time.Now(),
	}, nil
}

func candidateFromParts(source domain.SourceCatalog, title string, content string, externalID string, sourceURL string, mimeType string, publishedAt *time.Time) ingestion.RawDocumentCandidate {
	return ingestion.RawDocumentCandidate{
		ID:               "raw-" + ingestionID(source.ID, externalID, title),
		SourceID:         source.ID,
		IngestChannel:    source.IngestChannel,
		SourceType:       source.SourceType,
		SourceName:       source.SourceName,
		SourceURL:        firstNonEmpty(sourceURL, source.SourceURL),
		SourceExternalID: externalID,
		Title:            title,
		ContentText:      content,
		RawMIMEType:      mimeType,
		PublishedAt:      publishedAt,
		CollectedAt:      time.Now(),
		IngestStatus:     domain.IngestStatusCollected,
	}
}

func ingestionID(parts ...string) string {
	joined := strings.Join(parts, "\x00")
	if joined == "" {
		joined = time.Now().UTC().Format(time.RFC3339Nano)
	}
	sum := 0
	for _, r := range joined {
		sum = (sum*31 + int(r)) % 1000000007
	}
	return fmt.Sprintf("%08x", sum)
}

func readableText(content []byte) string {
	text := string(content)
	replacer := strings.NewReplacer("<html>", " ", "</html>", " ", "<body>", " ", "</body>", " ", "<h1>", "\n", "</h1>", "\n", "<p>", "\n", "</p>", "\n")
	text = replacer.Replace(text)
	return strings.Join(strings.Fields(text), " ")
}

func splitTitleAndBody(text string) (string, string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "untitled", ""
	}
	lines := strings.SplitN(text, "\n", 2)
	if len(lines) == 2 {
		return strings.TrimSpace(lines[0]), strings.TrimSpace(lines[1])
	}
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return "untitled", ""
	}
	return parts[0], text
}

func parseTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, time.RFC1123Z, time.RFC1123} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return &parsed
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type rssFeed struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
	Items []rssItem `xml:"entry"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	Description string `xml:"description"`
	Summary     string `xml:"summary"`
	Content     string `xml:"content"`
	PubDate     string `xml:"pubDate"`
}

type eastmoneyPayload struct {
	Data []eastmoneyItem `json:"data"`
}

type eastmoneyItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Content     string `json:"content"`
	PublishTime string `json:"publish_time"`
}
