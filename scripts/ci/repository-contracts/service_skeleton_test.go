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
		"data-service/backend/api/data/v1",
		"data-service/backend/cmd/server",
		"data-service/backend/internal/conf",
		"data-service/backend/internal/server",
		"miniapp/backend/api/miniapp/v1",
		"miniapp/backend/cmd/server",
		"miniapp/backend/internal/biz",
		"miniapp/backend/internal/conf",
		"miniapp/backend/internal/data",
		"miniapp/backend/internal/server",
		"miniapp/backend/internal/service",
		"admin-portal/backend/api/admin/v1",
		"admin-portal/backend/cmd/server",
		"admin-portal/backend/internal/biz",
		"admin-portal/backend/internal/conf",
		"admin-portal/backend/internal/data",
		"admin-portal/backend/internal/server",
		"admin-portal/backend/internal/service",
	} {
		if !hasPackageSuffix(packages, suffix) {
			t.Errorf("expected service-owned package %q to exist", suffix)
		}
	}

	for _, service := range []string{
		"data-service/backend",
		"miniapp/backend",
		"admin-portal/backend",
	} {
		for _, layer := range []string{"conf", "biz", "data", "service", "server"} {
			path := filepath.Join(repositoryRoot(), service, "internal", layer)
			if info, err := os.Stat(path); err != nil || !info.IsDir() {
				t.Errorf("expected Kratos layer directory %q to exist: %v", path, err)
			}
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

func TestCurrentServiceBinariesExcludeAgentRuntimeDependencies(t *testing.T) {
	commands := map[string]string{
		"data":         "./data-service/backend/cmd/server",
		"miniapp":      "./miniapp/backend/cmd/server",
		"admin-portal": "./admin-portal/backend/cmd/server",
	}
	for name, command := range commands {
		dependencies := listCommandDependencies(t, command)
		for _, dependency := range dependencies {
			if strings.HasPrefix(dependency, "github.com/cloudwego/eino") {
				t.Fatalf("%s binary unexpectedly includes Eino dependency %q", name, dependency)
			}
			if dependency == "github.com/gin-gonic/gin" {
				t.Fatalf("%s binary unexpectedly includes Gin", name)
			}
		}
	}
}

func TestDataSemanticProjectionRemainsRetired(t *testing.T) {
	root := repositoryRoot()
	if _, err := os.Stat(filepath.Join(root, "data-service", "backend", "internal", "data", "semanticprojection")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Data semantic projection adapter must be retired: %v", err)
	}
}

func TestKratosLayersFollowOneCentralDependencyPolicy(t *testing.T) {
	packages := listServicePackages(t)
	for _, pkg := range packages {
		local := localPackageName(pkg.ImportPath)
		service := deployableService(local)
		if service == "" {
			continue
		}
		root := serviceRoot(service)
		for _, imported := range pkg.Imports {
			switch {
			case strings.Contains(local, "/api/"):
				assertImportOutside(t, local, imported, root+"/internal/")
			case strings.Contains(local, "/internal/biz"):
				assertImportOutside(
					t,
					local,
					imported,
					root+"/internal/data",
					root+"/internal/service",
					root+"/internal/server",
					"database/sql",
					"net/http",
					"github.com/go-kratos/kratos/v3/transport/http",
				)
			case strings.Contains(local, "/internal/data"):
				assertImportOutside(
					t,
					local,
					imported,
					root+"/api/",
					root+"/internal/service",
					root+"/internal/server",
				)
			case strings.Contains(local, "/internal/service"):
				assertImportOutside(
					t,
					local,
					imported,
					root+"/internal/data",
					root+"/internal/server",
				)
			case strings.Contains(local, "/internal/server"):
				assertImportOutside(t, local, imported, root+"/internal/biz", root+"/internal/data")
			case strings.Contains(local, "/internal/conf"):
				assertImportOutside(
					t,
					local,
					imported,
					root+"/api/",
					root+"/internal/biz",
					root+"/internal/data",
					root+"/internal/service",
					root+"/internal/server",
				)
			}
		}
	}
}

func TestDeployableServicesUseExplicitConstructionWithoutWire(t *testing.T) {
	root := repositoryRoot()
	for _, service := range []string{
		"data-service/backend",
		"miniapp/backend",
		"admin-portal/backend",
	} {
		err := filepath.WalkDir(filepath.Join(root, service), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() && (entry.Name() == "wire.go" || entry.Name() == "wire_gen.go") {
				t.Errorf("%s must use explicit constructors; found %s", service, path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func assertImportOutside(t *testing.T, owner, imported string, forbidden ...string) {
	t.Helper()
	imported = localPackageName(imported)
	for _, prefix := range forbidden {
		if imported == prefix || strings.HasPrefix(imported, prefix+"/") {
			t.Fatalf("%s imports forbidden dependency %s", owner, imported)
		}
	}
}

func serviceRoot(service string) string {
	switch service {
	case "data-service":
		return "data-service/backend"
	case "miniapp":
		return "miniapp/backend"
	case "admin-portal":
		return "admin-portal/backend"
	default:
		return ""
	}
}

func deployableService(packageName string) string {
	services := map[string]string{
		"data-service/backend": "data-service",
		"miniapp/backend":      "miniapp",
		"admin-portal/backend": "admin-portal",
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
			case ".git", ".data", ".reference", "node_modules", "vendor":
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
		"./data-service/backend/...",
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

func listCommandDependencies(t *testing.T, commandPath string) []string {
	t.Helper()
	command := exec.Command("go", "list", "-deps", "-f", "{{.ImportPath}}", commandPath)
	command.Dir = repositoryRoot()
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go list dependencies for %s failed: %v\n%s", commandPath, err, output)
	}
	return strings.Fields(string(output))
}
