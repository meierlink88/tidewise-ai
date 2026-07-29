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
	action := flag.String("action", "", "create-database, drop-database, seed, or assert-empty-research")
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
	return seedEvent(
		ctx, database,
		"21000000-0000-4000-8000-000000000001",
		"21000000-0000-4000-8000-000000000002",
		"21000000-0000-4000-8000-000000000003",
		"synthetic:quarantined", "2026-07-28T09:00:00Z",
	)
}

func seedEntities(ctx context.Context, database *pgxpool.Pool) error {
	if _, err := database.Exec(ctx, `
INSERT INTO entity_nodes (
  id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status
) VALUES
  ('22000000-0000-4000-8000-000000000001', 'company:synthetic-wafer-fab',
   'company', 'company', 'Synthetic Wafer Fab', 'Synthetic Wafer Fab', ARRAY['SWF'], 'active'),
  ('22000000-0000-4000-8000-000000000002', 'product:synthetic-wafer',
   'product', 'product', 'Synthetic 8-inch Wafer', 'Synthetic 8-inch Wafer',
   ARRAY['Synthetic Wafer'], 'active')
`); err != nil {
		return err
	}
	_, err := database.Exec(ctx, `
INSERT INTO entity_edges (
  id, from_entity_id, to_entity_id, relation_type, evidence_note, status
) VALUES (
  '22000000-0000-4000-8000-000000000003',
  '22000000-0000-4000-8000-000000000001',
  '22000000-0000-4000-8000-000000000002',
  'produces', 'Synthetic acceptance fixture', 'active'
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
