package architecture

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceOwnedPackagesAndCommandsExist(t *testing.T) {
	packages := listServicePackages(t)
	for _, suffix := range []string{
		"services/data",
		"services/data/cmd",
		"services/miniapp",
		"services/miniapp/cmd",
		"services/adminportal",
		"services/adminportal/cmd",
	} {
		if !hasPackageSuffix(packages, suffix) {
			t.Errorf("expected service-owned package %q to exist", suffix)
		}
	}
}

func TestServiceOwnedBFFPackagesDoNotImportDataImplementation(t *testing.T) {
	packages := listServicePackages(t)
	for _, pkg := range packages {
		owner := localPackageName(pkg.ImportPath)
		if !strings.HasPrefix(owner, "services/miniapp") && !strings.HasPrefix(owner, "services/adminportal") {
			continue
		}
		for _, forbidden := range []string{
			"/services/data",
			"/internal/domain",
			"/internal/repositories",
			"/internal/platform/database",
			"/internal/platform/dbmigration",
			"/internal/platform/graphdb",
			"/internal/apps/ingestion/connectors",
			"/internal/apps/ingestion/parsers",
		} {
			assertNoImport(t, pkg, forbidden)
		}
	}
}

func TestLegacyHTTPCommandsUseServiceOwnedCompatibilityFacade(t *testing.T) {
	packages := listCommandPackages(t)
	assertPackageImports(t, packageWithSuffix(t, packages, "/cmd/api"), "/services/miniapp")
	assertPackageImports(t, packageWithSuffix(t, packages, "/cmd/admin-api"), "/services/adminportal")
}

func TestServiceSkeletonKeepsSingleBackendModule(t *testing.T) {
	backendRoot := filepath.Join("..", "..")
	var modules []string
	err := filepath.WalkDir(backendRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && entry.Name() == "vendor" {
			return filepath.SkipDir
		}
		if !entry.IsDir() && entry.Name() == "go.mod" {
			modules = append(modules, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk backend modules: %v", err)
	}
	wantModule := filepath.Clean(filepath.Join(backendRoot, "go.mod"))
	if len(modules) != 1 || filepath.Clean(modules[0]) != wantModule {
		t.Fatalf("backend modules = %v, want only backend/go.mod", modules)
	}
}

func listServicePackages(t *testing.T) []packageInfo {
	t.Helper()

	command := exec.Command("go", "list", "-json", "./services/...")
	command.Dir = "../.."
	output, err := command.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Fatalf("go list services failed: %v\n%s", err, string(exitErr.Stderr))
		}
		t.Fatalf("go list services failed: %v", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(output))
	var packages []packageInfo
	for {
		var pkg packageInfo
		if err := decoder.Decode(&pkg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatalf("decode service package list: %v", err)
		}
		packages = append(packages, pkg)
	}
	return packages
}

func assertPackageImports(t *testing.T, pkg packageInfo, wanted string) {
	t.Helper()
	for _, imported := range pkg.Imports {
		if strings.HasSuffix(imported, wanted) {
			return
		}
	}
	t.Fatalf("%s must import service-owned compatibility facade %s", pkg.ImportPath, wanted)
}
