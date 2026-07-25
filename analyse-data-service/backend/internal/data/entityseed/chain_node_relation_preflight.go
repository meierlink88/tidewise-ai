package entityseed

import (
	"context"
	"fmt"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/entityseed"
)

const relationDataBaselineSQL = `SELECT current_database(), current_setting('server_version'),
 (SELECT version_id FROM goose_db_version ORDER BY id DESC LIMIT 1),
 (SELECT count(*) FROM entity_nodes WHERE entity_type='chain_node' AND status='active'),
 (SELECT count(*) FROM chain_node_profiles p JOIN entity_nodes e ON e.id=p.entity_id WHERE e.entity_type='chain_node' AND e.status='active'),
 (SELECT count(*) FROM entity_external_identifiers),
 (SELECT count(*) FROM entity_edges),
 (SELECT count(*) FROM chain_node_relations),
 (SELECT count(*) FROM chain_node_relations WHERE relation_type='is_subcategory_of'),
 (SELECT count(*) FROM chain_node_relations WHERE relation_type='is_component_of'),
 (SELECT count(*) FROM chain_node_relations WHERE relation_type='input_to'),
 (SELECT count(*) FROM chain_node_relations WHERE relation_type='depends_on'),
 (SELECT count(*) FROM chain_node_physical_constraints)`

const relationDataSchemaSQL = `SELECT
 (SELECT string_agg(column_name||':'||udt_name||':'||is_nullable||':'||COALESCE(column_default,''),',' ORDER BY ordinal_position) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='chain_node_relations'),
 (SELECT string_agg(column_name||':'||udt_name||':'||is_nullable||':'||COALESCE(column_default,''),',' ORDER BY ordinal_position) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='chain_node_physical_constraints'),
 (SELECT count(*) FROM pg_constraint WHERE conrelid='chain_node_relations'::regclass AND contype='c'),
 (SELECT count(*) FROM pg_constraint WHERE conrelid='chain_node_relations'::regclass AND contype='f'),
 (SELECT count(*) FROM pg_constraint WHERE conrelid='chain_node_relations'::regclass AND contype='p'),
 (SELECT count(*) FROM pg_constraint WHERE conrelid='chain_node_relations'::regclass AND contype='u'),
 (SELECT count(*) FROM pg_constraint WHERE conrelid='chain_node_physical_constraints'::regclass AND contype='c'),
 (SELECT count(*) FROM pg_constraint WHERE conrelid='chain_node_physical_constraints'::regclass AND contype='f'),
 (SELECT count(*) FROM pg_constraint WHERE conrelid='chain_node_physical_constraints'::regclass AND contype='p'),
 (SELECT count(*) FROM pg_indexes WHERE schemaname=current_schema() AND tablename='chain_node_relations' AND indexname IN ('chain_node_relations_pkey','chain_node_relations_from_chain_node_entity_id_to_chain_nod_key','chain_node_relations_to_type_idx','chain_node_relations_input_dependency_mechanism_uidx')),
 (SELECT count(*) FROM pg_indexes WHERE schemaname=current_schema() AND tablename='chain_node_physical_constraints' AND indexname IN ('chain_node_physical_constraints_pkey','chain_node_physical_constraints_node_subject_idx','chain_node_physical_constraints_relation_subject_idx')),
 (SELECT count(*) FROM pg_trigger WHERE tgrelid IN ('chain_node_relations'::regclass,'chain_node_physical_constraints'::regclass) AND NOT tgisinternal)`

const relationColumnSignature = "id:uuid:NO:,from_chain_node_entity_id:uuid:NO:,to_chain_node_entity_id:uuid:NO:,relation_type:text:NO:,mechanism:text:NO:,condition_note:text:YES:,evidence_note:text:NO:,provenance:text:NO:,verified_at:timestamptz:NO:,status:text:NO:'active'::text,created_at:timestamptz:NO:now(),updated_at:timestamptz:NO:now()"
const physicalConstraintColumnSignature = "id:uuid:NO:,chain_node_entity_id:uuid:YES:,chain_node_relation_id:uuid:YES:,constraint_type:text:NO:,description:text:NO:,condition_note:text:YES:,evidence_note:text:NO:,provenance:text:NO:,verified_at:timestamptz:NO:,status:text:NO:'active'::text,created_at:timestamptz:NO:now(),updated_at:timestamptz:NO:now()"

func readChainNodeRelationDataBaseline(ctx context.Context, db postgresExecutor) (ChainNodeRelationDataPreflightReport, error) {
	var report ChainNodeRelationDataPreflightReport
	if err := db.QueryRowContext(ctx, relationDataBaselineSQL).Scan(&report.DatabaseName, &report.ServerVersion, &report.GooseVersion, &report.ActiveChainNodes, &report.ChainNodeProfiles, &report.ExternalIdentifiers, &report.EntityEdges, &report.ExistingRelations, &report.SubcategoryRelations, &report.ComponentRelations, &report.InputRelations, &report.DependsRelations, &report.ExistingConstraints); err != nil {
		return report, err
	}
	var relationColumns, constraintColumns string
	var relationChecks, relationFKs, relationPKs, relationUniques, constraintChecks, constraintFKs, constraintPKs, relationIndexes, constraintIndexes, triggers int
	if err := db.QueryRowContext(ctx, relationDataSchemaSQL).Scan(&relationColumns, &constraintColumns, &relationChecks, &relationFKs, &relationPKs, &relationUniques, &constraintChecks, &constraintFKs, &constraintPKs, &relationIndexes, &constraintIndexes, &triggers); err != nil {
		return report, err
	}
	report.SchemaValid = relationColumns == relationColumnSignature && constraintColumns == physicalConstraintColumnSignature && relationChecks == 7 && relationFKs == 2 && relationPKs == 1 && relationUniques == 1 && constraintChecks == 7 && constraintFKs == 2 && constraintPKs == 1 && relationIndexes == 4 && constraintIndexes == 3 && triggers == 0
	return report, nil
}

func assertChainNodeRelationDataBaseline(ctx context.Context, db postgresExecutor, expectedRelations int) (ChainNodeRelationDataPreflightReport, error) {
	report, err := readChainNodeRelationDataBaseline(ctx, db)
	if err != nil {
		return report, err
	}
	if err := biz.ValidateChainNodeRelationDataPreflight(report, expectedRelations); err != nil {
		return report, err
	}
	return report, nil
}

func preflightChainNodeRelationData(ctx context.Context, db postgresExecutor) (ChainNodeRelationDataPreflightReport, error) {
	report, err := assertChainNodeRelationDataBaseline(ctx, db, 100)
	if err != nil {
		return report, err
	}
	if err := biz.ValidateFrozenChainNodeRelationBaseline(frozenChainNodeRelationBaseline(report)); err != nil {
		return report, err
	}
	return report, nil
}

func (r PostgresRepository) PreflightFrozenChainNodeRelationData(ctx context.Context) (ChainNodeRelationDataPreflightReport, error) {
	if r.root == nil {
		return ChainNodeRelationDataPreflightReport{}, fmt.Errorf("postgres root database is required")
	}
	return preflightChainNodeRelationData(ctx, r.root)
}
