package runtime

import (
	"context"
	"sync"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/core"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

type SourceRegistry interface {
	ActiveSources(context.Context, repositories.SourceCatalogFilter) ([]domain.SourceCatalog, error)
}

type CredentialResolver interface {
	Resolve(string) (string, error)
}

type RateLimiter interface {
	Allow(string, core.RateLimitPolicy) error
}

type RawDocumentWriter interface {
	Write(context.Context, core.RawDocumentCandidate) (repositories.RawDocumentWriteResult, error)
}

type IngestionJob struct {
	sources     SourceRegistry
	registry    *core.Registry
	credentials CredentialResolver
	limiter     RateLimiter
	writer      RawDocumentWriter
	concurrency int
}

type IngestionJobOptions struct {
	Concurrency int
}

type IngestionReport struct {
	Total         int                     `json:"total"`
	Succeeded     int                     `json:"succeeded"`
	Failed        int                     `json:"failed"`
	Errors        []string                `json:"errors,omitempty"`
	SourceResults []SourceIngestionResult `json:"source_results,omitempty"`
}

type SourceIngestionStatus string

const (
	SourceIngestionStatusSucceeded SourceIngestionStatus = "succeeded"
	SourceIngestionStatusFailed    SourceIngestionStatus = "failed"
)

type SourceIngestionResult struct {
	SourceID           string                `json:"source_id"`
	Status             SourceIngestionStatus `json:"status"`
	DocumentsWritten   int                   `json:"documents_written"`
	DocumentsDuplicate int                   `json:"documents_duplicate"`
	Error              string                `json:"error,omitempty"`
	StartedAt          time.Time             `json:"started_at"`
	FinishedAt         time.Time             `json:"finished_at"`
	DurationMillis     int                   `json:"duration_millis"`
}

func NewIngestionJob(
	sources SourceRegistry,
	registry *core.Registry,
	credentials CredentialResolver,
	limiter RateLimiter,
	writer RawDocumentWriter,
) IngestionJob {
	return NewIngestionJobWithOptions(sources, registry, credentials, limiter, writer, IngestionJobOptions{Concurrency: 1})
}

func NewIngestionJobWithOptions(
	sources SourceRegistry,
	registry *core.Registry,
	credentials CredentialResolver,
	limiter RateLimiter,
	writer RawDocumentWriter,
	options IngestionJobOptions,
) IngestionJob {
	concurrency := options.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}
	return IngestionJob{
		sources:     sources,
		registry:    registry,
		credentials: credentials,
		limiter:     limiter,
		writer:      writer,
		concurrency: concurrency,
	}
}

func (j IngestionJob) Run(ctx context.Context, filter repositories.SourceCatalogFilter) IngestionReport {
	sources, err := j.sources.ActiveSources(ctx, filter)
	if err != nil {
		return IngestionReport{Failed: 1, Errors: []string{err.Error()}}
	}

	report := IngestionReport{Total: len(sources)}
	if j.concurrency <= 1 || len(sources) <= 1 {
		for _, source := range sources {
			result := j.runSource(ctx, source)
			report.SourceResults = append(report.SourceResults, result)
			if result.Status == SourceIngestionStatusFailed {
				report.Failed++
				report.Errors = append(report.Errors, result.Error)
				continue
			}
			report.Succeeded++
		}
		return report
	}

	workerCount := j.concurrency
	if workerCount > len(sources) {
		workerCount = len(sources)
	}
	sourceJobs := make(chan domain.SourceCatalog)
	results := make(chan SourceIngestionResult, len(sources))
	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for source := range sourceJobs {
				results <- j.runSource(ctx, source)
			}
		}()
	}
	for _, source := range sources {
		sourceJobs <- source
	}
	close(sourceJobs)
	wg.Wait()
	close(results)

	for result := range results {
		report.SourceResults = append(report.SourceResults, result)
		if result.Status == SourceIngestionStatusFailed {
			report.Failed++
			report.Errors = append(report.Errors, result.Error)
			continue
		}
		report.Succeeded++
	}

	return report
}

func (j IngestionJob) runSource(ctx context.Context, source domain.SourceCatalog) (result SourceIngestionResult) {
	result = SourceIngestionResult{
		SourceID:  source.ID,
		Status:    SourceIngestionStatusSucceeded,
		StartedAt: time.Now(),
	}
	defer func() {
		result.FinishedAt = time.Now()
		result.DurationMillis = int(result.FinishedAt.Sub(result.StartedAt).Milliseconds())
	}()

	credentialValue, err := j.credentials.Resolve(source.CredentialRef)
	if err != nil {
		return result.failed(err)
	}
	if err := j.limiter.Allow(source.ProviderKey, policyFromSource(source)); err != nil {
		return result.failed(err)
	}

	connector, err := j.registry.Connector(source.ConnectorKey)
	if err != nil {
		return result.failed(err)
	}
	parser, err := j.registry.Parser(source.ParserKey)
	if err != nil {
		return result.failed(err)
	}

	response, err := connector.Fetch(ctx, source, core.Credential{Value: credentialValue})
	if err != nil {
		return result.failed(err)
	}
	candidates, err := parser.Parse(ctx, source, response)
	if err != nil {
		return result.failed(err)
	}
	for _, candidate := range candidates {
		writeResult, err := j.writer.Write(ctx, candidate)
		if err != nil {
			return result.failed(err)
		}
		if writeResult.Created {
			result.DocumentsWritten++
		} else {
			result.DocumentsDuplicate++
		}
	}

	return result
}

func (r SourceIngestionResult) failed(err error) SourceIngestionResult {
	r.Status = SourceIngestionStatusFailed
	r.Error = err.Error()
	return r
}

func policyFromSource(source domain.SourceCatalog) core.RateLimitPolicy {
	value, ok := source.RateLimitPolicy["requests_per_minute"]
	if !ok {
		return core.RateLimitPolicy{}
	}
	switch typed := value.(type) {
	case int:
		return core.RateLimitPolicy{RequestsPerMinute: typed}
	case float64:
		return core.RateLimitPolicy{RequestsPerMinute: int(typed)}
	default:
		return core.RateLimitPolicy{}
	}
}
