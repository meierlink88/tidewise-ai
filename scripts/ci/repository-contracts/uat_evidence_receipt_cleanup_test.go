package architecture

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUATEvidenceReceiptCleanupExecutorGatesAndOutcomes(t *testing.T) {
	tests := []struct {
		name        string
		options     cleanupFixtureOptions
		wantError   string
		wantOutput  string
		wantApply   bool
		wantCleanup string
	}{
		{name: "requires recovery point", options: cleanupFixtureOptions{recoveryPointConfirmed: false}, wantError: "confirm_recovery_point=true is required"},
		{name: "timeout removes exact one-shot container", options: cleanupFixtureOptions{recoveryPointConfirmed: true, commandTimeout: true}, wantError: "operation timed out", wantCleanup: "tidewise-receipt-cleanup-fixture-run-preflight"},
		{name: "rejects unexpected ledger", options: cleanupFixtureOptions{recoveryPointConfirmed: true, preflightReport: `{"current_version":"42","pending":[{"Version":"000043"},{"Version":"000044"}]}`}, wantError: "unexpected cleanup state"},
		{name: "classifies migration failure before commit", options: cleanupFixtureOptions{recoveryPointConfirmed: true, applyFailure: true}, wantError: "not-verified-after-command-failure", wantApply: true},
		{name: "classifies migration failure after commit", options: cleanupFixtureOptions{recoveryPointConfirmed: true, applyFailure: true, applyCommitted: true}, wantError: "applied-but-command-failed", wantApply: true},
		{name: "applies exact cleanup", options: cleanupFixtureOptions{recoveryPointConfirmed: true}, wantOutput: "PASS evidence-receipt-cleanup: apply", wantApply: true},
		{name: "rejects protected fact drift", options: cleanupFixtureOptions{recoveryPointConfirmed: true, protectedDrift: true}, wantError: "protected table fact drift", wantApply: true},
		{name: "verified rerun is no-op", options: cleanupFixtureOptions{recoveryPointConfirmed: true, alreadyApplied: true}, wantOutput: "PASS evidence-receipt-cleanup: verified-noop", wantApply: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := runCleanupFixture(t, test.options)
			if test.wantError != "" {
				if result.err == nil || !strings.Contains(result.output, test.wantError) {
					t.Fatalf("error=%v output=%s", result.err, result.output)
				}
				if !strings.Contains(result.summary, "Operation outcome") || !strings.Contains(result.summary, "Service containers and release state: `unchanged`") {
					t.Fatalf("failure summary is incomplete: %s", result.summary)
				}
				if test.wantCleanup != "" && !strings.Contains(result.dockerLog, "rm -f "+test.wantCleanup) {
					t.Fatalf("timed out container was not removed: %s", result.dockerLog)
				}
				return
			}
			if test.wantOutput != "" && (result.err != nil || !strings.Contains(result.output, test.wantOutput)) {
				t.Fatalf("error=%v output=%s", result.err, result.output)
			}
			applied := strings.Contains(result.dockerLog, " -apply -target-version 44 ")
			if applied != test.wantApply {
				t.Fatalf("apply=%v want=%v log=%s", applied, test.wantApply, result.dockerLog)
			}
		})
	}
}

func TestUATEvidenceReceiptCleanupWorkflowIsIndependentAndImmutable(t *testing.T) {
	root := repositoryRoot()
	workflow := readContractFile(t, filepath.Join(root, ".github", "workflows", "uat-evidence-receipt-cleanup.yml"))
	for _, required := range []string{
		"workflow_dispatch:", "confirm_recovery_point:", "workflow_id: 'ci.yml'", "group: uat-deploy",
		"runs-on: [self-hosted, linux, x64, tidewise-uat-ecs]", "environment: uat",
		"receipt-cleanup", "@${DATA_DIGEST}", "@${CONTROL_DIGEST}", "run-evidence-receipt-cleanup.sh",
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("cleanup workflow missing %q", required)
		}
	}
	for _, forbidden := range []string{"infra/uat/deploy.sh", "docker compose up", "current.sha", "current.images.env"} {
		if strings.Contains(workflow, forbidden) {
			t.Fatalf("cleanup workflow crosses system deployment boundary %q", forbidden)
		}
	}
}

type cleanupFixtureOptions struct {
	recoveryPointConfirmed bool
	preflightReport        string
	alreadyApplied         bool
	applyFailure           bool
	applyCommitted         bool
	protectedDrift         bool
	commandTimeout         bool
}

type cleanupFixtureResult struct {
	output    string
	dockerLog string
	summary   string
	err       error
}

func runCleanupFixture(t *testing.T, options cleanupFixtureOptions) cleanupFixtureResult {
	t.Helper()
	temp := t.TempDir()
	root := filepath.Join(temp, "uat")
	state := filepath.Join(root, "state")
	bin := filepath.Join(temp, "bin")
	if err := os.MkdirAll(state, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0o750); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(root, "runtime.env"), "TIDEWISW_DB_PASSWORD=fixture\nDATA_SERVICE_TOKEN=fixture\n")
	writeFixture(t, filepath.Join(state, "current.images.env"), "DATA_IMAGE=fixture/data:old\nMINIAPP_IMAGE=fixture/miniapp:old\nADMINPORTAL_IMAGE=fixture/adminportal:old\nADMIN_IMAGE=fixture/admin:old\nAGENTRUN_IMAGE=fixture/agentrun:old\n")
	writeFixture(t, filepath.Join(state, "current.compose.yaml"), "name: tidewise-uat\nservices: {}\n")
	writeFixture(t, filepath.Join(state, "current.sha"), fixtureSHA+"\n")
	preflight := options.preflightReport
	objectsExist := true
	if options.alreadyApplied {
		preflight = `{"current_version":"44","pending":[],"remaining":[]}`
		objectsExist = false
	}
	if preflight == "" {
		preflight = `{"current_version":"43","pending":[{"Version":"000044"}],"remaining":[{"Version":"000044"}]}`
	}
	apply := `{"current_version":"44","pending":[{"Version":"000044"}],"applied":[{"Version":"000044"}],"remaining":[]}`
	verification := apply
	before := cleanupAuditJSON(objectsExist, "raw-fingerprint", "evidence-fingerprint")
	afterRaw := "raw-fingerprint"
	if options.protectedDrift {
		afterRaw = "changed-raw-fingerprint"
	}
	after := cleanupAuditJSON(false, afterRaw, "evidence-fingerprint")
	if options.applyFailure && !options.applyCommitted {
		verification = preflight
		after = before
	}
	writeFixture(t, filepath.Join(temp, "preflight.json"), preflight+"\n")
	writeFixture(t, filepath.Join(temp, "apply.json"), apply+"\n")
	writeFixture(t, filepath.Join(temp, "verification.json"), verification+"\n")
	writeFixture(t, filepath.Join(temp, "before.json"), before+"\n")
	writeFixture(t, filepath.Join(temp, "after.json"), after+"\n")
	dockerLog := filepath.Join(temp, "docker.log")
	writeExecutable(t, filepath.Join(bin, "flock"), "#!/bin/sh\nexit 0\n")
	writeExecutable(t, filepath.Join(bin, "timeout"), "#!/bin/sh\nif [ \"${FAKE_COMMAND_TIMEOUT:-false}\" = true ]; then echo 'operation timed out' >&2; exit 124; fi\nshift 3\nexec \"$@\"\n")
	writeExecutable(t, filepath.Join(bin, "docker"), `#!/bin/sh
set -eu
echo " $* " >> "$FAKE_DOCKER_LOG"
case " $* " in
  *" /usr/local/bin/dbmigrate -apply -target-version 44 "*)
    if [ "$FAKE_APPLY_FAILURE" = true ]; then exit 1; fi
    cat "$FAKE_APPLY_REPORT"
    ;;
  *" /usr/local/bin/dbmigrate "*)
    count=0
    if [ -f "$FAKE_MIGRATION_READ_COUNT" ]; then count="$(cat "$FAKE_MIGRATION_READ_COUNT")"; fi
    count=$((count + 1))
    echo "$count" > "$FAKE_MIGRATION_READ_COUNT"
    if [ "$count" -eq 1 ]; then cat "$FAKE_PREFLIGHT_REPORT"; else cat "$FAKE_VERIFICATION_REPORT"; fi
    ;;
  *" /usr/local/bin/uat-evidence-receipt-cleanup-audit "*)
    count=0
    if [ -f "$FAKE_AUDIT_COUNT" ]; then count="$(cat "$FAKE_AUDIT_COUNT")"; fi
    count=$((count + 1))
    echo "$count" > "$FAKE_AUDIT_COUNT"
    if [ "$count" -eq 1 ]; then cat "$FAKE_BEFORE_AUDIT"; else cat "$FAKE_AFTER_AUDIT"; fi
    ;;
esac
`)
	confirmation := "false"
	if options.recoveryPointConfirmed {
		confirmation = "true"
	}
	cmd := exec.Command("bash", filepath.Join(repositoryRoot(), "infra", "uat", "run-evidence-receipt-cleanup.sh"))
	cmd.Env = append(os.Environ(),
		"PATH="+bin+":"+os.Getenv("PATH"),
		"DEPLOY_ROOT="+root,
		"TARGET_DATA_IMAGE=fixture/data@sha256:"+strings.Repeat("a", 64),
		"RECOVERY_POINT_CONFIRMED="+confirmation,
		"RUNNER_TEMP="+temp,
		"GITHUB_RUN_ID=fixture-run",
		"GITHUB_STEP_SUMMARY="+filepath.Join(temp, "summary.md"),
		"FAKE_DOCKER_LOG="+dockerLog,
		"FAKE_PREFLIGHT_REPORT="+filepath.Join(temp, "preflight.json"),
		"FAKE_APPLY_REPORT="+filepath.Join(temp, "apply.json"),
		"FAKE_VERIFICATION_REPORT="+filepath.Join(temp, "verification.json"),
		"FAKE_MIGRATION_READ_COUNT="+filepath.Join(temp, "migration-read-count"),
		"FAKE_BEFORE_AUDIT="+filepath.Join(temp, "before.json"),
		"FAKE_AFTER_AUDIT="+filepath.Join(temp, "after.json"),
		"FAKE_AUDIT_COUNT="+filepath.Join(temp, "audit-count"),
		"FAKE_APPLY_FAILURE="+boolText(options.applyFailure),
		"FAKE_COMMAND_TIMEOUT="+boolText(options.commandTimeout),
	)
	output, err := cmd.CombinedOutput()
	log, _ := os.ReadFile(dockerLog)
	summary, _ := os.ReadFile(filepath.Join(temp, "summary.md"))
	return cleanupFixtureResult{output: string(output), dockerLog: string(log), summary: string(summary), err: err}
}

func cleanupAuditJSON(objectsExist bool, rawFingerprint, evidenceFingerprint string) string {
	value := "false"
	if objectsExist {
		value = "true"
	}
	rawRows := `"raw-1":"raw-row-1","raw-2":"raw-row-2","raw-3":"raw-row-3"`
	if rawFingerprint == "changed-raw-fingerprint" {
		rawRows = `"raw-1":"changed-raw-row-1","raw-2":"raw-row-2","raw-3":"raw-row-3"`
	}
	return `{"contract_version":"uat-evidence-receipt-cleanup-audit.v1","objects":{"raw_evidence_publication_receipts":` + value + `,"evidence_publication_receipts":` + value + `,"prevent_evidence_publication_receipt_mutation":` + value + `},"protected_tables":{"raw_evidences":{"row_count":3,"fingerprint":"` + rawFingerprint + `","row_fingerprints":{` + rawRows + `}},"evidences":{"row_count":5,"fingerprint":"` + evidenceFingerprint + `","row_fingerprints":{"evidence-1":"evidence-row-1","evidence-2":"evidence-row-2","evidence-3":"evidence-row-3","evidence-4":"evidence-row-4","evidence-5":"evidence-row-5"}}}}`
}
