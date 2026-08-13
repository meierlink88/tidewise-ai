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
		"data-service", "agent-run", "miniapp/backend",
		"admin-portal/backend", "admin-portal/frontend", "docs",
	} {
		if err := os.MkdirAll(filepath.Join(repository, directory), 0o750); err != nil {
			t.Fatal(err)
		}
	}

	writeFixture(t, filepath.Join(repository, "data-service", "service.go"), "data-v1\n")
	writeFixture(t, filepath.Join(repository, "agent-run", "service.go"), "agentrun-v1\n")
	writeFixture(t, filepath.Join(repository, "miniapp", "backend", "service.go"), "miniapp-v1\n")
	writeFixture(t, filepath.Join(repository, "admin-portal", "backend", "service.go"), "admin-backend-v1\n")
	writeFixture(t, filepath.Join(repository, "admin-portal", "frontend", "app.ts"), "admin-frontend-v1\n")
	writeFixture(t, filepath.Join(repository, "docs", "architecture.md"), "docs-v1\n")
	base := commitGitFixture(t, repository, "initial")

	writeFixture(t, filepath.Join(repository, "data-service", "service.go"), "data-v2\n")
	dataCommit := commitGitFixture(t, repository, "data")
	assertServicePlan(t, repository, base, dataCommit, map[string]string{
		"deploy_all": "false", "deploy_data": "true", "deploy_agentrun": "false",
		"deploy_miniapp": "false", "deploy_adminportal": "false", "deploy_admin": "false",
	})

	writeFixture(t, filepath.Join(repository, "agent-run", "service.go"), "agentrun-v2\n")
	writeFixture(t, filepath.Join(repository, "admin-portal", "frontend", "app.ts"), "admin-frontend-v2\n")
	multipleCommit := commitGitFixture(t, repository, "agentrun and admin frontend")
	assertServicePlan(t, repository, dataCommit, multipleCommit, map[string]string{
		"deploy_all": "false", "deploy_data": "false", "deploy_agentrun": "true",
		"deploy_miniapp": "false", "deploy_adminportal": "false", "deploy_admin": "true",
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
		"deploy_all": "true", "deploy_data": "true", "deploy_agentrun": "true",
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
	for _, want := range []string{"PASS deployment-lock", "PASS external-qdrant-ready", "PASS agentrun-artifact-write", "PASS migration-scope-gate", "PASS migration-apply", "PASS bff-to-service-read-paths", "PASS release-state-recorded"} {
		if !strings.Contains(result.output, want) {
			t.Fatalf("deploy output missing %q: %s", want, result.output)
		}
	}
	if strings.Contains(result.output, "fixture-admin-secret") || strings.Contains(result.output, "fixture-db-secret") {
		t.Fatalf("deploy output leaked a secret: %s", result.output)
	}
	assertFileContent(t, filepath.Join(result.root, "state", "current.sha"), fixtureSHA)
	assertFileContains(t, filepath.Join(result.root, "state", "current.images.env"), "fixture/data:"+fixtureSHA)
	assertFileContains(t, filepath.Join(result.root, "state", "current.images.env"), "fixture/agentrun:"+fixtureSHA)
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
		"http://127.0.0.1:9014/api/admin/v1/model-providers",
	} {
		if !strings.Contains(string(curlLog), want) {
			t.Fatalf("host verification missing %q: %s", want, curlLog)
		}
	}
	if strings.Contains(string(curlLog), "uat.example.test") {
		t.Fatalf("deployment attempted unsupported public-IP hairpin verification: %s", curlLog)
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dockerLog), "http://127.0.0.1:9080/readyz") {
		t.Fatalf("deployment did not enforce AgentRun readiness: %s", dockerLog)
	}
	artifactProbe := strings.Index(string(dockerLog), "/app/data/.uat-write-probe")
	externalQdrantProbe := strings.Index(string(dockerLog), "http://qdrant:6333/collections")
	migrationPreflight := strings.Index(string(dockerLog), "/usr/local/bin/dbmigrate")
	if externalQdrantProbe < 0 || artifactProbe < 0 || migrationPreflight < 0 ||
		externalQdrantProbe > artifactProbe || artifactProbe > migrationPreflight {
		t.Fatalf("external Qdrant and AgentRun Artifact probes must run before migrations: %s", dockerLog)
	}
}

func TestUATDeployExecutorStopsBeforeDatabaseWorkWhenExternalQdrantIsUnavailable(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{failExternalQdrant: true})
	if result.err == nil {
		t.Fatal("unavailable external Qdrant fixture unexpectedly succeeded")
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	logText := string(dockerLog)
	if !strings.Contains(logText, "http://qdrant:6333/collections") {
		t.Fatalf("deployment did not probe external Qdrant: %s", logText)
	}
	if strings.Contains(logText, "/app/data/.uat-write-probe") ||
		strings.Contains(logText, "/usr/local/bin/dbmigrate") {
		t.Fatalf("deployment performed protected work after external Qdrant probe failed: %s", logText)
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
	if strings.Contains(string(dockerLog), "http://qdrant:6333/collections") ||
		strings.Contains(string(dockerLog), "/usr/local/bin/dbmigrate") {
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
	})
	if result.err != nil {
		t.Fatalf("unchanged invalid current release did not allow full replacement: %v\n%s", result.err, result.output)
	}
	if !strings.Contains(result.output, "PASS release-state-plan-gate") ||
		!strings.Contains(result.output, "PASS release-state-recorded") {
		t.Fatalf("invalid state replacement missed release-state evidence: %s", result.output)
	}
	assertFileContent(t, filepath.Join(result.root, "state", "current.sha"), fixtureSHA)
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

func TestUATDeployExecutorStopsBeforeMigrationWhenAgentRunArtifactIsNotWritable(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{failArtifactProbe: true})
	if result.err == nil {
		t.Fatal("unwritable AgentRun Artifact fixture unexpectedly succeeded")
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dockerLog), "/app/data/.uat-write-probe") {
		t.Fatalf("deployment did not attempt the AgentRun Artifact write probe: %s", dockerLog)
	}
	if strings.Contains(string(dockerLog), "/usr/local/bin/dbmigrate") {
		t.Fatalf("deployment started migration after the AgentRun Artifact write probe failed: %s", dockerLog)
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

func TestUATDeployExecutorPreparesPost010AgentRunRollback(t *testing.T) {
	report := `{"current_version":"010","pending":[{"version":"011"},{"version":"012"},{"version":"013"},{"version":"014"}],"applied":[]}`
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:          true,
		failFirstUp:             true,
		agentrunMigrationReport: report,
	})
	if result.err == nil {
		t.Fatal("candidate failure fixture unexpectedly succeeded")
	}
	if !strings.Contains(result.output, "PASS agentrun-previous-release-database-compatibility") ||
		!strings.Contains(result.output, "PASS rollback: previous complete release restored") {
		t.Fatalf("AgentRun rollback compatibility was not prepared: %s", result.output)
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dockerLog), "--prepare-previous-release-rollback --previous-release-version 010") {
		t.Fatalf("rollback did not invoke AgentRun compatibility preparation: %s", dockerLog)
	}
}

func TestUATDeployExecutorPreservesPreviousPartialAgentRunMigrationVersion(t *testing.T) {
	report := `{"current_version":"013","pending":[{"version":"014"}],"applied":[]}`
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:          true,
		failFirstUp:             true,
		agentrunMigrationReport: report,
	})
	if result.err == nil {
		t.Fatal("candidate failure fixture unexpectedly succeeded")
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dockerLog), "--prepare-previous-release-rollback --previous-release-version 013") {
		t.Fatalf("rollback did not preserve the previous migration target: %s", dockerLog)
	}
}

func TestUATDeployExecutorPreservesPreviousAgentRunMigrationVersion014(t *testing.T) {
	report := `{"current_version":"014","pending":[{"version":"015"}],"applied":[]}`
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:          true,
		failFirstUp:             true,
		agentrunMigrationReport: report,
	})
	if result.err == nil {
		t.Fatal("candidate failure fixture unexpectedly succeeded")
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dockerLog), "--prepare-previous-release-rollback --previous-release-version 014") {
		t.Fatalf("rollback did not preserve AgentRun migration target 014: %s", dockerLog)
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
		{
			name: "AgentRun data migration",
			options: deployFixtureOptions{
				agentrunMigrationReport: `{"current_version":"014","pending":[{"version":"015"}],"applied":[]}`,
				agentrunMigrationScope:  "data",
				backupConfirmed:         true,
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
	fixtureSHA                = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	previousFixtureSHA        = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	fixtureRelationshipPkgSHA = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
)

type deployFixtureOptions struct {
	currentRelease          bool
	expectedCurrentSHA      string
	expectedCurrentMissing  bool
	releaseStateWritePhase  string
	invalidCurrentRelease   bool
	failFirstUp             bool
	failEveryUp             bool
	failFirstCurl           bool
	failEveryCurl           bool
	migrationReport         string
	agentrunMigrationReport string
	migrationRisk           string
	migrationScope          string
	agentrunMigrationScope  string
	backupConfirmed         bool
	failArtifactProbe       bool
	failExternalQdrant      bool
	legacyQdrantSnapshot    bool
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
	agentrunManifest := filepath.Join(temp, "agentrun-migration-risk.tsv")
	dockerLog := filepath.Join(temp, "docker.log")
	upCount := filepath.Join(temp, "up-count")
	curlCount := filepath.Join(temp, "curl-count")
	curlLog := filepath.Join(temp, "curl.log")
	writeFixture(t, runtimeEnv, "ADMIN_SERVICE_TOKEN=fixture-admin-secret\nAGENTRUN_SERVICE_TOKEN=fixture-agentrun-secret\nEMBEDDING_API_KEY="+fixtureEmbeddingCredential()+"\n")
	writeFixture(t, imagesEnv, "DATA_IMAGE=fixture/data:"+fixtureSHA+"\nMINIAPP_IMAGE=fixture/miniapp:"+fixtureSHA+"\nADMINPORTAL_IMAGE=fixture/adminportal:"+fixtureSHA+"\nADMIN_IMAGE=fixture/admin:"+fixtureSHA+"\nAGENTRUN_IMAGE=fixture/agentrun:"+fixtureSHA+"\n")
	writeFixture(t, compose, "name: tidewise-uat\nservices: {}\n")
	migrationRisk := options.migrationRisk
	if migrationRisk == "" {
		migrationRisk = "high"
	}
	migrationScope := options.migrationScope
	if migrationScope == "" {
		migrationScope = "schema"
	}
	agentrunMigrationScope := options.agentrunMigrationScope
	if agentrunMigrationScope == "" {
		agentrunMigrationScope = "schema"
	}
	writeFixture(t, manifest, "000025\t"+migrationRisk+"\t"+migrationScope+"\tfixture migration risk\n000024\thigh\t"+migrationScope+"\tfixture high risk\n")
	writeFixture(t, agentrunManifest, "001\tnormal\tschema\tfixture AgentRun migration\n002\tnormal\tschema\tfixture AgentRun migration\n003\tnormal\tschema\tfixture AgentRun migration\n004\tnormal\tschema\tfixture AgentRun migration\n005\tnormal\tschema\tfixture AgentRun migration\n006\tnormal\tschema\tfixture AgentRun migration\n007\tnormal\tschema\tfixture AgentRun migration\n008\tnormal\tschema\tfixture AgentRun migration\n009\tnormal\tschema\tfixture AgentRun migration\n010\tnormal\tschema\tfixture AgentRun migration\n011\tnormal\tschema\tfixture AgentRun migration\n012\tnormal\tschema\tfixture AgentRun migration\n013\tnormal\tschema\tfixture AgentRun migration\n014\tnormal\tschema\tfixture AgentRun migration\n015\tnormal\t"+agentrunMigrationScope+"\tfixture AgentRun migration\n")

	if options.currentRelease {
		writeFixture(t, filepath.Join(root, "runtime.env"), "ADMIN_SERVICE_TOKEN=previous-admin-secret\n")
		currentImages := "DATA_IMAGE=fixture/data:" + previousFixtureSHA + "\n" +
			"MINIAPP_IMAGE=fixture/miniapp:" + previousFixtureSHA + "\n" +
			"ADMINPORTAL_IMAGE=fixture/adminportal:" + previousFixtureSHA + "\n" +
			"ADMIN_IMAGE=fixture/admin:" + previousFixtureSHA + "\n" +
			"AGENTRUN_IMAGE=fixture/agentrun:" + previousFixtureSHA + "\n"
		currentCompose := "name: tidewise-uat\nservices: {}\n"
		if options.legacyQdrantSnapshot {
			currentCompose = "name: tidewise-uat\nservices:\n  qdrant: {}\n"
		}
		writeFixture(t, filepath.Join(state, "current.images.env"), currentImages)
		writeFixture(t, filepath.Join(state, "current.compose.yaml"), currentCompose)
		currentSHA := previousFixtureSHA
		if options.invalidCurrentRelease {
			currentSHA = "not-a-release-sha"
		}
		writeFixture(t, filepath.Join(state, "current.sha"), currentSHA+"\n")
	}
	if options.releaseStateWritePhase != "" {
		writeFixture(t, filepath.Join(state, "release-state-write-in-progress"), options.releaseStateWritePhase+"\n")
	}

	report := options.migrationReport
	if report == "" {
		report = `{"current_version":"24","pending":[],"applied":[],"remaining":[]}`
	}
	writeFixture(t, filepath.Join(temp, "migration.json"), report+"\n")
	agentrunReport := options.agentrunMigrationReport
	if agentrunReport == "" {
		agentrunReport = `{"current_version":"015","pending":[],"applied":[]}`
	}
	writeFixture(t, filepath.Join(temp, "agentrun-migration.json"), agentrunReport+"\n")
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
	  *" config --services "*)
	    compose_file=""
	    previous=""
	    for argument in "$@"; do
	      if [ "$previous" = "-f" ]; then compose_file="$argument"; fi
	      previous="$argument"
	    done
	    if [ -n "$compose_file" ] && grep -q 'qdrant:' "$compose_file"; then echo qdrant; fi
	    printf 'data\nagentrun\nminiapp\nadminportal\nadmin\n'
    ;;
  *"/app/data/.uat-write-probe."*)
    if [ "${FAKE_FAIL_ARTIFACT_PROBE:-false}" = true ]; then exit 1; fi
    ;;
  *" --check-only "*) cat "$FAKE_AGENTRUN_MIGRATION_REPORT" ;;
  *" run "*" /usr/local/bin/dbmigrate "*) cat "$FAKE_MIGRATION_REPORT" ;;
	  *"http://qdrant:6333/collections "*)
	    if [ "${FAKE_FAIL_EXTERNAL_QDRANT:-false}" = true ]; then exit 1; fi
	    printf '{"result":{"collections":[]}}\n'
	    ;;
  *" run "*" agentrun "*) echo "AgentRun database migrations are current" ;;
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
	}
	expectedStateFingerprint := releaseStateFingerprint(t, root)
	expectedDataImage := ""
	expectedMiniappImage := ""
	expectedAdminportalImage := ""
	expectedAdminImage := ""
	expectedAgentrunImage := ""
	if expectedAvailable {
		expectedDataImage = "fixture/data:" + previousFixtureSHA
		expectedMiniappImage = "fixture/miniapp:" + previousFixtureSHA
		expectedAdminportalImage = "fixture/adminportal:" + previousFixtureSHA
		expectedAdminImage = "fixture/admin:" + previousFixtureSHA
		expectedAgentrunImage = "fixture/agentrun:" + previousFixtureSHA
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
		"EXPECTED_CURRENT_AGENTRUN_IMAGE="+expectedAgentrunImage,
		"UAT_PUBLIC_BASE_URL=http://uat.example.test",
		"TIDEWISW_DB_PASSWORD=fixture-db-secret",
		"AGENTRUN_DB_PASSWORD=fixture-agentrun-db-secret",
		"COMPOSE_FILE="+compose,
		"MIGRATION_RISK_MANIFEST="+manifest,
		"AGENTRUN_MIGRATION_RISK_MANIFEST="+agentrunManifest,
		"HIGH_RISK_BACKUP_CONFIRMED="+boolText(options.backupConfirmed),
		"EMBEDDING_API_KEY="+fixtureEmbeddingCredential(),
		"RUNNER_TEMP="+temp,
		"GITHUB_RUN_ID=fixture",
		"GITHUB_STEP_SUMMARY="+filepath.Join(temp, "summary.md"),
		"FAKE_DOCKER_LOG="+dockerLog,
		"FAKE_MIGRATION_REPORT="+filepath.Join(temp, "migration.json"),
		"FAKE_AGENTRUN_MIGRATION_REPORT="+filepath.Join(temp, "agentrun-migration.json"),
		"FAKE_UP_COUNT="+upCount,
		"FAKE_FAIL_FIRST_UP="+boolText(options.failFirstUp),
		"FAKE_FAIL_EVERY_UP="+boolText(options.failEveryUp),
		"FAKE_CURL_COUNT="+curlCount,
		"FAKE_CURL_LOG="+curlLog,
		"FAKE_FAIL_FIRST_CURL="+boolText(options.failFirstCurl),
		"FAKE_FAIL_EVERY_CURL="+boolText(options.failEveryCurl),
		"FAKE_FAIL_ARTIFACT_PROBE="+boolText(options.failArtifactProbe),
		"FAKE_FAIL_EXTERNAL_QDRANT="+boolText(options.failExternalQdrant),
		"DATA_IMAGE=fixture/data:"+fixtureSHA,
		"MINIAPP_IMAGE=fixture/miniapp:"+fixtureSHA,
		"ADMINPORTAL_IMAGE=fixture/adminportal:"+fixtureSHA,
		"ADMIN_IMAGE=fixture/admin:"+fixtureSHA,
		"AGENTRUN_IMAGE=fixture/agentrun:"+fixtureSHA,
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

func fixtureNeo4jCredential() string {
	return strings.Join([]string{"fixture", "neo4j", "credential"}, "-")
}

func fixtureEmbeddingCredential() string {
	return strings.Join([]string{"fixture", "embedding", "credential"}, "-")
}

func conditionalValue(condition bool, value string) string {
	if condition {
		return value
	}
	return ""
}
