package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalApplicationComposeExcludesInfrastructureMiddleware(t *testing.T) {
	root := repositoryRoot()
	composePath := filepath.Join(root, "infra", "local", "docker-compose.yaml")
	readmePath := filepath.Join(root, "infra", "local", "README.md")

	compose, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatalf("read local compose: %v", err)
	}
	readme, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read local README: %v", err)
	}

	composeText := string(compose)
	readmeText := string(readme)

	for _, want := range []string{
		"data:",
		"data-migrate:",
		"miniapp:",
		"adminportal:",
		"admin:",
		"agentrun:",
		"agentrun-migrate:",
		"agentrun-agent-version:",
		"host.docker.internal:host-gateway",
		"9080",
	} {
		if !strings.Contains(composeText, want) {
			t.Fatalf("application compose missing %q", want)
		}
	}

	for _, forbidden := range []string{
		"\n  postgres:", "\n  neo4j:", "\n  qdrant:", "image: postgres:",
		"image: neo4j:", "image: qdrant/", "tidewise_postgres_data", "tidewise_neo4j_data",
		"tidewise_qdrant_data", "agentrun-db-init:",
	} {
		if strings.Contains(composeText, forbidden) {
			t.Fatalf("application Compose packages infrastructure middleware %q", forbidden)
		}
	}

	for _, want := range []string{
		"agent-run/backend",
		"TIDEWISW_DB_PASSWORD",
		"AGENTRUN_DB_PASSWORD",
		"DATA_SERVICE_TOKEN",
		"AGENTRUN_SERVICE_TOKEN",
		"EMBEDDING_API_KEY",
		"ADMIN_SERVICE_TOKEN",
		"run --rm",
		"externally provisioned infrastructure",
	} {
		if !strings.Contains(readmeText, want) {
			t.Fatalf("local README missing %q", want)
		}
	}

	for _, forbidden := range []string{
		"neo4j/password",
		"password: password",
		"NEO4J_PASSWORD=neo4j",
		"NEO4J_PASSWORD=password",
		"TIDEWISE_ENABLE_NEO4J_SMOKE",
		"cmd/graph-projector",
	} {
		if strings.Contains(composeText, forbidden) || strings.Contains(readmeText, forbidden) {
			t.Fatalf("local infra leaks forbidden secret pattern %q", forbidden)
		}
	}
}
