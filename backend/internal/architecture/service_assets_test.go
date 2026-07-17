package architecture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceOwnedDockerAssetsReplaceLegacyBackendImage(t *testing.T) {
	repoRoot := filepath.Join("..", "..", "..")
	assets := []struct {
		service    string
		binary     string
		mustCopyDB bool
	}{
		{service: "data", binary: "data-service", mustCopyDB: true},
		{service: "miniapp", binary: "miniapp-service"},
		{service: "adminportal", binary: "adminportal-service"},
	}

	for _, asset := range assets {
		path := filepath.Join(repoRoot, "backend", "services", asset.service, "Dockerfile")
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s Dockerfile: %v", asset.service, err)
		}
		text := string(contents)
		for _, required := range []string{
			"./services/" + asset.service + "/cmd",
			"/usr/local/bin/" + asset.binary,
			"/healthz",
			"/readyz",
			"CMD [\"/usr/local/bin/" + asset.binary + "\"]",
		} {
			if !strings.Contains(text, required) {
				t.Fatalf("%s Dockerfile missing %q", asset.service, required)
			}
		}
		if asset.mustCopyDB && !strings.Contains(text, "COPY migrations") {
			t.Fatal("Data Dockerfile must own migration assets")
		}
		if !strings.Contains(text, "COPY services/"+asset.service+"/config ./config") {
			t.Fatalf("%s Dockerfile must copy its service-owned start config", asset.service)
		}
		configPath := filepath.Join(repoRoot, "backend", "services", asset.service, "config", "config.local.yaml")
		configContents, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("read %s start config: %v", asset.service, err)
		}
		configText := string(configContents)
		for _, required := range []string{"host: 0.0.0.0", "port: 8080"} {
			if !strings.Contains(configText, required) {
				t.Fatalf("%s start config missing %q", asset.service, required)
			}
		}
		if !asset.mustCopyDB {
			for _, forbidden := range []string{"COPY migrations", "dbmigrate", "DATABASE_PASSWORD", "TIDEWISE_DATABASE_URL"} {
				if strings.Contains(text, forbidden) {
					t.Fatalf("%s Dockerfile carries Data DB concern %q", asset.service, forbidden)
				}
				if strings.Contains(configText, forbidden) || strings.Contains(configText, "database:") || strings.Contains(configText, "migration:") {
					t.Fatalf("%s start config carries Data DB concern %q", asset.service, forbidden)
				}
			}
		}
	}

	if _, err := os.Stat(filepath.Join(repoRoot, "backend", "Dockerfile")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy backend/Dockerfile must be removed after all local/CI consumers switch: %v", err)
	}
}

func TestLocalComposeOwnsOnlyThreeServicesAndDataStores(t *testing.T) {
	repoRoot := filepath.Join("..", "..", "..")
	contents, err := os.ReadFile(filepath.Join(repoRoot, "infra", "local", "docker-compose.yaml"))
	if err != nil {
		t.Fatalf("read local compose: %v", err)
	}
	text := string(contents)
	for _, required := range []string{
		"  data:", "  miniapp:", "  adminportal:", "  postgres:", "  neo4j:",
		"services/data/Dockerfile", "services/miniapp/Dockerfile", "services/adminportal/Dockerfile",
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
	repoRoot := filepath.Join("..", "..", "..")
	contents, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read CI workflow: %v", err)
	}
	text := string(contents)
	for _, required := range []string{
		"go test ./services/data/api ./services/miniapp/dataclient ./services/adminportal/dataclient",
		"go test ./internal/architecture",
		"go build -o /tmp/data-service ./services/data/cmd",
		"go build -o /tmp/miniapp-service ./services/miniapp/cmd",
		"go build -o /tmp/adminportal-service ./services/adminportal/cmd",
		"-f services/data/Dockerfile",
		"-f services/miniapp/Dockerfile",
		"-f services/adminportal/Dockerfile",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("CI workflow missing %q", required)
		}
	}
	if strings.Contains(text, "-f Dockerfile") || strings.Contains(text, "backend/Dockerfile") {
		t.Fatal("CI must not consume the legacy backend Dockerfile")
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
