package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeHealthPreservesServiceOwnershipBoundaries(t *testing.T) {
	root := repositoryRoot()
	admin := readRuntimeBoundaryFile(t, root, "admin-portal/backend/internal/data/runtime_health.go")
	for _, forbidden := range []string{"neo4j-go-driver", "/collections/", "QDRANT_API_KEY", "DATA_NEO4J_HEALTH_PASSWORD"} {
		if strings.Contains(admin, forbidden) {
			t.Fatalf("Admin runtime health crosses provider ownership with %q", forbidden)
		}
	}

	adminOpenAPI := readRuntimeBoundaryFile(t, root, "admin-portal/backend/api/admin/v1/openapi.yaml")
	for _, retired := range []string{"/api/admin/v1/agent-executions", "agentrun", "qdrant"} {
		if strings.Contains(strings.ToLower(adminOpenAPI), retired) {
			t.Fatalf("Admin runtime health retains retired dependency %q", retired)
		}
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
