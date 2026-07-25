package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	usecase "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz"
	dataclient "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/data"
	appservice "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/service"
)

func TestResearchReasoningTreeRoutesMapSharedFixturesWithOneDataCall(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		dataResult, expected := transportReasoningFixtureResult[usecase.ResearchReasoningTreeList](t, "01-reasoning-tree-list-result.json")
		calls := 0
		client := &usecase.Fake{ListResearchThemeReasoningTreesFunc: func(ctx context.Context, themeID string) (usecase.ResearchReasoningTreeList, error) {
			calls++
			if dataclient.RequestIDFromContext(ctx) != "miniapp-reasoning-1" || themeID != "11111111-1111-4111-8111-111111111111" {
				t.Fatalf("request ID/theme ID = %q/%q", dataclient.RequestIDFromContext(ctx), themeID)
			}
			return dataResult, nil
		}}
		request := httptest.NewRequest(http.MethodGet, "/api/miniapp/v1/research/themes/11111111-1111-4111-8111-111111111111/reasoning-trees", nil)
		request.Header.Set(dataclient.RequestIDHeader, "miniapp-reasoning-1")
		response := httptest.NewRecorder()
		researchTestRouter(usecase.NewResearchService(client)).ServeHTTP(response, request)

		if response.Code != http.StatusOK || calls != 1 {
			t.Fatalf("status/calls = %d/%d, body=%s", response.Code, calls, response.Body.String())
		}
		assertTransportJSONEquivalent(t, expected, response.Body.Bytes())
	})

	t.Run("detail", func(t *testing.T) {
		dataResult, expected := transportReasoningFixtureResult[usecase.ResearchReasoningTreeDetail](t, "02-reasoning-tree-with-contradiction-result.json")
		calls := 0
		client := &usecase.Fake{GetResearchThemeReasoningTreeFunc: func(_ context.Context, themeID, anchorID string) (usecase.ResearchReasoningTreeDetail, error) {
			calls++
			if themeID != "11111111-1111-4111-8111-111111111111" || anchorID != "534d83be-774b-51d9-ad00-cdee4ba91799" {
				t.Fatalf("theme/anchor IDs = %q/%q", themeID, anchorID)
			}
			return dataResult, nil
		}}
		response := serveResearch(t, usecase.NewResearchService(client), "/api/miniapp/v1/research/themes/11111111-1111-4111-8111-111111111111/reasoning-trees/534d83be-774b-51d9-ad00-cdee4ba91799")

		if response.Code != http.StatusOK || calls != 1 {
			t.Fatalf("status/calls = %d/%d, body=%s", response.Code, calls, response.Body.String())
		}
		assertTransportJSONEquivalent(t, expected, response.Body.Bytes())
	})
}

func TestResearchReasoningTreeRoutesRejectQueryAndInvalidUUIDBeforeDataCall(t *testing.T) {
	calls := 0
	client := &usecase.Fake{
		ListResearchThemeReasoningTreesFunc: func(context.Context, string) (usecase.ResearchReasoningTreeList, error) {
			calls++
			return usecase.ResearchReasoningTreeList{}, nil
		},
		GetResearchThemeReasoningTreeFunc: func(context.Context, string, string) (usecase.ResearchReasoningTreeDetail, error) {
			calls++
			return usecase.ResearchReasoningTreeDetail{}, nil
		},
	}
	service := usecase.NewResearchService(client)
	for _, path := range []string{
		"/api/miniapp/v1/research/themes/11111111-1111-4111-8111-111111111111/reasoning-trees?window_hours=24",
		"/api/miniapp/v1/research/themes/11111111-1111-4111-8111-111111111111/reasoning-trees/534d83be-774b-51d9-ad00-cdee4ba91799?unused=1",
		"/api/miniapp/v1/research/themes/11111111-1111-4111-8111-11111111111A/reasoning-trees",
		"/api/miniapp/v1/research/themes/11111111-1111-4111-8111-111111111111/reasoning-trees/NOT-A-UUID",
	} {
		response := serveResearch(t, service, path)
		assertReasoningError(t, response, http.StatusBadRequest, "INVALID_REQUEST")
	}
	if calls != 0 {
		t.Fatalf("Data calls = %d, want 0", calls)
	}
}

func TestResearchReasoningTreeInvalidQueryRunsKratosMiddleware(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	client := &usecase.Fake{
		ListResearchThemeReasoningTreesFunc: func(context.Context, string) (usecase.ResearchReasoningTreeList, error) {
			t.Fatal("Data must not be called for an invalid query")
			return usecase.ResearchReasoningTreeList{}, nil
		},
	}
	router := NewHTTPServer(
		testRuntimeConfig(),
		appservice.NewResearchService(usecase.NewResearchService(client)),
		logger,
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodGet,
			"/api/miniapp/v1/research/themes/11111111-1111-4111-8111-111111111111/reasoning-trees?unused=1",
			nil,
		),
	)

	assertReasoningError(t, response, http.StatusBadRequest, "INVALID_REQUEST")
	if !strings.Contains(logs.String(), `"msg":"business request failed"`) ||
		!strings.Contains(logs.String(), `"operation":"/miniapp.v1.ResearchService/ListResearchThemeReasoningTrees"`) ||
		!strings.Contains(logs.String(), `"status":400`) {
		t.Fatalf("invalid query bypassed Kratos middleware: %s", logs.String())
	}
}

func TestResearchReasoningTreeRoutesExposeStableErrorsWithoutUpstreamMetadata(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		upstream   error
		wantStatus int
		wantCode   string
	}{
		{
			name: "Theme missing", path: "/api/miniapp/v1/research/themes/11111111-1111-4111-8111-111111111111/reasoning-trees",
			upstream: usecase.ErrResearchThemeNotFound, wantStatus: http.StatusNotFound, wantCode: "RESEARCH_THEME_NOT_FOUND",
		},
		{
			name: "trees missing", path: "/api/miniapp/v1/research/themes/11111111-1111-4111-8111-111111111111/reasoning-trees",
			upstream: usecase.ErrResearchReasoningTreesNotFound, wantStatus: http.StatusNotFound, wantCode: "RESEARCH_REASONING_TREES_NOT_FOUND",
		},
		{
			name: "tree missing", path: "/api/miniapp/v1/research/themes/11111111-1111-4111-8111-111111111111/reasoning-trees/534d83be-774b-51d9-ad00-cdee4ba91799",
			upstream: usecase.ErrResearchReasoningTreeNotFound, wantStatus: http.StatusNotFound, wantCode: "RESEARCH_REASONING_TREE_NOT_FOUND",
		},
		{
			name: "unknown upstream 404", path: "/api/miniapp/v1/research/themes/11111111-1111-4111-8111-111111111111/reasoning-trees/534d83be-774b-51d9-ad00-cdee4ba91799",
			upstream: errors.New("unexpected upstream not found"), wantStatus: http.StatusBadGateway, wantCode: "RESEARCH_DATA_UNAVAILABLE",
		},
		{
			name: "network", path: "/api/miniapp/v1/research/themes/11111111-1111-4111-8111-111111111111/reasoning-trees/534d83be-774b-51d9-ad00-cdee4ba91799",
			upstream: errors.New("dial postgres password=must-not-leak"), wantStatus: http.StatusBadGateway, wantCode: "RESEARCH_DATA_UNAVAILABLE",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &usecase.Fake{
				ListResearchThemeReasoningTreesFunc: func(context.Context, string) (usecase.ResearchReasoningTreeList, error) {
					return usecase.ResearchReasoningTreeList{}, test.upstream
				},
				GetResearchThemeReasoningTreeFunc: func(context.Context, string, string) (usecase.ResearchReasoningTreeDetail, error) {
					return usecase.ResearchReasoningTreeDetail{}, test.upstream
				},
			}
			response := serveResearch(t, usecase.NewResearchService(client), test.path)
			assertReasoningError(t, response, test.wantStatus, test.wantCode)
			if body := response.Body.String(); containsAny(body, "must-not-leak", "postgres", "password") {
				t.Fatalf("upstream metadata leaked: %s", body)
			}
		})
	}
}

func transportReasoningFixtureResult[T any](t *testing.T, name string) (T, any) {
	t.Helper()
	var result T
	payload, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "testdata", "reasoning-tree-v1", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	var envelope struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatalf("decode fixture envelope: %v", err)
	}
	if err := json.Unmarshal(envelope.Result, &result); err != nil {
		t.Fatalf("decode typed fixture: %v", err)
	}
	var expected any
	if err := json.Unmarshal(envelope.Result, &expected); err != nil {
		t.Fatalf("decode expected fixture: %v", err)
	}
	return result, expected
}

func assertTransportJSONEquivalent(t *testing.T, want any, payload []byte) {
	t.Helper()
	var envelope struct {
		RequestID string `json:"request_id"`
		Result    any    `json:"result"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, payload)
	}
	if envelope.RequestID == "" {
		t.Fatal("response request_id is empty")
	}
	if !reflect.DeepEqual(envelope.Result, want) {
		t.Fatalf("response result = %#v, want %#v", envelope.Result, want)
	}
}

func assertReasoningError(t *testing.T, response *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()
	if response.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, wantStatus, response.Body.String())
	}
	var body struct {
		Error struct {
			Code    string         `json:"code"`
			Message string         `json:"message"`
			Details map[string]any `json:"details"`
		} `json:"error"`
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error: %v; body=%s", err, response.Body.String())
	}
	if body.Error.Code != wantCode || body.Error.Message == "" || body.Error.Details == nil || body.RequestID == "" {
		t.Fatalf("error = %#v, want code %s", body.Error, wantCode)
	}
	if response.Header().Get(dataclient.RequestIDHeader) != body.RequestID {
		t.Fatalf("header/body request IDs = %q/%q", response.Header().Get(dataclient.RequestIDHeader), body.RequestID)
	}
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if len(needle) > 0 && contains(value, needle) {
			return true
		}
	}
	return false
}

func contains(value, needle string) bool {
	for index := 0; index+len(needle) <= len(value); index++ {
		if value[index:index+len(needle)] == needle {
			return true
		}
	}
	return false
}
