package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/internal/ingestion"
)

func TestRSSFeedConnectorAndParser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<rss><channel><item><title>政策事件</title><link>https://example.com/a</link><description>事件摘要</description><pubDate>Wed, 08 Jul 2026 10:00:00 GMT</pubDate></item></channel></rss>`))
	}))
	defer server.Close()

	source := domain.SourceCatalog{
		ID:            "source-1",
		IngestChannel: "rss_feed",
		SourceType:    "news",
		SourceName:    "RSS 来源",
		SourceURL:     server.URL,
	}
	response, err := RSSFeedConnector{Client: server.Client()}.Fetch(context.Background(), source, ingestion.Credential{})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	docs, err := RSSItemParser{}.Parse(context.Background(), source, response)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("docs len = %d, want 1", len(docs))
	}
	if docs[0].Title != "政策事件" || docs[0].SourceExternalID != "https://example.com/a" {
		t.Fatalf("unexpected doc = %+v", docs[0])
	}
}

func TestEastmoneyConnectorAndParser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); !strings.Contains(got, "Mozilla") {
			t.Fatalf("User-Agent = %q, want browser-like UA", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{
				{
					"id":           "em-1",
					"title":        "东方财富事件",
					"url":          "https://eastmoney.example/a",
					"content":      "事件正文",
					"publish_time": "2026-07-08T10:00:00Z",
				},
			},
		})
	}))
	defer server.Close()

	source := domain.SourceCatalog{
		ID:            "source-1",
		IngestChannel: "http_eastmoney",
		SourceType:    "news",
		SourceName:    "Eastmoney",
		SourceURL:     server.URL,
	}
	response, err := EastmoneyConnector{Client: server.Client()}.Fetch(context.Background(), source, ingestion.Credential{})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	docs, err := EastmoneyJSONParser{}.Parse(context.Background(), source, response)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(docs) != 1 || docs[0].SourceExternalID != "em-1" {
		t.Fatalf("docs = %+v", docs)
	}
}

func TestRSSHubConnectorBuildsRouteFromBaseURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/finance/news" {
			t.Fatalf("path = %q, want /finance/news", r.URL.Path)
		}
		_, _ = w.Write([]byte(`<rss><channel><item><title>RSSHub 事件</title><link>https://example.com/rsshub</link></item></channel></rss>`))
	}))
	defer server.Close()

	source := domain.SourceCatalog{
		ID:            "source-1",
		IngestChannel: "rsshub_feed",
		SourceType:    "news",
		SourceName:    "RSSHub",
		RouteTemplate: "/finance/news",
	}
	response, err := RSSHubConnector{Client: server.Client(), BaseURL: server.URL}.Fetch(context.Background(), source, ingestion.Credential{})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(response.Content) == 0 {
		t.Fatal("response content is empty")
	}
}

func TestWebFetchConnectorAndTextParser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><h1>网页事件</h1><p>正文内容</p></body></html>`))
	}))
	defer server.Close()

	source := domain.SourceCatalog{
		ID:            "source-1",
		IngestChannel: "web_fetch",
		SourceType:    "web",
		SourceName:    "网页来源",
		SourceURL:     server.URL,
	}
	response, err := WebFetchConnector{Client: server.Client()}.Fetch(context.Background(), source, ingestion.Credential{})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	docs, err := TextParser{}.Parse(context.Background(), source, response)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(docs) != 1 || !strings.Contains(docs[0].ContentText, "网页事件") {
		t.Fatalf("docs = %+v", docs)
	}
}

func TestLocalFileConnectorAndParser(t *testing.T) {
	path := filepath.Join(t.TempDir(), "event.txt")
	if err := os.WriteFile(path, []byte("本地文件事件\n正文内容"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	source := domain.SourceCatalog{
		ID:            "source-1",
		IngestChannel: "local_file",
		SourceType:    "file",
		SourceName:    "本地文件",
		SourceURL:     path,
	}
	response, err := LocalFileConnector{}.Fetch(context.Background(), source, ingestion.Credential{})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	docs, err := TextParser{}.Parse(context.Background(), source, response)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(docs) != 1 || docs[0].Title != "本地文件事件" {
		t.Fatalf("docs = %+v", docs)
	}
}

func TestSDKStubConnectorReturnsBoundaryError(t *testing.T) {
	_, err := SDKStubConnector{Name: "sdk_tushare"}.Fetch(context.Background(), domain.SourceCatalog{}, ingestion.Credential{})
	if err == nil || !strings.Contains(err.Error(), "worker") {
		t.Fatalf("Fetch() error = %v, want worker boundary error", err)
	}
}
