package architecture

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUATDeployExecutorSuccessRecordsCompleteReleaseWithoutLeakingSecrets(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{})
	if result.err != nil {
		t.Fatalf("deploy success fixture failed: %v\n%s", result.err, result.output)
	}
	for _, want := range []string{"PASS deployment-lock", "PASS migration-apply", "PASS bff-to-data-read-paths", "PASS release-state-recorded"} {
		if !strings.Contains(result.output, want) {
			t.Fatalf("deploy output missing %q: %s", want, result.output)
		}
	}
	if strings.Contains(result.output, "fixture-admin-secret") || strings.Contains(result.output, "fixture-db-secret") {
		t.Fatalf("deploy output leaked a secret: %s", result.output)
	}
	assertFileContent(t, filepath.Join(result.root, "state", "current.sha"), fixtureSHA)
	assertFileContains(t, filepath.Join(result.root, "state", "current.images.env"), "fixture/data:"+fixtureSHA)
	curlLog, err := os.ReadFile(result.curlLog)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"http://127.0.0.1:9012/healthz",
		"http://127.0.0.1:9012/api/miniapp/v1/research/themes?limit=1",
	} {
		if !strings.Contains(string(curlLog), want) {
			t.Fatalf("host verification missing %q: %s", want, curlLog)
		}
	}
	if strings.Contains(string(curlLog), "uat.example.test") {
		t.Fatalf("deployment attempted unsupported public-IP hairpin verification: %s", curlLog)
	}
}

func TestUATDeployExecutorTreatsNullPendingAsNoMigrations(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		migrationReport: `{"current_version":"24","pending":null,"applied":null,"remaining":null}`,
	})
	if result.err != nil {
		t.Fatalf("null pending migration report failed deployment: %v\n%s", result.err, result.output)
	}
	for _, want := range []string{"PASS rds-tls-readonly", "PASS migration-risk-gate", "PASS migration-apply"} {
		if !strings.Contains(result.output, want) {
			t.Fatalf("deploy output missing %q: %s", want, result.output)
		}
	}
}

func TestUATDeployExecutorRestoresCurrentReleaseAfterCandidateHealthFailure(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{currentRelease: true, failFirstUp: true})
	if result.err == nil {
		t.Fatal("candidate failure fixture unexpectedly succeeded")
	}
	if !strings.Contains(result.output, "PASS rollback: previous complete release restored") {
		t.Fatalf("rollback output missing success evidence: %s", result.output)
	}
	assertFileContent(t, filepath.Join(result.root, "state", "current.sha"), previousFixtureSHA)
	if strings.Contains(result.output, "previous-admin-secret") {
		t.Fatalf("rollback output leaked previous secret: %s", result.output)
	}
}

func TestUATDeployExecutorRestoresCurrentReleaseAfterHostEntryFailure(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{currentRelease: true, failFirstCurl: true})
	if result.err == nil {
		t.Fatal("public health failure fixture unexpectedly succeeded")
	}
	if !strings.Contains(result.output, "PASS rollback: previous complete release restored") {
		t.Fatalf("rollback output missing success evidence: %s", result.output)
	}
	assertFileContent(t, filepath.Join(result.root, "state", "current.sha"), previousFixtureSHA)
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dockerLog), "resolved-data-image=fixture/data:"+previousFixtureSHA) {
		t.Fatalf("rollback did not select previous image file: %s", dockerLog)
	}
}

func TestUATDeployExecutorBlocksUnconfirmedHighRiskMigration(t *testing.T) {
	report := `{"current_version":"23","pending":[{"Version":"24","Name":"add research imports"}],"applied":[],"remaining":[]}`
	result := runDeployFixture(t, deployFixtureOptions{migrationReport: report})
	if result.err == nil || !strings.Contains(result.output, "FAIL migration-risk-gate") {
		t.Fatalf("high-risk fixture was not blocked: err=%v output=%s", result.err, result.output)
	}
	logContent, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(logContent), " up ") {
		t.Fatalf("high-risk gate started services: %s", logContent)
	}
}

func TestUATDeployExecutorBlocksReleaseIncompatibleMigrationEvenWithBackup(t *testing.T) {
	report := `{"current_version":"24","pending":[{"Version":"25","Name":"rebuild research anchors"}],"applied":[],"remaining":[]}`
	result := runDeployFixture(t, deployFixtureOptions{
		migrationReport: report,
		migrationRisk:   "blocked",
		backupConfirmed: true,
	})
	if result.err == nil || !strings.Contains(result.output, "FAIL migration-release-gate") {
		t.Fatalf("release-blocked fixture was not blocked: err=%v output=%s", result.err, result.output)
	}
	logContent, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(logContent), " up ") {
		t.Fatalf("release gate started services: %s", logContent)
	}
}

func TestUATDiagnosticsRedactsCredentials(t *testing.T) {
	repoRoot := filepath.Join("..", "..", "..", "..")
	temp := t.TempDir()
	bin := filepath.Join(temp, "bin")
	if err := os.MkdirAll(bin, 0o750); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(bin, "docker"), `#!/bin/sh
echo 'Authorization: Bearer visible-token'
echo 'DATABASE_URL=postgres://data:visible-password@rds.internal:5432/uat'
echo 'ADMIN_API_TOKEN=visible-admin-token'
`)
	runtimeEnv := filepath.Join(temp, "runtime.env")
	imagesEnv := filepath.Join(temp, "images.env")
	writeFixture(t, runtimeEnv, "ADMIN_API_TOKEN=fixture\n")
	writeFixture(t, imagesEnv, "DATA_IMAGE=fixture\n")
	cmd := exec.Command("bash", filepath.Join(repoRoot, "infra", "uat", "collect-diagnostics.sh"))
	cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "RUNTIME_ENV="+runtimeEnv, "CANDIDATE_IMAGES="+imagesEnv, "COMPOSE_FILE=fixture.yaml")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("collect diagnostics: %v: %s", err, output)
	}
	for _, secret := range []string{"visible-token", "visible-password", "visible-admin-token"} {
		if strings.Contains(string(output), secret) {
			t.Fatalf("diagnostics leaked %q: %s", secret, output)
		}
	}
	if !strings.Contains(string(output), "***") {
		t.Fatalf("diagnostics did not show redaction marker: %s", output)
	}
}

const (
	fixtureSHA         = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	previousFixtureSHA = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

type deployFixtureOptions struct {
	currentRelease  bool
	failFirstUp     bool
	failFirstCurl   bool
	migrationReport string
	migrationRisk   string
	backupConfirmed bool
}

type deployFixtureResult struct {
	root      string
	dockerLog string
	curlLog   string
	output    string
	err       error
}

func runDeployFixture(t *testing.T, options deployFixtureOptions) deployFixtureResult {
	t.Helper()
	repoRoot := filepath.Join("..", "..", "..", "..")
	temp := t.TempDir()
	root := filepath.Join(temp, "uat")
	state := filepath.Join(root, "state")
	bin := filepath.Join(temp, "bin")
	for _, dir := range []string{state, bin} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}

	runtimeEnv := filepath.Join(temp, "candidate.runtime.env")
	imagesEnv := filepath.Join(temp, "candidate.images.env")
	compose := filepath.Join(temp, "compose.yaml")
	manifest := filepath.Join(temp, "migration-risk.tsv")
	dockerLog := filepath.Join(temp, "docker.log")
	upCount := filepath.Join(temp, "up-count")
	curlCount := filepath.Join(temp, "curl-count")
	curlLog := filepath.Join(temp, "curl.log")
	writeFixture(t, runtimeEnv, "ADMIN_API_TOKEN=fixture-admin-secret\n")
	writeFixture(t, imagesEnv, "DATA_IMAGE=fixture/data:"+fixtureSHA+"\nMINIAPP_IMAGE=fixture/miniapp:"+fixtureSHA+"\nADMINPORTAL_IMAGE=fixture/adminportal:"+fixtureSHA+"\nADMIN_IMAGE=fixture/admin:"+fixtureSHA+"\n")
	writeFixture(t, compose, "name: tidewise-uat\nservices: {}\n")
	migrationRisk := options.migrationRisk
	if migrationRisk == "" {
		migrationRisk = "high"
	}
	writeFixture(t, manifest, "000025\t"+migrationRisk+"\tfixture migration risk\n000024\thigh\tfixture high risk\n")

	if options.currentRelease {
		writeFixture(t, filepath.Join(root, "runtime.env"), "ADMIN_API_TOKEN=previous-admin-secret\n")
		writeFixture(t, filepath.Join(state, "current.images.env"), "DATA_IMAGE=fixture/data:"+previousFixtureSHA+"\n")
		writeFixture(t, filepath.Join(state, "current.compose.yaml"), "name: tidewise-uat\nservices: {}\n")
		writeFixture(t, filepath.Join(state, "current.sha"), previousFixtureSHA+"\n")
	}

	report := options.migrationReport
	if report == "" {
		report = `{"current_version":"24","pending":[],"applied":[],"remaining":[]}`
	}
	writeFixture(t, filepath.Join(temp, "migration.json"), report+"\n")
	writeExecutable(t, filepath.Join(bin, "curl"), `#!/bin/sh
set -eu
echo " $* " >> "$FAKE_CURL_LOG"
count=0
if [ -f "$FAKE_CURL_COUNT" ]; then count="$(cat "$FAKE_CURL_COUNT")"; fi
count=$((count + 1))
echo "$count" > "$FAKE_CURL_COUNT"
if [ "${FAKE_FAIL_FIRST_CURL:-false}" = true ] && [ "$count" -eq 1 ]; then exit 1; fi
exit 0
`)
	writeExecutable(t, filepath.Join(bin, "flock"), "#!/bin/sh\nexit 0\n")
	writeExecutable(t, filepath.Join(bin, "docker"), `#!/bin/sh
set -eu
resolved_data_image="${DATA_IMAGE:-}"
if [ -z "$resolved_data_image" ]; then
  image_file=""
  previous=""
  for argument in "$@"; do
    if [ "$previous" = "--env-file" ]; then image_file="$argument"; fi
    previous="$argument"
  done
  if [ -n "$image_file" ] && [ -f "$image_file" ]; then
    resolved_data_image="$(sed -n 's/^DATA_IMAGE=//p' "$image_file" | tail -n 1)"
  fi
fi
echo "resolved-data-image=${resolved_data_image:-unset} $* " >> "$FAKE_DOCKER_LOG"
case " $* " in
  *" run "*" /usr/local/bin/dbmigrate "*) cat "$FAKE_MIGRATION_REPORT" ;;
  *" up "*)
    count=0
    if [ -f "$FAKE_UP_COUNT" ]; then count="$(cat "$FAKE_UP_COUNT")"; fi
    count=$((count + 1))
    echo "$count" > "$FAKE_UP_COUNT"
    if [ "${FAKE_FAIL_FIRST_UP:-false}" = true ] && [ "$count" -eq 1 ]; then exit 1; fi
    ;;
esac
exit 0
`)

	cmd := exec.Command("bash", filepath.Join(repoRoot, "infra", "uat", "deploy.sh"))
	cmd.Env = append(os.Environ(),
		"PATH="+bin+":"+os.Getenv("PATH"),
		"DEPLOY_ROOT="+root,
		"RUNTIME_ENV="+runtimeEnv,
		"CANDIDATE_IMAGES="+imagesEnv,
		"COMMIT_SHA="+fixtureSHA,
		"UAT_PUBLIC_BASE_URL=http://uat.example.test",
		"UAT_DATABASE_URL=postgres://fixture:fixture-db-secret@rds.example.test:5432/tidewise_uat?sslmode=require",
		"COMPOSE_FILE="+compose,
		"MIGRATION_RISK_MANIFEST="+manifest,
		"HIGH_RISK_BACKUP_CONFIRMED="+boolText(options.backupConfirmed),
		"RUNNER_TEMP="+temp,
		"GITHUB_RUN_ID=fixture",
		"GITHUB_STEP_SUMMARY="+filepath.Join(temp, "summary.md"),
		"FAKE_DOCKER_LOG="+dockerLog,
		"FAKE_MIGRATION_REPORT="+filepath.Join(temp, "migration.json"),
		"FAKE_UP_COUNT="+upCount,
		"FAKE_FAIL_FIRST_UP="+boolText(options.failFirstUp),
		"FAKE_CURL_COUNT="+curlCount,
		"FAKE_CURL_LOG="+curlLog,
		"FAKE_FAIL_FIRST_CURL="+boolText(options.failFirstCurl),
		"DATA_IMAGE=fixture/data:"+fixtureSHA,
		"MINIAPP_IMAGE=fixture/miniapp:"+fixtureSHA,
		"ADMINPORTAL_IMAGE=fixture/adminportal:"+fixtureSHA,
		"ADMIN_IMAGE=fixture/admin:"+fixtureSHA,
	)
	output, err := cmd.CombinedOutput()
	return deployFixtureResult{root: root, dockerLog: dockerLog, curlLog: curlLog, output: string(output), err: err}
}

func writeFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		t.Fatal(err)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(content)) != want {
		t.Fatalf("%s = %q, want %q", path, content, want)
	}
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), want) {
		t.Fatalf("%s missing %q: %s", path, want, content)
	}
}

func boolText(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
