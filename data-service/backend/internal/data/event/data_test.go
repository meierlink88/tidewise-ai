package event

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	eventbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/event"
	eventfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/event"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestListEventsRejectsInvalidPersistedRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 8, 11, 1, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT COUNT\(\*\)`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT id, title, summary`).WillReturnRows(sqlmock.NewRows([]string{
		"id", "title", "summary", "event_time", "first_seen_at", "knowable_at", "event_status", "fact_status", "dedupe_key",
	}).AddRow("event-1", "Title", "Summary", nil, now, now, "unknown", "verified", "event:key"))
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.ListEvents(context.Background(), eventbiz.EventListFilter{})
	if err == nil || !strings.Contains(err.Error(), "read Event invariant") {
		t.Fatalf("ListEvents() error = %v, want persisted invariant failure", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestResearchEventProviderReadsOnlyEligibleFormalFacts(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 8, 12, 1, 0, 0, 0, time.UTC)
	eventPayload := []byte(`{"id":"EVT10000000-0000-4000-8000-000000000001","title":"Title","summary":"Summary","occurred_at":null,"first_seen_at":"2026-08-12T01:00:00Z","knowledge_available_at":"2026-08-12T01:00:00Z","event_status":"confirmed","fact_status":"verified"}`)
	statementHash := sha256.Sum256([]byte("statement"))
	evidencePayload, err := json.Marshal([]eventbiz.ResearchEvidenceFact{
		{
			EvidenceID: "EEL20000000-0000-4000-8000-000000000001", EvidenceHash: hex.EncodeToString(statementHash[:]),
			Statement: "statement", SourceLevel: string(eventbiz.EventSourceLevelPrimary),
			Relation: string(eventbiz.EvidenceRelationSupports), SupportsFields: []string{eventbiz.EventFieldTitle},
			RawDocumentID: "EER30000000-0000-4000-8000-000000000001", SourceName: "Source", SourceType: "news",
			Title: "Article", FirstSeenAt: now, KnowledgeAvailableAt: now, AcceptedAt: now, StatementSource: "extractor",
		},
		{
			EvidenceID: "EEL20000000-0000-4000-8000-000000000002", EvidenceHash: hex.EncodeToString(statementHash[:]),
			Statement: "statement", SourceLevel: string(eventbiz.EventSourceLevelPrimary),
			Relation: string(eventbiz.EvidenceRelationSupports), SupportsFields: []string{eventbiz.EventFieldTitle},
			RawDocumentID: "EER30000000-0000-4000-8000-000000000002", SourceName: "Later Source", SourceType: "news",
			Title: "Later Article", FirstSeenAt: now.Add(time.Hour), KnowledgeAvailableAt: now.Add(time.Hour),
			AcceptedAt: now.Add(time.Hour), StatementSource: "extractor",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(`SELECT\s+jsonb_build_object`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), nil, nil, 2).
		WillReturnRows(sqlmock.NewRows([]string{"event", "evidence", "available"}).AddRow(eventPayload, evidencePayload, now))
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	page, err := store.ListResearchEvents(context.Background(), eventbiz.ResearchEventQuery{
		DiscoveryWindowStart: now.Add(-time.Hour), DiscoveryWindowEnd: now.Add(time.Hour),
		AnalysisAsOf: now, PageSize: 1,
	})
	if err != nil || len(page.Events) != 1 {
		t.Fatalf("ListResearchEvents() = %#v, %v", page, err)
	}
	page.Events[0].Event.Summary = ""
	if err := validateResearchEventRecord(page.Events[0]); err == nil {
		t.Fatal("persisted Research Event with an empty factual summary was accepted")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListActiveTagsRejectsInvalidPersistedRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectQuery(`SELECT id::text, tag_kind`).WillReturnRows(sqlmock.NewRows([]string{
		"id", "tag_kind", "code", "name", "is_active",
	}).AddRow("tag-1", "unknown", "technology", "Technology", true))
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.ListActiveTags(context.Background())
	if err == nil || !strings.Contains(err.Error(), "read active Event Tag invariant") {
		t.Fatalf("ListActiveTags() error = %v, want persisted invariant failure", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTransactionRejectsInvalidPersistedEvidenceRecord(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 8, 11, 1, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, artifact_id, content_hash`).WithArgs("artifact-1").WillReturnRows(sqlmock.NewRows([]string{
		"id", "artifact_id", "content_hash", "source_ref", "source_name", "source_type", "source_url", "title", "published_at", "collected_at", "language", "raw_mime_type",
	}).AddRow("raw-1", "different-artifact", strings.Repeat("a", 64), "source:1", "Source", "news", "", "Title", nil, now, "en", "text/plain"))
	mock.ExpectRollback()
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	err = store.InTransaction(context.Background(), func(tx eventbiz.Transaction) error {
		_, readErr := tx.StoredEventEvidenceRecord(context.Background(), "artifact-1")
		return readErr
	})
	if err == nil || !strings.Contains(err.Error(), "read Event Evidence Record invariant") {
		t.Fatalf("InTransaction() error = %v, want persisted invariant failure", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPersistedPublicationValidatorsRejectBrokenReferencesAndEnums(t *testing.T) {
	now := time.Date(2026, 8, 11, 1, 0, 0, 0, time.UTC)
	if err := validateStoredPublicationEvent(eventbiz.StoredEvent{
		ID: "event-1", DedupeKey: "event:key", Title: "Title", FactualSummary: "Summary",
		FactPayload: eventbiz.FactPayload{"metric": "value"}, FirstSeenAt: now, KnowableAt: now,
		EventStatus: eventbiz.EventStatusCandidate, FactStatus: eventbiz.FactStatusVerified,
	}, "event:key"); err == nil {
		t.Fatal("candidate persisted publication Event was accepted")
	}
	if err := validateStoredEvidenceLink(eventbiz.StoredEventEvidenceLink{
		ID: "link-1", EventID: "wrong", RawDocumentID: "raw-1", SourceLevel: "primary",
		EvidenceStatement: "statement", EvidenceHash: strings.Repeat("b", 64), EvidenceRelation: eventbiz.EvidenceRelationSupports,
		SupportsFields: []string{"title"},
	}, "event-1", "raw-1"); err == nil {
		t.Fatal("mismatched Event Evidence Link was accepted")
	}
	if err := validateStoredEvidenceLink(eventbiz.StoredEventEvidenceLink{
		ID: "link-1", EventID: "event-1", RawDocumentID: "raw-1", SourceLevel: "primary",
		EvidenceStatement: "statement", EvidenceHash: strings.Repeat("b", 64), EvidenceRelation: eventbiz.EvidenceRelationSupports,
		SupportsFields: []string{"title"},
	}, "event-1", "raw-1"); err == nil {
		t.Fatal("mismatched Event Evidence Link hash was accepted")
	}
	statementHash := sha256.Sum256([]byte("statement"))
	if err := validateStoredEvidenceLink(eventbiz.StoredEventEvidenceLink{
		ID: "link-1", EventID: "event-1", RawDocumentID: "raw-1", SourceLevel: eventbiz.EventSourceLevelPrimary,
		EvidenceStatement: "statement", EvidenceHash: hex.EncodeToString(statementHash[:]),
		SupportsFields: []string{eventbiz.EventFieldTitle},
	}, "event-1", "raw-1"); err == nil {
		t.Fatal("empty contract-v3 Event Evidence Link relation was accepted")
	}
	if err := validateStoredTagAssignment(eventbiz.StoredEventTagAssignment{
		ID: "map-1", EventID: "event-1", TagID: "tag-1", AssignSource: "ai",
		ReviewStatus: eventbiz.ReviewStatusCandidate, Confidence: "0.9", AssignmentReason: "reason",
	}, "event-1", "tag-1"); err == nil {
		t.Fatal("unapproved Event Tag Assignment was accepted")
	}
	if err := validateStoredTagAssignment(eventbiz.StoredEventTagAssignment{
		ID: "map-1", EventID: "event-1", TagID: "tag-1", AssignSource: "ai",
		ReviewStatus: eventbiz.ReviewStatusApproved, Confidence: "1.1", AssignmentReason: "reason",
	}, "event-1", "tag-1"); err == nil {
		t.Fatal("out-of-range Event Tag Assignment confidence was accepted")
	}
}

func TestPostgresEventAdapterRejectsCorruptedEvidenceHash(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := eventbiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	result, err := useCase.Import(context.Background(), "data-test", eventfixture.Publication("corrupted-hash"))
	if err != nil {
		t.Fatal(err)
	}
	eventID := result.Events[0].EventID
	rawDocumentID := result.RawDocuments[0].RawDocumentID
	if _, err := db.Exec(`UPDATE event_sources SET evidence_hash = $1 WHERE event_id = $2 AND raw_document_id = $3`, strings.Repeat("f", 64), eventID, rawDocumentID); err != nil {
		t.Fatal(err)
	}
	err = store.InTransaction(context.Background(), func(tx eventbiz.Transaction) error {
		_, readErr := tx.StoredEventEvidenceLink(context.Background(), eventID, rawDocumentID)
		return readErr
	})
	if err == nil || !strings.Contains(err.Error(), "hash does not match") {
		t.Fatalf("read corrupted Evidence Link error = %v", err)
	}
}

func openEventPublicationTestDatabase(t *testing.T) *sql.DB {
	return openEventPublicationTestDatabaseAt(t, 0)
}

func openEventPublicationTestDatabaseAt(t *testing.T, version int64) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_event_publication", migrationDir, version)
}
