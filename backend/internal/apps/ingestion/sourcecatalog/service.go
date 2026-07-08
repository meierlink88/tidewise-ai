package sourcecatalog

import (
	"context"
	"fmt"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type SeedRepository interface {
	SeedSource(context.Context, domain.SourceCatalog) error
}

type Report struct {
	TotalSources       int
	ByOriginSystem     map[string]int
	ByProviderKey      map[string]int
	ByIngestChannel    map[string]int
	BySourceType       map[string]int
	BySourceGroup      map[string]int
	ByStatus           map[string]int
	UniqueURLsByOrigin map[string]int
	Notes              []string
	Seeded             int
	Failed             int
}

type Service struct {
	repository SeedRepository
}

func NewService(repository SeedRepository) Service {
	return Service{repository: repository}
}

func (s Service) Apply(ctx context.Context, manifest Manifest) (Report, error) {
	report := newReport()
	uniqueURLs := map[string]map[string]struct{}{}
	seenVibeTrading := false
	for _, source := range manifest.Sources {
		report.TotalSources++
		increment(report.ByOriginSystem, source.OriginSystem)
		increment(report.ByProviderKey, source.ProviderKey)
		increment(report.ByIngestChannel, source.IngestChannel)
		increment(report.BySourceType, source.SourceType)
		increment(report.BySourceGroup, source.SourceGroup)
		increment(report.ByStatus, string(source.Status))
		if source.SourceURL != "" {
			if _, ok := uniqueURLs[source.OriginSystem]; !ok {
				uniqueURLs[source.OriginSystem] = map[string]struct{}{}
			}
			uniqueURLs[source.OriginSystem][source.SourceURL] = struct{}{}
		}
		if source.OriginSystem == "Vibe-Trading" {
			seenVibeTrading = true
		}

		if err := s.repository.SeedSource(ctx, source.SourceCatalog()); err != nil {
			report.Failed++
			return report, fmt.Errorf("seed source %q: %w", source.ID, err)
		}
		report.Seeded++
	}
	for origin, urls := range uniqueURLs {
		report.UniqueURLsByOrigin[origin] = len(urls)
	}
	if seenVibeTrading {
		report.Notes = append(report.Notes, "Vibe-Trading excludes auto and SDK-only loader sources: tushare, akshare, baostock, futu, mootdx")
	}
	return report, nil
}

func newReport() Report {
	return Report{
		ByOriginSystem:     map[string]int{},
		ByProviderKey:      map[string]int{},
		ByIngestChannel:    map[string]int{},
		BySourceType:       map[string]int{},
		BySourceGroup:      map[string]int{},
		ByStatus:           map[string]int{},
		UniqueURLsByOrigin: map[string]int{},
	}
}

func increment(counts map[string]int, key string) {
	if key == "" {
		key = "unknown"
	}
	counts[key]++
}
