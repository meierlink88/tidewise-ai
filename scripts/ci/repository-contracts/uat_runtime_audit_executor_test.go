package architecture

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUATRuntimeAuditExecutesReadOnlyAndRedactsEnvironmentValues(t *testing.T) {
	fixture := newUATRuntimeAuditFixture(t, "valid")
	output, err := fixture.run(t)
	if err != nil {
		t.Fatalf("audit failed: %v\n%s", err, output)
	}
	for _, required := range []string{
		"PASS miniapp-health",
		"PASS admin-health",
		"PASS agentos-health",
		"PASS minio-health",
		"PASS miniapp-report-read",
		"PASS retained-runtime",
		"retired-database=tidewise_ai_server present=true",
		"retired-role=agentrun_uat present=true",
		"ABSENT container reason-server-uat",
		"ABSENT container tidewise-uat-qdrant",
		"ABSENT container tidewise-infra-uat-mysql-1",
		"ABSENT container tidewise-uat-openspg-neo4j",
		"SECRET_KEY",
		"PRESENT legacy runtime keys file=",
		"AGENTRUN_SERVICE_TOKEN",
	} {
		if !strings.Contains(output, required) {
			t.Errorf("audit output missing %q\n%s", required, output)
		}
	}
	if strings.Contains(output, "super-secret-value") {
		t.Fatalf("audit leaked an environment value:\n%s", output)
	}
	if strings.Contains(output, "leaked-second-line") || strings.Contains(output, "super-secret-host-value") {
		t.Fatalf("audit leaked a multiline or host-state environment value:\n%s", output)
	}

	logContent, err := os.ReadFile(fixture.commandLog)
	if err != nil {
		t.Fatalf("read fake command log: %v", err)
	}
	transcript := string(logContent)
	for _, forbidden := range []string{
		"docker rm", "docker stop", "docker compose down", "docker volume rm",
		"DROP DATABASE", "DROP ROLE", "systemctl disable", "systemctl stop",
	} {
		if strings.Contains(transcript, forbidden) {
			t.Fatalf("audit executed forbidden operation %q:\n%s", forbidden, transcript)
		}
	}
	for _, path := range fixture.temporaryReports() {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("temporary report remains after successful audit: %s", path)
		}
	}
}

func TestUATRuntimeAuditCleansTemporaryReportAfterFailure(t *testing.T) {
	fixture := newUATRuntimeAuditFixture(t, "invalid")
	output, err := fixture.run(t)
	if err == nil {
		t.Fatalf("audit unexpectedly succeeded:\n%s", output)
	}
	migrationReport := fixture.temporaryReports()[0]
	if _, statErr := os.Stat(migrationReport); !os.IsNotExist(statErr) {
		t.Fatalf("migration report remains after failed audit: %s", migrationReport)
	}
}

type uatRuntimeAuditFixture struct {
	root       string
	fakeBin    string
	deployRoot string
	runnerTemp string
	commandLog string
	rdsAudit   string
	runID      string
	mode       string
}

func newUATRuntimeAuditFixture(t *testing.T, mode string) uatRuntimeAuditFixture {
	t.Helper()
	root := t.TempDir()
	fixture := uatRuntimeAuditFixture{
		root:       root,
		fakeBin:    filepath.Join(root, "bin"),
		deployRoot: filepath.Join(root, "deploy"),
		runnerTemp: filepath.Join(root, "runner-temp"),
		commandLog: filepath.Join(root, "commands.log"),
		rdsAudit:   filepath.Join(root, "rds-audit"),
		runID:      "contract-test",
		mode:       mode,
	}
	for _, directory := range []string{fixture.fakeBin, filepath.Join(fixture.deployRoot, "state"), fixture.runnerTemp} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatalf("create fixture directory: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(fixture.deployRoot, "state", "current.sha"), []byte("0123456789abcdef\n"), 0o644); err != nil {
		t.Fatalf("write release state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixture.deployRoot, "runtime.env"), []byte("AGENTRUN_SERVICE_TOKEN=super-secret-host-value\nSAFE_KEY=safe\n"), 0o600); err != nil {
		t.Fatalf("write runtime state: %v", err)
	}

	writeAuditFake(t, filepath.Join(fixture.fakeBin, "docker"), fakeDockerCommand)
	writeAuditFake(t, filepath.Join(fixture.fakeBin, "curl"), fakeCurlCommand)
	writeAuditFake(t, filepath.Join(fixture.fakeBin, "flock"), fakeSuccessCommand)
	writeAuditFake(t, filepath.Join(fixture.fakeBin, "ss"), fakeSuccessCommand)
	writeAuditFake(t, filepath.Join(fixture.fakeBin, "systemctl"), fakeSuccessCommand)
	writeAuditFake(t, filepath.Join(fixture.fakeBin, "getent"), "#!/bin/sh\nexit 2\n")
	writeAuditFake(t, fixture.rdsAudit, fakeRDSAuditCommand)
	return fixture
}

func (f uatRuntimeAuditFixture) run(t *testing.T) (string, error) {
	t.Helper()
	script := filepath.Join(repositoryRoot(), "infra", "uat", "audit-retired-runtime.sh")
	command := exec.Command("bash", script)
	command.Env = append(os.Environ(),
		"PATH="+f.fakeBin+":"+os.Getenv("PATH"),
		"DEPLOY_ROOT="+f.deployRoot,
		"UAT_RUNNER_NAME=contract-runner",
		"RUNNER_NAME=contract-runner",
		"RUNNER_TEMP="+f.runnerTemp,
		"GITHUB_RUN_ID="+f.runID,
		"RDS_AUDIT_BINARY="+f.rdsAudit,
		"FAKE_COMMAND_LOG="+f.commandLog,
		"FAKE_MIGRATION_MODE="+f.mode,
	)
	output, err := command.CombinedOutput()
	return string(output), err
}

func (f uatRuntimeAuditFixture) temporaryReports() []string {
	return []string{
		filepath.Join(f.runnerTemp, "tidewise-uat-runtime-audit-migration-"+f.runID+".json"),
		filepath.Join(f.runnerTemp, "tidewise-uat-runtime-audit-rds-"+f.runID+".json"),
	}
}

func writeAuditFake(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake command %s: %v", path, err)
	}
}

const fakeSuccessCommand = `#!/bin/sh
printf '%s %s\n' "$(basename "$0")" "$*" >> "$FAKE_COMMAND_LOG"
exit 0
`

const fakeCurlCommand = `#!/bin/sh
printf 'curl %s\n' "$*" >> "$FAKE_COMMAND_LOG"
case " $* " in
  *" -w "*) printf '200' ;;
esac
exit 0
`

const fakeRDSAuditCommand = `#!/bin/sh
printf 'rds-audit %s\n' "$*" >> "$FAKE_COMMAND_LOG"
printf '%s\n' '{"current_database":"tidewise_uat","current_role":"tidewise_uat","retired_database":"tidewise_ai_server","retired_database_present":true,"retired_role":"agentrun_uat","retired_role_present":true}'
`

const fakeDockerCommand = `#!/bin/sh
printf 'docker %s\n' "$*" >> "$FAKE_COMMAND_LOG"

retired=' tidewise-uat-agentrun-1 agentrun-service agentrun-migrate agentrun-agent-version reason-server-uat tidewise-uat-qdrant tidewise-infra-uat-mysql-1 tidewise-uat-openspg-neo4j '

case "$1" in
  inspect)
    shift
    format=''
    if [ "${1:-}" = '--format' ]; then
      format="$2"
      name="$3"
    else
      name="$1"
    fi
    case "$retired" in
      *" $name "*) exit 1 ;;
    esac
    case "$format" in
      *'.State.Status'*) printf '%s\n' 'running' ;;
      *'.State.Health'*) printf '%s\n' 'healthy' ;;
      *'.Config.Image'*) printf '%s\n' '  image=fixture@sha256:abc state=running health=healthy restart=unless-stopped' ;;
      *'.NetworkSettings.Networks'*) printf '%s\n' '  networks=tidewise-uat ' ;;
      *'.Mounts'*) : ;;
      *'.LogPath'*) printf '%s\n' '  log-path=/var/lib/docker/containers/fixture/fixture-json.log' ;;
      *'json .Config.Env'*)
        printf '%s\n' '["SAFE_KEY=safe-value","SECRET_KEY=super-secret-value\\nleaked-second-line","QDRANT_URL=http://retired.invalid"]'
        ;;
    esac
    ;;
  ps)
    case " $* " in
      *" --filter "*) : ;;
      *) printf '%s\n' 'container=tidewise-uat-data-1 | image=fixture@sha256:abc | state=running | status=Up | ports= | networks=tidewise-uat' ;;
    esac
    ;;
  volume)
    exit 1
    ;;
  exec)
    if [ "${FAKE_MIGRATION_MODE:-valid}" = invalid ]; then
      printf '%s\n' '{'
    else
      printf '%s\n' '{"current_version":"80","pending":[]}'
    fi
    ;;
  *)
    printf 'unsupported fake docker command: %s\n' "$*" >&2
    exit 2
    ;;
esac
`
