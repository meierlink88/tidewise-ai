package eventsemantics

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"
)

type paginationStoreStub struct {
	after   *EligibleEventCursor
	items   []EligibleEvent
	anchors []ResolutionAnchor
}

func (s *paginationStoreStub) ListEligibleEvents(
	_ context.Context,
	_ int,
	after *EligibleEventCursor,
) ([]EligibleEvent, error) {
	s.after = after
	return append([]EligibleEvent(nil), s.items...), nil
}

func (*paginationStoreStub) CreateContextLease(context.Context, ContextLeaseRequest) (ContextLease, error) {
	return ContextLease{}, nil
}
func (*paginationStoreStub) Context(context.Context, string) (Context, error) {
	return Context{}, nil
}
func (*paginationStoreStub) SubmissionContext(context.Context, string, Submission) (Context, error) {
	return Context{}, nil
}
func (*paginationStoreStub) Resolve(context.Context, string, []EntityMention) ([]EntityResolution, error) {
	return nil, nil
}
func (*paginationStoreStub) SearchDirectTargets(context.Context, string, string, []string) ([]DirectTarget, error) {
	return nil, nil
}
func (*paginationStoreStub) ListResolutionRoutes(context.Context, string, string) ([]ResolutionRoute, error) {
	return nil, nil
}
func (s *paginationStoreStub) ListResolutionAnchors(context.Context, string, string, string, []string) ([]ResolutionAnchor, error) {
	return append([]ResolutionAnchor(nil), s.anchors...), nil
}
func (*paginationStoreStub) ResolveChainNodeCandidates(context.Context, string, string, []string) ([]ResolutionCandidate, error) {
	return nil, nil
}
func (*paginationStoreStub) ReplaySubmission(context.Context, string, string) (SubmissionResult, bool, error) {
	return SubmissionResult{}, false, nil
}
func (*paginationStoreStub) CreateSubmission(context.Context, Submission, PrecheckResult, []byte, string) (SubmissionResult, error) {
	return SubmissionResult{}, nil
}
func (*paginationStoreStub) SubmitReview(context.Context, ReviewSubmission, []byte, string) (SubmissionResult, error) {
	return SubmissionResult{}, nil
}
func (*paginationStoreStub) GetEventSemantics(context.Context, string) (EventSemanticsResult, error) {
	return EventSemanticsResult{}, nil
}

func TestEligibleEventPaginationUsesStableOpaqueCursor(t *testing.T) {
	firstSeenAt := time.Date(2026, 7, 29, 8, 0, 0, 0, time.UTC)
	store := &paginationStoreStub{items: []EligibleEvent{
		{EventID: "11111111-1111-4111-8111-111111111111", FirstSeenAt: firstSeenAt},
		{EventID: "22222222-2222-4222-8222-222222222222", FirstSeenAt: firstSeenAt.Add(time.Minute)},
	}}
	service := NewService(store)

	first, err := service.ListEligibleEvents(context.Background(), 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Events) != 1 || first.NextCursor == "" {
		t.Fatalf("first page = %#v", first)
	}

	store.items = nil
	second, err := service.ListEligibleEvents(context.Background(), 1, first.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Events) != 0 || second.NextCursor != "" || store.after == nil {
		t.Fatalf("second page = %#v after = %#v", second, store.after)
	}
	if store.after.EventID != "11111111-1111-4111-8111-111111111111" ||
		!store.after.FirstSeenAt.Equal(firstSeenAt) {
		t.Fatalf("decoded cursor = %#v", store.after)
	}
}

func TestEligibleEventPaginationRejectsMalformedCursorBeforeStoreAccess(t *testing.T) {
	store := &paginationStoreStub{}
	service := NewService(store)

	_, err := service.ListEligibleEvents(context.Background(), 20, "not-a-cursor")

	if err == nil || store.after != nil {
		t.Fatalf("err = %v after = %#v", err, store.after)
	}
}

func TestEligibleEventPaginationRejectsUnsupportedCursorVersion(t *testing.T) {
	store := &paginationStoreStub{}
	service := NewService(store)
	stale := base64.RawURLEncoding.EncodeToString([]byte(
		`{"v":0,"first_seen_at":"2026-07-29T08:00:00Z","event_id":"11111111-1111-4111-8111-111111111111"}`,
	))

	_, err := service.ListEligibleEvents(context.Background(), 20, stale)

	if err == nil || store.after != nil {
		t.Fatalf("err = %v after = %#v", err, store.after)
	}
}

func TestResolutionAnchorPaginationDetectsSourceDrift(t *testing.T) {
	store := &paginationStoreStub{anchors: []ResolutionAnchor{
		{Entity: Entity{ID: "11111111-1111-4111-8111-111111111111"}, Partition: "11111111-1111-4111-8111-111111111111"},
		{Entity: Entity{ID: "22222222-2222-4222-8222-222222222222"}, Partition: "11111111-1111-4111-8111-111111111111"},
	}}
	service := NewService(store)
	first, err := service.ListResolutionAnchors(
		context.Background(), "lease", "chain-node-via-industry.v1", "11111111-1111-4111-8111-111111111111", nil, 1, "",
	)
	if err != nil || len(first.Anchors) != 1 || first.NextCursor == "" {
		t.Fatalf("first page=%#v err=%v", first, err)
	}
	store.anchors[1].Entity.ID = "33333333-3333-4333-8333-333333333333"
	_, err = service.ListResolutionAnchors(
		context.Background(), "lease", "chain-node-via-industry.v1", "11111111-1111-4111-8111-111111111111", nil, 1, first.NextCursor,
	)
	var drift *ContextDriftError
	if !errors.As(err, &drift) {
		t.Fatalf("drift error = %T %v", err, err)
	}
}

var _ Store = (*paginationStoreStub)(nil)
