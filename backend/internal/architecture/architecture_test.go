package architecture

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"
)

type packageInfo struct {
	ImportPath string
	Imports    []string
}

func TestBackendSubsystemPackagesExist(t *testing.T) {
	packages := listInternalPackages(t)
	expected := []string{
		"internal/apps/miniappapi",
		"internal/apps/adminapi",
		"internal/apps/entityfoundation",
		"internal/apps/entityfoundation/seed",
		"internal/apps/ingestion",
		"internal/apps/ingestion/core",
		"internal/apps/ingestion/scheduler",
		"internal/apps/ingestion/runtime",
		"internal/apps/ingestion/sourcecatalog",
		"internal/apps/ingestion/connectors",
		"internal/apps/ingestion/parsers",
		"internal/apps/ingestion/health",
		"internal/apps/graphprojection",
		"internal/platform",
		"internal/platform/database",
		"internal/platform/dbmigration",
		"internal/platform/graphdb",
	}

	for _, suffix := range expected {
		if !hasPackageSuffix(packages, suffix) {
			t.Fatalf("expected backend package %q to exist", suffix)
		}
	}
}

func TestLegacyIngestionCompatibilityPackagesAreRemoved(t *testing.T) {
	packages := listInternalPackages(t)
	legacy := []string{
		"internal/jobs",
		"internal/sourcecatalog",
		"internal/integrations",
		"internal/ingestion",
		"internal/entityseed",
		"internal/migrations",
		"internal/database",
	}

	for _, suffix := range legacy {
		if hasPackageSuffix(packages, suffix) {
			t.Fatalf("legacy backend package %q must be removed after ingestion subsystem migration", suffix)
		}
	}
}

func TestBackendForbiddenDependencies(t *testing.T) {
	packages := listInternalPackages(t)

	for _, pkg := range packages {
		switch {
		case containsPath(pkg.ImportPath, "/internal/integrations"):
			assertNoImport(t, pkg, "/internal/apps/")
		case containsPath(pkg.ImportPath, "/internal/apps/miniappapi"),
			containsPath(pkg.ImportPath, "/internal/apps/adminapi"):
			assertNoImport(t, pkg, "/internal/apps/ingestion/connectors")
			assertNoImport(t, pkg, "/internal/platform/graphdb")
		case containsPath(pkg.ImportPath, "/internal/apps/ingestion"):
			assertNoImport(t, pkg, "/internal/platform/graphdb")
		case containsPath(pkg.ImportPath, "/internal/apps/ingestion/connectors"):
			assertNoImport(t, pkg, "/internal/repositories")
			assertNoImport(t, pkg, "/internal/platform/database")
		case containsPath(pkg.ImportPath, "/internal/apps/ingestion/parsers"):
			assertNoImport(t, pkg, "/internal/repositories")
			assertNoImport(t, pkg, "/internal/platform/database")
			assertNoStdlibImport(t, pkg, "os")
			assertNoStdlibImport(t, pkg, "net/http")
		case containsPath(pkg.ImportPath, "/internal/platform"):
			assertNoImport(t, pkg, "/internal/apps/")
		}
	}
}

func listInternalPackages(t *testing.T) []packageInfo {
	t.Helper()

	command := exec.Command("go", "list", "-json", "./internal/...")
	command.Dir = "../.."
	output, err := command.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Fatalf("go list failed: %v\n%s", err, string(exitErr.Stderr))
		}
		t.Fatalf("go list failed: %v", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(output))
	var packages []packageInfo
	for {
		var pkg packageInfo
		if err := decoder.Decode(&pkg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatalf("decode go list output: %v", err)
		}
		packages = append(packages, pkg)
	}
	return packages
}

func hasPackageSuffix(packages []packageInfo, suffix string) bool {
	for _, pkg := range packages {
		if strings.HasSuffix(pkg.ImportPath, suffix) {
			return true
		}
	}
	return false
}

func assertNoImport(t *testing.T, pkg packageInfo, forbidden string) {
	t.Helper()

	for _, imported := range pkg.Imports {
		if containsPath(imported, forbidden) {
			t.Fatalf("%s must not import %s", pkg.ImportPath, imported)
		}
	}
}

func assertNoStdlibImport(t *testing.T, pkg packageInfo, forbidden string) {
	t.Helper()

	for _, imported := range pkg.Imports {
		if imported == forbidden {
			t.Fatalf("%s must not import %s", pkg.ImportPath, imported)
		}
	}
}

func containsPath(importPath string, fragment string) bool {
	return strings.Contains(importPath, fragment)
}
