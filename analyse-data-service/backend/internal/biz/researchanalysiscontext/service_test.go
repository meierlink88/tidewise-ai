package researchanalysiscontext

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

type contextStoreStub struct {
	query        StoreQuery
	page         StorePage
	dictionaries Dictionaries
}

func (s *contextStoreStub) ListBundles(
	_ context.Context,
	query StoreQuery,
) (StorePage, error) {
	s.query = query
	return s.page, nil
}

func (s *contextStoreStub) ReferenceClosure(
	_ context.Context,
	_ ReferenceClosureQuery,
) (Dictionaries, error) {
	return s.dictionaries, nil
}

func TestServiceReturnsStableCursorBoundToTheResearchWindow(t *testing.T) {
	store := &contextStoreStub{page: StorePage{
		Bundles: []BundleRecord{
			{
				KnowledgeAvailableAt: time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC),
				EventID:              "11111111-1111-4111-8111-111111111111",
				Bundle: EventSemanticBundle{Event: Event{
					ID: "11111111-1111-4111-8111-111111111111",
				}},
			},
			{
				KnowledgeAvailableAt: time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC),
				EventID:              "22222222-2222-4222-8222-222222222222",
				Bundle: EventSemanticBundle{Event: Event{
					ID: "22222222-2222-4222-8222-222222222222",
				}},
			},
		},
		HasMore: true,
	}}
	service := NewService(store)
	request := Request{
		DiscoveryWindowStart: "2026-07-28T00:00:00Z",
		DiscoveryWindowEnd:   "2026-07-29T00:00:00Z",
		AnalysisAsOf:         "2026-07-29T00:00:00Z",
		PageSize:             2,
	}

	first, err := service.List(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !first.HasMore || first.NextCursor == "" || len(first.EventSemanticBundles) != 2 {
		t.Fatalf("first page = %#v", first)
	}

	store.page = StorePage{}
	request.Cursor = first.NextCursor
	second, err := service.List(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if store.query.AfterEventID != "22222222-2222-4222-8222-222222222222" ||
		store.query.AfterKnowledgeAvailableAt == nil ||
		!store.query.AfterKnowledgeAvailableAt.Equal(time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("continuation query = %#v", store.query)
	}
	if second.EventSemanticBundles == nil || second.HasMore || second.NextCursor != "" {
		t.Fatalf("empty continuation = %#v", second)
	}
}

func TestServiceReturnsVersionedPageAndReferenceClosureFingerprints(t *testing.T) {
	store := &contextStoreStub{page: StorePage{
		Bundles: []BundleRecord{},
	}}
	result, err := NewService(store).List(context.Background(), Request{
		DiscoveryWindowStart: "2026-07-28T00:00:00Z",
		DiscoveryWindowEnd:   "2026-07-29T00:00:00Z",
		AnalysisAsOf:         "2026-07-29T00:00:00Z",
		PageSize:             20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ContractVersion != "research-analysis-context.v1" ||
		result.TBoxContractVersion != "event-semantics.phase-one@1" {
		t.Fatalf(
			"versions = contract %q TBox %q",
			result.ContractVersion,
			result.TBoxContractVersion,
		)
	}
	if !testHashPattern(result.EventPageFingerprint) ||
		!testHashPattern(result.ReferenceClosureFingerprint) {
		t.Fatalf(
			"fingerprints = event %q closure %q",
			result.EventPageFingerprint,
			result.ReferenceClosureFingerprint,
		)
	}
	if result.Dictionaries.Entities == nil ||
		result.EventSemanticBundles == nil {
		t.Fatalf("empty page must preserve empty arrays: %#v", result)
	}
}

func TestServiceRejectsInvalidResearchTimeBoundariesAndCursorMismatch(t *testing.T) {
	service := NewService(&contextStoreStub{})
	valid := Request{
		DiscoveryWindowStart: "2026-07-28T00:00:00Z",
		DiscoveryWindowEnd:   "2026-07-29T00:00:00Z",
		AnalysisAsOf:         "2026-07-29T00:00:00Z",
		PageSize:             20,
	}
	cases := []Request{
		{
			DiscoveryWindowStart: valid.DiscoveryWindowStart,
			DiscoveryWindowEnd:   "2026-07-30T00:00:00Z",
			AnalysisAsOf:         valid.AnalysisAsOf,
			PageSize:             valid.PageSize,
		},
		{
			DiscoveryWindowStart:   valid.DiscoveryWindowStart,
			DiscoveryWindowEnd:     valid.DiscoveryWindowEnd,
			AnalysisAsOf:           valid.AnalysisAsOf,
			PredictionHorizonStart: stringPointer("2026-07-28T12:00:00Z"),
			PredictionHorizonEnd:   stringPointer("2026-07-30T00:00:00Z"),
			PageSize:               valid.PageSize,
		},
		{
			DiscoveryWindowStart: "2025-07-27T00:00:00Z",
			DiscoveryWindowEnd:   valid.DiscoveryWindowEnd,
			AnalysisAsOf:         valid.AnalysisAsOf,
			PageSize:             valid.PageSize,
		},
	}
	for _, request := range cases {
		if _, err := service.List(context.Background(), request); err == nil {
			t.Fatalf("request unexpectedly accepted: %#v", request)
		}
	}

	store := &contextStoreStub{page: StorePage{Bundles: []BundleRecord{{
		KnowledgeAvailableAt: time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC),
		EventID:              "11111111-1111-4111-8111-111111111111",
		Bundle: EventSemanticBundle{Event: Event{
			ID: "11111111-1111-4111-8111-111111111111",
		}},
	}}, HasMore: true}}
	service = NewService(store)
	first, err := service.List(context.Background(), valid)
	if err != nil {
		t.Fatal(err)
	}
	valid.Cursor = first.NextCursor
	valid.DiscoveryWindowStart = "2026-07-27T00:00:00Z"
	if _, err := service.List(context.Background(), valid); err == nil {
		t.Fatal("cursor accepted with a changed discovery window")
	}
}

func TestServiceKeepsCursorValidAfterUnrelatedDictionaryChanges(t *testing.T) {
	store := &contextStoreStub{page: StorePage{
		Bundles: []BundleRecord{{
			KnowledgeAvailableAt: time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC),
			EventID:              "11111111-1111-4111-8111-111111111111",
			Bundle: EventSemanticBundle{Event: Event{
				ID: "11111111-1111-4111-8111-111111111111",
			}},
		}},
		HasMore: true,
	}}
	service := NewService(store)
	request := Request{
		DiscoveryWindowStart: "2026-07-28T00:00:00Z",
		DiscoveryWindowEnd:   "2026-07-29T00:00:00Z",
		AnalysisAsOf:         "2026-07-29T00:00:00Z",
		PageSize:             20,
	}
	first, err := service.List(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	request.Cursor = first.NextCursor
	store.page = StorePage{}
	store.dictionaries = Dictionaries{Entities: []Entity{{
		EntityID:   "33333333-3333-4333-8333-333333333333",
		EntityType: "company",
	}}, EntityTypeDefinitions: []EntityTypeContext{{
		TypeKey: "company",
	}}}
	if _, err := service.List(context.Background(), request); err != nil {
		t.Fatalf("cursor was invalidated by an unrelated dictionary change: %v", err)
	}
}

func TestServiceRequiresEveryEntityTypeReferenceInThePageClosure(t *testing.T) {
	store := &contextStoreStub{
		page: StorePage{},
		dictionaries: Dictionaries{
			Entities: []Entity{{
				EntityID:   "33333333-3333-4333-8333-333333333333",
				EntityType: "company",
			}},
		},
	}
	request := Request{
		DiscoveryWindowStart: "2026-07-28T00:00:00Z",
		DiscoveryWindowEnd:   "2026-07-29T00:00:00Z",
		AnalysisAsOf:         "2026-07-29T00:00:00Z",
		PageSize:             20,
	}

	if _, err := NewService(store).List(context.Background(), request); !errors.Is(
		err, ErrReferenceClosureInconsistent,
	) {
		t.Fatalf("error = %v, want unresolved EntityTypeContext", err)
	}
}

func TestServiceRejectsCursorWithInvalidTerminalEventID(t *testing.T) {
	request := Request{
		DiscoveryWindowStart: "2026-07-28T00:00:00Z",
		DiscoveryWindowEnd:   "2026-07-29T00:00:00Z",
		AnalysisAsOf:         "2026-07-29T00:00:00Z",
		PageSize:             20,
	}
	_, _, fingerprint, err := validateRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	request.Cursor, err = encodeCursor(contextCursor{
		Version:              1,
		Fingerprint:          fingerprint,
		KnowledgeAvailableAt: "2026-07-28T09:00:00Z",
		EventID:              "not-a-uuid",
	})
	if err != nil {
		t.Fatal(err)
	}

	store := &contextStoreStub{page: StorePage{}}
	if _, err := NewService(store).List(context.Background(), request); err == nil {
		t.Fatal("cursor with an invalid terminal Event ID was accepted")
	}
}

func TestServiceRequiresRestartWhenPageReferenceClosureIsInconsistent(t *testing.T) {
	eventID := "11111111-1111-4111-8111-111111111111"
	store := &contextStoreStub{page: StorePage{
		Bundles: []BundleRecord{{
			KnowledgeAvailableAt: time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC),
			EventID:              eventID,
			Bundle: EventSemanticBundle{
				Event: Event{ID: eventID},
				EntityLinks: []EntityLink{{
					EventEntityLinkID: "22222222-2222-4222-8222-222222222222",
					EntityID:          "33333333-3333-4333-8333-333333333333",
				}},
			},
		}},
	}}
	request := Request{
		DiscoveryWindowStart: "2026-07-28T00:00:00Z",
		DiscoveryWindowEnd:   "2026-07-29T00:00:00Z",
		AnalysisAsOf:         "2026-07-29T00:00:00Z",
		PageSize:             20,
	}

	if _, err := NewService(store).List(context.Background(), request); !errors.Is(
		err, ErrReferenceClosureInconsistent,
	) {
		t.Fatalf("error = %v, want reference closure inconsistency", err)
	}
}

func TestServiceFailsClosedForBundleClosureAndPageBudgets(t *testing.T) {
	request := Request{
		DiscoveryWindowStart: "2026-07-28T00:00:00Z",
		DiscoveryWindowEnd:   "2026-07-29T00:00:00Z",
		AnalysisAsOf:         "2026-07-29T00:00:00Z",
		PageSize:             20,
	}
	tests := []struct {
		name          string
		store         *contextStoreStub
		wantComponent string
	}{
		{
			name: "complete Event Bundle",
			store: &contextStoreStub{page: StorePage{Bundles: []BundleRecord{{
				KnowledgeAvailableAt: time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC),
				EventID:              "11111111-1111-4111-8111-111111111111",
				Bundle: EventSemanticBundle{Event: Event{
					ID:      "11111111-1111-4111-8111-111111111111",
					Summary: strings.Repeat("x", MaxEventSemanticBundleBytes),
				}},
			}}}},
			wantComponent: "event_semantic_bundle",
		},
		{
			name: "page reference closure",
			store: &contextStoreStub{dictionaries: Dictionaries{
				Entities: []Entity{{
					EntityID:   "33333333-3333-4333-8333-333333333333",
					EntityType: "company",
					Name:       strings.Repeat("x", MaxDictionaryBytes),
				}},
				EntityTypeDefinitions: []EntityTypeContext{{TypeKey: "company"}},
			}},
			wantComponent: "reference_closure",
		},
		{
			name:          "encoded Analysis Context page",
			store:         pageBudgetStore(),
			wantComponent: "analysis_context_page",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewService(test.store).List(context.Background(), request)
			var resourceLimit *ResourceLimitError
			if !errors.As(err, &resourceLimit) ||
				resourceLimit.Component != test.wantComponent ||
				resourceLimit.ActualBytes == nil ||
				resourceLimit.MaxBytes == nil {
				t.Fatalf("error = %#v", err)
			}
		})
	}
}

func pageBudgetStore() *contextStoreStub {
	bundles := make([]BundleRecord, 0, 10)
	for index := 1; index <= 10; index++ {
		eventID := fmt.Sprintf("40000000-0000-4000-8000-%012d", index)
		bundles = append(bundles, BundleRecord{
			KnowledgeAvailableAt: time.Date(2026, 7, 28, index, 0, 0, 0, time.UTC),
			EventID:              eventID,
			Bundle: EventSemanticBundle{Event: Event{
				ID: eventID, Summary: strings.Repeat("x", 450*1024),
			}},
		})
	}
	return &contextStoreStub{
		page: StorePage{Bundles: bundles},
		dictionaries: Dictionaries{
			Entities: []Entity{{
				EntityID:   "33333333-3333-4333-8333-333333333333",
				EntityType: "company",
				Name:       strings.Repeat("x", 3900*1024),
			}},
			EntityTypeDefinitions: []EntityTypeContext{{TypeKey: "company"}},
		},
	}
}

func stringPointer(value string) *string {
	return &value
}

func testHashPattern(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') &&
			(character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}
