package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
	reasoningtreeimport "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchreasoningtreeimport"
	themeimport "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
)

func TestResearchV1PostgresImportReplayReadAndImmutability(t *testing.T) {
	db := openResearchV1TestDatabase(t)
	seedResearchV1MasterData(t, db)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	now := time.Now().UTC().Truncate(time.Second)
	const (
		publisher    = "analyst-service"
		chainID      = "10000000-0000-4000-8000-000000000001"
		nodeID       = "10000000-0000-4000-8000-000000000002"
		eventID      = "10000000-0000-4000-8000-000000000003"
		draftEventID = "10000000-0000-4000-8000-000000000004"
	)

	themeBatch := themeimport.Batch{
		AnalysisBatchID: "research-v1-postgres-integration",
		AnalysisAsOf:    now.Format(time.RFC3339),
		WindowStart:     now.Add(-time.Hour).Format(time.RFC3339),
		WindowEnd:       now.Format(time.RFC3339),
		Themes: []themeimport.Theme{{
			ThemeKey:                  "ai.optical-module-demand",
			Title:                     "高速光模块需求验证",
			OneLineConclusion:         "云厂商端口计划可能增强高速光模块需求。",
			ConclusionDirection:       "positive",
			ImpactStrength:            "medium",
			TransmissionStage:         "validation",
			InvestmentGuidanceAction:  "focus",
			InvestmentGuidanceSummary: "优先验证采购订单与排产。",
			TimeHorizonCategory:       "short_term",
			Impacts: []themeimport.Impact{{
				ChainNodeEntityID:           nodeID,
				RelationRole:                "beneficiary",
				ImpactDirection:             "positive",
				PrimarySignalDisplaySummary: "模块需求：可能增加",
				DisplayOrder:                1,
			}},
			Events: []themeimport.Event{{
				EventID:      eventID,
				EvidenceRole: "driver",
			}},
		}},
	}
	themeService := themeimport.NewService(NewResearchThemeImportStore(db))
	draftEventBatch := themeBatch
	draftEventBatch.AnalysisBatchID = "research-v1-postgres-draft-event"
	draftEventBatch.Themes = append([]themeimport.Theme(nil), themeBatch.Themes...)
	draftEventBatch.Themes[0].ThemeKey = "ai.optical-module-draft-event"
	draftEventBatch.Themes[0].Events = []themeimport.Event{{
		EventID:      draftEventID,
		EvidenceRole: "driver",
	}}
	if _, err := themeService.Import(ctx, publisher, draftEventBatch); err == nil {
		t.Fatal("Theme import with an unconfirmed and unverified Event unexpectedly succeeded")
	} else {
		var referenceError *themeimport.ReferenceError
		if !errors.As(err, &referenceError) || referenceError.Reference != draftEventID {
			t.Fatalf("draft Event import error = %v, want a stable Event ReferenceError", err)
		}
	}
	var failedReceiptCount int
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM research_theme_import_receipts WHERE analysis_batch_id = $1`,
		draftEventBatch.AnalysisBatchID).Scan(&failedReceiptCount); err != nil {
		t.Fatal(err)
	}
	if failedReceiptCount != 0 {
		t.Fatalf("failed draft Event import persisted %d receipt rows, want 0", failedReceiptCount)
	}

	themeResult, err := themeService.Import(ctx, publisher, themeBatch)
	if err != nil {
		t.Fatalf("import Theme V1 publication: %v", err)
	}
	themeID := themeResult.ThemeIDsByKey["ai.optical-module-demand"]
	if themeID == "" {
		t.Fatal("Theme V1 import returned no deterministic Theme ID")
	}
	themeReplay, err := themeService.Import(ctx, publisher, themeBatch)
	if err != nil {
		t.Fatalf("replay Theme V1 publication: %v", err)
	}
	if !themeReplay.Replayed || themeReplay.ReceiptID != themeResult.ReceiptID {
		t.Fatalf("Theme replay = %#v, want the original receipt", themeReplay)
	}

	treePublication := reasoningtreeimport.Publication{
		ThemeID: themeID,
		ReasoningTrees: []reasoningtreeimport.ReasoningTree{{
			IndustryChainEntityID: chainID,
			Title:                 "高速光模块产业链",
			DisplayOrder:          1,
			OneLineConclusion:     "光模块备料需求可能上升。",
			ImpactDirection:       "positive",
			ImpactStrength:        "medium",
			Events: []reasoningtreeimport.Event{{
				EventID:      eventID,
				EvidenceRole: "driver",
				DisplayOrder: 1,
			}},
			Nodes: []reasoningtreeimport.Node{{
				Position:          1,
				ChainNodeEntityID: nodeID,
				ImpactDirection:   "positive",
				ImpactStrength:    "medium",
				Signals: []reasoningtreeimport.Signal{{
					VariableSignalKey: "optical-module.demand",
					SignalRole:        "primary",
					SignalDirection:   "increase",
					DisplaySummary:    "模块需求 ↑",
					DisplayOrder:      1,
				}},
			}},
		}},
	}
	treeService := reasoningtreeimport.NewService(NewResearchReasoningTreeImportStore(db))
	treeResult, err := treeService.Import(ctx, publisher, treePublication)
	if err != nil {
		t.Fatalf("import Reason Tree V1 publication: %v", err)
	}
	treeID := treeResult.ReasoningTreeIDsByIndustryChainEntityID[chainID]
	if treeID == "" {
		t.Fatal("Reason Tree V1 import returned no deterministic Tree ID")
	}
	treeReplay, err := treeService.Import(ctx, publisher, treePublication)
	if err != nil {
		t.Fatalf("replay Reason Tree V1 publication: %v", err)
	}
	if !treeReplay.Replayed || treeReplay.ReceiptID != treeResult.ReceiptID {
		t.Fatalf("Reason Tree replay = %#v, want the original receipt", treeReplay)
	}

	readService := research.NewService(NewResearchRepository(db), func() time.Time {
		return time.Now().UTC().Add(time.Minute)
	})
	page, err := readService.ListThemes(ctx, research.ResearchListRequest{WindowHours: 24, Limit: 10})
	if err != nil {
		t.Fatalf("list imported Themes: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != themeID ||
		page.Items[0].ReasoningTreeCount != 1 || len(page.Items[0].Impacts) != 1 {
		t.Fatalf("Theme read projection = %#v, want one Theme with one Impact and one Tree", page.Items)
	}
	treeList, err := readService.ListReasoningTrees(ctx, themeID)
	if err != nil {
		t.Fatalf("list imported Reason Trees: %v", err)
	}
	if len(treeList.ReasoningTrees) != 1 || treeList.ReasoningTrees[0].ReasoningTreeID != treeID {
		t.Fatalf("Reason Tree list = %#v, want Tree %s", treeList.ReasoningTrees, treeID)
	}
	detail, err := readService.GetReasoningTree(ctx, themeID, treeID)
	if err != nil {
		t.Fatalf("read imported Reason Tree: %v", err)
	}
	if len(detail.ReasoningTree.Nodes) != 1 ||
		detail.ReasoningTree.Nodes[0].PrimarySignal.VariableSignalKey != "optical-module.demand" ||
		len(detail.ImpactNodeIDs) != 1 || detail.ImpactNodeIDs[0] != nodeID {
		t.Fatalf("Reason Tree detail = %#v, want the imported one-node impact path", detail)
	}

	if _, err := db.ExecContext(ctx,
		`UPDATE research_themes SET title = 'mutated' WHERE id = $1`, themeID); err == nil {
		t.Fatal("published Theme update unexpectedly succeeded")
	}
	if _, err := db.ExecContext(ctx,
		`DELETE FROM research_reasoning_trees WHERE id = $1`, treeID); err == nil {
		t.Fatal("published Reason Tree delete unexpectedly succeeded")
	}
}

func openResearchV1TestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	databaseURL := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TIDEWISE_TEST_DATABASE_URL to run Research V1 PostgreSQL integration tests")
	}
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	host := parsed.Hostname()
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		t.Fatalf("Research V1 integration database must use a loopback host, got %q", host)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	admin, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("tw_research_v1_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		admin.Close()
		t.Fatal(err)
	}
	config.RuntimeParams["search_path"] = schema
	config.RuntimeParams["tidewise.phase_a_cleanup_write_authorized"] = "reviewed_backup_verified"
	config.RuntimeParams["tidewise.external_identifier_schema_write_authorized"] = "reviewed_backup_verified"
	config.RuntimeParams["tidewise.alliance_economy_schema_write_authorized"] = "reviewed_local_cleanup_verified"
	db := stdlib.OpenDB(*config)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		admin.Close()
		t.Fatal(err)
	}
	migrationDirectory, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpContext(ctx, db, migrationDirectory); err != nil {
		t.Fatalf("apply migrations in isolated Research V1 schema: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		_, _ = admin.ExecContext(context.Background(), `DROP SCHEMA IF EXISTS `+schema+` CASCADE`)
		admin.Close()
	})
	return db
}

func seedResearchV1MasterData(t *testing.T, db *sql.DB) {
	t.Helper()
	const (
		chainID      = "10000000-0000-4000-8000-000000000001"
		nodeID       = "10000000-0000-4000-8000-000000000002"
		eventID      = "10000000-0000-4000-8000-000000000003"
		draftEventID = "10000000-0000-4000-8000-000000000004"
	)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, statement := range []string{
		`INSERT INTO entity_nodes (
		    id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status
		) VALUES
		    ('` + chainID + `', 'industry-chain:optical-module', 'industry_chain', 'industry_chain',
		     '高速光模块产业链', '高速光模块产业链', '{}', 'active'),
		    ('` + nodeID + `', 'chain-node:optical-module', 'chain_node', 'chain_node',
		     '高速光模块', '高速光模块', '{}', 'active')`,
		`INSERT INTO chain_node_profiles (
		    entity_id, definition, boundary_note, review_status
		) VALUES ('` + nodeID + `', '高速光模块生产节点', '仅覆盖高速光模块', 'approved')`,
		`INSERT INTO industry_chain_definitions (
		    entity_id, scope, target_output, end_use, technology_route_qualifier,
		    observable_variables, geography, as_of_date, review_status, review_note
		) VALUES (
		    '` + chainID + `', '高速光模块供需链', '高速光模块', '数据中心互联', NULL,
		    ARRAY['采购数量'], '中国', CURRENT_DATE, 'approved', NULL
		)`,
		`INSERT INTO industry_chain_node_memberships (
		    industry_chain_entity_id, chain_node_entity_id, position, contextual_stage,
		    review_status, status, inclusion_reason, evidence_ids, source_name, source_url, verified_at
		) VALUES (
		    '` + chainID + `', '` + nodeID + `', 1, 'midstream',
		    'approved', 'active', '核心产出节点', ARRAY['evidence:integration'],
		    'integration fixture', 'artifact://research-v1-integration', now()
		)`,
		`INSERT INTO events (
		    id, title, summary, first_seen_at, event_status, fact_status, dedupe_key
		) VALUES
		(
		    '` + eventID + `', '端口计划上调', '端口计划上调 80%', now(),
		    'confirmed', 'verified', 'research-v1-integration-event'
		),
		(
		    '` + draftEventID + `', '待核验端口计划', '未经核验的端口计划', now(),
		    'candidate', 'unverified', 'research-v1-integration-draft-event'
		)`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("seed Research V1 master data: %v\n%s", err, statement)
		}
	}
}
