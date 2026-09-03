package architecture

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRetainedRuntimeSmokeTraversesReportsDetailsEvidenceAndAdmin(t *testing.T) {
	t.Helper()
	seen := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		seen[request.URL.Path]++
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/healthz", "/minio/health/live", "/":
			response.WriteHeader(http.StatusOK)
			_, _ = response.Write([]byte(`{"status":"ok"}`))
		case "/api/miniapp/v1/reports/home":
			writeRetirementJSON(t, response, map[string]any{"result": map[string]any{
				"reports": []any{map[string]any{
					"report": map[string]any{"id": "RPT11111111-1111-4111-8111-111111111111", "industry_chain_count": 2},
					"cards": []any{
						map[string]any{"kind": "geopolitics", "evidence_scope_token": "RPE11111111-1111-4111-8111-111111111111"},
						map[string]any{"kind": "macroeconomics"},
						map[string]any{"kind": "industry_chain", "detail_ref": map[string]any{"local_key": "chain-01"}},
					},
					"next_cursor": "next",
				}},
			}})
		case "/api/miniapp/v1/reports/RPT11111111-1111-4111-8111-111111111111/industry-chains":
			if request.URL.Query().Get("limit") != "20" || request.URL.Query().Get("cursor") != "next" {
				http.Error(response, "bad pagination query", http.StatusBadRequest)
				return
			}
			writeRetirementJSON(t, response, map[string]any{"result": map[string]any{
				"items":       []any{map[string]any{"kind": "industry_chain", "detail_ref": map[string]any{"local_key": "chain-02"}}},
				"next_cursor": nil,
			}})
		case "/api/miniapp/v1/reports/RPT11111111-1111-4111-8111-111111111111/layers/geopolitics",
			"/api/miniapp/v1/reports/RPT11111111-1111-4111-8111-111111111111/layers/macroeconomics":
			writeRetirementJSON(t, response, map[string]any{"result": map[string]any{"layer": map[string]any{
				"anchors": []any{map[string]any{"local_key": "anchor-01"}},
			}}})
		case "/api/miniapp/v1/reports/RPT11111111-1111-4111-8111-111111111111/industry-chains/chain-01":
			writeRetirementJSON(t, response, map[string]any{"result": map[string]any{"industry_chain": map[string]any{
				"nodes": []any{map[string]any{"local_key": "node-01"}},
			}}})
		case "/api/miniapp/v1/reports/RPT11111111-1111-4111-8111-111111111111/evidences":
			if request.URL.Query().Get("scope_token") != "RPE11111111-1111-4111-8111-111111111111" {
				http.Error(response, "bad evidence scope", http.StatusBadRequest)
				return
			}
			writeRetirementJSON(t, response, map[string]any{"result": map[string]any{"items": []any{map[string]any{"summary": "fixture"}}}})
		case "/api/admin/v1/events":
			if request.Header.Get("Authorization") != "Bearer fixture-admin-token" {
				http.Error(response, "unauthorized", http.StatusUnauthorized)
				return
			}
			writeRetirementJSON(t, response, map[string]any{"result": map[string]any{"items": []any{}}})
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	root := repositoryRoot()
	command := exec.Command("python3", filepath.Join(root, "infra", "uat", "verify-retained-runtime.py"))
	command.Env = append(os.Environ(),
		"MINIAPP_SMOKE_BASE_URL="+server.URL,
		"ADMIN_SMOKE_BASE_URL="+server.URL,
		"MINIO_SMOKE_BASE_URL="+server.URL,
		"ADMIN_SERVICE_TOKEN=fixture-admin-token",
		"EXPECTED_INDUSTRY_CHAIN_COUNT=2",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("retained runtime smoke failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "PASS retained-api-smoke reports=1 chains=2 evidence_items=1 admin_proxy=200") {
		t.Fatalf("retained runtime smoke missed receipt: %s", output)
	}
	for _, path := range []string{
		"/api/miniapp/v1/reports/home",
		"/api/miniapp/v1/reports/RPT11111111-1111-4111-8111-111111111111/industry-chains",
		"/api/miniapp/v1/reports/RPT11111111-1111-4111-8111-111111111111/layers/geopolitics",
		"/api/miniapp/v1/reports/RPT11111111-1111-4111-8111-111111111111/layers/macroeconomics",
		"/api/miniapp/v1/reports/RPT11111111-1111-4111-8111-111111111111/industry-chains/chain-01",
		"/api/miniapp/v1/reports/RPT11111111-1111-4111-8111-111111111111/evidences",
		"/api/admin/v1/events",
	} {
		if seen[path] == 0 {
			t.Fatalf("retained runtime smoke did not request %s", path)
		}
	}
}

func TestUATLegacyRuntimeRetirementRejectsWrongConfirmationBeforeMutation(t *testing.T) {
	root := repositoryRoot()
	audit := filepath.Join(t.TempDir(), "rds-audit")
	rootRetirement := filepath.Join(t.TempDir(), "root-retirement")
	if err := os.WriteFile(audit, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rootRetirement, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("bash", filepath.Join(root, "infra", "uat", "retire-legacy-runtime.sh"))
	command.Env = append(os.Environ(),
		"DEPLOY_ROOT="+t.TempDir(),
		"UAT_RUNNER_NAME=fixture-runner",
		"RUNNER_NAME=fixture-runner",
		"RETIREMENT_CONFIRMATION=wrong",
		"RDS_AUDIT_BINARY="+audit,
		"ROOT_RETIREMENT_BINARY="+rootRetirement,
		"ADMIN_SERVICE_TOKEN=fixture-admin-token",
	)
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "FAIL confirmation") {
		t.Fatalf("wrong confirmation did not fail closed: %v\n%s", err, output)
	}
}

func writeRetirementJSON(t *testing.T, response http.ResponseWriter, value any) {
	t.Helper()
	if err := json.NewEncoder(response).Encode(value); err != nil {
		t.Fatal(err)
	}
}
