package ingestion

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

func TestRegistryReturnsRegisteredConnectorAndParser(t *testing.T) {
	registry := NewRegistry()
	connector := fakeConnector{}
	parser := fakeParser{}

	registry.RegisterConnector("rss_feed", connector)
	registry.RegisterParser("rss_item", parser)

	if _, err := registry.Connector("rss_feed"); err != nil {
		t.Fatalf("Connector() error = %v", err)
	}
	if _, err := registry.Parser("rss_item"); err != nil {
		t.Fatalf("Parser() error = %v", err)
	}
	if _, err := registry.Connector("missing"); err == nil {
		t.Fatal("Connector() error = nil, want missing connector error")
	}
	if _, err := registry.Parser("missing"); err == nil {
		t.Fatal("Parser() error = nil, want missing parser error")
	}
}

func TestSourceRegistrySelectsActiveSources(t *testing.T) {
	repo := repositories.NewInMemoryRepository([]domain.SourceCatalog{
		{
			ID:            "source-1",
			ProviderKey:   "rss",
			IngestChannel: "rss_feed",
			Status:        domain.SourceCatalogStatusActive,
		},
		{
			ID:            "source-2",
			ProviderKey:   "rss",
			IngestChannel: "rss_feed",
			Status:        domain.SourceCatalogStatusDisabled,
		},
	})
	registry := NewSourceRegistry(repo)

	sources, err := registry.ActiveSources(context.Background(), repositories.SourceCatalogFilter{
		ProviderKey:   "rss",
		IngestChannel: "rss_feed",
	})
	if err != nil {
		t.Fatalf("ActiveSources() error = %v", err)
	}

	if len(sources) != 1 || sources[0].ID != "source-1" {
		t.Fatalf("sources = %+v, want source-1 only", sources)
	}
}

func TestEnvCredentialResolverReadsEnvironmentReferences(t *testing.T) {
	t.Setenv("TIDEWISE_TEST_TOKEN", "secret-value")

	resolver := EnvCredentialResolver{}
	value, err := resolver.Resolve("env:TIDEWISE_TEST_TOKEN")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if value != "secret-value" {
		t.Fatalf("resolved value = %q", value)
	}

	if _, err := resolver.Resolve("env:MISSING_TIDEWISE_TEST_TOKEN"); err == nil {
		t.Fatal("Resolve() error = nil, want missing env error")
	}
}

func TestRateLimiterLimitsCallsPerProvider(t *testing.T) {
	limiter := NewRateLimiter()
	policy := RateLimitPolicy{RequestsPerMinute: 60}

	if err := limiter.Allow("eastmoney", policy); err != nil {
		t.Fatalf("first Allow() error = %v", err)
	}
	if err := limiter.Allow("eastmoney", policy); err == nil {
		t.Fatal("second Allow() error = nil, want rate limited")
	}
	if err := limiter.Allow("rss", policy); err != nil {
		t.Fatalf("different provider Allow() error = %v", err)
	}
}

func TestLocalRawObjectStoreSavesContent(t *testing.T) {
	store := LocalRawObjectStore{Root: t.TempDir()}

	uri, err := store.Save(context.Background(), RawObject{
		Name:        "sample.xml",
		ContentType: "application/xml",
		Content:     []byte("<rss></rss>"),
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if !strings.HasPrefix(uri, "file://") {
		t.Fatalf("uri = %q, want file:// prefix", uri)
	}

	path := strings.TrimPrefix(uri, "file://")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("saved raw object missing: %v", err)
	}
}

func TestRawDocumentWriterComputesHashAndUsesRepository(t *testing.T) {
	repo := repositories.NewInMemoryRepository(nil)
	writer := NewRawDocumentWriter(repo)

	result, err := writer.Write(context.Background(), RawDocumentCandidate{
		ID:            "raw-1",
		SourceID:      "source-1",
		IngestChannel: "rss_feed",
		SourceType:    "news",
		SourceName:    "示例来源",
		Title:         "  示例标题  ",
		ContentText:   "示例正文",
		CollectedAt:   time.Now(),
		IngestStatus:  domain.IngestStatusCollected,
	})
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if !result.Created {
		t.Fatal("Write() should create first document")
	}
	if result.Document.ContentHash == "" {
		t.Fatal("ContentHash is empty")
	}
	if result.Document.Title != "示例标题" {
		t.Fatalf("Title = %q, want trimmed title", result.Document.Title)
	}
}

type fakeConnector struct{}

func (fakeConnector) Fetch(context.Context, domain.SourceCatalog, Credential) (RawResponse, error) {
	return RawResponse{}, nil
}

type fakeParser struct{}

func (fakeParser) Parse(context.Context, domain.SourceCatalog, RawResponse) ([]RawDocumentCandidate, error) {
	return nil, nil
}
