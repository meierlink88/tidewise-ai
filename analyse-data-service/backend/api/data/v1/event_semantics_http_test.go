package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

type eventSemanticsHTTPStub struct {
	testDataHTTPServer
	contextLeaseRequest *EventSemanticContextLeaseRequest
	eligibleRequest     *EligibleEventSemanticEventsRequest
}

func (s *eventSemanticsHTTPStub) ListEligibleEventSemanticEvents(
	_ context.Context,
	request *EligibleEventSemanticEventsRequest,
) (*Response[EligibleEventSemanticEvents], error) {
	s.eligibleRequest = request
	return &Response[EligibleEventSemanticEvents]{
		Status: StatusOK,
		Result: EligibleEventSemanticEvents{
			Events: []EligibleEventSemanticEvent{{
				EventID: "22222222-2222-4222-8222-222222222222",
			}},
			NextCursor: "opaque-next-page",
		},
	}, nil
}

func (s *eventSemanticsHTTPStub) CreateEventSemanticContextLease(_ context.Context, request *EventSemanticContextLeaseRequest) (*Response[EventSemanticContextLease], error) {
	s.contextLeaseRequest = request
	return &Response[EventSemanticContextLease]{Status: StatusCreated, Result: EventSemanticContextLease{
		ContextLeaseID: "11111111-1111-4111-8111-111111111111",
		EventID:        "22222222-2222-4222-8222-222222222222",
		Status:         "active", LeaseExpiresAt: "2026-07-29T10:05:00Z",
	}}, nil
}

func TestEventSemanticContextLeaseUsesStrictTypedContract(t *testing.T) {
	stub := &eventSemanticsHTTPStub{}
	server := kratoshttp.NewServer()
	RegisterDataHTTPServer(server, stub)
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/event-semantics/context-leases",
		bytes.NewBufferString(`{"event_id":"22222222-2222-4222-8222-222222222222","agent_execution_id":"semantic-execution-1","worker_id":"semantic-worker","lease_seconds":300}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if stub.contextLeaseRequest == nil ||
		stub.contextLeaseRequest.EventID != "22222222-2222-4222-8222-222222222222" ||
		stub.contextLeaseRequest.WorkerID != "semantic-worker" ||
		stub.contextLeaseRequest.LeaseSeconds != 300 {
		t.Fatalf("context lease request = %#v", stub.contextLeaseRequest)
	}
	var response EventSemanticContextLease
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Status != "active" {
		t.Fatalf("response = %#v", response)
	}
}

func TestEventSemanticContextLeaseRejectsUnknownWireFieldsBeforeCallingService(t *testing.T) {
	stub := &eventSemanticsHTTPStub{}
	server := kratoshttp.NewServer(
		kratoshttp.ErrorEncoder(func(response http.ResponseWriter, _ *http.Request, err error) {
			public, ok := err.(*PublicError)
			if !ok {
				t.Fatalf("error = %T %v", err, err)
			}
			response.WriteHeader(public.Status)
		}),
	)
	RegisterDataHTTPServer(server, stub)
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/event-semantics/context-leases",
		bytes.NewBufferString(`{"event_id":"22222222-2222-4222-8222-222222222222","agent_execution_id":"semantic-execution-1","worker_id":"semantic-worker","lease_seconds":300,"invented":true}`))
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest || stub.contextLeaseRequest != nil {
		t.Fatalf("status = %d called = %v", recorder.Code, stub.contextLeaseRequest != nil)
	}
}

func TestEligibleEventSemanticsBindsOpaqueCursorAndReturnsNextCursor(t *testing.T) {
	stub := &eventSemanticsHTTPStub{}
	server := kratoshttp.NewServer()
	RegisterDataHTTPServer(server, stub)
	request := httptest.NewRequest(
		http.MethodGet,
		APIPrefix+"/event-semantics/eligible-events?limit=20&cursor=opaque-current-page",
		nil,
	)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if stub.eligibleRequest == nil ||
		stub.eligibleRequest.Limit != 20 ||
		stub.eligibleRequest.Cursor != "opaque-current-page" {
		t.Fatalf("eligible request = %#v", stub.eligibleRequest)
	}
	var response EligibleEventSemanticEvents
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.NextCursor != "opaque-next-page" || len(response.Events) != 1 {
		t.Fatalf("response = %#v", response)
	}
}
