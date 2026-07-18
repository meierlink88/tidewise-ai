package eventimport

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
	domainimport "github.com/meierlink88/tidewise-ai/backend/services/data/domain/eventimport"
)

func TestServiceImportsOneReviewedEventAndPersistsReceipt(t *testing.T) {
	store := newFakeStore()
	service := NewService(store)
	pkg := validPackage()

	result, err := service.Import(context.Background(), pkg)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.EventID == "" || result.RawDocumentIDs[0] == "" || result.ReceiptID == "" {
		t.Fatalf("result IDs = %#v", result)
	}
	if got := store.tx.event.EventStatus; got != domain.EventStatusConfirmed {
		t.Fatalf("event status = %q, want confirmed", got)
	}
	if got := store.tx.event.FactStatus; got != domain.FactStatusVerified {
		t.Fatalf("fact status = %q, want verified", got)
	}
	if store.tx.commitCount != 1 || store.tx.rollbackCount != 0 {
		t.Fatalf("commit/rollback = %d/%d", store.tx.commitCount, store.tx.rollbackCount)
	}
}

func TestServiceReplaysSameKeyAndRejectsDifferentHash(t *testing.T) {
	store := newFakeStore()
	service := NewService(store)
	pkg := validPackage()

	first, err := service.Import(context.Background(), pkg)
	if err != nil {
		t.Fatalf("first Import() error = %v", err)
	}
	second, err := service.Import(context.Background(), pkg)
	if err != nil {
		t.Fatalf("replay Import() error = %v", err)
	}
	if !second.Replayed {
		t.Fatal("replay result is not marked as a transport replay")
	}
	second.Replayed = false
	if !reflect.DeepEqual(second, first) {
		t.Fatalf("replay result = %#v, want %#v", second, first)
	}

	pkg.Event.Title = "changed payload"
	if _, err := service.Import(context.Background(), pkg); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("conflict error = %v, want ErrIdempotencyConflict", err)
	}

	pkg = validPackage()
	pkg.PackageID = "pkg-2"
	pkg.Review.PackageID = pkg.PackageID
	if _, err := service.Import(context.Background(), pkg); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("identity-changing conflict error = %v, want ErrIdempotencyConflict", err)
	}
}

func TestServiceReplayValidatesReceiptResultIDs(t *testing.T) {
	store := newFakeStore()
	service := NewService(store)
	pkg := validPackage()
	if _, err := service.Import(context.Background(), pkg); err != nil {
		t.Fatal(err)
	}
	store.tx.receipt.RawDocumentIDs = []string{"00000000-0000-0000-0000-000000000000"}
	if _, err := service.Import(context.Background(), pkg); err == nil || !strings.Contains(err.Error(), "deterministic plan") {
		t.Fatalf("replay error = %v, want deterministic plan mismatch", err)
	}
	if store.tx.verifyCalls != 0 {
		t.Fatalf("verify calls = %d, want 0 for invalid receipt", store.tx.verifyCalls)
	}
}

func TestServiceAcceptsManualReviewWithoutSecondApproval(t *testing.T) {
	store := newFakeStore()
	service := NewService(store)
	pkg := validPackage()
	pkg.Review.Decision = "manual_review"
	pkg.Review.EventStatus = "candidate"
	pkg.Review.FactStatus = "unverified"
	pkg.Event.EventStatus = "candidate"
	pkg.Event.FactStatus = "unverified"

	if _, err := service.Import(context.Background(), pkg); err != nil {
		t.Fatalf("manual review Import() error = %v", err)
	}
}

func TestServiceRollsBackWhenTagIdentityIsUnknown(t *testing.T) {
	store := newFakeStore()
	store.tx.tag = domain.EventTagDef{}
	service := NewService(store)

	if _, err := service.Import(context.Background(), validPackage()); err == nil {
		t.Fatal("Import() error = nil, want unknown tag")
	}
	if store.tx.commitCount != 0 || store.tx.rollbackCount != 1 {
		t.Fatalf("commit/rollback = %d/%d", store.tx.commitCount, store.tx.rollbackCount)
	}
}

func TestServiceRejectsInactiveDatabaseTagAndRollsBack(t *testing.T) {
	store := newFakeStore()
	store.tx.tagActive = false
	if _, err := NewService(store).Import(context.Background(), validPackage()); err == nil {
		t.Fatal("Import() error = nil, want inactive tag rejection")
	}
	if store.tx.commitCount != 0 || store.tx.rollbackCount != 1 {
		t.Fatalf("commit/rollback = %d/%d", store.tx.commitCount, store.tx.rollbackCount)
	}
}

func TestServiceRejectsDuplicateRepositoryResultIDsAndRollsBack(t *testing.T) {
	store := newFakeStore()
	pkg := validPackage()
	second := pkg.RawDocuments[0]
	second.DocumentID = "sha256:doc-2"
	second.ContentHash = "fedcba9876543210"
	pkg.RawDocuments = append(pkg.RawDocuments, second)
	plan, err := NewService(store).Plan(pkg)
	if err != nil {
		t.Fatal(err)
	}
	store.tx.rawResults = []string{plan.RawDocumentIDs[0], plan.RawDocumentIDs[0]}
	if _, err := NewService(store).Import(context.Background(), pkg); err == nil || !strings.Contains(err.Error(), "duplicate raw document") {
		t.Fatalf("Import() error = %v, want duplicate raw document rejection", err)
	}
	if store.tx.commitCount != 0 || store.tx.rollbackCount != 1 {
		t.Fatalf("commit/rollback = %d/%d", store.tx.commitCount, store.tx.rollbackCount)
	}
}

func validPackage() domainimport.Package {
	collected := time.Date(2026, 7, 16, 7, 3, 49, 0, time.UTC)
	published := time.Date(2026, 7, 15, 1, 36, 49, 0, time.UTC)
	return domainimport.Package{
		IdempotencyKey: "idem-1",
		PackageID:      "pkg-1",
		RawDocuments: []domainimport.RawDocumentInput{{
			DocumentID: "sha256:doc-1", SourceName: "Example", SourceURL: "https://example.com/a", Title: "Document", ContentText: "Body", ContentLevel: "summary", PublishedAt: &published, CollectedAt: collected, ContentHash: "0123456789abcdef",
		}},
		Event:        domainimport.EventInput{DedupeKey: "event:v1:example", Title: "Event", FactualSummary: "A verifiable fact", FactStatus: "verified", EventStatus: "confirmed", FactPayload: map[string]any{"amount": 1}},
		EventSources: []domainimport.EventSourceInput{{DocumentID: "sha256:doc-1", EvidenceExcerpt: "Evidence", SourceURL: "https://example.com/a", EvidenceRelation: "supports", SupportsFields: []string{"title", "factual_summary"}, SourceLevel: "secondary", ContentLevel: "summary", EvidenceHash: "sha256:evidence-1"}},
		EventTags:    []domainimport.EventTagInput{{TagID: "b0fe1994-0db2-526c-a57f-97fa73c1b595", TagKind: "news_category", TagCode: "geopolitics", Confidence: "0.98", ReviewStatus: "approved", AssignmentReason: "material fact", AssignSource: "ai"}},
		Review:       domainimport.ReviewInput{ReviewID: "review-1", PackageID: "pkg-1", Decision: "auto_approved", EventStatus: "confirmed", FactStatus: "verified", EvidenceGrade: "single_source", Reasons: []string{"accepted"}, ComponentVersions: map[string]string{"review_policy": "v2"}},
	}
}

type fakeStore struct{ tx *fakeTx }

func newFakeStore() *fakeStore {
	return &fakeStore{tx: &fakeTx{source: domain.SourceCatalog{ID: FixedSourceID, Status: domain.SourceCatalogStatusActive, IngestChannel: "agent_reviewed_outbox", SourceType: "event_agent_reviewed_outbox"}, tag: domain.EventTagDef{ID: "b0fe1994-0db2-526c-a57f-97fa73c1b595", TagKind: "news_category", Code: "geopolitics", Name: "地缘政治"}, tagActive: true}}
}

func (s *fakeStore) InTransaction(_ context.Context, fn func(Transaction) error) error {
	if err := fn(s.tx); err != nil {
		s.tx.rollbackCount++
		return err
	}
	s.tx.commitCount++
	return nil
}

type fakeTx struct {
	source        domain.SourceCatalog
	tag           domain.EventTagDef
	tagActive     bool
	rawResult     string
	rawResults    []string
	rawCalls      int
	verifyCalls   int
	event         domain.Event
	receipt       *Receipt
	commitCount   int
	rollbackCount int
}

func (f *fakeTx) LockReceipt(_ context.Context, key string) (*Receipt, error) {
	if f.receipt == nil || f.receipt.IdempotencyKey != key {
		return nil, nil
	}
	copy := *f.receipt
	return &copy, nil
}
func (f *fakeTx) Source(_ context.Context, id string) (domain.SourceCatalog, error) {
	if id != f.source.ID || f.source.Status != domain.SourceCatalogStatusActive {
		return domain.SourceCatalog{}, errors.New("source unavailable")
	}
	return f.source, nil
}
func (f *fakeTx) UpsertRawDocument(_ context.Context, doc domain.RawDocument) (string, error) {
	if f.rawCalls < len(f.rawResults) {
		result := f.rawResults[f.rawCalls]
		f.rawCalls++
		return result, nil
	}
	if f.rawResult != "" {
		return f.rawResult, nil
	}
	return doc.ID, nil
}
func (f *fakeTx) VerifyReceiptResults(_ context.Context, _ Receipt) error {
	f.verifyCalls++
	return nil
}
func (f *fakeTx) UpsertEvent(_ context.Context, event domain.Event) (string, error) {
	f.event = event
	return event.ID, nil
}
func (f *fakeTx) AddEventSource(_ context.Context, source domain.EventSource) (string, error) {
	return source.ID, nil
}
func (f *fakeTx) Tag(_ context.Context, id, kind, code string) (domain.EventTagDef, error) {
	if !f.tagActive {
		return domain.EventTagDef{}, errors.New("tag inactive")
	}
	if f.tag.ID != id || f.tag.TagKind != kind || f.tag.Code != code {
		return domain.EventTagDef{}, errors.New("tag mismatch")
	}
	return f.tag, nil
}
func (f *fakeTx) AssignEventTag(_ context.Context, tagMap domain.EventTagMap) (string, error) {
	return tagMap.ID, nil
}
func (f *fakeTx) InsertReceipt(_ context.Context, receipt Receipt) error {
	f.receipt = &receipt
	return nil
}
