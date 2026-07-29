package researchanalysiscontext

import (
	"context"
	"errors"
	"testing"
	"time"
)

const testDictionaryFingerprint = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

type contextStoreStub struct {
	query StoreQuery
	page  StorePage
}

func (s *contextStoreStub) List(
	_ context.Context,
	query StoreQuery,
) (StorePage, error) {
	s.query = query
	return s.page, nil
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
		HasMore:               true,
		Dictionaries:          Dictionaries{},
		DictionaryFingerprint: testDictionaryFingerprint,
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

	store.page = StorePage{
		Dictionaries:          Dictionaries{},
		DictionaryFingerprint: testDictionaryFingerprint,
	}
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
	}}, HasMore: true, Dictionaries: Dictionaries{},
		DictionaryFingerprint: testDictionaryFingerprint}}
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

func TestServiceRejectsCursorAfterDictionaryVersionChanges(t *testing.T) {
	store := &contextStoreStub{page: StorePage{
		Bundles: []BundleRecord{{
			KnowledgeAvailableAt: time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC),
			EventID:              "11111111-1111-4111-8111-111111111111",
			Bundle: EventSemanticBundle{Event: Event{
				ID: "11111111-1111-4111-8111-111111111111",
			}},
		}},
		Dictionaries:          Dictionaries{},
		DictionaryFingerprint: testDictionaryFingerprint,
		HasMore:               true,
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
	store.page = StorePage{
		Dictionaries:          Dictionaries{},
		DictionaryFingerprint: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	if _, err := service.List(context.Background(), request); err == nil {
		t.Fatal("cursor remained valid after the dictionary fingerprint changed")
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
		Version:               1,
		Fingerprint:           fingerprint,
		DictionaryFingerprint: testDictionaryFingerprint,
		KnowledgeAvailableAt:  "2026-07-28T09:00:00Z",
		EventID:               "not-a-uuid",
	})
	if err != nil {
		t.Fatal(err)
	}

	store := &contextStoreStub{page: StorePage{
		Dictionaries:          Dictionaries{},
		DictionaryFingerprint: testDictionaryFingerprint,
	}}
	if _, err := NewService(store).List(context.Background(), request); err == nil {
		t.Fatal("cursor with an invalid terminal Event ID was accepted")
	}
}

func TestServiceRejectsHistoricalBundleWithUnresolvableEntityReference(t *testing.T) {
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
		Dictionaries:          Dictionaries{},
		DictionaryFingerprint: testDictionaryFingerprint,
	}}
	request := Request{
		DiscoveryWindowStart: "2026-07-28T00:00:00Z",
		DiscoveryWindowEnd:   "2026-07-29T00:00:00Z",
		AnalysisAsOf:         "2026-07-29T00:00:00Z",
		PageSize:             20,
	}

	if _, err := NewService(store).List(context.Background(), request); !errors.Is(
		err, ErrHistoricalSemanticsUnavailable,
	) {
		t.Fatalf("error = %v, want historical semantics unavailable", err)
	}
}

func stringPointer(value string) *string {
	return &value
}
