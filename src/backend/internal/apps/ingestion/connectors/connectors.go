package connectors

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/parsers"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	coreingestion "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/core"
)

type RSSFeedConnector struct {
	Client *http.Client
}

func RegisterContentConnectors(registry *coreingestion.Registry, client *http.Client, rsshubBaseURL string) {
	registry.RegisterConnector("rss_feed", RSSFeedConnector{Client: client})
	registry.RegisterConnector("rsshub_feed", RSSHubConnector{Client: client, BaseURL: rsshubBaseURL})
	registry.RegisterConnector("web_fetch", WebFetchConnector{Client: client})
	registry.RegisterConnector("local_file", LocalFileConnector{})
	registry.RegisterParser("rss_item", parsers.RSSItemParser{})
	registry.RegisterParser("text", parsers.TextParser{})
}

func (c RSSFeedConnector) Fetch(ctx context.Context, source domain.SourceCatalog, _ coreingestion.Credential) (coreingestion.RawResponse, error) {
	return fetchURL(ctx, c.Client, source.SourceURL, "")
}

type EastmoneyConnector struct {
	Client *http.Client
}

func (c EastmoneyConnector) Fetch(ctx context.Context, source domain.SourceCatalog, _ coreingestion.Credential) (coreingestion.RawResponse, error) {
	target, err := eastmoneyRequestURL(source)
	if err != nil {
		return coreingestion.RawResponse{}, err
	}
	return fetchURL(ctx, c.Client, target, "Mozilla/5.0 TidewiseBot/1.0")
}

type RSSHubConnector struct {
	Client  *http.Client
	BaseURL string
}

func (c RSSHubConnector) Fetch(ctx context.Context, source domain.SourceCatalog, _ coreingestion.Credential) (coreingestion.RawResponse, error) {
	base := strings.TrimRight(c.BaseURL, "/")
	route := "/" + strings.TrimLeft(source.RouteTemplate, "/")
	target := base + route
	if _, err := url.ParseRequestURI(target); err != nil {
		return coreingestion.RawResponse{}, fmt.Errorf("invalid rsshub url: %w", err)
	}
	return fetchURL(ctx, c.Client, target, "")
}

type WebFetchConnector struct {
	Client *http.Client
}

func (c WebFetchConnector) Fetch(ctx context.Context, source domain.SourceCatalog, _ coreingestion.Credential) (coreingestion.RawResponse, error) {
	return fetchURL(ctx, c.Client, source.SourceURL, "Mozilla/5.0 TidewiseBot/1.0")
}

type LocalFileConnector struct{}

func (LocalFileConnector) Fetch(_ context.Context, source domain.SourceCatalog, _ coreingestion.Credential) (coreingestion.RawResponse, error) {
	content, err := os.ReadFile(source.SourceURL)
	if err != nil {
		return coreingestion.RawResponse{}, fmt.Errorf("read local file: %w", err)
	}
	return coreingestion.RawResponse{
		ContentType: "text/plain",
		Content:     content,
		CollectedAt: time.Now(),
	}, nil
}

type SDKStubConnector struct {
	Name string
}

func (c SDKStubConnector) Fetch(context.Context, domain.SourceCatalog, coreingestion.Credential) (coreingestion.RawResponse, error) {
	return coreingestion.RawResponse{}, fmt.Errorf("%s requires a python worker, sidecar, or internal http wrapper", c.Name)
}

func fetchURL(ctx context.Context, client *http.Client, target string, userAgent string) (coreingestion.RawResponse, error) {
	if client == nil {
		client = http.DefaultClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return coreingestion.RawResponse{}, fmt.Errorf("build request: %w", err)
	}
	if userAgent != "" {
		request.Header.Set("User-Agent", userAgent)
	}

	response, err := client.Do(request)
	if err != nil {
		return coreingestion.RawResponse{}, fmt.Errorf("fetch url: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return coreingestion.RawResponse{}, fmt.Errorf("fetch url status %d", response.StatusCode)
	}
	content, err := io.ReadAll(response.Body)
	if err != nil {
		return coreingestion.RawResponse{}, fmt.Errorf("read response body: %w", err)
	}

	return coreingestion.RawResponse{
		ContentType: response.Header.Get("Content-Type"),
		Content:     content,
		CollectedAt: time.Now(),
	}, nil
}

func eastmoneyRequestURL(source domain.SourceCatalog) (string, error) {
	parsed, err := url.Parse(source.SourceURL)
	if err != nil {
		return "", fmt.Errorf("build eastmoney url: %w", err)
	}
	query := parsed.Query()
	kind := jsonStringValue(source.SourceConfig["kind"])
	switch kind {
	case "eastmoney_stock_list", "eastmoney_concept_board_list":
		setQueryDefault(query, "pn", "1")
		setQueryDefault(query, "pz", firstNonEmpty(jsonStringValue(firstConfigValue(source.SourceConfig, "page_size", "limit")), "20"))
		setQueryDefault(query, "po", "1")
		setQueryDefault(query, "np", "1")
		setQueryDefault(query, "fltt", "2")
		setQueryDefault(query, "invt", "2")
		setQueryDefault(query, "fid", "f3")
		setQueryDefault(query, "fs", jsonStringValue(source.SourceConfig["fs"]))
		setQueryDefault(query, "fields", "f12,f14,f2,f3,f104")
	case "eastmoney_stock_kline", "eastmoney_index_kline", "eastmoney_concept_board_kline":
		secid := jsonStringValue(source.SourceConfig["secid"])
		if secid == "" {
			market := jsonStringValue(source.SourceConfig["market"])
			symbol := jsonStringValue(source.SourceConfig["symbol"])
			if market != "" && symbol != "" {
				secid = market + "." + symbol
			}
		}
		setQueryDefault(query, "secid", secid)
		setQueryDefault(query, "klt", jsonStringValue(source.SourceConfig["klt"]))
		setQueryDefault(query, "fqt", jsonStringValue(source.SourceConfig["fqt"]))
		setQueryDefault(query, "lmt", firstNonEmpty(jsonStringValue(firstConfigValue(source.SourceConfig, "limit", "lmt")), "5"))
		setQueryDefault(query, "end", firstNonEmpty(jsonStringValue(source.SourceConfig["end"]), "20500101"))
		setQueryDefault(query, "iscca", "1")
		setQueryDefault(query, "fields1", "f1,f2,f3,f4,f5,f6")
		setQueryDefault(query, "fields2", "f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61")
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func setQueryDefault(query url.Values, key string, value string) {
	if query.Get(key) != "" || strings.TrimSpace(value) == "" {
		return
	}
	query.Set(key, strings.TrimSpace(value))
}

func firstConfigValue(config map[string]any, keys ...string) any {
	for _, key := range keys {
		value := config[key]
		if jsonStringValue(value) != "" {
			return value
		}
	}
	return nil
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
