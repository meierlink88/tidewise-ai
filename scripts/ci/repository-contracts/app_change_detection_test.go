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
			want: map[string]bool{"data": false, "miniapp": true, "adminportal": false},
		},
		{
			name: "Admin Portal backend",
			path: "admin-portal/backend/internal/service/collector.go",
			want: map[string]bool{"data": false, "miniapp": false, "adminportal": true},
		},
		{
			name: "Admin Portal Context",
			path: "docs/contexts/adminportal/CONTEXT.md",
			want: map[string]bool{"data": false, "miniapp": false, "adminportal": true},
		},
		{
			name: "Data API",
			path: "data-service/backend/api/data/v1/research.go",
			want: map[string]bool{"data": true, "miniapp": true, "adminportal": true},
		},
		{
			name: "Shared frontend lockfile",
			path: "package-lock.json",
			want: map[string]bool{"data": false, "miniapp": true, "adminportal": true},
		},
		{
			name: "Shared Go module",
			path: "go.mod",
			want: map[string]bool{"data": true, "miniapp": true, "adminportal": true},
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

			for _, scope := range []string{"data", "miniapp", "adminportal"} {
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

func TestRiskBoundaryDetectionSelectsOnlyAffectedSuites(t *testing.T) {
	tests := []struct {
		name  string
		scope string
		path  string
		want  map[string]bool
	}{
		{
			name:  "Data Biz change uses default seams",
			scope: "data",
			path:  "data-service/backend/internal/biz/research/biz.go",
			want:  map[string]bool{"default": true},
		},
		{
			name:  "Data API change does not select migration seams",
			scope: "data",
			path:  "data-service/backend/api/data/v1/event/http.go",
			want:  map[string]bool{"default": true, "provider_consumer": true},
		},
		{
			name:  "Data adapter change does not select migration seams",
			scope: "data",
			path:  "data-service/backend/internal/data/evidence/data.go",
			want:  map[string]bool{"default": true, "data": true},
		},
		{
			name:  "Data Object Schema selects the pinned OpenSPG parser",
			scope: "data",
			path:  "data-service/doctype/region.schema",
			want:  map[string]bool{"object_schema": true},
		},
		{
			name:  "Data migration SQL selects only the forward smoke",
			scope: "data",
			path:  "data-service/backend/migrations/000099_example.sql",
			want:  map[string]bool{"migration": true, "migration_smoke": true},
		},
		{
			name:  "Data migration framework selects unit tests and forward smoke",
			scope: "data",
			path:  "data-service/backend/internal/data/dbmigration/executor.go",
			want:  map[string]bool{"migration": true, "migration_smoke": true, "migration_framework": true},
		},
		{
			name:  "Miniapp frontend change does not select Backend suites",
			scope: "miniapp",
			path:  "miniapp/frontend/src/pages/index.tsx",
			want:  map[string]bool{"frontend": true},
		},
		{
			name:  "Miniapp Data adapter selects its provider contract",
			scope: "miniapp",
			path:  "miniapp/backend/internal/data/client.go",
			want:  map[string]bool{"default": true, "data": true, "provider_consumer": true},
		},
		{
			name:  "Shared Go module selects every Backend risk affected by dependencies",
			scope: "data",
			path:  "go.mod",
			want: map[string]bool{
				"default": true, "data": true, "migration": true,
				"migration_smoke": true, "migration_framework": true,
				"conf_lifecycle": true, "provider_consumer": true,
			},
		},
		{
			name:  "Shared frontend lockfile selects frontend behavior",
			scope: "miniapp",
			path:  "package-lock.json",
			want:  map[string]bool{"frontend": true},
		},
		{
			name:  "CI workflow selects container contracts",
			scope: "data",
			path:  ".github/workflows/ci.yml",
			want:  map[string]bool{"container": true},
		},
		{
			name:  "Runtime configuration selects the conditional lifecycle seam",
			scope: "adminportal",
			path:  "admin-portal/backend/internal/conf/config.go",
			want:  map[string]bool{"default": true, "conf_lifecycle": true},
		},
		{
			name:  "Container change does not repeat application tests",
			scope: "miniapp",
			path:  "miniapp/backend/Dockerfile",
			want:  map[string]bool{"container": true},
		},
		{
			name:  "Repository governance follows Backend dependency changes",
			scope: "repository",
			path:  "miniapp/backend/internal/biz/research.go",
			want:  map[string]bool{"architecture": true},
		},
		{
			name:  "Repository governance follows CI workflow changes",
			scope: "repository",
			path:  ".github/workflows/ci.yml",
			want:  map[string]bool{"architecture": true},
		},
	}

	script, err := filepath.Abs(filepath.Join(repositoryRoot(), "scripts", "ci", "detect-test-risk.sh"))
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

			outputFile := filepath.Join(repo, tt.scope+".output")
			cmd := exec.Command("bash", script, tt.scope)
			cmd.Dir = repo
			cmd.Env = append(os.Environ(),
				"BASE_SHA="+base,
				"HEAD_SHA="+head,
				"GITHUB_OUTPUT="+outputFile,
			)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("detect %s risks: %v: %s", tt.scope, err, output)
			}
			result, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatal(err)
			}
			got := parseRiskOutputs(string(result))
			for _, risk := range []string{
				"default", "frontend", "data", "migration", "migration_smoke", "migration_framework", "conf_lifecycle",
				"provider_consumer", "container", "architecture",
				"object_schema",
			} {
				if got[risk] != tt.want[risk] {
					t.Fatalf("%s = %t, want %t; output=%q", risk, got[risk], tt.want[risk], result)
				}
			}
		})
	}
}

func parseRiskOutputs(output string) map[string]bool {
	result := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if ok {
			result[key] = value == "true"
		}
	}
	return result
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
