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
	for _, want := range []string{"PASS deployment-lock", "PASS agentrun-artifact-write", "PASS excluded-fact-audit-before", "PASS migration-apply", "PASS excluded-fact-audit-unchanged", "PASS bff-to-service-read-paths", "PASS release-state-recorded"} {
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
	assertFileContains(t, filepath.Join(result.root, "state", "current.images.env"), "fixture/qdrant:v1.15.5@sha256:")
	curlLog, err := os.ReadFile(result.curlLog)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"http://127.0.0.1:9012/healthz",
		"http://127.0.0.1:9012/api/miniapp/v1/research/themes?limit=1",
		"http://127.0.0.1:9013/api/admin/v1/model-providers",
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
	excludedFactAudit := strings.Index(string(dockerLog), "/usr/local/bin/uat-excluded-fact-audit")
	migrationPreflight := strings.Index(string(dockerLog), "/usr/local/bin/dbmigrate")
	if artifactProbe < 0 || excludedFactAudit < 0 || migrationPreflight < 0 ||
		artifactProbe > excludedFactAudit || excludedFactAudit > migrationPreflight {
		t.Fatalf("AgentRun Artifact write probe must run before migrations: %s", dockerLog)
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
	if strings.Contains(string(dockerLog), "exec -T qdrant") {
		t.Fatalf("rollback required Qdrant from the previous Compose that predates Qdrant ownership: %s", dockerLog)
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

func TestUATDeployExecutorBlocksIndustryRelationshipImportWithoutRecoveryPoint(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{industryImport: true})
	if result.err == nil || !strings.Contains(result.output, "FAIL industry-relationship-import-gate") {
		t.Fatalf("relationship import without recovery point was not blocked: err=%v output=%s", result.err, result.output)
	}
	logContent, err := os.ReadFile(result.dockerLog)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if strings.Contains(string(logContent), "/usr/local/bin/dbmigrate") ||
		strings.Contains(string(logContent), "industry-relationship-import") {
		t.Fatalf("recovery-point gate allowed database work: %s", logContent)
	}
}

func TestUATDeployExecutorImportsIndustryRelationshipsAndVerifiesReplay(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		industryImport:  true,
		backupConfirmed: true,
	})
	if result.err != nil {
		t.Fatalf("relationship import fixture failed: %v\n%s", result.err, result.output)
	}
	for _, want := range []string{
		"PASS industry-relationship-import-dry-run",
		"PASS industry-relationship-import-apply",
		"PASS industry-relationship-import-replay",
	} {
		if !strings.Contains(result.output, want) {
			t.Fatalf("relationship import output missing %q: %s", want, result.output)
		}
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	logText := string(dockerLog)
	if strings.Count(logText, "/usr/local/bin/industry-relationship-import") != 3 {
		t.Fatalf("relationship import did not run dry-run/apply/replay exactly once: %s", logText)
	}
	replay := strings.LastIndex(logText, "/usr/local/bin/industry-relationship-import")
	serviceStart := strings.Index(logText, " up ")
	if replay < 0 || serviceStart < 0 || replay > serviceStart {
		t.Fatalf("relationship replay must complete before candidate service start: %s", logText)
	}
}

func TestUATDeployExecutorProjectsIndustryGraphAndVerifiesReplay(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{graphProjection: true})
	if result.err != nil {
		t.Fatalf("graph projection fixture failed: %v\n%s", result.err, result.output)
	}
	for _, want := range []string{
		"PASS industry-graph-projection-dry-run",
		"PASS industry-graph-projection-apply",
		"PASS industry-graph-projection-replay",
	} {
		if !strings.Contains(result.output, want) {
			t.Fatalf("graph projection output missing %q: %s", want, result.output)
		}
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	logText := string(dockerLog)
	if strings.Count(logText, "/usr/local/bin/industry-graph-projector") != 3 {
		t.Fatalf("graph projection did not run dry-run/apply/replay exactly once: %s", logText)
	}
	if !strings.Contains(logText, "-e NEO4J_PASSWORD") ||
		strings.Contains(logText, fixtureNeo4jCredential()) {
		t.Fatalf("Neo4j secret was not passed by name only: %s", logText)
	}
	replay := strings.LastIndex(logText, "/usr/local/bin/industry-graph-projector")
	serviceStart := strings.Index(logText, " up ")
	if replay < 0 || serviceStart < 0 || replay > serviceStart {
		t.Fatalf("graph replay must complete before candidate service start: %s", logText)
	}
}

func TestUATDeployExecutorRejectsMissingGraphCredentialsBeforeDatabaseWork(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		graphProjection:   true,
		omitNeo4jPassword: true,
	})
	if result.err == nil || !strings.Contains(result.output, "FAIL industry-graph-projection-gate") {
		t.Fatalf("missing graph credential was not blocked: err=%v output=%s", result.err, result.output)
	}
	logContent, err := os.ReadFile(result.dockerLog)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if strings.Contains(string(logContent), "/usr/local/bin/dbmigrate") ||
		strings.Contains(string(logContent), "industry-graph-projector") {
		t.Fatalf("graph credential gate allowed database work: %s", logContent)
	}
}

func TestUATDeployExecutorRejectsFrozenGraphFingerprintDrift(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		graphProjection:       true,
		graphFingerprintDrift: true,
	})
	if result.err == nil || !strings.Contains(result.output, "source node fingerprint does not match") {
		t.Fatalf("graph fingerprint drift was not blocked: err=%v output=%s", result.err, result.output)
	}
	logContent, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(logContent), " up ") {
		t.Fatalf("fingerprint drift allowed candidate service start: %s", logContent)
	}
}

func TestUATDeployExecutorProjectsEventSemanticsBeforeAgentRunStarts(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{eventSemanticProjection: true})
	if result.err != nil {
		t.Fatalf("Event Semantic projection fixture failed: %v\n%s", result.err, result.output)
	}
	for _, want := range []string{
		"PASS event-semantic-projection-qdrant-ready",
		"PASS event-semantic-projection-apply",
		"PASS event-semantic-projection-verify",
	} {
		if !strings.Contains(result.output, want) {
			t.Fatalf("Event Semantic projection output missing %q: %s", want, result.output)
		}
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	logText := string(dockerLog)
	projector := strings.Index(logText, "/usr/local/bin/event-semantic-projector -apply -allow-env uat")
	fullStart := strings.LastIndex(logText, " up -d --wait --wait-timeout 120 --remove-orphans ")
	if projector < 0 || fullStart < 0 || projector > fullStart {
		t.Fatalf("Event Semantic projection must complete before the complete release starts: %s", logText)
	}
	if !strings.Contains(logText, " stop agentrun ") || !strings.Contains(logText, "-e EMBEDDING_API_KEY") ||
		strings.Contains(logText, fixtureEmbeddingCredential()) {
		t.Fatalf("Event Semantic projection did not pause AgentRun or scope the secret by name: %s", logText)
	}
}

func TestUATDeployExecutorRejectsMissingEmbeddingSecretBeforeDatabaseWork(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		eventSemanticProjection: true,
		omitEmbeddingSecret:     true,
	})
	if result.err == nil || !strings.Contains(result.output, "FAIL event-semantic-projection-gate") {
		t.Fatalf("missing embedding secret was not blocked: err=%v output=%s", result.err, result.output)
	}
	logContent, err := os.ReadFile(result.dockerLog)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if strings.Contains(string(logContent), "/usr/local/bin/dbmigrate") ||
		strings.Contains(string(logContent), "event-semantic-projector") {
		t.Fatalf("embedding secret gate allowed database or projection work: %s", logContent)
	}
}

func TestUATDeployExecutorRejectsEventSemanticProjectionCountDrift(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{
		currentRelease:          true,
		eventSemanticProjection: true,
		semanticProjectionDrift: true,
	})
	if result.err == nil || !strings.Contains(result.output, "projection count does not match") {
		t.Fatalf("Event Semantic projection count drift was not blocked: err=%v output=%s", result.err, result.output)
	}
	if !strings.Contains(result.output, "PASS rollback: previous complete release restored") {
		t.Fatalf("projection drift did not restore the previous release: %s", result.output)
	}
}

func TestUATDeployExecutorRejectsExcludedPostgreSQLFactDrift(t *testing.T) {
	result := runDeployFixture(t, deployFixtureOptions{excludedFactDrift: true})
	if result.err == nil || !strings.Contains(result.output, "excluded PostgreSQL facts changed") {
		t.Fatalf("excluded fact drift was not blocked: err=%v output=%s", result.err, result.output)
	}
	dockerLog, err := os.ReadFile(result.dockerLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(dockerLog), " up -d --wait --wait-timeout 120 --remove-orphans ") {
		t.Fatalf("excluded fact drift allowed the complete release to start: %s", dockerLog)
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
	failFirstUp             bool
	failFirstCurl           bool
	migrationReport         string
	migrationRisk           string
	backupConfirmed         bool
	failArtifactProbe       bool
	industryImport          bool
	graphProjection         bool
	omitNeo4jPassword       bool
	graphFingerprintDrift   bool
	eventSemanticProjection bool
	omitEmbeddingSecret     bool
	semanticProjectionDrift bool
	excludedFactDrift       bool
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
	industryImportCount := filepath.Join(temp, "industry-import-count")
	industryGraphCount := filepath.Join(temp, "industry-graph-count")
	excludedFactAuditCount := filepath.Join(temp, "excluded-fact-audit-count")
	curlCount := filepath.Join(temp, "curl-count")
	curlLog := filepath.Join(temp, "curl.log")
	writeFixture(t, runtimeEnv, "ADMIN_SERVICE_TOKEN=fixture-admin-secret\nAGENTRUN_SERVICE_TOKEN=fixture-agentrun-secret\nEMBEDDING_API_KEY="+fixtureEmbeddingCredential()+"\n")
	writeFixture(t, imagesEnv, "DATA_IMAGE=fixture/data:"+fixtureSHA+"\nMINIAPP_IMAGE=fixture/miniapp:"+fixtureSHA+"\nADMINPORTAL_IMAGE=fixture/adminportal:"+fixtureSHA+"\nADMIN_IMAGE=fixture/admin:"+fixtureSHA+"\nAGENTRUN_IMAGE=fixture/agentrun:"+fixtureSHA+"\nQDRANT_IMAGE=fixture/qdrant:v1.15.5@sha256:"+fixtureRelationshipPkgSHA+"\n")
	writeFixture(t, compose, "name: tidewise-uat\nservices:\n  qdrant: {}\n")
	migrationRisk := options.migrationRisk
	if migrationRisk == "" {
		migrationRisk = "high"
	}
	writeFixture(t, manifest, "000025\t"+migrationRisk+"\tfixture migration risk\n000024\thigh\tfixture high risk\n")
	writeFixture(t, agentrunManifest, "001\tnormal\tfixture AgentRun migration\n002\tnormal\tfixture AgentRun migration\n003\tnormal\tfixture AgentRun migration\n004\tnormal\tfixture AgentRun migration\n005\tnormal\tfixture AgentRun migration\n006\tnormal\tfixture AgentRun migration\n")

	if options.currentRelease {
		writeFixture(t, filepath.Join(root, "runtime.env"), "ADMIN_SERVICE_TOKEN=previous-admin-secret\n")
		writeFixture(t, filepath.Join(state, "current.images.env"), "DATA_IMAGE=fixture/data:"+previousFixtureSHA+"\nQDRANT_IMAGE=fixture/qdrant:v1.15.5@sha256:"+fixtureRelationshipPkgSHA+"\n")
		writeFixture(t, filepath.Join(state, "current.compose.yaml"), "name: tidewise-uat\nservices: {}\n")
		writeFixture(t, filepath.Join(state, "current.sha"), previousFixtureSHA+"\n")
	}

	report := options.migrationReport
	if report == "" {
		report = `{"current_version":"24","pending":[],"applied":[],"remaining":[]}`
	}
	writeFixture(t, filepath.Join(temp, "migration.json"), report+"\n")
	writeFixture(t, filepath.Join(temp, "agentrun-migration.json"), `{"current_version":"006","pending":[],"applied":[]}`+"\n")
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
  *" /usr/local/bin/industry-relationship-import "*)
    count=0
    if [ -f "$FAKE_INDUSTRY_IMPORT_COUNT" ]; then count="$(cat "$FAKE_INDUSTRY_IMPORT_COUNT")"; fi
    count=$((count + 1))
    echo "$count" > "$FAKE_INDUSTRY_IMPORT_COUNT"
    case " $* " in
      *" -dry-run "*)
        printf '{"package_sha256":"%s","package_counts":{"industry_chain":708},"dry_run":true,"unchanged":false}\n' "$FAKE_INDUSTRY_PACKAGE_SHA"
        ;;
      *)
        unchanged=false
        if [ "$count" -ge 3 ]; then unchanged=true; fi
        printf '{"package_sha256":"%s","package_counts":{"industry_chain":708},"dry_run":false,"unchanged":%s}\n' "$FAKE_INDUSTRY_PACKAGE_SHA" "$unchanged"
        ;;
    esac
    ;;
  *" /usr/local/bin/industry-graph-projector "*)
    count=0
    if [ -f "$FAKE_INDUSTRY_GRAPH_COUNT" ]; then count="$(cat "$FAKE_INDUSTRY_GRAPH_COUNT")"; fi
    count=$((count + 1))
    echo "$count" > "$FAKE_INDUSTRY_GRAPH_COUNT"
    unchanged=false
    dry_run=false
    applied=true
    if [ "$count" -eq 1 ]; then
      dry_run=true
      applied=false
    elif [ "$count" -ge 3 ]; then
      unchanged=true
      applied=false
    fi
    summary='{"node_count":4449,"relationship_count":7867,"node_fingerprint":"'"$FAKE_NODE_FINGERPRINT"'","relationship_fingerprint":"'"$FAKE_RELATIONSHIP_FINGERPRINT"'","node_type_counts":{"industry":512,"concept":180,"industry_chain":708,"chain_node":3049},"relationship_type_counts":{"MAPPED_TO_INDUSTRY":716,"MAPPED_TO_CONCEPT":521,"HAS_NODE":3350,"INPUT_TO":1537,"IS_COMPONENT_OF":704,"DEPENDS_ON":404,"IS_SUBCATEGORY_OF":635},"orphan_count":0,"duplicate_node_count":0,"duplicate_relationship_count":0,"self_loop_count":0,"missing_chain_identity_count":0}'
    printf '{"namespace":"tidewise-industry-v1","contract_version":"industry-graph-projection-v1","package_sha256":"%s","node_count":4449,"relationship_count":7867,"source":%s,"current_neo4j":%s,"final_neo4j":%s,"current_integrity_violation_count":0,"final_integrity_violation_count":0,"dry_run":%s,"applied":%s,"unchanged":%s}\n' "$FAKE_INDUSTRY_PACKAGE_SHA" "$summary" "$summary" "$summary" "$dry_run" "$applied" "$unchanged"
    ;;
  *" /usr/local/bin/event-semantic-projector -apply -allow-env uat "*)
    entity_count=4973
    if [ "${FAKE_SEMANTIC_PROJECTION_DRIFT:-false}" = true ]; then entity_count=4974; fi
    printf '{"projection_version":"event-semantic-projection.v1","embedding_model":"text-embedding-v4","entity_count":%s,"variable_definition_count":12}\n' "$entity_count"
    ;;
  *"http://qdrant:6333/collections/entity_semantic_v1"*)
    printf '{"result":{"status":"green","points_count":4973,"config":{"params":{"vectors":{"size":1024,"distance":"Cosine"}}}}}\n'
    ;;
  *"http://qdrant:6333/collections/variable_definition_semantic_v1"*)
    printf '{"result":{"status":"green","points_count":12,"config":{"params":{"vectors":{"size":1024,"distance":"Cosine"}}}}}\n'
    ;;
  *" /usr/local/bin/uat-excluded-fact-audit "*)
    count=0
    if [ -f "$FAKE_EXCLUDED_FACT_AUDIT_COUNT" ]; then count="$(cat "$FAKE_EXCLUDED_FACT_AUDIT_COUNT")"; fi
    count=$((count + 1))
    echo "$count" > "$FAKE_EXCLUDED_FACT_AUDIT_COUNT"
    fingerprint=0123456789abcdef0123456789abcdef
    if [ "${FAKE_EXCLUDED_FACT_DRIFT:-false}" = true ] && [ "$count" -ge 2 ]; then
      fingerprint=abcdef0123456789abcdef0123456789
    fi
    printf '{"contract_version":"uat-excluded-fact-audit.v1","tables":{"events":{"row_count":3,"fingerprint":"%s"}}}\n' "$fingerprint"
    ;;
  *" run "*" agentrun "*) echo "AgentRun database migrations are current" ;;
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
	neo4jPasswordEnv := strings.Join([]string{"NEO4J", "PASSWORD"}, "_")
	cmd.Env = append(os.Environ(),
		"PATH="+bin+":"+os.Getenv("PATH"),
		"DEPLOY_ROOT="+root,
		"RUNTIME_ENV="+runtimeEnv,
		"CANDIDATE_IMAGES="+imagesEnv,
		"COMMIT_SHA="+fixtureSHA,
		"UAT_PUBLIC_BASE_URL=http://uat.example.test",
		"TIDEWISW_DB_PASSWORD=fixture-db-secret",
		"AGENTRUN_DB_PASSWORD=fixture-agentrun-db-secret",
		"COMPOSE_FILE="+compose,
		"MIGRATION_RISK_MANIFEST="+manifest,
		"AGENTRUN_MIGRATION_RISK_MANIFEST="+agentrunManifest,
		"HIGH_RISK_BACKUP_CONFIRMED="+boolText(options.backupConfirmed),
		"INDUSTRY_RELATIONSHIP_IMPORT_ENABLED="+boolText(options.industryImport),
		"INDUSTRY_RELATIONSHIP_PACKAGE_SHA="+conditionalValue(options.industryImport, fixtureRelationshipPkgSHA),
		"INDUSTRY_GRAPH_PROJECTION_ENABLED="+boolText(options.graphProjection),
		"INDUSTRY_GRAPH_PACKAGE_SHA="+conditionalValue(options.graphProjection, fixtureRelationshipPkgSHA),
		"EVENT_SEMANTIC_PROJECTION_ENABLED="+boolText(options.eventSemanticProjection),
		"EMBEDDING_API_KEY="+conditionalValue(!options.omitEmbeddingSecret, fixtureEmbeddingCredential()),
		"NEO4J_URI=bolt://host.docker.internal:7687",
		"NEO4J_USERNAME=neo4j",
		neo4jPasswordEnv+"="+conditionalValue(!options.omitNeo4jPassword, fixtureNeo4jCredential()),
		"NEO4J_DATABASE=neo4j",
		"RUNNER_TEMP="+temp,
		"GITHUB_RUN_ID=fixture",
		"GITHUB_STEP_SUMMARY="+filepath.Join(temp, "summary.md"),
		"FAKE_DOCKER_LOG="+dockerLog,
		"FAKE_MIGRATION_REPORT="+filepath.Join(temp, "migration.json"),
		"FAKE_AGENTRUN_MIGRATION_REPORT="+filepath.Join(temp, "agentrun-migration.json"),
		"FAKE_UP_COUNT="+upCount,
		"FAKE_INDUSTRY_IMPORT_COUNT="+industryImportCount,
		"FAKE_INDUSTRY_GRAPH_COUNT="+industryGraphCount,
		"FAKE_EXCLUDED_FACT_AUDIT_COUNT="+excludedFactAuditCount,
		"FAKE_NODE_FINGERPRINT="+conditionalValue(!options.graphFingerprintDrift, "4229146e37ee554cd58377843743f93dc753bdfd92bbe7f2c9afac61c2003d63"),
		"FAKE_RELATIONSHIP_FINGERPRINT=aba6be387c0dad1b93c6fd14a4f9216b77a625d206cae9e7b977854f0cacec94",
		"FAKE_INDUSTRY_PACKAGE_SHA="+fixtureRelationshipPkgSHA,
		"FAKE_SEMANTIC_PROJECTION_DRIFT="+boolText(options.semanticProjectionDrift),
		"FAKE_EXCLUDED_FACT_DRIFT="+boolText(options.excludedFactDrift),
		"FAKE_FAIL_FIRST_UP="+boolText(options.failFirstUp),
		"FAKE_CURL_COUNT="+curlCount,
		"FAKE_CURL_LOG="+curlLog,
		"FAKE_FAIL_FIRST_CURL="+boolText(options.failFirstCurl),
		"FAKE_FAIL_ARTIFACT_PROBE="+boolText(options.failArtifactProbe),
		"DATA_IMAGE=fixture/data:"+fixtureSHA,
		"MINIAPP_IMAGE=fixture/miniapp:"+fixtureSHA,
		"ADMINPORTAL_IMAGE=fixture/adminportal:"+fixtureSHA,
		"ADMIN_IMAGE=fixture/admin:"+fixtureSHA,
		"AGENTRUN_IMAGE=fixture/agentrun:"+fixtureSHA,
		"QDRANT_IMAGE=fixture/qdrant:v1.15.5@sha256:"+fixtureRelationshipPkgSHA,
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
