package parsers

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	coreingestion "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/core"
)

type RSSItemParser struct{}

func (RSSItemParser) Parse(_ context.Context, source domain.SourceCatalog, response coreingestion.RawResponse) ([]coreingestion.RawDocumentCandidate, error) {
	var feed rssFeed
	if err := xml.Unmarshal(response.Content, &feed); err != nil {
		return nil, fmt.Errorf("parse rss feed: %w", err)
	}

	items := feed.Channel.Items
	if len(items) == 0 {
		items = feed.Items
	}

	docs := make([]coreingestion.RawDocumentCandidate, 0, len(items))
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

func (EastmoneyJSONParser) Parse(_ context.Context, source domain.SourceCatalog, response coreingestion.RawResponse) ([]coreingestion.RawDocumentCandidate, error) {
	var envelope eastmoneyEnvelope
	if err := json.Unmarshal(response.Content, &envelope); err != nil {
		return nil, fmt.Errorf("parse eastmoney json: %w", err)
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil, nil
	}

	var items []eastmoneyItem
	if err := json.Unmarshal(envelope.Data, &items); err == nil {
		return eastmoneyNewsCandidates(source, response, items), nil
	}

	var payload eastmoneyDataPayload
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		return nil, fmt.Errorf("parse eastmoney json: %w", err)
	}
	if len(payload.Diff) > 0 {
		return eastmoneyListCandidates(source, response, payload.Diff), nil
	}
	if len(payload.Klines) > 0 {
		return eastmoneyKlineCandidates(source, response, payload), nil
	}
	return nil, nil
}

func eastmoneyNewsCandidates(source domain.SourceCatalog, response coreingestion.RawResponse, items []eastmoneyItem) []coreingestion.RawDocumentCandidate {
	docs := make([]coreingestion.RawDocumentCandidate, 0, len(items))
	for _, item := range items {
		publishedAt := parseTime(item.PublishTime)
		title := strings.TrimSpace(item.Title)
		content := strings.TrimSpace(item.Content)
		externalID := strings.TrimSpace(firstNonEmpty(item.ID, item.URL, title))
		docs = append(docs, candidateFromParts(source, title, content, externalID, item.URL, response.ContentType, publishedAt))
	}
	return docs
}

func eastmoneyListCandidates(source domain.SourceCatalog, response coreingestion.RawResponse, items []map[string]any) []coreingestion.RawDocumentCandidate {
	docs := make([]coreingestion.RawDocumentCandidate, 0, len(items))
	for _, item := range items {
		code := jsonStringValue(item["f12"])
		name := jsonStringValue(item["f14"])
		title := strings.TrimSpace(firstNonEmpty(strings.TrimSpace(name+" "+code), source.SourceName))
		content := compactJSON(item)
		docs = append(docs, candidateFromParts(source, title, content, code, source.SourceURL, response.ContentType, nil))
	}
	return docs
}

func eastmoneyKlineCandidates(source domain.SourceCatalog, response coreingestion.RawResponse, payload eastmoneyDataPayload) []coreingestion.RawDocumentCandidate {
	code := firstNonEmpty(payload.Code, jsonStringValue(source.SourceConfig["symbol"]), jsonStringValue(source.SourceConfig["board_code"]))
	name := firstNonEmpty(payload.Name, jsonStringValue(source.SourceConfig["board_name"]), source.SourceName)
	docs := make([]coreingestion.RawDocumentCandidate, 0, len(payload.Klines))
	for _, kline := range payload.Klines {
		date := strings.SplitN(kline, ",", 2)[0]
		externalID := firstNonEmpty(strings.TrimSpace(code+":"+date), date, kline)
		title := strings.TrimSpace(firstNonEmpty(strings.TrimSpace(name+" "+code+" K线"), source.SourceName))
		content := compactJSON(map[string]any{
			"code":   code,
			"name":   name,
			"kind":   jsonStringValue(source.SourceConfig["kind"]),
			"kline":  kline,
			"source": source.SourceName,
		})
		docs = append(docs, candidateFromParts(source, title, content, externalID, source.SourceURL, response.ContentType, parseTime(date)))
	}
	return docs
}

type TextParser struct{}

func (TextParser) Parse(_ context.Context, source domain.SourceCatalog, response coreingestion.RawResponse) ([]coreingestion.RawDocumentCandidate, error) {
	text := readableText(response.Content)
	title, body := splitTitleAndBody(text)
	return []coreingestion.RawDocumentCandidate{
		candidateFromParts(source, title, body, source.SourceURL, source.SourceURL, response.ContentType, nil),
	}, nil
}

func candidateFromParts(source domain.SourceCatalog, title string, content string, externalID string, sourceURL string, mimeType string, publishedAt *time.Time) coreingestion.RawDocumentCandidate {
	return coreingestion.RawDocumentCandidate{
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

func compactJSON(value any) string {
	content, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(content)
}

func jsonStringValue(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strings.TrimSpace(fmt.Sprintf("%g", v))
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(v))
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
	for _, layout := range []string{time.RFC3339, time.RFC1123Z, time.RFC1123, "2006-01-02"} {
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

type eastmoneyEnvelope struct {
	Data json.RawMessage `json:"data"`
}

type eastmoneyDataPayload struct {
	Diff   []map[string]any `json:"diff"`
	Code   string           `json:"code"`
	Name   string           `json:"name"`
	Klines []string         `json:"klines"`
}

type eastmoneyItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Content     string `json:"content"`
	PublishTime string `json:"publish_time"`
}
