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
	root := repositoryRoot()
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
	root := repositoryRoot()
	workflow := readContractFile(t, filepath.Join(root, ".github", "workflows", "ci.yml"))

	for _, required := range []string{
		"name: Repository Governance",
		"name: Data Service",
		"name: Miniapp",
		"name: Admin Portal",
		"name: AgentRun",
		"name: Security",
		"bash scripts/ci/detect-app-change.sh data",
		"bash scripts/ci/detect-app-change.sh miniapp",
		"bash scripts/ci/detect-app-change.sh adminportal",
		"bash scripts/ci/detect-app-change.sh agentrun",
		"bash scripts/ci/detect-test-risk.sh repository",
		"bash scripts/ci/detect-test-risk.sh data",
		"bash scripts/ci/detect-test-risk.sh miniapp",
		"bash scripts/ci/detect-test-risk.sh adminportal",
		"bash scripts/ci/detect-test-risk.sh agentrun",
		"Test Data Biz and API seams",
		"Test Miniapp Biz and API seams",
		"Test Admin Portal Biz and API seams",
		"Test AgentRun Biz, API and Eino seams",
		"Test Data PostgreSQL boundaries",
		"Test Data migration chain",
		"Test AgentRun Data, migration and provider boundaries",
		"scripts/ci/check-prettier-diff.sh",
		"npm run test:miniapp",
		"npm run test:admin",
		"npm run build:weapp",
		"npm run build:tt",
		"npm run lint",
		"bash scripts/ci/scan-git-secrets.sh",
		"actions/upload-artifact@",
		"actions/dependency-review-action@",
		"bash scripts/ci/check-sensitive-diff.sh",
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("CI workflow missing quality or security gate %q", required)
		}
	}
	for _, forbidden := range []string{
		"name: Backend",
		"name: Frontend",
		"name: Detect changed applications",
		"go test -race ./analyse-data-service/backend/...",
		"go test -race ./miniapp/backend/...",
		"go test -race ./admin-portal/backend/...",
		"go test -race ./agent-run/backend/...",
	} {
		if strings.Contains(workflow, forbidden) {
			t.Fatalf("CI workflow contains obsolete top-level task %q", forbidden)
		}
	}
}

func TestGitSecretScanHandlesImportedRootHistory(t *testing.T) {
	root := repositoryRoot()
	script := readContractFile(t, filepath.Join(root, "scripts", "ci", "scan-git-secrets.sh"))

	for _, required := range []string{
		"GITLEAKS_VERSION=\"8.30.1\"",
		"BASE_SHA",
		"HEAD_SHA",
		"--log-opts=\"--diff-merges=first-parent ${BASE_SHA}..${HEAD_SHA}\"",
		"--report-format=sarif",
		"--report-path=",
	} {
		if !strings.Contains(script, required) {
			t.Fatalf("Git secret scan missing %q", required)
		}
	}
	for _, forbidden := range []string{"^..", "--all", "--log-opts=\"--first-parent", "--no-merges"} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("Git secret scan must cover merge and imported root history; found %q", forbidden)
		}
	}
}

func TestGitSecretAllowlistUsesExactFindingFingerprints(t *testing.T) {
	root := repositoryRoot()
	allowlist := readContractFile(t, filepath.Join(root, ".gitleaksignore"))
	fingerprint := regexp.MustCompile(`^[0-9a-f]{40}:[^:]+:[a-z0-9-]+:[0-9]+$`)
	expected := map[string]bool{
		"69737b9814275ab6374c4a3d1e261492f76d2660:internal/agentrun/persistence/postgres/store_test.go:generic-api-key:647":   false,
		"69737b9814275ab6374c4a3d1e261492f76d2660:internal/agentrun/persistence/postgres/store_test.go:generic-api-key:648":   false,
		"14973cbd75b64cfd0eb3e6fd3fb89b63d8f605c2:internal/agentrun/persistence/postgres/store_test.go:generic-api-key:285":   false,
		"14973cbd75b64cfd0eb3e6fd3fb89b63d8f605c2:internal/agentrun/persistence/postgres/store_test.go:generic-api-key:286":   false,
		"d89ba1e5d08890918b24e5a1dfc983b60fafeb37:agent-run/backend/internal/data/postgres/store_test.go:generic-api-key:872": false,
		"d89ba1e5d08890918b24e5a1dfc983b60fafeb37:agent-run/backend/internal/data/postgres/store_test.go:generic-api-key:873": false,
	}

	for _, line := range strings.Split(allowlist, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !fingerprint.MatchString(line) {
			t.Fatalf("Gitleaks allowlist entry must be an exact finding fingerprint: %q", line)
		}
		if _, ok := expected[line]; !ok {
			t.Fatalf("Gitleaks allowlist contains an unreviewed finding fingerprint: %q", line)
		}
		expected[line] = true
	}
	for entry, found := range expected {
		if !found {
			t.Fatalf("Gitleaks allowlist missing reviewed test fixture fingerprint: %q", entry)
		}
	}
}

func TestCIUploadsGitSecretReportAndFailsClosed(t *testing.T) {
	root := repositoryRoot()
	workflow := readContractFile(t, filepath.Join(root, ".github", "workflows", "ci.yml"))

	for _, required := range []string{
		"continue-on-error: true",
		"HEAD_SHA: ${{ github.event.pull_request.head.sha || github.sha }}",
		"if: always()",
		"path: ${{ runner.temp }}/gitleaks-results.sarif",
		"if-no-files-found: warn",
		"if: steps.gitleaks.outcome == 'failure'",
		"run: exit 1",
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("CI Git secret scan must upload SARIF and fail closed; missing %q", required)
		}
	}
}

func TestSensitiveDiffAllowsOnlyTheExactSKHynixPrimarySourceFalsePositive(t *testing.T) {
	root := repositoryRoot()
	script := readContractFile(t, filepath.Join(root, "scripts", "ci", "check-sensitive-diff.sh"))

	for _, required := range []string{
		`sk_hynix_source_slug='sk''-hynix-begins-volume-production-of-the-world-first-12-layer-hbm3e'`,
		`news\\.skhynix\\.com/${sk_hynix_source_slug}/`,
		`grep -Ev "$reviewed_source_pattern"`,
	} {
		if !strings.Contains(script, required) {
			t.Fatalf("sensitive diff scan is missing exact SK hynix source guard %q", required)
		}
	}
	for _, broadAllowlist := range []string{
		`news\.skhynix\.com/.*`,
		`sk-hynix-.*`,
		`source_url.*sk-`,
	} {
		if strings.Contains(script, broadAllowlist) {
			t.Fatalf("sensitive diff scan contains an overbroad credential allowlist %q", broadAllowlist)
		}
	}
}

func TestGitHubActionsArePinnedToImmutableSHAs(t *testing.T) {
	root := repositoryRoot()
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
