package dbmigration

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
)

const (
	typedMasterDataMigration           = "000027_add_typed_master_data_schema.sql"
	chainNodeReviewValidationMigration = "000028_validate_chain_node_review_status.sql"
)

func TestTypedMasterDataMigrationIsSchemaOnly(t *testing.T) {
	raw := readMigration(t, typedMasterDataMigration)
	up, down := migrationSections(t, raw)
	normalized := strings.ToLower(up)

	for _, fragment := range []string{
		"create unique index ux_entity_nodes_nonblank_key",
		"on entity_nodes (entity_key)",
		"using gin (aliases)",
		"create table industry_profiles",
		"classification_system",
		"classification_version",
		"classification_level",
		"parent_industry_entity_id",
		"hierarchy_path_codes",
		"create table concept_profiles",
		"concept_type",
		"alter table chain_node_profiles",
		"add column review_status",
		"create table industry_chain_definitions",
		"create table industry_chain_node_memberships",
		"contextual_stage",
		"create table industry_chain_graph_edges",
		"segment_kind",
		"compressed_candidate",
		"create table entity_redirects",
		"redirect_kind",
		"create trigger",
		"with recursive",
	} {
		if !strings.Contains(normalized, fragment) {
			t.Errorf("typed master data migration Up must contain %q", fragment)
		}
	}

	requireTableColumns(t, normalized, "industry_profiles", []string{
		"entity_id", "classification_system", "classification_version", "industry_code",
		"classification_level", "parent_industry_entity_id", "hierarchy_path_codes",
		"definition", "boundary_note", "review_status", "created_at", "updated_at",
	})
	requireTableColumns(t, normalized, "concept_profiles", []string{
		"entity_id", "concept_type", "definition", "boundary_note", "review_status", "created_at", "updated_at",
	})
	requireTableColumns(t, normalized, "industry_chain_definitions", []string{
		"entity_id", "scope", "target_output", "end_use", "geography", "as_of_date",
		"review_status", "review_note", "created_at", "updated_at",
	})
	requireTableColumns(t, normalized, "industry_chain_node_memberships", []string{
		"industry_chain_entity_id", "chain_node_entity_id", "position", "contextual_stage",
		"review_status", "status", "created_at", "updated_at",
	})
	requireTableColumns(t, normalized, "industry_chain_graph_edges", []string{
		"id", "industry_chain_entity_id", "from_chain_node_entity_id", "to_chain_node_entity_id",
		"relation_type", "mechanism", "condition_note", "segment_kind", "omitted_step_note",
		"review_status", "status", "created_at", "updated_at",
	})
	requireTableColumns(t, normalized, "entity_redirects", []string{
		"source_entity_id", "target_entity_id", "redirect_kind", "reason", "review_status", "created_at", "updated_at",
	})

	requireTableDefinitionFragments(t, normalized, "industry_profiles", []string{
		"entity_id uuid primary key references entity_nodes(id) on delete restrict",
		"classification_system text not null", "classification_version text not null", "industry_code text not null",
		"classification_level smallint not null", "parent_industry_entity_id uuid references industry_profiles(entity_id) on delete restrict",
		"hierarchy_path_codes text[] not null", "definition text not null", "boundary_note text not null",
		"review_status varchar(32) not null", "created_at timestamptz not null default now()", "updated_at timestamptz not null default now()",
	})
	requireTableDefinitionFragments(t, normalized, "concept_profiles", []string{
		"entity_id uuid primary key references entity_nodes(id) on delete restrict", "concept_type varchar(32) not null",
		"definition text not null", "boundary_note text not null", "review_status varchar(32) not null",
		"created_at timestamptz not null default now()", "updated_at timestamptz not null default now()",
	})
	requireTableDefinitionFragments(t, normalized, "industry_chain_definitions", []string{
		"entity_id uuid primary key references entity_nodes(id) on delete restrict", "scope text not null",
		"target_output text not null", "end_use text not null", "geography text not null", "as_of_date date not null",
		"review_status varchar(32) not null", "review_note text", "created_at timestamptz not null default now()",
		"updated_at timestamptz not null default now()",
	})
	requireTableDefinitionFragments(t, normalized, "industry_chain_node_memberships", []string{
		"industry_chain_entity_id uuid not null references industry_chain_definitions(entity_id) on delete restrict",
		"chain_node_entity_id uuid not null references chain_node_profiles(entity_id) on delete restrict", "position integer not null",
		"contextual_stage varchar(32) not null", "review_status varchar(32) not null",
		"status varchar(32) not null default 'active'", "created_at timestamptz not null default now()",
		"updated_at timestamptz not null default now()", "primary key (industry_chain_entity_id, chain_node_entity_id)",
	})
	requireTableDefinitionFragments(t, normalized, "industry_chain_graph_edges", []string{
		"id uuid primary key", "industry_chain_entity_id uuid not null", "from_chain_node_entity_id uuid not null",
		"to_chain_node_entity_id uuid not null", "relation_type varchar(32) not null", "mechanism text not null",
		"condition_note text", "segment_kind varchar(32) not null", "omitted_step_note text",
		"review_status varchar(32) not null", "status varchar(32) not null default 'active'",
		"created_at timestamptz not null default now()", "updated_at timestamptz not null default now()",
		"foreign key (industry_chain_entity_id, from_chain_node_entity_id) references industry_chain_node_memberships (industry_chain_entity_id, chain_node_entity_id) on delete restrict",
		"foreign key (industry_chain_entity_id, to_chain_node_entity_id) references industry_chain_node_memberships (industry_chain_entity_id, chain_node_entity_id) on delete restrict",
	})
	requireTableDefinitionFragments(t, normalized, "entity_redirects", []string{
		"source_entity_id uuid primary key references entity_nodes(id) on delete restrict",
		"target_entity_id uuid not null references entity_nodes(id) on delete restrict", "redirect_kind varchar(32) not null",
		"reason text not null", "review_status varchar(32) not null", "created_at timestamptz not null default now()",
		"updated_at timestamptz not null default now()",
	})

	for _, objectName := range []string{
		"uq_industry_profile_classification_identity",
		"chk_industry_profile_system_nonblank", "chk_industry_profile_version_nonblank",
		"chk_industry_profile_code_nonblank", "chk_industry_profile_level",
		"chk_industry_profile_parent_presence", "chk_industry_profile_path_length",
		"chk_industry_profile_path_leaf", "chk_industry_profile_definition_nonblank",
		"chk_industry_profile_boundary_nonblank", "chk_industry_profile_review_status",
		"chk_concept_profile_type", "chk_concept_profile_definition_nonblank",
		"chk_concept_profile_boundary_nonblank", "chk_concept_profile_review_status",
		"chk_chain_node_profile_review_status",
		"chk_industry_chain_definition_scope_nonblank", "chk_industry_chain_definition_target_output_nonblank",
		"chk_industry_chain_definition_end_use_nonblank", "chk_industry_chain_definition_geography_nonblank",
		"chk_industry_chain_definition_review_status", "chk_industry_chain_definition_review_note_nonblank",
		"chk_industry_chain_node_membership_position", "chk_industry_chain_node_membership_stage",
		"chk_industry_chain_node_membership_review_status", "chk_industry_chain_node_membership_status",
		"fk_industry_chain_graph_from_membership", "fk_industry_chain_graph_to_membership",
		"uq_industry_chain_graph_semantic_edge", "chk_industry_chain_graph_self",
		"chk_industry_chain_graph_relation_type", "chk_industry_chain_graph_mechanism_nonblank",
		"chk_industry_chain_graph_condition_nonblank", "chk_industry_chain_graph_segment_kind",
		"chk_industry_chain_graph_omitted_step", "chk_industry_chain_graph_review_status",
		"chk_industry_chain_graph_status", "chk_entity_redirect_self", "chk_entity_redirect_kind",
		"chk_entity_redirect_reason_nonblank", "chk_entity_redirect_review_status",
		"ux_entity_nodes_nonblank_key", "idx_entity_nodes_aliases_gin",
		"idx_industry_profiles_parent", "idx_industry_profiles_classification_level",
		"idx_industry_profiles_review_status", "idx_concept_profiles_type_review",
		"idx_industry_chain_definitions_review_date",
		"idx_industry_chain_node_memberships_chain_status_position",
		"idx_industry_chain_node_memberships_node_status", "idx_industry_chain_graph_chain_status",
		"idx_industry_chain_graph_to_node_status", "idx_entity_redirects_target",
		"idx_entity_redirects_review_status", "trg_industry_profile_entity_type",
		"trg_concept_profile_entity_type", "trg_chain_node_profile_entity_type",
		"trg_industry_chain_definition_entity_type", "trg_protect_profiled_entity_identity",
		"trg_validate_industry_profile_hierarchy", "trg_protect_active_industry_chain_membership",
		"trg_reject_industry_chain_graph_cycle",
		"trg_validate_entity_redirect", "assert_entity_profile_type", "protect_profiled_entity_identity",
		"validate_industry_profile_hierarchy", "protect_active_industry_chain_membership",
		"reject_industry_chain_graph_cycle", "validate_entity_redirect",
	} {
		if !strings.Contains(normalized, objectName) {
			t.Errorf("typed master data migration Up must declare %q", objectName)
		}
	}

	dml := regexp.MustCompile(`(?mi)^\s*(insert\s+into|update\s+|delete\s+from|truncate\s+)`)
	if match := dml.FindString(up); match != "" {
		t.Fatalf("typed master data migration must contain no business DML, found %q", strings.TrimSpace(match))
	}
	for _, forbidden := range []string{
		"source_mapping",
		"quarantine",
		"keep_separate",
		"research_themes",
		"research_anchors",
		"events",
		"event_sources",
		"seed",
	} {
		if strings.Contains(normalized, forbidden) {
			t.Fatalf("typed master data migration contains out-of-scope fragment %q", forbidden)
		}
	}
	if !strings.Contains(strings.ToLower(down), "migration 000027 is forward-only") || !strings.Contains(strings.ToLower(down), "raise exception") {
		t.Fatal("typed master data migration Down must fail closed as forward-only")
	}
}

func TestChainNodeReviewValidationMigrationIsSchemaOnly(t *testing.T) {
	raw := readMigration(t, chainNodeReviewValidationMigration)
	up, down := migrationSections(t, raw)
	normalized := strings.ToLower(up)

	for _, fragment := range []string{
		"create trigger trg_chain_node_profile_review_status_entity_type",
		"before update of review_status on chain_node_profiles",
		"new.review_status is not null",
		"new.review_status is distinct from old.review_status",
		"execute function assert_entity_profile_type('chain_node')",
	} {
		if !strings.Contains(normalized, fragment) {
			t.Errorf("chain node review validation migration Up must contain %q", fragment)
		}
	}

	dml := regexp.MustCompile(`(?mi)^\s*(insert\s+into|update\s+|delete\s+from|truncate\s+)`)
	if match := dml.FindString(up); match != "" {
		t.Fatalf("chain node review validation migration must contain no business DML, found %q", strings.TrimSpace(match))
	}
	if !strings.Contains(strings.ToLower(down), "migration 000028 is forward-only") || !strings.Contains(strings.ToLower(down), "raise exception") {
		t.Fatal("chain node review validation migration Down must fail closed as forward-only")
	}
}

func TestTypedMasterDataIntegrationTargetRejectsCurrentLocalDatabase(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		rawURL  string
		allowCI bool
		wantErr bool
	}{
		{name: "dedicated test database", rawURL: "postgres://tidewise:secret@localhost:5432/tidewise_schema_test?sslmode=disable"},
		{name: "current local database", rawURL: "postgres://tidewise:secret@localhost:5432/tidewise_local?sslmode=disable", wantErr: true},
		{name: "disposable CI database", rawURL: "postgres://tidewise:secret@localhost:5432/tidewise_local?sslmode=disable", allowCI: true},
		{name: "remote uat database", rawURL: "postgres://tidewise:secret@uat-db.example.com:5432/tidewise_uat?sslmode=require", wantErr: true},
		{name: "loopback non-test database", rawURL: "postgres://tidewise:secret@localhost:5432/tidewise_shared?sslmode=disable", wantErr: true},
		{name: "missing database name", rawURL: "postgres://tidewise:secret@localhost:5432", wantErr: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			err := validateTypedMasterDataTestDatabaseURL(testCase.rawURL, testCase.allowCI)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("validateTypedMasterDataTestDatabaseURL() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func TestPostgresTypedMasterDataMigration(t *testing.T) {
	requireDisposableTypedMasterDataDatabase(t)
	db := openIsolatedMigrationDatabase(t)
	prepareTypedMasterDataBaseSchema(t, db)
	applyTypedMasterDataMigration(t, db)

	var (
		entityType   string
		entityKey    string
		definition   string
		boundaryNote sql.NullString
		reviewStatus sql.NullString
	)
	if err := db.QueryRow(`
SELECT n.entity_type, n.entity_key, p.definition, p.boundary_note, p.review_status
FROM entity_nodes n
JOIN chain_node_profiles p ON p.entity_id = n.id
WHERE n.id = $1`, typedSchemaID(1)).Scan(&entityType, &entityKey, &definition, &boundaryNote, &reviewStatus); err != nil {
		t.Fatal(err)
	}
	if entityType != "chain_node" || entityKey != "chain_node:legacy" || definition != "历史节点" || boundaryNote.Valid || reviewStatus.Valid {
		t.Fatalf("legacy row changed: type=%q key=%q definition=%q boundary=%v review=%v", entityType, entityKey, definition, boundaryNote, reviewStatus)
	}

	if _, err := db.Exec(`
UPDATE chain_node_profiles
SET definition = '仍允许维护历史定义'
WHERE entity_id = $1`, typedSchemaID(2)); err != nil {
		t.Fatalf("ordinary legacy profile update must remain compatible: %v", err)
	}
	expectPostgresStatementFailure(t, db, `
UPDATE chain_node_profiles
SET review_status = 'candidate'
WHERE entity_id = $1`, typedSchemaID(2))
	expectPostgresStatementFailure(t, db, `
UPDATE chain_node_profiles
SET review_status = 'approved'
WHERE entity_id = $1`, typedSchemaID(3))
	if _, err := db.Exec(`
UPDATE chain_node_profiles
SET review_status = 'candidate'
WHERE entity_id = $1`, typedSchemaID(1)); err != nil {
		t.Fatalf("valid legacy chain node must be promotable to candidate: %v", err)
	}
	if _, err := db.Exec(`
UPDATE chain_node_profiles
SET review_status = 'approved'
WHERE entity_id = $1`, typedSchemaID(1)); err != nil {
		t.Fatalf("valid legacy chain node must be promotable to approved: %v", err)
	}

	insertTypedSchemaEntity(t, db, 10, "industry", "industry:artificial_intelligence", "人工智能")
	insertTypedSchemaEntity(t, db, 11, "concept", "concept:artificial_intelligence", "人工智能")
	insertTypedSchemaEntity(t, db, 12, "chain_node", "chain_node:artificial_intelligence", "人工智能")
	expectPostgresStatementFailure(t, db, `
INSERT INTO entity_nodes (id, entity_type, entity_key, name)
VALUES ($1, 'concept', 'industry:artificial_intelligence', '重复全局键')`, typedSchemaID(13))

	if _, err := db.Exec(`
INSERT INTO industry_profiles (
    entity_id, classification_system, classification_version, industry_code,
    classification_level, parent_industry_entity_id, hierarchy_path_codes,
    definition, boundary_note, review_status
) VALUES ($1, 'sw', 'workbook_snapshot_v1', '801000', 1, NULL, ARRAY['801000'], '一级行业', '行业边界', 'approved')`, typedSchemaID(10)); err != nil {
		t.Fatal(err)
	}

	insertTypedSchemaEntity(t, db, 14, "industry", "industry:sw:workbook_snapshot_v1:801010", "二级行业")
	if _, err := db.Exec(`
INSERT INTO industry_profiles (
    entity_id, classification_system, classification_version, industry_code,
    classification_level, parent_industry_entity_id, hierarchy_path_codes,
    definition, boundary_note, review_status
) VALUES ($1, 'sw', 'workbook_snapshot_v1', '801010', 2, $2, ARRAY['801000','801010'], '二级行业', '二级边界', 'approved')`, typedSchemaID(14), typedSchemaID(10)); err != nil {
		t.Fatal(err)
	}
	expectPostgresStatementFailure(t, db, `
UPDATE industry_profiles
SET classification_level = 3,
    parent_industry_entity_id = entity_id,
    hierarchy_path_codes = ARRAY['801000','801010','801010']
WHERE entity_id = $1`, typedSchemaID(14))

	insertTypedSchemaEntity(t, db, 15, "industry", "industry:sw:workbook_snapshot_v1:801011", "三级行业")
	if _, err := db.Exec(`
INSERT INTO industry_profiles (
    entity_id, classification_system, classification_version, industry_code,
    classification_level, parent_industry_entity_id, hierarchy_path_codes,
    definition, boundary_note, review_status
) VALUES ($1, 'sw', 'workbook_snapshot_v1', '801011', 3, $2, ARRAY['801000','801010','801011'], '三级行业', '三级边界', 'candidate')`, typedSchemaID(15), typedSchemaID(14)); err != nil {
		t.Fatal(err)
	}

	insertTypedSchemaEntity(t, db, 16, "industry", "industry:invalid_parent", "错误行业")
	expectPostgresStatementFailure(t, db, `
INSERT INTO industry_profiles (
    entity_id, classification_system, classification_version, industry_code,
    classification_level, parent_industry_entity_id, hierarchy_path_codes,
    definition, boundary_note, review_status
	) VALUES ($1, 'sw', 'workbook_snapshot_v1', '801099', 3, $2, ARRAY['801000','801099'], '错误层级', '错误边界', 'approved')`, typedSchemaID(16), typedSchemaID(10))
	insertTypedSchemaEntity(t, db, 17, "industry", "industry:invalid_version", "错误版本行业")
	expectPostgresStatementFailure(t, db, `
INSERT INTO industry_profiles (
    entity_id, classification_system, classification_version, industry_code,
    classification_level, parent_industry_entity_id, hierarchy_path_codes,
    definition, boundary_note, review_status
) VALUES ($1, 'sw', 'another_snapshot', '801019', 2, $2, ARRAY['801000','801019'], '错误版本', '错误边界', 'approved')`, typedSchemaID(17), typedSchemaID(10))

	if _, err := db.Exec(`
INSERT INTO concept_profiles (entity_id, concept_type, definition, boundary_note, review_status)
VALUES ($1, 'technology', '跨行业技术聚合', '不是正式行业或产业链节点', 'approved')`, typedSchemaID(11)); err != nil {
		t.Fatal(err)
	}
	expectPostgresStatementFailure(t, db, `
INSERT INTO concept_profiles (entity_id, concept_type, definition, boundary_note, review_status)
VALUES ($1, 'technology', '错误类型', '错误边界', 'approved')`, typedSchemaID(12))

	if _, err := db.Exec(`
INSERT INTO chain_node_profiles (entity_id, definition, boundary_note, review_status)
VALUES ($1, '稳定投入产出节点', '不等同于人工智能概念', 'approved')`, typedSchemaID(12)); err != nil {
		t.Fatal(err)
	}
	expectPostgresStatementFailure(t, db, `UPDATE entity_nodes SET entity_type = 'concept' WHERE id = $1`, typedSchemaID(12))

	insertTypedSchemaEntity(t, db, 20, "industry_chain", "industry_chain:ai_compute", "AI 算力产业链")
	if _, err := db.Exec(`
INSERT INTO industry_chain_definitions (
    entity_id, scope, target_output, end_use, geography, as_of_date, review_status, review_note
) VALUES ($1, '从芯片制造到算力交付', '可用算力', 'AI 训练与推理', 'global_with_china_research_focus', DATE '2026-07-22', 'candidate', '待证据审核')`, typedSchemaID(20)); err != nil {
		t.Fatal(err)
	}
	expectPostgresStatementFailure(t, db, `
INSERT INTO industry_chain_definitions (
    entity_id, scope, target_output, end_use, geography, as_of_date, review_status
) VALUES ($1, '错误类型', '输出', '用途', 'global', DATE '2026-07-22', 'candidate')`, typedSchemaID(11))

	for _, item := range []struct {
		id       int
		key      string
		name     string
		position int
		stage    string
	}{
		{id: 21, key: "chain_node:wafer", name: "晶圆制造", position: 1, stage: "upstream"},
		{id: 22, key: "chain_node:accelerator", name: "AI 加速器", position: 2, stage: "midstream"},
		{id: 23, key: "chain_node:server", name: "AI 服务器", position: 2, stage: "downstream"},
	} {
		insertTypedSchemaEntity(t, db, item.id, "chain_node", item.key, item.name)
		if _, err := db.Exec(`
INSERT INTO chain_node_profiles (entity_id, definition, boundary_note, review_status)
VALUES ($1, $2, '稳定经济节点', 'approved')`, typedSchemaID(item.id), item.name+"的投入产出定义"); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`
INSERT INTO industry_chain_node_memberships (
    industry_chain_entity_id, chain_node_entity_id, position, contextual_stage, review_status
) VALUES ($1, $2, $3, $4, 'candidate')`, typedSchemaID(20), typedSchemaID(item.id), item.position, item.stage); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := db.Exec(`
INSERT INTO industry_chain_graph_edges (
    id, industry_chain_entity_id, from_chain_node_entity_id, to_chain_node_entity_id,
    relation_type, mechanism, segment_kind, review_status
) VALUES ($1, $2, $3, $4, 'input_to', '晶圆进入芯片生产', 'direct_candidate', 'candidate')`,
		typedSchemaID(30), typedSchemaID(20), typedSchemaID(21), typedSchemaID(22)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO industry_chain_graph_edges (
    id, industry_chain_entity_id, from_chain_node_entity_id, to_chain_node_entity_id,
    relation_type, mechanism, segment_kind, omitted_step_note, review_status
) VALUES ($1, $2, $3, $4, 'input_to', '芯片能力传导至服务器交付', 'compressed_candidate', '省略板卡集成环节', 'candidate')`,
		typedSchemaID(31), typedSchemaID(20), typedSchemaID(22), typedSchemaID(23)); err != nil {
		t.Fatal(err)
	}
	expectPostgresStatementFailure(t, db, `
INSERT INTO industry_chain_graph_edges (
    id, industry_chain_entity_id, from_chain_node_entity_id, to_chain_node_entity_id,
    relation_type, mechanism, segment_kind, review_status
) VALUES ($1, $2, $3, $4, 'depends_on', '形成循环', 'direct_candidate', 'candidate')`,
		typedSchemaID(32), typedSchemaID(20), typedSchemaID(23), typedSchemaID(21))
	expectPostgresStatementFailure(t, db, `
INSERT INTO industry_chain_graph_edges (
    id, industry_chain_entity_id, from_chain_node_entity_id, to_chain_node_entity_id,
    relation_type, mechanism, segment_kind, omitted_step_note, review_status
		) VALUES ($1, $2, $3, $4, 'input_to', '缺少压缩说明', 'compressed_candidate', NULL, 'candidate')`,
		typedSchemaID(33), typedSchemaID(20), typedSchemaID(21), typedSchemaID(23))
	expectPostgresStatementFailure(t, db, `
UPDATE industry_chain_node_memberships
SET status = 'inactive'
WHERE industry_chain_entity_id = $1 AND chain_node_entity_id = $2`, typedSchemaID(20), typedSchemaID(23))

	insertTypedSchemaEntity(t, db, 40, "concept", "concept:ai_infrastructure", "AI 基建")
	insertTypedSchemaEntity(t, db, 41, "concept", "concept:ai_compute", "AI 算力")
	if _, err := db.Exec(`
INSERT INTO entity_redirects (source_entity_id, target_entity_id, redirect_kind, reason, review_status)
VALUES ($1, $2, 'merge', '同类型重复概念', 'approved')`, typedSchemaID(40), typedSchemaID(41)); err != nil {
		t.Fatal(err)
	}
	expectPostgresStatementFailure(t, db, `
INSERT INTO entity_redirects (source_entity_id, target_entity_id, redirect_kind, reason, review_status)
VALUES ($1, $2, 'merge', '跨类型错误合并', 'approved')`, typedSchemaID(12), typedSchemaID(41))
	if _, err := db.Exec(`
INSERT INTO entity_redirects (source_entity_id, target_entity_id, redirect_kind, reason, review_status)
VALUES ($1, $2, 'reclassification', '历史节点重分类为概念', 'approved')`, typedSchemaID(12), typedSchemaID(41)); err != nil {
		t.Fatal(err)
	}
	expectPostgresStatementFailure(t, db, `UPDATE entity_nodes SET entity_type = 'industry' WHERE id = $1`, typedSchemaID(40))
	expectPostgresStatementFailure(t, db, `
INSERT INTO entity_redirects (source_entity_id, target_entity_id, redirect_kind, reason, review_status)
VALUES ($1, $2, 'merge', '形成循环', 'approved')`, typedSchemaID(41), typedSchemaID(40))
}

func requireTableColumns(t *testing.T, migrationSQL, tableName string, columnNames []string) {
	t.Helper()
	definition := migrationTableDefinition(t, migrationSQL, tableName)
	for _, columnName := range columnNames {
		columnPattern := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(columnName) + `\s+`)
		if !columnPattern.MatchString(definition) {
			t.Errorf("table %s must declare column %q", tableName, columnName)
		}
	}
}

func requireTableDefinitionFragments(t *testing.T, migrationSQL, tableName string, fragments []string) {
	t.Helper()
	definition := strings.Join(strings.Fields(migrationTableDefinition(t, migrationSQL, tableName)), " ")
	for _, fragment := range fragments {
		normalizedFragment := strings.Join(strings.Fields(fragment), " ")
		if !strings.Contains(definition, normalizedFragment) {
			t.Errorf("table %s must contain definition %q", tableName, normalizedFragment)
		}
	}
}

func migrationTableDefinition(t *testing.T, migrationSQL, tableName string) string {
	t.Helper()
	pattern := regexp.MustCompile(`(?s)create table\s+` + regexp.QuoteMeta(tableName) + `\s*\((.*?)\n\);`)
	match := pattern.FindStringSubmatch(migrationSQL)
	if len(match) != 2 {
		t.Fatalf("typed master data migration does not declare table %q", tableName)
	}
	return match[1]
}

func requireDisposableTypedMasterDataDatabase(t *testing.T) {
	t.Helper()
	rawURL := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if rawURL == "" {
		return
	}
	if err := validateTypedMasterDataTestDatabaseURL(rawURL, os.Getenv("CI") == "true"); err != nil {
		t.Fatal(err)
	}
}

func validateTypedMasterDataTestDatabaseURL(rawURL string, allowDisposableCI bool) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed.Scheme != "postgres" {
		return fmt.Errorf("typed master data integration test requires a valid postgres URL")
	}
	databaseName := strings.TrimPrefix(parsed.Path, "/")
	if databaseName == "" {
		return fmt.Errorf("typed master data integration test requires an explicit database name")
	}
	hostname := parsed.Hostname()
	address := net.ParseIP(hostname)
	if hostname != "localhost" && (address == nil || !address.IsLoopback()) {
		return fmt.Errorf("typed master data integration test requires a loopback database host")
	}
	if databaseName == "tidewise_local" {
		if allowDisposableCI {
			return nil
		}
		return fmt.Errorf("typed master data integration test refuses the current tidewise_local database")
	}
	if !strings.HasPrefix(databaseName, "tw_") && !strings.HasSuffix(databaseName, "_test") {
		return fmt.Errorf("typed master data integration test requires a dedicated test database name")
	}
	return nil
}

func prepareTypedMasterDataBaseSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, statement := range []string{
		`CREATE TABLE entity_nodes (
            id UUID PRIMARY KEY,
            entity_type TEXT NOT NULL,
            entity_key TEXT NOT NULL DEFAULT '',
            name TEXT NOT NULL,
            aliases TEXT[] NOT NULL DEFAULT '{}',
            status TEXT NOT NULL DEFAULT 'active'
        )`,
		`CREATE INDEX idx_entity_nodes_entity_key ON entity_nodes (entity_key)`,
		`CREATE TABLE chain_node_profiles (
            entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id),
            definition TEXT NOT NULL,
            boundary_note TEXT
        )`,
		`INSERT INTO entity_nodes (id, entity_type, entity_key, name)
         VALUES ('00000000-0000-4000-8000-000000000001', 'chain_node', 'chain_node:legacy', '历史节点')`,
		`INSERT INTO chain_node_profiles (entity_id, definition, boundary_note)
	         VALUES ('00000000-0000-4000-8000-000000000001', '历史节点', NULL)`,
		`INSERT INTO entity_nodes (id, entity_type, entity_key, name)
	         VALUES ('00000000-0000-4000-8000-000000000002', 'concept', 'concept:legacy_misclassified', '错误类型历史节点')`,
		`INSERT INTO chain_node_profiles (entity_id, definition, boundary_note)
	         VALUES ('00000000-0000-4000-8000-000000000002', '错误类型历史节点', NULL)`,
		`INSERT INTO entity_nodes (id, entity_type, entity_key, name)
	         VALUES ('00000000-0000-4000-8000-000000000003', 'chain_node', '', '空键历史节点')`,
		`INSERT INTO chain_node_profiles (entity_id, definition, boundary_note)
	         VALUES ('00000000-0000-4000-8000-000000000003', '空键历史节点', NULL)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := goose.EnsureDBVersionContext(context.Background(), db); err != nil {
		t.Fatal(err)
	}
}

func applyTypedMasterDataMigration(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	migrations, err := goose.CollectMigrations(migrationDirectory(), 26, 28)
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) != 2 {
		t.Fatalf("typed master data migrations = %d, want 2", len(migrations))
	}
	for _, migration := range migrations {
		if err := migration.UpContext(context.Background(), db); err != nil {
			t.Fatal(err)
		}
	}
}

func insertTypedSchemaEntity(t *testing.T, db *sql.DB, id int, entityType, entityKey, name string) {
	t.Helper()
	if _, err := db.Exec(`
INSERT INTO entity_nodes (id, entity_type, entity_key, name)
VALUES ($1, $2, $3, $4)`, typedSchemaID(id), entityType, entityKey, name); err != nil {
		t.Fatal(err)
	}
}

func typedSchemaID(value int) string {
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", value)
}
