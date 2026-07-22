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

func TestInternalLayoutIsCapabilityFirst(t *testing.T) {
	t.Parallel()

	repositoryRoot := filepath.Clean(filepath.Join("..", ".."))
	forbidden := []string{
		"internal/connectors",
		"internal/httpapi",
		"internal/materialize",
		"internal/storage",
	}
	for _, relative := range forbidden {
		if _, err := os.Stat(filepath.Join(repositoryRoot, relative)); !os.IsNotExist(err) {
			t.Errorf("technical-layer package %s must move under its owning capability or platform adapter", relative)
		}
	}

	required := []string{
		"internal/agentrun/execution.go",
		"internal/agentrun/persistence/postgres/store.go",
		"internal/collector/application/application.go",
		"internal/collector/artifacts/file.go",
		"internal/collector/connectors/http.go",
		"internal/collector/httpapi/handler.go",
		"internal/collector/planning/query_planner.go",
		"internal/collector/workflow/workflow.go",
	}
	for _, relative := range required {
		if _, err := os.Stat(filepath.Join(repositoryRoot, relative)); err != nil {
			t.Errorf("required capability-first path %s is missing: %v", relative, err)
		}
	}
}

func TestAgentRunPlatformDoesNotImportAgentCapabilities(t *testing.T) {
	t.Parallel()

	agentRunRoot := filepath.Clean(filepath.Join("..", "agentrun"))
	err := filepath.WalkDir(agentRunRoot, func(path string, entry os.DirEntry, walkErr error) error {
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
			pathValue, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return err
			}
			if strings.Contains(pathValue, "/internal/collector") {
				t.Errorf("AgentRun platform package %s imports Collector capability %s", path, pathValue)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
