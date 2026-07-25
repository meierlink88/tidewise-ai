package architecture_test

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
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
	Uses string            `yaml:"uses"`
	Run  string            `yaml:"run"`
	If   string            `yaml:"if"`
	Env  map[string]string `yaml:"env"`
	With map[string]any    `yaml:"with"`
}

func TestCIWorkflowEnforcesGoAndPostgresContracts(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(monorepoRoot(), ".github", "workflows", "ci.yml"))
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
	if workflow.Permissions["contents"] != "read" || workflow.Permissions["pull-requests"] != "read" {
		t.Fatalf("CI permissions = %#v, want read-only repository and pull-request access", workflow.Permissions)
	}
	if workflow.Concurrency.Group != "ci-${{ github.workflow }}-${{ github.event.pull_request.number || github.ref }}" || !workflow.Concurrency.CancelInProgress {
		t.Fatalf("CI concurrency = %#v, want grouped runs with cancellation", workflow.Concurrency)
	}
	for _, job := range workflow.Jobs {
		assertAllActionsPinned(t, job.Steps)
	}

	if _, ok := workflow.Jobs["changes"]; ok {
		t.Fatal("AgentRun path detection must not be exposed as a top-level CI job")
	}

	quality, ok := workflow.Jobs["agentrun"]
	if !ok {
		t.Fatal("CI workflow must define the path-aware AgentRun job")
	}
	checkout := findPinnedAction(t, quality.Steps, "actions/checkout")
	if fmt.Sprint(checkout.With["fetch-depth"]) != "0" {
		t.Fatalf("AgentRun checkout fetch-depth = %#v, want 0", checkout.With["fetch-depth"])
	}
	assertRunsCommand(t, quality.Steps, "bash scripts/ci/detect-app-change.sh agentrun")
	assertSetupGoConfiguration(t, quality.Steps)
	for _, command := range []string{
		"go vet ./agent-run/backend/...",
		"go test -race ./agent-run/backend/... -count=1",
		"go build -o /tmp/agentrun ./agent-run/backend/cmd/server",
		"go build -o /tmp/agentrun-migrate ./agent-run/backend/cmd/migrate",
		"go build -o /tmp/agentrun-config ./agent-run/backend/cmd/config",
		"go build -o /tmp/agentrun-artifacts ./agent-run/backend/cmd/artifacts",
		"docker build -f agent-run/backend/Dockerfile -t tidewise-agentrun:ci .",
		"docker build -f admin-portal/backend/Dockerfile -t tidewise-adminportal:ci .",
		"bash scripts/ci/smoke-admin-agentrun-compose.sh",
	} {
		assertRunsCommand(t, quality.Steps, command)
	}

	if _, ok := workflow.Jobs["agentrun-postgres"]; ok {
		t.Fatal("AgentRun PostgreSQL verification must not be exposed as a separate CI job")
	}
	if quality.Services["postgres"].Image != "postgres:16" {
		t.Fatalf("PostgreSQL service image = %q, want postgres:16", quality.Services["postgres"].Image)
	}
	integration := findStepRunning(t, quality.Steps, "go test ./agent-run/backend/cmd/server ./agent-run/backend/cmd/config ./agent-run/backend/internal/data/postgres ./agent-run/backend/internal/data/scheduler ./agent-run/backend/internal/server ./agent-run/backend/internal/service -count=1")
	databaseURL := integration.Env["AGENTRUN_TEST_DATABASE_URL"]
	if !strings.Contains(databaseURL, "/tidewise_ai_server_test?") || strings.Contains(databaseURL, "tidewise_local") {
		t.Fatalf("unsafe integration database URL %q", databaseURL)
	}
	if integration.If != "steps.paths.outputs.changed == 'true'" {
		t.Fatalf("AgentRun PostgreSQL step condition = %q", integration.If)
	}
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
	findStepRunning(t, steps, command)
}

func findStepRunning(t *testing.T, steps []ciStep, command string) ciStep {
	t.Helper()
	for _, step := range steps {
		for _, line := range strings.Split(step.Run, "\n") {
			if strings.TrimSpace(line) == command {
				return step
			}
		}
	}
	t.Fatalf("CI workflow does not run command %q", command)
	return ciStep{}
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
