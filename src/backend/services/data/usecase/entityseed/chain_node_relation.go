package seed

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

type ChainNodeRelationManifest struct {
	Relations           []domain.ChainNodeRelation           `json:"relations"`
	PhysicalConstraints []domain.ChainNodePhysicalConstraint `json:"physical_constraints,omitempty"`
}

type ChainNodeRelationReport struct {
	Created        int                                  `json:"created"`
	Updated        int                                  `json:"updated"`
	Unchanged      int                                  `json:"unchanged"`
	ByRelationType map[domain.ChainNodeRelationType]int `json:"by_relation_type"`
}

type chainNodeRelationSnapshotRepository interface {
	IsActiveChainNode(context.Context, string) (bool, error)
	FindChainNodeRelationByID(context.Context, string) (domain.ChainNodeRelation, bool, error)
	FindChainNodeRelationByTuple(context.Context, domain.ChainNodeRelation) (domain.ChainNodeRelation, bool, error)
}

func LoadChainNodeRelationManifest(path string) (ChainNodeRelationManifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return ChainNodeRelationManifest{}, err
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	decoder.DisallowUnknownFields()
	var manifest ChainNodeRelationManifest
	if err := decoder.Decode(&manifest); err != nil {
		return manifest, fmt.Errorf("decode chain node relation manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return manifest, fmt.Errorf("chain node relation manifest must contain a single JSON document")
		}
		return manifest, fmt.Errorf("chain node relation manifest must contain a single JSON document: %w", err)
	}
	if err := domain.ValidateChainNodeRelationBatch(manifest.Relations); err != nil {
		return manifest, err
	}
	for _, constraint := range manifest.PhysicalConstraints {
		if err := constraint.Validate(); err != nil {
			return manifest, err
		}
	}
	return manifest, nil
}

func DryRunChainNodeRelations(ctx context.Context, repo chainNodeRelationSnapshotRepository, relations []domain.ChainNodeRelation) (ChainNodeRelationReport, error) {
	report := ChainNodeRelationReport{ByRelationType: map[domain.ChainNodeRelationType]int{}}
	if err := domain.ValidateChainNodeRelationBatch(relations); err != nil {
		return report, err
	}
	for _, relation := range relations {
		for _, endpoint := range []string{relation.FromChainNodeEntityID, relation.ToChainNodeEntityID} {
			active, err := repo.IsActiveChainNode(ctx, endpoint)
			if err != nil {
				return report, err
			}
			if !active {
				return report, fmt.Errorf("relation %q requires active chain_node endpoint %q", relation.ID, endpoint)
			}
		}
		existing, found, err := repo.FindChainNodeRelationByID(ctx, relation.ID)
		if err != nil {
			return report, err
		}
		if !found {
			if tuple, tupleFound, err := repo.FindChainNodeRelationByTuple(ctx, relation); err != nil {
				return report, err
			} else if tupleFound && tuple.ID != relation.ID {
				return report, fmt.Errorf("chain node relation %q tuple conflict with %q", relation.ID, tuple.ID)
			}
			report.Created++
		} else if existing.FromChainNodeEntityID != relation.FromChainNodeEntityID || existing.ToChainNodeEntityID != relation.ToChainNodeEntityID || existing.RelationType != relation.RelationType {
			return report, fmt.Errorf("chain node relation %q identity conflict", relation.ID)
		} else if reflect.DeepEqual(existing, relation) {
			report.Unchanged++
		} else {
			report.Updated++
		}
		report.ByRelationType[relation.RelationType]++
	}
	return report, nil
}

func (r PostgresRepository) DryRunChainNodeRelationManifest(ctx context.Context, manifest ChainNodeRelationManifest) (ChainNodeRelationReport, error) {
	if err := ValidateChainNodeRelationDryRunManifest(manifest); err != nil {
		return ChainNodeRelationReport{}, err
	}
	return r.DryRunChainNodeRelationBatch(ctx, manifest.Relations)
}

func ValidateChainNodeRelationDryRunManifest(manifest ChainNodeRelationManifest) error {
	if len(manifest.PhysicalConstraints) != 0 {
		return fmt.Errorf("relation dry-run rejects physical_constraints until its repository and dry-run contract are implemented")
	}
	return nil
}

type chainNodeRelationMemoryRepository struct {
	active map[string]struct{}
	values map[string]domain.ChainNodeRelation
}

func newChainNodeRelationMemoryRepository(active []string) *chainNodeRelationMemoryRepository {
	r := &chainNodeRelationMemoryRepository{active: map[string]struct{}{}, values: map[string]domain.ChainNodeRelation{}}
	for _, id := range active {
		r.active[id] = struct{}{}
	}
	return r
}
func (r *chainNodeRelationMemoryRepository) IsActiveChainNode(_ context.Context, id string) (bool, error) {
	_, ok := r.active[id]
	return ok, nil
}
func (r *chainNodeRelationMemoryRepository) FindChainNodeRelationByID(_ context.Context, id string) (domain.ChainNodeRelation, bool, error) {
	v, ok := r.values[id]
	return v, ok, nil
}
func (r *chainNodeRelationMemoryRepository) FindChainNodeRelationByTuple(_ context.Context, wanted domain.ChainNodeRelation) (domain.ChainNodeRelation, bool, error) {
	for _, v := range r.values {
		if v.FromChainNodeEntityID == wanted.FromChainNodeEntityID && v.ToChainNodeEntityID == wanted.ToChainNodeEntityID && v.RelationType == wanted.RelationType {
			return v, true, nil
		}
	}
	return domain.ChainNodeRelation{}, false, nil
}
func (r *chainNodeRelationMemoryRepository) UpsertChainNodeRelation(ctx context.Context, relation domain.ChainNodeRelation) (WriteResult, error) {
	report, err := DryRunChainNodeRelations(ctx, r, []domain.ChainNodeRelation{relation})
	if err != nil {
		return WriteResult{}, err
	}
	action := WriteCreated
	if report.Updated == 1 {
		action = WriteUpdated
	}
	if report.Unchanged == 1 {
		action = WriteUnchanged
	}
	r.values[relation.ID] = relation
	return WriteResult{Key: relation.ID, Action: action}, nil
}

func (r PostgresRepository) IsActiveChainNode(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM entity_nodes WHERE id=$1::uuid AND entity_type='chain_node' AND status='active')`, id).Scan(&exists)
	return exists, err
}

func (r PostgresRepository) FindChainNodeRelationByID(ctx context.Context, id string) (domain.ChainNodeRelation, bool, error) {
	var v domain.ChainNodeRelation
	err := r.db.QueryRowContext(ctx, `SELECT id,from_chain_node_entity_id,to_chain_node_entity_id,relation_type,mechanism,COALESCE(condition_note,''),evidence_note,provenance,verified_at,status FROM chain_node_relations WHERE id=$1::uuid`, id).Scan(&v.ID, &v.FromChainNodeEntityID, &v.ToChainNodeEntityID, &v.RelationType, &v.Mechanism, &v.ConditionNote, &v.EvidenceNote, &v.Provenance, &v.VerifiedAt, &v.Status)
	if err == sql.ErrNoRows {
		return v, false, nil
	}
	return v, err == nil, err
}
func (r PostgresRepository) FindChainNodeRelationByTuple(ctx context.Context, wanted domain.ChainNodeRelation) (domain.ChainNodeRelation, bool, error) {
	var v domain.ChainNodeRelation
	err := r.db.QueryRowContext(ctx, `SELECT id,from_chain_node_entity_id,to_chain_node_entity_id,relation_type,mechanism,COALESCE(condition_note,''),evidence_note,provenance,verified_at,status FROM chain_node_relations WHERE from_chain_node_entity_id=$1::uuid AND to_chain_node_entity_id=$2::uuid AND relation_type=$3`, wanted.FromChainNodeEntityID, wanted.ToChainNodeEntityID, wanted.RelationType).Scan(&v.ID, &v.FromChainNodeEntityID, &v.ToChainNodeEntityID, &v.RelationType, &v.Mechanism, &v.ConditionNote, &v.EvidenceNote, &v.Provenance, &v.VerifiedAt, &v.Status)
	if err == sql.ErrNoRows {
		return v, false, nil
	}
	return v, err == nil, err
}

func (r PostgresRepository) UpsertChainNodeRelation(ctx context.Context, relation domain.ChainNodeRelation) (WriteResult, error) {
	report, err := DryRunChainNodeRelations(ctx, r, []domain.ChainNodeRelation{relation})
	if err != nil {
		return WriteResult{}, err
	}
	if report.Unchanged == 1 {
		return WriteResult{Key: relation.ID, Action: WriteUnchanged}, nil
	}
	action, err := r.queryWriteAction(ctx, `INSERT INTO chain_node_relations(id,from_chain_node_entity_id,to_chain_node_entity_id,relation_type,mechanism,condition_note,evidence_note,provenance,verified_at,status) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,NULLIF($6,''),$7,$8,$9,$10) ON CONFLICT(id) DO UPDATE SET mechanism=EXCLUDED.mechanism,condition_note=EXCLUDED.condition_note,evidence_note=EXCLUDED.evidence_note,provenance=EXCLUDED.provenance,verified_at=EXCLUDED.verified_at,status=EXCLUDED.status,updated_at=now() WHERE chain_node_relations.from_chain_node_entity_id=EXCLUDED.from_chain_node_entity_id AND chain_node_relations.to_chain_node_entity_id=EXCLUDED.to_chain_node_entity_id AND chain_node_relations.relation_type=EXCLUDED.relation_type AND (chain_node_relations.mechanism,chain_node_relations.condition_note,chain_node_relations.evidence_note,chain_node_relations.provenance,chain_node_relations.verified_at,chain_node_relations.status) IS DISTINCT FROM (EXCLUDED.mechanism,EXCLUDED.condition_note,EXCLUDED.evidence_note,EXCLUDED.provenance,EXCLUDED.verified_at,EXCLUDED.status) RETURNING CASE WHEN xmax=0 THEN 'created' ELSE 'updated' END`, relation.ID, relation.FromChainNodeEntityID, relation.ToChainNodeEntityID, relation.RelationType, relation.Mechanism, relation.ConditionNote, relation.EvidenceNote, relation.Provenance, relation.VerifiedAt, relation.Status)
	if err != nil {
		return WriteResult{}, err
	}
	return WriteResult{Key: relation.ID, Action: action}, nil
}
