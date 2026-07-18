package seed

import (
	"context"
	"fmt"
	"strings"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

type ApplyScope string

const (
	ApplyScopeAll ApplyScope = ""
)

type ApplyOptions struct {
	IncludeInactive bool
	Scope           ApplyScope
}

type WriteStats struct {
	Created   int `json:"created"`
	Updated   int `json:"updated"`
	Unchanged int `json:"unchanged"`
}

type Report struct {
	TotalEntities       int
	ByEntityType        map[domain.EntityType]int
	ByLayerCode         map[string]int
	ProfileCounts       map[string]int
	SourceMappingCount  int
	EdgeCounts          map[string]int
	IndustryChainCounts map[string]int
	Created             int
	Updated             int
	Unchanged           int
	Failed              int
	Skipped             int
	Scope               string
	OperationCounts     WriteStats
	FinalTableImpact    map[string]WriteStats
	finalTableActions   map[string]map[string]WriteAction
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return Service{repository: repository}
}

func (s Service) Apply(ctx context.Context, manifest Manifest, options ApplyOptions) (Report, error) {
	report := newReport()
	if err := ValidateProductionManifest(manifest); err != nil {
		return report, err
	}
	scope, err := ParseApplyScope(string(options.Scope))
	if err != nil {
		return report, err
	}
	report.Scope = string(scope)
	hasLegacy, err := s.repository.HasRetiredIndustryEntities(ctx)
	if err != nil {
		return report, fmt.Errorf("check retired industry entities: %w", err)
	}
	if hasLegacy {
		return report, fmt.Errorf("retired sector or industry_chain data requires approved Phase A cleanup")
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
		recordFinalTableAction(&report, "entity_nodes", entity.Key, result.Action)
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
		recordFinalTableAction(&report, "entity_edges", relationship.Key, result.Action)
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
	recordFinalTableAction(report, table, profile.EntityKey, result.Action)
	return nil
}

func newReport() Report {
	return Report{
		ByEntityType:        map[domain.EntityType]int{},
		ByLayerCode:         map[string]int{},
		ProfileCounts:       map[string]int{},
		EdgeCounts:          map[string]int{},
		IndustryChainCounts: map[string]int{},
		FinalTableImpact:    map[string]WriteStats{},
		finalTableActions:   map[string]map[string]WriteAction{},
	}
}

func ParseApplyScope(value string) (ApplyScope, error) {
	scope := ApplyScope(strings.TrimSpace(value))
	switch scope {
	case ApplyScopeAll:
		return scope, nil
	default:
		return "", fmt.Errorf("unsupported apply scope %q", value)
	}
}

func shouldSkipEntity(entity Entity, options ApplyOptions) bool {
	return entity.Status == domain.StatusInactive && !options.IncludeInactive
}

func applyWriteResult(report *Report, result WriteResult) {
	switch result.Action {
	case WriteCreated:
		report.Created++
		report.OperationCounts.Created++
	case WriteUpdated:
		report.Updated++
		report.OperationCounts.Updated++
	case WriteUnchanged:
		report.Unchanged++
		report.OperationCounts.Unchanged++
	}
}

func recordFinalTableAction(report *Report, table string, key string, action WriteAction) {
	if report.finalTableActions[table] == nil {
		report.finalTableActions[table] = map[string]WriteAction{}
	}
	previous, exists := report.finalTableActions[table][key]
	if exists && writeActionRank(previous) >= writeActionRank(action) {
		return
	}
	stats := report.FinalTableImpact[table]
	if exists {
		decrementWriteStats(&stats, previous)
	}
	incrementWriteStats(&stats, action)
	report.FinalTableImpact[table] = stats
	report.finalTableActions[table][key] = action
}

func writeActionRank(action WriteAction) int {
	switch action {
	case WriteCreated:
		return 3
	case WriteUpdated:
		return 2
	case WriteUnchanged:
		return 1
	default:
		return 0
	}
}

func incrementWriteStats(stats *WriteStats, action WriteAction) {
	switch action {
	case WriteCreated:
		stats.Created++
	case WriteUpdated:
		stats.Updated++
	case WriteUnchanged:
		stats.Unchanged++
	}
}

func decrementWriteStats(stats *WriteStats, action WriteAction) {
	switch action {
	case WriteCreated:
		stats.Created--
	case WriteUpdated:
		stats.Updated--
	case WriteUnchanged:
		stats.Unchanged--
	}
}
