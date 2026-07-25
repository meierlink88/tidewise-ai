package architecture

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdminPortalUsesKratosApplicationLayout(t *testing.T) {
	backendRoot := filepath.Clean(filepath.Join("..", ".."))
	for _, path := range []string{
		"api/admin/v1",
		"cmd/server",
		"configs",
		"internal/conf",
		"internal/biz",
		"internal/data",
		"internal/service",
		"internal/server",
	} {
		if _, err := os.Stat(filepath.Join(backendRoot, path)); err != nil {
			t.Errorf("Admin Portal Kratos path %q is missing: %v", path, err)
		}
	}

	for _, path := range []string{
		"api/openapi.yaml",
		"api/document.go",
		"cmd/main.go",
		"config",
		"dataclient",
		"agentrunclient",
		"transport",
		"usecase",
		"service.go",
		"http_runtime.go",
	} {
		if _, err := os.Stat(filepath.Join(backendRoot, path)); !os.IsNotExist(err) {
			t.Errorf("legacy Admin Portal path %q still exists", path)
		}
	}
}

func TestAdminPortalBinaryDoesNotDependOnGin(t *testing.T) {
	command := exec.Command("go", "list", "-deps", "./admin-portal/backend/cmd/server")
	command.Dir = filepath.Clean(filepath.Join("..", "..", "..", ".."))
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("list Admin Portal binary dependencies: %v\n%s", err, output)
	}
	if strings.Contains(string(output), "github.com/gin-gonic/gin") {
		t.Fatal("Admin Portal Kratos binary dependency closure must not include Gin")
	}
}

func TestAdminPortalDoesNotUseGoogleWireArtifacts(t *testing.T) {
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
			t.Fatalf("scan Admin Portal for %s: %v", name, err)
		}
		if len(matches) != 0 {
			t.Fatalf("Admin Portal must use explicit constructors, found Google Wire artifacts: %v", matches)
		}
	}
}

func TestAdminPortalBizDoesNotDependOnOuterLayers(t *testing.T) {
	backendRoot := filepath.Clean(filepath.Join("..", ".."))
	assertSourceImportsExclude(t, filepath.Join(backendRoot, "internal", "biz"),
		"net/http",
		"/admin-portal/backend/internal/data",
		"/admin-portal/backend/internal/service",
		"/admin-portal/backend/internal/server",
		"github.com/go-kratos/kratos/v3/transport/http",
	)
}

func TestAdminPortalKratosLayersFollowDependencyDirection(t *testing.T) {
	backendRoot := filepath.Clean(filepath.Join("..", ".."))
	assertSourceImportsExclude(t, filepath.Join(backendRoot, "internal", "service"),
		"/admin-portal/backend/internal/data",
		"/admin-portal/backend/internal/server",
	)
	assertSourceImportsExclude(t, filepath.Join(backendRoot, "internal", "server"),
		"/admin-portal/backend/internal/biz",
		"/admin-portal/backend/internal/data",
	)
	assertSourceImportsExclude(t, filepath.Join(backendRoot, "internal", "data"),
		"/admin-portal/backend/api/",
		"/admin-portal/backend/internal/service",
		"/admin-portal/backend/internal/server",
	)
	assertSourceImportsExclude(t, filepath.Join(backendRoot, "internal", "conf"),
		"/admin-portal/backend/api/",
		"/admin-portal/backend/internal/biz",
		"/admin-portal/backend/internal/data",
		"/admin-portal/backend/internal/service",
		"/admin-portal/backend/internal/server",
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
