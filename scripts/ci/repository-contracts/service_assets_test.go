package architecture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceOwnedDockerAssetsReplaceLegacyBackendImage(t *testing.T) {
	repoRoot := repositoryRoot()
	assets := []struct {
		name       string
		root       string
		binary     string
		port       string
		command    string
		configDir  string
		mustCopyDB bool
	}{
		{name: "analyse-data-service", root: "analyse-data-service/backend", binary: "data-service", port: "9011", command: "cmd/server", configDir: "configs", mustCopyDB: true},
		{name: "miniapp", root: "miniapp/backend", binary: "miniapp-service", port: "9012", command: "cmd/server", configDir: "configs"},
		{name: "admin-portal", root: "admin-portal/backend", binary: "adminportal-service", port: "9013", command: "cmd/server", configDir: "configs"},
	}

	for _, asset := range assets {
		path := filepath.Join(repoRoot, filepath.FromSlash(asset.root), "Dockerfile")
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s Dockerfile: %v", asset.name, err)
		}
		text := string(contents)
		for _, required := range []string{
			"./" + asset.root + "/" + asset.command,
			"/usr/local/bin/" + asset.binary,
			"/healthz",
			"/readyz",
			"CMD [\"/usr/local/bin/" + asset.binary + "\"]",
		} {
			if !strings.Contains(text, required) {
				t.Fatalf("%s Dockerfile missing %q", asset.name, required)
			}
		}
		if asset.mustCopyDB && !strings.Contains(text, "COPY analyse-data-service/backend/migrations ./migrations") {
			t.Fatal("Data Dockerfile must own migration assets")
		}
		if !strings.Contains(text, "COPY "+asset.root+"/"+asset.configDir+" ./config") {
			t.Fatalf("%s Dockerfile must copy its service-owned start config", asset.name)
		}
		configPath := filepath.Join(repoRoot, filepath.FromSlash(asset.root), asset.configDir, "config.local.yaml")
		configContents, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("read %s start config: %v", asset.name, err)
		}
		configText := string(configContents)
		for _, required := range []string{"host: 0.0.0.0", "port: " + asset.port} {
			if !strings.Contains(configText, required) {
				t.Fatalf("%s start config missing %q", asset.name, required)
			}
		}
		if !asset.mustCopyDB {
			for _, forbidden := range []string{"COPY migrations", "dbmigrate", "TIDEWISW_DB_PASSWORD", "AGENTRUN_DB_PASSWORD"} {
				if strings.Contains(text, forbidden) {
					t.Fatalf("%s Dockerfile carries Data DB concern %q", asset.name, forbidden)
				}
				if strings.Contains(configText, forbidden) || strings.Contains(configText, "database:") || strings.Contains(configText, "migration:") {
					t.Fatalf("%s start config carries Data DB concern %q", asset.name, forbidden)
				}
			}
		}
	}

	if _, err := os.Stat(filepath.Join(repoRoot, "Dockerfile")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("repository root must not define a legacy aggregate Dockerfile: %v", err)
	}
}

func TestDataImageCarriesUATIndustryRelationshipAndGraphAssets(t *testing.T) {
	repoRoot := repositoryRoot()
	dockerfile := readContractFile(t, filepath.Join(
		repoRoot,
		"analyse-data-service",
		"backend",
		"Dockerfile",
	))
	deploy := readContractFile(t, filepath.Join(repoRoot, "infra", "uat", "deploy.sh"))

	const (
		importerBinary  = "/usr/local/bin/industry-relationship-import"
		projectorBinary = "/usr/local/bin/industry-graph-projector"
		packagePath     = "/app/data/industry_relationships/2026-07-27-v1"
	)
	for _, required := range []string{
		importerBinary,
		projectorBinary,
		`industry_package_path="` + packagePath + `"`,
	} {
		if !strings.Contains(deploy, required) {
			t.Fatalf("UAT deploy contract missing %q", required)
		}
	}

	if !strings.Contains(
		dockerfile,
		"-o /out/industry-relationship-import ./analyse-data-service/backend/cmd/industry-relationship-import",
	) {
		t.Fatal("Data Dockerfile must build the UAT Industry relationship importer")
	}
	if !strings.Contains(
		dockerfile,
		"-o /out/industry-graph-projector ./analyse-data-service/backend/cmd/industry-graph-projector",
	) {
		t.Fatal("Data Dockerfile must build the UAT Industry graph projector")
	}
	runtimeStageStart := strings.LastIndex(dockerfile, "\nFROM ")
	if runtimeStageStart < 0 {
		t.Fatal("Data Dockerfile must define a separate runtime stage")
	}
	runtimeStage := dockerfile[runtimeStageStart:]
	for _, required := range []string{
		"COPY --from=builder /out/industry-relationship-import " + importerBinary,
		"COPY --from=builder /out/industry-graph-projector " + projectorBinary,
		"COPY analyse-data-service/backend/data ./data",
	} {
		if !strings.Contains(runtimeStage, required) {
			t.Fatalf("Data runtime image missing %q", required)
		}
	}
}

func TestDataImageCarriesUATEventSemanticProjectionAssets(t *testing.T) {
	repoRoot := repositoryRoot()
	dockerfile := readContractFile(t, filepath.Join(
		repoRoot,
		"analyse-data-service",
		"backend",
		"Dockerfile",
	))
	deploy := readContractFile(t, filepath.Join(repoRoot, "infra", "uat", "deploy.sh"))

	for _, required := range []string{
		"-o /out/event-semantic-projector ./analyse-data-service/backend/cmd/event-semantic-projector",
		"-o /out/uat-excluded-fact-audit ./analyse-data-service/backend/cmd/uat-excluded-fact-audit",
		"COPY --from=builder /out/event-semantic-projector /usr/local/bin/event-semantic-projector",
		"COPY --from=builder /out/uat-excluded-fact-audit /usr/local/bin/uat-excluded-fact-audit",
	} {
		if !strings.Contains(dockerfile, required) {
			t.Fatalf("Data image missing UAT Event Semantic asset %q", required)
		}
	}
	for _, required := range []string{
		"/usr/local/bin/event-semantic-projector",
		"/usr/local/bin/uat-excluded-fact-audit",
	} {
		if !strings.Contains(deploy, required) {
			t.Fatalf("UAT deploy contract missing Data maintenance command %q", required)
		}
	}
}

func TestServiceImagesCarryEventSemanticHistoryMaintenanceCommands(t *testing.T) {
	repoRoot := repositoryRoot()
	dataDockerfile := readContractFile(t, filepath.Join(
		repoRoot,
		"analyse-data-service",
		"backend",
		"Dockerfile",
	))
	agentrunDockerfile := readContractFile(t, filepath.Join(
		repoRoot,
		"agent-run",
		"backend",
		"Dockerfile",
	))
	uatRunbook := readContractFile(t, filepath.Join(
		repoRoot,
		"infra",
		"uat",
		"README.md",
	))

	for _, required := range []string{
		"-o /out/event-semantic-history-audit ./analyse-data-service/backend/cmd/event-semantic-history-audit",
		"COPY --from=builder /out/event-semantic-history-audit /usr/local/bin/event-semantic-history-audit",
	} {
		if !strings.Contains(dataDockerfile, required) {
			t.Fatalf("Data runtime image missing Event Semantic history contract %q", required)
		}
	}
	for _, required := range []string{
		"-o /out/agentrun-event-semantic-history ./agent-run/backend/cmd/event-semantic-history",
		"COPY --from=build /out/agentrun-event-semantic-history /app/agentrun-event-semantic-history",
	} {
		if !strings.Contains(agentrunDockerfile, required) {
			t.Fatalf("AgentRun runtime image missing Event Semantic history contract %q", required)
		}
	}
	for _, required := range []string{
		"/usr/local/bin/event-semantic-history-audit",
		"/app/agentrun-event-semantic-history",
	} {
		if !strings.Contains(uatRunbook, required) {
			t.Fatalf("UAT runbook missing Event Semantic history command %q", required)
		}
	}
}

func TestApplicationRootsAreCanonical(t *testing.T) {
	repoRoot := repositoryRoot()
	for _, path := range []string{
		"miniapp/frontend",
		"miniapp/backend",
		"admin-portal/frontend",
		"admin-portal/backend",
		"analyse-data-service/backend",
		"agent-run/backend",
	} {
		info, err := os.Lstat(filepath.Join(repoRoot, filepath.FromSlash(path)))
		if err != nil {
			t.Fatalf("canonical application source %q is missing: %v", path, err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("canonical application source %q must be a real directory", path)
		}
	}
	for _, path := range []string{"src", "backend", "frontend", "services"} {
		if _, err := os.Lstat(filepath.Join(repoRoot, path)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("legacy product source %q must be absent: %v", path, err)
		}
	}
}

func TestRetiredDataGraphProjectorIsAbsent(t *testing.T) {
	repoRoot := repositoryRoot()
	for _, path := range []string{
		"analyse-data-service/backend/adapters/graphdb",
		"analyse-data-service/backend/cmd/graph-projector",
		"analyse-data-service/backend/internal/biz/graphprojection",
		"analyse-data-service/backend/internal/data/graphdb",
		"analyse-data-service/backend/internal/data/postgres/graph_projection.go",
		"analyse-data-service/backend/usecase/graphprojection",
	} {
		if _, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(path))); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("retired Data graph projection path %q must be absent: %v", path, err)
		}
	}

	for _, path := range []string{
		"analyse-data-service/backend/configs/config.local.yaml",
		"analyse-data-service/backend/configs/config.uat.yaml",
	} {
		contents, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(path)))
		if err != nil {
			t.Fatalf("read Data config %q: %v", path, err)
		}
		if strings.Contains(string(contents), "\nneo4j:") {
			t.Fatalf("Data Server config %q retains retired Neo4j runtime configuration", path)
		}
	}
}

func TestAgentRunMigrationManifestAccountsForEveryFrozenTrackedFile(t *testing.T) {
	repoRoot := repositoryRoot()
	path := filepath.Join(repoRoot, "docs", "architecture", "agentrun", "agentrun-monorepo-file-disposition.tsv")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read AgentRun file disposition manifest: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(contents)), "\n")
	if len(lines) != 110 {
		t.Fatalf("AgentRun file disposition rows = %d, want header plus 109 frozen files", len(lines))
	}
	if lines[0] != "source_path\tdisposition\tdestination" {
		t.Fatalf("AgentRun file disposition header = %q", lines[0])
	}
	allowed := map[string]bool{
		"moved":                   true,
		"merged":                  true,
		"superseded-as-duplicate": true,
		"retired-low-value-test":  true,
	}
	seen := make(map[string]bool, len(lines)-1)
	for lineNumber, line := range lines[1:] {
		fields := strings.Split(line, "\t")
		if len(fields) != 3 {
			t.Fatalf("AgentRun file disposition line %d has %d fields", lineNumber+2, len(fields))
		}
		source, disposition, destinations := fields[0], fields[1], fields[2]
		if seen[source] {
			t.Fatalf("AgentRun source asset %q is classified more than once", source)
		}
		seen[source] = true
		if !allowed[disposition] {
			t.Fatalf("AgentRun source asset %q has unsupported disposition %q", source, disposition)
		}
		if disposition == "retired-low-value-test" {
			if destinations != "-" {
				t.Fatalf("retired AgentRun test %q destination = %q, want -", source, destinations)
			}
			continue
		}
		for _, destination := range strings.Split(destinations, ";") {
			if destination == "" {
				t.Fatalf("AgentRun source asset %q has an empty destination", source)
			}
			if _, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(destination))); err != nil {
				t.Fatalf("AgentRun source asset %q destination %q is missing: %v", source, destination, err)
			}
		}
	}
}

func TestLocalComposeOwnsOnlyApplicationServices(t *testing.T) {
	repoRoot := repositoryRoot()
	contents, err := os.ReadFile(filepath.Join(repoRoot, "infra", "local", "docker-compose.yaml"))
	if err != nil {
		t.Fatalf("read local compose: %v", err)
	}
	text := string(contents)
	for _, required := range []string{
		"  data:", "  data-migrate:", "  miniapp:",
		"  adminportal:", "  admin:", "  agentrun:", "  agentrun-migrate:",
		"context: ../..",
		"analyse-data-service/backend/Dockerfile", "miniapp/backend/Dockerfile",
		"admin-portal/backend/Dockerfile", "admin-portal/frontend/Dockerfile", "agent-run/backend/Dockerfile",
		"tidewise-local", "/healthz", "/readyz",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("local compose missing %q", required)
		}
	}
	for _, forbidden := range []string{"\n  postgres:", "\n  neo4j:", "\n  qdrant:", "agentrun-db-init:", "image: postgres:", "image: neo4j:", "image: qdrant/"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("local application Compose owns infrastructure middleware %q", forbidden)
		}
	}
	for _, forbidden := range []string{"ingestion-scheduler", "source-ingest", "ingest-smoke"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("local compose revives retired runtime %q", forbidden)
		}
	}
	for _, bff := range []string{"miniapp", "adminportal"} {
		section := composeServiceSection(t, text, bff)
		for _, forbidden := range []string{"TIDEWISW_DB_PASSWORD", "AGENTRUN_DB_PASSWORD", "POSTGRES_", "NEO4J_"} {
			if strings.Contains(section, forbidden) {
				t.Fatalf("%s compose service carries Data credential %q", bff, forbidden)
			}
		}
	}
	data := composeServiceSection(t, text, "data")
	for _, required := range []string{"DATA_NEO4J_HEALTH_USERNAME", "DATA_NEO4J_HEALTH_PASSWORD"} {
		if !strings.Contains(data, required) {
			t.Fatalf("Data compose service is missing runtime health dependency %q", required)
		}
	}
	for _, forbidden := range []string{"\n      NEO4J_USERNAME:", "\n      NEO4J_PASSWORD:"} {
		if strings.Contains(data, forbidden) {
			t.Fatalf("Data compose service retains retired graph projection dependency %q", forbidden)
		}
	}
	agentrun := composeServiceSection(t, text, "agentrun")
	for _, required := range []string{"AGENTRUN_CONFIG_DIR: /app/configs", "AGENTRUN_DB_HOST", "AGENTRUN_QDRANT_URL", "AGENTRUN_DB_PASSWORD", "AGENTRUN_SERVICE_TOKEN", "DATA_SERVICE_TOKEN", "EMBEDDING_API_KEY", "QDRANT_API_KEY", "agentrun_artifacts"} {
		if !strings.Contains(agentrun, required) {
			t.Fatalf("AgentRun compose service missing %q", required)
		}
	}
	dataComposeConfig := readContractFile(t, filepath.Join(repoRoot, "analyse-data-service", "backend", "configs", "config.local.yaml"))
	agentRunComposeConfig := readContractFile(t, filepath.Join(repoRoot, "agent-run", "backend", "configs", "config.dev.yaml"))
	for service, config := range map[string]string{"Data": dataComposeConfig, "AgentRun": agentRunComposeConfig} {
		if !strings.Contains(config, "host: host.docker.internal") {
			t.Fatalf("%s local config must use a container-reachable external infrastructure host", service)
		}
	}
	if !strings.Contains(dataComposeConfig, "uri: bolt://host.docker.internal:7687") {
		t.Fatal("Data local config must use a container-reachable external Neo4j host")
	}
	for _, required := range []string{"base_url: http://data:9011", "qdrant_url: http://host.docker.internal:6333"} {
		if !strings.Contains(agentRunComposeConfig, required) {
			t.Fatalf("AgentRun local config missing %q", required)
		}
	}
	if !strings.Contains(data, "TIDEWISE_CONFIG_DIR: /app/configs") {
		t.Fatal("Data local Compose service must use the canonical image config directory")
	}
	for _, path := range []string{
		"analyse-data-service/backend/configs/compose",
		"agent-run/backend/configs/compose",
	} {
		if _, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(path))); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Docker-only runtime retains duplicate config tree %q", path)
		}
	}
}

func TestCIConsumesServiceOwnedImagesAndBoundaryContracts(t *testing.T) {
	repoRoot := repositoryRoot()
	contents, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read CI workflow: %v", err)
	}
	text := string(contents)
	for _, required := range []string{
		"go-version-file: go.mod",
		"cache-dependency-path: go.sum",
		"go test ./analyse-data-service/backend/api/data/v1 ./miniapp/backend/internal/data",
		"go test ./analyse-data-service/backend/api/data/v1 ./agent-run/backend/api/agentrun/v1 ./admin-portal/backend/internal/data",
		"go test ./scripts/ci/repository-contracts -count=1",
		"go build -o /tmp/data-service ./analyse-data-service/backend/cmd/server",
		"go build -o /tmp/miniapp-service ./miniapp/backend/cmd/server",
		"go build -o /tmp/adminportal-service ./admin-portal/backend/cmd/server",
		"go build -o /tmp/agentrun ./agent-run/backend/cmd/server",
		"-f analyse-data-service/backend/Dockerfile",
		"-f miniapp/backend/Dockerfile",
		"-f miniapp/frontend/Dockerfile",
		"-f admin-portal/backend/Dockerfile",
		"-f agent-run/backend/Dockerfile",
		"Test AgentRun Biz, API and Eino seams",
		"Test AgentRun Data, migration and provider boundaries",
		"docker compose --env-file infra/local/.env.example -f infra/local/docker-compose.yaml config --quiet",
		"docker compose --env-file infra/local/.env.example -f infra/local/miniapp-builder.compose.yaml --profile miniapp-h5 --profile miniapp-weapp --profile miniapp-tt config --quiet",
		"docker compose --env-file infra/uat/.env.example -f infra/uat/docker-compose.yaml config --quiet",
		"bash scripts/ci/smoke-miniapp-data-compose.sh",
		"bash scripts/ci/smoke-miniapp-builder.sh",
		"cache-dependency-path: package-lock.json",
		"npm run test:miniapp",
		"npm run test:admin",
		"npm run typecheck:miniapp",
		"npm run typecheck:admin",
		"npm run build:weapp",
		"npm run build:admin",
		"docker build -f admin-portal/frontend/Dockerfile -t tidewise-admin:ci .",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("CI workflow missing %q", required)
		}
	}
	if strings.Contains(text, "docker build -f Dockerfile") {
		t.Fatal("CI must not consume the legacy backend Dockerfile")
	}
	for _, job := range []string{"governance", "data", "miniapp", "adminportal", "agentrun", "security"} {
		if strings.Count(text, "\n  "+job+":") != 1 {
			t.Fatalf("CI must expose exactly one top-level %s job", job)
		}
	}
	if strings.Contains(text, "\n  changes:") || strings.Contains(text, "needs: changes") {
		t.Fatal("application path detection must stay inside each application job")
	}
	if strings.Contains(text, "\n  agentrun-postgres:") {
		t.Fatal("AgentRun PostgreSQL verification must stay inside the AgentRun job")
	}
	for _, required := range []string{
		"name: Data Service",
		"name: Repository Governance",
		"name: Miniapp",
		"name: Admin Portal",
		"name: AgentRun",
		"bash scripts/ci/detect-app-change.sh data",
		"bash scripts/ci/detect-app-change.sh miniapp",
		"bash scripts/ci/detect-app-change.sh adminportal",
		"bash scripts/ci/detect-app-change.sh agentrun",
		"POSTGRES_DB: tidewise_ai_server_test",
		"Test AgentRun Data, migration and provider boundaries",
		"Build Data and Admin Portal images for AgentRun smoke",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("unified AgentRun CI job missing %q", required)
		}
	}

	agentRunSmokeContents, err := os.ReadFile(filepath.Join(
		repoRoot, "scripts", "ci", "smoke-admin-agentrun-compose.sh",
	))
	if err != nil {
		t.Fatalf("read Admin Portal to AgentRun Compose smoke script: %v", err)
	}
	agentRunSmoke := string(agentRunSmokeContents)
	for _, required := range []string{
		"data-migrate",
		"agentrun-migrate",
		"DATA_SERVICE_IMAGE=\"tidewise-data:ci\"",
		"AGENTRUN_SERVICE_IMAGE=\"tidewise-agentrun:ci\"",
		"ADMIN_SERVICE_IMAGE=\"tidewise-adminportal:ci\"",
		"TIDEWISE_SMOKE_AGENTRUN_DATA_PORT",
	} {
		if !strings.Contains(agentRunSmoke, required) {
			t.Fatalf("Admin Portal to AgentRun Compose smoke script missing %q", required)
		}
	}

	smokeContents, err := os.ReadFile(filepath.Join(repoRoot, "scripts", "ci", "smoke-miniapp-data-compose.sh"))
	if err != nil {
		t.Fatalf("read Miniapp Compose smoke script: %v", err)
	}
	smoke := string(smokeContents)
	for _, required := range []string{
		"data-migrate",
		"PGOPTIONS",
		"tidewise.phase_a_cleanup_write_authorized=reviewed_backup_verified",
		"tidewise.external_identifier_schema_write_authorized=reviewed_backup_verified",
		"tidewise.alliance_economy_schema_write_authorized=reviewed_local_cleanup_verified",
		"/api/miniapp/v1/research/themes",
		"APP_ENV=prod",
		"/docs/",
		"--signal=SIGTERM",
		"did not stop within 15 seconds",
		"COMPOSE_NETWORK_NAME",
		"DATA_SERVICE_IMAGE=\"tidewise-data:ci\"",
		"MINIAPP_SERVICE_IMAGE=\"tidewise-miniapp:ci\"",
	} {
		if !strings.Contains(smoke, required) {
			t.Fatalf("Miniapp Compose smoke script missing %q", required)
		}
	}
}

func composeServiceSection(t *testing.T, compose, service string) string {
	t.Helper()
	startMarker := "  " + service + ":\n"
	start := strings.Index(compose, startMarker)
	if start < 0 {
		t.Fatalf("compose service %q is missing", service)
	}
	remainder := compose[start+len(startMarker):]
	lines := strings.Split(remainder, "\n")
	end := len(lines)
	for index, line := range lines {
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && strings.TrimSpace(line) != "" {
			end = index
			break
		}
	}
	return strings.Join(lines[:end], "\n")
}
