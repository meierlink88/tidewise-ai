package architecture_test

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type ciWorkflow struct {
	On          map[string]any    `yaml:"on"`
	Permissions map[string]string `yaml:"permissions"`
	Concurrency ciConcurrency     `yaml:"concurrency"`
	Jobs        map[string]ciJob  `yaml:"jobs"`
}

type ciConcurrency struct {
	Group            string `yaml:"group"`
	CancelInProgress bool   `yaml:"cancel-in-progress"`
}

type ciJob struct {
	Env      map[string]string    `yaml:"env"`
	Services map[string]ciService `yaml:"services"`
	Steps    []ciStep             `yaml:"steps"`
}

type ciService struct {
	Image string `yaml:"image"`
}

type ciStep struct {
	Uses string         `yaml:"uses"`
	Run  string         `yaml:"run"`
	With map[string]any `yaml:"with"`
}

func TestCIWorkflowEnforcesGoAndPostgresContracts(t *testing.T) {
	content, err := os.ReadFile("../../.github/workflows/ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var workflow ciWorkflow
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		t.Fatalf("parse CI workflow: %v", err)
	}
	if _, ok := workflow.On["pull_request"]; !ok {
		t.Fatal("CI workflow must run for pull requests")
	}
	push, ok := workflow.On["push"].(map[string]any)
	if !ok {
		t.Fatal("CI workflow must run for pushes to main")
	}
	if !containsValue(push["branches"], "main") || collectionLength(push["branches"]) != 1 {
		t.Fatalf("CI push branches = %#v, want only main", push["branches"])
	}
	if len(workflow.Permissions) != 1 || workflow.Permissions["contents"] != "read" {
		t.Fatalf("CI permissions = %#v, want only contents: read", workflow.Permissions)
	}
	if workflow.Concurrency.Group != "ci-${{ github.workflow }}-${{ github.ref }}" || !workflow.Concurrency.CancelInProgress {
		t.Fatalf("CI concurrency = %#v, want grouped runs with cancellation", workflow.Concurrency)
	}
	for _, job := range workflow.Jobs {
		assertAllActionsPinned(t, job.Steps)
	}

	quality, ok := workflow.Jobs["quality"]
	if !ok {
		t.Fatal("CI workflow must define the quality job")
	}
	checkout := findPinnedAction(t, quality.Steps, "actions/checkout")
	if fmt.Sprint(checkout.With["fetch-depth"]) != "0" {
		t.Fatalf("quality checkout fetch-depth = %#v, want 0", checkout.With["fetch-depth"])
	}
	assertSetupGoConfiguration(t, quality.Steps)
	for _, command := range []string{
		"go mod download",
		"go vet ./...",
		"go test ./... -count=1",
		"go test -race ./... -count=1",
		"go build ./cmd/...",
		"docker build --tag tidewise-ai-agentrun:ci .",
	} {
		assertRunsCommand(t, quality.Steps, command)
	}
	for _, command := range []string{
		"unformatted=\"$(gofmt -l $(git ls-files '*.go'))\"",
		"if ! git diff --check \"$base\" \"$GITHUB_SHA\" >/dev/null 2>&1; then",
		"changed_paths=\"$(mktemp \"$RUNNER_TEMP/agentrun-changed-paths.",
		"added_diff=\"$(mktemp \"$RUNNER_TEMP/agentrun-added-diff.",
		"trap 'rm -f \"$changed_paths\" \"$added_diff\"' EXIT",
		"git diff --name-only \"$base\" \"$GITHUB_SHA\" >\"$changed_paths\"",
		"if grep -Eq '^(data|\\.reference)",
		"git diff --unified=0 \"$base\" \"$GITHUB_SHA\" >\"$added_diff\"",
		"if grep -Eq '^\\+.*(sk-",
	} {
		assertRunLineStartsWith(t, quality.Steps, command)
	}
	assertRunLineCount(t, quality.Steps, "if [ \"$grep_status\" -ne 1 ]; then", 2)

	integration, ok := workflow.Jobs["postgres-integration"]
	if !ok {
		t.Fatal("CI workflow must define the postgres-integration job")
	}
	findPinnedAction(t, integration.Steps, "actions/checkout")
	assertSetupGoConfiguration(t, integration.Steps)
	if integration.Services["postgres"].Image != "postgres:16" {
		t.Fatalf("PostgreSQL service image = %q, want postgres:16", integration.Services["postgres"].Image)
	}
	databaseURL := integration.Env["AGENTRUN_TEST_DATABASE_URL"]
	if !strings.Contains(databaseURL, "/tidewise_ai_server_test?") || strings.Contains(databaseURL, "tidewise_local") {
		t.Fatalf("unsafe integration database URL %q", databaseURL)
	}
	assertRunsCommand(t, integration.Steps, "go test ./cmd/server ./cmd/config ./internal/data/postgres ./internal/data/scheduler ./internal/server ./internal/service -count=1")
}

func assertAllActionsPinned(t *testing.T, steps []ciStep) {
	t.Helper()
	for _, step := range steps {
		if step.Uses == "" {
			continue
		}
		parts := strings.Split(step.Uses, "@")
		if len(parts) != 2 {
			t.Fatalf("invalid action reference %q", step.Uses)
		}
		decoded, err := hex.DecodeString(parts[1])
		if err != nil || len(decoded) != 20 {
			t.Fatalf("action %s must use a full 40-character commit SHA", parts[0])
		}
	}
}

func assertSetupGoConfiguration(t *testing.T, steps []ciStep) {
	t.Helper()
	setupGo := findPinnedAction(t, steps, "actions/setup-go")
	if setupGo.With["go-version-file"] != "go.mod" || setupGo.With["cache-dependency-path"] != "go.sum" {
		t.Fatalf("setup-go configuration = %#v", setupGo.With)
	}
}

func findPinnedAction(t *testing.T, steps []ciStep, action string) ciStep {
	t.Helper()
	for _, step := range steps {
		prefix := action + "@"
		if !strings.HasPrefix(step.Uses, prefix) {
			continue
		}
		ref := strings.TrimPrefix(step.Uses, prefix)
		decoded, err := hex.DecodeString(ref)
		if err != nil || len(decoded) != 20 {
			t.Fatalf("action %s must use a full 40-character commit SHA, got %q", action, ref)
		}
		return step
	}
	t.Fatalf("CI workflow does not use %s", action)
	return ciStep{}
}

func assertRunsCommand(t *testing.T, steps []ciStep, command string) {
	t.Helper()
	for _, step := range steps {
		for _, line := range strings.Split(step.Run, "\n") {
			if strings.TrimSpace(line) == command {
				return
			}
		}
	}
	t.Fatalf("CI workflow does not run command %q", command)
}

func assertRunLineStartsWith(t *testing.T, steps []ciStep, prefix string) {
	t.Helper()
	for _, step := range steps {
		for _, line := range strings.Split(step.Run, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), prefix) {
				return
			}
		}
	}
	t.Fatalf("CI workflow does not run a command starting with %q", prefix)
}

func assertRunLineCount(t *testing.T, steps []ciStep, command string, expected int) {
	t.Helper()
	count := 0
	for _, step := range steps {
		for _, line := range strings.Split(step.Run, "\n") {
			if strings.TrimSpace(line) == command {
				count++
			}
		}
	}
	if count != expected {
		t.Fatalf("CI workflow command %q count = %d, want %d", command, count, expected)
	}
}

func containsValue(value any, expected string) bool {
	items, ok := value.([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		if item == expected {
			return true
		}
	}
	return false
}

func collectionLength(value any) int {
	items, _ := value.([]any)
	return len(items)
}
