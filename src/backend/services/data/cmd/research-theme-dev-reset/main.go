package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/adapters/database"
	"github.com/meierlink88/tidewise-ai/backend/services/data/config"
)

const (
	localDatabaseName = "tidewise_local"
	resetLockKey      = "tidewise:research-theme-dev-reset:v1"

	currentDatabaseSQL          = `SELECT current_database()`
	acquireResetLockSQL         = `SELECT pg_try_advisory_xact_lock(hashtextextended($1, 0))`
	themeReceiptTriggerStateSQL = `SELECT tgenabled::text
FROM pg_trigger
WHERE tgrelid = 'research_theme_import_receipts'::regclass
  AND tgname = 'trg_research_theme_import_receipts_immutable'`
	anchorReceiptTriggerStateSQL = `SELECT tgenabled::text
FROM pg_trigger
WHERE tgrelid = 'research_anchor_import_receipts'::regclass
  AND tgname = 'trg_research_anchor_import_receipts_immutable'`
	publicationCountsSQL = `SELECT
    (SELECT COUNT(*) FROM research_themes),
    (SELECT COUNT(*) FROM research_theme_chain_nodes),
    (SELECT COUNT(*) FROM research_theme_indices),
    (SELECT COUNT(*) FROM research_theme_events),
    (SELECT COUNT(*) FROM research_theme_import_receipts),
    (SELECT COUNT(*) FROM research_anchor_import_receipts),
    (SELECT COUNT(*) FROM research_anchors),
    (SELECT COUNT(*) FROM research_anchor_chain_nodes),
    (SELECT COUNT(*) FROM research_anchor_events)`
	protectedCountsSQL = `SELECT
    (SELECT COUNT(*) FROM events),
    (SELECT COUNT(*) FROM entity_nodes),
    (SELECT COUNT(*) FROM chain_node_profiles),
    (SELECT COUNT(*) FROM index_profiles),
    (SELECT COUNT(*) FROM event_tag_defs),
    (SELECT COUNT(*) FROM event_tag_maps),
	(SELECT COUNT(*) FROM raw_documents)`
	disableThemeReceiptTriggerSQL = `ALTER TABLE research_theme_import_receipts
DISABLE TRIGGER trg_research_theme_import_receipts_immutable`
	disableAnchorReceiptTriggerSQL = `ALTER TABLE research_anchor_import_receipts
DISABLE TRIGGER trg_research_anchor_import_receipts_immutable`
	deleteAnchorsSQL              = `DELETE FROM research_anchors`
	deleteAnchorReceiptsSQL       = `DELETE FROM research_anchor_import_receipts`
	deleteThemesSQL               = `DELETE FROM research_themes`
	deleteThemeReceiptsSQL        = `DELETE FROM research_theme_import_receipts`
	enableAnchorReceiptTriggerSQL = `ALTER TABLE research_anchor_import_receipts
ENABLE TRIGGER trg_research_anchor_import_receipts_immutable`
	enableThemeReceiptTriggerSQL = `ALTER TABLE research_theme_import_receipts
ENABLE TRIGGER trg_research_theme_import_receipts_immutable`
)

type resetOptions struct {
	Execute         bool
	ConfirmDatabase string
}

type publicationCounts struct {
	ResearchThemes               int64 `json:"research_themes"`
	ResearchThemeChainNodes      int64 `json:"research_theme_chain_nodes"`
	ResearchThemeIndices         int64 `json:"research_theme_indices"`
	ResearchThemeEvents          int64 `json:"research_theme_events"`
	ResearchThemeImportReceipts  int64 `json:"research_theme_import_receipts"`
	ResearchAnchorImportReceipts int64 `json:"research_anchor_import_receipts"`
	ResearchAnchors              int64 `json:"research_anchors"`
	ResearchAnchorChainNodes     int64 `json:"research_anchor_chain_nodes"`
	ResearchAnchorEvents         int64 `json:"research_anchor_events"`
}

func (c publicationCounts) isZero() bool {
	return c == (publicationCounts{})
}

type protectedCounts struct {
	Events            int64 `json:"events"`
	EntityNodes       int64 `json:"entity_nodes"`
	ChainNodeProfiles int64 `json:"chain_node_profiles"`
	IndexProfiles     int64 `json:"index_profiles"`
	EventTagDefs      int64 `json:"event_tag_defs"`
	EventTagMaps      int64 `json:"event_tag_maps"`
	RawDocuments      int64 `json:"raw_documents"`
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
	execute := flag.Bool("execute", false, "delete all local Research Theme and Research Anchor publication data")
	confirmDatabase := flag.String("confirm-database", "", "must equal tidewise_local when --execute is used")
	flag.Parse()

	options := resetOptions{Execute: *execute, ConfirmDatabase: *confirmDatabase}
	if err := validateExecutionGate(options); err != nil {
		log.Fatal(err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := validateResetTarget(cfg); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := database.Open(ctx, cfg)
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
	if !options.Execute {
		return nil
	}
	if options.ConfirmDatabase != localDatabaseName {
		return fmt.Errorf("execution requires --execute --confirm-database tidewise_local")
	}
	return nil
}

func validateResetTarget(cfg config.Config) error {
	if cfg.App.Env != config.EnvLocal {
		return fmt.Errorf("research publication development reset is local-only, got %q", cfg.App.Env)
	}

	host, databaseName := cfg.Database.Host, cfg.Database.Name
	if cfg.Secrets.DatabaseURL != "" {
		parsed, err := url.ParseRequestURI(cfg.Secrets.DatabaseURL)
		if err != nil || parsed.Hostname() == "" {
			return fmt.Errorf("research publication development reset requires a valid PostgreSQL URL")
		}
		host = parsed.Hostname()
		databaseName = strings.TrimPrefix(parsed.EscapedPath(), "/")
	}
	if !isLoopbackHost(host) {
		return fmt.Errorf("research publication development reset requires a loopback PostgreSQL host")
	}
	if databaseName != localDatabaseName {
		return fmt.Errorf("research publication development reset requires database tidewise_local")
	}
	return nil
}

func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(host)
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
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
	if err := requireReceiptTriggersEnabled(ctx, tx); err != nil {
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
		Database:        databaseName,
		Mode:            "dry-run",
		Before:          before,
		After:           before,
		ProtectedBefore: protectedBefore,
		ProtectedAfter:  protectedBefore,
		TriggerRestored: true,
	}

	if options.Execute {
		if err := execResetSQL(ctx, tx, disableThemeReceiptTriggerSQL, "disable immutable research theme receipt trigger"); err != nil {
			return resetReport{}, err
		}
		if err := execResetSQL(ctx, tx, disableAnchorReceiptTriggerSQL, "disable immutable research anchor receipt trigger"); err != nil {
			return resetReport{}, err
		}
		if err := execResetSQL(ctx, tx, deleteAnchorsSQL, "delete research anchors"); err != nil {
			return resetReport{}, err
		}
		if err := execResetSQL(ctx, tx, deleteAnchorReceiptsSQL, "delete research anchor import receipts"); err != nil {
			return resetReport{}, err
		}
		if err := execResetSQL(ctx, tx, deleteThemesSQL, "delete research themes"); err != nil {
			return resetReport{}, err
		}
		if err := execResetSQL(ctx, tx, deleteThemeReceiptsSQL, "delete research theme import receipts"); err != nil {
			return resetReport{}, err
		}
		if err := execResetSQL(ctx, tx, enableAnchorReceiptTriggerSQL, "restore immutable research anchor receipt trigger"); err != nil {
			return resetReport{}, err
		}
		if err := execResetSQL(ctx, tx, enableThemeReceiptTriggerSQL, "restore immutable research theme receipt trigger"); err != nil {
			return resetReport{}, err
		}
		if err := requireReceiptTriggersEnabled(ctx, tx); err != nil {
			return resetReport{}, fmt.Errorf("verify restored immutable receipt triggers: %w", err)
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
		report.Mode = "execute"
		report.Executed = true
		report.After = after
		report.ProtectedAfter = protectedAfter
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

func requireReceiptTriggersEnabled(ctx context.Context, tx *sql.Tx) error {
	if err := requireReceiptTriggerEnabled(ctx, tx, themeReceiptTriggerStateSQL, "research theme"); err != nil {
		return err
	}
	return requireReceiptTriggerEnabled(ctx, tx, anchorReceiptTriggerStateSQL, "research anchor")
}

func requireReceiptTriggerEnabled(ctx context.Context, tx *sql.Tx, stateSQL, receiptType string) error {
	var state string
	if err := tx.QueryRowContext(ctx, stateSQL).Scan(&state); err != nil {
		return fmt.Errorf("read immutable %s receipt trigger state: %w", receiptType, err)
	}
	if state != "O" {
		return fmt.Errorf("immutable %s receipt trigger is not enabled, state=%q", receiptType, state)
	}
	return nil
}

func readPublicationCounts(ctx context.Context, tx *sql.Tx) (publicationCounts, error) {
	var counts publicationCounts
	if err := tx.QueryRowContext(ctx, publicationCountsSQL).Scan(
		&counts.ResearchThemes,
		&counts.ResearchThemeChainNodes,
		&counts.ResearchThemeIndices,
		&counts.ResearchThemeEvents,
		&counts.ResearchThemeImportReceipts,
		&counts.ResearchAnchorImportReceipts,
		&counts.ResearchAnchors,
		&counts.ResearchAnchorChainNodes,
		&counts.ResearchAnchorEvents,
	); err != nil {
		return publicationCounts{}, fmt.Errorf("read research publication counts: %w", err)
	}
	return counts, nil
}

func readProtectedCounts(ctx context.Context, tx *sql.Tx) (protectedCounts, error) {
	var counts protectedCounts
	if err := tx.QueryRowContext(ctx, protectedCountsSQL).Scan(
		&counts.Events,
		&counts.EntityNodes,
		&counts.ChainNodeProfiles,
		&counts.IndexProfiles,
		&counts.EventTagDefs,
		&counts.EventTagMaps,
		&counts.RawDocuments,
	); err != nil {
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
