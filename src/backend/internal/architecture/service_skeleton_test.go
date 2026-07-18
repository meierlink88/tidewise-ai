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

func TestMiniappApplicationBackendOwnsUseCaseAndTransport(t *testing.T) {
	packages := listServicePackages(t)
	for _, suffix := range []string{
		"services/miniapp/usecase",
		"services/miniapp/transport",
		"services/miniapp/config",
	} {
		if !hasPackageSuffix(packages, suffix) {
			t.Errorf("Miniapp Application Backend Service package %q is missing", suffix)
		}
	}
}

func TestAdminPortalApplicationBackendOwnsUseCaseAndTransport(t *testing.T) {
	packages := listServicePackages(t)
	for _, suffix := range []string{
		"services/adminportal/usecase",
		"services/adminportal/transport",
		"services/adminportal/config",
	} {
		if !hasPackageSuffix(packages, suffix) {
			t.Errorf("Admin Portal Application Backend Service package %q is missing", suffix)
		}
	}
}

func TestDataDomainServiceOwnsBusinessAndAdapters(t *testing.T) {
	packages := listServicePackages(t)
	for _, suffix := range []string{
		"services/data/usecase/adminquery",
		"services/data/usecase/eventimport",
		"services/data/usecase/rawimport",
		"services/data/usecase/research",
		"services/data/usecase/sourcecatalog",
		"services/data/usecase/sourcemetadata",
		"services/data/domain",
		"services/data/repositories",
		"services/data/adapters/database",
		"services/data/adapters/dbmigration",
		"services/data/adapters/graphdb",
		"services/data/transport/internalapi",
		"services/data/config",
	} {
		if !hasPackageSuffix(packages, suffix) {
			t.Errorf("Data Domain Service package %q is missing", suffix)
		}
	}
}

func TestDeployableServicesDoNotImportEachOther(t *testing.T) {
	packages := listServicePackages(t)
	for _, pkg := range packages {
		owner := localPackageName(pkg.ImportPath)
		ownerService := deployableService(owner)
		if ownerService == "" {
			continue
		}
		for _, imported := range pkg.Imports {
			importedService := deployableService(localPackageName(imported))
			if importedService != "" && importedService != ownerService {
				t.Fatalf("%s must not import implementation from %s", pkg.ImportPath, imported)
			}
		}
	}
}

func deployableService(packageName string) string {
	for _, service := range []string{"data", "miniapp", "adminportal"} {
		prefix := "services/" + service
		if packageName == prefix || strings.HasPrefix(packageName, prefix+"/") {
			return service
		}
	}
	return ""
}

func TestRootCompatibilityCommandsAreRemoved(t *testing.T) {
	backendRoot := filepath.Join("..", "..")
	if _, err := os.Stat(filepath.Join(backendRoot, "cmd")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("src/backend/cmd must be absent after commands move to their owning services: %v", err)
	}
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
		t.Fatalf("backend modules = %v, want only src/backend/go.mod", modules)
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
