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
		"analyse-data-service/backend",
		"analyse-data-service/backend/cmd",
		"miniapp/backend/api/miniapp/v1",
		"miniapp/backend/cmd/server",
		"admin-portal/backend",
		"admin-portal/backend/cmd",
	} {
		if !hasPackageSuffix(packages, suffix) {
			t.Errorf("expected service-owned package %q to exist", suffix)
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
	services := map[string]string{
		"analyse-data-service/backend": "analyse-data-service",
		"miniapp/backend":              "miniapp",
		"admin-portal/backend":         "admin-portal",
	}
	for prefix, service := range services {
		if packageName == prefix || strings.HasPrefix(packageName, prefix+"/") {
			return service
		}
	}
	return ""
}

func TestRootCompatibilityCommandsAreRemoved(t *testing.T) {
	repoRoot := repositoryRoot()
	for _, legacy := range []string{"src", "services", "backend", "frontend", "internal"} {
		if _, err := os.Stat(filepath.Join(repoRoot, legacy)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("legacy root path %q must be absent after app-oriented migration: %v", legacy, err)
		}
	}
}

func TestRepositoryKeepsSingleRootGoModule(t *testing.T) {
	repoRoot := repositoryRoot()
	var modules []string
	err := filepath.WalkDir(repoRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".data", "node_modules", "vendor":
				return filepath.SkipDir
			}
		}
		if !entry.IsDir() && entry.Name() == "go.mod" {
			modules = append(modules, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repository modules: %v", err)
	}
	wantModule := filepath.Clean(filepath.Join(repoRoot, "go.mod"))
	if len(modules) != 1 || filepath.Clean(modules[0]) != wantModule {
		t.Fatalf("Go modules = %v, want only root go.mod", modules)
	}
}

func listServicePackages(t *testing.T) []packageInfo {
	t.Helper()

	command := exec.Command(
		"go", "list", "-json",
		"./analyse-data-service/backend/...",
		"./miniapp/backend/...",
		"./admin-portal/backend/...",
	)
	command.Dir = repositoryRoot()
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
