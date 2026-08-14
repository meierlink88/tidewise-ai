package country_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	eventsemanticdata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/eventsemantic"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestCountryMigrationRetiresLegacyEconomyReferencesAndPreservesLinkInvariants(t *testing.T) {
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	db := postgresfixture.OpenIsolated(t, "tw_country_cutover", migrationDir, 45)
	ctx := context.Background()
	const (
		economyID = "10000000-0000-4000-8000-000000000101"
		companyID = "10000000-0000-4000-8000-000000000102"
		eventID   = "10000000-0000-4000-8000-000000000103"
		linkID    = "10000000-0000-4000-8000-000000000105"
		signalID  = "10000000-0000-4000-8000-000000000108"
		impactID  = "10000000-0000-4000-8000-000000000109"
	)
	if _, err := db.ExecContext(ctx, `
INSERT INTO entity_nodes (id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status)
VALUES
    ($1, 'economy:test', 'economy', 'economy', 'Legacy Economy', 'Legacy Economy', '{}', 'active'),
	($2, 'company:test', 'company', 'company', 'Test Company', 'Test Company', '{}', 'active')`, economyID, companyID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO economy_profiles (entity_id, country_code, currency_code, region)
VALUES ($1, 'TST', 'TST', 'legacy')`, economyID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO entity_edges (id, from_entity_id, to_entity_id, relation_type)
VALUES ('10000000-0000-4000-8000-000000000104', $1, $2, 'legacy_relation')`, economyID, companyID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO events (id, title, summary, first_seen_at, event_status, fact_status, dedupe_key, fact_payload)
VALUES ($1, 'Legacy Economy event', '', now(), 'confirmed', 'verified', 'country-cutover:test', '{}')`, eventID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO event_entity_links (
    id, event_id, entity_id, entity_role, assign_source, review_status, evidence_note, provenance
) VALUES (
	$3, $2, $1, 'subject', 'manual', 'accepted', '', 'legacy'
)`, economyID, eventID, linkID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO event_semantic_context_leases (
    id, event_id, agent_execution_id, worker_id, status, lease_expires_at, context_snapshot
) VALUES (
    '10000000-0000-4000-8000-000000000110', $1,
    'country-cutover-agent', 'country-cutover-worker', 'consumed', now() + interval '1 hour', '{}'
)`, eventID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO event_semantic_submissions (
    id, context_lease_id, event_id, agent_execution_id, agent_key, agent_version,
    generator_prompt_hash, generator_model, reviewer_prompt_hash, reviewer_model,
    ontology_version, acceptance_policy_key, acceptance_policy_version,
    canonical_payload_hash, status, finalized_at
) VALUES (
    '10000000-0000-4000-8000-000000000111',
    '10000000-0000-4000-8000-000000000110', $1,
    'country-cutover-agent', 'country-cutover', '1',
    repeat('a', 64), 'test-model', repeat('b', 64), 'test-reviewer',
    'test-ontology', 'event-semantics.objective-v2', 1, repeat('c', 64), 'accepted', now()
)`, eventID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
UPDATE event_entity_links
SET provenance = 'semantic',
    semantic_submission_id = '10000000-0000-4000-8000-000000000111',
    candidate_key = 'country-cutover-link',
    evidence_ids = ARRAY['10000000-0000-4000-8000-000000000112'::uuid]
WHERE id = $1`, linkID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO event_semantic_candidate_snapshots (
    id, semantic_submission_id, payload, canonical_payload_hash
) VALUES (
    '10000000-0000-4000-8000-000000000113',
    '10000000-0000-4000-8000-000000000111', '{}', repeat('c', 64)
)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO event_semantic_review_snapshots (
    id, semantic_submission_id, reviewer_execution_key, payload, canonical_payload_hash
) VALUES (
    '10000000-0000-4000-8000-000000000114',
    '10000000-0000-4000-8000-000000000111', 'country-cutover-review', '{}', repeat('d', 64)
)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO variable_signals (
    id, semantic_submission_id, candidate_key, source_event_id,
    subject_event_entity_link_id, variable_key, variable_version, direction,
    assertion_modality, evidence_ids, review_status
) VALUES (
    $1, '10000000-0000-4000-8000-000000000111', 'country-cutover-signal', $2,
    $3, 'market_supply', 1, 'increase', 'actual',
    ARRAY['10000000-0000-4000-8000-000000000112'::uuid], 'accepted'
)`, signalID, eventID, linkID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO direct_impact_assertions (
    id, semantic_submission_id, candidate_key, source_variable_signal_id,
    target_entity_id, affected_variable_key, affected_variable_version,
    affected_direction, derivation_type, mechanism_summary, evidence_ids,
    entity_relation_id, review_status
) VALUES (
    $1, '10000000-0000-4000-8000-000000000111', 'country-cutover-impact', $2,
    $3, 'market_demand', 1, 'increase', 'event_explicit', 'legacy impact',
    ARRAY['10000000-0000-4000-8000-000000000112'::uuid],
    '10000000-0000-4000-8000-000000000104', 'accepted'
)`, impactID, signalID, companyID); err != nil {
		t.Fatal(err)
	}

	postgresfixture.ApplyMigration(t, db, migrationDir, 46)

	for label, query := range map[string]string{
		"Economy Entity":      `SELECT count(*) FROM entity_nodes WHERE entity_type = 'economy'`,
		"Economy profile":     `SELECT count(*) FROM pg_class WHERE oid = to_regclass('economy_profiles')`,
		"Economy edge":        `SELECT count(*) FROM entity_edges WHERE from_entity_id = '` + economyID + `' OR to_entity_id = '` + economyID + `'`,
		"Economy Event link":  `SELECT count(*) FROM event_entity_links WHERE entity_id = '` + economyID + `'`,
		"Economy Signal":      `SELECT count(*) FROM variable_signals WHERE id = '` + signalID + `'`,
		"Economy Impact":      `SELECT count(*) FROM direct_impact_assertions WHERE id = '` + impactID + `'`,
		"Semantic submission": `SELECT count(*) FROM event_semantic_submissions WHERE id = '10000000-0000-4000-8000-000000000111'`,
		"Candidate snapshot":  `SELECT count(*) FROM event_semantic_candidate_snapshots WHERE semantic_submission_id = '10000000-0000-4000-8000-000000000111'`,
		"Review snapshot":     `SELECT count(*) FROM event_semantic_review_snapshots WHERE semantic_submission_id = '10000000-0000-4000-8000-000000000111'`,
	} {
		var count int
		if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
			t.Fatalf("query %s: %v", label, err)
		}
		if count != 0 {
			t.Fatalf("%s count = %d, want 0", label, count)
		}
	}
	eventSemanticStore, err := eventsemanticdata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	semanticResult, err := eventSemanticStore.GetEventSemantics(ctx, eventID)
	if err != nil {
		t.Fatalf("read Event Semantics after Economy cutover: %v", err)
	}
	if len(semanticResult.Submissions) != 0 {
		t.Fatalf("retired Event Semantic submissions = %#v", semanticResult.Submissions)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO countries (id, code, name, name_en)
VALUES ('COU_TST', 'TST', '测试国', 'Test Country')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO event_entity_links (
    id, event_id, entity_id, country_id, entity_role, assign_source, review_status, evidence_note, provenance
) VALUES (
    '10000000-0000-4000-8000-000000000106', $1, NULL, 'COU_TST', 'subject', 'manual', 'accepted', '', 'legacy'
)`, eventID); err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `
INSERT INTO event_entity_links (
    id, event_id, entity_id, country_id, entity_role, assign_source, review_status, evidence_note, provenance
) VALUES (
    '10000000-0000-4000-8000-000000000107', $1, NULL, 'COU_TST', 'subject', 'manual', 'accepted', '', 'legacy'
)`, eventID)
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) || postgresError.Code != "23505" {
		t.Fatalf("duplicate accepted Country link error = %v, want PostgreSQL 23505", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM countries WHERE id = 'COU_TST'`); err == nil {
		t.Fatal("deleting a referenced Country succeeded")
	}
}
