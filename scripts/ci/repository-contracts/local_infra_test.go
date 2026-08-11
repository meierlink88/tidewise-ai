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
		"miniapp-h5:",
		"miniapp-weapp:",
		"miniapp-tt:",
		"adminportal:",
		"admin:",
		"agentrun:",
		"agentrun-migrate:",
		"host.docker.internal:host-gateway",
		"${NEO4J_USERNAME",
		"${NEO4J_PASSWORD",
		"7687",
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
		"NEO4J_USERNAME",
		"NEO4J_PASSWORD",
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

func TestDockerOnlyRuntimeHasNoHostNativeEntrypoints(t *testing.T) {
	root := repositoryRoot()
	for _, path := range []string{
		"infra/local/README.md",
		"agent-run/backend/README.md",
		"analyse-data-service/backend/migrations/README.md",
		"analyse-data-service/backend/data/research_themes/README.md",
		"miniapp/frontend/README.md",
		"package.json",
	} {
		contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatalf("read Docker-only runtime contract %q: %v", path, err)
		}
		for _, forbidden := range []string{"go run ./", "npm run dev --"} {
			if strings.Contains(string(contents), forbidden) {
				t.Fatalf("Docker-only runtime contract %q retains host command %q", path, forbidden)
			}
		}
	}
}

func TestComposeSmokesProvisionInfrastructureOnlyAsTestFixtures(t *testing.T) {
	root := repositoryRoot()
	for _, path := range []string{
		"scripts/ci/smoke-miniapp-data-compose.sh",
		"scripts/ci/smoke-admin-agentrun-compose.sh",
	} {
		contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatalf("read Compose smoke %q: %v", path, err)
		}
		text := string(contents)
		for _, required := range []string{"-fixture", "postgres:16", "neo4j:5-community", "TIDEWISE_DB_HOST"} {
			if !strings.Contains(text, required) {
				t.Fatalf("Compose smoke %q missing isolated infrastructure fixture %q", path, required)
			}
		}
		for _, forbidden := range []string{"up -d --wait postgres", "data /usr/local/bin/dbmigrate"} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("Compose smoke %q expects middleware from the application Compose: %q", path, forbidden)
			}
		}
	}
}
