package data

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/config"
)

func TestHealthAndReadiness(t *testing.T) {
	assertServiceHealth(t, NewHandler(testConfig()), ServiceName)
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
			Status      string             `json:"status"`
			Service     string             `json:"service"`
			Environment config.Environment `json:"environment"`
			Checks      map[string]string  `json:"checks"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode %s: %v", test.path, err)
		}
		if body.Status != test.wantStatus || body.Service != service || body.Environment != config.EnvLocal {
			t.Fatalf("%s response = %+v", test.path, body)
		}
		if test.path == "/readyz" && body.Checks["config"] != "ok" {
			t.Fatalf("%s checks = %v, want config=ok", test.path, body.Checks)
		}
	}
}

func testConfig() config.Config {
	return config.Config{
		App: config.AppConfig{Env: config.EnvLocal},
		Server: config.ServerConfig{
			Host:                "127.0.0.1",
			Port:                18081,
			ReadTimeoutSeconds:  5,
			WriteTimeoutSeconds: 10,
		},
	}
}
