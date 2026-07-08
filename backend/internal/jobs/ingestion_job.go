package jobs

import (
	"context"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/internal/ingestion"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

type SourceRegistry interface {
	ActiveSources(context.Context, repositories.SourceCatalogFilter) ([]domain.SourceCatalog, error)
}

type CredentialResolver interface {
	Resolve(string) (string, error)
}

type RateLimiter interface {
	Allow(string, ingestion.RateLimitPolicy) error
}

type RawDocumentWriter interface {
	Write(context.Context, ingestion.RawDocumentCandidate) (repositories.RawDocumentWriteResult, error)
}

type IngestionJob struct {
	sources     SourceRegistry
	registry    *ingestion.Registry
	credentials CredentialResolver
	limiter     RateLimiter
	writer      RawDocumentWriter
}

type IngestionReport struct {
	Total     int      `json:"total"`
	Succeeded int      `json:"succeeded"`
	Failed    int      `json:"failed"`
	Errors    []string `json:"errors,omitempty"`
}

func NewIngestionJob(
	sources SourceRegistry,
	registry *ingestion.Registry,
	credentials CredentialResolver,
	limiter RateLimiter,
	writer RawDocumentWriter,
) IngestionJob {
	return IngestionJob{
		sources:     sources,
		registry:    registry,
		credentials: credentials,
		limiter:     limiter,
		writer:      writer,
	}
}

func (j IngestionJob) Run(ctx context.Context, filter repositories.SourceCatalogFilter) IngestionReport {
	sources, err := j.sources.ActiveSources(ctx, filter)
	if err != nil {
		return IngestionReport{Failed: 1, Errors: []string{err.Error()}}
	}

	report := IngestionReport{Total: len(sources)}
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

	response, err := connector.Fetch(ctx, source, ingestion.Credential{Value: credentialValue})
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

func policyFromSource(source domain.SourceCatalog) ingestion.RateLimitPolicy {
	value, ok := source.RateLimitPolicy["requests_per_minute"]
	if !ok {
		return ingestion.RateLimitPolicy{}
	}
	switch typed := value.(type) {
	case int:
		return ingestion.RateLimitPolicy{RequestsPerMinute: typed}
	case float64:
		return ingestion.RateLimitPolicy{RequestsPerMinute: int(typed)}
	default:
		return ingestion.RateLimitPolicy{}
	}
}
