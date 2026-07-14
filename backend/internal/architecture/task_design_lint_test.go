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

func TestTaskDesignGateBudgetMutationsFailClosed(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(proposal, tasks string) (string, string)
		code   string
	}{
		{"risk enum", func(p, tasks string) (string, string) {
			return strings.Replace(p, "| 1 | Proposal Review | R1 |", "| 1 | Proposal Review | R9 |", 1), tasks
		}, "gate-risk"},
		{"human enum", func(p, tasks string) (string, string) {
			return strings.Replace(p, "| 1 | Proposal Review | R1 | yes |", "| 1 | Proposal Review | R1 | YES |", 1), tasks
		}, "gate-human"},
		{"reason enum", func(p, tasks string) (string, string) {
			return strings.Replace(p, "| 1 | Proposal Review | R1 | yes | SPEC_SEMANTICS |", "| 1 | Proposal Review | R1 | yes | IMPLEMENTATION |", 1), tasks
		}, "gate-reason"},
		{"duplicate package", func(p, tasks string) (string, string) {
			return strings.Replace(p, "| 2 | Apply Package |", "| 1 | Apply Package |", 1), tasks
		}, "gate-package"},
		{"gate package mapping", func(p, tasks string) (string, string) {
			return p, strings.Replace(tasks, "## 2. Apply Package", "## 4. Apply Package", 1)
		}, "package-mapping"},
		{"budget key order", func(p, tasks string) (string, string) {
			return strings.Replace(p, "| human_gates | 2 |", "| checkpoints | 2 |", 1), tasks
		}, "budget-keys"},
		{"budget unsigned integer", func(p, tasks string) (string, string) {
			return strings.Replace(p, "| checkpoints | 2 |", "| checkpoints | -1 |", 1), tasks
		}, "budget-integer"},
		{"human gate count", func(p, tasks string) (string, string) {
			return strings.Replace(p, "| human_gates | 2 |", "| human_gates | 1 |", 1), tasks
		}, "budget-human-gates"},
		{"selector missing package", func(p, tasks string) (string, string) {
			return strings.Replace(p, "packages:2", "packages:4", 1), tasks
		}, "budget-selector"},
		{"selector human package", func(p, tasks string) (string, string) {
			return strings.Replace(p, "packages:2", "packages:1", 1), tasks
		}, "budget-selector"},
		{"selector duplicate", func(p, tasks string) (string, string) {
			return strings.Replace(p, "packages:2", "packages:2,2", 1), tasks
		}, "budget-selector"},
		{"selector out of order", func(p, tasks string) (string, string) {
			return strings.Replace(p, "packages:2", "packages:2-1", 1), tasks
		}, "budget-selector"},
		{"selector leading zero", func(p, tasks string) (string, string) {
			return strings.Replace(p, "packages:2", "packages:02", 1), tasks
		}, "budget-selector"},
		{"gate row missing", func(p, tasks string) (string, string) {
			return strings.Replace(p, "| 3 | Apply-final Review |", "", 1), tasks
		}, "gate-mismatch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, proposal := readTaskDesignFixture(t, "compliant-zero", "proposal.md")
			_, tasks := readTaskDesignFixture(t, "compliant-zero", "tasks.md")
			proposal, tasks = tt.mutate(proposal, tasks)
			result := lintTaskDesignArtifacts("proposal.md", proposal, "tasks.md", tasks)
			assertTaskDesignDiagnostic(t, result.Errors, tt.code)
		})
	}
}

func TestTaskDesignStatefulMutationsFailClosed(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(string) string
		code   string
	}{
		{"header", func(s string) string { return strings.Replace(s, "| Layer | Package |", "| Wrong | Package |", 1) }, "stateful-header"},
		{"row count", func(s string) string { return strings.Replace(s, "| seed-layer | 2 |", "", 1) }, "stateful-count"},
		{"package risk", func(s string) string { return strings.Replace(s, "| schema-layer | 2 |", "| schema-layer | 1 |", 1) }, "stateful-package"},
		{"environment", func(s string) string {
			return strings.Replace(s, "| schema-layer | 2 | local |", "| schema-layer | 2 | remote |", 1)
		}, "stateful-environment"},
		{"order", func(s string) string {
			return strings.Replace(s, "| seed-layer | 2 | local | 2 |", "| seed-layer | 2 | local | 3 |", 1)
		}, "stateful-order"},
		{"required field", func(s string) string {
			return strings.Replace(s, "| schema-layer | 2 | local | 1 | schema v1 |", "| schema-layer | 2 | local | 1 |  |", 1)
		}, "stateful-field"},
		{"exclusions required", func(s string) string {
			return strings.Replace(s, "| schema-layer | 2 | local | 1 | schema v1 | none |", "| schema-layer | 2 | local | 1 | schema v1 |  |", 1)
		}, "stateful-field"},
		{"before assertions required", func(s string) string { return strings.Replace(s, "| identity scope count hash schema |", "|  |", 1) }, "stateful-field"},
		{"after assertions required", func(s string) string { return strings.Replace(s, "| schema=v1 |", "|  |", 1) }, "stateful-field"},
		{"stop conditions required", func(s string) string { return strings.Replace(s, "| drift or failure |", "|  |", 1) }, "stateful-field"},
		{"layer duplicate", func(s string) string { return strings.Replace(s, "| seed-layer |", "| schema-layer |", 1) }, "stateful-layer"},
		{"disposable environment", func(s string) string {
			s = strings.Replace(s, "| schema-layer | 2 | local | 1 |", "| schema-layer | 2 | shared-local | 1 |", 1)
			return strings.Replace(s, "| schema-layer | 2 | shared-local | 1 | schema v1 | none | backup |", "| schema-layer | 2 | shared-local | 1 | schema v1 | none | approved-disposable-recovery |", 1)
		}, "stateful-recovery"},
		{"reuse baseline", func(s string) string { return strings.Replace(s, "reuse:local-window", "reuse:missing-window", 1) }, "stateful-baseline"},
		{"baseline malformed", func(s string) string { return strings.Replace(s, "new:local-window", "bad-baseline", 1) }, "stateful-baseline"},
		{"baseline duplicate new", func(s string) string { return strings.Replace(s, "reuse:local-window", "new:local-window", 1) }, "stateful-baseline"},
		{"baseline forward reuse", func(s string) string { return strings.Replace(s, "new:local-window", "reuse:local-window", 1) }, "stateful-baseline"},
		{"baseline different environment", func(s string) string {
			return strings.Replace(s, "| seed-layer | 2 | local |", "| seed-layer | 2 | shared-local |", 1)
		}, "stateful-baseline"},
		{"expected state", func(s string) string {
			return strings.Replace(s, "counts=1;hash=abc;schema=v1", "counts=1;hash=abc", 1)
		}, "stateful-expected"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, proposal := readTaskDesignFixture(t, "compliant-stateful", "proposal.md")
			_, tasks := readTaskDesignFixture(t, "compliant-stateful", "tasks.md")
			proposal, tasks = tt.mutate(proposal), tt.mutate(tasks)
			result := lintTaskDesignArtifacts("proposal.md", proposal, "tasks.md", tasks)
			assertTaskDesignDiagnostic(t, result.Errors, tt.code)
		})
	}
}

func TestTaskDesignLocalNeo4jR3DisposableRecovery(t *testing.T) {
	proposalPath, proposal := readTaskDesignFixture(t, "compliant-local-neo4j-r3", "proposal.md")
	tasksPath, tasks := readTaskDesignFixture(t, "compliant-local-neo4j-r3", "tasks.md")
	result := lintTaskDesignArtifacts(proposalPath, proposal, tasksPath, tasks)
	if len(result.Errors) != 0 {
		t.Fatalf("real local Neo4j R3 rows errors = %v", result.Errors)
	}

	caseInsensitive := func(s string) string {
		s = strings.Replace(s, "local-neo4j-foundation-cleanup", "local-neo4j-foundation-maintenance", 1)
		s = strings.Replace(s, "仅清空 local Neo4j", "CLEANUP local nEo4J", 1)
		return strings.Replace(s, "PG projection baseline", "pOsTgReSqL projection BaSeLiNe", 1)
	}
	result = lintTaskDesignArtifacts("proposal.md", caseInsensitive(proposal), "tasks.md", caseInsensitive(tasks))
	if len(result.Errors) != 0 {
		t.Fatalf("case-insensitive anchors errors = %v", result.Errors)
	}
}

func TestTaskDesignLocalNeo4jR3DisposableRecoveryFailsClosed(t *testing.T) {
	_, baseProposal := readTaskDesignFixture(t, "compliant-local-neo4j-r3", "proposal.md")
	_, baseTasks := readTaskDesignFixture(t, "compliant-local-neo4j-r3", "tasks.md")
	tests := []struct {
		name   string
		mutate func(string) string
		code   string
	}{
		{"shared local", replaceFirst("| 1 | local | 1 |", "| 1 | shared-local | 1 |"), "stateful-recovery"},
		{"uat", replaceFirst("| 1 | local | 1 |", "| 1 | uat | 1 |"), "stateful-recovery"},
		{"prod", replaceFirst("| 1 | local | 1 |", "| 1 | prod | 1 |"), "stateful-recovery"},
		{"human no", replaceFirst("| 1 | Local Neo4j cleanup and rebuild | R3 | yes | R3_OPERATION |", "| 1 | Local Neo4j cleanup and rebuild | R3 | no | NONE |"), "stateful-recovery"},
		{"non Neo4j", chainedMutation(
			replaceFirst("local-neo4j-foundation-cleanup", "local-graph-foundation-cleanup"),
			replaceFirst("仅清空 local Neo4j", "仅清空 local graph store"),
		), "stateful-recovery"},
		{"cleanup suffix", chainedMutation(
			replaceFirst("local-neo4j-foundation-cleanup", "local-neo4j-foundation-maintenance"),
			replaceFirst("仅清空 local Neo4j", "cleanupSuffix local Neo4j"),
		), "stateful-recovery"},
		{"resync", chainedMutation(
			replaceFirst("local-neo4j-foundation-cleanup", "local-neo4j-foundation-maintenance"),
			replaceFirst("仅清空 local Neo4j", "resync local Neo4j"),
		), "stateful-recovery"},
		{"neo4j backup suffix", chainedMutation(
			replaceFirst("local-neo4j-foundation-cleanup", "local-graph-foundation-cleanup"),
			replaceFirst("仅清空 local Neo4j", "cleanup local neo4jBackup"),
		), "stateful-recovery"},
		{"missing baseline", replaceFirst("PG projection baseline identity", "PG projection snapshot identity"), "stateful-recovery"},
		{"non PG baseline", replaceFirst("PG projection baseline identity", "Redis projection baseline identity"), "stateful-recovery"},
		{"cross layer baseline", replaceFirst("| local-neo4j-foundation-rebuild | 1 | local | 2 |", "| local-neo4j-foundation-rebuild | 2 | local | 2 |"), "stateful-baseline"},
		{"incomplete after assertions", replaceFirst("| Tidewise namespace 节点与关系为零；database、约束、索引、配置和 PG 业务数据不变 |", "|  |"), "stateful-field"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposal, tasks := tt.mutate(baseProposal), tt.mutate(baseTasks)
			result := lintTaskDesignArtifacts("proposal.md", proposal, "tasks.md", tasks)
			assertTaskDesignDiagnostic(t, result.Errors, tt.code)
		})
	}
}

func TestTaskDesignLocalR2DisposableRecoveryRegression(t *testing.T) {
	proposalPath, proposal := readTaskDesignFixture(t, "compliant-stateful", "proposal.md")
	tasksPath, tasks := readTaskDesignFixture(t, "compliant-stateful", "tasks.md")
	mutate := func(s string) string {
		return strings.Replace(s, "| schema-layer | 2 | local | 1 | schema v1 | none | backup |", "| schema-layer | 2 | local | 1 | schema v1 | none | approved-disposable-recovery |", 1)
	}
	result := lintTaskDesignArtifacts(proposalPath, mutate(proposal), tasksPath, mutate(tasks))
	if len(result.Errors) != 0 {
		t.Fatalf("local R2 disposable recovery errors = %v", result.Errors)
	}
}

func replaceFirst(old, replacement string) func(string) string {
	return func(s string) string {
		return strings.Replace(s, old, replacement, 1)
	}
}

func chainedMutation(mutations ...func(string) string) func(string) string {
	return func(s string) string {
		for _, mutate := range mutations {
			s = mutate(s)
		}
		return s
	}
}

func TestTaskDesignArtifactMismatchAndHeadingBoundaries(t *testing.T) {
	_, proposal := readTaskDesignFixture(t, "compliant-stateful", "proposal.md")
	_, tasks := readTaskDesignFixture(t, "compliant-stateful", "tasks.md")
	for _, tt := range []struct {
		name   string
		mutate func(string) string
		code   string
	}{
		{"gate mismatch", func(s string) string {
			return strings.Replace(s, "| 1 | Proposal Review |", "| 1 | Changed Review |", 1)
		}, "gate-mismatch"},
		{"budget mismatch", func(s string) string { return strings.Replace(s, "| checkpoints | 2 |", "| checkpoints | 3 |", 1) }, "budget-mismatch"},
		{"stateful mismatch", func(s string) string { return strings.Replace(s, "schema v1", "schema v2", 1) }, "stateful-mismatch"},
		{"proposal heading order", func(s string) string { return strings.Replace(s, "## Why", "## Context", 1) }, "gate-heading"},
		{"tasks heading order", func(s string) string { return "## Context\n\n" + s }, "gate-heading"},
		{"budget heading order", func(s string) string { return strings.Replace(s, "## Complexity Budget", "## Other", 1) }, "budget-heading"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mutatedProposal, mutatedTasks := proposal, tasks
			if strings.Contains(tt.name, "tasks") {
				mutatedTasks = tt.mutate(tasks)
			} else {
				mutatedProposal = tt.mutate(proposal)
			}
			result := lintTaskDesignArtifacts("proposal.md", mutatedProposal, "tasks.md", mutatedTasks)
			assertTaskDesignDiagnostic(t, result.Errors, tt.code)
		})
	}

	zeroProposalPath, zeroProposal := readTaskDesignFixture(t, "compliant-zero", "proposal.md")
	zeroTasksPath, zeroTasks := readTaskDesignFixture(t, "compliant-zero", "tasks.md")
	emptyLayerTable := "\n## Stateful Layer Map\n\n| Layer | Package | Environment | Order | Scope | Exclusions | Recovery Evidence | Recovery Baseline | Expected Counts/Hash/Schema | Before Assertions | After Assertions | Stop Conditions |\n|---|---|---|---|---|---|---|---|---|---|---|---|\n"
	result := lintTaskDesignArtifacts(zeroProposalPath, zeroProposal+emptyLayerTable, zeroTasksPath, zeroTasks+emptyLayerTable)
	if len(result.Errors) != 0 {
		t.Fatalf("stateful_layers=0 empty map errors = %v", result.Errors)
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
