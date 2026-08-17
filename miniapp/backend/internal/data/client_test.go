package data

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz"
)

func TestHTTPClientListsResearchThemesWithIdentityAndRequestID(t *testing.T) {
	t.Parallel()
	var gotAuthorization, gotRequestID, gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotAuthorization = request.Header.Get("Authorization")
		gotRequestID = request.Header.Get(RequestIDHeader)
		gotQuery = request.URL.RawQuery
		if request.URL.Path != ResearchThemesPath {
			t.Fatalf("path = %q, want %q", request.URL.Path, ResearchThemesPath)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"request_id":"data-req-1","result":{"as_of":"2026-07-17T01:02:03Z","items":[{"id":"11111111-1111-5111-8111-111111111111","title":"theme","conclusion_direction":"positive","impact_strength":"medium","transmission_stage":"diffusion","investment_guidance_summary":"流动性改善后风险偏好可能回升","impacts":[{"chain_node_id":"ENT22222222-2222-5222-8222-222222222222","name":"算力基础设施","relation_role":"driver","impact_direction":"positive","impact_summary":"资本开支上升","display_order":1}]}],"next_cursor":null}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, server.Client(), "miniapp-service-token")
	page, err := client.ListResearchThemes(WithRequestID(context.Background(), "req-123"), biz.ResearchListQuery{WindowHours: 24, Limit: 20, Cursor: "cursor value"})
	if err != nil {
		t.Fatal(err)
	}
	if gotAuthorization != "Bearer miniapp-service-token" || gotRequestID != "req-123" {
		t.Fatalf("auth/request ID = %q/%q", gotAuthorization, gotRequestID)
	}
	for _, fragment := range []string{"window_hours=24", "limit=20", "cursor=cursor+value"} {
		if !strings.Contains(gotQuery, fragment) {
			t.Fatalf("query = %q, want %q", gotQuery, fragment)
		}
	}
	if len(page.Items) != 1 || page.Items[0].Title != "theme" || page.Items[0].ImpactStrength != "medium" || page.Items[0].TransmissionStage != "diffusion" || page.Items[0].InvestmentGuidanceSummary != "流动性改善后风险偏好可能回升" || !page.AsOf.Equal(time.Date(2026, 7, 17, 1, 2, 3, 0, time.UTC)) {
		t.Fatalf("page = %#v", page)
	}
	if len(page.Items[0].Impacts) != 1 || *page.Items[0].Impacts[0].ImpactSummary != "资本开支上升" {
		t.Fatalf("theme impacts = %#v", page.Items[0].Impacts)
	}
}

func TestHTTPClientEscapesResearchDetailID(t *testing.T) {
	t.Parallel()
	var gotPath, gotQuery, gotRequestID string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotPath, gotQuery, gotRequestID = request.URL.EscapedPath(), request.URL.RawQuery, request.Header.Get(RequestIDHeader)
		_, _ = writer.Write([]byte(`{"request_id":"data-req-2","result":{"theme":{"id":"theme/id","title":"detail"},"events":[]}}`))
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, server.Client(), "token")

	result, err := client.GetResearchTheme(context.Background(), "theme/id")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != ResearchThemesPath+"/theme%2Fid" || gotQuery != "" || gotRequestID == "" || result.Theme.Title != "detail" {
		t.Fatalf("path/query/request ID/result = %q/%q/%q/%#v", gotPath, gotQuery, gotRequestID, result)
	}
}

func TestResearchListPathForwardsExplicitPublicationRange(t *testing.T) {
	publishedFrom := time.Date(2026, 7, 4, 16, 0, 0, 0, time.UTC)
	publishedTo := time.Date(2026, 8, 3, 16, 0, 0, 0, time.UTC)
	path := researchListPath(ResearchThemesPath, biz.ResearchListQuery{
		PublishedFrom: &publishedFrom, PublishedTo: &publishedTo, Limit: 5, Cursor: "data-cursor",
	})
	parsed, err := url.Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	if query.Get("published_from") != "2026-07-04T16:00:00Z" || query.Get("published_to") != "2026-08-03T16:00:00Z" || query.Get("limit") != "5" || query.Get("cursor") != "data-cursor" || query.Has("window_hours") {
		t.Fatalf("query = %q", parsed.RawQuery)
	}
}

func TestHTTPClientReadsResearchReasoningTreeList(t *testing.T) {
	t.Parallel()
	var gotAuthorization, gotRequestID, gotPath, gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotAuthorization = request.Header.Get("Authorization")
		gotRequestID = request.Header.Get(RequestIDHeader)
		gotPath = request.URL.EscapedPath()
		gotQuery = request.URL.RawQuery
		writer.Header().Set("Content-Type", "application/json")
		if _, err := writer.Write([]byte(`{"request_id":"data-req-reasoning-list","result":{"theme":{"id":"c26337f2-a79f-5089-84f4-63d57bc32230"},"reasoning_trees":[{"industry_chain_name":"高速光模块产业链"},{}]}}`)); err != nil {
			t.Errorf("write reasoning tree list response: %v", err)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, server.Client(), "miniapp-service-token")
	result, err := client.ListResearchThemeReasoningTrees(WithRequestID(context.Background(), "req-reasoning-list"), "theme/id")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != ResearchThemesPath+"/theme%2Fid/reasoning-trees" || gotQuery != "" {
		t.Fatalf("path/query = %q/%q", gotPath, gotQuery)
	}
	if gotAuthorization != "Bearer miniapp-service-token" || gotRequestID != "req-reasoning-list" {
		t.Fatalf("auth/request ID = %q/%q", gotAuthorization, gotRequestID)
	}
	if result.Theme.ID != "c26337f2-a79f-5089-84f4-63d57bc32230" || len(result.ReasoningTrees) != 2 || result.ReasoningTrees[0].IndustryChainName != "高速光模块产业链" {
		t.Fatalf("result = %#v", result)
	}
}

func TestHTTPClientReadsResearchReasoningTreeDetail(t *testing.T) {
	t.Parallel()
	var gotPath, gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotPath = request.URL.EscapedPath()
		gotQuery = request.URL.RawQuery
		writer.Header().Set("Content-Type", "application/json")
		if _, err := writer.Write([]byte(`{"request_id":"data-req-reasoning-detail","result":{"theme_id":"c26337f2-a79f-5089-84f4-63d57bc32230","reasoning_tree":{"event_count":2,"events":[{}, {"evidence_role":"contradicting"}],"nodes":[{}, {}, {"primary_signal":{"variable_signal_key":"dsp-demand"}}]}}}`)); err != nil {
			t.Errorf("write reasoning tree detail response: %v", err)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, server.Client(), "miniapp-service-token")
	result, err := client.GetResearchThemeReasoningTree(context.Background(), "theme/id", "tree/id")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != ResearchThemesPath+"/theme%2Fid/reasoning-trees/tree%2Fid" || gotQuery != "" {
		t.Fatalf("path/query = %q/%q", gotPath, gotQuery)
	}
	if result.ThemeID != "c26337f2-a79f-5089-84f4-63d57bc32230" || result.ReasoningTree.EventCount != 2 || result.ReasoningTree.Events[1].EvidenceRole != "contradicting" || result.ReasoningTree.Nodes[2].PrimarySignal.VariableSignalKey != "dsp-demand" {
		t.Fatalf("result = %#v", result)
	}
}

func TestHTTPClientPreservesReasoningTreesNotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusNotFound)
		if _, err := writer.Write([]byte(`{"request_id":"data-req-reasoning-not-found","error":{"code":"RESEARCH_REASONING_TREES_NOT_FOUND","message":"research Theme has no published reasoning trees","details":{}}}`)); err != nil {
			t.Errorf("write reasoning tree not-found response: %v", err)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, server.Client(), "miniapp-service-token")
	_, err := client.ListResearchThemeReasoningTrees(context.Background(), "dddddddd-dddd-4ddd-8ddd-dddddddddddd")
	if !errors.Is(err, biz.ErrResearchReasoningTreesNotFound) {
		t.Fatalf("error = %#v, want stable Biz not-found error", err)
	}
}

func TestHTTPClientRejectsMalformedSuccessEnvelope(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"items":[]}`))
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, server.Client(), "token")

	var envelope responseEnvelope[wireResearchThemePage]
	err := client.doJSON(context.Background(), http.MethodGet, ResearchThemesPath, nil, &envelope)
	_, err = unwrapEnvelope(envelope, err)
	assertErrorKind(t, err, ErrorKindDecode)

	_, err = client.ListResearchThemes(context.Background(), biz.ResearchListQuery{})
	if !errors.Is(err, biz.ErrResearchDataService) {
		t.Fatalf("public repository error = %v, want stable Biz data-service error", err)
	}
}

func TestHTTPClientRetriesOnlySafeRetryableReads(t *testing.T) {
	t.Parallel()
	var readAttempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodGet {
			if readAttempts.Add(1) == 1 {
				writer.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			_, _ = writer.Write([]byte(`{"request_id":"data-req-3","result":{"items":[]}}`))
			return
		}
		readAttempts.Add(1)
		writer.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, server.Client(), "token")

	if _, err := client.ListResearchThemes(context.Background(), biz.ResearchListQuery{}); err != nil {
		t.Fatalf("safe read error = %v", err)
	}
	if got := readAttempts.Load(); got != 2 {
		t.Fatalf("safe read attempts = %d, want 2", got)
	}

	readAttempts.Store(0)
	err := client.doJSON(context.Background(), http.MethodPost, ResearchThemesPath, map[string]string{"value": "mutation"}, nil)
	if err == nil {
		t.Fatal("mutation error = nil")
	}
	if got := readAttempts.Load(); got != 1 {
		t.Fatalf("mutation attempts = %d, want 1", got)
	}
}

func TestHTTPClientClassifiesHTTPFailuresWithoutLeakingSecrets(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		status int
		kind   ErrorKind
	}{
		{name: "client", status: http.StatusBadRequest, kind: ErrorKindClient},
		{name: "conflict", status: http.StatusConflict, kind: ErrorKindConflict},
		{name: "server", status: http.StatusInternalServerError, kind: ErrorKindServer},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var attempts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				attempts.Add(1)
				writer.Header().Set(RequestIDHeader, "response-request-id")
				writer.WriteHeader(test.status)
				_, _ = writer.Write([]byte(`{"error":{"code":"UPSTREAM_CODE","message":"secret-response-body"}}`))
			}))
			defer server.Close()
			client := newTestClient(t, server.URL, server.Client(), "secret-service-token")

			err := client.doJSON(context.Background(), http.MethodGet, ResearchThemesPath+"/11111111-1111-5111-8111-111111111111", nil, nil)
			var clientErr *Error
			if !errors.As(err, &clientErr) || clientErr.Kind != test.kind || clientErr.StatusCode != test.status || clientErr.Code != "UPSTREAM_CODE" || clientErr.RequestID != "response-request-id" {
				t.Fatalf("error = %#v", err)
			}
			if strings.Contains(err.Error(), "secret-service-token") || strings.Contains(err.Error(), "secret-response-body") {
				t.Fatalf("unsafe error = %q", err)
			}
			wantAttempts := int32(1)
			if test.status >= 500 {
				wantAttempts = 2
			}
			if attempts.Load() != wantAttempts {
				t.Fatalf("attempts = %d, want %d", attempts.Load(), wantAttempts)
			}
		})
	}
}

func TestHTTPClientClassifiesConnectionFailureAndDeadline(t *testing.T) {
	t.Parallel()
	var connectionAttempts atomic.Int32
	connectionClient := newTestClient(t, "http://data.invalid", &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		connectionAttempts.Add(1)
		return nil, fmt.Errorf("dial failed with secret-service-token")
	})}, "secret-service-token")
	err := connectionClient.doJSON(context.Background(), http.MethodGet, ResearchThemesPath, nil, nil)
	assertErrorKind(t, err, ErrorKindConnection)
	if connectionAttempts.Load() != 2 || strings.Contains(err.Error(), "secret-service-token") {
		t.Fatalf("connection attempts/error = %d/%q", connectionAttempts.Load(), err)
	}

	timeoutClient, err := NewHTTPClient(HTTPConfig{
		BaseURL:         "http://data.invalid",
		ServiceToken:    "token",
		Timeout:         10 * time.Millisecond,
		MaxReadAttempts: 2,
		HTTPClient: &http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
			<-request.Context().Done()
			return nil, request.Context().Err()
		})},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = timeoutClient.doJSON(context.Background(), http.MethodGet, ResearchThemesPath, nil, nil)
	assertErrorKind(t, err, ErrorKindTimeout)

	transportTimeoutClient := newTestClient(t, "http://data.invalid", &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("transport secret: %w", context.DeadlineExceeded)
	})}, "token")
	err = transportTimeoutClient.doJSON(context.Background(), http.MethodGet, ResearchThemesPath, nil, nil)
	assertErrorKind(t, err, ErrorKindTimeout)
	if strings.Contains(err.Error(), "transport secret") {
		t.Fatalf("unsafe timeout error = %q", err)
	}
}

func newTestClient(t *testing.T, baseURL string, httpClient *http.Client, token string) *HTTPClient {
	t.Helper()
	client, err := NewHTTPClient(HTTPConfig{BaseURL: baseURL, ServiceToken: token, Timeout: time.Second, MaxReadAttempts: 2, HTTPClient: httpClient})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func assertErrorKind(t *testing.T, err error, want ErrorKind) {
	t.Helper()
	var clientErr *Error
	if !errors.As(err, &clientErr) || clientErr.Kind != want {
		t.Fatalf("error = %#v, want kind %q", err, want)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}
