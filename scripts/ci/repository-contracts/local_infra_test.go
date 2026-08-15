package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalDockerProjectsSeparateApplicationsAndInfrastructure(t *testing.T) {
	root := repositoryRoot()
	applicationComposePath := filepath.Join(root, "infra", "local", "docker-compose.yaml")
	infrastructureComposePath := filepath.Join(root, "infra", "local", "docker-compose.infra.yaml")
	readmePath := filepath.Join(root, "infra", "local", "README.md")

	applicationCompose, err := os.ReadFile(applicationComposePath)
	if err != nil {
		t.Fatalf("read local application compose: %v", err)
	}
	infrastructureCompose, err := os.ReadFile(infrastructureComposePath)
	if err != nil {
		t.Fatalf("read local infrastructure compose: %v", err)
	}
	readme, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read local README: %v", err)
	}

	applicationText := string(applicationCompose)
	infrastructureText := string(infrastructureCompose)
	readmeText := string(readme)

	for _, want := range []string{
		"name: tidewise-app",
		"container_name: data-migrate",
		"container_name: data-service",
		"container_name: agentrun-service",
		"container_name: agentrun-migrate",
		"container_name: agentrun-agent-version",
		"container_name: miniapp-service",
		"container_name: admin-portal-service",
		"container_name: admin-portal-web",
		"disable: true",
		"data:",
		"data-migrate:",
		"miniapp:",
		"adminportal:",
		"admin:",
		"agentrun:",
		"agentrun-migrate:",
		"agentrun-agent-version:",
		"external: true",
		"9080",
	} {
		if !strings.Contains(applicationText, want) {
			t.Fatalf("application compose missing %q", want)
		}
	}

	for _, forbidden := range []string{
		"\n  postgres:", "\n  neo4j:", "\n  qdrant:", "image: postgres:",
		"image: neo4j:", "image: qdrant/", "tidewise_postgres_data", "tidewise_neo4j_data",
		"tidewise_qdrant_data", "agentrun-db-init:",
	} {
		if strings.Contains(applicationText, forbidden) {
			t.Fatalf("application Compose packages infrastructure middleware %q", forbidden)
		}
	}

	for _, want := range []string{
		"name: tidewise-infra", "  postgres:", "  mysql:", "  neo4j:", "  minio:", "  qdrant:",
		"external: true",
		"local_tidewise_postgres_data", "tidewise-reason_mysql-data", "tidewise-reason_neo4j-data",
		"tidewise-reason_minio-data", "tidewise-qdrant-local-storage", "name: '${COMPOSE_NETWORK_NAME:-tidewise-local}'",
	} {
		if !strings.Contains(infrastructureText, want) {
			t.Fatalf("infrastructure compose missing %q", want)
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
		"independently operated `tidewise-infra`",
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
		if strings.Contains(applicationText, forbidden) || strings.Contains(infrastructureText, forbidden) || strings.Contains(readmeText, forbidden) {
			t.Fatalf("local infra leaks forbidden secret pattern %q", forbidden)
		}
	}
}
