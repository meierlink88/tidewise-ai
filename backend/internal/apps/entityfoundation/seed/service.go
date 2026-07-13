package seed

import (
	"context"
	"fmt"
	"strings"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type ApplyScope string

const (
	ApplyScopeAll                 ApplyScope = ""
	ApplyScopeIndustryChainMaster ApplyScope = "industry-chain-master"
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
	scope, err := ParseApplyScope(string(options.Scope))
	if err != nil {
		return report, err
	}
	report.Scope = string(scope)
	manifest, err = applyManifestScope(manifest, scope)
	if err != nil {
		return report, err
	}
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
		recordFinalTableAction(&report, "sector_source_mappings", sectorSourceMappingIdentity(normalizeSectorSourceMapping(mapping)), result.Action)
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

	if len(manifest.IndustryChainMemberships) > 0 || len(manifest.IndustryChainTopologyEdges) > 0 || len(manifest.IndustryChainPhysicalConstraints) > 0 {
		repository, ok := s.repository.(interface {
			UpsertIndustryChainBatch(context.Context, IndustryChainBatch) (IndustryChainWriteReport, error)
		})
		if !ok {
			report.Failed++
			return report, fmt.Errorf("repository does not support industry chain batch")
		}
		batch := manifestIndustryChainBatch(manifest)
		result, err := repository.UpsertIndustryChainBatch(ctx, batch)
		if err != nil {
			report.Failed++
			return report, fmt.Errorf("upsert industry chain batch: %w", err)
		}
		report.IndustryChainCounts["membership"] = len(batch.Memberships)
		report.IndustryChainCounts["topology"] = len(batch.TopologyEdges)
		report.IndustryChainCounts["physical_constraint"] = len(batch.PhysicalConstraints)
		report.Created += result.Created
		report.Updated += result.Updated
		report.Unchanged += result.Unchanged
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
	case ApplyScopeAll, ApplyScopeIndustryChainMaster:
		return scope, nil
	default:
		return "", fmt.Errorf("unsupported apply scope %q", value)
	}
}

func applyManifestScope(manifest Manifest, scope ApplyScope) (Manifest, error) {
	if scope == ApplyScopeAll {
		return manifest, nil
	}
	if scope != ApplyScopeIndustryChainMaster {
		return Manifest{}, fmt.Errorf("unsupported apply scope %q", scope)
	}

	keys := map[string]struct{}{}
	for _, membership := range manifest.IndustryChainMemberships {
		keys[membership.IndustryChainKey] = struct{}{}
		keys[membership.ChainNodeKey] = struct{}{}
	}
	if len(keys) == 0 {
		return Manifest{}, fmt.Errorf("industry-chain-master scope requires reviewed memberships to identify master entities")
	}

	scoped := Manifest{}
	for _, entity := range manifest.Entities {
		if _, ok := keys[entity.Key]; ok {
			scoped.Entities = append(scoped.Entities, entity)
		}
	}
	for _, profile := range manifest.Profiles {
		if _, ok := keys[profile.EntityKey]; ok {
			scoped.Profiles = append(scoped.Profiles, profile)
		}
	}
	if err := Validate(scoped); err != nil {
		return Manifest{}, fmt.Errorf("validate industry-chain-master scope: %w", err)
	}
	return scoped, nil
}

func manifestIndustryChainBatch(manifest Manifest) IndustryChainBatch {
	batch := IndustryChainBatch{
		Memberships:         make([]domain.IndustryChainMembership, 0, len(manifest.IndustryChainMemberships)),
		TopologyEdges:       make([]domain.IndustryChainTopologyEdge, 0, len(manifest.IndustryChainTopologyEdges)),
		PhysicalConstraints: make([]domain.IndustryChainPhysicalConstraint, 0, len(manifest.IndustryChainPhysicalConstraints)),
		ApprovalGate:        domain.IndustryChainApprovalGate{HumanApprovedConstraintIDs: map[string]struct{}{}},
	}
	for _, item := range manifest.IndustryChainMemberships {
		batch.Memberships = append(batch.Memberships, domain.IndustryChainMembership{ID: relationshipSeedUUID(item.ID), IndustryChainEntityID: entitySeedUUID(item.IndustryChainKey), ChainNodeEntityID: entitySeedUUID(item.ChainNodeKey), StageCode: item.StageCode, RoleCode: item.RoleCode, StageOrder: item.StageOrder, IsCore: item.IsCore, SourceName: item.SourceName, SourceURL: item.SourceURL, VerifiedAt: item.VerifiedAt, Status: item.Status})
	}
	for _, item := range manifest.IndustryChainTopologyEdges {
		batch.TopologyEdges = append(batch.TopologyEdges, domain.IndustryChainTopologyEdge{ID: relationshipSeedUUID(item.ID), IndustryChainEntityID: entitySeedUUID(item.IndustryChainKey), FromChainNodeEntityID: entitySeedUUID(item.FromChainNodeKey), ToChainNodeEntityID: entitySeedUUID(item.ToChainNodeKey), RelationType: item.RelationType, EvidenceNote: item.EvidenceNote, SourceName: item.SourceName, SourceURL: item.SourceURL, VerifiedAt: item.VerifiedAt, Status: item.Status})
	}
	for _, item := range manifest.IndustryChainPhysicalConstraints {
		constraint := domain.IndustryChainPhysicalConstraint{ID: relationshipSeedUUID(item.ID), IndustryChainEntityID: entitySeedUUID(item.IndustryChainKey), ConstraintType: item.ConstraintType, Mechanism: item.Mechanism, PhysicalLimitNote: item.PhysicalLimitNote, MitigationPath: item.MitigationPath, SourceName: item.SourceName, SourceURL: item.SourceURL, VerifiedAt: item.VerifiedAt, ReviewStatus: item.ReviewStatus, Status: item.Status, GeneratedByAI: item.GeneratedByAI}
		if item.ChainNodeKey != "" {
			constraint.ChainNodeEntityID = entitySeedUUID(item.ChainNodeKey)
		}
		if item.TopologyEdgeID != "" {
			constraint.TopologyEdgeID = relationshipSeedUUID(item.TopologyEdgeID)
		}
		batch.PhysicalConstraints = append(batch.PhysicalConstraints, constraint)
		if item.ApprovedByHuman {
			batch.ApprovalGate.HumanApprovedConstraintIDs[constraint.ID] = struct{}{}
		}
	}
	return batch
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
