package runtime

import (
	"context"
	"sync"

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
	Total     int      `json:"total"`
	Succeeded int      `json:"succeeded"`
	Failed    int      `json:"failed"`
	Errors    []string `json:"errors,omitempty"`
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
			if err := j.runSource(ctx, source); err != nil {
				report.Failed++
				report.Errors = append(report.Errors, err.Error())
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
	results := make(chan error, len(sources))
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

	for err := range results {
		if err != nil {
			report.Failed++
			report.Errors = append(report.Errors, err.Error())
			continue
		}
		report.Succeeded++
	}

	return report
}

func (j IngestionJob) runSource(ctx context.Context, source domain.SourceCatalog) error {
	credentialValue, err := j.credentials.Resolve(source.CredentialRef)
	if err != nil {
		return err
	}
	if err := j.limiter.Allow(source.ProviderKey, policyFromSource(source)); err != nil {
		return err
	}

	connector, err := j.registry.Connector(source.ConnectorKey)
	if err != nil {
		return err
	}
	parser, err := j.registry.Parser(source.ParserKey)
	if err != nil {
		return err
	}

	response, err := connector.Fetch(ctx, source, core.Credential{Value: credentialValue})
	if err != nil {
		return err
	}
	candidates, err := parser.Parse(ctx, source, response)
	if err != nil {
		return err
	}
	for _, candidate := range candidates {
		if _, err := j.writer.Write(ctx, candidate); err != nil {
			return err
		}
	}

	return nil
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
