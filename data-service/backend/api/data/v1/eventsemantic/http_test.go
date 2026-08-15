package eventsemantic_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	countryapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/country"
	organizationapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/organization"
	eventapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/event"
	eventsemanticapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/eventsemantic"
	evidenceapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/evidence"
	rawdocumentapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/rawdocument"
	runtimehealthapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/runtimehealth"
	eventbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/eventsemantic"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	eventsemanticdata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/eventsemantic"
	serverpkg "github.com/meierlink88/tidewise-ai/data-service/backend/internal/server"
	eventsemanticservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/eventsemantic"
	eventsemanticfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/eventsemantic"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
	researchfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/research"
)

type eventSemanticHTTPOrganizationStub struct{}

func (eventSemanticHTTPOrganizationStub) Create(context.Context, *organizationapi.CreateRequest) (*v1.Response[organizationapi.Organization], error) {
	return &v1.Response[organizationapi.Organization]{Status: http.StatusNoContent}, nil
}
func (eventSemanticHTTPOrganizationStub) List(context.Context, *organizationapi.ListRequest) (*v1.Response[organizationapi.OrganizationList], error) {
	return &v1.Response[organizationapi.OrganizationList]{Status: http.StatusNoContent}, nil
}
func (eventSemanticHTTPOrganizationStub) Get(context.Context, *organizationapi.GetRequest) (*v1.Response[organizationapi.Organization], error) {
	return &v1.Response[organizationapi.Organization]{Status: http.StatusNoContent}, nil
}
func (eventSemanticHTTPOrganizationStub) Update(context.Context, *organizationapi.UpdateRequest) (*v1.Response[organizationapi.Organization], error) {
	return &v1.Response[organizationapi.Organization]{Status: http.StatusNoContent}, nil
}
func (eventSemanticHTTPOrganizationStub) ReplaceDomainTags(context.Context, *organizationapi.ReplaceDomainTagsRequest) (*v1.Response[organizationapi.Organization], error) {
	return &v1.Response[organizationapi.Organization]{Status: http.StatusNoContent}, nil
}
func (eventSemanticHTTPOrganizationStub) GetCatalog(context.Context, *organizationapi.CatalogRequest) (*v1.Response[organizationapi.Catalog], error) {
	return &v1.Response[organizationapi.Catalog]{Status: http.StatusNoContent}, nil
}
func (eventSemanticHTTPOrganizationStub) ListMembers(context.Context, *organizationapi.ListMembersRequest) (*v1.Response[organizationapi.MemberList], error) {
	return &v1.Response[organizationapi.MemberList]{Status: http.StatusNoContent}, nil
}
func (eventSemanticHTTPOrganizationStub) CreateMember(context.Context, *organizationapi.CreateMemberRequest) (*v1.Response[organizationapi.Member], error) {
	return &v1.Response[organizationapi.Member]{Status: http.StatusNoContent}, nil
}
func (eventSemanticHTTPOrganizationStub) UpdateMember(context.Context, *organizationapi.UpdateMemberRequest) (*v1.Response[organizationapi.Member], error) {
	return &v1.Response[organizationapi.Member]{Status: http.StatusNoContent}, nil
}
func (eventSemanticHTTPOrganizationStub) DeleteMember(context.Context, *organizationapi.DeleteMemberRequest) (*v1.Response[organizationapi.DeleteResult], error) {
	return &v1.Response[organizationapi.DeleteResult]{Status: http.StatusNoContent}, nil
}

type eventSemanticHTTPStubBase struct{}

func (eventSemanticHTTPStubBase) ListEligibleEventSemanticEvents(context.Context, *eventsemanticapi.EligibleEventSemanticEventsRequest) (*v1.Response[eventsemanticapi.EligibleEventSemanticEvents], error) {
	return &v1.Response[eventsemanticapi.EligibleEventSemanticEvents]{Status: v1.StatusOK}, nil
}

func (eventSemanticHTTPStubBase) CreateEventSemanticContextLease(context.Context, *eventsemanticapi.EventSemanticContextLeaseRequest) (*v1.Response[eventsemanticapi.EventSemanticContextLease], error) {
	return &v1.Response[eventsemanticapi.EventSemanticContextLease]{Status: v1.StatusCreated}, nil
}

func (eventSemanticHTTPStubBase) GetEventSemanticContext(context.Context, *eventsemanticapi.EventSemanticContextRequest) (*v1.Response[eventsemanticapi.EventSemanticContext], error) {
	return &v1.Response[eventsemanticapi.EventSemanticContext]{Status: v1.StatusOK}, nil
}

func (eventSemanticHTTPStubBase) CreateEventSemanticSubmission(context.Context, *eventsemanticapi.EventSemanticSubmissionRequest) (*v1.Response[eventsemanticapi.EventSemanticSubmissionResult], error) {
	return &v1.Response[eventsemanticapi.EventSemanticSubmissionResult]{Status: v1.StatusCreated}, nil
}

func (eventSemanticHTTPStubBase) SubmitEventSemanticReview(context.Context, *eventsemanticapi.EventSemanticReviewRequest) (*v1.Response[eventsemanticapi.EventSemanticSubmissionResult], error) {
	return &v1.Response[eventsemanticapi.EventSemanticSubmissionResult]{Status: v1.StatusOK}, nil
}

func (eventSemanticHTTPStubBase) GetEventSemantics(context.Context, *eventsemanticapi.GetEventSemanticsRequest) (*v1.Response[eventsemanticapi.EventSemanticsResult], error) {
	return &v1.Response[eventsemanticapi.EventSemanticsResult]{Status: v1.StatusOK}, nil
}

type eventSemanticsHTTPStub struct {
	eventSemanticHTTPStubBase
	contextLeaseRequest *eventsemanticapi.EventSemanticContextLeaseRequest
	eligibleRequest     *eventsemanticapi.EligibleEventSemanticEventsRequest
	contextResponse     eventsemanticapi.EventSemanticContext
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
		{name: "eligible", method: http.MethodGet, path: "/event-semantics/eligible-events?limit=20", operation: eventsemanticapi.OperationListEligibleEvents, wantStatus: http.StatusOK},
		{name: "lease", method: http.MethodPost, path: "/event-semantics/context-leases", body: `{"event_id":"22222222-2222-4222-8222-222222222222","agent_execution_id":"semantic-execution-1","worker_id":"semantic-worker","lease_seconds":300}`, operation: eventsemanticapi.OperationCreateContextLease, wantStatus: http.StatusCreated},
		{name: "context", method: http.MethodGet, path: "/event-semantics/context-leases/11111111-1111-4111-8111-111111111111/context", operation: eventsemanticapi.OperationGetContext, wantStatus: http.StatusOK},
		{name: "submission", method: http.MethodPost, path: "/event-semantics/submissions", body: `{}`, operation: eventsemanticapi.OperationCreateSubmission, wantStatus: http.StatusCreated},
		{name: "review", method: http.MethodPost, path: "/event-semantics/submissions/11111111-1111-4111-8111-111111111111/reviews", body: `{}`, operation: eventsemanticapi.OperationSubmitReview, wantStatus: http.StatusOK},
		{name: "result", method: http.MethodGet, path: "/events/22222222-2222-4222-8222-222222222222/semantics", operation: eventsemanticapi.OperationGetSemantics, wantStatus: http.StatusOK},
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
			eventsemanticapi.RegisterHTTPServer(server, eventSemanticHTTPStubBase{})
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
	_ *eventsemanticapi.EventSemanticContextRequest,
) (*v1.Response[eventsemanticapi.EventSemanticContext], error) {
	return &v1.Response[eventsemanticapi.EventSemanticContext]{Status: v1.StatusOK, Result: s.contextResponse}, nil
}

func (s *eventSemanticsHTTPStub) ListEligibleEventSemanticEvents(
	_ context.Context,
	request *eventsemanticapi.EligibleEventSemanticEventsRequest,
) (*v1.Response[eventsemanticapi.EligibleEventSemanticEvents], error) {
	s.eligibleRequest = request
	return &v1.Response[eventsemanticapi.EligibleEventSemanticEvents]{
		Status: v1.StatusOK,
		Result: eventsemanticapi.EligibleEventSemanticEvents{
			Events: []eventsemanticapi.EligibleEventSemanticEvent{{
				EventID: "22222222-2222-4222-8222-222222222222",
			}},
			NextCursor: "opaque-next-page",
		},
	}, nil
}

func (s *eventSemanticsHTTPStub) CreateEventSemanticContextLease(_ context.Context, request *eventsemanticapi.EventSemanticContextLeaseRequest) (*v1.Response[eventsemanticapi.EventSemanticContextLease], error) {
	s.contextLeaseRequest = request
	return &v1.Response[eventsemanticapi.EventSemanticContextLease]{Status: v1.StatusCreated, Result: eventsemanticapi.EventSemanticContextLease{
		ContextLeaseID: "11111111-1111-4111-8111-111111111111",
		EventID:        "22222222-2222-4222-8222-222222222222",
		Status:         "active", LeaseExpiresAt: "2026-07-29T10:05:00Z",
	}}, nil
}

func TestEventSemanticContextLeaseUsesStrictTypedContract(t *testing.T) {
	stub := &eventSemanticsHTTPStub{}
	server := kratoshttp.NewServer()
	eventsemanticapi.RegisterHTTPServer(server, stub)
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
	var response eventsemanticapi.EventSemanticContextLease
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
	eventsemanticapi.RegisterHTTPServer(server, stub)
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
	eventsemanticapi.RegisterHTTPServer(server, stub)
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
	var response eventsemanticapi.EligibleEventSemanticEvents
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
	eventsemanticapi.RegisterHTTPServer(server, stub)
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
	stub := &eventSemanticsHTTPStub{contextResponse: eventsemanticapi.EventSemanticContext{
		ContextLeaseID:          "11111111-1111-4111-8111-111111111111",
		OntologyVersion:         "event-ontology.v1",
		AcceptancePolicyVersion: "event-semantic-acceptance.v1",
	}}
	server := kratoshttp.NewServer()
	eventsemanticapi.RegisterHTTPServer(server, stub)
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
	eventsemanticapi.RegisterHTTPServer(server, stub)
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

func openEventPublicationTestDatabase(t *testing.T) *sql.DB {
	return openEventPublicationTestDatabaseAt(t, 0)
}

func openEventPublicationTestDatabaseAt(t *testing.T, version int64) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_event_semantic", migrationDir, version)
}

func TestPostgresEventSemanticHTTPFlowPreservesLeaseSubmissionReviewAndRead(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	eventsemanticfixture.SeedScenario(t, db, true)
	store, err := eventsemanticdata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := eventbiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	application, err := eventsemanticservice.NewService(useCase)
	if err != nil {
		t.Fatal(err)
	}
	handler := newEventSemanticHTTPHandler(t, application)

	leaseRequest := eventsemanticapi.EventSemanticContextLeaseRequest{
		EventID: eventsemanticfixture.EventID, AgentExecutionID: "semantic-http-flow",
		WorkerID: "semantic-http-fixture", LeaseSeconds: 900,
	}
	var leaseEnvelope struct {
		Result eventsemanticapi.EventSemanticContextLease `json:"result"`
	}
	semanticAPIRequest(t, handler, http.MethodPost, "/event-semantics/context-leases", leaseRequest, http.StatusCreated, &leaseEnvelope)
	if leaseEnvelope.Result.ContextLeaseID == "" || leaseEnvelope.Result.EventID != eventsemanticfixture.EventID {
		t.Fatalf("lease = %#v", leaseEnvelope.Result)
	}

	var contextEnvelope struct {
		Result eventsemanticapi.EventSemanticContext `json:"result"`
	}
	semanticAPIRequest(t, handler, http.MethodGet,
		"/event-semantics/context-leases/"+leaseEnvelope.Result.ContextLeaseID+"/context",
		nil, http.StatusOK, &contextEnvelope)
	if contextEnvelope.Result.Event.ID != eventsemanticfixture.EventID || len(contextEnvelope.Result.Evidence) != 1 {
		t.Fatalf("context = %#v", contextEnvelope.Result)
	}

	submissionInput := eventsemanticfixture.Submission(leaseEnvelope.Result.ContextLeaseID, "semantic-http-flow", "")
	submissionRequest := eventSemanticSubmissionRequest(submissionInput)
	var submissionEnvelope struct {
		Result eventsemanticapi.EventSemanticSubmissionResult `json:"result"`
	}
	semanticAPIRequest(t, handler, http.MethodPost, "/event-semantics/submissions", submissionRequest, http.StatusCreated, &submissionEnvelope)
	if submissionEnvelope.Result.Status != string(eventbiz.StatusPendingReview) {
		t.Fatalf("submission = %#v", submissionEnvelope.Result)
	}

	reviewItems := eventsemanticfixture.ReviewItems("pass")
	reviewRequest := eventsemanticapi.EventSemanticReviewRequest{
		ReviewerExecutionKey: "semantic-http-flow:reviewer",
		PromptHash:           strings.Repeat("b", 64), Model: "fixture-reviewer",
		Items: []eventsemanticapi.EventSemanticReviewItem{
			{CandidateType: string(reviewItems[0].CandidateType), CandidateKey: reviewItems[0].CandidateKey, Decision: string(reviewItems[0].Decision), ReasonCodes: reviewItems[0].ReasonCodes, EvidenceIDs: reviewItems[0].EvidenceIDs},
			{CandidateType: string(reviewItems[1].CandidateType), CandidateKey: reviewItems[1].CandidateKey, Decision: string(reviewItems[1].Decision), ReasonCodes: reviewItems[1].ReasonCodes, EvidenceIDs: reviewItems[1].EvidenceIDs},
		},
	}
	var reviewEnvelope struct {
		Result eventsemanticapi.EventSemanticSubmissionResult `json:"result"`
	}
	semanticAPIRequest(t, handler, http.MethodPost,
		"/event-semantics/submissions/"+submissionEnvelope.Result.SubmissionID+"/reviews",
		reviewRequest, http.StatusOK, &reviewEnvelope)
	if reviewEnvelope.Result.Status != string(eventbiz.StatusAccepted) {
		t.Fatalf("review = %#v", reviewEnvelope.Result)
	}

	var resultEnvelope struct {
		Result eventsemanticapi.EventSemanticsResult `json:"result"`
	}
	semanticAPIRequest(t, handler, http.MethodGet, "/events/"+eventsemanticfixture.EventID+"/semantics", nil, http.StatusOK, &resultEnvelope)
	if len(resultEnvelope.Result.Submissions) != 1 || resultEnvelope.Result.Submissions[0].Status != string(eventbiz.StatusAccepted) {
		t.Fatalf("semantic result = %#v", resultEnvelope.Result)
	}
}

func eventSemanticSubmissionRequest(input eventbiz.Submission) eventsemanticapi.EventSemanticSubmissionRequest {
	request := eventsemanticapi.EventSemanticSubmissionRequest{
		ContextLeaseID: input.ContextLeaseID, EventID: input.EventID, AgentExecutionID: input.AgentExecutionID,
		AgentKey: input.AgentKey, AgentVersion: input.AgentVersion, SupersedesSubmissionID: input.SupersedesSubmissionID,
		GeneratorPromptHash: input.GeneratorPromptHash, GeneratorModel: input.GeneratorModel,
		ReviewerPromptHash: input.ReviewerPromptHash, ReviewerModel: input.ReviewerModel,
		AdjudicatorPromptHash: input.AdjudicatorPromptHash, AdjudicatorModel: input.AdjudicatorModel,
		OntologyVersion: input.OntologyVersion, AcceptancePolicyVersion: input.AcceptancePolicyVersion,
	}
	for _, link := range input.EntityLinks {
		request.EntityLinks = append(request.EntityLinks, eventsemanticapi.EventSemanticV3EntityLinkCandidate{
			CandidateKey: link.Key, Mention: link.Mention, EntityID: link.EntityID,
			ProjectedEntityType: link.ProjectedEntityType, EntityRole: link.EntityRole,
			EvidenceIDs: link.EvidenceIDs, ResolutionMethod: link.ResolutionMethod,
			ResolutionConfidence: link.ResolutionConfidence,
		})
	}
	for _, signal := range input.VariableSignals {
		item := eventsemanticapi.EventSemanticVariableSignalCandidate{
			CandidateKey: signal.Key, SubjectLinkKey: signal.SubjectLinkKey,
			VariableKey: signal.VariableKey, VariableVersion: signal.VariableVersion,
			Direction: signal.Direction, AssertionModality: signal.AssertionModality,
			EvidenceIDs: signal.EvidenceIDs, ExtractionConfidence: signal.ExtractionConfidence,
		}
		for _, measurement := range signal.Measurements {
			item.Measurements = append(item.Measurements, eventsemanticapi.EventSemanticMeasurement{
				MeasurementText: measurement.Text, EvidenceIDs: measurement.EvidenceIDs,
			})
		}
		request.VariableSignals = append(request.VariableSignals, item)
	}
	return request
}

func semanticAPIRequest(t *testing.T, handler http.Handler, method, path string, body any, wantStatus int, target any) {
	t.Helper()
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, v1.APIPrefix+path, bytes.NewReader(payload))
	request.Header.Set("Authorization", "Bearer semantic-token")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != wantStatus {
		t.Fatalf("%s %s status = %d, want %d; body=%s", method, path, response.Code, wantStatus, response.Body)
	}
	if target != nil {
		if err := json.Unmarshal(response.Body.Bytes(), target); err != nil {
			t.Fatalf("decode %s %s: %v; body=%s", method, path, err, response.Body)
		}
	}
}

func newEventSemanticHTTPHandler(t *testing.T, application *eventsemanticservice.Service) http.Handler {
	t.Helper()
	authenticator, err := serverpkg.NewAuthenticator([]serverpkg.Credential{{
		Secret: "semantic-token",
		Principal: v1.Principal{Identity: "semantic-publisher", Scopes: []string{
			serverpkg.ScopeEventSemanticsRead, serverpkg.ScopeEventSemanticsWrite,
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	httpServer, err := serverpkg.NewHTTPServer(
		conf.Config{App: conf.AppConfig{Env: conf.EnvLocal}, Server: conf.ServerConfig{Host: "127.0.0.1", Port: 18082, ReadTimeoutSeconds: 5, WriteTimeoutSeconds: 10}},
		semanticTestRuntimeHealthService{}, researchfixture.Service{}, semanticTestEventService{}, application,
		semanticTestEvidenceService{}, semanticTestRawDocumentService{}, semanticTestCountryService{}, eventSemanticHTTPOrganizationStub{}, authenticator, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return httpServer.Server.Handler
}

type semanticTestEventService struct{}

type semanticTestCountryService struct{}

func (semanticTestCountryService) Create(context.Context, *countryapi.CreateRequest) (*v1.Response[countryapi.Country], error) {
	return &v1.Response[countryapi.Country]{Status: http.StatusNoContent}, nil
}

func (semanticTestCountryService) List(context.Context, *countryapi.ListRequest) (*v1.Response[countryapi.CountryList], error) {
	return &v1.Response[countryapi.CountryList]{Status: http.StatusNoContent}, nil
}

func (semanticTestCountryService) Get(context.Context, *countryapi.GetRequest) (*v1.Response[countryapi.Country], error) {
	return &v1.Response[countryapi.Country]{Status: http.StatusNoContent}, nil
}

func (semanticTestCountryService) Update(context.Context, *countryapi.UpdateRequest) (*v1.Response[countryapi.Country], error) {
	return &v1.Response[countryapi.Country]{Status: http.StatusNoContent}, nil
}

func (semanticTestCountryService) ReplaceRegions(context.Context, *countryapi.ReplaceRegionsRequest) (*v1.Response[countryapi.Country], error) {
	return &v1.Response[countryapi.Country]{Status: http.StatusNoContent}, nil
}

type semanticTestRuntimeHealthService struct{}

func (semanticTestRuntimeHealthService) GetRuntimeHealth(context.Context, *runtimehealthapi.Request) (*v1.Response[runtimehealthapi.Result], error) {
	return &v1.Response[runtimehealthapi.Result]{Status: http.StatusNoContent}, nil
}

type semanticTestRawDocumentService struct{}

func (semanticTestRawDocumentService) List(context.Context, *rawdocumentapi.ListRequest) (*v1.Response[rawdocumentapi.Page], error) {
	return &v1.Response[rawdocumentapi.Page]{Status: http.StatusNoContent}, nil
}

func (semanticTestEventService) PublishReviewedEvents(context.Context, *eventapi.PublicationRequest) (*v1.Response[eventapi.PublicationResult], error) {
	return &v1.Response[eventapi.PublicationResult]{Status: http.StatusNoContent}, nil
}

func (semanticTestEventService) ListActiveEventTags(context.Context, *eventapi.TagCatalogRequest) (*v1.Response[eventapi.TagCatalog], error) {
	return &v1.Response[eventapi.TagCatalog]{Status: http.StatusNoContent}, nil
}

func (semanticTestEventService) ListEvents(context.Context, *eventapi.ListRequest) (*v1.Response[eventapi.Page], error) {
	return &v1.Response[eventapi.Page]{Status: http.StatusNoContent}, nil
}

type semanticTestEvidenceService struct{}

func (semanticTestEvidenceService) PublishRawEvidence(context.Context, *evidenceapi.RawEvidencePublicationRequest) (*v1.Response[evidenceapi.RawEvidencePublicationResult], error) {
	return &v1.Response[evidenceapi.RawEvidencePublicationResult]{Status: http.StatusNoContent}, nil
}

func (semanticTestEvidenceService) GetRawEvidence(context.Context, *evidenceapi.GetRawEvidenceRequest) (*v1.Response[evidenceapi.RawEvidenceReadResult], error) {
	return &v1.Response[evidenceapi.RawEvidenceReadResult]{Status: http.StatusNoContent}, nil
}

func (semanticTestEvidenceService) PublishEvidence(context.Context, *evidenceapi.EvidencePublicationRequest) (*v1.Response[evidenceapi.EvidencePublicationResult], error) {
	return &v1.Response[evidenceapi.EvidencePublicationResult]{Status: http.StatusNoContent}, nil
}
