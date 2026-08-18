package architecture

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUATServiceReleasePlannerUsesLastSuccessfulReleaseRange(t *testing.T) {
	repository := t.TempDir()
	runGitFixture(t, repository, "init")
	runGitFixture(t, repository, "config", "user.email", "uat-planner@example.test")
	runGitFixture(t, repository, "config", "user.name", "UAT Planner")
	for _, directory := range []string{
		"data-service", "miniapp/backend",
		"admin-portal/backend", "admin-portal/frontend", "docs",
	} {
		if err := os.MkdirAll(filepath.Join(repository, directory), 0o750); err != nil {
			t.Fatal(err)
		}
	}

	writeFixture(t, filepath.Join(repository, "data-service", "service.go"), "data-v1\n")
	writeFixture(t, filepath.Join(repository, "miniapp", "backend", "service.go"), "miniapp-v1\n")
	writeFixture(t, filepath.Join(repository, "admin-portal", "backend", "service.go"), "admin-backend-v1\n")
	writeFixture(t, filepath.Join(repository, "admin-portal", "frontend", "app.ts"), "admin-frontend-v1\n")
	writeFixture(t, filepath.Join(repository, "docs", "architecture.md"), "docs-v1\n")
	base := commitGitFixture(t, repository, "initial")

	writeFixture(t, filepath.Join(repository, "data-service", "service.go"), "data-v2\n")
	dataCommit := commitGitFixture(t, repository, "data")
	assertServicePlan(t, repository, base, dataCommit, map[string]string{
		"deploy_all": "false", "deploy_data": "true",
		"deploy_miniapp": "false", "deploy_adminportal": "false", "deploy_admin": "false",
	})

	writeFixture(t, filepath.Join(repository, "miniapp", "backend", "service.go"), "miniapp-v2\n")
	writeFixture(t, filepath.Join(repository, "admin-portal", "frontend", "app.ts"), "admin-frontend-v2\n")
	multipleCommit := commitGitFixture(t, repository, "miniapp and admin frontend")
	assertServicePlan(t, repository, dataCommit, multipleCommit, map[string]string{
		"deploy_all": "false", "deploy_data": "false",
		"deploy_miniapp": "true", "deploy_adminportal": "false", "deploy_admin": "true",
	})

	writeFixture(t, filepath.Join(repository, "docs", "architecture.md"), "docs-v2\n")
	outsideCommit := commitGitFixture(t, repository, "outside application directories")
	assertFullServicePlan(t, repository, multipleCommit, outsideCommit)
	assertFullServicePlan(t, repository, outsideCommit, outsideCommit)
	assertFullServicePlan(t, repository, "", outsideCommit)

	runGitFixture(t, repository, "checkout", "-b", "divergent", base)
	writeFixture(t, filepath.Join(repository, "miniapp", "backend", "service.go"), "miniapp-divergent\n")
	divergentCommit := commitGitFixture(t, repository, "divergent")
	assertFullServicePlan(t, repository, outsideCommit, divergentCommit)
}

func assertFullServicePlan(t *testing.T, repository, base, target string) {
	t.Helper()
	assertServicePlan(t, repository, base, target, map[string]string{
		"deploy_all": "true", "deploy_data": "true",
		"deploy_miniapp": "true", "deploy_adminportal": "true", "deploy_admin": "true",
	})
}

func assertServicePlan(t *testing.T, repository, base, target string, expected map[string]string) {
	t.Helper()
	root := repositoryRoot()
	planner, err := filepath.Abs(filepath.Join(root, "infra", "uat", "plan-service-release.sh"))
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command("bash", planner, base, target)
	command.Dir = repository
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("plan service release: %v\n%s", err, output)
	}
	actual := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if ok {
			actual[key] = value
		}
	}
	for key, value := range expected {
		if actual[key] != value {
			t.Fatalf("service plan %s = %q, want %q; output=%s", key, actual[key], value, output)
		}
	}
}

func commitGitFixture(t *testing.T, repository, message string) string {
	t.Helper()
	runGitFixture(t, repository, "add", ".")
	runGitFixture(t, repository, "commit", "-m", message)
	return strings.TrimSpace(runGitFixture(t, repository, "rev-parse", "HEAD"))
}

func runGitFixture(t *testing.T, repository string, arguments ...string) string {
	t.Helper()
	command := exec.Command("git", arguments...)
	command.Dir = repository
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(arguments, " "), err, output)
	}
	return string(output)
}

func TestUATDeployExecutorSuccessRecordsCompleteReleaseWithoutLeakingSecrets(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{})
	if result.err != nil {
		t.Fatalf("deploy success fixture failed: %v\n%s", result.err, result.output)
	}
	for _, want := range []string{"PASS deployment-lock", "PASS migration-scope-gate", "PASS migration-apply", "PASS bff-to-service-read-paths", "PASS release-state-recorded"} {
		if !strings.Contains(result.output, want) {
			t.Fatalf("deploy output missing %q: %s", want, result.output)
		}
	}
	if strings.Contains(result.output, "fixture-admin-secret") || strings.Contains(result.output, "fixture-db-secret") {
		t.Fatalf("deploy output leaked a secret: %s", result.output)
	}
	assertFileContent(t, filepath.Join(result.root, "state", "current.sha"), fixtureSHA)
	assertFileContains(t, filepath.Join(result.root, "state", "current.images.env"), "fixture/data:"+fixtureSHA)
	images, err := os.ReadFile(filepath.Join(result.root, "state", "current.images.env"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(images), "QDRANT_IMAGE") {
		t.Fatalf("application release state owns Qdrant image: %s", images)
	}
	curlLog, err := os.ReadFile(result.curlLog)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"http://127.0.0.1:9012/healthz",
		"http://127.0.0.1:9012/api/miniapp/v1/research/themes?limit=1",
		"http://127.0.0.1:9014/api/admin/v1/events?page=1&page_size=1",
	} {
		if !strings.Contains(string(curlLog), want) {
			t.Fatalf("host verification missing %q: %s", want, curlLog)
		}
	}
	if strings.Contains(string(curlLog), "uat.example.test") {
		t.Fatalf("deployment attempted unsupported public-IP hairpin verification: %s", curlLog)
	}
}

func TestUATDeployExecutorStopsBeforeDatabaseWorkWhenPlannedReleaseStateDrifts(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:     true,
		expectedCurrentSHA: fixtureSHA,
	})
	if result.err == nil || !strings.Contains(result.output, "FAIL release-state-plan-gate") {
		t.Fatalf("drifted current release state did not fail closed: %v\n%s", result.err, result.output)
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if strings.Contains(string(dockerLog), "/usr/local/bin/dbmigrate") {
		t.Fatalf("deployment performed protected work after release-state drift: %s", dockerLog)
	}
}

func TestUATDeployExecutorRequiresReplanAfterInterruptedReleaseStateRecovery(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:         true,
		releaseStateWritePhase: "committed",
		expectedCurrentMissing: true,
	})
	if result.err == nil || !strings.Contains(result.output, "FAIL release-state-plan-gate") {
		t.Fatalf("recovered state did not require a fresh deployment plan: %v\n%s", result.err, result.output)
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if strings.Contains(string(dockerLog), "/usr/local/bin/dbmigrate") {
		t.Fatalf("deployment started migration after interrupted-state recovery: %s", dockerLog)
	}
}

func TestUATDeployExecutorReplacesUnchangedInvalidCurrentReleaseWithFullDeployment(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:         true,
		invalidCurrentRelease:  true,
		expectedCurrentMissing: true,
		legacyAgentRunMarkers:  true,
	})
	if result.err != nil {
		t.Fatalf("unchanged invalid current release did not allow full replacement: %v\n%s", result.err, result.output)
	}
	if !strings.Contains(result.output, "PASS release-state-plan-gate") ||
		!strings.Contains(result.output, "PASS release-state-recorded") {
		t.Fatalf("invalid state replacement missed release-state evidence: %s", result.output)
	}
	assertFileContent(t, filepath.Join(result.root, "state", "current.sha"), fixtureSHA)
	for _, retiredMarker := range []string{
		"agentrun-010-rollback-required",
		"agentrun-agent-version-publication.json",
	} {
		if _, err := os.Stat(filepath.Join(result.root, "state", retiredMarker)); !os.IsNotExist(err) {
			t.Fatalf("retired AgentRun release marker %q survived replacement: %v", retiredMarker, err)
		}
	}
}

func TestUATDeployExecutorDoesNotRollbackToUnavailableInvalidRelease(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:         true,
		invalidCurrentRelease:  true,
		expectedCurrentMissing: true,
		failFirstUp:            true,
	})
	if result.err == nil || !strings.Contains(result.output, "FAIL rollback: no trusted previous") {
		t.Fatalf("candidate failure did not reject invalid rollback state: %v\n%s", result.err, result.output)
	}
	if strings.Contains(result.output, "PASS rollback: previous complete release restored") {
		t.Fatalf("invalid unavailable state was reported as restored: %s", result.output)
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
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(dockerLog), "exec -T qdrant") ||
		strings.Contains(string(dockerLog), " up -d --wait --wait-timeout 120 qdrant ") {
		t.Fatalf("rollback attempted to manage independently operated Qdrant: %s", dockerLog)
	}
}

func TestUATDeployExecutorDoesNotReportPassWhenRollbackStartFails(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{currentRelease: true, failEveryUp: true})
	if result.err == nil {
		t.Fatal("failed candidate and rollback fixture unexpectedly succeeded")
	}
	if !strings.Contains(result.output, "FAIL rollback-start") {
		t.Fatalf("rollback start failure was not reported: %s", result.output)
	}
	if strings.Contains(result.output, "PASS rollback:") {
		t.Fatalf("failed rollback reported success: %s", result.output)
	}
}

func TestUATDeployExecutorDoesNotReportPassWhenRollbackHealthFails(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease: true,
		failFirstUp:    true,
		failEveryCurl:  true,
	})
	if result.err == nil {
		t.Fatal("unhealthy rollback fixture unexpectedly succeeded")
	}
	if !strings.Contains(result.output, "FAIL rollback-health") {
		t.Fatalf("rollback health failure was not reported: %s", result.output)
	}
	if strings.Contains(result.output, "PASS rollback:") {
		t.Fatalf("unhealthy rollback reported success: %s", result.output)
	}
}

func TestUATDeployExecutorRejectsQdrantOwningRollbackSnapshotBeforeDatabaseWork(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:       true,
		legacyQdrantSnapshot: true,
	})
	if result.err == nil {
		t.Fatal("legacy Qdrant-owning rollback snapshot unexpectedly succeeded")
	}
	if !strings.Contains(result.output, "FAIL rollback-qdrant-ownership") {
		t.Fatalf("rollback did not reject Qdrant ownership: %s", result.output)
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	logText := string(dockerLog)
	if strings.Contains(logText, "/usr/local/bin/dbmigrate") {
		t.Fatalf("deployment reached database work without an application-only rollback snapshot: %s", logText)
	}
	if strings.Contains(logText, " exec -T qdrant ") ||
		strings.Contains(logText, " up -d --wait --wait-timeout 120 qdrant ") ||
		strings.Contains(logText, " --remove-orphans ") {
		t.Fatalf("rollback attempted to manage Qdrant: %s", logText)
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
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if strings.Contains(string(logContent), " up ") {
		t.Fatalf("high-risk gate started services: %s", logContent)
	}
}

func TestUATDeployExecutorAppliesConfirmedHighRiskSchemaMigration(t *testing.T) {
	report := `{"current_version":"23","pending":[{"Version":"24","Name":"add schema contract"}],"applied":[],"remaining":[]}`
	result := runDeployFixture(t, deployFixtureOptions{
		migrationReport: report,
		migrationScope:  "schema",
		backupConfirmed: true,
	})
	if result.err != nil {
		t.Fatalf("confirmed high-risk schema migration failed: %v\n%s", result.err, result.output)
	}
	for _, want := range []string{"PASS migration-scope-gate", "PASS migration-risk-gate", "PASS migration-apply"} {
		if !strings.Contains(result.output, want) {
			t.Fatalf("schema migration output missing %q: %s", want, result.output)
		}
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dockerLog), "dbmigrate -apply") {
		t.Fatalf("confirmed schema migration was not applied: %s", dockerLog)
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
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if strings.Contains(string(logContent), " up ") {
		t.Fatalf("release gate started services: %s", logContent)
	}
}

func TestUATDeployExecutorBlocksDataPublicationMigrationsBeforeApply(t *testing.T) {
	tests := []struct {
		name    string
		options deployFixtureOptions
	}{
		{
			name: "Data data-only migration",
			options: deployFixtureOptions{
				migrationReport: `{"current_version":"23","pending":[{"Version":"24","Name":"publish research data"}],"applied":[],"remaining":[]}`,
				migrationScope:  "data",
				backupConfirmed: true,
			},
		},
		{
			name: "Data mixed migration",
			options: deployFixtureOptions{
				migrationReport: `{"current_version":"23","pending":[{"Version":"24","Name":"schema and data backfill"}],"applied":[],"remaining":[]}`,
				migrationScope:  "mixed",
				backupConfirmed: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := runDeployFixture(t, test.options)
			if result.err == nil || !strings.Contains(result.output, "FAIL migration-scope-gate") {
				t.Fatalf("data publication migration was not blocked: err=%v output=%s", result.err, result.output)
			}
			dockerLog, err := os.ReadFile(result.dockerLog)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(dockerLog), "dbmigrate -apply") {
				t.Fatalf("scope gate allowed migration apply: %s", dockerLog)
			}
			if strings.Contains(string(dockerLog), " up ") {
				t.Fatalf("scope gate started application services: %s", dockerLog)
			}
		})
	}
}

func TestUATDeployExecutorRunsBoundedData2CutoverWithAllWritersStopped(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:       true,
		deploymentMode:       "tidewise_2_cutover",
		destructiveConfirmed: true,
		backupConfirmed:      true,
		migrationReport:      data2CutoverPendingReport(),
		migrationApplyReport: `{"current_version":"58","pending":[],"applied":[],"remaining":[]}`,
	})
	if result.err != nil {
		t.Fatalf("Data 2.0 cutover fixture failed: %v\n%s", result.err, result.output)
	}
	for _, want := range []string{
		"PASS data2-cutover-gate",
		"PASS application-write-stop",
		"PASS data2-target-version",
		"PASS release-state-recorded",
	} {
		if !strings.Contains(result.output, want) {
			t.Fatalf("cutover output missing %q: %s", want, result.output)
		}
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	logText := string(dockerLog)
	stop := strings.Index(logText, " stop ")
	apply := strings.Index(logText, "dbmigrate -apply -target-version 58")
	start := strings.Index(logText, " up -d --remove-orphans --wait --wait-timeout 120")
	if stop < 0 || apply < 0 || start < 0 || stop > apply || apply > start {
		t.Fatalf("cutover must stop writers before the bounded migration and start candidates afterward: %s", logText)
	}
	if _, err := os.Stat(filepath.Join(result.root, "state", "tidewise-2-cutover-in-progress")); !os.IsNotExist(err) {
		t.Fatalf("successful cutover retained recovery marker: %v", err)
	}
	assertFileContent(t, filepath.Join(result.root, "state", "current.sha"), fixtureSHA)
}

func TestUATDeployExecutorRunsBoundedData59CutoverWithAllWritersStopped(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:           true,
		existingPreData2Snapshot: true,
		deploymentMode:           "data_59_cutover",
		destructiveConfirmed:     true,
		backupConfirmed:          true,
		migrationReport:          `{"current_version":"58","pending":[{"Version":"59"}],"applied":[],"remaining":[]}`,
		migrationApplyReport:     `{"current_version":"59","pending":[],"applied":[],"remaining":[]}`,
	})
	if result.err != nil {
		t.Fatalf("Data 59 cutover fixture failed: %v\n%s", result.err, result.output)
	}
	for _, want := range []string{
		"PASS data59-cutover-gate",
		"PASS application-write-stop",
		"PASS data59-target-version",
		"PASS release-state-recorded",
	} {
		if !strings.Contains(result.output, want) {
			t.Fatalf("Data 59 cutover output missing %q: %s", want, result.output)
		}
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	logText := string(dockerLog)
	stop := strings.Index(logText, " stop ")
	apply := strings.Index(logText, "dbmigrate -apply -target-version 59")
	start := strings.Index(logText, " up -d --remove-orphans --wait --wait-timeout 120")
	if stop < 0 || apply < 0 || start < 0 || stop > apply || apply > start {
		t.Fatalf("Data 59 cutover must stop writers before migration and start candidates afterward: %s", logText)
	}
	if _, err := os.Stat(filepath.Join(result.root, "state", "tidewise-2-cutover-in-progress")); !os.IsNotExist(err) {
		t.Fatalf("successful Data 59 cutover retained recovery marker: %v", err)
	}
	assertFileContent(t, filepath.Join(result.root, "state", "pre-data2.sha"), historicalPreData2FixtureSHA)
	assertFileContent(t, filepath.Join(result.root, "state", "pre-data59.sha"), previousFixtureSHA)
}

func TestUATDeployExecutorRejectsUnexpectedData59MigrationRangeBeforeStoppingServices(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:       true,
		deploymentMode:       "data_59_cutover",
		destructiveConfirmed: true,
		backupConfirmed:      true,
		migrationReport:      `{"current_version":"57","pending":[{"Version":"58"},{"Version":"59"}],"applied":[],"remaining":[]}`,
	})
	if result.err == nil || !strings.Contains(result.output, "FAIL data59-cutover-gate") {
		t.Fatalf("unexpected Data 59 migration range was not blocked: err=%v output=%s", result.err, result.output)
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(dockerLog), " stop ") || strings.Contains(string(dockerLog), "dbmigrate -apply") {
		t.Fatalf("invalid Data 59 cutover reached destructive work: %s", dockerLog)
	}
}

func TestUATDeployExecutorRequiresData59CutoverConfirmations(t *testing.T) {
	for _, test := range []struct {
		name                   string
		destructiveConfirmed   bool
		backupConfirmed        bool
		expectedFailureMessage string
	}{
		{name: "destructive change", backupConfirmed: true, expectedFailureMessage: "confirm_destructive_data_change=true is required"},
		{name: "recovery point", destructiveConfirmed: true, expectedFailureMessage: "confirm_high_risk_backup=true is required"},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := runDeployFixture(t, deployFixtureOptions{
				currentRelease:       true,
				deploymentMode:       "data_59_cutover",
				destructiveConfirmed: test.destructiveConfirmed,
				backupConfirmed:      test.backupConfirmed,
				migrationReport:      `{"current_version":"58","pending":[{"Version":"59"}],"applied":[],"remaining":[]}`,
			})
			if result.err == nil || !strings.Contains(result.output, test.expectedFailureMessage) {
				t.Fatalf("missing Data 59 confirmation was not blocked: err=%v output=%s", result.err, result.output)
			}
		})
	}
}

func TestUATDeployExecutorNeverRestartsOldImagesAfterData59MigrationStarts(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:       true,
		deploymentMode:       "data_59_cutover",
		destructiveConfirmed: true,
		backupConfirmed:      true,
		migrationReport:      `{"current_version":"58","pending":[{"Version":"59"}],"applied":[],"remaining":[]}`,
		migrationApplyReport: `{"current_version":"59","pending":[],"applied":[],"remaining":[]}`,
		failFirstUp:          true,
	})
	if result.err == nil {
		t.Fatal("failed Data 59 candidate fixture unexpectedly succeeded")
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(dockerLog), "\n") {
		if strings.Contains(line, "resolved-data-image=fixture/data:"+previousFixtureSHA) && strings.Contains(line, " up ") {
			t.Fatalf("Data 59 cutover restarted the incompatible old Data image: %s", dockerLog)
		}
	}
	marker := filepath.Join(result.root, "state", "tidewise-2-cutover-in-progress")
	assertFileContains(t, marker, "phase=data-migrated")
	assertFileContains(t, marker, "target_version=59")
}

func TestUATDeployExecutorResumesSameReleaseData59Cutover(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:             true,
		deploymentMode:             "data_59_cutover",
		destructiveConfirmed:       true,
		backupConfirmed:            true,
		cutoverMarkerPhase:         "migration-started",
		cutoverMarkerTargetVersion: "59",
		migrationReport:            `{"current_version":"58","pending":[{"Version":"59"}],"applied":[],"remaining":[]}`,
		migrationApplyReport:       `{"current_version":"59","pending":[],"applied":[],"remaining":[]}`,
	})
	if result.err != nil || !strings.Contains(result.output, "PASS data59-cutover-recovery-gate") {
		t.Fatalf("same-release Data 59 forward recovery failed: err=%v output=%s", result.err, result.output)
	}
}

func TestUATDeployExecutorRecoversInterruptedData59ReleaseStateWrite(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:             true,
		releaseStateWritePhase:     "pre-data59",
		deploymentMode:             "data_59_cutover",
		destructiveConfirmed:       true,
		backupConfirmed:            true,
		cutoverMarkerPhase:         "data-migrated",
		cutoverMarkerTargetVersion: "59",
		migrationReport:            `{"current_version":"59","pending":[],"applied":[],"remaining":[]}`,
		migrationApplyReport:       `{"current_version":"59","pending":[],"applied":[],"remaining":[]}`,
	})
	if result.err != nil {
		t.Fatalf("interrupted Data 59 release-state recovery failed: %v\n%s", result.err, result.output)
	}
	for _, want := range []string{"PASS recovered-interrupted-release-state", "PASS data59-cutover-recovery-gate", "PASS release-state-recorded"} {
		if !strings.Contains(result.output, want) {
			t.Fatalf("interrupted Data 59 recovery output missing %q: %s", want, result.output)
		}
	}
}

func TestUATDeployExecutorTreatsAdvancedData59LedgerAsPostMigrationBeforeWriterStop(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:             true,
		deploymentMode:             "data_59_cutover",
		destructiveConfirmed:       true,
		backupConfirmed:            true,
		cutoverMarkerPhase:         "prepared",
		cutoverMarkerTargetVersion: "59",
		migrationReport:            `{"current_version":"59","pending":[],"applied":[],"remaining":[]}`,
		migrationApplyReport:       `{"current_version":"59","pending":[],"applied":[],"remaining":[]}`,
		failRunningServiceProbe:    true,
	})
	if result.err == nil || !strings.Contains(result.output, "FAIL application-write-stop") {
		t.Fatalf("advanced Data 59 ledger did not fail closed at writer-stop proof: err=%v output=%s", result.err, result.output)
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(dockerLog), "\n") {
		if strings.Contains(line, "resolved-data-image=fixture/data:"+previousFixtureSHA) && strings.Contains(line, " up ") {
			t.Fatalf("advanced Data 59 ledger restarted migration-58 images: %s", dockerLog)
		}
	}
	marker := filepath.Join(result.root, "state", "tidewise-2-cutover-in-progress")
	assertFileContains(t, marker, "phase=migration-started")
}

func TestUATDeployExecutorStopsOldServicesWhenAdvancedData59MarkerRewriteFails(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:             true,
		deploymentMode:             "data_59_cutover",
		destructiveConfirmed:       true,
		backupConfirmed:            true,
		cutoverMarkerPhase:         "prepared",
		cutoverMarkerTargetVersion: "59",
		migrationReport:            `{"current_version":"59","pending":[],"applied":[],"remaining":[]}`,
		migrationApplyReport:       `{"current_version":"59","pending":[],"applied":[],"remaining":[]}`,
		failCutoverMarkerSync:      true,
	})
	if result.err == nil {
		t.Fatalf("advanced Data 59 marker rewrite failure unexpectedly succeeded: %s", result.output)
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dockerLog), " stop ") {
		t.Fatalf("advanced Data 59 marker rewrite failure left old services running: %s", dockerLog)
	}
	for _, line := range strings.Split(string(dockerLog), "\n") {
		if strings.Contains(line, "resolved-data-image=fixture/data:"+previousFixtureSHA) && strings.Contains(line, " up ") {
			t.Fatalf("advanced Data 59 marker rewrite failure restarted migration-58 images: %s", dockerLog)
		}
	}
}

func TestUATDeployExecutorNeverRestartsOldImagesAfterData2MigrationStarts(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:       true,
		deploymentMode:       "tidewise_2_cutover",
		destructiveConfirmed: true,
		backupConfirmed:      true,
		migrationReport:      data2CutoverPendingReport(),
		migrationApplyReport: `{"current_version":"58","pending":[],"applied":[],"remaining":[]}`,
		failFirstUp:          true,
	})
	if result.err == nil {
		t.Fatal("failed Data 2.0 candidate fixture unexpectedly succeeded")
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	logText := string(dockerLog)
	oldReleaseStarted := false
	for _, line := range strings.Split(logText, "\n") {
		if strings.Contains(line, "resolved-data-image=fixture/data:"+previousFixtureSHA) && strings.Contains(line, " up ") {
			oldReleaseStarted = true
		}
	}
	if oldReleaseStarted {
		t.Fatalf("cutover restarted the incompatible old Data image: %s", logText)
	}
	candidateStopped := false
	for _, line := range strings.Split(logText, "\n") {
		if strings.Contains(line, "resolved-data-image=fixture/data:"+fixtureSHA) && strings.Contains(line, " stop ") {
			candidateStopped = true
		}
	}
	if strings.Count(logText, " stop ") < 2 || !candidateStopped {
		t.Fatalf("cutover failure did not stop the candidate application services: %s", logText)
	}
	marker := filepath.Join(result.root, "state", "tidewise-2-cutover-in-progress")
	assertFileContains(t, marker, "release_sha="+fixtureSHA)
	assertFileContains(t, marker, "phase=data-migrated")
}

func TestUATDeployExecutorRejectsUnexpectedData2MigrationRangeBeforeStoppingServices(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:       true,
		deploymentMode:       "tidewise_2_cutover",
		destructiveConfirmed: true,
		backupConfirmed:      true,
		migrationReport:      `{"current_version":"45","pending":[{"Version":"46"}],"applied":[],"remaining":[]}`,
	})
	if result.err == nil || !strings.Contains(result.output, "FAIL data2-cutover-gate") {
		t.Fatalf("unexpected Data migration range was not blocked: err=%v output=%s", result.err, result.output)
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(dockerLog), " stop ") || strings.Contains(string(dockerLog), "dbmigrate -apply") {
		t.Fatalf("invalid cutover reached destructive work: %s", dockerLog)
	}
}

func TestUATDeployExecutorResumesSameReleaseFromContiguousData2Suffix(t *testing.T) {
	pending := `{"current_version":"50","pending":[{"Version":"51"},{"Version":"52"},{"Version":"53"},{"Version":"54"},{"Version":"55"},{"Version":"56"},{"Version":"57"},{"Version":"58"}],"applied":[],"remaining":[]}`
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:       true,
		deploymentMode:       "tidewise_2_cutover",
		destructiveConfirmed: true,
		backupConfirmed:      true,
		cutoverMarkerPhase:   "migration-started",
		migrationReport:      pending,
		migrationApplyReport: `{"current_version":"58","pending":[],"applied":[],"remaining":[]}`,
	})
	if result.err != nil || !strings.Contains(result.output, "PASS data2-cutover-recovery-gate") {
		t.Fatalf("same-release forward recovery failed: err=%v output=%s", result.err, result.output)
	}
}

func TestUATDeployExecutorRecoversInterruptedData2ReleaseStateWrite(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:         true,
		releaseStateWritePhase: "pre-data2",
		deploymentMode:         "tidewise_2_cutover",
		destructiveConfirmed:   true,
		backupConfirmed:        true,
		cutoverMarkerPhase:     "data-migrated",
		migrationReport:        `{"current_version":"58","pending":[],"applied":[],"remaining":[]}`,
		migrationApplyReport:   `{"current_version":"58","pending":[],"applied":[],"remaining":[]}`,
	})
	if result.err != nil {
		t.Fatalf("interrupted cutover release-state recovery failed: %v\n%s", result.err, result.output)
	}
	for _, want := range []string{"PASS recovered-interrupted-release-state", "PASS data2-cutover-recovery-gate", "PASS release-state-recorded"} {
		if !strings.Contains(result.output, want) {
			t.Fatalf("interrupted cutover recovery output missing %q: %s", want, result.output)
		}
	}
}

func TestUATDeployExecutorFinalizesCommittedData2ReleaseStateWithoutReapplying(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:          true,
		currentReleaseCandidate: true,
		releaseStateWritePhase:  "committed",
		deploymentMode:          "tidewise_2_cutover",
		destructiveConfirmed:    true,
		backupConfirmed:         true,
		cutoverMarkerPhase:      "data-migrated",
		migrationReport:         `{"current_version":"58","pending":[],"applied":[],"remaining":[]}`,
	})
	if result.err != nil || !strings.Contains(result.output, "PASS data2-cutover-committed-recovery") {
		t.Fatalf("committed cutover finalization failed: err=%v output=%s", result.err, result.output)
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if strings.Contains(string(dockerLog), "dbmigrate -apply") || strings.Contains(string(dockerLog), " stop ") {
		t.Fatalf("committed cutover recovery repeated deployment work: %s", dockerLog)
	}
	if _, err := os.Stat(filepath.Join(result.root, "state", "tidewise-2-cutover-in-progress")); !os.IsNotExist(err) {
		t.Fatalf("committed cutover recovery retained cutover marker: %v", err)
	}
}

func TestUATDeployExecutorRejectsCommittedStateWithoutMatchingCutoverMarker(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:          true,
		currentReleaseCandidate: true,
		releaseStateWritePhase:  "committed",
		deploymentMode:          "tidewise_2_cutover",
		destructiveConfirmed:    true,
		backupConfirmed:         true,
	})
	if result.err == nil || !strings.Contains(result.output, "matching data-migrated cutover marker is required") {
		t.Fatalf("generic committed state impersonated cutover completion: err=%v output=%s", result.err, result.output)
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if strings.Contains(string(dockerLog), "dbmigrate") {
		t.Fatalf("invalid committed recovery reached database work: %s", dockerLog)
	}
}

func TestUATDeployExecutorRejectsCommittedCutoverBeforeDataReaches58(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:          true,
		currentReleaseCandidate: true,
		releaseStateWritePhase:  "committed",
		deploymentMode:          "tidewise_2_cutover",
		destructiveConfirmed:    true,
		backupConfirmed:         true,
		cutoverMarkerPhase:      "data-migrated",
		migrationReport:         `{"current_version":"50","pending":[{"Version":"51"},{"Version":"52"},{"Version":"53"},{"Version":"54"},{"Version":"55"},{"Version":"56"},{"Version":"57"},{"Version":"58"}],"applied":[],"remaining":[]}`,
	})
	if result.err == nil || !strings.Contains(result.output, "Data must be at migration 58 with no pending migrations") {
		t.Fatalf("incomplete Data ledger impersonated committed cutover: err=%v output=%s", result.err, result.output)
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(dockerLog), "dbmigrate -apply") {
		t.Fatalf("invalid committed recovery attempted migration after state was declared committed: %s", dockerLog)
	}
}

func TestUATDeployExecutorFailsClosedWhenWriterStopCannotBeProved(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:          true,
		deploymentMode:          "tidewise_2_cutover",
		destructiveConfirmed:    true,
		backupConfirmed:         true,
		migrationReport:         data2CutoverPendingReport(),
		migrationApplyReport:    `{"current_version":"58","pending":[],"applied":[],"remaining":[]}`,
		failRunningServiceProbe: true,
	})
	if result.err == nil || !strings.Contains(result.output, "FAIL application-write-stop") {
		t.Fatalf("failed writer-stop inspection did not block cutover: err=%v output=%s", result.err, result.output)
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(dockerLog), "dbmigrate -apply") {
		t.Fatalf("cutover migrated after writer-stop inspection failed: %s", dockerLog)
	}
}

func TestUATDeployExecutorNeverRestartsOldImagesWhenRecoveryWriterStopFails(t *testing.T) {
	for _, markerPhase := range []string{"migration-started", "data-migrated"} {
		t.Run(markerPhase, func(t *testing.T) {
			result := runDeployFixture(t, deployFixtureOptions{
				currentRelease:          true,
				deploymentMode:          "tidewise_2_cutover",
				destructiveConfirmed:    true,
				backupConfirmed:         true,
				cutoverMarkerPhase:      markerPhase,
				migrationReport:         data2CutoverPendingReport(),
				migrationApplyReport:    `{"current_version":"58","pending":[],"applied":[],"remaining":[]}`,
				failRunningServiceProbe: true,
			})
			if result.err == nil || !strings.Contains(result.output, "FAIL application-write-stop") {
				t.Fatalf("failed recovery writer-stop inspection did not block cutover: err=%v output=%s", result.err, result.output)
			}
			dockerLog, err := os.ReadFile(result.dockerLog)
			if err != nil {
				t.Fatal(err)
			}
			for _, line := range strings.Split(string(dockerLog), "\n") {
				if strings.Contains(line, "resolved-data-image=fixture/data:"+previousFixtureSHA) && strings.Contains(line, " up ") {
					t.Fatalf("post-migration recovery failure restarted the old Data image: %s", dockerLog)
				}
			}
			marker := filepath.Join(result.root, "state", "tidewise-2-cutover-in-progress")
			assertFileContains(t, marker, "phase="+markerPhase)
		})
	}
}

func TestUATDeployExecutorBlocksNormalModeWhileData2RecoveryMarkerExists(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:     true,
		cutoverMarkerPhase: "data-migrated",
	})
	if result.err == nil || !strings.Contains(result.output, "FAIL data2-cutover-recovery") {
		t.Fatalf("normal deployment ignored Data 2.0 recovery marker: err=%v output=%s", result.err, result.output)
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(dockerLog), "dbmigrate -apply") || strings.Contains(string(dockerLog), " up ") {
		t.Fatalf("normal deployment performed writes while a cutover marker exists: %s", dockerLog)
	}
}

func TestUATDeployExecutorRebuildsOnlyDataSchemaAsExplicitCutoverFallback(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:         true,
		deploymentMode:         "tidewise_2_cutover",
		destructiveConfirmed:   true,
		backupConfirmed:        true,
		rebuildEmptyDataSchema: true,
		cutoverMarkerPhase:     "migration-started",
		migrationReport:        data2EmptySchemaPendingReport(),
		migrationApplyReport:   `{"current_version":"58","pending":[],"applied":[],"remaining":[]}`,
	})
	if result.err != nil {
		t.Fatalf("explicit empty Data schema fallback failed: %v\n%s", result.err, result.output)
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	logText := string(dockerLog)
	if !strings.Contains(logText, "dbmigrate -apply -target-version 58 -rebuild-empty-schema") {
		t.Fatalf("fallback did not use the bounded Data schema rebuild entrypoint: %s", logText)
	}
	for _, authorization := range []string{
		"tidewise.phase_a_cleanup_write_authorized=reviewed_backup_verified",
		"tidewise.external_identifier_schema_write_authorized=reviewed_backup_verified",
		"tidewise.alliance_economy_schema_write_authorized=reviewed_local_cleanup_verified",
	} {
		if !strings.Contains(logText, authorization) {
			t.Fatalf("fallback omitted historical empty-schema authorization %q: %s", authorization, logText)
		}
	}
	if strings.Contains(logText, "qdrant") && strings.Contains(logText, " down ") {
		t.Fatalf("Data fallback attempted to rebuild independently owned runtime state: %s", logText)
	}
}

func TestUATDeployExecutorStopsOrphanedCutoverWriterBeforeEmptySchemaRecovery(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:         true,
		deploymentMode:         "tidewise_2_cutover",
		destructiveConfirmed:   true,
		backupConfirmed:        true,
		rebuildEmptyDataSchema: true,
		cutoverMarkerPhase:     "migration-started",
		migrationReport:        data2EmptySchemaPendingReport(),
		migrationApplyReport:   `{"current_version":"58","pending":[],"applied":[],"remaining":[]}`,
		orphanedCutoverWriter:  true,
	})
	if result.err != nil {
		t.Fatalf("empty-schema recovery did not stop the orphaned cutover writer: %v\n%s", result.err, result.output)
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	logText := string(dockerLog)
	writerProbe := strings.Index(logText, "label=com.docker.compose.service=data")
	writerStop := strings.Index(logText, "stop orphaned-cutover-writer")
	rebuild := strings.Index(logText, "dbmigrate -apply -target-version 58 -rebuild-empty-schema")
	if writerProbe < 0 || writerStop < 0 || rebuild < 0 || writerProbe > writerStop || writerStop > rebuild {
		t.Fatalf("recovery must stop the orphaned writer before rebuilding Data: %s", logText)
	}
}

func TestUATDeployExecutorStopsRestartingCutoverWriterBeforeEmptySchemaRecovery(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:                  true,
		deploymentMode:                  "tidewise_2_cutover",
		destructiveConfirmed:            true,
		backupConfirmed:                 true,
		rebuildEmptyDataSchema:          true,
		cutoverMarkerPhase:              "migration-started",
		migrationReport:                 data2EmptySchemaPendingReport(),
		migrationApplyReport:            `{"current_version":"58","pending":[],"applied":[],"remaining":[]}`,
		orphanedRestartingCutoverWriter: true,
	})
	if result.err != nil {
		t.Fatalf("empty-schema recovery did not stop the restarting cutover writer: %v\n%s", result.err, result.output)
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	logText := string(dockerLog)
	writerProbe := strings.Index(logText, "ps --all --filter label=com.docker.compose.project=tidewise-uat")
	writerStop := strings.Index(logText, "stop restarting-cutover-writer")
	rebuild := strings.Index(logText, "dbmigrate -apply -target-version 58 -rebuild-empty-schema")
	if writerProbe < 0 || writerStop < 0 || rebuild < 0 || writerProbe > writerStop || writerStop > rebuild {
		t.Fatalf("recovery must enumerate and stop the restarting writer before rebuilding Data: %s", logText)
	}
}

func TestUATDeployExecutorRejectsEmptyDataSchemaRebuildWithoutRecoveryMarker(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:         true,
		deploymentMode:         "tidewise_2_cutover",
		destructiveConfirmed:   true,
		backupConfirmed:        true,
		rebuildEmptyDataSchema: true,
		migrationReport:        data2EmptySchemaPendingReport(),
	})
	if result.err == nil || !strings.Contains(result.output, "empty Data schema rebuild requires an existing cutover recovery marker") {
		t.Fatalf("unmarked Data schema rebuild was not blocked: err=%v output=%s", result.err, result.output)
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(dockerLog), "-rebuild-empty-schema") {
		t.Fatalf("unmarked fallback reached the destructive entrypoint: %s", dockerLog)
	}
}

func data2EmptySchemaPendingReport() string {
	versions := make([]string, 0, 58)
	for version := 1; version <= 58; version++ {
		versions = append(versions, fmt.Sprintf(`{"Version":"%d"}`, version))
	}
	return `{"current_version":"0","pending":[` + strings.Join(versions, ",") + `],"applied":[],"remaining":[]}`
}

func data2CutoverPendingReport() string {
	versions := make([]string, 0, 14)
	for version := 45; version <= 58; version++ {
		versions = append(versions, fmt.Sprintf(`{"Version":"%d"}`, version))
	}
	return `{"current_version":"44","pending":[` + strings.Join(versions, ",") + `],"applied":[],"remaining":[]}`
}

func TestUATDiagnosticsRedactsCredentials(t *testing.T) {
	repoRoot := repositoryRoot()
	temp := t.TempDir()
	bin := filepath.Join(temp, "bin")
	if err := os.MkdirAll(bin, 0o750); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(bin, "docker"), `#!/bin/sh
echo 'Authorization: Bearer visible-token'
echo 'DATABASE_URL=postgres://data:visible-password@rds.internal:5432/uat'
echo 'ADMIN_SERVICE_TOKEN=visible-admin-token'
echo 'EMBEDDING_API_KEY=visible-embedding-key'
`)
	runtimeEnv := filepath.Join(temp, "runtime.env")
	imagesEnv := filepath.Join(temp, "images.env")
	writeFixture(t, runtimeEnv, "ADMIN_SERVICE_TOKEN=fixture\n")
	writeFixture(t, imagesEnv, "DATA_IMAGE=fixture\n")
	cmd := exec.Command("bash", filepath.Join(repoRoot, "infra", "uat", "collect-diagnostics.sh"))
	cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "RUNTIME_ENV="+runtimeEnv, "CANDIDATE_IMAGES="+imagesEnv, "COMPOSE_FILE=fixture.yaml")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("collect diagnostics: %v: %s", err, output)
	}
	for _, secret := range []string{"visible-token", "visible-password", "visible-admin-token", "visible-embedding-key"} {
		if strings.Contains(string(output), secret) {
			t.Fatalf("diagnostics leaked %q: %s", secret, output)
		}
	}
	if !strings.Contains(string(output), "***") {
		t.Fatalf("diagnostics did not show redaction marker: %s", output)
	}
}

const (
	fixtureSHA                   = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	previousFixtureSHA           = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	historicalPreData2FixtureSHA = "dddddddddddddddddddddddddddddddddddddddd"
	fixtureRelationshipPkgSHA    = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
)

type deployFixtureOptions struct {
	currentRelease                  bool
	existingPreData2Snapshot        bool
	currentReleaseCandidate         bool
	expectedCurrentSHA              string
	expectedCurrentMissing          bool
	releaseStateWritePhase          string
	invalidCurrentRelease           bool
	failFirstUp                     bool
	failEveryUp                     bool
	failFirstCurl                   bool
	failEveryCurl                   bool
	migrationReport                 string
	migrationApplyReport            string
	migrationRisk                   string
	migrationScope                  string
	backupConfirmed                 bool
	deploymentMode                  string
	destructiveConfirmed            bool
	cutoverMarkerPhase              string
	cutoverMarkerTargetVersion      string
	failCutoverMarkerSync           bool
	failRunningServiceProbe         bool
	rebuildEmptyDataSchema          bool
	orphanedCutoverWriter           bool
	orphanedRestartingCutoverWriter bool
	legacyQdrantSnapshot            bool
	legacyAgentRunMarkers           bool
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
	repoRoot := repositoryRoot()
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
	orphanedWriter := filepath.Join(temp, "orphaned-cutover-writer")
	restartingWriter := filepath.Join(temp, "restarting-cutover-writer")
	if options.orphanedCutoverWriter {
		writeFixture(t, orphanedWriter, "running\n")
	}
	if options.orphanedRestartingCutoverWriter {
		writeFixture(t, restartingWriter, "restarting\n")
	}
	writeFixture(t, runtimeEnv, "ADMIN_SERVICE_TOKEN=fixture-admin-secret\n")
	writeFixture(t, imagesEnv, "DATA_IMAGE=fixture/data:"+fixtureSHA+"\nMINIAPP_IMAGE=fixture/miniapp:"+fixtureSHA+"\nADMINPORTAL_IMAGE=fixture/adminportal:"+fixtureSHA+"\nADMIN_IMAGE=fixture/admin:"+fixtureSHA+"\n")
	composeContent := "name: tidewise-uat\nservices:\n  data: {}\n  miniapp: {}\n  adminportal: {}\n  admin: {}\n"
	writeFixture(t, compose, composeContent)
	if options.legacyAgentRunMarkers {
		writeFixture(t, filepath.Join(state, "agentrun-010-rollback-required"), "legacy rollback state\n")
		writeFixture(t, filepath.Join(state, "agentrun-agent-version-publication.json"), "{}\n")
	}
	migrationRisk := options.migrationRisk
	if migrationRisk == "" {
		migrationRisk = "high"
	}
	migrationScope := options.migrationScope
	if migrationScope == "" {
		migrationScope = "schema"
	}
	manifestRows := ""
	for version := 1; version <= 59; version++ {
		risk := "normal"
		scope := "schema"
		reason := "fixture migration"
		if version == 24 {
			risk = "high"
			scope = migrationScope
			reason = "fixture high risk"
		}
		if version == 25 {
			risk = migrationRisk
			scope = migrationScope
			reason = "fixture migration risk"
		}
		if version >= 45 {
			risk = "high"
			scope = "mixed"
			reason = "fixture Data 2.0 cutover migration"
			if version == 52 {
				scope = "data"
			}
			if version == 53 || version == 54 {
				scope = "schema"
			}
		}
		if version == 59 {
			risk = "high"
			scope = "mixed"
			reason = "fixture Data 59 cutover migration"
		}
		manifestRows += fmt.Sprintf("%06d\t%s\t%s\t%s\n", version, risk, scope, reason)
	}
	writeFixture(t, manifest, manifestRows)
	if options.currentRelease {
		writeFixture(t, filepath.Join(root, "runtime.env"), "ADMIN_SERVICE_TOKEN=previous-admin-secret\n")
		currentReleaseSHA := previousFixtureSHA
		if options.currentReleaseCandidate {
			currentReleaseSHA = fixtureSHA
		}
		currentImages := "DATA_IMAGE=fixture/data:" + currentReleaseSHA + "\n" +
			"MINIAPP_IMAGE=fixture/miniapp:" + currentReleaseSHA + "\n" +
			"ADMINPORTAL_IMAGE=fixture/adminportal:" + currentReleaseSHA + "\n" +
			"ADMIN_IMAGE=fixture/admin:" + currentReleaseSHA + "\n"
		currentCompose := composeContent
		if options.legacyQdrantSnapshot {
			currentCompose += "  qdrant: {}\n"
		}
		writeFixture(t, filepath.Join(state, "current.images.env"), currentImages)
		writeFixture(t, filepath.Join(state, "current.compose.yaml"), currentCompose)
		currentSHA := currentReleaseSHA
		if options.invalidCurrentRelease {
			currentSHA = "not-a-release-sha"
		}
		writeFixture(t, filepath.Join(state, "current.sha"), currentSHA+"\n")
		if options.existingPreData2Snapshot {
			writeFixture(t, filepath.Join(root, "pre-data2.runtime.env"), "ADMIN_SERVICE_TOKEN=historical-pre-data2-secret\n")
			writeFixture(t, filepath.Join(state, "pre-data2.images.env"), "DATA_IMAGE=fixture/data:"+historicalPreData2FixtureSHA+"\n")
			writeFixture(t, filepath.Join(state, "pre-data2.compose.yaml"), composeContent)
			writeFixture(t, filepath.Join(state, "pre-data2.sha"), historicalPreData2FixtureSHA+"\n")
		}
	}
	if options.releaseStateWritePhase != "" {
		writeFixture(t, filepath.Join(state, "release-state-write-in-progress"), options.releaseStateWritePhase+"\n")
		if options.releaseStateWritePhase == "pre-data2" {
			writeFixture(t, filepath.Join(root, "pre-data2.runtime.env"), "ADMIN_SERVICE_TOKEN=previous-admin-secret\n")
			writeFixture(t, filepath.Join(state, "pre-data2.images.env"), "DATA_IMAGE=fixture/data:"+previousFixtureSHA+"\nMINIAPP_IMAGE=fixture/miniapp:"+previousFixtureSHA+"\nADMINPORTAL_IMAGE=fixture/adminportal:"+previousFixtureSHA+"\nADMIN_IMAGE=fixture/admin:"+previousFixtureSHA+"\n")
			writeFixture(t, filepath.Join(state, "pre-data2.compose.yaml"), composeContent)
			writeFixture(t, filepath.Join(state, "pre-data2.sha"), previousFixtureSHA+"\n")
		}
		if options.releaseStateWritePhase == "pre-data59" {
			writeFixture(t, filepath.Join(root, "pre-data59.runtime.env"), "ADMIN_SERVICE_TOKEN=previous-admin-secret\n")
			writeFixture(t, filepath.Join(state, "pre-data59.images.env"), "DATA_IMAGE=fixture/data:"+previousFixtureSHA+"\nMINIAPP_IMAGE=fixture/miniapp:"+previousFixtureSHA+"\nADMINPORTAL_IMAGE=fixture/adminportal:"+previousFixtureSHA+"\nADMIN_IMAGE=fixture/admin:"+previousFixtureSHA+"\n")
			writeFixture(t, filepath.Join(state, "pre-data59.compose.yaml"), composeContent)
			writeFixture(t, filepath.Join(state, "pre-data59.sha"), previousFixtureSHA+"\n")
		}
	}
	if options.cutoverMarkerPhase != "" {
		marker := "release_sha=" + fixtureSHA + "\nphase=" + options.cutoverMarkerPhase + "\n"
		if options.cutoverMarkerTargetVersion != "" {
			marker += "target_version=" + options.cutoverMarkerTargetVersion + "\n"
		}
		writeFixture(t, filepath.Join(state, "tidewise-2-cutover-in-progress"), marker)
	}

	report := options.migrationReport
	if report == "" {
		report = `{"current_version":"24","pending":[],"applied":[],"remaining":[]}`
	}
	writeFixture(t, filepath.Join(temp, "migration.json"), report+"\n")
	applyReport := options.migrationApplyReport
	if applyReport == "" {
		applyReport = report
	}
	writeFixture(t, filepath.Join(temp, "migration-apply.json"), applyReport+"\n")
	writeExecutable(t, filepath.Join(bin, "curl"), `#!/bin/sh
set -eu
echo " $* " >> "$FAKE_CURL_LOG"
count=0
if [ -f "$FAKE_CURL_COUNT" ]; then count="$(cat "$FAKE_CURL_COUNT")"; fi
count=$((count + 1))
echo "$count" > "$FAKE_CURL_COUNT"
if [ "${FAKE_FAIL_EVERY_CURL:-false}" = true ]; then exit 1; fi
if [ "${FAKE_FAIL_FIRST_CURL:-false}" = true ] && [ "$count" -eq 1 ]; then exit 1; fi
exit 0
`)
	writeExecutable(t, filepath.Join(bin, "flock"), "#!/bin/sh\nexit 0\n")
	writeExecutable(t, filepath.Join(bin, "sync"), `#!/bin/sh
set -eu
if [ "${FAKE_FAIL_CUTOVER_MARKER_SYNC:-false}" = true ]; then
  for argument in "$@"; do
    case "$argument" in
      *tidewise-2-cutover.*) exit 1 ;;
    esac
  done
fi
exit 0
`)
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
	  *" stop orphaned-cutover-writer "*)
	    rm -f "$FAKE_ORPHANED_WRITER"
	    ;;
	  *" stop restarting-cutover-writer "*)
	    rm -f "$FAKE_RESTARTING_WRITER"
	    ;;
	  *" ps --status running --services "*)
	    if [ -f "$FAKE_ORPHANED_WRITER" ]; then echo data; fi
	    ;;
	  *" ps "*" label=com.docker.compose.project=tidewise-uat "*)
	    if [ "${FAKE_FAIL_RUNNING_SERVICE_PROBE:-false}" = true ]; then exit 1; fi
	    if [ -f "$FAKE_ORPHANED_WRITER" ] && printf '%s\n' " $* " | grep -q 'label=com.docker.compose.service=data'; then
	      echo orphaned-cutover-writer
	    fi
	    if [ -f "$FAKE_RESTARTING_WRITER" ] && printf '%s\n' " $* " | grep -q ' --all ' && printf '%s\n' " $* " | grep -q 'label=com.docker.compose.service=data'; then
	      echo restarting-cutover-writer
	    fi
	    ;;
	  *" config --services "*)
	    compose_file=""
	    previous=""
	    for argument in "$@"; do
	      if [ "$previous" = "-f" ]; then compose_file="$argument"; fi
	      previous="$argument"
	    done
	    if [ -n "$compose_file" ] && grep -q 'qdrant:' "$compose_file"; then echo qdrant; fi
	    printf 'data\nminiapp\nadminportal\nadmin\n'
    ;;
	  *" run "*" /usr/local/bin/dbmigrate -apply -target-version 58 "*|*" run "*" /usr/local/bin/dbmigrate -apply -target-version 59 "*)
	    touch "$FAKE_CUTOVER_APPLIED"
	    cat "$FAKE_MIGRATION_APPLY_REPORT"
	    ;;
  *" run "*" /usr/local/bin/dbmigrate "*)
	    if [ -f "$FAKE_CUTOVER_APPLIED" ]; then cat "$FAKE_MIGRATION_APPLY_REPORT"; else cat "$FAKE_MIGRATION_REPORT"; fi
	    ;;
	  *" up "*)
    count=0
    if [ -f "$FAKE_UP_COUNT" ]; then count="$(cat "$FAKE_UP_COUNT")"; fi
    count=$((count + 1))
    echo "$count" > "$FAKE_UP_COUNT"
	    if [ "${FAKE_FAIL_EVERY_UP:-false}" = true ]; then exit 1; fi
	    if [ "${FAKE_FAIL_FIRST_UP:-false}" = true ] && [ "$count" -eq 1 ]; then exit 1; fi
    ;;
esac
exit 0
`)

	cmd := exec.Command("bash", filepath.Join(repoRoot, "infra", "uat", "deploy.sh"))
	expectedAvailable := options.currentRelease && !options.expectedCurrentMissing
	expectedSHA := options.expectedCurrentSHA
	if expectedSHA == "" && expectedAvailable {
		expectedSHA = previousFixtureSHA
		if options.currentReleaseCandidate {
			expectedSHA = fixtureSHA
		}
	}
	expectedStateFingerprint := releaseStateFingerprint(t, root)
	expectedDataImage := ""
	expectedMiniappImage := ""
	expectedAdminportalImage := ""
	expectedAdminImage := ""
	if expectedAvailable {
		expectedImageSHA := previousFixtureSHA
		if options.currentReleaseCandidate {
			expectedImageSHA = fixtureSHA
		}
		expectedDataImage = "fixture/data:" + expectedImageSHA
		expectedMiniappImage = "fixture/miniapp:" + expectedImageSHA
		expectedAdminportalImage = "fixture/adminportal:" + expectedImageSHA
		expectedAdminImage = "fixture/admin:" + expectedImageSHA
	}
	cmd.Env = append(os.Environ(),
		"PATH="+bin+":"+os.Getenv("PATH"),
		"DEPLOY_ROOT="+root,
		"RUNTIME_ENV="+runtimeEnv,
		"CANDIDATE_IMAGES="+imagesEnv,
		"COMMIT_SHA="+fixtureSHA,
		"EXPECTED_CURRENT_RELEASE_AVAILABLE="+boolText(expectedAvailable),
		"EXPECTED_CURRENT_RELEASE_STATE_FINGERPRINT="+expectedStateFingerprint,
		"EXPECTED_CURRENT_RELEASE_SHA="+expectedSHA,
		"EXPECTED_CURRENT_DATA_IMAGE="+expectedDataImage,
		"EXPECTED_CURRENT_MINIAPP_IMAGE="+expectedMiniappImage,
		"EXPECTED_CURRENT_ADMINPORTAL_IMAGE="+expectedAdminportalImage,
		"EXPECTED_CURRENT_ADMIN_IMAGE="+expectedAdminImage,
		"UAT_PUBLIC_BASE_URL=http://uat.example.test",
		"TIDEWISW_DB_PASSWORD=fixture-db-secret",
		"COMPOSE_FILE="+compose,
		"MIGRATION_RISK_MANIFEST="+manifest,
		"HIGH_RISK_BACKUP_CONFIRMED="+boolText(options.backupConfirmed),
		"DEPLOYMENT_MODE="+options.deploymentMode,
		"DESTRUCTIVE_DATA_CHANGE_CONFIRMED="+boolText(options.destructiveConfirmed),
		"EMPTY_DATA_SCHEMA_REBUILD_REQUESTED="+boolText(options.rebuildEmptyDataSchema),
		"RUNNER_TEMP="+temp,
		"GITHUB_RUN_ID=fixture",
		"GITHUB_STEP_SUMMARY="+filepath.Join(temp, "summary.md"),
		"FAKE_DOCKER_LOG="+dockerLog,
		"FAKE_MIGRATION_REPORT="+filepath.Join(temp, "migration.json"),
		"FAKE_MIGRATION_APPLY_REPORT="+filepath.Join(temp, "migration-apply.json"),
		"FAKE_CUTOVER_APPLIED="+filepath.Join(temp, "cutover-applied"),
		"FAKE_UP_COUNT="+upCount,
		"FAKE_FAIL_FIRST_UP="+boolText(options.failFirstUp),
		"FAKE_FAIL_EVERY_UP="+boolText(options.failEveryUp),
		"FAKE_CURL_COUNT="+curlCount,
		"FAKE_CURL_LOG="+curlLog,
		"FAKE_FAIL_FIRST_CURL="+boolText(options.failFirstCurl),
		"FAKE_FAIL_EVERY_CURL="+boolText(options.failEveryCurl),
		"FAKE_FAIL_RUNNING_SERVICE_PROBE="+boolText(options.failRunningServiceProbe),
		"FAKE_FAIL_CUTOVER_MARKER_SYNC="+boolText(options.failCutoverMarkerSync),
		"FAKE_ORPHANED_WRITER="+orphanedWriter,
		"FAKE_RESTARTING_WRITER="+restartingWriter,
		"DATA_IMAGE=fixture/data:"+fixtureSHA,
		"MINIAPP_IMAGE=fixture/miniapp:"+fixtureSHA,
		"ADMINPORTAL_IMAGE=fixture/adminportal:"+fixtureSHA,
		"ADMIN_IMAGE=fixture/admin:"+fixtureSHA,
	)
	output, err := cmd.CombinedOutput()
	return deployFixtureResult{root: root, dockerLog: dockerLog, curlLog: curlLog, output: string(output), err: err}
}

func releaseStateFingerprint(t *testing.T, root string) string {
	t.Helper()
	paths := []string{
		filepath.Join(root, "runtime.env"),
		filepath.Join(root, "state", "current.sha"),
		filepath.Join(root, "state", "current.images.env"),
		filepath.Join(root, "state", "current.compose.yaml"),
		filepath.Join(root, "state", "release-state-write-in-progress"),
		filepath.Join(root, "state", "tidewise-2-cutover-in-progress"),
	}
	records := ""
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err == nil {
			digest := sha256.Sum256(content)
			records += fmt.Sprintf("%x  %s\n", digest, path)
		} else if os.IsNotExist(err) {
			records += "missing  " + path + "\n"
		} else {
			t.Fatal(err)
		}
	}
	digest := sha256.Sum256([]byte(records))
	return fmt.Sprintf("%x", digest)
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

func conditionalValue(condition bool, value string) string {
	if condition {
		return value
	}
	return ""
}
