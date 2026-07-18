package connectors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	coreingestion "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/core"
)

func TestContentConnectorsRegisterAndParseFixtureSources(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "local-event.txt")
	if err := os.WriteFile(filePath, []byte("本地文件事件\n本地文件正文"), 0o600); err != nil {
		t.Fatalf("write local fixture: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rss.xml":
			w.Header().Set("Content-Type", "application/rss+xml")
			_, _ = w.Write([]byte(`<rss><channel><item><title>RSS 事件</title><link>https://example.com/rss-event</link><description>RSS 正文</description></item></channel></rss>`))
		case "/rsshub/finance":
			w.Header().Set("Content-Type", "application/rss+xml")
			_, _ = w.Write([]byte(`<rss><channel><item><title>RSSHub 事件</title><link>https://example.com/rsshub-event</link><description>RSSHub 正文</description></item></channel></rss>`))
		case "/article":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<html><body><h1>网页事件</h1><p>网页正文</p></body></html>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	registry := coreingestion.NewRegistry()
	RegisterContentConnectors(registry, server.Client(), server.URL)

	fixtures := []struct {
		name        string
		source      domain.SourceCatalog
		wantTitle   string
		wantContent string
	}{
		{
			name: "rss",
			source: domain.SourceCatalog{
				ID:            "content-rss",
				IngestChannel: "rss_feed",
				ConnectorKey:  "rss_feed",
				ParserKey:     "rss_item",
				SourceType:    "news",
				SourceName:    "RSS fixture",
				SourceURL:     server.URL + "/rss.xml",
			},
			wantTitle:   "RSS 事件",
			wantContent: "RSS 正文",
		},
		{
			name: "rsshub",
			source: domain.SourceCatalog{
				ID:            "content-rsshub",
				IngestChannel: "rsshub_feed",
				ConnectorKey:  "rsshub_feed",
				ParserKey:     "rss_item",
				SourceType:    "news",
				SourceName:    "RSSHub fixture",
				RouteTemplate: "/rsshub/finance",
			},
			wantTitle:   "RSSHub 事件",
			wantContent: "RSSHub 正文",
		},
		{
			name: "web",
			source: domain.SourceCatalog{
				ID:            "content-web",
				IngestChannel: "web_fetch",
				ConnectorKey:  "web_fetch",
				ParserKey:     "text",
				SourceType:    "news",
				SourceName:    "Web fixture",
				SourceURL:     server.URL + "/article",
			},
			wantTitle:   "网页事件",
			wantContent: "网页正文",
		},
		{
			name: "local-file",
			source: domain.SourceCatalog{
				ID:            "content-local",
				IngestChannel: "local_file",
				ConnectorKey:  "local_file",
				ParserKey:     "text",
				SourceType:    "file",
				SourceName:    "Local fixture",
				SourceURL:     filePath,
			},
			wantTitle:   "本地文件事件",
			wantContent: "本地文件正文",
		},
	}

	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			connector, err := registry.Connector(fixture.source.ConnectorKey)
			if err != nil {
				t.Fatalf("Connector() error = %v", err)
			}
			parser, err := registry.Parser(fixture.source.ParserKey)
			if err != nil {
				t.Fatalf("Parser() error = %v", err)
			}

			response, err := connector.Fetch(context.Background(), fixture.source, coreingestion.Credential{})
			if err != nil {
				t.Fatalf("Fetch() error = %v", err)
			}
			docs, err := parser.Parse(context.Background(), fixture.source, response)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if len(docs) != 1 {
				t.Fatalf("docs len = %d, want 1", len(docs))
			}
			if docs[0].Title != fixture.wantTitle {
				t.Fatalf("Title = %q, want %q", docs[0].Title, fixture.wantTitle)
			}
			if !strings.Contains(docs[0].ContentText, fixture.wantContent) {
				t.Fatalf("ContentText = %q, want containing %q", docs[0].ContentText, fixture.wantContent)
			}
		})
	}
}
