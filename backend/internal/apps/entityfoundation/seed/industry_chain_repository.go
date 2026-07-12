package seed

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type IndustryChainBatch struct {
	Profiles            []domain.IndustryChainProfile
	Memberships         []domain.IndustryChainMembership
	TopologyEdges       []domain.IndustryChainTopologyEdge
	PhysicalConstraints []domain.IndustryChainPhysicalConstraint
	ApprovalGate        domain.IndustryChainApprovalGate
}

type IndustryChainWriteReport struct {
	Created   int
	Updated   int
	Unchanged int
}

func (r *MemoryRepository) UpsertIndustryChainBatch(_ context.Context, batch IndustryChainBatch) (IndustryChainWriteReport, error) {
	if err := validateIndustryChainBatch(batch); err != nil {
		return IndustryChainWriteReport{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	profiles := cloneTypedMap(r.industryChainProfiles)
	memberships := cloneTypedMap(r.industryChainMemberships)
	topology := cloneTypedMap(r.industryChainTopologyEdges)
	constraints := cloneTypedMap(r.industryChainPhysicalConstraints)
	report := IndustryChainWriteReport{}
	for _, value := range batch.Profiles {
		if prior, ok := profiles[value.EntityID]; ok && prior.ChainCode != value.ChainCode {
			return IndustryChainWriteReport{}, fmt.Errorf("industry chain profile identity is immutable")
		}
		report.add(upsertTyped(profiles, value.EntityID, value))
	}
	for _, value := range batch.Memberships {
		if prior, ok := memberships[value.ID]; ok && (prior.IndustryChainEntityID != value.IndustryChainEntityID || prior.ChainNodeEntityID != value.ChainNodeEntityID) {
			return IndustryChainWriteReport{}, fmt.Errorf("industry chain membership identity is immutable")
		}
		report.add(upsertTyped(memberships, value.ID, value))
	}
	for _, value := range batch.TopologyEdges {
		if prior, ok := topology[value.ID]; ok && (prior.IndustryChainEntityID != value.IndustryChainEntityID || prior.FromChainNodeEntityID != value.FromChainNodeEntityID || prior.ToChainNodeEntityID != value.ToChainNodeEntityID) {
			return IndustryChainWriteReport{}, fmt.Errorf("industry chain topology identity is immutable")
		}
		report.add(upsertTyped(topology, value.ID, value))
	}
	for _, value := range batch.PhysicalConstraints {
		if prior, ok := constraints[value.ID]; ok && (prior.IndustryChainEntityID != value.IndustryChainEntityID || prior.ChainNodeEntityID != value.ChainNodeEntityID || prior.TopologyEdgeID != value.TopologyEdgeID) {
			return IndustryChainWriteReport{}, fmt.Errorf("industry chain physical constraint identity is immutable")
		}
		report.add(upsertTyped(constraints, value.ID, value))
	}
	r.industryChainProfiles, r.industryChainMemberships = profiles, memberships
	r.industryChainTopologyEdges, r.industryChainPhysicalConstraints = topology, constraints
	return report, nil
}

func (r *MemoryRepository) IndustryChainRowCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.industryChainProfiles) + len(r.industryChainMemberships) + len(r.industryChainTopologyEdges) + len(r.industryChainPhysicalConstraints)
}

func (r PostgresRepository) UpsertIndustryChainBatch(ctx context.Context, batch IndustryChainBatch) (IndustryChainWriteReport, error) {
	if err := validateIndustryChainBatch(batch); err != nil {
		return IndustryChainWriteReport{}, err
	}
	if r.root == nil {
		return IndustryChainWriteReport{}, fmt.Errorf("postgres root database is required")
	}
	tx, err := r.root.BeginTx(ctx, nil)
	if err != nil {
		return IndustryChainWriteReport{}, err
	}
	report := IndustryChainWriteReport{}
	rollback := func(cause error) (IndustryChainWriteReport, error) {
		_ = tx.Rollback()
		return IndustryChainWriteReport{}, cause
	}
	for _, value := range batch.Profiles {
		action, err := queryIndustryChainAction(ctx, tx, industryChainProfileUpsertSQL, value.EntityID, value.ChainCode, value.Definition, value.BoundaryNote, value.ScopeType, nullableString(value.PrimaryEconomyEntityID), value.Version, value.ReviewStatus, value.SourceName, value.SourceURL, value.VerifiedAt)
		if err != nil {
			return rollback(fmt.Errorf("upsert industry chain profile: %w", err))
		}
		report.add(action)
	}
	for _, value := range batch.Memberships {
		action, err := queryIndustryChainAction(ctx, tx, industryChainMembershipUpsertSQL, value.ID, value.IndustryChainEntityID, value.ChainNodeEntityID, value.StageCode, value.RoleCode, value.StageOrder, value.IsCore, value.SourceName, value.SourceURL, value.VerifiedAt, value.Status)
		if err != nil {
			return rollback(fmt.Errorf("upsert industry chain membership: %w", err))
		}
		report.add(action)
	}
	for _, value := range batch.TopologyEdges {
		action, err := queryIndustryChainAction(ctx, tx, industryChainTopologyUpsertSQL, value.ID, value.IndustryChainEntityID, value.FromChainNodeEntityID, value.ToChainNodeEntityID, value.RelationType, value.EvidenceNote, value.SourceName, value.SourceURL, value.VerifiedAt, value.Status)
		if err != nil {
			return rollback(fmt.Errorf("upsert industry chain topology: %w", err))
		}
		report.add(action)
	}
	for _, value := range batch.PhysicalConstraints {
		action, err := queryIndustryChainAction(ctx, tx, industryChainConstraintUpsertSQL, value.ID, value.IndustryChainEntityID, nullableString(value.ChainNodeEntityID), nullableString(value.TopologyEdgeID), value.ConstraintType, value.Mechanism, value.PhysicalLimitNote, value.MitigationPath, value.SourceName, value.SourceURL, value.VerifiedAt, value.ReviewStatus, value.Status, value.GeneratedByAI)
		if err != nil {
			return rollback(fmt.Errorf("upsert industry chain physical constraint: %w", err))
		}
		report.add(action)
	}
	if err := tx.Commit(); err != nil {
		return IndustryChainWriteReport{}, err
	}
	return report, nil
}

const industryChainProfileUpsertSQL = `WITH identity_guard AS (
SELECT EXISTS (SELECT 1 FROM industry_chain_profiles WHERE entity_id = $1 AND chain_code IS DISTINCT FROM $2) AS conflict
), upsert AS (
INSERT INTO industry_chain_profiles (entity_id, chain_code, definition, boundary_note, scope_type, primary_economy_entity_id, version, review_status, source_name, source_url, verified_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (entity_id) DO UPDATE SET chain_code=EXCLUDED.chain_code, definition=EXCLUDED.definition, boundary_note=EXCLUDED.boundary_note, scope_type=EXCLUDED.scope_type, primary_economy_entity_id=EXCLUDED.primary_economy_entity_id, version=EXCLUDED.version, review_status=EXCLUDED.review_status, source_name=EXCLUDED.source_name, source_url=EXCLUDED.source_url, verified_at=EXCLUDED.verified_at
WHERE (industry_chain_profiles.chain_code, industry_chain_profiles.definition, industry_chain_profiles.boundary_note, industry_chain_profiles.scope_type, industry_chain_profiles.primary_economy_entity_id, industry_chain_profiles.version, industry_chain_profiles.review_status, industry_chain_profiles.source_name, industry_chain_profiles.source_url, industry_chain_profiles.verified_at)
IS DISTINCT FROM (EXCLUDED.chain_code, EXCLUDED.definition, EXCLUDED.boundary_note, EXCLUDED.scope_type, EXCLUDED.primary_economy_entity_id, EXCLUDED.version, EXCLUDED.review_status, EXCLUDED.source_name, EXCLUDED.source_url, EXCLUDED.verified_at)
RETURNING xmax = 0 AS inserted)
SELECT CASE WHEN (SELECT conflict FROM identity_guard) THEN 'identity_conflict' ELSE COALESCE((SELECT CASE WHEN inserted THEN 'created' ELSE 'updated' END FROM upsert), 'unchanged') END`
const industryChainMembershipUpsertSQL = `WITH identity_guard AS (
SELECT EXISTS (SELECT 1 FROM industry_chain_memberships WHERE id = $1 AND (industry_chain_entity_id, chain_node_entity_id) IS DISTINCT FROM ($2::uuid, $3::uuid)) AS conflict
), upsert AS (
INSERT INTO industry_chain_memberships (id, industry_chain_entity_id, chain_node_entity_id, stage_code, role_code, stage_order, is_core, source_name, source_url, verified_at, status)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (id) DO UPDATE SET stage_code=EXCLUDED.stage_code, role_code=EXCLUDED.role_code, stage_order=EXCLUDED.stage_order, is_core=EXCLUDED.is_core, source_name=EXCLUDED.source_name, source_url=EXCLUDED.source_url, verified_at=EXCLUDED.verified_at, status=EXCLUDED.status
WHERE (industry_chain_memberships.stage_code, industry_chain_memberships.role_code, industry_chain_memberships.stage_order, industry_chain_memberships.is_core, industry_chain_memberships.source_name, industry_chain_memberships.source_url, industry_chain_memberships.verified_at, industry_chain_memberships.status)
IS DISTINCT FROM (EXCLUDED.stage_code, EXCLUDED.role_code, EXCLUDED.stage_order, EXCLUDED.is_core, EXCLUDED.source_name, EXCLUDED.source_url, EXCLUDED.verified_at, EXCLUDED.status)
RETURNING xmax = 0 AS inserted)
SELECT CASE WHEN (SELECT conflict FROM identity_guard) THEN 'identity_conflict' ELSE COALESCE((SELECT CASE WHEN inserted THEN 'created' ELSE 'updated' END FROM upsert), 'unchanged') END`
const industryChainTopologyUpsertSQL = `WITH identity_guard AS (
SELECT EXISTS (SELECT 1 FROM industry_chain_topology_edges WHERE id = $1 AND (industry_chain_entity_id, from_chain_node_entity_id, to_chain_node_entity_id) IS DISTINCT FROM ($2::uuid, $3::uuid, $4::uuid)) AS conflict
), upsert AS (
INSERT INTO industry_chain_topology_edges (id, industry_chain_entity_id, from_chain_node_entity_id, to_chain_node_entity_id, relation_type, evidence_note, source_name, source_url, verified_at, status)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (id) DO UPDATE SET relation_type=EXCLUDED.relation_type, evidence_note=EXCLUDED.evidence_note, source_name=EXCLUDED.source_name, source_url=EXCLUDED.source_url, verified_at=EXCLUDED.verified_at, status=EXCLUDED.status
WHERE (industry_chain_topology_edges.relation_type, industry_chain_topology_edges.evidence_note, industry_chain_topology_edges.source_name, industry_chain_topology_edges.source_url, industry_chain_topology_edges.verified_at, industry_chain_topology_edges.status)
IS DISTINCT FROM (EXCLUDED.relation_type, EXCLUDED.evidence_note, EXCLUDED.source_name, EXCLUDED.source_url, EXCLUDED.verified_at, EXCLUDED.status)
RETURNING xmax = 0 AS inserted)
SELECT CASE WHEN (SELECT conflict FROM identity_guard) THEN 'identity_conflict' ELSE COALESCE((SELECT CASE WHEN inserted THEN 'created' ELSE 'updated' END FROM upsert), 'unchanged') END`
const industryChainConstraintUpsertSQL = `WITH identity_guard AS (
SELECT EXISTS (SELECT 1 FROM industry_chain_physical_constraints WHERE id = $1 AND (industry_chain_entity_id, chain_node_entity_id, topology_edge_id) IS DISTINCT FROM ($2::uuid, $3::uuid, $4::uuid)) AS conflict
), upsert AS (
INSERT INTO industry_chain_physical_constraints (id, industry_chain_entity_id, chain_node_entity_id, topology_edge_id, constraint_type, mechanism, physical_limit_note, mitigation_path, source_name, source_url, verified_at, review_status, status, generated_by_ai)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
ON CONFLICT (id) DO UPDATE SET constraint_type=EXCLUDED.constraint_type, mechanism=EXCLUDED.mechanism, physical_limit_note=EXCLUDED.physical_limit_note, mitigation_path=EXCLUDED.mitigation_path, source_name=EXCLUDED.source_name, source_url=EXCLUDED.source_url, verified_at=EXCLUDED.verified_at, review_status=EXCLUDED.review_status, status=EXCLUDED.status, generated_by_ai=EXCLUDED.generated_by_ai
WHERE (industry_chain_physical_constraints.constraint_type, industry_chain_physical_constraints.mechanism, industry_chain_physical_constraints.physical_limit_note, industry_chain_physical_constraints.mitigation_path, industry_chain_physical_constraints.source_name, industry_chain_physical_constraints.source_url, industry_chain_physical_constraints.verified_at, industry_chain_physical_constraints.review_status, industry_chain_physical_constraints.status, industry_chain_physical_constraints.generated_by_ai)
IS DISTINCT FROM (EXCLUDED.constraint_type, EXCLUDED.mechanism, EXCLUDED.physical_limit_note, EXCLUDED.mitigation_path, EXCLUDED.source_name, EXCLUDED.source_url, EXCLUDED.verified_at, EXCLUDED.review_status, EXCLUDED.status, EXCLUDED.generated_by_ai)
RETURNING xmax = 0 AS inserted)
SELECT CASE WHEN (SELECT conflict FROM identity_guard) THEN 'identity_conflict' ELSE COALESCE((SELECT CASE WHEN inserted THEN 'created' ELSE 'updated' END FROM upsert), 'unchanged') END`

func queryIndustryChainAction(ctx context.Context, tx *sql.Tx, query string, args ...any) (WriteAction, error) {
	var action WriteAction
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&action); err != nil {
		return "", err
	}
	if action == "identity_conflict" {
		return "", fmt.Errorf("immutable industry chain identity conflict")
	}
	return action, nil
}

func validateIndustryChainBatch(batch IndustryChainBatch) error {
	for _, value := range batch.Profiles {
		if err := value.Validate(); err != nil {
			return err
		}
	}
	return domain.ValidateIndustryChainBatch(batch.Memberships, batch.TopologyEdges, batch.PhysicalConstraints, batch.ApprovalGate)
}

func (r *IndustryChainWriteReport) add(action WriteAction) {
	switch action {
	case WriteCreated:
		r.Created++
	case WriteUpdated:
		r.Updated++
	case WriteUnchanged:
		r.Unchanged++
	}
}

func upsertTyped[T any](values map[string]T, key string, value T) WriteAction {
	prior, ok := values[key]
	if ok && reflect.DeepEqual(prior, value) {
		return WriteUnchanged
	}
	values[key] = value
	if ok {
		return WriteUpdated
	}
	return WriteCreated
}

func cloneTypedMap[T any](values map[string]T) map[string]T {
	clone := make(map[string]T, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
