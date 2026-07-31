package dataclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
)

func TestEventSemanticClientConsumesFrozenProviderFixtures(t *testing.T) {
	eligibleFixture := readSemanticFixture(t, "supply-eligible-events.json")
	contextLeaseFixture := readSemanticFixture(t, "supply-context-lease.json")
	contextFixture := readSemanticFixture(t, "supply-context.json")
	resolutionFixture := readSemanticFixture(t, "supply-resolution.json")
	targetFixture := readSemanticFixture(t, "supply-targets.json")
	runFixture := readSemanticFixture(t, "supply-submission-accepted.json")
	reviewFixture := readSemanticFixture(t, "supply-review-accepted.json")
	readFixture := readSemanticFixture(t, "supply-event-semantics.json")
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer semantic-token" {
			t.Fatalf("Authorization = %q", request.Header.Get("Authorization"))
		}
		response.Header().Set("Content-Type", "application/json")
		response.Header().Set("X-Request-ID", request.Header.Get("X-Request-ID"))
		switch {
		case request.Method == http.MethodGet &&
			request.URL.Path == "/api/data/v1/event-semantics/eligible-events":
			writeSemanticFixture(t, response, request, eligibleFixture)
		case request.Method == http.MethodPost &&
			request.URL.Path == "/api/data/v1/event-semantics/context-leases":
			response.WriteHeader(http.StatusCreated)
			writeSemanticFixture(t, response, request, contextLeaseFixture)
		case request.Method == http.MethodGet &&
			request.URL.Path == "/api/data/v1/event-semantics/context-leases/11111111-1111-4111-8111-111111111111/context":
			writeSemanticFixture(t, response, request, contextFixture)
		case request.Method == http.MethodPost &&
			request.URL.Path == "/api/data/v1/event-semantics/entity-resolutions":
			writeSemanticFixture(t, response, request, resolutionFixture)
		case request.Method == http.MethodPost &&
			request.URL.Path == "/api/data/v1/event-semantics/direct-targets:search":
			writeSemanticFixture(t, response, request, targetFixture)
		case request.Method == http.MethodPost && request.URL.Path == "/api/data/v1/event-semantics/submissions":
			body, _ := io.ReadAll(request.Body)
			if len(body) == 0 {
				t.Fatal("run request body is empty")
			}
			response.WriteHeader(http.StatusCreated)
			writeSemanticFixture(t, response, request, runFixture)
		case request.Method == http.MethodPost &&
			request.URL.Path == "/api/data/v1/event-semantics/submissions/66666666-6666-4666-8666-666666666666/reviews":
			writeSemanticFixture(t, response, request, reviewFixture)
		case request.Method == http.MethodGet &&
			request.URL.Path == "/api/data/v1/events/88888888-8888-4888-8888-888888888888/semantics":
			writeSemanticFixture(t, response, request, readFixture)
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()
	client, err := New(Config{
		BaseURL: server.URL, ServiceToken: "semantic-token",
		Timeout: time.Second, MaxResponseBytes: 1024 * 1024,
	})
	if err != nil {
		t.Fatal(err)
	}

	eligible, err := client.ListEligibleEvents(context.Background(), 20, "")
	if err != nil || len(eligible.Events) != 1 ||
		eligible.Events[0].EventID != "88888888-8888-4888-8888-888888888888" {
		t.Fatalf("eligible=%#v err=%v", eligible, err)
	}
	contextLease, err := client.CreateContextLease(context.Background(), eventsemantic.ContextLeaseRequest{
		EventID:          "88888888-8888-4888-8888-888888888888",
		AgentExecutionID: "semantic-execution-1", WorkerID: "contract-worker", LeaseSeconds: 300,
	})
	if err != nil || contextLease.ContextLeaseID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("context lease=%#v err=%v", contextLease, err)
	}
	contextSnapshot, err := client.Context(context.Background(), "11111111-1111-4111-8111-111111111111")
	if err != nil {
		t.Fatal(err)
	}
	if contextSnapshot.Event.ID != "88888888-8888-4888-8888-888888888888" ||
		len(contextSnapshot.VariableDefinitions) != 2 ||
		contextSnapshot.DirectTransmissionRules[0].RuleKey != "production_decrease_reduces_product_supply" {
		t.Fatalf("context = %#v", contextSnapshot)
	}
	resolutions, err := client.Resolve(context.Background(), contextLease.ContextLeaseID, []eventsemantic.EntityMention{{
		Mention: "某晶圆厂", AllowedEntityTypes: []string{"company"},
	}})
	if err != nil || len(resolutions) != 1 ||
		resolutions[0].Candidates[0].EntityID != "33333333-3333-4333-8333-333333333333" {
		t.Fatalf("resolutions = %#v err=%v", resolutions, err)
	}
	targets, err := client.SearchDirectTargets(
		context.Background(), contextLease.ContextLeaseID,
		"33333333-3333-4333-8333-333333333333", []string{"product"},
	)
	if err != nil || len(targets) != 1 ||
		targets[0].Entity.EntityID != "44444444-4444-4444-8444-444444444444" {
		t.Fatalf("targets = %#v err=%v", targets, err)
	}
	run, err := client.CreateSubmission(context.Background(), eventsemantic.SubmissionRequest{
		ContextLeaseID: "11111111-1111-4111-8111-111111111111",
		EventID:        "88888888-8888-4888-8888-888888888888",
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "accepted" || run.DirectImpacts[0].CandidateKey != "supply" {
		t.Fatalf("run = %#v", run)
	}
	reviewed, err := client.SubmitReview(
		context.Background(), run.SubmissionID, eventsemantic.ReviewRequest{
			ReviewerExecutionKey: "execution-supply-001:reviewer",
			PromptHash:           "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			Model:                "fixture-reviewer",
			Items: []eventsemantic.ReviewItem{{
				CandidateType: "variable_signal", CandidateKey: "production",
				Decision: "pass", EvidenceIDs: []string{"22222222-2222-4222-8222-222222222222"},
			}},
		},
	)
	if err != nil || reviewed.Status != "accepted" {
		t.Fatalf("reviewed = %#v err=%v", reviewed, err)
	}
	semantics, err := client.GetEventSemantics(context.Background(), "88888888-8888-4888-8888-888888888888")
	if err != nil {
		t.Fatal(err)
	}
	if semantics.EventID != contextSnapshot.Event.ID || len(semantics.Submissions) != 1 ||
		semantics.Submissions[0].SubmissionID != run.SubmissionID {
		t.Fatalf("semantics = %#v", semantics)
	}
	readRun := semantics.Submissions[0]
	if readRun.AuditWorkPackage == nil ||
		readRun.AuditWorkPackage.Evidence[0].RawDocumentID != "99999999-9999-4999-8999-999999999999" ||
		readRun.VariableSignals[0].RecordID == "" ||
		len(readRun.ReviewSnapshots) != 1 ||
		readRun.GeneratorPromptHash == "" {
		t.Fatalf("complete semantics audit = %#v", readRun)
	}
}

func TestEventSemanticEligibleEventsCanBeEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestID := request.Header.Get("X-Request-ID")
		response.Header().Set("X-Request-ID", requestID)
		_, _ = response.Write([]byte(`{"request_id":"` + requestID + `","result":{"events":[]}}`))
	}))
	defer server.Close()
	client, err := New(Config{
		BaseURL: server.URL, ServiceToken: "token",
		Timeout: time.Second, MaxResponseBytes: 4096,
	})
	if err != nil {
		t.Fatal(err)
	}
	page, listErr := client.ListEligibleEvents(context.Background(), 20, "")
	if listErr != nil || len(page.Events) != 0 || page.NextCursor != "" {
		t.Fatalf("page=%#v err=%v", page, listErr)
	}
}

func TestEventSemanticEligibleEventsCarriesOpaqueCursorAcrossTheContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("limit") != "20" ||
			request.URL.Query().Get("cursor") != "opaque-current-page" {
			t.Fatalf("query = %q", request.URL.RawQuery)
		}
		requestID := request.Header.Get("X-Request-ID")
		response.Header().Set("X-Request-ID", requestID)
		_, _ = response.Write([]byte(`{"request_id":"` + requestID + `","result":{"events":[{"event_id":"88888888-8888-4888-8888-888888888888"}],"next_cursor":"opaque-next-page"}}`))
	}))
	defer server.Close()
	client, err := New(Config{
		BaseURL: server.URL, ServiceToken: "token",
		Timeout: time.Second, MaxResponseBytes: 4096,
	})
	if err != nil {
		t.Fatal(err)
	}

	page, listErr := client.ListEligibleEvents(
		context.Background(), 20, "opaque-current-page",
	)

	if listErr != nil || len(page.Events) != 1 ||
		page.Events[0].EventID != "88888888-8888-4888-8888-888888888888" ||
		page.NextCursor != "opaque-next-page" {
		t.Fatalf("page=%#v err=%v", page, listErr)
	}
}

func TestEventSemanticClientReplaysIdempotentRunWriteAfterUnknownResult(t *testing.T) {
	calls := 0
	fixture := readSemanticFixture(t, "supply-submission-accepted.json")
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		calls++
		if request.URL.Path != "/api/data/v1/event-semantics/submissions" {
			http.NotFound(response, request)
			return
		}
		if calls == 1 {
			requestID := request.Header.Get("X-Request-ID")
			response.Header().Set("X-Request-ID", requestID)
			response.WriteHeader(http.StatusBadGateway)
			_, _ = response.Write([]byte(`{"request_id":"` + requestID + `","error":{"code":"UNKNOWN_RESULT","message":"unknown result","details":{}}}`))
			return
		}
		response.Header().Set("X-Request-ID", request.Header.Get("X-Request-ID"))
		response.WriteHeader(http.StatusCreated)
		writeSemanticFixture(t, response, request, fixture)
	}))
	defer server.Close()
	client, err := New(Config{
		BaseURL: server.URL, ServiceToken: "token",
		Timeout: time.Second, MaxResponseBytes: 4096,
	})
	if err != nil {
		t.Fatal(err)
	}

	result, callErr := client.CreateSubmission(context.Background(), eventsemantic.SubmissionRequest{
		ContextLeaseID: "11111111-1111-4111-8111-111111111111",
		EventID:        "88888888-8888-4888-8888-888888888888",
	})
	if callErr != nil {
		t.Fatal(callErr)
	}
	if calls != 2 || result.SubmissionID == "" {
		t.Fatalf("calls=%d result=%#v", calls, result)
	}
}

func TestEventSemanticClientBoundsReplaySafeWriteAttempts(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		wantCalls int
	}{
		{name: "conflict is not retried", status: http.StatusConflict, wantCalls: 1},
		{name: "service unavailable exhausts one replay", status: http.StatusServiceUnavailable, wantCalls: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				calls++
				if request.URL.Path != "/api/data/v1/event-semantics/submissions" {
					http.NotFound(response, request)
					return
				}
				requestID := request.Header.Get("X-Request-ID")
				response.Header().Set("X-Request-ID", requestID)
				response.WriteHeader(test.status)
				_, _ = response.Write([]byte(`{"request_id":"` + requestID + `","error":{"code":"semantic_write_failed","message":"write failed","details":{}}}`))
			}))
			defer server.Close()
			client, err := New(Config{
				BaseURL: server.URL, ServiceToken: "token",
				Timeout: time.Second, MaxResponseBytes: 4096,
			})
			if err != nil {
				t.Fatal(err)
			}
			_, callErr := client.CreateSubmission(context.Background(), eventsemantic.SubmissionRequest{
				ContextLeaseID: "11111111-1111-4111-8111-111111111111",
				EventID:        "88888888-8888-4888-8888-888888888888",
			})
			if callErr == nil {
				t.Fatal("expected write failure")
			}
			if calls != test.wantCalls {
				t.Fatalf("calls = %d, want %d", calls, test.wantCalls)
			}
		})
	}
}

func TestEventSemanticClientRejectsInvalidErrorEnvelopeRequestIDContract(t *testing.T) {
	tests := []struct {
		name         string
		headerID     func(string) string
		responseJSON func(string) string
	}{
		{
			name:     "missing response header",
			headerID: func(string) string { return "" },
			responseJSON: func(requestID string) string {
				return `{"request_id":"` + requestID + `","error":{"code":"FAILED","message":"failed","details":{}}}`
			},
		},
		{
			name:     "mismatched body request id",
			headerID: func(requestID string) string { return requestID },
			responseJSON: func(string) string {
				return `{"request_id":"different","error":{"code":"FAILED","message":"failed","details":{}}}`
			},
		},
		{
			name:     "malformed error envelope",
			headerID: func(requestID string) string { return requestID },
			responseJSON: func(requestID string) string {
				return `{"request_id":"` + requestID + `","error":{"code":"FAILED","message":"failed"}}`
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				requestID := request.Header.Get("X-Request-ID")
				if responseID := test.headerID(requestID); responseID != "" {
					response.Header().Set("X-Request-ID", responseID)
				}
				response.WriteHeader(http.StatusConflict)
				_, _ = response.Write([]byte(test.responseJSON(requestID)))
			}))
			defer server.Close()
			client, err := New(Config{
				BaseURL: server.URL, ServiceToken: "token",
				Timeout: time.Second, MaxResponseBytes: 4096,
			})
			if err != nil {
				t.Fatal(err)
			}
			_, callErr := client.CreateSubmission(context.Background(), eventsemantic.SubmissionRequest{
				ContextLeaseID: "11111111-1111-4111-8111-111111111111",
				EventID:        "88888888-8888-4888-8888-888888888888",
			})
			var remote *eventsemantic.RemoteError
			if !errors.As(callErr, &remote) || remote.Code != "data_response_invalid" {
				t.Fatalf("error = %#v, want data_response_invalid", callErr)
			}
		})
	}
}

func readSemanticFixture(t *testing.T, name string) []byte {
	t.Helper()
	payload, err := os.ReadFile(filepath.Join(
		"..", "..", "..", "..", "..", "contracts", "event-semantics", "v1", name,
	))
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func writeSemanticFixture(
	t *testing.T,
	response http.ResponseWriter,
	request *http.Request,
	fixture []byte,
) {
	t.Helper()
	requestID := request.Header.Get("X-Request-ID")
	if requestID == "" || request.Header.Get("X-Tidewise-Operation") == "" ||
		request.Header.Get("X-Tidewise-Path-Template") == "" {
		t.Fatalf("semantic request metadata is incomplete: %#v", request.Header)
	}
	response.Header().Set("X-Request-ID", requestID)
	var envelope map[string]any
	if err := json.Unmarshal(fixture, &envelope); err != nil {
		t.Fatal(err)
	}
	envelope["request_id"] = requestID
	payload, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = response.Write(payload)
}
