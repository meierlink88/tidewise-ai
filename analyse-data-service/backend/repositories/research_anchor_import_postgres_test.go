package repositories

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestResearchAnchorImportUsesThemeScopedTransactionLock(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	repositoryTx := &postgresResearchAnchorImportTx{tx: tx}

	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_xact_lock(hashtextextended($1, 0))")).
		WithArgs("research-anchor:11111111-1111-4111-8111-111111111111").
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repositoryTx.LockResearchAnchorImportTheme(context.Background(), "11111111-1111-4111-8111-111111111111"); err != nil {
		t.Fatal(err)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestResearchAnchorImportReceiptDecodesFrozenResult(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	repositoryTx := &postgresResearchAnchorImportTx{tx: tx}
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)
	query := `SELECT id, theme_id, publisher_subject, payload_hash,
       anchor_ids_by_center_chain_node_id, write_counts, published_at, imported_at
FROM research_anchor_import_receipts WHERE theme_id = $1`
	mock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs("11111111-1111-4111-8111-111111111111").WillReturnRows(sqlmock.NewRows([]string{
		"id", "theme_id", "publisher_subject", "payload_hash", "anchor_ids", "write_counts", "published_at", "imported_at",
	}).AddRow(
		"99999999-9999-4999-8999-999999999999", "11111111-1111-4111-8111-111111111111", "service:ai-research-analyst",
		"316ae969f3a946d6ffb2e58bc13ccabae81d95cd7e27575006670890909cb4eb",
		[]byte(`{"22222222-2222-4222-8222-222222222222":"534d83be-774b-51d9-ad00-cdee4ba91799"}`),
		[]byte(`{"anchors":1,"event_associations":2,"path_nodes":2,"receipts":1}`), now, now,
	))

	receipt, err := repositoryTx.ResearchAnchorImportReceipt(context.Background(), "11111111-1111-4111-8111-111111111111")
	if err != nil {
		t.Fatal(err)
	}
	if receipt == nil || receipt.AnchorIDsByCenterChainNodeID["22222222-2222-4222-8222-222222222222"] != "534d83be-774b-51d9-ad00-cdee4ba91799" {
		t.Fatalf("receipt = %#v", receipt)
	}
	if receipt.Counts != (ResearchAnchorImportCounts{Anchors: 1, EventAssociations: 2, PathNodes: 2, Receipts: 1}) {
		t.Fatalf("counts = %#v", receipt.Counts)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestResearchAnchorImportPersistsBranchSummaries(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	repositoryTx := &postgresResearchAnchorImportTx{tx: tx}
	counterSummary := "交付节奏仍有分化"
	query := `INSERT INTO research_anchors (
    id, theme_id, center_chain_node_entity_id, import_receipt_id,
    one_line_conclusion, fact_summary, net_direction_summary, support_summary,
    counter_summary, trading_direction, next_checkpoint
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`
	mock.ExpectExec(regexp.QuoteMeta(query)).WithArgs(
		"anchor-id", "theme-id", "center-id", "receipt-id",
		"结论", "事实", "方向", "当前支持", counterSummary, "交易", "检查",
	).WillReturnResult(sqlmock.NewResult(0, 1))

	err = repositoryTx.InsertResearchAnchor(context.Background(), ResearchAnchorImportAnchor{
		ID: "anchor-id", ThemeID: "theme-id", CenterChainNodeEntityID: "center-id", ImportReceiptID: "receipt-id",
		OneLineConclusion: "结论", FactSummary: "事实", NetDirectionSummary: "方向",
		SupportSummary: "当前支持", CounterSummary: &counterSummary,
		TradingDirection: "交易", NextCheckpoint: "检查",
	})
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestResearchAnchorImportReadsParentPublicationAndAssociationBoundaries(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	repositoryTx := &postgresResearchAnchorImportTx{tx: tx}
	const themeID = "11111111-1111-4111-8111-111111111111"

	mock.ExpectQuery("SELECT t.id::text").WithArgs(themeID).WillReturnRows(sqlmock.NewRows([]string{
		"id", "import_receipt_id", "publisher_subject",
	}).AddRow(themeID, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "service:ai-research-analyst"))
	publication, err := repositoryTx.ResearchAnchorImportThemePublication(context.Background(), themeID)
	if err != nil {
		t.Fatal(err)
	}
	if publication == nil || publication.PublisherSubject != "service:ai-research-analyst" {
		t.Fatalf("publication = %#v", publication)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT chain_node_entity_id::text FROM research_theme_chain_nodes WHERE theme_id = $1")).
		WithArgs(themeID).
		WillReturnRows(sqlmock.NewRows([]string{"chain_node_entity_id"}).AddRow("22222222-2222-4222-8222-222222222222"))
	centers, err := repositoryTx.ResearchAnchorImportThemeChainNodes(context.Background(), themeID)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := centers["22222222-2222-4222-8222-222222222222"]; !exists {
		t.Fatalf("centers = %#v", centers)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT event_id::text FROM research_theme_events WHERE theme_id = $1")).
		WithArgs(themeID).
		WillReturnRows(sqlmock.NewRows([]string{"event_id"}).AddRow("55555555-5555-4555-8555-555555555555"))
	events, err := repositoryTx.ResearchAnchorImportThemeEvents(context.Background(), themeID)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := events["55555555-5555-4555-8555-555555555555"]; !exists {
		t.Fatalf("events = %#v", events)
	}

	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
