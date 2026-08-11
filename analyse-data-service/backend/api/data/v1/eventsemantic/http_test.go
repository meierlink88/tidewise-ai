package eventsemantic

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
)

type eventSemanticHTTPStubBase struct{}

func (eventSemanticHTTPStubBase) ListEligibleEventSemanticEvents(context.Context, *EligibleEventSemanticEventsRequest) (*v1.Response[EligibleEventSemanticEvents], error) {
	return &v1.Response[EligibleEventSemanticEvents]{Status: v1.StatusOK}, nil
}
func (eventSemanticHTTPStubBase) CreateEventSemanticContextLease(context.Context, *EventSemanticContextLeaseRequest) (*v1.Response[EventSemanticContextLease], error) {
	return &v1.Response[EventSemanticContextLease]{Status: v1.StatusCreated}, nil
}
func (eventSemanticHTTPStubBase) GetEventSemanticContext(context.Context, *EventSemanticContextRequest) (*v1.Response[EventSemanticContext], error) {
	return &v1.Response[EventSemanticContext]{Status: v1.StatusOK}, nil
}
func (eventSemanticHTTPStubBase) CreateEventSemanticSubmission(context.Context, *EventSemanticSubmissionRequest) (*v1.Response[EventSemanticSubmissionResult], error) {
	return &v1.Response[EventSemanticSubmissionResult]{Status: v1.StatusCreated}, nil
}
func (eventSemanticHTTPStubBase) SubmitEventSemanticReview(context.Context, *EventSemanticReviewRequest) (*v1.Response[EventSemanticSubmissionResult], error) {
	return &v1.Response[EventSemanticSubmissionResult]{Status: v1.StatusOK}, nil
}
func (eventSemanticHTTPStubBase) GetEventSemantics(context.Context, *GetEventSemanticsRequest) (*v1.Response[EventSemanticsResult], error) {
	return &v1.Response[EventSemanticsResult]{Status: v1.StatusOK}, nil
}

type eventSemanticsHTTPStub struct {
	eventSemanticHTTPStubBase
	contextLeaseRequest *EventSemanticContextLeaseRequest
	eligibleRequest     *EligibleEventSemanticEventsRequest
	contextResponse     EventSemanticContext
}

func TestEventSemanticRuntimeRoutesPreserveOperations(t *testing.T) {
	for _, test := range []struct {
		name       string
		method     string
		path       string
		body       string
		operation  string
		wantStatus int
	}{
		{name: "eligible", method: http.MethodGet, path: "/event-semantics/eligible-events?limit=20", operation: OperationListEligibleEvents, wantStatus: http.StatusOK},
		{name: "lease", method: http.MethodPost, path: "/event-semantics/context-leases", body: `{"event_id":"22222222-2222-4222-8222-222222222222","agent_execution_id":"semantic-execution-1","worker_id":"semantic-worker","lease_seconds":300}`, operation: OperationCreateContextLease, wantStatus: http.StatusCreated},
		{name: "context", method: http.MethodGet, path: "/event-semantics/context-leases/11111111-1111-4111-8111-111111111111/context", operation: OperationGetContext, wantStatus: http.StatusOK},
		{name: "submission", method: http.MethodPost, path: "/event-semantics/submissions", body: `{}`, operation: OperationCreateSubmission, wantStatus: http.StatusCreated},
		{name: "review", method: http.MethodPost, path: "/event-semantics/submissions/11111111-1111-4111-8111-111111111111/reviews", body: `{}`, operation: OperationSubmitReview, wantStatus: http.StatusOK},
		{name: "result", method: http.MethodGet, path: "/events/22222222-2222-4222-8222-222222222222/semantics", operation: OperationGetSemantics, wantStatus: http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			var operation string
			recorder := func(next middleware.Handler) middleware.Handler {
				return func(ctx context.Context, request any) (any, error) {
					if serverTransport, ok := transport.FromServerContext(ctx); ok {
						operation = serverTransport.Operation()
					}
					return next(ctx, request)
				}
			}
			server := kratoshttp.NewServer(kratoshttp.Middleware(recorder))
			RegisterHTTPServer(server, eventSemanticHTTPStubBase{})
			response := httptest.NewRecorder()
			request := httptest.NewRequest(test.method, v1.APIPrefix+test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			server.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.wantStatus, response.Body)
			}
			if operation != test.operation {
				t.Fatalf("operation = %q, want %q", operation, test.operation)
			}
		})
	}
}

func (s *eventSemanticsHTTPStub) GetEventSemanticContext(
	_ context.Context,
	_ *EventSemanticContextRequest,
) (*v1.Response[EventSemanticContext], error) {
	return &v1.Response[EventSemanticContext]{Status: v1.StatusOK, Result: s.contextResponse}, nil
}

func (s *eventSemanticsHTTPStub) ListEligibleEventSemanticEvents(
	_ context.Context,
	request *EligibleEventSemanticEventsRequest,
) (*v1.Response[EligibleEventSemanticEvents], error) {
	s.eligibleRequest = request
	return &v1.Response[EligibleEventSemanticEvents]{
		Status: v1.StatusOK,
		Result: EligibleEventSemanticEvents{
			Events: []EligibleEventSemanticEvent{{
				EventID: "22222222-2222-4222-8222-222222222222",
			}},
			NextCursor: "opaque-next-page",
		},
	}, nil
}

func (s *eventSemanticsHTTPStub) CreateEventSemanticContextLease(_ context.Context, request *EventSemanticContextLeaseRequest) (*v1.Response[EventSemanticContextLease], error) {
	s.contextLeaseRequest = request
	return &v1.Response[EventSemanticContextLease]{Status: v1.StatusCreated, Result: EventSemanticContextLease{
		ContextLeaseID: "11111111-1111-4111-8111-111111111111",
		EventID:        "22222222-2222-4222-8222-222222222222",
		Status:         "active", LeaseExpiresAt: "2026-07-29T10:05:00Z",
	}}, nil
}

func TestEventSemanticContextLeaseUsesStrictTypedContract(t *testing.T) {
	stub := &eventSemanticsHTTPStub{}
	server := kratoshttp.NewServer()
	RegisterHTTPServer(server, stub)
	request := httptest.NewRequest(http.MethodPost, v1.APIPrefix+"/event-semantics/context-leases",
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
			public, ok := err.(*v1.PublicError)
			if !ok {
				t.Fatalf("error = %T %v", err, err)
			}
			response.WriteHeader(public.Status)
		}),
	)
	RegisterHTTPServer(server, stub)
	request := httptest.NewRequest(http.MethodPost, v1.APIPrefix+"/event-semantics/context-leases",
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
	RegisterHTTPServer(server, stub)
	request := httptest.NewRequest(
		http.MethodGet,
		v1.APIPrefix+"/event-semantics/eligible-events?limit=20&cursor=opaque-current-page&pagination=cursor",
		nil,
	)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if stub.eligibleRequest == nil ||
		stub.eligibleRequest.Limit != 20 ||
		stub.eligibleRequest.Cursor != "opaque-current-page" ||
		stub.eligibleRequest.Pagination != "cursor" {
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

func TestEligibleEventSemanticsRejectsUnknownPaginationCapability(t *testing.T) {
	stub := &eventSemanticsHTTPStub{}
	server := kratoshttp.NewServer(
		kratoshttp.ErrorEncoder(func(response http.ResponseWriter, _ *http.Request, err error) {
			public, ok := err.(*v1.PublicError)
			if !ok {
				t.Fatalf("error = %T %v", err, err)
			}
			response.WriteHeader(public.Status)
		}),
	)
	RegisterHTTPServer(server, stub)
	request := httptest.NewRequest(
		http.MethodGet,
		v1.APIPrefix+"/event-semantics/eligible-events?pagination=offset",
		nil,
	)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest || stub.eligibleRequest != nil {
		t.Fatalf(
			"status = %d called = %v body = %s",
			recorder.Code, stub.eligibleRequest != nil, recorder.Body.String(),
		)
	}
}

func TestEventSemanticContextIsCompactAndOmitsABoxCatalog(t *testing.T) {
	stub := &eventSemanticsHTTPStub{contextResponse: EventSemanticContext{
		ContextLeaseID:          "11111111-1111-4111-8111-111111111111",
		OntologyVersion:         "event-ontology.v1",
		AcceptancePolicyVersion: "event-semantic-acceptance.v1",
	}}
	server := kratoshttp.NewServer()
	RegisterHTTPServer(server, stub)
	request := httptest.NewRequest(
		http.MethodGet,
		v1.APIPrefix+"/event-semantics/context-leases/11111111-1111-4111-8111-111111111111/context",
		nil,
	)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.Len() >= 100_000 {
		t.Fatalf("compact context response = %d bytes, want < 100000", recorder.Body.Len())
	}
	var response map[string]json.RawMessage
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"entities", "relations"} {
		if _, ok := response[forbidden]; ok {
			t.Fatalf("compact context contains forbidden %q: %s", forbidden, recorder.Body.String())
		}
	}
}

func TestEventSemanticV2DoesNotRegisterLegacyResolutionRoutes(t *testing.T) {
	stub := &eventSemanticsHTTPStub{}
	server := kratoshttp.NewServer()
	RegisterHTTPServer(server, stub)
	request := httptest.NewRequest(
		http.MethodPost,
		v1.APIPrefix+"/event-semantics/resolution-routes:list",
		bytes.NewBufferString(`{"context_lease_id":"11111111-1111-4111-8111-111111111111","target_entity_type":"chain_node"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d", recorder.Code)
	}
}
