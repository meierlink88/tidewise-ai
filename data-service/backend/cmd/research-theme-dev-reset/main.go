package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	data "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data"
)

const (
	localDatabaseName = "tidewise_local"
	resetLockKey      = "tidewise:research-theme-dev-reset:v1"

	currentDatabaseSQL       = `SELECT current_database()`
	acquireResetLockSQL      = `SELECT pg_try_advisory_xact_lock(hashtextextended($1, 0))`
	immutableTriggerStateSQL = `SELECT COUNT(*), COUNT(*) FILTER (WHERE t.tgenabled = 'O')
FROM pg_trigger AS t
JOIN pg_class AS c ON c.oid = t.tgrelid
JOIN pg_namespace AS n ON n.oid = c.relnamespace
WHERE n.nspname = current_schema()
  AND t.tgname IN (
    'trg_research_theme_receipts_immutable',
    'trg_research_themes_immutable',
    'trg_research_theme_impacts_immutable',
    'trg_research_theme_events_immutable',
    'trg_research_reasoning_tree_receipts_immutable',
    'trg_research_reasoning_trees_immutable',
    'trg_research_reasoning_tree_events_immutable',
    'trg_research_reasoning_tree_nodes_immutable',
    'trg_research_reasoning_tree_node_signals_immutable'
)`
	publicationCountsSQL = `SELECT
    (SELECT COUNT(*) FROM research_theme_import_receipts),
    (SELECT COUNT(*) FROM research_themes),
    (SELECT COUNT(*) FROM research_theme_impacts),
    (SELECT COUNT(*) FROM research_theme_events),
    (SELECT COUNT(*) FROM research_reasoning_tree_import_receipts),
    (SELECT COUNT(*) FROM research_reasoning_trees),
    (SELECT COUNT(*) FROM research_reasoning_tree_events),
    (SELECT COUNT(*) FROM research_reasoning_tree_nodes),
    (SELECT COUNT(*) FROM research_reasoning_tree_node_signals)`
	protectedCountsSQL = `SELECT
    (SELECT COUNT(*) FROM events),
    (SELECT COUNT(*) FROM entity_nodes),
    (SELECT COUNT(*) FROM chain_node_profiles),
    (SELECT COUNT(*) FROM industry_chain_definitions),
    (SELECT COUNT(*) FROM industry_chain_graph_edges),
    (SELECT COUNT(*) FROM index_profiles),
    (SELECT COUNT(*) FROM event_tag_defs),
    (SELECT COUNT(*) FROM event_tag_maps),
    (SELECT COUNT(*) FROM raw_documents)`
	disablePublicationTriggersSQL = `ALTER TABLE research_theme_import_receipts DISABLE TRIGGER trg_research_theme_receipts_immutable;
ALTER TABLE research_themes DISABLE TRIGGER trg_research_themes_immutable;
ALTER TABLE research_theme_impacts DISABLE TRIGGER trg_research_theme_impacts_immutable;
ALTER TABLE research_theme_events DISABLE TRIGGER trg_research_theme_events_immutable;
ALTER TABLE research_reasoning_tree_import_receipts DISABLE TRIGGER trg_research_reasoning_tree_receipts_immutable;
ALTER TABLE research_reasoning_trees DISABLE TRIGGER trg_research_reasoning_trees_immutable;
ALTER TABLE research_reasoning_tree_events DISABLE TRIGGER trg_research_reasoning_tree_events_immutable;
ALTER TABLE research_reasoning_tree_nodes DISABLE TRIGGER trg_research_reasoning_tree_nodes_immutable;
ALTER TABLE research_reasoning_tree_node_signals DISABLE TRIGGER trg_research_reasoning_tree_node_signals_immutable`
	deletePublicationsSQL = `DELETE FROM research_reasoning_tree_node_signals;
DELETE FROM research_reasoning_tree_nodes;
DELETE FROM research_reasoning_tree_events;
DELETE FROM research_reasoning_trees;
DELETE FROM research_reasoning_tree_import_receipts;
DELETE FROM research_theme_events;
DELETE FROM research_theme_impacts;
DELETE FROM research_themes;
DELETE FROM research_theme_import_receipts`
	enablePublicationTriggersSQL = `ALTER TABLE research_theme_import_receipts ENABLE TRIGGER trg_research_theme_receipts_immutable;
ALTER TABLE research_themes ENABLE TRIGGER trg_research_themes_immutable;
ALTER TABLE research_theme_impacts ENABLE TRIGGER trg_research_theme_impacts_immutable;
ALTER TABLE research_theme_events ENABLE TRIGGER trg_research_theme_events_immutable;
ALTER TABLE research_reasoning_tree_import_receipts ENABLE TRIGGER trg_research_reasoning_tree_receipts_immutable;
ALTER TABLE research_reasoning_trees ENABLE TRIGGER trg_research_reasoning_trees_immutable;
ALTER TABLE research_reasoning_tree_events ENABLE TRIGGER trg_research_reasoning_tree_events_immutable;
ALTER TABLE research_reasoning_tree_nodes ENABLE TRIGGER trg_research_reasoning_tree_nodes_immutable;
ALTER TABLE research_reasoning_tree_node_signals ENABLE TRIGGER trg_research_reasoning_tree_node_signals_immutable`
)

type resetOptions struct {
	Execute         bool
	ConfirmDatabase string
}

type publicationCounts struct {
	ResearchThemeImportReceipts         int64 `json:"research_theme_import_receipts"`
	ResearchThemes                      int64 `json:"research_themes"`
	ResearchThemeImpacts                int64 `json:"research_theme_impacts"`
	ResearchThemeEvents                 int64 `json:"research_theme_events"`
	ResearchReasoningTreeImportReceipts int64 `json:"research_reasoning_tree_import_receipts"`
	ResearchReasoningTrees              int64 `json:"research_reasoning_trees"`
	ResearchReasoningTreeEvents         int64 `json:"research_reasoning_tree_events"`
	ResearchReasoningTreeNodes          int64 `json:"research_reasoning_tree_nodes"`
	ResearchReasoningTreeNodeSignals    int64 `json:"research_reasoning_tree_node_signals"`
}

func (c publicationCounts) isZero() bool { return c == (publicationCounts{}) }

type protectedCounts struct {
	Events                   int64 `json:"events"`
	EntityNodes              int64 `json:"entity_nodes"`
	ChainNodeProfiles        int64 `json:"chain_node_profiles"`
	IndustryChainDefinitions int64 `json:"industry_chain_definitions"`
	IndustryChainGraphEdges  int64 `json:"industry_chain_graph_edges"`
	IndexProfiles            int64 `json:"index_profiles"`
	EventTagDefs             int64 `json:"event_tag_defs"`
	EventTagMaps             int64 `json:"event_tag_maps"`
	RawDocuments             int64 `json:"raw_documents"`
}

type resetReport struct {
	Database        string            `json:"database"`
	Mode            string            `json:"mode"`
	Executed        bool              `json:"executed"`
	Before          publicationCounts `json:"before"`
	After           publicationCounts `json:"after"`
	ProtectedBefore protectedCounts   `json:"protected_before"`
	ProtectedAfter  protectedCounts   `json:"protected_after"`
	TriggerRestored bool              `json:"trigger_restored"`
}

func main() {
	execute := flag.Bool("execute", false, "delete all local Research Theme and Reason Tree publication data")
	confirmDatabase := flag.String("confirm-database", "", "must equal tidewise_local when --execute is used")
	flag.Parse()
	options := resetOptions{Execute: *execute, ConfirmDatabase: *confirmDatabase}
	if err := validateExecutionGate(options); err != nil {
		log.Fatal(err)
	}
	cfg, err := conf.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := validateResetTarget(cfg); err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := data.OpenPostgres(ctx, cfg)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()
	report, err := runReset(ctx, db, options)
	if err != nil {
		log.Fatal(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
		log.Fatalf("encode report: %v", err)
	}
}

func validateExecutionGate(options resetOptions) error {
	if options.Execute && options.ConfirmDatabase != localDatabaseName {
		return fmt.Errorf("execution requires --execute --confirm-database tidewise_local")
	}
	return nil
}

func validateResetTarget(cfg conf.Config) error {
	if cfg.App.Env != conf.EnvLocal {
		return fmt.Errorf("research publication development reset is local-only, got %q", cfg.App.Env)
	}
	if !conf.IsExternalLocalInfrastructureHost(cfg.Database.Host) {
		return fmt.Errorf("research publication development reset requires the external local PostgreSQL infrastructure host")
	}
	if cfg.Database.Name != localDatabaseName {
		return fmt.Errorf("research publication development reset requires database tidewise_local")
	}
	return nil
}

func runReset(ctx context.Context, db *sql.DB, options resetOptions) (resetReport, error) {
	if err := validateExecutionGate(options); err != nil {
		return resetReport{}, err
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return resetReport{}, fmt.Errorf("begin research publication reset transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	databaseName, err := currentDatabase(ctx, tx)
	if err != nil {
		return resetReport{}, err
	}
	if databaseName != localDatabaseName {
		return resetReport{}, fmt.Errorf("connected database is %q, require tidewise_local", databaseName)
	}
	if err := acquireResetLock(ctx, tx); err != nil {
		return resetReport{}, err
	}
	if err := requirePublicationTriggersEnabled(ctx, tx); err != nil {
		return resetReport{}, err
	}
	before, err := readPublicationCounts(ctx, tx)
	if err != nil {
		return resetReport{}, err
	}
	protectedBefore, err := readProtectedCounts(ctx, tx)
	if err != nil {
		return resetReport{}, err
	}
	report := resetReport{
		Database: databaseName, Mode: "dry-run", Before: before, After: before,
		ProtectedBefore: protectedBefore, ProtectedAfter: protectedBefore, TriggerRestored: true,
	}

	if options.Execute {
		if err := execResetSQL(ctx, tx, disablePublicationTriggersSQL, "disable immutable research publication triggers"); err != nil {
			return resetReport{}, err
		}
		if err := execResetSQL(ctx, tx, deletePublicationsSQL, "delete Research Theme and Reason Tree publications"); err != nil {
			return resetReport{}, err
		}
		if err := execResetSQL(ctx, tx, enablePublicationTriggersSQL, "restore immutable research publication triggers"); err != nil {
			return resetReport{}, err
		}
		if err := requirePublicationTriggersEnabled(ctx, tx); err != nil {
			return resetReport{}, fmt.Errorf("verify restored immutable publication triggers: %w", err)
		}
		after, err := readPublicationCounts(ctx, tx)
		if err != nil {
			return resetReport{}, err
		}
		if !after.isZero() {
			return resetReport{}, fmt.Errorf("research publication reset left non-zero data counts: %+v", after)
		}
		protectedAfter, err := readProtectedCounts(ctx, tx)
		if err != nil {
			return resetReport{}, err
		}
		if protectedAfter != protectedBefore {
			return resetReport{}, fmt.Errorf("protected data counts changed: before=%+v after=%+v", protectedBefore, protectedAfter)
		}
		report.Mode, report.Executed, report.After, report.ProtectedAfter = "execute", true, after, protectedAfter
	}

	if err := tx.Commit(); err != nil {
		return resetReport{}, fmt.Errorf("commit research publication reset transaction: %w", err)
	}
	committed = true
	return report, nil
}

func currentDatabase(ctx context.Context, tx *sql.Tx) (string, error) {
	var databaseName string
	if err := tx.QueryRowContext(ctx, currentDatabaseSQL).Scan(&databaseName); err != nil {
		return "", fmt.Errorf("read connected database name: %w", err)
	}
	return databaseName, nil
}

func acquireResetLock(ctx context.Context, tx *sql.Tx) error {
	var locked bool
	if err := tx.QueryRowContext(ctx, acquireResetLockSQL, resetLockKey).Scan(&locked); err != nil {
		return fmt.Errorf("acquire research publication reset lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("another research publication reset is already running")
	}
	return nil
}

func requirePublicationTriggersEnabled(ctx context.Context, tx *sql.Tx) error {
	var total, enabled int
	if err := tx.QueryRowContext(ctx, immutableTriggerStateSQL).Scan(&total, &enabled); err != nil {
		return fmt.Errorf("read immutable research publication trigger state: %w", err)
	}
	if total != 9 || enabled != total {
		return fmt.Errorf("immutable research publication triggers are incomplete or disabled: total=%d enabled=%d", total, enabled)
	}
	return nil
}

func readPublicationCounts(ctx context.Context, tx *sql.Tx) (publicationCounts, error) {
	var counts publicationCounts
	err := tx.QueryRowContext(ctx, publicationCountsSQL).Scan(
		&counts.ResearchThemeImportReceipts, &counts.ResearchThemes,
		&counts.ResearchThemeImpacts, &counts.ResearchThemeEvents,
		&counts.ResearchReasoningTreeImportReceipts, &counts.ResearchReasoningTrees,
		&counts.ResearchReasoningTreeEvents, &counts.ResearchReasoningTreeNodes,
		&counts.ResearchReasoningTreeNodeSignals,
	)
	if err != nil {
		return publicationCounts{}, fmt.Errorf("read research publication counts: %w", err)
	}
	return counts, nil
}

func readProtectedCounts(ctx context.Context, tx *sql.Tx) (protectedCounts, error) {
	var counts protectedCounts
	err := tx.QueryRowContext(ctx, protectedCountsSQL).Scan(
		&counts.Events, &counts.EntityNodes, &counts.ChainNodeProfiles,
		&counts.IndustryChainDefinitions, &counts.IndustryChainGraphEdges,
		&counts.IndexProfiles, &counts.EventTagDefs, &counts.EventTagMaps, &counts.RawDocuments,
	)
	if err != nil {
		return protectedCounts{}, fmt.Errorf("read protected data counts: %w", err)
	}
	return counts, nil
}

func execResetSQL(ctx context.Context, tx *sql.Tx, statement, operation string) error {
	if _, err := tx.ExecContext(ctx, statement); err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return nil
}
