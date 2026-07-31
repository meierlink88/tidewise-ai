package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventsemantics"
)

type eventSemanticsPaginationServiceStub struct{}

type eventSemanticsDriftServiceStub struct {
	eventSemanticsPaginationServiceStub
}

func (eventSemanticsDriftServiceStub) Context(context.Context, string) (eventsemantics.Context, error) {
	return eventsemantics.Context{}, &eventsemantics.ContextDriftError{Reason: "manifest drift"}
}

func (eventSemanticsDriftServiceStub) Resolve(context.Context, string, []eventsemantics.EntityMention) ([]eventsemantics.EntityResolution, error) {
	return nil, &eventsemantics.ContextDriftError{Reason: "manifest drift"}
}

func (eventSemanticsDriftServiceStub) SearchDirectTargets(context.Context, string, string, []string) ([]eventsemantics.DirectTarget, error) {
	return nil, &eventsemantics.ContextDriftError{Reason: "manifest drift"}
}

func (eventSemanticsDriftServiceStub) ListResolutionRoutes(context.Context, string, string) ([]eventsemantics.ResolutionRoute, error) {
	return nil, &eventsemantics.ContextDriftError{Reason: "manifest drift"}
}

func (eventSemanticsPaginationServiceStub) ListEligibleEvents(
	context.Context,
	int,
	string,
) (eventsemantics.EligibleEventPage, error) {
	return eventsemantics.EligibleEventPage{
		Events: []eventsemantics.EligibleEvent{{
			EventID: "11111111-1111-4111-8111-111111111111",
		}},
		NextCursor: "opaque-next-page",
	}, nil
}

func (eventSemanticsPaginationServiceStub) CreateContextLease(
	context.Context,
	eventsemantics.ContextLeaseRequest,
) (eventsemantics.ContextLease, error) {
	return eventsemantics.ContextLease{}, nil
}

func (eventSemanticsPaginationServiceStub) Context(
	context.Context,
	string,
) (eventsemantics.Context, error) {
	return eventsemantics.Context{}, nil
}

func (eventSemanticsPaginationServiceStub) Resolve(
	context.Context,
	string,
	[]eventsemantics.EntityMention,
) ([]eventsemantics.EntityResolution, error) {
	return nil, nil
}

func (eventSemanticsPaginationServiceStub) SearchDirectTargets(
	context.Context,
	string,
	string,
	[]string,
) ([]eventsemantics.DirectTarget, error) {
	return nil, nil
}

func (eventSemanticsPaginationServiceStub) ListResolutionRoutes(
	context.Context, string, string,
) ([]eventsemantics.ResolutionRoute, error) {
	return nil, nil
}

func (eventSemanticsPaginationServiceStub) ListResolutionAnchors(
	context.Context, string, string, string, []string, int, string,
) (eventsemantics.ResolutionAnchorPage, error) {
	return eventsemantics.ResolutionAnchorPage{}, nil
}

func (eventSemanticsPaginationServiceStub) ResolveChainNodeCandidates(
	context.Context, string, string, []string, int, string,
) (eventsemantics.ResolutionCandidatePage, error) {
	return eventsemantics.ResolutionCandidatePage{}, nil
}

func (eventSemanticsPaginationServiceStub) CreateSubmission(
	context.Context,
	eventsemantics.Submission,
) (eventsemantics.SubmissionResult, error) {
	return eventsemantics.SubmissionResult{}, nil
}

func (eventSemanticsPaginationServiceStub) SubmitReview(
	context.Context,
	eventsemantics.ReviewSubmission,
) (eventsemantics.SubmissionResult, error) {
	return eventsemantics.SubmissionResult{}, nil
}

func (eventSemanticsPaginationServiceStub) Get(
	context.Context,
	string,
) (eventsemantics.EventSemanticsResult, error) {
	return eventsemantics.EventSemanticsResult{}, nil
}

func TestEligiblePaginationIsOptInForPreviousAgentRunCompatibility(t *testing.T) {
	service := NewDataService(Dependencies{
		EventSemantics: eventSemanticsPaginationServiceStub{},
	})
	legacy, err := service.ListEligibleEventSemanticEvents(
		context.Background(),
		&v1.EligibleEventSemanticEventsRequest{Limit: 20},
	)
	if err != nil {
		t.Fatal(err)
	}
	if legacy.Result.NextCursor != "" {
		t.Fatalf("legacy response exposed next_cursor: %#v", legacy.Result)
	}
	payload, err := json.Marshal(legacy.Result)
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var previousAgentRun struct {
		Events []struct {
			EventID string `json:"event_id"`
		} `json:"events"`
	}
	if err := decoder.Decode(&previousAgentRun); err != nil {
		t.Fatalf("previous AgentRun rejected legacy response: %v", err)
	}

	current, err := service.ListEligibleEventSemanticEvents(
		context.Background(),
		&v1.EligibleEventSemanticEventsRequest{
			Limit: 20, Pagination: "cursor",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if current.Result.NextCursor != "opaque-next-page" {
		t.Fatalf("cursor-capable response = %#v", current.Result)
	}
}

func TestResolutionEmptyPagesEncodeCollectionsAsArrays(t *testing.T) {
	service := NewDataService(Dependencies{EventSemantics: eventSemanticsPaginationServiceStub{}})
	anchors, err := service.ListEventSemanticResolutionAnchors(context.Background(), &v1.EventSemanticResolutionAnchorRequest{
		ContextLeaseID: "lease", RouteID: "chain-node-via-industry.v1",
		Partition: "11111111-1111-4111-8111-111111111111", PageSize: 50,
	})
	if err != nil {
		t.Fatal(err)
	}
	candidates, err := service.ResolveEventSemanticChainNodeCandidates(context.Background(), &v1.EventSemanticResolutionCandidateRequest{
		ContextLeaseID: "lease", RouteID: "chain-node-via-industry.v1",
		TargetEntityType: "chain_node", MatchMode: "any",
		AnchorEntityIDs: []string{"11111111-1111-4111-8111-111111111111"}, PageSize: 50,
	})
	if err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]any{"anchors": anchors.Result, "candidates": candidates.Result} {
		payload, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(payload, []byte("null")) || !bytes.Contains(payload, []byte("[]")) {
			t.Fatalf("%s empty page = %s, want JSON array", name, payload)
		}
	}
}

func TestManifestReadingOperationsMapContextDriftToStableConflict(t *testing.T) {
	service := NewDataService(Dependencies{EventSemantics: eventSemanticsDriftServiceStub{}})
	tests := map[string]func() error{
		"context": func() error {
			_, err := service.GetEventSemanticContext(context.Background(), &v1.EventSemanticContextRequest{ContextLeaseID: "lease"})
			return err
		},
		"entity resolution": func() error {
			_, err := service.ResolveEventSemanticEntities(context.Background(), &v1.EventSemanticEntityResolutionRequest{
				ContextLeaseID: "lease", Mentions: []v1.EventSemanticEntityMention{{Mention: "mention", AllowedEntityTypes: []string{"company"}}},
			})
			return err
		},
		"direct targets": func() error {
			_, err := service.SearchEventSemanticDirectTargets(context.Background(), &v1.EventSemanticDirectTargetSearchRequest{
				ContextLeaseID: "lease", SubjectEntityID: "11111111-1111-4111-8111-111111111111", AllowedTargetTypes: []string{"company"},
			})
			return err
		},
		"resolution routes": func() error {
			_, err := service.ListEventSemanticResolutionRoutes(context.Background(), &v1.EventSemanticResolutionRouteRequest{
				ContextLeaseID: "lease", TargetEntityType: "chain_node",
			})
			return err
		},
	}
	for name, operation := range tests {
		t.Run(name, func(t *testing.T) {
			var public *v1.PublicError
			if err := operation(); !errors.As(err, &public) || public.Status != v1.StatusConflict || public.Code != "EVENT_SEMANTIC_CONTEXT_DRIFT" {
				t.Fatalf("error = %T %#v", err, err)
			}
		})
	}
}

var _ EventSemanticsService = eventSemanticsPaginationServiceStub{}
var _ EventSemanticsService = eventSemanticsDriftServiceStub{}
