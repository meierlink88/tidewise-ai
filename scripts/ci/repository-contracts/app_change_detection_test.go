package architecture

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplicationChangeDetectionRoutesAffectedBoundaries(t *testing.T) {
	tests := []struct {
		name string
		path string
		want map[string]bool
	}{
		{
			name: "Miniapp frontend",
			path: "miniapp/frontend/src/pages/index.tsx",
			want: map[string]bool{"data": false, "miniapp": true, "adminportal": false, "agentrun": false},
		},
		{
			name: "Admin Portal backend",
			path: "admin-portal/backend/internal/service/collector.go",
			want: map[string]bool{"data": false, "miniapp": false, "adminportal": true, "agentrun": true},
		},
		{
			name: "Data API",
			path: "analyse-data-service/backend/api/data/v1/research.go",
			want: map[string]bool{"data": true, "miniapp": true, "adminportal": true, "agentrun": false},
		},
		{
			name: "AgentRun API",
			path: "agent-run/backend/api/agentrun/v1/admin.go",
			want: map[string]bool{"data": false, "miniapp": false, "adminportal": true, "agentrun": true},
		},
		{
			name: "Shared frontend lockfile",
			path: "package-lock.json",
			want: map[string]bool{"data": false, "miniapp": true, "adminportal": true, "agentrun": false},
		},
		{
			name: "Shared Go module",
			path: "go.mod",
			want: map[string]bool{"data": true, "miniapp": true, "adminportal": true, "agentrun": true},
		},
	}

	script, err := filepath.Abs(filepath.Join(repositoryRoot(), "scripts", "ci", "detect-app-change.sh"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := t.TempDir()
			runTestCommand(t, repo, "git", "init", "-q")
			runTestCommand(t, repo, "git", "config", "user.name", "CI Contract")
			runTestCommand(t, repo, "git", "config", "user.email", "ci-contract@example.invalid")
			writeChangeDetectionFixture(t, repo, "README.md", "baseline\n")
			runTestCommand(t, repo, "git", "add", ".")
			runTestCommand(t, repo, "git", "commit", "-q", "-m", "baseline")
			base := strings.TrimSpace(runTestCommand(t, repo, "git", "rev-parse", "HEAD"))

			writeChangeDetectionFixture(t, repo, tt.path, "changed\n")
			runTestCommand(t, repo, "git", "add", ".")
			runTestCommand(t, repo, "git", "commit", "-q", "-m", "change")
			head := strings.TrimSpace(runTestCommand(t, repo, "git", "rev-parse", "HEAD"))

			for _, scope := range []string{"data", "miniapp", "adminportal", "agentrun"} {
				outputFile := filepath.Join(repo, scope+".output")
				cmd := exec.Command("bash", script, scope)
				cmd.Dir = repo
				cmd.Env = append(os.Environ(),
					"BASE_SHA="+base,
					"HEAD_SHA="+head,
					"GITHUB_OUTPUT="+outputFile,
				)
				output, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("detect %s changes: %v: %s", scope, err, output)
				}
				result, err := os.ReadFile(outputFile)
				if err != nil {
					t.Fatal(err)
				}
				got := strings.TrimSpace(string(result)) == "changed=true"
				if got != tt.want[scope] {
					t.Fatalf("%s changed = %t, want %t; output=%q", scope, got, tt.want[scope], result)
				}
			}
		})
	}
}

func writeChangeDetectionFixture(t *testing.T, root, name, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func runTestCommand(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v: %s", name, strings.Join(args, " "), err, output)
	}
	return string(output)
}
