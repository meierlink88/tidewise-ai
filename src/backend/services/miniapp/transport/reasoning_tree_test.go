package transport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/dataclient"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/usecase"
)

func TestResearchReasoningTreeRoutesMapSharedFixturesWithOneDataCall(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		dataResult, expected := transportReasoningFixtureResult[dataclient.ResearchReasoningTreeList](t, "01-reasoning-tree-list-result.json")
		calls := 0
		client := &dataclient.Fake{ListResearchThemeReasoningTreesFunc: func(ctx context.Context, themeID string) (dataclient.ResearchReasoningTreeList, error) {
			calls++
			if dataclient.RequestIDFromContext(ctx) != "miniapp-reasoning-1" || themeID != "11111111-1111-4111-8111-111111111111" {
				t.Fatalf("request ID/theme ID = %q/%q", dataclient.RequestIDFromContext(ctx), themeID)
			}
			return dataResult, nil
		}}
		request := httptest.NewRequest(http.MethodGet, "/api/v1/miniapp/research/themes/11111111-1111-4111-8111-111111111111/reasoning-trees", nil)
		request.Header.Set(dataclient.RequestIDHeader, "miniapp-reasoning-1")
		response := httptest.NewRecorder()
		researchTestRouter(usecase.NewResearchService(client)).ServeHTTP(response, request)

		if response.Code != http.StatusOK || calls != 1 {
			t.Fatalf("status/calls = %d/%d, body=%s", response.Code, calls, response.Body.String())
		}
		assertTransportJSONEquivalent(t, expected, response.Body.Bytes())
	})

	t.Run("detail", func(t *testing.T) {
		dataResult, expected := transportReasoningFixtureResult[dataclient.ResearchReasoningTreeDetail](t, "02-reasoning-tree-with-contradiction-result.json")
		calls := 0
		client := &dataclient.Fake{GetResearchThemeReasoningTreeFunc: func(_ context.Context, themeID, anchorID string) (dataclient.ResearchReasoningTreeDetail, error) {
			calls++
			if themeID != "11111111-1111-4111-8111-111111111111" || anchorID != "534d83be-774b-51d9-ad00-cdee4ba91799" {
				t.Fatalf("theme/anchor IDs = %q/%q", themeID, anchorID)
			}
			return dataResult, nil
		}}
		response := serveResearch(t, usecase.NewResearchService(client), "/api/v1/miniapp/research/themes/11111111-1111-4111-8111-111111111111/reasoning-trees/534d83be-774b-51d9-ad00-cdee4ba91799")

		if response.Code != http.StatusOK || calls != 1 {
			t.Fatalf("status/calls = %d/%d, body=%s", response.Code, calls, response.Body.String())
		}
		assertTransportJSONEquivalent(t, expected, response.Body.Bytes())
	})
}

func TestResearchReasoningTreeRoutesRejectQueryAndInvalidUUIDBeforeDataCall(t *testing.T) {
	calls := 0
	client := &dataclient.Fake{
		ListResearchThemeReasoningTreesFunc: func(context.Context, string) (dataclient.ResearchReasoningTreeList, error) {
			calls++
			return dataclient.ResearchReasoningTreeList{}, nil
		},
		GetResearchThemeReasoningTreeFunc: func(context.Context, string, string) (dataclient.ResearchReasoningTreeDetail, error) {
			calls++
			return dataclient.ResearchReasoningTreeDetail{}, nil
		},
	}
	service := usecase.NewResearchService(client)
	for _, path := range []string{
		"/api/v1/miniapp/research/themes/11111111-1111-4111-8111-111111111111/reasoning-trees?window_hours=24",
		"/api/v1/miniapp/research/themes/11111111-1111-4111-8111-111111111111/reasoning-trees/534d83be-774b-51d9-ad00-cdee4ba91799?unused=1",
		"/api/v1/miniapp/research/themes/11111111-1111-4111-8111-11111111111A/reasoning-trees",
		"/api/v1/miniapp/research/themes/11111111-1111-4111-8111-111111111111/reasoning-trees/NOT-A-UUID",
	} {
		response := serveResearch(t, service, path)
		assertReasoningError(t, response, http.StatusBadRequest, "INVALID_REQUEST")
	}
	if calls != 0 {
		t.Fatalf("Data calls = %d, want 0", calls)
	}
}

func TestResearchReasoningTreeRoutesExposeStableErrorsWithoutUpstreamMetadata(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		code       string
		upstream   error
		wantStatus int
		wantCode   string
	}{
		{
			name: "Theme missing", path: "/api/v1/miniapp/research/themes/11111111-1111-4111-8111-111111111111/reasoning-trees",
			code: "RESEARCH_THEME_NOT_FOUND", wantStatus: http.StatusNotFound, wantCode: "RESEARCH_THEME_NOT_FOUND",
		},
		{
			name: "trees missing", path: "/api/v1/miniapp/research/themes/11111111-1111-4111-8111-111111111111/reasoning-trees",
			code: "RESEARCH_REASONING_TREES_NOT_FOUND", wantStatus: http.StatusNotFound, wantCode: "RESEARCH_REASONING_TREES_NOT_FOUND",
		},
		{
			name: "tree missing", path: "/api/v1/miniapp/research/themes/11111111-1111-4111-8111-111111111111/reasoning-trees/534d83be-774b-51d9-ad00-cdee4ba91799",
			code: "RESEARCH_REASONING_TREE_NOT_FOUND", wantStatus: http.StatusNotFound, wantCode: "RESEARCH_REASONING_TREE_NOT_FOUND",
		},
		{
			name: "unknown upstream 404", path: "/api/v1/miniapp/research/themes/11111111-1111-4111-8111-111111111111/reasoning-trees/534d83be-774b-51d9-ad00-cdee4ba91799",
			code: "UNEXPECTED_NOT_FOUND", wantStatus: http.StatusBadGateway, wantCode: "RESEARCH_DATA_UNAVAILABLE",
		},
		{
			name: "network", path: "/api/v1/miniapp/research/themes/11111111-1111-4111-8111-111111111111/reasoning-trees/534d83be-774b-51d9-ad00-cdee4ba91799",
			upstream: errors.New("dial postgres password=must-not-leak"), wantStatus: http.StatusBadGateway, wantCode: "RESEARCH_DATA_UNAVAILABLE",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			upstream := test.upstream
			if upstream == nil {
				upstream = &dataclient.Error{
					Kind: dataclient.ErrorKindClient, StatusCode: http.StatusNotFound,
					Code: test.code, RequestID: "must-not-leak",
				}
			}
			client := &dataclient.Fake{
				ListResearchThemeReasoningTreesFunc: func(context.Context, string) (dataclient.ResearchReasoningTreeList, error) {
					return dataclient.ResearchReasoningTreeList{}, upstream
				},
				GetResearchThemeReasoningTreeFunc: func(context.Context, string, string) (dataclient.ResearchReasoningTreeDetail, error) {
					return dataclient.ResearchReasoningTreeDetail{}, upstream
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
	var got any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, payload)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("response = %#v, want %#v", got, want)
	}
}

func assertReasoningError(t *testing.T, response *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()
	if response.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, wantStatus, response.Body.String())
	}
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error: %v; body=%s", err, response.Body.String())
	}
	if body.Error.Code != wantCode || body.Error.Message == "" {
		t.Fatalf("error = %#v, want code %s", body.Error, wantCode)
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
