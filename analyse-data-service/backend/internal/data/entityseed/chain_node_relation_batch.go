package entityseed

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/entityseed"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"
)

var chainNodeRelationColumns = []string{"id", "from_chain_node_entity_id", "to_chain_node_entity_id", "relation_type", "mechanism", "condition_note", "evidence_note", "provenance", "verified_at", "status"}

type plannedChainNodeRelation struct {
	item   model.ChainNodeRelation
	action WriteAction
}

func chainNodeRelationActiveEndpointsSQL(lock bool) string {
	query := "SELECT count(*) FROM chain_node_profiles p JOIN entity_nodes e ON e.id=p.entity_id WHERE p.entity_id IN ($1::uuid,$2::uuid) AND e.entity_type='chain_node' AND e.status='active'"
	if lock {
		return "SELECT count(*) FROM (SELECT p.entity_id FROM chain_node_profiles p JOIN entity_nodes e ON e.id=p.entity_id WHERE p.entity_id IN ($1::uuid,$2::uuid) AND e.entity_type='chain_node' AND e.status='active' FOR SHARE OF p,e) locked_endpoints"
	}
	return query
}

func chainNodeRelationByIDSQL(lock bool) string {
	query := "SELECT id,from_chain_node_entity_id,to_chain_node_entity_id,relation_type,mechanism,COALESCE(condition_note,''),evidence_note,provenance,verified_at,status FROM chain_node_relations WHERE id=$1::uuid"
	if lock {
		query += " FOR UPDATE"
	}
	return query
}

func chainNodeRelationByTupleSQL(lock bool) string {
	query := "SELECT id,from_chain_node_entity_id,to_chain_node_entity_id,relation_type,mechanism,COALESCE(condition_note,''),evidence_note,provenance,verified_at,status FROM chain_node_relations WHERE from_chain_node_entity_id=$1::uuid AND to_chain_node_entity_id=$2::uuid AND relation_type=$3"
	if lock {
		query += " FOR UPDATE"
	}
	return query
}

func chainNodeRelationTransactionLockSQL() string {
	return "SELECT pg_advisory_xact_lock(hashtextextended($1,0))"
}

func chainNodeRelationInsertSQL() string {
	return "INSERT INTO chain_node_relations(id,from_chain_node_entity_id,to_chain_node_entity_id,relation_type,mechanism,condition_note,evidence_note,provenance,verified_at,status) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,NULLIF($6,''),$7,$8,$9,$10) RETURNING id"
}

func relationArgs(relation model.ChainNodeRelation) []driver.Value {
	return []driver.Value{relation.ID, relation.FromChainNodeEntityID, relation.ToChainNodeEntityID, relation.RelationType, relation.Mechanism, relation.ConditionNote, relation.EvidenceNote, relation.Provenance, relation.VerifiedAt, relation.Status}
}

func relationQueryArgs(relation model.ChainNodeRelation) []any {
	return []any{relation.ID, relation.FromChainNodeEntityID, relation.ToChainNodeEntityID, relation.RelationType, relation.Mechanism, relation.ConditionNote, relation.EvidenceNote, relation.Provenance, relation.VerifiedAt, relation.Status}
}

func scanChainNodeRelation(row *sql.Row) (model.ChainNodeRelation, error) {
	var relation model.ChainNodeRelation
	err := row.Scan(&relation.ID, &relation.FromChainNodeEntityID, &relation.ToChainNodeEntityID, &relation.RelationType, &relation.Mechanism, &relation.ConditionNote, &relation.EvidenceNote, &relation.Provenance, &relation.VerifiedAt, &relation.Status)
	return relation, err
}

func planChainNodeRelations(ctx context.Context, tx *sql.Tx, relations []model.ChainNodeRelation, readOnly bool) ([]plannedChainNodeRelation, error) {
	if err := model.ValidateChainNodeRelationBatch(relations); err != nil {
		return nil, err
	}
	planned := make([]plannedChainNodeRelation, 0, len(relations))
	for _, relation := range relations {
		identity := relation.FromChainNodeEntityID + "|" + string(relation.RelationType) + "|" + relation.ToChainNodeEntityID
		if !readOnly {
			if _, err := tx.ExecContext(ctx, chainNodeRelationTransactionLockSQL(), identity); err != nil {
				return nil, err
			}
		}
		var endpoints int
		if err := tx.QueryRowContext(ctx, chainNodeRelationActiveEndpointsSQL(!readOnly), relation.FromChainNodeEntityID, relation.ToChainNodeEntityID).Scan(&endpoints); err != nil {
			return nil, err
		}
		snapshot := biz.ChainNodeRelationWriteSnapshot{ActiveEndpoints: endpoints}
		existing, err := scanChainNodeRelation(tx.QueryRowContext(ctx, chainNodeRelationByIDSQL(!readOnly), relation.ID))
		switch err {
		case nil:
			snapshot.ExistingByID = &existing
		case sql.ErrNoRows:
		default:
			return nil, err
		}
		if snapshot.ExistingByID == nil {
			tuple, tupleErr := scanChainNodeRelation(tx.QueryRowContext(ctx, chainNodeRelationByTupleSQL(!readOnly), relation.FromChainNodeEntityID, relation.ToChainNodeEntityID, relation.RelationType))
			if tupleErr == nil {
				snapshot.ExistingByTuple = &tuple
			} else if tupleErr != sql.ErrNoRows {
				return nil, tupleErr
			}
		}
		action, err := biz.ClassifyChainNodeRelationWrite(relation, snapshot)
		if err != nil {
			return nil, err
		}
		planned = append(planned, plannedChainNodeRelation{item: relation, action: action})
	}
	return planned, nil
}

func relationReport(planned []plannedChainNodeRelation) ChainNodeRelationReport {
	report := ChainNodeRelationReport{ByRelationType: map[model.ChainNodeRelationType]int{}}
	for _, plan := range planned {
		switch plan.action {
		case WriteCreated:
			report.Created++
		case WriteUpdated:
			report.Updated++
		case WriteUnchanged:
			report.Unchanged++
		}
		report.ByRelationType[plan.item.RelationType]++
	}
	return report
}

func frozenChainNodeRelationBaseline(report ChainNodeRelationDataPreflightReport) biz.FrozenChainNodeRelationBaseline {
	return biz.FrozenChainNodeRelationBaseline{
		ExistingRelations:    report.ExistingRelations,
		SubcategoryRelations: report.SubcategoryRelations,
		ComponentRelations:   report.ComponentRelations,
		InputRelations:       report.InputRelations,
		DependsRelations:     report.DependsRelations,
	}
}

func frozenChainNodeRelationActions(planned []plannedChainNodeRelation) []biz.WriteAction {
	actions := make([]biz.WriteAction, 0, len(planned))
	for _, plan := range planned {
		actions = append(actions, plan.action)
	}
	return actions
}

func (r PostgresRepository) DryRunChainNodeRelationBatch(ctx context.Context, relations []model.ChainNodeRelation) (ChainNodeRelationReport, error) {
	return r.dryRunChainNodeRelationBatch(ctx, relations, false)
}

func (r PostgresRepository) DryRunFrozenChainNodeRelations(ctx context.Context, relations []model.ChainNodeRelation) (ChainNodeRelationReport, error) {
	return r.dryRunChainNodeRelationBatch(ctx, relations, true)
}

func (r PostgresRepository) dryRunChainNodeRelationBatch(ctx context.Context, relations []model.ChainNodeRelation, requireFrozenBaseline bool) (ChainNodeRelationReport, error) {
	if r.root == nil {
		return ChainNodeRelationReport{}, fmt.Errorf("postgres root database is required")
	}
	tx, err := r.root.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return ChainNodeRelationReport{}, err
	}
	defer tx.Rollback()
	var baseline ChainNodeRelationDataPreflightReport
	if requireFrozenBaseline {
		baseline, err = readChainNodeRelationDataBaseline(ctx, tx)
		if err != nil {
			return ChainNodeRelationReport{}, err
		}
		if err := biz.ValidateFrozenChainNodeRelationBaseline(frozenChainNodeRelationBaseline(baseline)); err != nil {
			return ChainNodeRelationReport{}, err
		}
		if _, err := assertChainNodeRelationDataBaseline(ctx, tx, baseline.ExistingRelations); err != nil {
			return ChainNodeRelationReport{}, err
		}
	}
	planned, err := planChainNodeRelations(ctx, tx, relations, true)
	if err != nil {
		return ChainNodeRelationReport{}, err
	}
	report := relationReport(planned)
	if requireFrozenBaseline {
		if err := biz.ValidateFrozenChainNodeRelationActions(frozenChainNodeRelationBaseline(baseline), frozenChainNodeRelationActions(planned)); err != nil {
			return ChainNodeRelationReport{}, err
		}
		if err := biz.ValidateFrozenChainNodeRelationPlan(frozenChainNodeRelationBaseline(baseline), report); err != nil {
			return ChainNodeRelationReport{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return ChainNodeRelationReport{}, err
	}
	return report, nil
}

func (r PostgresRepository) ApplyChainNodeRelationBatch(ctx context.Context, relations []model.ChainNodeRelation) (ChainNodeRelationReport, error) {
	return r.applyChainNodeRelationBatch(ctx, relations, false)
}

func (r PostgresRepository) ApplyFrozenChainNodeRelations(ctx context.Context, relations []model.ChainNodeRelation) (ChainNodeRelationReport, error) {
	return r.applyChainNodeRelationBatch(ctx, relations, true)
}

func (r PostgresRepository) applyChainNodeRelationBatch(ctx context.Context, relations []model.ChainNodeRelation, requireFrozenBaseline bool) (ChainNodeRelationReport, error) {
	if r.root == nil {
		return ChainNodeRelationReport{}, fmt.Errorf("postgres root database is required")
	}
	tx, err := r.root.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return ChainNodeRelationReport{}, err
	}
	defer tx.Rollback()
	var baseline ChainNodeRelationDataPreflightReport
	if requireFrozenBaseline {
		baseline, err = preflightChainNodeRelationData(ctx, tx)
		if err != nil {
			return ChainNodeRelationReport{}, err
		}
	}
	planned, err := planChainNodeRelations(ctx, tx, relations, false)
	if err != nil {
		return ChainNodeRelationReport{}, err
	}
	report := relationReport(planned)
	if requireFrozenBaseline {
		if err := biz.ValidateFrozenChainNodeRelationActions(frozenChainNodeRelationBaseline(baseline), frozenChainNodeRelationActions(planned)); err != nil {
			return ChainNodeRelationReport{}, err
		}
		if err := biz.ValidateFrozenChainNodeRelationPlan(frozenChainNodeRelationBaseline(baseline), report); err != nil {
			return ChainNodeRelationReport{}, err
		}
	}
	for _, plan := range planned {
		switch plan.action {
		case WriteCreated:
			var id string
			if err := tx.QueryRowContext(ctx, chainNodeRelationInsertSQL(), relationQueryArgs(plan.item)...).Scan(&id); err != nil {
				return ChainNodeRelationReport{}, fmt.Errorf("insert chain node relation %q: %w", plan.item.ID, err)
			}
		case WriteUpdated:
			if _, err := tx.ExecContext(ctx, "UPDATE chain_node_relations SET mechanism=$1,condition_note=NULLIF($2,''),evidence_note=$3,provenance=$4,verified_at=$5,status=$6,updated_at=now() WHERE id=$7::uuid", plan.item.Mechanism, plan.item.ConditionNote, plan.item.EvidenceNote, plan.item.Provenance, plan.item.VerifiedAt, plan.item.Status, plan.item.ID); err != nil {
				return ChainNodeRelationReport{}, err
			}
		}
	}
	if err := verifyChainNodeRelationBatchPostWrite(ctx, tx, planned, report); err != nil {
		return ChainNodeRelationReport{}, err
	}
	if requireFrozenBaseline {
		if err := verifyFrozenChainNodeRelationPostWrite(ctx, tx); err != nil {
			return ChainNodeRelationReport{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return ChainNodeRelationReport{}, err
	}
	return report, nil
}

const frozenChainNodeRelationAggregateSQL = `SELECT count(*),
 count(*) FILTER (WHERE relation_type='is_subcategory_of'),
 count(*) FILTER (WHERE relation_type='is_component_of'),
 count(*) FILTER (WHERE relation_type='input_to'),
 count(*) FILTER (WHERE relation_type='depends_on'),
 count(*) FILTER (WHERE status<>'active' OR btrim(mechanism)='' OR btrim(evidence_note)='' OR btrim(provenance)='' OR verified_at IS NULL),
 count(*) FILTER (WHERE from_chain_node_entity_id=to_chain_node_entity_id),
 count(*)-(SELECT count(*) FROM (SELECT DISTINCT from_chain_node_entity_id,to_chain_node_entity_id,relation_type FROM chain_node_relations) tuples),
 count(*) FILTER (WHERE fp.entity_id IS NULL OR tp.entity_id IS NULL)
 FROM chain_node_relations r
 LEFT JOIN chain_node_profiles fp ON fp.entity_id=r.from_chain_node_entity_id
 LEFT JOIN chain_node_profiles tp ON tp.entity_id=r.to_chain_node_entity_id`

func verifyFrozenChainNodeRelationPostWrite(ctx context.Context, tx *sql.Tx) error {
	if _, err := assertChainNodeRelationDataBaseline(ctx, tx, 212); err != nil {
		return err
	}
	var aggregate biz.FrozenChainNodeRelationAggregate
	if err := tx.QueryRowContext(ctx, frozenChainNodeRelationAggregateSQL).Scan(
		&aggregate.Total,
		&aggregate.Subcategory,
		&aggregate.Component,
		&aggregate.Input,
		&aggregate.Depends,
		&aggregate.Incomplete,
		&aggregate.SelfLoops,
		&aggregate.Duplicates,
		&aggregate.Orphans,
	); err != nil {
		return err
	}
	return biz.ValidateFrozenChainNodeRelationAggregate(aggregate)
}

func verifyChainNodeRelationBatchPostWrite(ctx context.Context, tx *sql.Tx, planned []plannedChainNodeRelation, report ChainNodeRelationReport) error {
	if report.Created+report.Updated+report.Unchanged != len(planned) {
		return fmt.Errorf("chain node relation report count mismatch")
	}
	for _, plan := range planned {
		got, err := scanChainNodeRelation(tx.QueryRowContext(ctx, chainNodeRelationByIDSQL(true), plan.item.ID))
		if err != nil {
			return fmt.Errorf("verify chain node relation %q: %w", plan.item.ID, err)
		}
		if !biz.ChainNodeRelationsEqual(got, plan.item) {
			return fmt.Errorf("verify chain node relation %q did not match manifest", plan.item.ID)
		}
	}
	return nil
}
