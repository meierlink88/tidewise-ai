package seed

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	repoids "github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

const FrozenFirstBatchChainNodeRelationManifestSHA256 = "7651e0b591df1e03838df00ebc9acd6101ebcc76da18a6a314ff478c9f42990e"

type ChainNodeRelationDataPreflightReport struct {
	DatabaseName        string `json:"database_name"`
	ServerVersion       string `json:"server_version"`
	GooseVersion        int    `json:"goose_version"`
	ActiveChainNodes    int    `json:"active_chain_nodes"`
	ChainNodeProfiles   int    `json:"chain_node_profiles"`
	ExternalIdentifiers int    `json:"external_identifiers"`
	EntityEdges         int    `json:"entity_edges"`
	ExistingRelations   int    `json:"existing_relations"`
	ExistingConstraints int    `json:"existing_constraints"`
	SchemaValid         bool   `json:"schema_valid"`
}

func ValidateFrozenFirstBatchChainNodeRelationManifest(path string, manifest ChainNodeRelationManifest) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if fmt.Sprintf("%x", sha256.Sum256(content)) != FrozenFirstBatchChainNodeRelationManifestSHA256 {
		return fmt.Errorf("chain node relation manifest hash does not match approved first batch")
	}
	if len(manifest.Relations) != 96 || len(manifest.PhysicalConstraints) != 0 {
		return fmt.Errorf("approved relation manifest requires 96 relations and zero physical constraints")
	}
	counts := map[domain.ChainNodeRelationType]int{}
	for _, relation := range manifest.Relations {
		counts[relation.RelationType]++
		identity := relation.FromChainNodeEntityID + "|" + string(relation.RelationType) + "|" + relation.ToChainNodeEntityID
		if relation.ID != repoids.NormalizeUUID("chain_node_relation", identity) {
			return fmt.Errorf("relation %q deterministic id mismatch", relation.ID)
		}
		if relation.Status != domain.StatusActive || relation.VerifiedAt.IsZero() {
			return fmt.Errorf("relation %q must be active and verified", relation.ID)
		}
		for _, evidence := range []string{"final-seed-candidate-artifacts/node-profile-seed-manifest.json", "9141571579ce2eee4b7524f686c6244572be6dd1bce2a1ee0494bcafd6aef05e", "derivation_rule=", "ffb243e", "main-serenity"} {
			if !strings.Contains(relation.Provenance, evidence) {
				return fmt.Errorf("relation %q provenance misses %q", relation.ID, evidence)
			}
		}
	}
	if counts[domain.ChainNodeRelationSubcategoryOf] != 95 || counts[domain.ChainNodeRelationComponentOf] != 1 || counts[domain.ChainNodeRelationInputTo] != 0 || counts[domain.ChainNodeRelationDependsOn] != 0 {
		return fmt.Errorf("approved relation type counts do not match 95/1/0/0")
	}
	return domain.ValidateChainNodeRelationBatch(manifest.Relations)
}

const relationDataBaselineSQL = `SELECT current_database(), current_setting('server_version'),
 (SELECT version_id FROM goose_db_version ORDER BY id DESC LIMIT 1),
 (SELECT count(*) FROM entity_nodes WHERE entity_type='chain_node' AND status='active'),
 (SELECT count(*) FROM chain_node_profiles p JOIN entity_nodes e ON e.id=p.entity_id WHERE e.entity_type='chain_node' AND e.status='active'),
 (SELECT count(*) FROM entity_external_identifiers),
 (SELECT count(*) FROM entity_edges),
 (SELECT count(*) FROM chain_node_relations),
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
	if err := db.QueryRowContext(ctx, relationDataBaselineSQL).Scan(&report.DatabaseName, &report.ServerVersion, &report.GooseVersion, &report.ActiveChainNodes, &report.ChainNodeProfiles, &report.ExternalIdentifiers, &report.EntityEdges, &report.ExistingRelations, &report.ExistingConstraints); err != nil {
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
	if report.DatabaseName != "tidewise_local" || !strings.HasPrefix(report.ServerVersion, "16.14") || report.GooseVersion != 17 || report.ActiveChainNodes != 842 || report.ChainNodeProfiles != 842 || report.ExternalIdentifiers != 1169 || report.EntityEdges != 331 || report.ExistingRelations != expectedRelations || report.ExistingConstraints != 0 || !report.SchemaValid {
		return report, fmt.Errorf("relation data preflight baseline mismatch: %+v", report)
	}
	return report, nil
}

func preflightChainNodeRelationData(ctx context.Context, db postgresExecutor) (ChainNodeRelationDataPreflightReport, error) {
	return assertChainNodeRelationDataBaseline(ctx, db, 0)
}

func (r PostgresRepository) PreflightFrozenChainNodeRelationData(ctx context.Context) (ChainNodeRelationDataPreflightReport, error) {
	if r.root == nil {
		return ChainNodeRelationDataPreflightReport{}, fmt.Errorf("postgres root database is required")
	}
	return preflightChainNodeRelationData(ctx, r.root)
}
