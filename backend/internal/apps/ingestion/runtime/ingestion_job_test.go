package runtime

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	coreingestion "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/core"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

func TestRuntimeRunsSourcesWithFailureIsolation(t *testing.T) {
	sources := []domain.SourceCatalog{
		activeRuntimeSource("source-1", "fixture_connector", "source_aware_parser"),
		activeRuntimeSource("source-2", "missing_connector", "source_aware_parser"),
	}
	registry := coreingestion.NewRegistry()
	registry.RegisterConnector("fixture_connector", runtimeConnector{})
	registry.RegisterParser("source_aware_parser", runtimeParser{})
	repo := repositories.NewInMemoryRepository(sources)
	writer := coreingestion.NewRawDocumentWriter(repo)
	job := NewIngestionJobWithOptions(
		coreingestion.NewSourceRegistry(repo),
		registry,
		runtimeCredentialResolver{},
		&runtimeLimiter{},
		writer,
		IngestionJobOptions{Concurrency: 2},
	)

	report := job.Run(context.Background(), repositories.SourceCatalogFilter{})
	if report.Succeeded != 1 || report.Failed != 1 {
		t.Fatalf("report = %+v, want one success and one failure", report)
	}
	if len(report.Errors) != 1 || !strings.Contains(report.Errors[0], "missing_connector") {
		t.Fatalf("errors = %+v, want missing connector error", report.Errors)
	}
	if _, ok := repo.RawDocument("raw-source-1"); !ok {
		t.Fatal("successful source raw document was not written")
	}

	resultsBySource := map[string]SourceIngestionResult{}
	for _, result := range report.SourceResults {
		resultsBySource[result.SourceID] = result
	}
	if resultsBySource["source-1"].Status != SourceIngestionStatusSucceeded {
		t.Fatalf("source-1 status = %q, want succeeded", resultsBySource["source-1"].Status)
	}
	if resultsBySource["source-1"].DocumentsWritten != 1 {
		t.Fatalf("source-1 DocumentsWritten = %d, want 1", resultsBySource["source-1"].DocumentsWritten)
	}
	if resultsBySource["source-2"].Status != SourceIngestionStatusFailed {
		t.Fatalf("source-2 status = %q, want failed", resultsBySource["source-2"].Status)
	}
	if !strings.Contains(resultsBySource["source-2"].Error, "missing_connector") {
		t.Fatalf("source-2 error = %q, want missing connector", resultsBySource["source-2"].Error)
	}
}

func TestRuntimeUsesProviderLimiterForEachConcurrentSource(t *testing.T) {
	sources := []domain.SourceCatalog{
		activeRuntimeSource("source-1", "fixture_connector", "source_aware_parser"),
		activeRuntimeSource("source-2", "fixture_connector", "source_aware_parser"),
		activeRuntimeSource("source-3", "fixture_connector", "source_aware_parser"),
	}
	for i := range sources {
		sources[i].ProviderKey = "eastmoney"
		sources[i].RateLimitPolicy = map[string]any{"requests_per_minute": 30}
	}
	registry := coreingestion.NewRegistry()
	registry.RegisterConnector("fixture_connector", runtimeConnector{})
	registry.RegisterParser("source_aware_parser", runtimeParser{})
	repo := repositories.NewInMemoryRepository(sources)
	writer := coreingestion.NewRawDocumentWriter(repo)
	limiter := &runtimeLimiter{}
	job := NewIngestionJobWithOptions(
		coreingestion.NewSourceRegistry(repo),
		registry,
		runtimeCredentialResolver{},
		limiter,
		writer,
		IngestionJobOptions{Concurrency: 3},
	)

	report := job.Run(context.Background(), repositories.SourceCatalogFilter{})
	if report.Succeeded != 3 || report.Failed != 0 {
		t.Fatalf("report = %+v, want three successes", report)
	}
	calls := limiter.Calls()
	if len(calls) != 3 {
		t.Fatalf("limiter calls = %d, want 3", len(calls))
	}
	for _, call := range calls {
		if call.provider != "eastmoney" || call.policy.RequestsPerMinute != 30 {
			t.Fatalf("limiter call = %+v, want eastmoney/30", call)
		}
	}
}

func activeRuntimeSource(id string, connector string, parser string) domain.SourceCatalog {
	return domain.SourceCatalog{
		ID:            id,
		IngestChannel: "rss_feed",
		ProviderKey:   "rss",
		ConnectorKey:  connector,
		ParserKey:     parser,
		SourceType:    "news",
		SourceName:    id,
		Status:        domain.SourceCatalogStatusActive,
	}
}

type runtimeConnector struct{}

func (runtimeConnector) Fetch(context.Context, domain.SourceCatalog, coreingestion.Credential) (coreingestion.RawResponse, error) {
	return coreingestion.RawResponse{Content: []byte("ok"), CollectedAt: time.Now()}, nil
}

type runtimeParser struct{}

func (runtimeParser) Parse(_ context.Context, source domain.SourceCatalog, _ coreingestion.RawResponse) ([]coreingestion.RawDocumentCandidate, error) {
	return []coreingestion.RawDocumentCandidate{
		{
			ID:               "raw-" + source.ID,
			SourceID:         source.ID,
			SourceExternalID: "external-" + source.ID,
			IngestChannel:    source.IngestChannel,
			SourceType:       source.SourceType,
			SourceName:       source.SourceName,
			Title:            "runtime event " + source.ID,
			ContentText:      "runtime content " + source.ID,
			CollectedAt:      time.Now(),
			IngestStatus:     domain.IngestStatusCollected,
		},
	}, nil
}

type runtimeCredentialResolver struct{}

func (runtimeCredentialResolver) Resolve(string) (string, error) {
	return "", nil
}

type runtimeLimiterCall struct {
	provider string
	policy   coreingestion.RateLimitPolicy
}

type runtimeLimiter struct {
	mu    sync.Mutex
	calls []runtimeLimiterCall
}

func (l *runtimeLimiter) Allow(provider string, policy coreingestion.RateLimitPolicy) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = append(l.calls, runtimeLimiterCall{provider: provider, policy: policy})
	return nil
}

func (l *runtimeLimiter) Calls() []runtimeLimiterCall {
	l.mu.Lock()
	defer l.mu.Unlock()
	calls := make([]runtimeLimiterCall, len(l.calls))
	copy(calls, l.calls)
	return calls
}
