package architecture_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/guanchaojia/tidewise-ai-agentrun"

func TestRepositoryUsesKratosServiceLayout(t *testing.T) {
	t.Parallel()

	root := repositoryRoot()
	required := []string{
		"api/agentrun/v1/openapi.yaml",
		"cmd/server/main.go",
		"cmd/migrate/main.go",
		"cmd/config/main.go",
		"cmd/artifacts/main.go",
		"configs/config.dev.yaml",
		"configs/config.uat.yaml",
		"internal/conf",
		"internal/biz/platform",
		"internal/biz/agents/collector",
		"internal/biz/agents/collector/materialization",
		"internal/data/postgres",
		"internal/data/modelprovider/deepseek",
		"internal/data/connectors",
		"internal/data/artifacts",
		"internal/data/scheduler",
		"internal/service",
		"internal/server",
	}
	for _, relative := range required {
		if _, err := os.Stat(filepath.Join(root, relative)); err != nil {
			t.Errorf("required Kratos service path %s is missing: %v", relative, err)
		}
	}

	forbidden := []string{
		"cmd/agentrun-server",
		"cmd/agentrun-migrate",
		"cmd/agentrun-config",
		"cmd/agentrun-artifacts",
		"internal/agentrun",
		"internal/collector",
	}
	for _, relative := range forbidden {
		if _, err := os.Stat(filepath.Join(root, relative)); !os.IsNotExist(err) {
			t.Errorf("legacy path %s must be removed after the Kratos cutover", relative)
		}
	}
}

func TestBizDoesNotImportInfrastructureOrTransport(t *testing.T) {
	t.Parallel()

	forbidden := []string{
		modulePath + "/internal/data",
		modulePath + "/internal/service",
		modulePath + "/internal/server",
		"github.com/go-kratos/kratos",
		"github.com/jackc/pgx",
		"github.com/go-co-op/gocron",
		"github.com/cloudwego/eino-ext",
	}
	assertImportsAbsent(t, filepath.Join(repositoryRoot(), "internal", "biz"), forbidden)
}

func TestDataDoesNotImportServiceOrServer(t *testing.T) {
	t.Parallel()

	assertImportsAbsent(t, filepath.Join(repositoryRoot(), "internal", "data"), []string{
		modulePath + "/internal/service",
		modulePath + "/internal/server",
	})
}

func TestConcreteEinoProviderLivesOnlyInDataAdapter(t *testing.T) {
	t.Parallel()

	root := filepath.Join(repositoryRoot(), "internal")
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range file.Imports {
			value, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return err
			}
			if strings.HasPrefix(value, "github.com/cloudwego/eino-ext/") &&
				!strings.Contains(filepath.ToSlash(path), "/internal/data/modelprovider/") {
				t.Errorf("concrete Eino provider import %s must stay in internal/data/modelprovider: %s", value, path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCompositionRootDoesNotUseWireOrImplementBusinessRules(t *testing.T) {
	t.Parallel()

	root := repositoryRoot()
	for _, relative := range []string{"cmd/server/wire.go", "cmd/server/wire_gen.go"} {
		if _, err := os.Stat(filepath.Join(root, relative)); !os.IsNotExist(err) {
			t.Errorf("explicit construction is required; %s must not exist", relative)
		}
	}
}

func assertImportsAbsent(t *testing.T, root string, forbidden []string) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range file.Imports {
			value, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return err
			}
			for _, prefix := range forbidden {
				if strings.HasPrefix(value, prefix) {
					t.Errorf("%s imports forbidden dependency %s", path, value)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func repositoryRoot() string {
	return filepath.Clean(filepath.Join("..", ".."))
}
