package seed

import (
	"context"
	"fmt"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type ApplyOptions struct {
	IncludeInactive bool
}

type Report struct {
	TotalEntities      int
	ByEntityType       map[domain.EntityType]int
	ByLayerCode        map[string]int
	ProfileCounts      map[string]int
	SourceMappingCount int
	EdgeCounts         map[string]int
	Created            int
	Updated            int
	Unchanged          int
	Failed             int
	Skipped            int
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return Service{repository: repository}
}

func (s Service) Apply(ctx context.Context, manifest Manifest, options ApplyOptions) (Report, error) {
	report := newReport()
	hasLegacy, err := s.repository.HasActiveLegacySectors(ctx)
	if err != nil {
		return report, fmt.Errorf("check active legacy sectors: %w", err)
	}
	if hasLegacy {
		return report, fmt.Errorf("active legacy sector requires explicit sector convergence")
	}
	skippedEntities := map[string]struct{}{}

	for _, entity := range manifest.Entities {
		if shouldSkipEntity(entity, options) {
			skippedEntities[entity.Key] = struct{}{}
			report.Skipped++
			continue
		}
		report.TotalEntities++
		report.ByEntityType[entity.EntityType]++
		report.ByLayerCode[entity.LayerCode]++

		result, err := s.repository.UpsertEntity(ctx, entity)
		if err != nil {
			report.Failed++
			return report, fmt.Errorf("upsert entity %q: %w", entity.Key, err)
		}
		applyWriteResult(&report, result)
	}

	for _, entity := range manifest.Entities {
		if _, skipped := skippedEntities[entity.Key]; skipped {
			continue
		}
		profile := Profile{
			EntityKey:  entity.Key,
			EntityType: entity.EntityType,
			Data:       entity.Profile,
		}
		if err := s.applyProfile(ctx, profile, &report); err != nil {
			return report, err
		}
	}

	for _, profile := range manifest.Profiles {
		if _, skipped := skippedEntities[profile.EntityKey]; skipped {
			report.Skipped++
			continue
		}
		if err := s.applyProfile(ctx, profile, &report); err != nil {
			return report, err
		}
	}

	for _, mapping := range manifest.SectorSourceMappings {
		if _, skipped := skippedEntities[mapping.SectorEntityKey]; skipped {
			report.Skipped++
			continue
		}
		result, err := s.repository.UpsertSectorSourceMapping(ctx, mapping)
		if err != nil {
			report.Failed++
			return report, fmt.Errorf("upsert sector source mapping %q: %w", sectorSourceMappingIdentity(mapping), err)
		}
		report.SourceMappingCount++
		applyWriteResult(&report, result)
	}

	for _, relationship := range manifest.Relationships {
		if _, skipped := skippedEntities[relationship.From]; skipped {
			report.Skipped++
			continue
		}
		if _, skipped := skippedEntities[relationship.To]; skipped {
			report.Skipped++
			continue
		}
		result, err := s.repository.UpsertRelationship(ctx, relationship)
		if err != nil {
			report.Failed++
			return report, fmt.Errorf("upsert relationship %q: %w", relationship.Key, err)
		}
		report.EdgeCounts[relationship.RelationType]++
		applyWriteResult(&report, result)
	}

	return report, nil
}

func (s Service) applyProfile(ctx context.Context, profile Profile, report *Report) error {
	table, err := profileTableName(profile.EntityType)
	if err != nil {
		report.Failed++
		return err
	}
	result, err := s.repository.UpsertProfile(ctx, profile)
	if err != nil {
		report.Failed++
		return fmt.Errorf("upsert profile %q: %w", profile.EntityKey, err)
	}
	report.ProfileCounts[table]++
	applyWriteResult(report, result)
	return nil
}

func newReport() Report {
	return Report{
		ByEntityType:  map[domain.EntityType]int{},
		ByLayerCode:   map[string]int{},
		ProfileCounts: map[string]int{},
		EdgeCounts:    map[string]int{},
	}
}

func shouldSkipEntity(entity Entity, options ApplyOptions) bool {
	return entity.Status == domain.StatusInactive && !options.IncludeInactive
}

func applyWriteResult(report *Report, result WriteResult) {
	switch result.Action {
	case WriteCreated:
		report.Created++
	case WriteUpdated:
		report.Updated++
	case WriteUnchanged:
		report.Unchanged++
	}
}
