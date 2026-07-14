package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTaskDesignLintFixtures(t *testing.T) {
	tests := []struct {
		name        string
		fixture     string
		wantError   string
		wantWarning string
	}{
		{name: "compliant zero stateful", fixture: "compliant-zero"},
		{name: "compliant multi layer", fixture: "compliant-stateful"},
		{name: "invalid gate schema", fixture: "invalid-gate", wantError: "gate-header"},
		{name: "missing stateful map", fixture: "invalid-stateful", wantError: "stateful-map-missing"},
		{name: "micro package warning", fixture: "warning-micro", wantWarning: "micro-package"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposalPath, proposal := readTaskDesignFixture(t, tt.fixture, "proposal.md")
			tasksPath, tasks := readTaskDesignFixture(t, tt.fixture, "tasks.md")
			result := lintTaskDesignArtifacts(proposalPath, proposal, tasksPath, tasks)
			assertTaskDesignDiagnostic(t, result.Errors, tt.wantError)
			assertTaskDesignDiagnostic(t, result.Warnings, tt.wantWarning)
		})
	}
}

func TestTaskDesignBaselineWarnings(t *testing.T) {
	root := t.TempDir()
	writeTaskDesignTestFile(t, root, ".agents/openspec-task-lint-baseline.tsv", "change_name\treason\nlegacy-active\tactive before delivery\nlegacy-active\tduplicate row\nlegacy-archived\tarchived later\nlegacy-unknown\tmissing branch\n")
	proposalPath, proposal := readTaskDesignFixture(t, "invalid-gate", "proposal.md")
	tasksPath, tasks := readTaskDesignFixture(t, "invalid-gate", "tasks.md")
	writeTaskDesignTestFile(t, root, "openspec/changes/legacy-active/proposal.md", proposal)
	writeTaskDesignTestFile(t, root, "openspec/changes/legacy-active/tasks.md", tasks)
	writeTaskDesignTestFile(t, root, "openspec/changes/archive/2026-01-01-legacy-archived/proposal.md", proposal)
	writeTaskDesignTestFile(t, root, "openspec/changes/archive/2026-01-01-legacy-archived/tasks.md", tasks)

	result := lintTaskDesignRepository(root, "")
	if len(result.Errors) != 0 {
		t.Fatalf("active baseline errors = %v", result.Errors)
	}
	for _, code := range []string{"baseline-skip", "baseline-duplicate", "baseline-archived", "baseline-unknown"} {
		assertTaskDesignDiagnostic(t, result.Warnings, code)
	}
	_ = proposalPath
	_ = tasksPath
}

func TestTaskDesignBaselineFormatFails(t *testing.T) {
	root := t.TempDir()
	writeTaskDesignTestFile(t, root, ".agents/openspec-task-lint-baseline.tsv", "change\treason\textra\nlegacy-active\tbad\textra\n")
	result := lintTaskDesignRepository(root, "")
	assertTaskDesignDiagnostic(t, result.Errors, "baseline-format")
}

func TestTaskDesignExplicitModeBypassesBaseline(t *testing.T) {
	root := t.TempDir()
	writeTaskDesignTestFile(t, root, ".agents/openspec-task-lint-baseline.tsv", "change_name\treason\nlegacy-active\tactive before delivery\n")
	_, proposal := readTaskDesignFixture(t, "invalid-gate", "proposal.md")
	_, tasks := readTaskDesignFixture(t, "invalid-gate", "tasks.md")
	writeTaskDesignTestFile(t, root, "openspec/changes/legacy-active/proposal.md", proposal)
	writeTaskDesignTestFile(t, root, "openspec/changes/legacy-active/tasks.md", tasks)

	activeResult := lintTaskDesignRepository(root, "")
	if len(activeResult.Errors) != 0 {
		t.Fatalf("active mode errors = %v", activeResult.Errors)
	}
	assertTaskDesignDiagnostic(t, activeResult.Warnings, "baseline-skip")

	explicitResult := lintTaskDesignRepository(root, "legacy-active")
	assertTaskDesignDiagnostic(t, explicitResult.Errors, "gate-header")
}

func TestTaskDesignExplicitModeRejectsArchiveAndUnknown(t *testing.T) {
	root := t.TempDir()
	writeTaskDesignTestFile(t, root, ".agents/openspec-task-lint-baseline.tsv", "change_name\treason\n")
	for _, change := range []string{"archive", "../legacy", "missing-change"} {
		result := lintTaskDesignRepository(root, change)
		assertTaskDesignDiagnostic(t, result.Errors, "explicit-change")
	}
}

func TestTaskDesignActiveModeExcludesArchive(t *testing.T) {
	root := t.TempDir()
	writeTaskDesignTestFile(t, root, ".agents/openspec-task-lint-baseline.tsv", "change_name\treason\n")
	_, proposal := readTaskDesignFixture(t, "invalid-gate", "proposal.md")
	_, tasks := readTaskDesignFixture(t, "invalid-gate", "tasks.md")
	writeTaskDesignTestFile(t, root, "openspec/changes/archive/2026-01-01-invalid/proposal.md", proposal)
	writeTaskDesignTestFile(t, root, "openspec/changes/archive/2026-01-01-invalid/tasks.md", tasks)
	result := lintTaskDesignRepository(root, "")
	if len(result.Errors) != 0 || len(result.Warnings) != 0 {
		t.Fatalf("archive-only result = %#v", result)
	}
}

func TestOpenSpecTaskDesignLint(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	result := lintTaskDesignRepository(root, os.Getenv("OPENSPEC_TASK_LINT_CHANGE"))
	for _, warning := range result.Warnings {
		t.Logf("task-design warning [%s] %s: %s", warning.Code, warning.Path, warning.Message)
	}
	for _, lintError := range result.Errors {
		t.Errorf("task-design error [%s] %s: %s", lintError.Code, lintError.Path, lintError.Message)
	}
}

func readTaskDesignFixture(t *testing.T, fixture, name string) (string, string) {
	t.Helper()
	path := filepath.Join("testdata", "task_design", fixture, name)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return path, string(content)
}

func writeTaskDesignTestFile(t *testing.T, root, path, content string) {
	t.Helper()
	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("create fixture dir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func assertTaskDesignDiagnostic(t *testing.T, diagnostics []taskDesignDiagnostic, wantCode string) {
	t.Helper()
	if wantCode == "" {
		if len(diagnostics) != 0 {
			t.Fatalf("unexpected diagnostics = %v", diagnostics)
		}
		return
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == wantCode || strings.Contains(diagnostic.Message, wantCode) {
			return
		}
	}
	t.Fatalf("diagnostics %v missing code %q", diagnostics, wantCode)
}
