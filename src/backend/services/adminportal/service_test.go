package adminportal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/platform/runtimeconfig"
	adminconfig "github.com/meierlink88/tidewise-ai/backend/services/adminportal/config"
	"github.com/meierlink88/tidewise-ai/backend/services/adminportal/dataclient"
)

func TestHealthAndReadiness(t *testing.T) {
	assertServiceHealth(t, NewHandler(testConfig(), nil, ""), ServiceName)
}

func TestNewHandlerComposesAdminBFFWithOneDataServiceCall(t *testing.T) {
	calls := 0
	client := &dataclient.Fake{ListRawDocumentsFunc: func(context.Context, dataclient.RawDocumentListQuery) (dataclient.RawDocumentPage, error) {
		calls++
		return dataclient.RawDocumentPage{Items: []dataclient.RawDocument{}, Page: 1, PageSize: 50}, nil
	}}
	handler := NewHandler(testConfig(), client, "admin-token")
	assertServiceHealth(t, handler, ServiceName)

	request := httptest.NewRequest(http.MethodGet, "/admin/raw-documents", nil)
	request.Header.Set("Authorization", "Bearer admin-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || calls != 1 {
		t.Fatalf("status/calls = %d/%d, body=%s", response.Code, calls, response.Body.String())
	}
}

func TestNewServerPreservesCompatibilityHandler(t *testing.T) {
	legacy := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	})
	server := NewServer(testConfig(), legacy)
	response := httptest.NewRecorder()
	server.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/admin/raw-documents", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("compatibility status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func assertServiceHealth(t *testing.T, handler http.Handler, service string) {
	t.Helper()
	for _, test := range []struct {
		path       string
		wantStatus string
	}{
		{path: "/healthz", wantStatus: "ok"},
		{path: "/readyz", wantStatus: "ready"},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d", test.path, response.Code, http.StatusOK)
		}
		var body struct {
			Status      string                    `json:"status"`
			Service     string                    `json:"service"`
			Environment runtimeconfig.Environment `json:"environment"`
			Checks      map[string]string         `json:"checks"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode %s: %v", test.path, err)
		}
		if body.Status != test.wantStatus || body.Service != service || body.Environment != runtimeconfig.EnvLocal {
			t.Fatalf("%s response = %+v", test.path, body)
		}
		if test.path == "/readyz" && body.Checks["config"] != "ok" {
			t.Fatalf("%s checks = %v, want config=ok", test.path, body.Checks)
		}
	}
}

func testConfig() adminconfig.RuntimeConfig {
	return adminconfig.RuntimeConfig{
		App: runtimeconfig.AppConfig{Env: runtimeconfig.EnvLocal},
		Server: runtimeconfig.ServerConfig{
			Host:                "127.0.0.1",
			Port:                18083,
			ReadTimeoutSeconds:  5,
			WriteTimeoutSeconds: 10,
		},
	}
}
