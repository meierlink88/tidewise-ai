package architecture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTransitionalBFFDataDependencyAllowlist(t *testing.T) {
	packages := append(listInternalPackages(t), listCommandPackages(t)...)
	for _, owner := range []string{"internal/apps/miniappapi", "cmd/api", "internal/apps/adminapi", "cmd/admin-api"} {
		pkg := packageWithSuffix(t, packages, "/"+owner)
		for _, forbidden := range []string{
			"/internal/domain",
			"/internal/repositories",
			"/internal/platform/database",
			"/internal/platform/dbmigration",
			"/internal/apps/ingestion/connectors",
			"/internal/apps/ingestion/parsers",
			"/internal/apps/ingestion/runtime",
			"/internal/apps/ingestion/scheduler",
			"/internal/platform/graphdb",
		} {
			assertNoImport(t, pkg, forbidden)
		}
	}
}

func TestRetiredIngestionRuntimeIsAbsent(t *testing.T) {
	packages := append(listInternalPackages(t), listCommandPackages(t)...)
	for _, suffix := range []string{
		"/cmd/ingest-smoke",
		"/cmd/ingestion-scheduler",
		"/cmd/source-ingest",
		"/internal/apps/ingestion/health",
		"/internal/apps/ingestion/runtime",
		"/internal/apps/ingestion/scheduler",
	} {
		if hasPackageSuffix(packages, suffix) {
			t.Fatalf("retired ingestion package %q must be absent after Package 8", suffix)
		}
	}

	backendRoot := filepath.Join("..", "..")
	for _, path := range []string{
		"cmd/ingestion-scheduler",
		"cmd/source-ingest",
		"cmd/ingest-smoke",
		"internal/apps/ingestion/scheduler",
		"internal/apps/ingestion/runtime",
		"internal/apps/ingestion/health",
	} {
		if _, err := os.Stat(filepath.Join(backendRoot, path)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("retired ingestion path %q must be absent after Package 8: %v", path, err)
		}
	}

	assertFileContainsNone(t, filepath.Join(backendRoot, "internal", "config", "config.go"), []string{
		"type IngestionConfig struct",
		"yaml:\"ingestion\"",
		"SchedulerTickSeconds",
		"SchedulerTimezone",
	})
	for _, environment := range []string{"local", "uat", "prod"} {
		assertFileContainsNone(t, filepath.Join(backendRoot, "config", "config."+environment+".yaml"), []string{
			"ingestion:",
			"scheduler_tick_seconds:",
			"scheduler_timezone:",
		})
	}
}

func packageWithSuffix(t *testing.T, packages []packageInfo, suffix string) packageInfo {
	t.Helper()
	for _, pkg := range packages {
		if strings.HasSuffix(pkg.ImportPath, suffix) {
			return pkg
		}
	}
	t.Fatalf("package with suffix %q is missing", suffix)
	return packageInfo{}
}

func localPackageName(importPath string) string {
	const marker = "/backend/"
	if index := strings.Index(importPath, marker); index >= 0 {
		return importPath[index+len(marker):]
	}
	return importPath
}

func assertFileContainsNone(t *testing.T, path string, needles []string) {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for _, needle := range needles {
		if strings.Contains(string(contents), needle) {
			t.Fatalf("%s must not contain retired ingestion configuration %q after Package 8", path, needle)
		}
	}
}
