package jobs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/internal/ingestion"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

func TestIngestionJobRunsActiveSources(t *testing.T) {
	source := domain.SourceCatalog{
		ID:            "source-1",
		IngestChannel: "rss_feed",
		ProviderKey:   "rss",
		ConnectorKey:  "fixture_connector",
		ParserKey:     "fixture_parser",
		SourceType:    "news",
		SourceName:    "fixture",
		Status:        domain.SourceCatalogStatusActive,
	}
	registry := ingestion.NewRegistry()
	registry.RegisterConnector("fixture_connector", fakeJobConnector{})
	registry.RegisterParser("fixture_parser", fakeJobParser{})
	repo := repositories.NewInMemoryRepository([]domain.SourceCatalog{source})
	writer := ingestion.NewRawDocumentWriter(repo)
	job := NewIngestionJob(ingestion.NewSourceRegistry(repo), registry, fakeCredentialResolver{}, fakeLimiter{}, writer)

	report := job.Run(context.Background(), repositories.SourceCatalogFilter{})
	if report.Succeeded != 1 || report.Failed != 0 {
		t.Fatalf("report = %+v, want one success", report)
	}

	if _, ok := repo.RawDocument("raw-1"); !ok {
		t.Fatal("raw document was not written")
	}
}

func TestIngestionJobContinuesAfterSourceFailure(t *testing.T) {
	sources := []domain.SourceCatalog{
		{
			ID:            "source-1",
			IngestChannel: "rss_feed",
			ProviderKey:   "rss",
			ConnectorKey:  "fixture_connector",
			ParserKey:     "fixture_parser",
			SourceType:    "news",
			SourceName:    "fixture",
			Status:        domain.SourceCatalogStatusActive,
		},
		{
			ID:            "source-2",
			IngestChannel: "rss_feed",
			ProviderKey:   "rss",
			ConnectorKey:  "missing_connector",
			ParserKey:     "fixture_parser",
			SourceType:    "news",
			SourceName:    "broken",
			Status:        domain.SourceCatalogStatusActive,
		},
	}
	registry := ingestion.NewRegistry()
	registry.RegisterConnector("fixture_connector", fakeJobConnector{})
	registry.RegisterParser("fixture_parser", fakeJobParser{})
	repo := repositories.NewInMemoryRepository(sources)
	writer := ingestion.NewRawDocumentWriter(repo)
	job := NewIngestionJob(ingestion.NewSourceRegistry(repo), registry, fakeCredentialResolver{}, fakeLimiter{}, writer)

	report := job.Run(context.Background(), repositories.SourceCatalogFilter{})
	if report.Succeeded != 1 || report.Failed != 1 {
		t.Fatalf("report = %+v, want one success and one failure", report)
	}
	if len(report.Errors) != 1 {
		t.Fatalf("errors len = %d, want 1", len(report.Errors))
	}
}

func TestIngestionJobReportContainsNoAnalysisConclusion(t *testing.T) {
	report := IngestionReport{}

	content, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	for _, forbidden := range []string{"score", "prediction", "impact", "recommendation", "投资建议"} {
		if contains := jsonContains(string(content), forbidden); contains {
			t.Fatalf("report must not contain analysis field %q", forbidden)
		}
	}
}

func TestSourceCatalogFixtureContainsNoSecrets(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("testdata", "source_catalogs.fixture.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	for _, forbidden := range []string{"api_key", "token", "cookie", "secret"} {
		if jsonContains(string(content), forbidden) {
			t.Fatalf("fixture must not contain %q", forbidden)
		}
	}
}

type fakeJobConnector struct{}

func (fakeJobConnector) Fetch(context.Context, domain.SourceCatalog, ingestion.Credential) (ingestion.RawResponse, error) {
	return ingestion.RawResponse{Content: []byte("ok"), CollectedAt: time.Now()}, nil
}

type fakeJobParser struct{}

func (fakeJobParser) Parse(_ context.Context, source domain.SourceCatalog, _ ingestion.RawResponse) ([]ingestion.RawDocumentCandidate, error) {
	return []ingestion.RawDocumentCandidate{
		{
			ID:            "raw-1",
			SourceID:      source.ID,
			IngestChannel: source.IngestChannel,
			SourceType:    source.SourceType,
			SourceName:    source.SourceName,
			Title:         "fixture event",
			ContentText:   "fixture content",
			CollectedAt:   time.Now(),
			IngestStatus:  domain.IngestStatusCollected,
		},
	}, nil
}

type fakeCredentialResolver struct{}

func (fakeCredentialResolver) Resolve(string) (string, error) {
	return "", nil
}

type fakeLimiter struct{}

func (fakeLimiter) Allow(string, ingestion.RateLimitPolicy) error {
	return nil
}

func jsonContains(content string, fragment string) bool {
	return len(content) >= len(fragment) && strings.Contains(content, fragment)
}
