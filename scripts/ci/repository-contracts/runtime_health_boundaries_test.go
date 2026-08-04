package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeHealthPreservesServiceOwnershipBoundaries(t *testing.T) {
	root := repositoryRoot()
	frontend := readRuntimeBoundaryFile(t, root, "admin-portal/frontend/src/api/agentManagement.ts")
	if !strings.Contains(frontend, "'/api/admin/v1/runtime-health'") ||
		strings.Contains(frontend, "/api/data/v1/runtime-health") ||
		(strings.Contains(frontend, "qdrant") && strings.Contains(frontend, "/collections/")) ||
		strings.Contains(frontend, "bolt://") {
		t.Fatal("Admin frontend runtime health must call only the Admin BFF")
	}

	admin := readRuntimeBoundaryFile(t, root, "admin-portal/backend/internal/data/runtime_health.go")
	for _, forbidden := range []string{"neo4j-go-driver", "/collections/", "QDRANT_API_KEY", "DATA_NEO4J_HEALTH_PASSWORD"} {
		if strings.Contains(admin, forbidden) {
			t.Fatalf("Admin runtime health crosses provider ownership with %q", forbidden)
		}
	}

	adminOpenAPI := readRuntimeBoundaryFile(t, root, "admin-portal/backend/api/admin/v1/openapi.yaml")
	agentRunOpenAPI := readRuntimeBoundaryFile(t, root, "agent-run/backend/api/agentrun/v1/openapi.yaml")
	if strings.Contains(adminOpenAPI, "/api/admin/v1/agent-executions") {
		t.Fatal("Admin BFF retains the Collector Configuration-only execution history route")
	}
	if !strings.Contains(agentRunOpenAPI, "/api/admin/v1/agent-executions") {
		t.Fatal("AgentRun owner execution history route was removed")
	}
}

func readRuntimeBoundaryFile(t *testing.T, root, path string) string {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(contents)
}
