package jobs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

func TestIngestionSmokeRunnerWritesLimitedRSSDocuments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Smoke Feed</title>
    <item>
      <title>First item</title>
      <link>https://example.com/first</link>
      <guid>first</guid>
      <description>First description</description>
      <pubDate>Wed, 08 Jul 2026 10:00:00 GMT</pubDate>
    </item>
    <item>
      <title>Second item</title>
      <link>https://example.com/second</link>
      <guid>second</guid>
      <description>Second description</description>
      <pubDate>Wed, 08 Jul 2026 10:01:00 GMT</pubDate>
    </item>
  </channel>
</rss>`))
	}))
	defer server.Close()

	repo := repositories.NewInMemoryRepository(nil)
	runner := NewIngestionSmokeRunner(repo, server.Client())

	report, err := runner.Run(context.Background(), IngestionSmokeOptions{
		SourceURL:    server.URL,
		SourceName:   "测试 RSS",
		MaxDocuments: 1,
		Timeout:      time.Second,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if report.Sources != 1 || report.SucceededSources != 1 {
		t.Fatalf("report sources = %+v, want one succeeded source", report)
	}
	if report.Created != 1 {
		t.Fatalf("Created = %d, want 1", report.Created)
	}
	if report.RawDocumentCount != 1 {
		t.Fatalf("RawDocumentCount = %d, want 1", report.RawDocumentCount)
	}
}

func TestIngestionSmokeRunnerReportsFetchFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	repo := repositories.NewInMemoryRepository(nil)
	runner := NewIngestionSmokeRunner(repo, server.Client())

	report, err := runner.Run(context.Background(), IngestionSmokeOptions{
		SourceURL:    server.URL,
		MaxDocuments: 1,
		Timeout:      time.Second,
	})
	if err == nil {
		t.Fatal("Run() error = nil, want fetch failure")
	}
	if report.FailedSources != 1 {
		t.Fatalf("FailedSources = %d, want 1", report.FailedSources)
	}
	if report.Created != 0 {
		t.Fatalf("Created = %d, want 0", report.Created)
	}
}
