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
		"analyse-data-service/backend/api/data/v1",
		"analyse-data-service/backend/cmd/server",
		"analyse-data-service/backend/internal/conf",
		"analyse-data-service/backend/internal/server",
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
		"agent-run/backend/api/agentrun/v1",
		"agent-run/backend/cmd/server",
		"agent-run/backend/cmd/migrate",
		"agent-run/backend/cmd/config",
		"agent-run/backend/cmd/artifacts",
		"agent-run/backend/internal/conf",
		"agent-run/backend/internal/server",
		"agent-run/backend/internal/service",
	} {
		if !hasPackageSuffix(packages, suffix) {
			t.Errorf("expected service-owned package %q to exist", suffix)
		}
	}

	for _, service := range []string{
		"analyse-data-service/backend",
		"miniapp/backend",
		"admin-portal/backend",
		"agent-run/backend",
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

func TestEinoDependenciesStayInAgentRunBinaryClosure(t *testing.T) {
	commands := map[string]string{
		"data":         "./analyse-data-service/backend/cmd/server",
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

	agentRunDependencies := listCommandDependencies(t, "./agent-run/backend/cmd/server")
	for _, required := range []string{
		"github.com/cloudwego/eino",
		"github.com/cloudwego/eino-ext/components/embedding/openai",
		"github.com/cloudwego/eino-ext/components/model/openai",
	} {
		found := false
		for _, dependency := range agentRunDependencies {
			if dependency == required || strings.HasPrefix(dependency, required+"/") {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("AgentRun binary closure is missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"github.com/cohesion-org/deepseek-go",
		"github.com/gin-gonic/gin",
		"github.com/ollama/ollama",
	} {
		for _, dependency := range agentRunDependencies {
			if dependency == forbidden || strings.HasPrefix(dependency, forbidden+"/") {
				t.Fatalf("AgentRun binary unexpectedly includes vulnerable dependency %q", dependency)
			}
		}
	}
}

func TestEventSemanticEmbeddingOwnership(t *testing.T) {
	root := repositoryRoot()
	agentAdapter := string(readContractFile(t, filepath.Join(
		root, "agent-run", "backend", "internal", "data", "semanticretrieval", "client.go",
	)))
	for _, forbidden := range []string{"/embeddings", "encoding_format", `json:"embedding"`} {
		if strings.Contains(agentAdapter, forbidden) {
			t.Fatalf("AgentRun Qdrant adapter reimplements embedding wire contract %q", forbidden)
		}
	}
	if !strings.Contains(agentAdapter, "embedding.Embedder") ||
		!strings.Contains(agentAdapter, ".EmbedStrings(ctx, texts)") {
		t.Fatal("AgentRun Qdrant adapter must batch through an injected Eino embedding.Embedder")
	}

	if _, err := os.Stat(filepath.Join(root, "analyse-data-service", "backend", "internal", "data", "semanticprojection")); !errors.Is(err, os.ErrNotExist) {
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
			if service == "agent-run" &&
				strings.HasPrefix(imported, "github.com/cloudwego/eino-ext/") &&
				!strings.Contains(local, "/internal/data/modelprovider/") {
				t.Fatalf("%s imports concrete Eino provider %s outside internal/data/modelprovider", local, imported)
			}
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
		"analyse-data-service/backend",
		"miniapp/backend",
		"admin-portal/backend",
		"agent-run/backend",
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
	case "analyse-data-service":
		return "analyse-data-service/backend"
	case "miniapp":
		return "miniapp/backend"
	case "admin-portal":
		return "admin-portal/backend"
	case "agent-run":
		return "agent-run/backend"
	default:
		return ""
	}
}

func deployableService(packageName string) string {
	services := map[string]string{
		"analyse-data-service/backend": "analyse-data-service",
		"miniapp/backend":              "miniapp",
		"admin-portal/backend":         "admin-portal",
		"agent-run/backend":            "agent-run",
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
		"./analyse-data-service/backend/...",
		"./miniapp/backend/...",
		"./admin-portal/backend/...",
		"./agent-run/backend/...",
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

func TestAgentRunStandaloneRepositoryAssetsAreConverged(t *testing.T) {
	repoRoot := repositoryRoot()
	for _, standalone := range []string{
		"agent-run/backend/go.mod",
		"agent-run/backend/go.sum",
		"agent-run/backend/.github",
		"agent-run/backend/.codex",
		"agent-run/backend/AGENTS.md",
		"agent-run/backend/CONTEXT.md",
		"agent-run/backend/docs",
	} {
		if _, err := os.Stat(filepath.Join(repoRoot, standalone)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("standalone AgentRun asset %q must be converged into the monorepo: %v", standalone, err)
		}
	}
	for _, required := range []string{
		"docs/contexts/agentrun/CONTEXT.md",
		"docs/contexts/agentrun/adr/0001-limit-plaintext-provider-credentials-to-development.md",
		"docs/contexts/agentrun/adr/0002-use-kratos-as-service-shell-and-eino-inside-agent-capabilities.md",
		"docs/architecture/agentrun/collector-agent-v1-platform-foundation.md",
		"docs/research/agentrun/agent-schedule-go-library-options.md",
	} {
		if _, err := os.Stat(filepath.Join(repoRoot, required)); err != nil {
			t.Errorf("converged AgentRun asset %q is missing: %v", required, err)
		}
	}
}
