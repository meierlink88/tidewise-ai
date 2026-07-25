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
		{name: "analyse-data-service", root: "analyse-data-service/backend", binary: "data-service", port: "9011", command: "cmd", configDir: "config", mustCopyDB: true},
		{name: "miniapp", root: "miniapp/backend", binary: "miniapp-service", port: "9012", command: "cmd/server", configDir: "configs"},
		{name: "admin-portal", root: "admin-portal/backend", binary: "adminportal-service", port: "9013", command: "cmd", configDir: "config"},
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
			for _, forbidden := range []string{"COPY migrations", "dbmigrate", "DATABASE_PASSWORD", "TIDEWISE_DATABASE_URL"} {
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

func TestApplicationRootsAreCanonical(t *testing.T) {
	repoRoot := repositoryRoot()
	for _, path := range []string{
		"miniapp/frontend",
		"miniapp/backend",
		"admin-portal/frontend",
		"admin-portal/backend",
		"analyse-data-service/backend",
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

func TestLocalComposeOwnsOnlyThreeServicesAndDataStores(t *testing.T) {
	repoRoot := repositoryRoot()
	contents, err := os.ReadFile(filepath.Join(repoRoot, "infra", "local", "docker-compose.yaml"))
	if err != nil {
		t.Fatalf("read local compose: %v", err)
	}
	text := string(contents)
	for _, required := range []string{
		"  data:", "  miniapp:", "  adminportal:", "  postgres:", "  neo4j:",
		"context: ../..",
		"analyse-data-service/backend/Dockerfile", "miniapp/backend/Dockerfile", "admin-portal/backend/Dockerfile",
		"tidewise-local", "/healthz", "/readyz",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("local compose missing %q", required)
		}
	}
	for _, forbidden := range []string{"ingestion-scheduler", "source-ingest", "ingest-smoke"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("local compose revives retired runtime %q", forbidden)
		}
	}
	for _, bff := range []string{"miniapp", "adminportal"} {
		section := composeServiceSection(t, text, bff)
		for _, forbidden := range []string{"DATABASE_PASSWORD", "TIDEWISE_DATABASE_URL", "POSTGRES_", "NEO4J_"} {
			if strings.Contains(section, forbidden) {
				t.Fatalf("%s compose service carries Data credential %q", bff, forbidden)
			}
		}
	}
}

func TestCIConsumesThreeServiceOwnedImagesAndBoundaryContracts(t *testing.T) {
	repoRoot := repositoryRoot()
	contents, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read CI workflow: %v", err)
	}
	text := string(contents)
	for _, required := range []string{
		"go-version-file: go.mod",
		"cache-dependency-path: go.sum",
		"go test ./analyse-data-service/backend/api ./miniapp/backend/internal/data ./admin-portal/backend/dataclient",
		"go test ./scripts/ci/repository-contracts",
		"go build -o /tmp/data-service ./analyse-data-service/backend/cmd",
		"go build -o /tmp/miniapp-service ./miniapp/backend/cmd/server",
		"go build -o /tmp/adminportal-service ./admin-portal/backend/cmd",
		"-f analyse-data-service/backend/Dockerfile",
		"-f miniapp/backend/Dockerfile",
		"-f admin-portal/backend/Dockerfile",
		"docker compose --env-file infra/local/.env.example -f infra/local/docker-compose.yaml config --quiet",
		"docker compose --env-file infra/uat/.env.example -f infra/uat/docker-compose.yaml config --quiet",
		"bash scripts/ci/smoke-miniapp-data-compose.sh",
		"cache-dependency-path: package-lock.json",
		"npm run test",
		"npm run typecheck",
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

	smokeContents, err := os.ReadFile(filepath.Join(repoRoot, "scripts", "ci", "smoke-miniapp-data-compose.sh"))
	if err != nil {
		t.Fatalf("read Miniapp Compose smoke script: %v", err)
	}
	smoke := string(smokeContents)
	for _, required := range []string{
		"dbmigrate -apply",
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
	if end := strings.Index(remainder, "\n  "); end >= 0 {
		return remainder[:end]
	}
	return remainder
}
