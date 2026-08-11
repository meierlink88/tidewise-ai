package dbmigration

import (
	"testing"

	postgresfixture "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/testsupport/postgres"
)

func TestEvidencePublicationSchemaContract(t *testing.T) {
	db := postgresfixture.OpenIsolated(t, "tw_evidence_schema", migrationDirectory(), 42)

	var expressionIndexCount, expressionNonUniqueCount int
	if err := db.QueryRow(`
SELECT count(*), count(*) FILTER (WHERE NOT i.indisunique)
FROM pg_index i
JOIN pg_class c ON c.oid = i.indexrelid
WHERE c.relname = 'idx_evidences_expression_key'`).Scan(
		&expressionIndexCount,
		&expressionNonUniqueCount,
	); err != nil {
		t.Fatal(err)
	}
	if expressionIndexCount != 1 || expressionNonUniqueCount != 1 {
		t.Fatalf(
			"expression key index count=%d nonunique=%d",
			expressionIndexCount,
			expressionNonUniqueCount,
		)
	}

	var splitDefault, rawTextComment string
	var restrictFKs, quotedChecks int
	if err := db.QueryRow(`
SELECT column_default
FROM information_schema.columns
WHERE table_schema = current_schema()
  AND table_name = 'evidences'
  AND column_name = 'split_order'`).Scan(&splitDefault); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`
SELECT col_description('raw_evidences'::regclass, a.attnum)
FROM pg_attribute a
WHERE a.attrelid = 'raw_evidences'::regclass
  AND a.attname = 'raw_text'`).Scan(&rawTextComment); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`
SELECT count(*)
FROM pg_constraint
WHERE conrelid = 'evidences'::regclass
  AND contype = 'f'
  AND confdeltype = 'r'`).Scan(&restrictFKs); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`
SELECT count(*)
FROM pg_constraint
WHERE conrelid = 'raw_evidences'::regclass
  AND conname IN (
    'chk_raw_evidences_quoted_source_id',
    'chk_raw_evidences_quoted_source_name'
  )`).Scan(&quotedChecks); err != nil {
		t.Fatal(err)
	}
	if splitDefault != "0" ||
		rawTextComment != "原始文章完整正文。" ||
		restrictFKs != 1 ||
		quotedChecks != 2 {
		t.Fatalf(
			"Evidence schema default=%q comment=%q restrict_fks=%d quoted_checks=%d",
			splitDefault,
			rawTextComment,
			restrictFKs,
			quotedChecks,
		)
	}
}
