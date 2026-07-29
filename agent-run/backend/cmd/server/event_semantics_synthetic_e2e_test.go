package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

	syntheticQuarantinedRawDocumentID = "21000000-0000-4000-8000-000000000001"
	syntheticQuarantinedEventID       = "21000000-0000-4000-8000-000000000002"
	syntheticQuarantinedEvidenceID    = "21000000-0000-4000-8000-000000000003"
)

func TestSyntheticEventSemanticsEndToEnd(t *testing.T) {
	if os.Getenv("EVENT_SEMANTICS_SYNTHETIC_E2E") != "1" {
		t.Skip("set EVENT_SEMANTICS_SYNTHETIC_E2E=1 to run the isolated cross-service acceptance")
	}
	dataDatabaseURL := requireSyntheticEnvironment(t, "TIDEWISE_TEST_DATABASE_URL")
	agentRunDatabaseURL := requireSyntheticEnvironment(t, "AGENTRUN_TEST_DATABASE_URL")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	t.Cleanup(cancel)

	repositoryRoot := syntheticRepositoryRoot(t)
	isolatedDataURL := createSyntheticDatabase(t, ctx, dataDatabaseURL, "tw_semantic_acceptance")
	dataBaseURL := startSyntheticDataService(t, ctx, repositoryRoot, isolatedDataURL)
	dataDatabase, err := pgxpool.New(ctx, isolatedDataURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(dataDatabase.Close)
	seedSyntheticEntities(t, ctx, dataDatabase)
	assertSyntheticResearchOutputs(t, ctx, dataDatabase, 0, 0)

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
		BaseURL: dataBaseURL, ServiceToken: syntheticDataToken,
		Timeout: 5 * time.Second, MaxResponseBytes: 4 * 1024 * 1024,
	})
	if err != nil {
		t.Fatal(err)
	}

	generator := &syntheticQueuedModel{responses: []string{
		syntheticNativeCandidates(syntheticAcceptedEvidenceID),
		syntheticDirectImpact(syntheticAcceptedEvidenceID),
		syntheticNativeCandidates(syntheticQuarantinedEvidenceID),
		syntheticDirectImpact(syntheticQuarantinedEvidenceID),
	}}
	reviewer := &syntheticQueuedModel{responses: []string{
		syntheticReview("pass", syntheticAcceptedEvidenceID),
		syntheticReview("indeterminate", syntheticQuarantinedEvidenceID),
		syntheticReview("indeterminate", syntheticQuarantinedEvidenceID),
	}}
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

	seedSyntheticEvent(
		t, ctx, dataDatabase,
		syntheticAcceptedRawDocumentID, syntheticAcceptedEventID, syntheticAcceptedEvidenceID,
		"synthetic:accepted", "2026-07-28T08:00:00Z",
	)
	if err := semanticApplication.Tick(ctx); err != nil {
		t.Fatalf("run accepted synthetic Event: %v", err)
	}
	accepted, err := dataClient.GetEventSemantics(ctx, syntheticAcceptedEventID)
	if err != nil {
		t.Fatal(err)
	}
	assertSyntheticAcceptedSemantics(t, accepted)

	seedSyntheticEvent(
		t, ctx, dataDatabase,
		syntheticQuarantinedRawDocumentID, syntheticQuarantinedEventID,
		syntheticQuarantinedEvidenceID, "synthetic:quarantined", "2026-07-28T09:00:00Z",
	)
	if err := semanticApplication.Tick(ctx); err != nil {
		t.Fatalf("run quarantined synthetic Event: %v", err)
	}
	quarantined, err := dataClient.GetEventSemantics(ctx, syntheticQuarantinedEventID)
	if err != nil {
		t.Fatal(err)
	}
	assertSyntheticQuarantinedSemantics(t, quarantined)

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
	assertSyntheticResearchOutputs(t, ctx, dataDatabase, 0, 0)
	t.Logf(
		"accepted_event=%s accepted_submission=%s quarantined_event=%s quarantined_submission=%s executions=%d",
		syntheticAcceptedEventID, accepted.Submissions[0].SubmissionID,
		syntheticQuarantinedEventID, quarantined.Submissions[0].SubmissionID,
		executions.TotalItems,
	)
}

type syntheticQueuedModel struct {
	mu        sync.Mutex
	responses []string
}

func (m *syntheticQueuedModel) Generate(
	_ context.Context,
	_ []*schema.Message,
	_ ...model.Option,
) (*schema.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.responses) == 0 {
		return nil, errors.New("unexpected synthetic model call")
	}
	response := m.responses[0]
	m.responses = m.responses[1:]
	return schema.AssistantMessage(response, nil), nil
}

func (*syntheticQueuedModel) Stream(
	context.Context,
	[]*schema.Message,
	...model.Option,
) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("synthetic model does not stream")
}

func requireSyntheticEnvironment(t *testing.T, key string) string {
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

func createSyntheticDatabase(
	t *testing.T,
	ctx context.Context,
	baseURL string,
	prefix string,
) string {
	t.Helper()
	base, err := pgxpool.New(ctx, baseURL)
	if err != nil {
		t.Fatal(err)
	}
	databaseName := prefix + "_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	identifier := pgx.Identifier{databaseName}.Sanitize()
	if _, err := base.Exec(ctx, "CREATE DATABASE "+identifier); err != nil {
		base.Close()
		t.Fatal(err)
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Path = "/" + databaseName
	t.Cleanup(func() {
		cleanupContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = base.Exec(cleanupContext, "DROP DATABASE "+identifier+" WITH (FORCE)")
		base.Close()
	})
	return parsed.String()
}

func startSyntheticDataService(
	t *testing.T,
	ctx context.Context,
	repositoryRoot string,
	databaseURL string,
) string {
	t.Helper()
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	password, _ := parsed.User.Password()
	port := reserveSyntheticPort(t)
	temporary := t.TempDir()
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
	binaryDirectory := filepath.Join(temporary, "bin")
	if err := os.Mkdir(binaryDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	migrateBinary := filepath.Join(binaryDirectory, "data-migrate")
	serverBinary := filepath.Join(binaryDirectory, "data-server")
	buildSyntheticBinary(
		t, ctx, repositoryRoot, migrateBinary, "./analyse-data-service/backend/cmd/dbmigrate",
	)
	buildSyntheticBinary(
		t, ctx, repositoryRoot, serverBinary, "./analyse-data-service/backend/cmd/server",
	)
	environment := append(os.Environ(),
		"APP_ENV=local",
		"TIDEWISE_CONFIG_DIR="+configDirectory,
		"TIDEWISW_DB_PASSWORD="+password,
		"DATA_SERVICE_TOKEN="+syntheticDataToken,
		"PGOPTIONS=-c tidewise.phase_a_cleanup_write_authorized=reviewed_backup_verified "+
			"-c tidewise.external_identifier_schema_write_authorized=reviewed_backup_verified "+
			"-c tidewise.alliance_economy_schema_write_authorized=reviewed_local_cleanup_verified",
	)
	migration := exec.CommandContext(ctx, migrateBinary, "-apply")
	migration.Dir = repositoryRoot
	migration.Env = environment
	if output, err := migration.CombinedOutput(); err != nil {
		t.Fatalf("migrate synthetic Data database: %v\n%s", err, output)
	}

	command := exec.Command(serverBinary)
	command.Dir = repositoryRoot
	command.Env = environment
	var logs strings.Builder
	command.Stdout = &logs
	command.Stderr = &logs
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	finished := make(chan error, 1)
	go func() { finished <- command.Wait() }()
	stop := func() {
		if command.Process == nil {
			return
		}
		_ = command.Process.Kill()
		select {
		case <-finished:
		case <-time.After(5 * time.Second):
		}
	}
	t.Cleanup(stop)
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(10 * time.Second)
	for {
		request, _ := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/healthz", nil)
		request.Header.Set("Authorization", "Bearer "+syntheticDataToken)
		response, requestErr := http.DefaultClient.Do(request)
		if requestErr == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return baseURL
			}
		}
		select {
		case runErr := <-finished:
			t.Fatalf("synthetic Data Service exited: %v\n%s", runErr, logs.String())
		default:
		}
		if time.Now().After(deadline) {
			stop()
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

func seedSyntheticEntities(t *testing.T, ctx context.Context, database *pgxpool.Pool) {
	t.Helper()
	if _, err := database.Exec(ctx, `
INSERT INTO entity_nodes (
  id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status
) VALUES
  ($1, 'company:synthetic-wafer-fab', 'company', 'company',
   'Synthetic Wafer Fab', 'Synthetic Wafer Fab', ARRAY['SWF'], 'active'),
  ($2, 'product:synthetic-wafer', 'product', 'product',
   'Synthetic 8-inch Wafer', 'Synthetic 8-inch Wafer', ARRAY['Synthetic Wafer'], 'active')
`, syntheticCompanyID, syntheticProductID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(ctx, `
INSERT INTO entity_edges (
  id, from_entity_id, to_entity_id, relation_type, evidence_note, status
) VALUES ($1, $2, $3, 'produces', 'Synthetic acceptance fixture', 'active')
`, syntheticRelationID, syntheticCompanyID, syntheticProductID); err != nil {
		t.Fatal(err)
	}
}

func seedSyntheticEvent(
	t *testing.T,
	ctx context.Context,
	database *pgxpool.Pool,
	rawDocumentID string,
	eventID string,
	evidenceID string,
	dedupeKey string,
	occurredAt string,
) {
	t.Helper()
	contentHash := fmt.Sprintf("%x", sha256.Sum256([]byte("raw:"+eventID)))
	evidenceHash := fmt.Sprintf("%x", sha256.Sum256([]byte("evidence:"+evidenceID)))
	if _, err := database.Exec(ctx, `
INSERT INTO raw_documents (
  id, ingest_channel, source_type, source_name, source_url, title, content_text,
  raw_mime_type, language, published_at, collected_at, content_hash, ingest_status
) VALUES (
  $1, 'synthetic_acceptance', 'news', 'Synthetic Primary Source',
  'https://synthetic.invalid/wafer', 'Synthetic wafer production update',
  'Synthetic Wafer Fab production fell 10%', 'text/plain', 'en',
  $2, $2::timestamptz + interval '1 minute', $3, 'collected'
)
`, rawDocumentID, occurredAt, contentHash); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(ctx, `
INSERT INTO events (
  id, title, summary, event_time, first_seen_at, knowable_at,
  event_status, fact_status, dedupe_key, fact_payload
) VALUES (
  $1, 'Synthetic Wafer Fab production fell 10%',
  'Synthetic Wafer Fab confirmed a 10% production decline.',
  $3, $3::timestamptz + interval '1 minute', $3::timestamptz + interval '1 minute',
  'confirmed', 'verified', $2, '{"production_change_percent":-10}'::jsonb
)
`, eventID, dedupeKey, occurredAt); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(ctx, `
INSERT INTO event_sources (
  id, event_id, raw_document_id, source_level, evidence_excerpt, evidence_hash,
  evidence_relation, supports_fields, is_primary
) VALUES (
  $1, $2, $3, 'primary', 'Synthetic Wafer Fab production fell 10%',
  $4, 'supports',
  ARRAY['title','factual_summary','occurred_at','fact_payload'], true
)
`, evidenceID, eventID, rawDocumentID, evidenceHash); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(
		ctx, `UPDATE events SET primary_source_id = $2 WHERE id = $1`, eventID, evidenceID,
	); err != nil {
		t.Fatal(err)
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
		len(submission.EntityLinks) != 1 || submission.EntityLinks[0].Status != "accepted" ||
		len(submission.VariableSignals) != 1 || submission.VariableSignals[0].Status != "accepted" ||
		submission.VariableSignals[0].RecordID == "" ||
		len(submission.DirectImpacts) != 1 || submission.DirectImpacts[0].Status != "accepted" ||
		submission.DirectImpacts[0].RecordID == "" ||
		submission.AuditWorkPackage == nil ||
		submission.AuditWorkPackage.Evidence[0].RawDocumentID != syntheticAcceptedRawDocumentID ||
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

func assertSyntheticResearchOutputs(
	t *testing.T,
	ctx context.Context,
	database *pgxpool.Pool,
	wantThemes int,
	wantReasoningTrees int,
) {
	t.Helper()
	var themes, reasoningTrees int
	if err := database.QueryRow(ctx, `SELECT count(*) FROM research_themes`).Scan(&themes); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow(ctx, `SELECT count(*) FROM research_reasoning_trees`).Scan(&reasoningTrees); err != nil {
		t.Fatal(err)
	}
	if themes != wantThemes || reasoningTrees != wantReasoningTrees {
		t.Fatalf(
			"research outputs themes=%d reasoning_trees=%d, want %d/%d",
			themes, reasoningTrees, wantThemes, wantReasoningTrees,
		)
	}
}
