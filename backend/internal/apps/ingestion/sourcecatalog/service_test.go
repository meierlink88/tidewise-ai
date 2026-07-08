package sourcecatalog

import (
	"context"
	"fmt"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestSourceCatalogServiceAppliesSeedsAndReportsStats(t *testing.T) {
	repository := newSourceCatalogRepository()
	service := NewService(repository)
	manifest := Manifest{
		Sources: []Source{
			sourceCatalogFixture("source-rss", "Vibe-Research", "rss", "rss_feed", "news", "content_events", domain.SourceCatalogStatusActive),
			sourceCatalogFixture("source-market", "Stock", "eastmoney", "eastmoney", "market", "market_data", domain.SourceCatalogStatusInactive),
		},
	}

	report, err := service.Apply(context.Background(), manifest)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if report.TotalSources != 2 || report.Seeded != 2 || report.Failed != 0 {
		t.Fatalf("report = %+v, want two seeded sources", report)
	}
	if report.ByProviderKey["eastmoney"] != 1 || report.BySourceGroup["content_events"] != 1 {
		t.Fatalf("report distributions are incomplete: %+v", report)
	}
	if repository.sources["source-market"].Status != domain.SourceCatalogStatusInactive {
		t.Fatalf("seeded status = %q, want inactive", repository.sources["source-market"].Status)
	}
}

func TestSourceCatalogServiceStopsOnRepositoryFailure(t *testing.T) {
	repository := newSourceCatalogRepository()
	repository.failOnID = "source-b"
	service := NewService(repository)
	manifest := Manifest{
		Sources: []Source{
			sourceCatalogFixture("source-a", "Tidewise", "rss", "rss_feed", "news", "content_events", domain.SourceCatalogStatusActive),
			sourceCatalogFixture("source-b", "Tidewise", "rss", "rss_feed", "news", "content_events", domain.SourceCatalogStatusActive),
			sourceCatalogFixture("source-c", "Tidewise", "rss", "rss_feed", "news", "content_events", domain.SourceCatalogStatusActive),
		},
	}

	report, err := service.Apply(context.Background(), manifest)
	if err == nil {
		t.Fatal("Apply() error = nil, want repository failure")
	}
	if report.Seeded != 1 || report.Failed != 1 {
		t.Fatalf("report = %+v, want one seeded and one failed", report)
	}
	if len(repository.calls) != 2 {
		t.Fatalf("repository calls = %v, want source-a and source-b", repository.calls)
	}
}

func sourceCatalogFixture(id string, origin string, provider string, channel string, sourceType string, group string, status domain.SourceCatalogStatus) Source {
	connector := channel
	parser := "provider_metadata"
	sourceURL := ""
	if channel == "rss_feed" {
		connector = "rss_feed"
		parser = "rss_item"
		sourceURL = "https://example.com/" + id + ".xml"
	}
	if channel == "eastmoney" {
		connector = "eastmoney"
		parser = "eastmoney_json"
		sourceURL = "https://example.com/" + id + ".json"
	}
	return Source{
		ID:            id,
		OriginSystem:  origin,
		Stage:         "test",
		IngestChannel: channel,
		ProviderKey:   provider,
		ConnectorKey:  connector,
		ParserKey:     parser,
		SourceType:    sourceType,
		SourceGroup:   group,
		SourceName:    id,
		SourceURL:     sourceURL,
		SourceLevel:   "secondary",
		TopicHint:     "test",
		AuthType:      "none",
		UsagePolicy:   "test",
		SourceConfig:  map[string]any{"kind": "test"},
		Status:        status,
	}
}

type sourceCatalogRepository struct {
	sources  map[string]domain.SourceCatalog
	calls    []string
	failOnID string
}

func newSourceCatalogRepository() *sourceCatalogRepository {
	return &sourceCatalogRepository{sources: map[string]domain.SourceCatalog{}}
}

func (r *sourceCatalogRepository) SeedSource(_ context.Context, source domain.SourceCatalog) error {
	r.calls = append(r.calls, source.ID)
	if source.ID == r.failOnID {
		return fmt.Errorf("forced failure for %s", source.ID)
	}
	r.sources[source.ID] = source
	return nil
}
