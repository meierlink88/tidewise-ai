package research

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	researchbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/research"
)

func TestResearchReceiptAdapterRejectsMalformedPersistedRows(t *testing.T) {
	now := time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC)
	valid := researchbiz.Receipt{
		ID: "RTI11111111-1111-4111-8111-111111111111", AnalysisBatchID: "batch:one",
		PublisherSubject: "agentos", PayloadHash: strings.Repeat("a", 64),
		ThemeID: "RTH22222222-2222-4222-8222-222222222222", ThemeKey: "theme:one",
		ContractVersion: 3, PublicationMode: researchbiz.SnapshotPublicationMode,
		ReasoningTreeIDsByIndustryChainEntityID: map[string]string{},
		ReasoningTreeIDsByTreeKey:               map[string]string{"tree:one": "RRT33333333-3333-4333-8333-333333333333"},
		Counts:                                  researchbiz.Counts{Themes: 1, Impacts: 1, ReasoningTrees: 1, Nodes: 1, SignalAssociations: 1, Receipts: 2},
		PublishedAt:                             now, ImportedAt: now,
	}
	if err := validatePersistedResearchReceipt(valid); err != nil {
		t.Fatalf("valid persisted Receipt rejected: %v", err)
	}
	invalid := valid
	invalid.PayloadHash = strings.Repeat("A", 64)
	if err := validatePersistedResearchReceipt(invalid); err == nil {
		t.Fatal("malformed persisted Receipt hash was accepted")
	}
	invalid = valid
	invalid.Counts.ReasoningTrees = 2
	if err := validatePersistedResearchReceipt(invalid); err == nil {
		t.Fatal("inconsistent persisted Receipt identity count was accepted")
	}
}

func TestPostgresResearchPublicationTransactionObservesCancellation(t *testing.T) {
	db := openResearchV1TestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	err = store.InResearchPublicationTransaction(ctx, func(tx researchbiz.PublicationTransaction) error {
		cancel()
		return tx.Lock(ctx, "canceled-research-publication")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled transaction error = %v, want context.Canceled", err)
	}
}

func TestResearchPublicationTransactionCommitsOrRollsBackTheWholeAggregate(t *testing.T) {
	for _, test := range []struct {
		name      string
		callback  error
		expectEnd func(sqlmock.Sqlmock)
	}{
		{name: "commit", expectEnd: func(mock sqlmock.Sqlmock) { mock.ExpectCommit() }},
		{name: "rollback", callback: errors.New("child write failed"), expectEnd: func(mock sqlmock.Sqlmock) { mock.ExpectRollback() }},
	} {
		t.Run(test.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mock.ExpectBegin()
			test.expectEnd(mock)
			store := Store{db: db}
			err = store.InResearchPublicationTransaction(context.Background(), func(_ researchbiz.PublicationTransaction) error {
				return test.callback
			})
			if !errors.Is(err, test.callback) {
				t.Fatalf("transaction error = %v, want %v", err, test.callback)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
