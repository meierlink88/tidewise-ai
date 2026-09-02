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
		"container_name: data-service",
		"container_name: miniapp-service",
		"container_name: admin-portal-service",
		"container_name: admin-portal-web",
		"disable: true",
		"data:",
		"data-migrate:",
		"miniapp:",
		"adminportal:",
		"admin:",
		"external: true",
	} {
		if !strings.Contains(applicationText, want) {
			t.Fatalf("application compose missing %q", want)
		}
	}
	if strings.Contains(applicationText, "container_name: data-migrate") {
		t.Fatal("ephemeral Data migration template must not assign a stable container name")
	}

	for _, forbidden := range []string{
		"\n  postgres:", "\n  mysql:", "\n  neo4j:", "\n  qdrant:", "image: postgres:",
		"image: mysql:", "openspg-mysql", "image: neo4j:", "image: qdrant/", "tidewise_postgres_data", "tidewise_neo4j_data",
		"tidewise_qdrant_data", "agentrun-db-init:", "agentrun", "AGENTRUN_", "9080",
	} {
		if strings.Contains(applicationText, forbidden) {
			t.Fatalf("application Compose packages infrastructure middleware %q", forbidden)
		}
	}

	for _, want := range []string{
		"name: tidewise-infra", "  postgres:", "  minio:",
		"external: true",
		"local_tidewise_postgres_data", "tidewise-reason_minio-data",
		"name: '${COMPOSE_NETWORK_NAME:-tidewise-local}'",
	} {
		if !strings.Contains(infrastructureText, want) {
			t.Fatalf("infrastructure compose missing %q", want)
		}
	}
	for _, forbidden := range []string{"\n  neo4j:", "image: neo4j:", "tidewise-reason_neo4j"} {
		if strings.Contains(infrastructureText, forbidden) {
			t.Fatalf("Tidewise AI infrastructure still owns reasoning Neo4j %q", forbidden)
		}
	}

	for _, want := range []string{
		"DATA_SERVICE_TOKEN",
		"ADMIN_SERVICE_TOKEN",
		"`tidewise-infra`",
		"`tidewise-app`",
	} {
		if !strings.Contains(readmeText, want) {
			t.Fatalf("local README missing %q", want)
		}
	}
	for _, forbidden := range []string{"AgentRun", "agent-run", "agentrun", "AGENTRUN_"} {
		if strings.Contains(readmeText, forbidden) {
			t.Fatalf("local README still presents retired AgentRun contract %q", forbidden)
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

func TestLocalInfrastructureRetiresUnusedMySQLAndQdrant(t *testing.T) {
	root := repositoryRoot()
	infrastructureCompose := mustReadText(t, filepath.Join(root, "infra", "local", "docker-compose.infra.yaml"))
	files := []struct {
		name     string
		contents string
	}{
		{name: "Compose", contents: infrastructureCompose},
		{name: "environment", contents: mustReadText(t, filepath.Join(root, "infra", "local", ".env.example"))},
		{name: "volume bootstrap", contents: mustReadText(t, filepath.Join(root, "infra", "local", "ensure-volumes.sh"))},
	}
	for _, file := range files {
		for _, retired := range []string{
			"openspg-mysql", "tidewise-reason_mysql-data", "MYSQL_PORT", "OPENSPG_MYSQL_ROOT_PASSWORD",
			"qdrant/qdrant", "tidewise-qdrant-local-storage", "QDRANT_HTTP_PORT", "QDRANT_GRPC_PORT",
		} {
			if strings.Contains(file.contents, retired) {
				t.Errorf("local %s retains retired infrastructure value %q", file.name, retired)
			}
		}
	}
	for _, retiredComposeValue := range []string{
		"\n  mysql:", "\n  qdrant:", "\n  mysql-data:", "\n  qdrant-data:",
		"3306:3306", "6333:6333", "6334:6334",
	} {
		if strings.Contains(infrastructureCompose, retiredComposeValue) {
			t.Errorf("local infrastructure Compose retains retired value %q", retiredComposeValue)
		}
	}
}

func TestLocalInfrastructureDoesNotOwnReasoningNeo4j(t *testing.T) {
	root := repositoryRoot()
	compose := mustReadText(t, filepath.Join(root, "infra", "local", "docker-compose.infra.yaml"))
	ensureVolumes := mustReadText(t, filepath.Join(root, "infra", "local", "ensure-volumes.sh"))
	packageJSON := mustReadText(t, filepath.Join(root, "package.json"))

	for _, forbidden := range []string{
		"\n  neo4j:", "image: neo4j:", "openspg-neo4j", "release-openspg-neo4j",
		"tidewise-reason_neo4j-data", "tidewise-reason_neo4j-logs", "NEO4J_",
	} {
		if strings.Contains(compose, forbidden) {
			t.Fatalf("local infrastructure Compose retains reasoning Neo4j contract %q", forbidden)
		}
		if strings.Contains(ensureVolumes, forbidden) {
			t.Fatalf("local volume provisioning retains reasoning Neo4j contract %q", forbidden)
		}
		if strings.Contains(packageJSON, forbidden) {
			t.Fatalf("package scripts retain reasoning Neo4j contract %q", forbidden)
		}
	}
	for _, relativePath := range []string{"infra/local/verify-neo4j.sh", "infra/local/verify-openspg-neo4j-consumer.sh"} {
		if _, err := os.Stat(filepath.Join(root, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("reasoning Neo4j lifecycle entrypoint still exists: %s", relativePath)
		}
	}
}

func mustReadText(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(contents)
}
