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

type packageInfo struct {
	ImportPath string
	Imports    []string
}

func TestApplicationBackendUsesKratosServiceLayers(t *testing.T) {
	packages := listMiniappPackages(t)
	for _, suffix := range []string{
		"miniapp/backend/api/miniapp/v1",
		"miniapp/backend/cmd/server",
		"miniapp/backend/internal/biz",
		"miniapp/backend/internal/conf",
		"miniapp/backend/internal/data",
		"miniapp/backend/internal/server",
		"miniapp/backend/internal/service",
	} {
		if !hasPackageSuffix(packages, suffix) {
			t.Errorf("Miniapp Application Backend Service package %q is missing", suffix)
		}
	}
}

func TestKratosBinaryDoesNotDependOnGin(t *testing.T) {
	command := exec.Command("go", "list", "-deps", "./miniapp/backend/cmd/server")
	command.Dir = repositoryRoot()
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("list Miniapp binary dependencies: %v\n%s", err, output)
	}
	if strings.Contains(string(output), "github.com/gin-gonic/gin") {
		t.Fatal("Miniapp Kratos binary dependency closure must not include Gin")
	}
}

func TestDoesNotUseGoogleWireArtifacts(t *testing.T) {
	miniappRoot := filepath.Join(repositoryRoot(), "miniapp", "backend")
	for _, name := range []string{"wire.go", "wire_gen.go"} {
		var matches []string
		err := filepath.WalkDir(miniappRoot, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() && entry.Name() == name {
				matches = append(matches, path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan Miniapp for %s: %v", name, err)
		}
		if len(matches) != 0 {
			t.Fatalf("Miniapp must use explicit constructors, found Google Wire artifact names: %v", matches)
		}
	}
}

func TestKratosLayersFollowDependencyDirection(t *testing.T) {
	for _, pkg := range listMiniappPackages(t) {
		owner := localPackageName(pkg.ImportPath)
		switch {
		case owner == "miniapp/backend/internal/biz" || strings.HasPrefix(owner, "miniapp/backend/internal/biz/"):
			assertImportsExclude(t, pkg,
				"net/http",
				"github.com/go-kratos/kratos/v3/transport/http",
				"/miniapp/backend/internal/data",
				"/miniapp/backend/internal/service",
				"/miniapp/backend/internal/server",
			)
		case owner == "miniapp/backend/internal/service" || strings.HasPrefix(owner, "miniapp/backend/internal/service/"):
			assertImportsExclude(t, pkg,
				"/miniapp/backend/internal/data",
				"/miniapp/backend/internal/server",
			)
		case owner == "miniapp/backend/internal/server" || strings.HasPrefix(owner, "miniapp/backend/internal/server/"):
			assertImportsExclude(t, pkg,
				"/miniapp/backend/internal/biz",
				"/miniapp/backend/internal/data",
			)
		case owner == "miniapp/backend/internal/data" || strings.HasPrefix(owner, "miniapp/backend/internal/data/"):
			assertImportsExclude(t, pkg,
				"/miniapp/backend/api/",
				"/miniapp/backend/internal/service",
				"/miniapp/backend/internal/server",
			)
		case owner == "miniapp/backend/internal/conf" || strings.HasPrefix(owner, "miniapp/backend/internal/conf/"):
			assertImportsExclude(t, pkg,
				"/miniapp/backend/api/",
				"/miniapp/backend/internal/biz",
				"/miniapp/backend/internal/data",
				"/miniapp/backend/internal/service",
				"/miniapp/backend/internal/server",
			)
		}
	}
}

func TestLegacyRuntimeLayoutIsRemoved(t *testing.T) {
	miniappRoot := filepath.Join(repositoryRoot(), "miniapp", "backend")
	for _, legacy := range []string{
		"config",
		"dataclient",
		"usecase",
		"transport",
		"service.go",
		filepath.Join("cmd", "main.go"),
	} {
		path := filepath.Join(miniappRoot, legacy)
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("legacy Miniapp runtime path %q must be absent: %v", path, err)
		}
	}
}

func listMiniappPackages(t *testing.T) []packageInfo {
	t.Helper()
	command := exec.Command("go", "list", "-json", "./miniapp/backend/...")
	command.Dir = repositoryRoot()
	output, err := command.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Fatalf("go list Miniapp failed: %v\n%s", err, string(exitErr.Stderr))
		}
		t.Fatalf("go list Miniapp failed: %v", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(output))
	var packages []packageInfo
	for {
		var pkg packageInfo
		if err := decoder.Decode(&pkg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatalf("decode Miniapp package list: %v", err)
		}
		packages = append(packages, pkg)
	}
	return packages
}

func assertImportsExclude(t *testing.T, pkg packageInfo, forbidden ...string) {
	t.Helper()
	for _, imported := range pkg.Imports {
		for _, fragment := range forbidden {
			if imported == fragment || strings.Contains(imported, fragment) {
				t.Fatalf("%s must not import %s across its Kratos layer seam", pkg.ImportPath, imported)
			}
		}
	}
}

func hasPackageSuffix(packages []packageInfo, suffix string) bool {
	for _, pkg := range packages {
		if strings.HasSuffix(pkg.ImportPath, suffix) {
			return true
		}
	}
	return false
}

func localPackageName(importPath string) string {
	return strings.TrimPrefix(importPath, "github.com/meierlink88/tidewise-ai/")
}

func repositoryRoot() string {
	return filepath.Clean(filepath.Join("..", "..", "..", ".."))
}
