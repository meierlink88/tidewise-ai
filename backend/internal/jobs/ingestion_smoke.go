package jobs

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/internal/ingestion"
	"github.com/meierlink88/tidewise-ai/backend/internal/integrations"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

const defaultSmokeSourceURL = "https://feeds.bbci.co.uk/news/business/rss.xml"

type IngestionSmokeRepository interface {
	repositories.SourceCatalogRepository
	repositories.RawDocumentRepository
	SeedSource(context.Context, domain.SourceCatalog) error
	RawDocumentCount(context.Context, string) (int, error)
}

type IngestionSmokeOptions struct {
	SourceURL    string
	SourceName   string
	MaxDocuments int
	Timeout      time.Duration
}

type IngestionSmokeReport struct {
	Sources          int      `json:"sources"`
	SucceededSources int      `json:"succeeded_sources"`
	FailedSources    int      `json:"failed_sources"`
	Created          int      `json:"created"`
	Duplicates       int      `json:"duplicates"`
	RawDocumentCount int      `json:"raw_document_count"`
	Errors           []string `json:"errors,omitempty"`
}

type IngestionSmokeRunner struct {
	repository IngestionSmokeRepository
	client     *http.Client
}

func NewIngestionSmokeRunner(repository IngestionSmokeRepository, client *http.Client) IngestionSmokeRunner {
	if client == nil {
		client = http.DefaultClient
	}
	return IngestionSmokeRunner{
		repository: repository,
		client:     client,
	}
}

func (r IngestionSmokeRunner) Run(ctx context.Context, options IngestionSmokeOptions) (IngestionSmokeReport, error) {
	options = normalizeSmokeOptions(options)
	ctx, cancel := context.WithTimeout(ctx, options.Timeout)
	defer cancel()

	source := smokeSource(options)
	if err := r.repository.SeedSource(ctx, source); err != nil {
		return IngestionSmokeReport{}, err
	}

	registry := ingestion.NewRegistry()
	registry.RegisterConnector("rss_feed", integrations.RSSFeedConnector{Client: r.client})
	registry.RegisterParser("rss_item", integrations.RSSItemParser{})

	writer := &countingSmokeWriter{
		writer:       ingestion.NewRawDocumentWriter(r.repository),
		maxDocuments: options.MaxDocuments,
	}
	job := NewIngestionJob(
		ingestion.NewSourceRegistry(r.repository),
		registry,
		ingestion.EnvCredentialResolver{},
		ingestion.NewRateLimiter(),
		writer,
	)

	jobReport := job.Run(ctx, repositories.SourceCatalogFilter{
		ProviderKey:   source.ProviderKey,
		IngestChannel: source.IngestChannel,
	})

	count, err := r.repository.RawDocumentCount(ctx, source.ID)
	if err != nil {
		return IngestionSmokeReport{}, err
	}

	report := IngestionSmokeReport{
		Sources:          jobReport.Total,
		SucceededSources: jobReport.Succeeded,
		FailedSources:    jobReport.Failed,
		Created:          writer.created,
		Duplicates:       writer.duplicates,
		RawDocumentCount: count,
		Errors:           jobReport.Errors,
	}
	if jobReport.Failed > 0 {
		return report, fmt.Errorf("ingestion smoke failed: %v", jobReport.Errors)
	}

	return report, nil
}

type countingSmokeWriter struct {
	writer       ingestion.RawDocumentWriter
	maxDocuments int
	seen         int
	created      int
	duplicates   int
}

func (w *countingSmokeWriter) Write(ctx context.Context, candidate ingestion.RawDocumentCandidate) (repositories.RawDocumentWriteResult, error) {
	if w.maxDocuments > 0 && w.seen >= w.maxDocuments {
		return repositories.RawDocumentWriteResult{}, nil
	}
	result, err := w.writer.Write(ctx, candidate)
	if err != nil {
		return repositories.RawDocumentWriteResult{}, err
	}
	w.seen++
	if result.Created {
		w.created++
	} else {
		w.duplicates++
	}
	return result, nil
}

func normalizeSmokeOptions(options IngestionSmokeOptions) IngestionSmokeOptions {
	if options.SourceURL == "" {
		options.SourceURL = defaultSmokeSourceURL
	}
	if options.SourceName == "" {
		options.SourceName = "Tidewise RSS smoke source"
	}
	if options.MaxDocuments <= 0 {
		options.MaxDocuments = 3
	}
	if options.Timeout <= 0 {
		options.Timeout = 10 * time.Second
	}
	return options
}

func smokeSource(options IngestionSmokeOptions) domain.SourceCatalog {
	return domain.SourceCatalog{
		ID:              "tidewise-local-rss-smoke-source",
		IngestChannel:   "rss_feed",
		ProviderKey:     "rss_smoke",
		ConnectorKey:    "rss_feed",
		ParserKey:       "rss_item",
		SourceType:      "news",
		SourceName:      options.SourceName,
		SourceURL:       options.SourceURL,
		SourceLevel:     "secondary",
		AuthType:        "none",
		RateLimitPolicy: map[string]any{},
		UsagePolicy:     "local smoke verification only",
		Status:          domain.SourceCatalogStatusActive,
	}
}
