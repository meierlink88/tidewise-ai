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
		"\n  postgres:", "\n  neo4j:", "\n  qdrant:", "image: postgres:",
		"image: neo4j:", "image: qdrant/", "tidewise_postgres_data", "tidewise_neo4j_data",
		"tidewise_qdrant_data", "agentrun-db-init:", "agentrun", "AGENTRUN_", "9080",
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

func TestLocalNeo4jMatchesUATProviderContract(t *testing.T) {
	root := repositoryRoot()
	compose := mustReadText(t, filepath.Join(root, "infra", "local", "docker-compose.infra.yaml"))
	ensureVolumes := mustReadText(t, filepath.Join(root, "infra", "local", "ensure-volumes.sh"))
	packageJSON := mustReadText(t, filepath.Join(root, "package.json"))

	for _, want := range []string{
		"image: spg-registry.cn-hangzhou.cr.aliyuncs.com/spg/openspg-neo4j@sha256:4bc5b7f6b83d333b1d2c8f60ac145c068d77d50bca65b3a07c927f9e2a541eb9",
		"NEO4J_PLUGINS: '[\"apoc\"]'",
		"NEO4J_dbms_security_procedures_unrestricted: '*'",
		"NEO4J_dbms_security_procedures_allowlist: '*'",
		"name: tidewise-reason_neo4j-data",
		"name: tidewise-reason_neo4j-logs",
		"release-openspg-neo4j",
	} {
		if !strings.Contains(compose, want) {
			t.Fatalf("local Neo4j compose contract missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"neo4j:5.26.28-community",
		"neo4j-5.26-data",
		"neo4j-5.26-logs",
		"neo4j-5.26-plugins",
		"NEO4J_server_bolt_advertised__address",
	} {
		if strings.Contains(compose, forbidden) {
			t.Fatalf("local Neo4j compose retains abandoned 5.26 contract %q", forbidden)
		}
	}
	for _, want := range []string{
		"tidewise-reason_neo4j-data",
		"tidewise-reason_neo4j-logs",
	} {
		if !strings.Contains(ensureVolumes, want) {
			t.Fatalf("local volume provisioning missing %q", want)
		}
	}
	for _, forbidden := range []string{"neo4j-5.26-data", "neo4j-5.26-logs", "neo4j-5.26-plugins"} {
		if strings.Contains(ensureVolumes, forbidden) {
			t.Fatalf("local volume provisioning retains abandoned target %q", forbidden)
		}
	}
	if !strings.Contains(packageJSON, "infra/local/verify-neo4j.sh") {
		t.Fatal("local Neo4j verification entrypoint is missing")
	}
	for _, forbidden := range []string{"prepare-neo4j-plugins.sh", "upgrade-neo4j.sh", "rollback-neo4j.sh"} {
		if strings.Contains(packageJSON, forbidden) {
			t.Fatalf("package scripts retain abandoned Neo4j lifecycle entrypoint %q", forbidden)
		}
	}
	for _, relativePath := range []string{"infra/local/verify-neo4j.sh", "infra/local/verify-openspg-neo4j-consumer.sh"} {
		info, err := os.Stat(filepath.Join(root, relativePath))
		if err != nil {
			t.Fatalf("stat local Neo4j lifecycle entrypoint %s: %v", relativePath, err)
		}
		if info.Mode()&0o111 == 0 {
			t.Fatalf("local Neo4j lifecycle entrypoint is not executable: %s", relativePath)
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
