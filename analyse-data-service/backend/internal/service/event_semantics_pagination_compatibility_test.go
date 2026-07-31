package service

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventsemantics"
)

type eventSemanticsPaginationServiceStub struct{}

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

var _ EventSemanticsService = eventSemanticsPaginationServiceStub{}
