package architecture

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDataServiceUsesKratosApplicationLayout(t *testing.T) {
	backendRoot := filepath.Clean(filepath.Join("..", ".."))
	for _, path := range []string{
		"api/data/v1",
		"cmd/server",
		"configs",
		"internal/conf",
		"internal/biz/eventpublication",
		"internal/biz/researchthemeimport",
		"internal/biz/researchanchorimport",
		"internal/biz/research",
		"internal/biz/adminquery",
		"internal/biz/entityseed",
		"internal/data/postgres",
		"internal/data/dbmigration",
		"internal/data/eventpublication",
		"internal/data/researchthemeimport",
		"internal/data/researchanchorimport",
		"internal/data/research",
		"internal/data/adminquery",
		"internal/data/entityseed",
		"internal/service",
		"internal/server",
	} {
		if _, err := os.Stat(filepath.Join(backendRoot, path)); err != nil {
			t.Errorf("Data Kratos path %q is missing: %v", path, err)
		}
	}

	for _, path := range []string{
		"api/openapi.yaml",
		"api/document.go",
		"cmd/main.go",
		"config",
		"domain",
		"repositories",
		"adapters",
		"transport",
		"usecase",
		"service.go",
		"http_runtime.go",
	} {
		if _, err := os.Stat(filepath.Join(backendRoot, path)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("legacy Data runtime path %q still exists", path)
		}
	}
}

func TestDataServerBinaryUsesKratosWithoutGin(t *testing.T) {
	command := exec.Command("go", "list", "-deps", "./analyse-data-service/backend/cmd/server")
	command.Dir = repositoryRoot()
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("list Data binary dependencies: %v\n%s", err, output)
	}
	if strings.Contains(string(output), "github.com/gin-gonic/gin") {
		t.Fatal("Data Kratos binary dependency closure must not include Gin")
	}
	if !strings.Contains(string(output), "github.com/go-kratos/kratos/v3") {
		t.Fatal("Data Server binary must use Kratos v3")
	}
}

func TestDataDoesNotUseGoogleWireArtifacts(t *testing.T) {
	backendRoot := filepath.Clean(filepath.Join("..", ".."))
	for _, name := range []string{"wire.go", "wire_gen.go"} {
		var matches []string
		err := filepath.WalkDir(backendRoot, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !entry.IsDir() && entry.Name() == name {
				matches = append(matches, path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan Data for %s: %v", name, err)
		}
		if len(matches) != 0 {
			t.Fatalf("Data must use explicit constructors, found Google Wire artifact names: %v", matches)
		}
	}
}

func TestDataKratosLayersFollowDependencyDirection(t *testing.T) {
	backendRoot := filepath.Clean(filepath.Join("..", ".."))
	assertSourceImportsExclude(t, filepath.Join(backendRoot, "api"),
		"/analyse-data-service/backend/internal/",
	)
	assertSourceImportsExclude(t, filepath.Join(backendRoot, "internal", "biz"),
		"database/sql",
		"net/http",
		"/analyse-data-service/backend/internal/data",
		"/analyse-data-service/backend/internal/service",
		"/analyse-data-service/backend/internal/server",
		"github.com/go-kratos/kratos/v3/transport/http",
	)
	assertSourceImportsExclude(t, filepath.Join(backendRoot, "internal", "service"),
		"net/http",
		"github.com/go-kratos/kratos/v3/transport/http",
		"/analyse-data-service/backend/internal/data",
		"/analyse-data-service/backend/internal/server",
	)
	assertSourceImportsExclude(t, filepath.Join(backendRoot, "internal", "server"),
		"/analyse-data-service/backend/internal/biz",
		"/analyse-data-service/backend/internal/data",
	)
	assertSourceImportsExclude(t, filepath.Join(backendRoot, "internal", "data"),
		"/analyse-data-service/backend/api/",
		"/analyse-data-service/backend/internal/service",
		"/analyse-data-service/backend/internal/server",
	)
	assertSourceImportsExclude(t, filepath.Join(backendRoot, "internal", "conf"),
		"/analyse-data-service/backend/api/",
		"/analyse-data-service/backend/internal/biz",
		"/analyse-data-service/backend/internal/data",
		"/analyse-data-service/backend/internal/service",
		"/analyse-data-service/backend/internal/server",
	)
}

func assertSourceImportsExclude(t *testing.T, root string, forbidden ...string) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			for _, fragment := range forbidden {
				if strings.Contains(line, fragment) {
					t.Errorf("%s imports forbidden outer layer %q", path, fragment)
				}
			}
		}
		return scanner.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
}

func repositoryRoot() string {
	return filepath.Clean(filepath.Join("..", "..", "..", ".."))
}
