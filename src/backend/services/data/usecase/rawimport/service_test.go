package rawimport

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
	"github.com/meierlink88/tidewise-ai/backend/services/data/repositories"
)

const (
	testSourceID = "22222222-2222-5222-8222-222222222222"
)

func TestPlanFreezesCanonicalHashAndReceiptIdentity(t *testing.T) {
	service := NewService(&fakeStore{}, nil)
	plan, err := service.Plan("agent-run", "batch-1", validBatch())
	if err != nil {
		t.Fatal(err)
	}
	const wantHash = "8c02bdd23978cce794a845b722331c17609b6c5a6e668615fb732ad480307913"
	if plan.PayloadHash != wantHash {
		t.Fatalf("payload hash = %q, want %q", plan.PayloadHash, wantHash)
	}
	wantReceipt := repositories.NormalizeUUID("raw_document_import_receipt", "agent-run", "batch-1")
	if plan.ReceiptID != wantReceipt {
		t.Fatalf("receipt id = %q, want %q", plan.ReceiptID, wantReceipt)
	}
	if plan.Version != CanonicalVersion {
		t.Fatalf("canonical version = %q, want %q", plan.Version, CanonicalVersion)
	}

	reordered := validBatch()
	reordered.Items = append(reordered.Items, secondCandidate())
	first, err := service.Plan("agent-run", "batch-1", reordered)
	if err != nil {
		t.Fatal(err)
	}
	reordered.Items[0], reordered.Items[1] = reordered.Items[1], reordered.Items[0]
	second, err := service.Plan("agent-run", "batch-1", reordered)
	if err != nil {
		t.Fatal(err)
	}
	if first.PayloadHash == second.PayloadHash {
		t.Fatal("candidate order must change the canonical payload hash")
	}
}

func TestPlanRejectsCallerKeyBoundsAndInvalidBatchBeforeTransaction(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, nil)
	for _, test := range []struct {
		name   string
		caller string
		key    string
		batch  Batch
	}{
		{name: "blank caller", key: "key", batch: validBatch()},
		{name: "long caller", caller: strings.Repeat("c", 201), key: "key", batch: validBatch()},
		{name: "blank key", caller: "agent-run", batch: validBatch()},
		{name: "long key", caller: "agent-run", key: strings.Repeat("k", 201), batch: validBatch()},
		{name: "empty batch", caller: "agent-run", key: "key", batch: Batch{}},
		{name: "oversized batch", caller: "agent-run", key: "key", batch: Batch{Items: make([]Candidate, MaxBatchItems+1)}},
		{name: "invalid item", caller: "agent-run", key: "key", batch: Batch{Items: []Candidate{{SourceID: testSourceID}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := service.Import(context.Background(), test.caller, test.key, test.batch); ErrorCode(err) != CodeInvalidRequest {
				t.Fatalf("error = %v, code = %q, want %q", err, ErrorCode(err), CodeInvalidRequest)
			}
		})
	}
	if store.transactionCalls != 0 || store.insertCalls != 0 {
		t.Fatalf("invalid requests reached transaction/DML: transactions=%d inserts=%d", store.transactionCalls, store.insertCalls)
	}
}

func TestImportReplaysStoredResultBeforeMutableSourceValidation(t *testing.T) {
	importedAt := time.Date(2026, 7, 17, 2, 0, 0, 0, time.UTC)
	service := NewService(&fakeStore{}, func() time.Time { return importedAt })
	plan, err := service.Plan("agent-run", "batch-1", validBatch())
	if err != nil {
		t.Fatal(err)
	}
	stored := Result{
		ReceiptID: plan.ReceiptID, PayloadHash: plan.PayloadHash,
		RawDocumentIDs: []string{testRawID()},
		Items:          []ItemResult{{RawDocumentID: testRawID(), Disposition: DispositionCreated}},
		ImportedAt:     importedAt,
	}
	store := &fakeStore{receipt: &Receipt{
		ID: plan.ReceiptID, CallerIdentity: "agent-run", IdempotencyKey: "batch-1",
		PayloadHash: plan.PayloadHash, RawDocumentIDs: stored.RawDocumentIDs, Result: stored, ImportedAt: importedAt,
	}}
	service = NewService(store, func() time.Time { return importedAt.Add(time.Hour) })

	got, err := service.Import(context.Background(), "agent-run", "batch-1", validBatch())
	if err != nil {
		t.Fatal(err)
	}
	if !got.Replayed {
		t.Fatal("replayed result is not marked as a transport replay")
	}
	got.Replayed = false
	if !reflect.DeepEqual(got, stored) {
		t.Fatalf("replay result = %#v, want stored %#v", got, stored)
	}
	if store.sourceCalls != 0 || store.rawLockCalls != 0 || store.insertCalls != 0 {
		t.Fatalf("replay reran mutable work: source=%d locks=%d inserts=%d", store.sourceCalls, store.rawLockCalls, store.insertCalls)
	}

	changed := validBatch()
	changed.Items[0].Title = "changed"
	if _, err := service.Import(context.Background(), "agent-run", "batch-1", changed); ErrorCode(err) != CodeIdempotencyConflict {
		t.Fatalf("changed replay error = %v, code = %q", err, ErrorCode(err))
	}
	if store.insertCalls != 0 {
		t.Fatalf("changed replay performed %d inserts", store.insertCalls)
	}
}

func TestImportCreatesAtomicOrderedResultAndStatus(t *testing.T) {
	importedAt := time.Date(2026, 7, 17, 3, 0, 0, 0, time.UTC)
	store := &fakeStore{
		sources: map[string]domain.SourceCatalog{
			testSourceID: activeSource(),
		},
	}
	service := NewService(store, func() time.Time { return importedAt })
	result, err := service.Import(context.Background(), "agent-run", "batch-1", validBatch())
	if err != nil {
		t.Fatal(err)
	}
	if result.ReceiptID == "" || len(result.RawDocumentIDs) != 1 || result.RawDocumentIDs[0] != testRawID() {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Items[0].Disposition != DispositionCreated || !result.ImportedAt.Equal(importedAt) {
		t.Fatalf("unexpected item/timestamp: %#v", result)
	}
	if store.committedReceipt == nil || !reflect.DeepEqual(store.committedReceipt.Result, result) {
		t.Fatalf("receipt did not store exact result: %#v", store.committedReceipt)
	}
	if !sortStringsAreStrict(store.lockTexts) {
		t.Fatalf("raw lock texts are not sorted/deduplicated: %v", store.lockTexts)
	}

	store.statusReceipt = store.committedReceipt
	status, err := service.Status(context.Background(), "agent-run", "batch-1")
	if err != nil {
		t.Fatal(err)
	}
	if status.State != StatusCompleted || status.Result == nil || !reflect.DeepEqual(*status.Result, result) {
		t.Fatalf("completed status = %#v", status)
	}
	store.statusReceipt = nil
	status, err = service.Status(context.Background(), "agent-run", "missing")
	if err != nil {
		t.Fatal(err)
	}
	if status.State != StatusUnknown || status.Result != nil {
		t.Fatalf("unknown status = %#v", status)
	}
}

func TestImportValidatesEveryCandidateAgainstCachedSourceAttribution(t *testing.T) {
	batch := validBatch()
	second := secondCandidate()
	second.IngestChannel = "api"
	batch.Items = append(batch.Items, second)
	store := &fakeStore{sources: map[string]domain.SourceCatalog{testSourceID: activeSource()}}

	_, err := NewService(store, nil).Import(context.Background(), "agent-run", "attribution-mismatch", batch)
	if ErrorCode(err) != CodeInvalidRequest {
		t.Fatalf("same-source attribution mismatch error = %v, code = %q", err, ErrorCode(err))
	}
	if store.sourceCalls != 1 {
		t.Fatalf("source lookups = %d, want one cached lookup", store.sourceCalls)
	}
	if store.rawLockCalls != 0 || store.insertCalls != 0 || store.committedReceipt != nil {
		t.Fatalf("invalid attribution performed mutable work: locks=%d inserts=%d receipt=%#v", store.rawLockCalls, store.insertCalls, store.committedReceipt)
	}
}

func TestImportDoesNotClassifySourceStoreFailureAsCallerValidation(t *testing.T) {
	store := &fakeStore{sourceErr: errors.New("pq: connection failed for password=must-not-leak")}
	_, err := NewService(store, nil).Import(context.Background(), "agent-run", "source-store-failure", validBatch())
	if err == nil || ErrorCode(err) != "" {
		t.Fatalf("source store error = %v, code = %q; want unclassified internal error", err, ErrorCode(err))
	}
	if store.rawLockCalls != 0 || store.insertCalls != 0 || store.committedReceipt != nil {
		t.Fatalf("source store failure performed mutable work: locks=%d inserts=%d receipt=%#v", store.rawLockCalls, store.insertCalls, store.committedReceipt)
	}
}

func TestImportRejectsDivergentIdentityAndCollapsedBatch(t *testing.T) {
	firstID := "aaaaaaaa-aaaa-5aaa-8aaa-aaaaaaaaaaaa"
	secondID := "bbbbbbbb-bbbb-5bbb-8bbb-bbbbbbbbbbbb"
	store := &fakeStore{
		sources:  map[string]domain.SourceCatalog{testSourceID: activeSource()},
		external: map[string]string{identityKey(testSourceID, "story-1"): firstID},
		hashes:   map[string]string{identityKey(testSourceID, strings.Repeat("a", 64)): secondID},
	}
	service := NewService(store, nil)
	if _, err := service.Import(context.Background(), "agent-run", "conflict", validBatch()); ErrorCode(err) != CodeIdentityConflict {
		t.Fatalf("identity conflict error = %v, code = %q", err, ErrorCode(err))
	}
	if store.committedReceipt != nil {
		t.Fatal("identity conflict inserted a receipt")
	}

	collapsed := validBatch()
	collapsed.Items = append(collapsed.Items, secondCandidate())
	store = &fakeStore{
		sources: map[string]domain.SourceCatalog{testSourceID: activeSource()},
		external: map[string]string{
			identityKey(testSourceID, "story-1"): firstID,
			identityKey(testSourceID, "story-2"): firstID,
		},
		hashes: map[string]string{
			identityKey(testSourceID, strings.Repeat("a", 64)): firstID,
			identityKey(testSourceID, strings.Repeat("b", 64)): firstID,
		},
	}
	service = NewService(store, nil)
	if _, err := service.Import(context.Background(), "agent-run", "collision", collapsed); ErrorCode(err) != CodeBatchCollision {
		t.Fatalf("batch collision error = %v, code = %q", err, ErrorCode(err))
	}
	if store.committedReceipt != nil {
		t.Fatal("batch collision inserted a receipt")
	}
}

func validBatch() Batch {
	published := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)
	return Batch{Items: []Candidate{{
		SourceID: testSourceID, SourceExternalID: "story-1",
		IngestChannel: "rss", SourceType: "news", SourceName: "Example", SourceURL: "https://example.test/feed",
		Title: "Title", ContentText: "Body", ContentLevel: "full", Language: "en",
		PublishedAt: &published, CollectedAt: time.Date(2026, 7, 17, 1, 0, 0, 0, time.UTC),
		ContentHash: strings.Repeat("a", 64),
	}}}
}

func secondCandidate() Candidate {
	candidate := validBatch().Items[0]
	candidate.SourceExternalID = "story-2"
	candidate.Title = "Second"
	candidate.ContentHash = strings.Repeat("b", 64)
	return candidate
}

func activeSource() domain.SourceCatalog {
	return domain.SourceCatalog{ID: testSourceID, IngestChannel: "rss", SourceType: "news", SourceName: "Example", SourceURL: "https://example.test/feed", Status: domain.SourceCatalogStatusActive}
}

type fakeStore struct {
	receipt          *Receipt
	statusReceipt    *Receipt
	committedReceipt *Receipt
	sources          map[string]domain.SourceCatalog
	sourceErr        error
	external         map[string]string
	hashes           map[string]string
	transactionCalls int
	sourceCalls      int
	rawLockCalls     int
	insertCalls      int
	lockTexts        []string
}

func (s *fakeStore) InRawImportTransaction(ctx context.Context, fn func(Transaction) error) error {
	s.transactionCalls++
	return fn((*fakeTransaction)(s))
}

func (s *fakeStore) RawImportReceipt(context.Context, string, string) (*Receipt, error) {
	return s.statusReceipt, nil
}

type fakeTransaction fakeStore

func (t *fakeTransaction) LockReceipt(context.Context, string, string, string) (*Receipt, error) {
	return t.receipt, nil
}

func (t *fakeTransaction) Source(_ context.Context, sourceID string) (domain.SourceCatalog, error) {
	t.sourceCalls++
	if t.sourceErr != nil {
		return domain.SourceCatalog{}, t.sourceErr
	}
	return t.sources[sourceID], nil
}

func (t *fakeTransaction) LockRawIdentities(_ context.Context, texts []string) error {
	t.rawLockCalls++
	t.lockTexts = append([]string(nil), texts...)
	return nil
}

func (t *fakeTransaction) RawDocumentByExternalID(_ context.Context, sourceID, externalID string) (string, error) {
	return t.external[identityKey(sourceID, externalID)], nil
}

func (t *fakeTransaction) RawDocumentByContentHash(_ context.Context, sourceID, hash string) (string, error) {
	return t.hashes[identityKey(sourceID, hash)], nil
}

func (t *fakeTransaction) InsertRawDocument(_ context.Context, document domain.RawDocument) (bool, error) {
	t.insertCalls++
	if t.external == nil {
		t.external = map[string]string{}
	}
	if t.hashes == nil {
		t.hashes = map[string]string{}
	}
	if document.SourceExternalID != "" {
		t.external[identityKey(document.SourceID, document.SourceExternalID)] = document.ID
	}
	t.hashes[identityKey(document.SourceID, document.ContentHash)] = document.ID
	return true, nil
}

func (t *fakeTransaction) InsertReceipt(_ context.Context, receipt Receipt) error {
	copy := receipt
	t.committedReceipt = &copy
	return nil
}

func identityKey(left, right string) string { return left + "\x00" + right }

func testRawID() string {
	return repositories.RawDocumentUUID(testSourceID, "", "story-1", strings.Repeat("a", 64))
}

func sortStringsAreStrict(values []string) bool {
	for index := 1; index < len(values); index++ {
		if values[index-1] >= values[index] {
			return false
		}
	}
	return true
}
