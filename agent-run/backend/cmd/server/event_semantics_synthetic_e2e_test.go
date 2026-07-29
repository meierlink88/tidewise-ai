package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
	semanticusecase "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic/usecase"
	semanticworkflow "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic/workflow"
	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform/admin"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/data/dataclient"
	agentrunpostgres "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/data/postgres"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/testsupport"
)

const (
	syntheticDataToken = "synthetic-event-semantics-token"

	syntheticCompanyID  = "22000000-0000-4000-8000-000000000001"
	syntheticProductID  = "22000000-0000-4000-8000-000000000002"
	syntheticRelationID = "22000000-0000-4000-8000-000000000003"

	syntheticAcceptedRawDocumentID = "20000000-0000-4000-8000-000000000001"
	syntheticAcceptedEventID       = "20000000-0000-4000-8000-000000000002"
	syntheticAcceptedEvidenceID    = "20000000-0000-4000-8000-000000000003"

	syntheticQuarantinedEventID    = "21000000-0000-4000-8000-000000000002"
	syntheticQuarantinedEvidenceID = "21000000-0000-4000-8000-000000000003"
)

func TestSyntheticEventSemanticsEndToEnd(t *testing.T) {
	if os.Getenv("EVENT_SEMANTICS_SYNTHETIC_E2E") != "1" {
		t.Skip("set EVENT_SEMANTICS_SYNTHETIC_E2E=1 to run the isolated cross-service acceptance")
	}
	dataDatabaseURL := requireSyntheticDatabaseEnvironment(t, "TIDEWISE_TEST_DATABASE_URL")
	agentRunDatabaseURL := requireSyntheticDatabaseEnvironment(t, "AGENTRUN_TEST_DATABASE_URL")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	t.Cleanup(cancel)

	dataFixture := startSyntheticDataService(
		t, ctx, syntheticRepositoryRoot(t), dataDatabaseURL,
	)
	dataFixture.assertEmptyResearch(t, ctx)

	isolatedAgentRunURL, cleanupAgentRun, err := testsupport.IsolatedPostgresDatabase(
		ctx, agentRunDatabaseURL, "event_semantics_synthetic_acceptance",
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanupAgentRun)
	agentRunDatabase, err := agentrunpostgres.Open(ctx, isolatedAgentRunURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(agentRunDatabase.Close)
	if err := agentrunpostgres.Migrate(ctx, agentRunDatabase); err != nil {
		t.Fatal(err)
	}
	store := agentrunpostgres.New(agentRunDatabase)
	dataClient, err := dataclient.New(dataclient.Config{
		BaseURL: dataFixture.baseURL, ServiceToken: syntheticDataToken,
		Timeout: 5 * time.Second, MaxResponseBytes: 4 * 1024 * 1024,
	})
	if err != nil {
		t.Fatal(err)
	}

	generator := &syntheticSemanticModel{kind: "generator"}
	reviewer := &syntheticSemanticModel{kind: "reviewer"}
	runnable, err := semanticworkflow.New(ctx, dataClient, generator, reviewer)
	if err != nil {
		t.Fatal(err)
	}
	semanticApplication, err := semanticusecase.New(
		store,
		dataClient,
		func(context.Context) (semanticusecase.Runtime, error) {
			return semanticusecase.Runtime{
				GeneratorModel: "synthetic-generator-v1",
				ReviewerModel:  "synthetic-reviewer-v1",
				Run:            runnable,
			}, nil
		},
		time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}

	for index := 0; index < 2; index++ {
		if err := semanticApplication.Tick(ctx); err != nil {
			t.Fatalf("run synthetic Event %d: %v", index+1, err)
		}
	}
	accepted, err := dataClient.GetEventSemantics(ctx, syntheticAcceptedEventID)
	if err != nil {
		t.Fatal(err)
	}
	assertSyntheticAcceptedSemantics(t, accepted)
	quarantined, err := dataClient.GetEventSemantics(ctx, syntheticQuarantinedEventID)
	if err != nil {
		t.Fatal(err)
	}
	assertSyntheticQuarantinedSemantics(t, quarantined)

	generatorCalls, reviewerCalls := generator.calls, reviewer.calls
	if err := semanticApplication.Tick(ctx); err != nil {
		t.Fatalf("replay empty Event Semantic tick: %v", err)
	}
	if generator.calls != generatorCalls || reviewer.calls != reviewerCalls {
		t.Fatalf(
			"idempotent tick repeated model calls: generator=%d/%d reviewer=%d/%d",
			generator.calls, generatorCalls, reviewer.calls, reviewerCalls,
		)
	}

	adminService, err := admin.New(store, admin.Registry{}, "dev")
	if err != nil {
		t.Fatal(err)
	}
	statuses, err := adminService.ListAgentStatuses(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertSyntheticAgentStatus(t, statuses)
	executions, err := adminService.ListAgentExecutions(ctx, agentrun.ExecutionListQuery{
		AgentKey: eventsemantic.AgentKey, Page: 1, PageSize: 20, Ascending: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if executions.TotalItems != 2 || len(executions.Items) != 2 {
		t.Fatalf("Event Semantic executions = %#v, want two", executions)
	}
	for _, execution := range executions.Items {
		if execution.Status != "succeeded" {
			t.Fatalf("execution %s status = %q, want succeeded", execution.ID, execution.Status)
		}
	}
	dataFixture.assertEmptyResearch(t, ctx)
	t.Logf(
		"accepted_event=%s accepted_submission=%s quarantined_event=%s quarantined_submission=%s executions=%d",
		syntheticAcceptedEventID, accepted.Submissions[0].SubmissionID,
		syntheticQuarantinedEventID, quarantined.Submissions[0].SubmissionID,
		executions.TotalItems,
	)
}

type syntheticSemanticModel struct {
	kind  string
	calls int
}

func (m *syntheticSemanticModel) Generate(
	_ context.Context,
	input []*schema.Message,
	_ ...model.Option,
) (*schema.Message, error) {
	m.calls++
	var content strings.Builder
	for _, message := range input {
		content.WriteString(message.Content)
	}
	payload := content.String()
	evidenceID := ""
	switch {
	case strings.Contains(payload, syntheticAcceptedEvidenceID):
		evidenceID = syntheticAcceptedEvidenceID
	case strings.Contains(payload, syntheticQuarantinedEvidenceID):
		evidenceID = syntheticQuarantinedEvidenceID
	default:
		return nil, errors.New("synthetic model input has no fixture Evidence")
	}
	if m.kind == "generator" {
		if strings.Contains(payload, "direct_targets_by_link_key") {
			return schema.AssistantMessage(syntheticDirectImpact(evidenceID), nil), nil
		}
		return schema.AssistantMessage(syntheticNativeCandidates(evidenceID), nil), nil
	}
	if m.kind == "reviewer" {
		decision := "indeterminate"
		if evidenceID == syntheticAcceptedEvidenceID {
			decision = "pass"
		}
		return schema.AssistantMessage(syntheticReview(decision, evidenceID), nil), nil
	}
	return nil, errors.New("synthetic model kind is invalid")
}

func (*syntheticSemanticModel) Stream(
	context.Context,
	[]*schema.Message,
	...model.Option,
) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("synthetic model does not stream")
}

type syntheticDataService struct {
	baseURL            string
	fixtureBinary      string
	fixtureEnvironment []string
	repositoryRoot     string
}

func (s syntheticDataService) assertEmptyResearch(
	t *testing.T,
	ctx context.Context,
) {
	t.Helper()
	runSyntheticFixture(
		t, ctx, s.repositoryRoot, s.fixtureBinary, s.fixtureEnvironment,
		"-action", "assert-empty-research",
	)
}

func requireSyntheticDatabaseEnvironment(t *testing.T, key string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		t.Fatalf("%s is required when EVENT_SEMANTICS_SYNTHETIC_E2E=1", key)
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Hostname() != "localhost" && parsed.Hostname() != "127.0.0.1" &&
		parsed.Hostname() != "::1" {
		t.Fatalf("%s must point to a loopback PostgreSQL instance", key)
	}
	return value
}

func syntheticRepositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	return root
}

func startSyntheticDataService(
	t *testing.T,
	ctx context.Context,
	repositoryRoot string,
	baseDatabaseURL string,
) syntheticDataService {
	t.Helper()
	temporary := t.TempDir()
	binaryDirectory := filepath.Join(temporary, "bin")
	if err := os.Mkdir(binaryDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	fixtureBinary := filepath.Join(binaryDirectory, "data-fixture")
	migrateBinary := filepath.Join(binaryDirectory, "data-migrate")
	serverBinary := filepath.Join(binaryDirectory, "data-server")
	buildSyntheticBinary(
		t, ctx, repositoryRoot, fixtureBinary,
		"./analyse-data-service/backend/cmd/event-semantics-synthetic-fixture",
	)
	buildSyntheticBinary(
		t, ctx, repositoryRoot, migrateBinary, "./analyse-data-service/backend/cmd/dbmigrate",
	)
	buildSyntheticBinary(
		t, ctx, repositoryRoot, serverBinary, "./analyse-data-service/backend/cmd/server",
	)
	createOutput := runSyntheticFixture(
		t, ctx, repositoryRoot, fixtureBinary,
		append(os.Environ(), "EVENT_SEMANTICS_SYNTHETIC_FIXTURE=1"),
		"-action", "create-database", "-base-url", baseDatabaseURL,
	)
	var created struct {
		DatabaseName string `json:"database_name"`
		DatabaseURL  string `json:"database_url"`
	}
	if err := json.Unmarshal(createOutput, &created); err != nil ||
		created.DatabaseName == "" || created.DatabaseURL == "" {
		t.Fatalf("decode Data fixture database identity: %v\n%s", err, createOutput)
	}
	t.Cleanup(func() {
		cleanupContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		command := exec.CommandContext(
			cleanupContext, fixtureBinary,
			"-action", "drop-database",
			"-base-url", baseDatabaseURL,
			"-target-database", created.DatabaseName,
		)
		command.Dir = repositoryRoot
		command.Env = append(os.Environ(), "EVENT_SEMANTICS_SYNTHETIC_FIXTURE=1")
		_ = command.Run()
	})

	parsed, err := url.Parse(created.DatabaseURL)
	if err != nil {
		t.Fatal(err)
	}
	password, _ := parsed.User.Password()
	port := reserveSyntheticPort(t)
	configDirectory := filepath.Join(temporary, "config")
	if err := os.Mkdir(configDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	config := fmt.Sprintf(`app:
  name: data
  env: local
server:
  host: 127.0.0.1
  port: %d
  read_timeout_seconds: 5
  write_timeout_seconds: 15
log:
  level: error
database:
  host: %s
  port: %s
  name: %s
  user: %s
  ssl_mode: %s
  max_open_conns: 5
  max_idle_conns: 2
  conn_max_lifetime_seconds: 60
  connect_timeout_seconds: 5
migration:
  directory: %s
  auto_apply: false
  lock_key: synthetic_event_semantics_acceptance
`,
		port,
		parsed.Hostname(),
		parsed.Port(),
		strings.TrimPrefix(parsed.Path, "/"),
		parsed.User.Username(),
		parsed.Query().Get("sslmode"),
		filepath.Join(repositoryRoot, "analyse-data-service", "backend", "migrations"),
	)
	if err := os.WriteFile(
		filepath.Join(configDirectory, "config.local.yaml"), []byte(config), 0o600,
	); err != nil {
		t.Fatal(err)
	}
	fixtureEnvironment := append(os.Environ(),
		"APP_ENV=local",
		"TIDEWISE_CONFIG_DIR="+configDirectory,
		"TIDEWISW_DB_PASSWORD="+password,
		"DATA_SERVICE_TOKEN="+syntheticDataToken,
		"EVENT_SEMANTICS_SYNTHETIC_FIXTURE=1",
		"PGOPTIONS=-c tidewise.phase_a_cleanup_write_authorized=reviewed_backup_verified "+
			"-c tidewise.external_identifier_schema_write_authorized=reviewed_backup_verified "+
			"-c tidewise.alliance_economy_schema_write_authorized=reviewed_local_cleanup_verified",
	)
	migration := exec.CommandContext(ctx, migrateBinary, "-apply")
	migration.Dir = repositoryRoot
	migration.Env = fixtureEnvironment
	if output, err := migration.CombinedOutput(); err != nil {
		t.Fatalf("migrate synthetic Data database: %v\n%s", err, output)
	}
	runSyntheticFixture(
		t, ctx, repositoryRoot, fixtureBinary, fixtureEnvironment, "-action", "seed",
	)

	command := exec.Command(serverBinary)
	command.Dir = repositoryRoot
	command.Env = fixtureEnvironment
	var logs strings.Builder
	command.Stdout = &logs
	command.Stderr = &logs
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	finished := make(chan error, 1)
	go func() { finished <- command.Wait() }()
	t.Cleanup(func() {
		if command.Process == nil {
			return
		}
		_ = command.Process.Kill()
		select {
		case <-finished:
		case <-time.After(5 * time.Second):
		}
	})
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	waitForSyntheticDataService(t, ctx, baseURL, finished, &logs)
	return syntheticDataService{
		baseURL: baseURL, fixtureBinary: fixtureBinary,
		fixtureEnvironment: fixtureEnvironment, repositoryRoot: repositoryRoot,
	}
}

func runSyntheticFixture(
	t *testing.T,
	ctx context.Context,
	repositoryRoot string,
	fixtureBinary string,
	environment []string,
	args ...string,
) []byte {
	t.Helper()
	command := exec.CommandContext(ctx, fixtureBinary, args...)
	command.Dir = repositoryRoot
	command.Env = environment
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run Data-owned synthetic fixture %v: %v\n%s", args, err, output)
	}
	return output
}

func waitForSyntheticDataService(
	t *testing.T,
	ctx context.Context,
	baseURL string,
	finished <-chan error,
	logs *strings.Builder,
) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		request, _ := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/healthz", nil)
		request.Header.Set("Authorization", "Bearer "+syntheticDataToken)
		response, requestErr := http.DefaultClient.Do(request)
		if requestErr == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}
		select {
		case runErr := <-finished:
			t.Fatalf("synthetic Data Service exited: %v\n%s", runErr, logs.String())
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("synthetic Data Service did not become ready\n%s", logs.String())
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func reserveSyntheticPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}

func buildSyntheticBinary(
	t *testing.T,
	ctx context.Context,
	repositoryRoot string,
	output string,
	pkg string,
) {
	t.Helper()
	command := exec.CommandContext(ctx, "go", "build", "-o", output, pkg)
	command.Dir = repositoryRoot
	command.Env = append(os.Environ(), "GOCACHE=/tmp/tidewise-go-cache")
	if result, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", pkg, err, result)
	}
}

func syntheticNativeCandidates(evidenceID string) string {
	return fmt.Sprintf(
		`{"entity_links":[{"candidate_key":"company","mention":"Synthetic Wafer Fab","entity_id":"%s","entity_role":"subject","evidence_ids":["%s"],"resolution_method":"model_guess","resolution_confidence":"0.99"}],"variable_signals":[{"candidate_key":"production","subject_link_key":"company","variable_key":"production_volume","variable_version":1,"direction":"decrease","assertion_modality":"actual","evidence_ids":["%s"],"measurements":[{"measurement_role":"relative_change","value_shape":"exact","raw_value":"-10","raw_unit":"%%","canonical_value":"-10","canonical_unit":"percent","raw_text":"production fell 10%%","is_approximate":false,"evidence_id":"%s"}],"extraction_confidence":"0.98"}]}`,
		syntheticCompanyID, evidenceID, evidenceID, evidenceID,
	)
}

func syntheticDirectImpact(evidenceID string) string {
	return fmt.Sprintf(
		`{"direct_impacts":[{"candidate_key":"supply","source_signal_key":"production","target_entity_id":"%s","affected_variable_key":"market_supply","affected_variable_version":1,"affected_direction":"decrease","derivation_type":"rule_inferred","mechanism_summary":"Synthetic Wafer Fab produces Synthetic 8-inch Wafer; lower production reduces its directly available supply.","entity_relation_id":"%s","rule_key":"production_decrease_reduces_product_supply","rule_version":1,"evidence_ids":["%s"],"assertion_confidence":"0.96"}]}`,
		syntheticProductID, syntheticRelationID, evidenceID,
	)
}

func syntheticReview(decision string, evidenceID string) string {
	return fmt.Sprintf(
		`{"items":[{"candidate_type":"entity_link","candidate_key":"company","decision":"%s","reason_codes":["synthetic_review"],"evidence_ids":["%s"]},{"candidate_type":"variable_signal","candidate_key":"production","decision":"%s","reason_codes":["synthetic_review"],"evidence_ids":["%s"]},{"candidate_type":"direct_impact","candidate_key":"supply","decision":"%s","reason_codes":["synthetic_review"],"evidence_ids":["%s"]}]}`,
		decision, evidenceID, decision, evidenceID, decision, evidenceID,
	)
}

func assertSyntheticAcceptedSemantics(t *testing.T, semantics eventsemantic.EventSemantics) {
	t.Helper()
	if semantics.EventID != syntheticAcceptedEventID || len(semantics.Submissions) != 1 {
		t.Fatalf("accepted Event semantics = %#v", semantics)
	}
	submission := semantics.Submissions[0]
	if submission.Status != "accepted" ||
		submission.ContextLeaseID == "" || submission.AgentExecutionID == "" ||
		submission.AgentKey != eventsemantic.AgentKey ||
		submission.AgentVersion != eventsemantic.AgentVersion ||
		len(submission.GeneratorPromptHash) != 64 ||
		submission.GeneratorModel != "synthetic-generator-v1" ||
		len(submission.ReviewerPromptHash) != 64 ||
		submission.ReviewerModel != "synthetic-reviewer-v1" ||
		len(submission.AdjudicatorPromptHash) != 64 ||
		submission.AdjudicatorModel != "synthetic-reviewer-v1" ||
		submission.OntologyVersion != "event-semantics.phase-one@1" ||
		submission.AcceptancePolicyVersion != "event-semantics.phase-one@1" ||
		len(submission.CandidateSnapshot) == 0 ||
		len(submission.EntityLinks) != 1 || submission.EntityLinks[0].Status != "accepted" ||
		len(submission.VariableSignals) != 1 || submission.VariableSignals[0].Status != "accepted" ||
		submission.VariableSignals[0].RecordID == "" ||
		len(submission.DirectImpacts) != 1 || submission.DirectImpacts[0].Status != "accepted" ||
		submission.DirectImpacts[0].RecordID == "" ||
		submission.AuditWorkPackage == nil ||
		submission.AuditWorkPackage.Evidence[0].RawDocumentID != syntheticAcceptedRawDocumentID ||
		submission.AuditWorkPackage.DirectImpacts[0].RuleKey !=
			"production_decrease_reduces_product_supply" ||
		submission.AuditWorkPackage.DirectImpacts[0].RuleVersion != 1 ||
		len(submission.ReviewSnapshots) != 1 {
		t.Fatalf("accepted Submission = %#v", submission)
	}
}

func assertSyntheticQuarantinedSemantics(t *testing.T, semantics eventsemantic.EventSemantics) {
	t.Helper()
	if semantics.EventID != syntheticQuarantinedEventID || len(semantics.Submissions) != 1 {
		t.Fatalf("quarantined Event semantics = %#v", semantics)
	}
	submission := semantics.Submissions[0]
	if submission.Status != "quarantined" ||
		submission.AgentExecutionID == "" ||
		len(submission.EntityLinks) != 1 || submission.EntityLinks[0].Status != "quarantined" ||
		len(submission.VariableSignals) != 1 || submission.VariableSignals[0].Status != "quarantined" ||
		len(submission.DirectImpacts) != 1 || submission.DirectImpacts[0].Status != "quarantined" ||
		len(submission.ReviewSnapshots) != 2 {
		t.Fatalf("quarantined Submission = %#v", submission)
	}
}

func assertSyntheticAgentStatus(t *testing.T, statuses []agentrun.AgentStatus) {
	t.Helper()
	for _, status := range statuses {
		if status.AgentKey != eventsemantic.AgentKey {
			continue
		}
		if status.IsWorking || status.CurrentExecutionStatus != "idle" {
			t.Fatalf("Event Semantic Agent status = %#v", status)
		}
		return
	}
	t.Fatalf("Event Semantic Agent status missing from %#v", statuses)
}
