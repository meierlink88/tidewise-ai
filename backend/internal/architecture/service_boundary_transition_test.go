package architecture

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestTransitionalBFFDataDependencyAllowlist(t *testing.T) {
	packages := append(listInternalPackages(t), listCommandPackages(t)...)
	expected := map[string][]string{
		"internal/apps/miniappapi": {"internal/domain", "internal/repositories"},
		"internal/apps/adminapi":   {"internal/domain", "internal/repositories"},
		"cmd/api":                  {"internal/platform/database", "internal/platform/dbmigration", "internal/repositories"},
		"cmd/admin-api":            {"internal/platform/database", "internal/platform/dbmigration", "internal/repositories"},
	}

	for owner, wanted := range expected {
		pkg := packageWithSuffix(t, packages, "/"+owner)
		var actual []string
		for _, imported := range pkg.Imports {
			for _, boundary := range []string{
				"/internal/domain",
				"/internal/repositories",
				"/internal/platform/database",
				"/internal/platform/dbmigration",
			} {
				if strings.HasSuffix(imported, boundary) {
					actual = append(actual, strings.TrimPrefix(boundary, "/"))
				}
			}
		}
		sort.Strings(actual)
		sort.Strings(wanted)
		if !reflect.DeepEqual(actual, wanted) {
			t.Fatalf("transitional Data dependency manifest for %s changed: got %v, want %v; do not add callers, and remove this allowlist as Packages 5-6 decouple the BFF", owner, actual, wanted)
		}

		for _, forbidden := range []string{
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

func TestLegacyIngestionPreRetirementImportAllowlist(t *testing.T) {
	packages := append(listInternalPackages(t), listCommandPackages(t)...)
	assertImportersEqual(t, packages, "/internal/apps/ingestion/runtime", []string{
		"cmd/ingest-smoke",
		"cmd/ingestion-scheduler",
		"cmd/source-ingest",
		"internal/apps/ingestion/scheduler",
	})
	assertImportersEqual(t, packages, "/internal/apps/ingestion/scheduler", []string{
		"cmd/ingestion-scheduler",
	})

	backendRoot := filepath.Join("..", "..")
	for _, path := range []string{
		"cmd/ingestion-scheduler",
		"cmd/source-ingest",
		"cmd/ingest-smoke",
		"internal/apps/ingestion/scheduler",
		"internal/apps/ingestion/runtime",
		"internal/apps/ingestion/health",
	} {
		if _, err := os.Stat(filepath.Join(backendRoot, path)); err != nil {
			t.Fatalf("pre-retirement path %q changed before Package 7 manifest verification: %v", path, err)
		}
	}

	assertFileContainsAll(t, filepath.Join(backendRoot, "internal", "config", "config.go"), []string{
		"type IngestionConfig struct",
		"yaml:\"ingestion\"",
		"SchedulerTickSeconds",
		"SchedulerTimezone",
	})
	for _, environment := range []string{"local", "uat", "prod"} {
		assertFileContainsAll(t, filepath.Join(backendRoot, "config", "config."+environment+".yaml"), []string{
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

func assertImportersEqual(t *testing.T, packages []packageInfo, importedSuffix string, expected []string) {
	t.Helper()
	var actual []string
	for _, pkg := range packages {
		for _, imported := range pkg.Imports {
			if strings.HasSuffix(imported, importedSuffix) {
				actual = append(actual, localPackageName(pkg.ImportPath))
				break
			}
		}
	}
	sort.Strings(actual)
	sort.Strings(expected)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("pre-retirement importers of %s changed: got %v, want %v", importedSuffix, actual, expected)
	}
}

func localPackageName(importPath string) string {
	const marker = "/backend/"
	if index := strings.Index(importPath, marker); index >= 0 {
		return importPath[index+len(marker):]
	}
	return importPath
}

func assertFileContainsAll(t *testing.T, path string, needles []string) {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for _, needle := range needles {
		if !strings.Contains(string(contents), needle) {
			t.Fatalf("%s must contain %q until Package 7 replaces the pre-retirement manifest with absence checks", path, needle)
		}
	}
}
