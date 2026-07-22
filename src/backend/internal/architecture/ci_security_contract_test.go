package architecture

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestCIWorkflowRunsOncePerPullRequestWithLeastPrivilege(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	workflow := readContractFile(t, filepath.Join(root, ".github", "workflows", "ci.yml"))

	for _, required := range []string{
		"pull_request:",
		"- main",
		"permissions:",
		"contents: read",
		"pull-requests: read",
		"concurrency:",
		"cancel-in-progress: true",
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("CI workflow missing %q", required)
		}
	}
	if strings.Contains(workflow, "codex/**") {
		t.Fatal("CI must not run both push and pull_request events for codex branches")
	}
}

func TestCIWorkflowEnforcesQualityAndSecurityGates(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	workflow := readContractFile(t, filepath.Join(root, ".github", "workflows", "ci.yml"))

	for _, required := range []string{
		"name: Backend",
		"name: Frontend",
		"name: Security",
		"gofmt -l .",
		"go vet ./...",
		"go test -race ./...",
		"scripts/ci/check-prettier-diff.sh",
		"npm run lint",
		"gitleaks/gitleaks-action@",
		"actions/dependency-review-action@",
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("CI workflow missing quality or security gate %q", required)
		}
	}
}

func TestGitHubActionsArePinnedToImmutableSHAs(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	workflows, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yml"))
	if err != nil {
		t.Fatal(err)
	}
	immutableAction := regexp.MustCompile(`^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+@[0-9a-f]{40}$`)

	for _, workflow := range workflows {
		file, err := os.Open(workflow)
		if err != nil {
			t.Fatal(err)
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if !strings.HasPrefix(line, "uses:") {
				continue
			}
			action := strings.TrimSpace(strings.TrimPrefix(line, "uses:"))
			if comment := strings.Index(action, " #"); comment >= 0 {
				action = action[:comment]
			}
			if strings.HasPrefix(action, "./") {
				continue
			}
			if !immutableAction.MatchString(action) {
				file.Close()
				t.Fatalf("workflow action must use a full commit SHA: %s: %s", filepath.Base(workflow), line)
			}
		}
		if err := scanner.Err(); err != nil {
			file.Close()
			t.Fatal(err)
		}
		file.Close()
	}
}
