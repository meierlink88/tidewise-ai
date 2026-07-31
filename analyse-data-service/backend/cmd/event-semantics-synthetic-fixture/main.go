package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/conf"
)

const syntheticDatabasePrefix = "tw_semantic_acceptance_"

func main() {
	action := flag.String("action", "", "create-database, drop-database, seed, assert-empty-research, or assert-research-publication")
	baseURL := flag.String("base-url", "", "loopback PostgreSQL URL for database lifecycle actions")
	targetDatabase := flag.String("target-database", "", "isolated database name for drop-database")
	flag.Parse()

	if os.Getenv("EVENT_SEMANTICS_SYNTHETIC_FIXTURE") != "1" {
		log.Fatal("EVENT_SEMANTICS_SYNTHETIC_FIXTURE=1 is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var err error
	switch *action {
	case "create-database":
		err = createDatabase(ctx, *baseURL)
	case "drop-database":
		err = dropDatabase(ctx, *baseURL, *targetDatabase)
	case "seed":
		err = seed(ctx)
	case "assert-empty-research":
		err = assertEmptyResearch(ctx)
	case "assert-research-publication":
		err = assertResearchPublication(ctx)
	default:
		err = errors.New("unsupported fixture action")
	}
	if err != nil {
		log.Fatal(err)
	}
}

func createDatabase(ctx context.Context, baseURL string) error {
	parsed, err := validateLoopbackDatabaseURL(baseURL)
	if err != nil {
		return err
	}
	database, err := pgxpool.New(ctx, parsed.String())
	if err != nil {
		return err
	}
	defer database.Close()
	databaseName := syntheticDatabasePrefix + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := database.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{databaseName}.Sanitize()); err != nil {
		return err
	}
	parsed.Path = "/" + databaseName
	return json.NewEncoder(os.Stdout).Encode(map[string]string{
		"database_name": databaseName,
		"database_url":  parsed.String(),
	})
}

func dropDatabase(ctx context.Context, baseURL string, databaseName string) error {
	parsed, err := validateLoopbackDatabaseURL(baseURL)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(databaseName, syntheticDatabasePrefix) {
		return errors.New("target database is outside the synthetic fixture namespace")
	}
	database, err := pgxpool.New(ctx, parsed.String())
	if err != nil {
		return err
	}
	defer database.Close()
	_, err = database.Exec(
		ctx, "DROP DATABASE "+pgx.Identifier{databaseName}.Sanitize()+" WITH (FORCE)",
	)
	return err
}

func validateLoopbackDatabaseURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return nil, errors.New("fixture database URL must be PostgreSQL")
	}
	switch parsed.Hostname() {
	case "localhost", "127.0.0.1", "::1":
	default:
		return nil, errors.New("fixture database URL must use a loopback host")
	}
	if parsed.User == nil || parsed.Path == "" || parsed.Path == "/" {
		return nil, errors.New("fixture database URL must include credentials and database")
	}
	return parsed, nil
}

func openFixtureDatabase(ctx context.Context) (*pgxpool.Pool, error) {
	cfg, err := conf.LoadDatabaseOperation()
	if err != nil {
		return nil, err
	}
	if cfg.App.Env != conf.EnvLocal ||
		cfg.Database.Host != "localhost" && cfg.Database.Host != "127.0.0.1" &&
			cfg.Database.Host != "::1" ||
		!strings.HasPrefix(cfg.Database.Name, syntheticDatabasePrefix) {
		return nil, errors.New("fixture command requires an isolated local Data database")
	}
	dsn, err := cfg.PostgresURL()
	if err != nil {
		return nil, err
	}
	return pgxpool.New(ctx, dsn)
}

func seed(ctx context.Context) error {
	database, err := openFixtureDatabase(ctx)
	if err != nil {
		return err
	}
	defer database.Close()
	if err := seedEntities(ctx, database); err != nil {
		return err
	}
	if err := seedEvent(
		ctx, database,
		"20000000-0000-4000-8000-000000000001",
		"20000000-0000-4000-8000-000000000002",
		"20000000-0000-4000-8000-000000000003",
		"synthetic:accepted", "2026-07-28T08:00:00Z",
	); err != nil {
		return err
	}
	if err := seedEvent(
		ctx, database,
		"21000000-0000-4000-8000-000000000001",
		"21000000-0000-4000-8000-000000000002",
		"21000000-0000-4000-8000-000000000003",
		"synthetic:quarantined", "2026-07-28T09:00:00Z",
	); err != nil {
		return err
	}
	if err := seedEvent(
		ctx, database,
		"24000000-0000-4000-8000-000000000001",
		"24000000-0000-4000-8000-000000000002",
		"24000000-0000-4000-8000-000000000003",
		"synthetic:forecast-no-impact", "2026-07-28T10:00:00Z",
	); err != nil {
		return err
	}
	_, err = database.Exec(ctx, `
UPDATE raw_documents
SET title = 'Synthetic demand forecast',
    content_text = 'Synthetic Wafer Fab forecasts wafer demand growth of 12 percent'
WHERE id = '24000000-0000-4000-8000-000000000001';
UPDATE events
SET title = 'Synthetic Wafer Fab forecasts demand growth',
    summary = 'Synthetic Wafer Fab forecasts wafer demand growth of 12 percent.',
    fact_payload = jsonb_build_object(
        'statement_source', 'Synthetic Wafer Fab',
        'forecast_demand_change_percent', 12
    )
WHERE id = '24000000-0000-4000-8000-000000000002';
UPDATE event_sources
SET evidence_excerpt = 'Synthetic Wafer Fab forecasts wafer demand growth of 12 percent',
    supports_fields = ARRAY['title','factual_summary','occurred_at','fact_payload']
WHERE id = '24000000-0000-4000-8000-000000000003'
`)
	return err
}

func seedEntities(ctx context.Context, database *pgxpool.Pool) error {
	if _, err := database.Exec(ctx, `
INSERT INTO entity_nodes (
  id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status
) VALUES
  ('22000000-0000-4000-8000-000000000001', 'company:synthetic-wafer-fab',
   'chain_node', 'chain_node', 'Synthetic Wafer Production', 'Synthetic Wafer Production',
   ARRAY['Synthetic Wafer Fab'], 'active'),
  ('22000000-0000-4000-8000-000000000002', 'product:synthetic-wafer',
   'chain_node', 'chain_node', 'Synthetic 8-inch Wafer Supply', 'Synthetic 8-inch Wafer Supply',
   ARRAY['Synthetic Wafer'], 'active'),
  ('23000000-0000-4000-8000-000000000001', 'industry-chain:synthetic-wafer',
   'industry_chain', 'industry_chain', 'Synthetic Wafer Chain', 'Synthetic Wafer Chain',
   ARRAY['Synthetic Chain'], 'active'),
  ('23000000-0000-4000-8000-000000000002', 'industry:synthetic-semiconductor',
   'industry', 'industry', 'Synthetic Semiconductor Manufacturing',
   'Synthetic Semiconductor Manufacturing', ARRAY['Synthetic Manufacturing'], 'active')
`); err != nil {
		return err
	}
	if _, err := database.Exec(ctx, `
INSERT INTO chain_node_profiles (entity_id, definition, boundary_note, review_status) VALUES
  ('22000000-0000-4000-8000-000000000001', 'Synthetic wafer producer', NULL, 'approved'),
  ('22000000-0000-4000-8000-000000000002', 'Synthetic wafer product supply', NULL, 'approved');
INSERT INTO industry_profiles (
  entity_id, classification_system, classification_version, industry_code,
  classification_level, hierarchy_path_codes, definition, boundary_note, review_status
) VALUES (
  '23000000-0000-4000-8000-000000000002', 'synthetic', 'v1', 'S01', 1,
  ARRAY['S01'], 'Synthetic semiconductor manufacturing', 'Synthetic fixture only', 'approved'
);
INSERT INTO industry_chain_definitions (
  entity_id, scope, target_output, end_use, geography, as_of_date, review_status,
  observable_variables
) VALUES (
  '23000000-0000-4000-8000-000000000001',
  'Synthetic acceptance scope', 'Synthetic 8-inch Wafer', 'Semiconductor manufacturing',
  'Global', '2026-07-28', 'approved', ARRAY['production_volume','market_supply']
);
INSERT INTO industry_chain_node_memberships (
  industry_chain_entity_id, chain_node_entity_id, position, contextual_stage,
  review_status, status, inclusion_reason, evidence_ids, source_name, source_url, verified_at
) VALUES
  ('23000000-0000-4000-8000-000000000001',
   '22000000-0000-4000-8000-000000000001', 1, 'upstream', 'approved', 'active',
   'Synthetic producer node', ARRAY['synthetic-membership-1'], 'Synthetic Fixture',
   'artifact://synthetic/membership/1', '2026-07-28T00:00:00Z'),
  ('23000000-0000-4000-8000-000000000001',
   '22000000-0000-4000-8000-000000000002', 2, 'midstream', 'approved', 'active',
   'Synthetic supply node', ARRAY['synthetic-membership-2'], 'Synthetic Fixture',
   'artifact://synthetic/membership/2', '2026-07-28T00:00:00Z')
;
INSERT INTO direct_transmission_rules (
  rule_key, version, source_entity_type, source_variable_key, source_variable_version,
  source_direction, relation_type, target_entity_type,
  affected_variable_key, affected_variable_version, affected_direction,
  condition_summary, mechanism_template, status
) VALUES (
  'synthetic_production_decrease_reduces_chain_supply', 1,
  'chain_node', 'production_volume', 1, 'decrease', 'produces', 'chain_node',
  'market_supply', 1, 'decrease', 'Synthetic fixture only',
  '{source_entity} production decline reduces {target_entity} supply', 'approved'
)
`); err != nil {
		return err
	}
	_, err := database.Exec(ctx, `
INSERT INTO entity_edges (
  id, from_entity_id, to_entity_id, relation_type, evidence_note, status
) VALUES
(
  '22000000-0000-4000-8000-000000000003',
  '22000000-0000-4000-8000-000000000001',
  '22000000-0000-4000-8000-000000000002',
  'produces', 'Synthetic acceptance fixture', 'active'
),
(
  '23000000-0000-4000-8000-000000000003',
  '23000000-0000-4000-8000-000000000001',
  '23000000-0000-4000-8000-000000000002',
  'mapped_to_industry', 'Synthetic formal anchor route', 'active'
)
`)
	return err
}

func seedEvent(
	ctx context.Context,
	database *pgxpool.Pool,
	rawDocumentID string,
	eventID string,
	evidenceID string,
	dedupeKey string,
	occurredAt string,
) error {
	contentHash := fmt.Sprintf("%x", sha256.Sum256([]byte("raw:"+eventID)))
	evidenceHash := fmt.Sprintf("%x", sha256.Sum256([]byte("evidence:"+evidenceID)))
	if _, err := database.Exec(ctx, `
INSERT INTO raw_documents (
  id, ingest_channel, source_type, source_name, source_url, title, content_text,
  raw_mime_type, language, published_at, collected_at, content_hash, ingest_status
) VALUES (
  $1, 'synthetic_acceptance', 'news', 'Synthetic Primary Source',
  'https://synthetic.invalid/wafer', 'Synthetic wafer production update',
  'Synthetic Wafer Fab production fell 10%', 'text/plain', 'en',
  $2, $2::timestamptz + interval '1 minute', $3, 'collected'
)
`, rawDocumentID, occurredAt, contentHash); err != nil {
		return err
	}
	if _, err := database.Exec(ctx, `
INSERT INTO events (
  id, title, summary, event_time, first_seen_at, knowable_at,
  event_status, fact_status, dedupe_key, fact_payload
) VALUES (
  $1, 'Synthetic Wafer Fab production fell 10%',
  'Synthetic Wafer Fab confirmed a 10% production decline.',
  $3, $3::timestamptz + interval '1 minute', $3::timestamptz + interval '1 minute',
  'confirmed', 'verified', $2, '{"production_change_percent":-10}'::jsonb
)
`, eventID, dedupeKey, occurredAt); err != nil {
		return err
	}
	if _, err := database.Exec(ctx, `
INSERT INTO event_sources (
  id, event_id, raw_document_id, source_level, evidence_excerpt, evidence_hash,
  evidence_relation, supports_fields, is_primary
) VALUES (
  $1, $2, $3, 'primary', 'Synthetic Wafer Fab production fell 10%',
  $4, 'supports', ARRAY['title','factual_summary','occurred_at','fact_payload'], true
)
`, evidenceID, eventID, rawDocumentID, evidenceHash); err != nil {
		return err
	}
	_, err := database.Exec(
		ctx, `UPDATE events SET primary_source_id = $2 WHERE id = $1`, eventID, evidenceID,
	)
	return err
}

func assertEmptyResearch(ctx context.Context) error {
	database, err := openFixtureDatabase(ctx)
	if err != nil {
		return err
	}
	defer database.Close()
	var themes, reasoningTrees int
	if err := database.QueryRow(ctx, `SELECT count(*) FROM research_themes`).Scan(&themes); err != nil {
		return err
	}
	if err := database.QueryRow(ctx, `SELECT count(*) FROM research_reasoning_trees`).Scan(&reasoningTrees); err != nil {
		return err
	}
	if themes != 0 || reasoningTrees != 0 {
		return fmt.Errorf("unexpected research outputs: themes=%d reasoning_trees=%d", themes, reasoningTrees)
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]int{
		"research_themes": themes, "research_reasoning_trees": reasoningTrees,
	})
}

func assertResearchPublication(ctx context.Context) error {
	database, err := openFixtureDatabase(ctx)
	if err != nil {
		return err
	}
	defer database.Close()
	var themes, trees, formalSignals, formalImpacts, inferenceSignals int
	if err := database.QueryRow(ctx, `SELECT
    (SELECT count(*) FROM research_themes),
    (SELECT count(*) FROM research_reasoning_trees),
    (SELECT count(*) FROM research_reasoning_tree_node_signals WHERE source_kind = 'formal_signal'),
    (SELECT count(*) FROM research_reasoning_tree_nodes WHERE incoming_source_kind = 'formal_direct_impact'),
    (SELECT count(*) FROM research_reasoning_tree_node_signals WHERE source_kind = 'analyst_inference')
`).Scan(&themes, &trees, &formalSignals, &formalImpacts, &inferenceSignals); err != nil {
		return err
	}
	if themes != 1 || trees != 1 || formalSignals != 1 || formalImpacts != 1 || inferenceSignals != 1 {
		return fmt.Errorf(
			"unexpected research publication: themes=%d trees=%d formal_signals=%d formal_impacts=%d inference_signals=%d",
			themes, trees, formalSignals, formalImpacts, inferenceSignals,
		)
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]int{
		"research_themes": themes, "research_reasoning_trees": trees,
		"formal_signals": formalSignals, "formal_direct_impacts": formalImpacts,
		"analyst_inference_signals": inferenceSignals,
	})
}
