package data

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

type objectSchemaContract struct {
	name          string
	schemaFile    string
	table         string
	schemaMembers []string
	columns       []schemaColumn
	enums         []schemaEnum
	constraints   []schemaConstraint
}

type schemaColumn struct {
	name            string
	nullable        string
	dataType        string
	maxLength       int64
	defaultContains string
}

type schemaEnum struct {
	name   string
	values []string
}

type schemaConstraint struct {
	name           string
	requiredTokens []string
}

func TestPublishedObjectSchemasAndPersistenceStayAligned(t *testing.T) {
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	schemaDir, err := filepath.Abs(filepath.Join("..", "..", "..", "doctype"))
	if err != nil {
		t.Fatal(err)
	}
	db := postgresfixture.OpenIsolated(t, "tw_object_schema", migrationDir, 0)

	contracts := []objectSchemaContract{
		{
			name:       "Evidence",
			schemaFile: "evidence.schema",
			table:      "evidences",
			schemaMembers: []string{
				"Evidence(原子证据): EntityType",
				"rawEvidenceId(所属原始证据标识): Text",
				"isSplit(是否拆分): Text",
				"summary(事实摘要): Text",
				"keywords(关键词): Text",
				"actors(业务主体): Text",
				"action(核心动作): Text",
				"objects(作用对象): Text",
				"stage(现实阶段): Text",
				"modality(事实情态): Text",
				"time(语义时间): Text",
				"jurisdictions(作用辖区): Text",
				"reason(发生原因): Text",
				"method(执行方式): Text",
				"metrics(业务指标): Text",
				"attribution(信息归因): Text",
				"createdAt(创建时间): Text",
				`Enum="TRUE,FALSE"`,
				`Enum="OCCURRED,ANNOUNCED,EFFECTIVE,IMPLEMENTED,UPDATED,SUSPENDED,TERMINATED,EXPECTED"`,
				`Enum="FACT,PLAN,SPEC"`,
			},
			columns: []schemaColumn{
				{name: "id", nullable: "NO", dataType: "varchar", maxLength: 39},
				{name: "raw_evidence_id", nullable: "NO", dataType: "varchar", maxLength: 39},
				{name: "is_split", nullable: "NO", dataType: "bool"},
				{name: "summary", nullable: "NO", dataType: "varchar", maxLength: 200},
				{name: "keywords", nullable: "NO", dataType: "_text"},
				{name: "created_at", nullable: "YES", dataType: "timestamptz", defaultContains: "transaction_timestamp()"},
				{name: "semantic", nullable: "NO", dataType: "jsonb"},
			},
			constraints: []schemaConstraint{
				{name: "chk_evidences_created_at_new_rows", requiredTokens: []string{"created_at IS NOT NULL", "NOT VALID"}},
				{name: "chk_evidences_domain_identity", requiredTokens: []string{"^EVD[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"}},
				{name: "chk_evidences_id", requiredTokens: []string{"btrim(id::text) <> ''::text"}},
				{name: "chk_evidences_keywords", requiredTokens: []string{"array_ndims(keywords)", "cardinality(keywords) >= 1", "cardinality(keywords) <= 5", "array_position(keywords, NULL::text) IS NULL"}},
				{name: "chk_evidences_semantic", requiredTokens: evidenceSemanticConstraintTokens()},
				{name: "chk_evidences_summary", requiredTokens: []string{"btrim(summary::text) <> ''::text"}},
				{name: "evidences_pkey", requiredTokens: []string{"PRIMARY KEY (id)"}},
				{name: "evidences_raw_evidence_id_fkey", requiredTokens: []string{"FOREIGN KEY (raw_evidence_id)", "REFERENCES raw_evidences(id)", "ON DELETE RESTRICT"}},
			},
		},
		{
			name:       "GeopoliticDomain",
			schemaFile: "geopolitic-domain.schema",
			table:      "geopolitic_domains",
			schemaMembers: []string{
				"GeopoliticDomain(地缘政治领域): EntityType",
				"code(领域代码): Text",
				"name(领域中文名称): Text",
				"description(领域描述): Text",
				"tactics(手段数组): Text",
				"createdAt(创建时间): Text",
				"updatedAt(更新时间): Text",
			},
			columns: []schemaColumn{
				{name: "id", nullable: "NO", dataType: "varchar", maxLength: 39},
				{name: "code", nullable: "NO", dataType: "varchar", maxLength: 50},
				{name: "name", nullable: "NO", dataType: "varchar", maxLength: 50},
				{name: "description", nullable: "NO", dataType: "text"},
				{name: "tactics", nullable: "NO", dataType: "jsonb"},
				{name: "created_at", nullable: "NO", dataType: "timestamptz", defaultContains: "now()"},
				{name: "updated_at", nullable: "NO", dataType: "timestamptz", defaultContains: "now()"},
			},
			constraints: []schemaConstraint{
				{name: "chk_geopolitic_domains_identity", requiredTokens: []string{"^GPD[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"}},
				{name: "chk_geopolitic_domains_code", requiredTokens: []string{"^[A-Z][A-Z0-9_]{0,49}$"}},
				{name: "chk_geopolitic_domains_required_text", requiredTokens: []string{"btrim(name::text) <> ''::text", "btrim(description) <> ''::text"}},
				{name: "chk_geopolitic_domains_tactics", requiredTokens: []string{"validate_geopolitic_domain_tactics(tactics)"}},
				{name: "chk_geopolitic_domains_timestamp_order", requiredTokens: []string{"updated_at >= created_at"}},
				{name: "geopolitic_domains_pkey", requiredTokens: []string{"PRIMARY KEY (id)"}},
				{name: "geopolitic_domains_code_key", requiredTokens: []string{"UNIQUE (code)"}},
				{name: "geopolitic_domains_name_key", requiredTokens: []string{"UNIQUE (name)"}},
			},
		},
		{
			name:       "GeopoliticRivalry",
			schemaFile: "geopolitic-rivalry.schema",
			table:      "geopolitic_rivalries",
			schemaMembers: []string{
				"GeopoliticRivalry(地缘政治故事线): EntityType",
				"name(故事线中文名称): Text",
				"category(故事线分类): Text",
				"geopoliticDomainId(领域标识): Text",
				"coreProposition(核心命题): Text",
				"coreActors(核心参与方): Text",
				"mainTransmission(主要传导): Text",
				"createdAt(创建时间): Text",
				"updatedAt(更新时间): Text",
			},
			columns: []schemaColumn{
				{name: "id", nullable: "NO", dataType: "varchar", maxLength: 39},
				{name: "name", nullable: "NO", dataType: "varchar", maxLength: 100},
				{name: "category", nullable: "NO", dataType: "varchar", maxLength: 100},
				{name: "geopolitic_domain_id", nullable: "NO", dataType: "varchar", maxLength: 39},
				{name: "core_proposition", nullable: "NO", dataType: "text"},
				{name: "core_actors", nullable: "NO", dataType: "text"},
				{name: "main_transmission", nullable: "NO", dataType: "text"},
				{name: "created_at", nullable: "NO", dataType: "timestamptz", defaultContains: "now()"},
				{name: "updated_at", nullable: "NO", dataType: "timestamptz", defaultContains: "now()"},
			},
			constraints: []schemaConstraint{
				{name: "chk_geopolitic_rivalries_identity", requiredTokens: []string{"^GPR[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"}},
				{name: "chk_geopolitic_rivalries_required_text", requiredTokens: []string{"btrim(name::text) <> ''::text", "btrim(category::text) <> ''::text", "btrim(core_proposition) <> ''::text", "btrim(core_actors) <> ''::text", "btrim(main_transmission) <> ''::text"}},
				{name: "chk_geopolitic_rivalries_timestamp_order", requiredTokens: []string{"updated_at >= created_at"}},
				{name: "geopolitic_rivalries_pkey", requiredTokens: []string{"PRIMARY KEY (id)"}},
				{name: "geopolitic_rivalries_name_key", requiredTokens: []string{"UNIQUE (name)"}},
				{name: "geopolitic_rivalries_geopolitic_domain_id_fkey", requiredTokens: []string{"FOREIGN KEY (geopolitic_domain_id)", "REFERENCES geopolitic_domains(id)", "ON DELETE RESTRICT"}},
			},
		},
		{
			name:       "MacroEconomic",
			schemaFile: "macro-economic.schema",
			table:      "macro_economics",
			schemaMembers: []string{
				"MacroEconomic(宏观经济叙事蓝图): EntityType",
				"name(蓝图中文名称): Text",
				"nameEn(蓝图英文名称): Text",
				"macroType(宏观类型): Text",
				"description(蓝图说明): Text",
				"status(生命周期状态): Text",
				"createdAt(创建时间): Text",
				"updatedAt(更新时间): Text",
				`Enum="MONETARY,FISCAL,TRADE_POLICY,REGULATORY,DATA_ECONOMIC"`,
				`Enum="ACTIVE,DORMANT,ARCHIVED"`,
			},
			columns: []schemaColumn{
				{name: "id", nullable: "NO", dataType: "varchar", maxLength: 39},
				{name: "name", nullable: "NO", dataType: "varchar", maxLength: 100},
				{name: "name_en", nullable: "NO", dataType: "varchar", maxLength: 100},
				{name: "macro_type", nullable: "NO", dataType: "macro_economic_type"},
				{name: "description", nullable: "NO", dataType: "text"},
				{name: "status", nullable: "NO", dataType: "macro_economic_status", defaultContains: "ACTIVE"},
				{name: "created_at", nullable: "NO", dataType: "timestamptz", defaultContains: "now()"},
				{name: "updated_at", nullable: "NO", dataType: "timestamptz", defaultContains: "now()"},
			},
			enums: []schemaEnum{
				{name: "macro_economic_type", values: []string{"MONETARY", "FISCAL", "TRADE_POLICY", "REGULATORY", "DATA_ECONOMIC"}},
				{name: "macro_economic_status", values: []string{"ACTIVE", "DORMANT", "ARCHIVED"}},
			},
			constraints: []schemaConstraint{
				{name: "chk_macro_economics_identity", requiredTokens: []string{"^MEC[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"}},
				{name: "chk_macro_economics_required_text", requiredTokens: []string{"btrim(name::text) <> ''::text", "btrim(name_en::text) <> ''::text", "btrim(description) <> ''::text"}},
				{name: "macro_economics_pkey", requiredTokens: []string{"PRIMARY KEY (id)"}},
			},
		},
	}

	for _, contract := range contracts {
		t.Run(contract.name, func(t *testing.T) {
			assertObjectSchemaMembers(t, filepath.Join(schemaDir, contract.schemaFile), contract.schemaMembers)
			assertObjectSchemaColumns(t, db, contract.table, contract.columns)
			assertObjectSchemaEnums(t, db, contract.enums)
			assertObjectSchemaConstraints(t, db, contract.table, contract.constraints)
		})
	}
}

func TestEvidenceSemanticCutoverRemovesLegacyEvidenceView(t *testing.T) {
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	db := postgresfixture.OpenIsolated(t, "tw_evidence_cutover", migrationDir, 73)
	if _, err := db.ExecContext(context.Background(), `
CREATE VIEW evidence_view AS
SELECT evidence.id AS evidence_id, raw.keywords
FROM evidences AS evidence
JOIN raw_evidences AS raw ON raw.id = evidence.raw_evidence_id`); err != nil {
		t.Fatal(err)
	}

	postgresfixture.ApplyMigration(t, db, migrationDir, 74)

	var legacyViewExists bool
	if err := db.QueryRowContext(context.Background(), `SELECT to_regclass(current_schema() || '.evidence_view') IS NOT NULL`).Scan(&legacyViewExists); err != nil {
		t.Fatal(err)
	}
	if legacyViewExists {
		t.Fatal("legacy evidence_view still exists after Evidence semantic cutover")
	}
	for table, want := range map[string]bool{"raw_evidences": false, "evidences": true} {
		var exists bool
		if err := db.QueryRowContext(context.Background(), `
SELECT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = current_schema() AND table_name = $1 AND column_name = 'keywords'
)`, table).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if exists != want {
			t.Fatalf("%s.keywords exists = %t, want %t", table, exists, want)
		}
	}
}

func TestCurrentGeopoliticalSchemaRetiresLegacyStorylineObjects(t *testing.T) {
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	db := postgresfixture.OpenIsolated(t, "tw_geopolitical_retirement", migrationDir, 0)

	for _, table := range []string{
		"storyline_domain_tactics",
		"storyline_domains",
		"storyline_event_links",
		"storylines",
	} {
		var exists bool
		if err := db.QueryRowContext(context.Background(), `
SELECT to_regclass(current_schema() || '.' || $1) IS NOT NULL`, table).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if exists {
			t.Errorf("retired table %q still exists", table)
		}
	}

	for _, typeName := range []string{
		"storyline_data_alignment_status",
		"storyline_status",
		"storyline_type",
		"storyline_domain_category",
		"geopolitic_rivalry_status",
		"geopolitic_rivalry_type",
	} {
		var exists bool
		if err := db.QueryRowContext(context.Background(), `
SELECT EXISTS (
    SELECT 1
    FROM pg_type
    JOIN pg_namespace ON pg_namespace.oid = pg_type.typnamespace
    WHERE pg_namespace.nspname = current_schema() AND pg_type.typname = $1
)`, typeName).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if exists {
			t.Errorf("retired type %q still exists", typeName)
		}
	}
}

func evidenceSemanticConstraintTokens() []string {
	return []string{
		"jsonb_typeof(semantic) = 'object'::text",
		"semantic = jsonb_build_object(",
		"jsonb_typeof(semantic -> 'actors'::text) = 'array'::text",
		"jsonb_typeof(semantic -> 'action'::text) = 'string'::text",
		"btrim(semantic ->> 'action'::text) <> ''::text",
		"jsonb_typeof(semantic -> 'objects'::text) = 'array'::text",
		"semantic ->> 'stage'::text",
		"'OCCURRED'::text",
		"'EXPECTED'::text",
		"semantic ->> 'modality'::text",
		"'FACT'::text",
		"'SPEC'::text",
		"jsonb_typeof(semantic -> 'time'::text) = 'object'::text",
		"jsonb_typeof(semantic -> 'jurisdictions'::text) = 'array'::text",
		"jsonb_typeof(semantic -> 'metrics'::text) = 'array'::text",
		"jsonb_typeof(semantic -> 'attribution'::text) = 'object'::text",
	}
}

func assertObjectSchemaMembers(t *testing.T, schemaPath string, members []string) {
	t.Helper()
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	source := string(schema)
	for _, member := range members {
		if !strings.Contains(source, member) {
			t.Fatalf("Object Schema %s is missing %q", schemaPath, member)
		}
	}
}

func assertObjectSchemaColumns(t *testing.T, db *sql.DB, table string, want []schemaColumn) {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), `
SELECT column_name, is_nullable, udt_name, character_maximum_length, column_default
FROM information_schema.columns
WHERE table_schema = current_schema() AND table_name = $1
ORDER BY ordinal_position`, table)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	got := make(map[string]schemaColumn, len(want))
	for rows.Next() {
		var column schemaColumn
		var maxLength sql.NullInt64
		var defaultSQL sql.NullString
		if err := rows.Scan(&column.name, &column.nullable, &column.dataType, &maxLength, &defaultSQL); err != nil {
			t.Fatal(err)
		}
		if maxLength.Valid {
			column.maxLength = maxLength.Int64
		}
		if defaultSQL.Valid {
			column.defaultContains = defaultSQL.String
		}
		got[column.name] = column
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("%s columns = %#v, want %#v", table, got, want)
	}
	for _, expected := range want {
		actual, ok := got[expected.name]
		if !ok {
			t.Fatalf("%s is missing column %s", table, expected.name)
		}
		if actual.nullable != expected.nullable || actual.dataType != expected.dataType || actual.maxLength != expected.maxLength {
			t.Fatalf("%s.%s = %#v, want %#v", table, expected.name, actual, expected)
		}
		if expected.defaultContains == "" && actual.defaultContains != "" {
			t.Fatalf("%s.%s default = %q, want none", table, expected.name, actual.defaultContains)
		}
		if expected.defaultContains != "" && !strings.Contains(actual.defaultContains, expected.defaultContains) {
			t.Fatalf("%s.%s default = %q, want token %q", table, expected.name, actual.defaultContains, expected.defaultContains)
		}
	}
}

func assertObjectSchemaEnums(t *testing.T, db *sql.DB, enums []schemaEnum) {
	t.Helper()
	for _, enum := range enums {
		rows, err := db.QueryContext(context.Background(), `
SELECT enumlabel
FROM pg_enum
JOIN pg_type ON pg_type.oid = pg_enum.enumtypid
JOIN pg_namespace ON pg_namespace.oid = pg_type.typnamespace
WHERE pg_type.typname = $1 AND pg_namespace.nspname = current_schema()
ORDER BY enumsortorder`, enum.name)
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for rows.Next() {
			var value string
			if err := rows.Scan(&value); err != nil {
				_ = rows.Close()
				t.Fatal(err)
			}
			got = append(got, value)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			t.Fatal(err)
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, enum.values) {
			t.Fatalf("%s enum = %q, want %q", enum.name, got, enum.values)
		}
	}
}

func assertObjectSchemaConstraints(t *testing.T, db *sql.DB, table string, want []schemaConstraint) {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), `
SELECT pg_constraint.conname, pg_get_constraintdef(pg_constraint.oid, true)
FROM pg_constraint
JOIN pg_class ON pg_class.oid = pg_constraint.conrelid
JOIN pg_namespace ON pg_namespace.oid = pg_class.relnamespace
WHERE pg_namespace.nspname = current_schema() AND pg_class.relname = $1
ORDER BY pg_constraint.conname`, table)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	got := make(map[string]string, len(want))
	for rows.Next() {
		var name, definition string
		if err := rows.Scan(&name, &definition); err != nil {
			t.Fatal(err)
		}
		got[name] = definition
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("%s constraints = %#v, want %#v", table, got, want)
	}
	for _, constraint := range want {
		definition, ok := got[constraint.name]
		if !ok {
			t.Fatalf("%s is missing constraint %s", table, constraint.name)
		}
		for _, token := range constraint.requiredTokens {
			if !strings.Contains(definition, token) {
				t.Fatalf("%s.%s = %q, want token %q", table, constraint.name, definition, token)
			}
		}
	}
}
