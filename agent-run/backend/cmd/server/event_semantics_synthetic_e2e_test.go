package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	syntheticChainID    = "23000000-0000-4000-8000-000000000001"
	syntheticIndustryID = "23000000-0000-4000-8000-000000000002"

	syntheticAcceptedRawDocumentID = "20000000-0000-4000-8000-000000000001"
	syntheticAcceptedEventID       = "20000000-0000-4000-8000-000000000002"
	syntheticAcceptedEvidenceID    = "20000000-0000-4000-8000-000000000003"

	syntheticQuarantinedEventID    = "21000000-0000-4000-8000-000000000002"
	syntheticQuarantinedEvidenceID = "21000000-0000-4000-8000-000000000003"

	syntheticForecastRawDocumentID = "24000000-0000-4000-8000-000000000001"
	syntheticForecastEventID       = "24000000-0000-4000-8000-000000000002"
	syntheticForecastEvidenceID    = "24000000-0000-4000-8000-000000000003"
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
	responseRecorder := &semanticContextResponseRecorder{base: http.DefaultTransport}
	dataClient, err := dataclient.New(dataclient.Config{
		BaseURL: dataFixture.baseURL, ServiceToken: syntheticDataToken,
		Timeout: 5 * time.Second, MaxResponseBytes: 1024 * 1024,
		HTTPClient: &http.Client{Transport: responseRecorder, Timeout: 5 * time.Second},
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

	for index := 0; index < 3; index++ {
		if err := semanticApplication.Tick(ctx); err != nil {
			diagnostics, _ := admin.New(store, admin.Registry{}, "dev")
			executions, _ := diagnostics.ListAgentExecutions(ctx, agentrun.ExecutionListQuery{
				AgentKey: eventsemantic.AgentKey, Page: 1, PageSize: 20, Ascending: true,
			})
			t.Fatalf("run synthetic Event %d: %v executions=%#v", index+1, err, executions)
		}
	}
	responseRecorder.assertCompactContext(t)
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
	forecast, err := dataClient.GetEventSemantics(ctx, syntheticForecastEventID)
	if err != nil {
		t.Fatal(err)
	}
	assertSyntheticForecastSemantics(t, forecast)

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
	if executions.TotalItems != 3 || len(executions.Items) != 3 {
		t.Fatalf("Event Semantic executions = %#v, want three", executions)
	}
	for _, execution := range executions.Items {
		if execution.Status != "succeeded" {
			t.Fatalf("execution %s status = %q, want succeeded", execution.ID, execution.Status)
		}
	}
	asOf := time.Now().UTC().Add(time.Minute).Truncate(time.Second)
	dataFixture.assertAnalysisContext(t, ctx, asOf)
	publication := syntheticResearchPublication(accepted, asOf)
	invalidLineage := cloneSyntheticResearchPublication(t, publication)
	firstTree := invalidLineage.ReasoningTrees[0].(map[string]any)
	nodes := firstTree["nodes"].([]any)
	firstNode := nodes[0].(map[string]any)
	signals := firstNode["signals"].([]any)
	firstSignal := signals[0].(map[string]any)
	lineage := firstSignal["lineage"].(map[string]any)
	lineage["evidence_hash"] = strings.Repeat("f", 64)
	dataFixture.publishResearch(t, ctx, invalidLineage, http.StatusUnprocessableEntity)
	dataFixture.assertEmptyResearch(t, ctx)

	fakeImpact := cloneSyntheticResearchPublication(t, publication)
	fakeImpactTree := fakeImpact.ReasoningTrees[0].(map[string]any)
	fakeImpactNodes := fakeImpactTree["nodes"].([]any)
	fakeImpactIncoming := fakeImpactNodes[1].(map[string]any)["incoming_lineage"].(map[string]any)
	fakeImpactIncoming["direct_impact_assertion_id"] = "99000000-0000-4000-8000-000000000001"
	dataFixture.publishResearch(t, ctx, fakeImpact, http.StatusUnprocessableEntity)
	dataFixture.assertEmptyResearch(t, ctx)

	wrongSubmission := cloneSyntheticResearchPublication(t, publication)
	wrongSubmissionTree := wrongSubmission.ReasoningTrees[0].(map[string]any)
	wrongSubmissionNode := wrongSubmissionTree["nodes"].([]any)[0].(map[string]any)
	wrongSubmissionSignal := wrongSubmissionNode["signals"].([]any)[0].(map[string]any)
	wrongSubmissionLineage := wrongSubmissionSignal["lineage"].(map[string]any)
	wrongSubmissionLineage["semantic_submission_id"] = "99000000-0000-4000-8000-000000000002"
	dataFixture.publishResearch(t, ctx, wrongSubmission, http.StatusUnprocessableEntity)
	dataFixture.assertEmptyResearch(t, ctx)

	wrongEventCoverage := cloneSyntheticResearchPublication(t, publication)
	wrongEventCoverage.Theme["events"].([]any)[0].(map[string]any)["event_id"] =
		syntheticForecastEventID
	wrongEventTree := wrongEventCoverage.ReasoningTrees[0].(map[string]any)
	wrongEventTree["events"].([]any)[0].(map[string]any)["event_id"] =
		syntheticForecastEventID
	dataFixture.publishResearch(t, ctx, wrongEventCoverage, http.StatusUnprocessableEntity)
	dataFixture.assertEmptyResearch(t, ctx)

	wrongImpactSnapshot := cloneSyntheticResearchPublication(t, publication)
	wrongImpactTree := wrongImpactSnapshot.ReasoningTrees[0].(map[string]any)
	wrongImpactNodes := wrongImpactTree["nodes"].([]any)
	wrongImpactIncoming := wrongImpactNodes[1].(map[string]any)["incoming_lineage"].(map[string]any)
	wrongImpactIncoming["affected_direction"] = "increase"
	dataFixture.publishResearch(t, ctx, wrongImpactSnapshot, http.StatusUnprocessableEntity)
	dataFixture.assertEmptyResearch(t, ctx)

	invalidContract := cloneSyntheticResearchPublication(t, publication)
	invalidContract.ReasoningTrees = []any{}
	dataFixture.publishResearch(t, ctx, invalidContract, http.StatusBadRequest)
	dataFixture.assertEmptyResearch(t, ctx)

	unknownField := cloneSyntheticResearchPublication(t, publication)
	unknownField.Theme["invented_field"] = true
	dataFixture.publishResearch(t, ctx, unknownField, http.StatusBadRequest)
	dataFixture.assertEmptyResearch(t, ctx)

	first := dataFixture.publishResearch(t, ctx, publication, http.StatusCreated)
	replayed := dataFixture.publishResearch(t, ctx, publication, http.StatusOK)
	if first.Result.ThemeID == "" || first.Result.ThemeID != replayed.Result.ThemeID ||
		!replayed.Result.Replayed {
		t.Fatalf("research publication replay mismatch: first=%#v replay=%#v", first, replayed)
	}
	dataFixture.assertPublishedResearchReadback(t, ctx, first)
	conflict := publication
	conflict.Theme["title"] = "Conflicting Synthetic Theme"
	dataFixture.publishResearch(t, ctx, conflict, http.StatusConflict)
	dataFixture.assertResearchPublication(t, ctx)
	t.Logf(
		"accepted_event=%s accepted_submission=%s forecast_event=%s forecast_submission=%s quarantined_event=%s quarantined_submission=%s executions=%d",
		syntheticAcceptedEventID, accepted.Submissions[0].SubmissionID,
		syntheticForecastEventID, forecast.Submissions[0].SubmissionID,
		syntheticQuarantinedEventID, quarantined.Submissions[0].SubmissionID,
		executions.TotalItems,
	)
}

func cloneSyntheticResearchPublication(
	t *testing.T,
	input syntheticResearchPublicationPayload,
) syntheticResearchPublicationPayload {
	t.Helper()
	payload, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	var cloned syntheticResearchPublicationPayload
	if err := json.Unmarshal(payload, &cloned); err != nil {
		t.Fatal(err)
	}
	return cloned
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
	case strings.Contains(payload, syntheticForecastEvidenceID):
		evidenceID = syntheticForecastEvidenceID
	case strings.Contains(payload, syntheticQuarantinedEvidenceID):
		evidenceID = syntheticQuarantinedEvidenceID
	default:
		return nil, errors.New("synthetic model input has no fixture Evidence")
	}
	if m.kind == "generator" {
		if strings.Contains(payload, "direct_targets_by_link_key") {
			return schema.AssistantMessage(syntheticDirectImpact(evidenceID), nil), nil
		}
		if strings.Contains(payload, "ChainNode 路由选择器") {
			return schema.AssistantMessage(
				`{"route_id":"chain-node-via-industry.v1","partition":"`+syntheticIndustryID+`","unresolved":false}`,
				nil,
			), nil
		}
		if strings.Contains(payload, "正式锚点选择器") {
			return schema.AssistantMessage(
				`{"anchor_entity_id":"`+syntheticIndustryID+`","unresolved":false}`,
				nil,
			), nil
		}
		if strings.Contains(payload, "ChainNode 消歧器") {
			return schema.AssistantMessage(
				`{"target_entity_id":"`+syntheticCompanyID+`","unresolved":false}`,
				nil,
			), nil
		}
		return schema.AssistantMessage(syntheticNativeCandidates(evidenceID), nil), nil
	}
	if m.kind == "reviewer" {
		decision := "indeterminate"
		if evidenceID == syntheticAcceptedEvidenceID ||
			evidenceID == syntheticForecastEvidenceID {
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

type semanticContextResponseRecorder struct {
	base  http.RoundTripper
	mu    sync.Mutex
	sizes []int
}

func (r *semanticContextResponseRecorder) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := r.base.RoundTrip(request)
	if err != nil || request.Method != http.MethodGet ||
		!strings.Contains(request.URL.Path, "/event-semantics/context-leases/") ||
		!strings.HasSuffix(request.URL.Path, "/context") {
		return response, err
	}
	payload, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		response.Body.Close()
		return nil, readErr
	}
	response.Body.Close()
	response.Body = io.NopCloser(bytes.NewReader(payload))
	r.mu.Lock()
	r.sizes = append(r.sizes, len(payload))
	r.mu.Unlock()
	return response, nil
}

func (r *semanticContextResponseRecorder) assertCompactContext(t *testing.T) {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.sizes) == 0 {
		t.Fatal("cross-service E2E did not observe a Semantic Context response")
	}
	for _, size := range r.sizes {
		if size >= 100_000 {
			t.Fatalf("compact Semantic Context response bytes=%d, want <100000", size)
		}
	}
}

type syntheticResearchPublicationResponse struct {
	Result struct {
		ThemeID        string            `json:"theme_id"`
		ReasoningTrees map[string]string `json:"reasoning_tree_ids_by_industry_chain_entity_id"`
		Replayed       bool              `json:"replayed"`
	} `json:"result"`
}

type syntheticResearchPublicationPayload struct {
	AnalysisBatchID      string         `json:"analysis_batch_id"`
	AnalysisAsOf         string         `json:"analysis_as_of"`
	DiscoveryWindowStart string         `json:"discovery_window_start"`
	DiscoveryWindowEnd   string         `json:"discovery_window_end"`
	Theme                map[string]any `json:"theme"`
	ReasoningTrees       []any          `json:"reasoning_trees"`
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

func (s syntheticDataService) assertResearchPublication(t *testing.T, ctx context.Context) {
	t.Helper()
	runSyntheticFixture(
		t, ctx, s.repositoryRoot, s.fixtureBinary, s.fixtureEnvironment,
		"-action", "assert-research-publication",
	)
}

func (s syntheticDataService) assertPublishedResearchReadback(
	t *testing.T,
	ctx context.Context,
	publication syntheticResearchPublicationResponse,
) {
	t.Helper()
	treeID := publication.Result.ReasoningTrees[syntheticChainID]
	if treeID == "" {
		t.Fatalf("publication has no Reason Tree for synthetic chain: %#v", publication)
	}
	var themeEnvelope struct {
		Result struct {
			Theme struct {
				ID                 string `json:"id"`
				Title              string `json:"title"`
				ReasoningTreeCount int    `json:"reasoning_tree_count"`
			} `json:"theme"`
		} `json:"result"`
	}
	s.getResearch(t, ctx,
		"/api/data/v1/research/themes/"+publication.Result.ThemeID+"?window_hours=24",
		&themeEnvelope,
	)
	if themeEnvelope.Result.Theme.ID != publication.Result.ThemeID ||
		themeEnvelope.Result.Theme.Title != "Synthetic wafer supply contraction" ||
		themeEnvelope.Result.Theme.ReasoningTreeCount != 1 {
		t.Fatalf("published Theme readback = %#v", themeEnvelope.Result.Theme)
	}

	var treeListEnvelope struct {
		Result struct {
			ReasoningTrees []struct {
				ID string `json:"reasoning_tree_id"`
			} `json:"reasoning_trees"`
		} `json:"result"`
	}
	s.getResearch(t, ctx,
		"/api/data/v1/research/themes/"+publication.Result.ThemeID+"/reasoning-trees",
		&treeListEnvelope,
	)
	if len(treeListEnvelope.Result.ReasoningTrees) != 1 ||
		treeListEnvelope.Result.ReasoningTrees[0].ID != treeID {
		t.Fatalf("published Reason Tree list readback = %#v", treeListEnvelope.Result)
	}

	var treeDetailEnvelope struct {
		Result struct {
			ThemeID       string   `json:"theme_id"`
			ImpactNodeIDs []string `json:"impact_node_ids"`
			ReasoningTree struct {
				ID    string `json:"reasoning_tree_id"`
				Nodes []struct {
					Signals []map[string]any `json:"signals"`
				} `json:"nodes"`
			} `json:"reasoning_tree"`
		} `json:"result"`
	}
	s.getResearch(t, ctx,
		"/api/data/v1/research/themes/"+publication.Result.ThemeID+
			"/reasoning-trees/"+treeID,
		&treeDetailEnvelope,
	)
	if treeDetailEnvelope.Result.ThemeID != publication.Result.ThemeID ||
		treeDetailEnvelope.Result.ReasoningTree.ID != treeID ||
		len(treeDetailEnvelope.Result.ImpactNodeIDs) != 1 ||
		len(treeDetailEnvelope.Result.ReasoningTree.Nodes) != 2 ||
		len(treeDetailEnvelope.Result.ReasoningTree.Nodes[0].Signals) != 1 ||
		len(treeDetailEnvelope.Result.ReasoningTree.Nodes[1].Signals) != 1 {
		t.Fatalf("published Reason Tree detail readback = %#v", treeDetailEnvelope.Result)
	}
}

func (s syntheticDataService) getResearch(
	t *testing.T,
	ctx context.Context,
	path string,
	target any,
) {
	t.Helper()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+syntheticDataToken)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		var failure any
		_ = json.NewDecoder(response.Body).Decode(&failure)
		t.Fatalf("research readback %s status=%d body=%#v", path, response.StatusCode, failure)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatal(err)
	}
}

func (s syntheticDataService) assertAnalysisContext(t *testing.T, ctx context.Context, asOf time.Time) {
	t.Helper()
	query := url.Values{
		"discovery_window_start": {"2026-07-28T00:00:00Z"},
		"discovery_window_end":   {asOf.Format(time.RFC3339)},
		"analysis_as_of":         {asOf.Format(time.RFC3339)},
		"page_size":              {"20"},
	}
	request, err := http.NewRequestWithContext(
		ctx, http.MethodGet, s.baseURL+"/api/data/v1/research-analysis-context?"+query.Encode(), nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+syntheticDataToken)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var envelope struct {
		Result struct {
			ContractVersion             string  `json:"contract_version"`
			TBoxContractVersion         string  `json:"tbox_contract_version"`
			TemporalSemantics           string  `json:"temporal_semantics"`
			EventPageFingerprint        string  `json:"event_page_fingerprint"`
			ReferenceClosureFingerprint string  `json:"reference_closure_fingerprint"`
			DictionaryFingerprint       *string `json:"dictionary_fingerprint"`
			Bundles                     []struct {
				Event struct {
					ID string `json:"id"`
				} `json:"event"`
				Evidence []struct {
					PublishedAt     *string `json:"published_at"`
					StatementSource string  `json:"statement_source"`
				} `json:"evidence"`
				VariableSignals []struct {
					ID                  string           `json:"variable_signal_id"`
					AssertionModality   string           `json:"assertion_modality"`
					StatementAt         *string          `json:"statement_at"`
					ForecastPeriodStart *string          `json:"forecast_period_start"`
					ForecastPeriodEnd   *string          `json:"forecast_period_end"`
					Measurements        []map[string]any `json:"measurements"`
					DirectImpacts       []map[string]any `json:"direct_impacts"`
				} `json:"variable_signals"`
			} `json:"event_semantic_bundles"`
		} `json:"result"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK ||
		envelope.Result.ContractVersion != "research-analysis-context.v1" ||
		envelope.Result.TBoxContractVersion == "" ||
		envelope.Result.TemporalSemantics != "retrospective_reconstruction" {
		t.Fatalf("Analysis Context status=%d result=%#v", response.StatusCode, envelope.Result)
	}
	if len(envelope.Result.EventPageFingerprint) != 64 ||
		len(envelope.Result.ReferenceClosureFingerprint) != 64 ||
		envelope.Result.DictionaryFingerprint != nil {
		t.Fatalf("corrected Analysis Context v1 fingerprints = %#v", envelope.Result)
	}
	var acceptedFound, forecastFound bool
	for _, bundle := range envelope.Result.Bundles {
		switch bundle.Event.ID {
		case syntheticAcceptedEventID:
			if len(bundle.Evidence) == 0 || len(bundle.VariableSignals) != 1 ||
				len(bundle.VariableSignals[0].DirectImpacts) != 1 {
				t.Fatalf("accepted Analysis Context bundle = %#v", bundle)
			}
			acceptedFound = true
		case syntheticForecastEventID:
			if len(bundle.Evidence) == 0 ||
				bundle.Evidence[0].PublishedAt == nil ||
				bundle.Evidence[0].StatementSource != "Synthetic Wafer Fab" ||
				len(bundle.VariableSignals) != 1 ||
				bundle.VariableSignals[0].AssertionModality != "source_forecast" ||
				bundle.VariableSignals[0].StatementAt == nil ||
				bundle.VariableSignals[0].ForecastPeriodStart == nil ||
				bundle.VariableSignals[0].ForecastPeriodEnd == nil ||
				len(bundle.VariableSignals[0].Measurements) != 1 ||
				bundle.VariableSignals[0].DirectImpacts == nil ||
				len(bundle.VariableSignals[0].DirectImpacts) != 0 {
				t.Fatalf("forecast Analysis Context bundle = %#v", bundle)
			}
			forecastFound = true
		}
	}
	if !acceptedFound || !forecastFound {
		t.Fatalf("accepted/forecast Events missing from Analysis Context: %#v", envelope.Result.Bundles)
	}
}

func (s syntheticDataService) publishResearch(
	t *testing.T,
	ctx context.Context,
	payload syntheticResearchPublicationPayload,
	wantStatus int,
) syntheticResearchPublicationResponse {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(
		ctx, http.MethodPost, s.baseURL+"/api/data/v1/research-theme-imports",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+syntheticDataToken)
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var result syntheticResearchPublicationResponse
	if response.StatusCode == http.StatusCreated || response.StatusCode == http.StatusOK {
		if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
			t.Fatal(err)
		}
	} else {
		var failure any
		if err := json.NewDecoder(response.Body).Decode(&failure); err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != wantStatus {
			t.Fatalf("publish research status=%d want=%d body=%#v", response.StatusCode, wantStatus, failure)
		}
		return result
	}
	if response.StatusCode != wantStatus {
		t.Fatalf("publish research status=%d want=%d result=%#v", response.StatusCode, wantStatus, result)
	}
	return result
}

func syntheticResearchPublication(
	semantics eventsemantic.EventSemantics,
	asOf time.Time,
) syntheticResearchPublicationPayload {
	submission := semantics.Submissions[0]
	signalID := submission.VariableSignals[0].RecordID
	impactID := submission.DirectImpacts[0].RecordID
	evidenceHash := fmt.Sprintf(
		"%x", sha256.Sum256([]byte("evidence:"+syntheticAcceptedEvidenceID)),
	)
	theme := map[string]any{
		"theme_key": "synthetic-wafer-supply", "title": "Synthetic wafer supply contraction",
		"one_line_conclusion":  "Lower fab production tightens synthetic wafer supply.",
		"conclusion_direction": "positive", "impact_strength": "medium",
		"attention_level": "high", "conclusion_status": "supported",
		"transmission_stage": "validation", "investment_guidance_action": "focus",
		"investment_guidance_summary": "Track product supply and producer output.",
		"time_horizon_category":       "short_term", "time_horizon_summary": "Near-term fixture",
		"transmission_summary": "Production decline transmits directly to product supply.",
		"checkpoint_summary":   "Watch production recovery.", "risk_summary": "Fixture only.",
		"impacts": []any{map[string]any{
			"chain_node_entity_id": syntheticProductID, "relation_role": "beneficiary",
			"impact_direction": "positive", "impact_summary": "Tighter supply", "display_order": 1,
		}},
		"events": []any{map[string]any{
			"event_id": syntheticAcceptedEventID, "evidence_role": "driver",
			"supported_claim": "Production fell 10%.",
		}},
	}
	formalSignalLineage := map[string]any{
		"source_kind": "formal_signal", "variable_signal_id": signalID,
		"semantic_submission_id": submission.SubmissionID,
		"evidence_id":            syntheticAcceptedEvidenceID, "evidence_hash": evidenceHash,
		"upstream_variable_signal_id": nil, "upstream_direct_impact_assertion_id": nil,
		"entity_relation_id": nil, "industry_chain_graph_edge_id": nil,
	}
	formalImpactLineage := map[string]any{
		"source_kind": "formal_direct_impact", "direct_impact_assertion_id": impactID,
		"semantic_submission_id": submission.SubmissionID,
		"evidence_id":            syntheticAcceptedEvidenceID, "evidence_hash": evidenceHash,
		"affected_variable_key": "market_supply", "affected_direction": "decrease",
		"upstream_variable_signal_id": nil, "upstream_direct_impact_assertion_id": nil,
		"entity_relation_id": nil,
	}
	inferenceLineage := map[string]any{
		"source_kind": "analyst_inference", "variable_signal_id": nil,
		"semantic_submission_id": nil, "evidence_id": nil, "evidence_hash": nil,
		"upstream_variable_signal_id": nil, "upstream_direct_impact_assertion_id": impactID,
		"entity_relation_id": syntheticRelationID, "industry_chain_graph_edge_id": nil,
	}
	tree := map[string]any{
		"industry_chain_entity_id": syntheticChainID, "title": "Synthetic wafer chain",
		"display_order": 1, "one_line_conclusion": "Production decline tightens product supply.",
		"fact_summary":         "Production fell 10%.",
		"transmission_summary": "Producer output maps to product supply.",
		"impact_direction":     "positive", "impact_strength": "medium",
		"impact_summary": "Synthetic product scarcity", "conclusion_boundary_summary": "Fixture boundary",
		"support_summary": "Formal Event semantics", "counter_summary": "No counter signal",
		"invalidation_conditions": []string{"Production recovers"},
		"checkpoints":             []any{map[string]any{"type": "metric", "summary": "Production volume"}},
		"events": []any{map[string]any{
			"event_id": syntheticAcceptedEventID, "evidence_role": "driver", "display_order": 1,
		}},
		"nodes": []any{
			map[string]any{
				"position": 1, "chain_node_entity_id": syntheticCompanyID,
				"state_summary": "Producer output fell", "impact_direction": "negative",
				"impact_strength": "medium", "impact_summary": "Lower output",
				"reasoning_basis_summary": "Formal production Signal", "evidence_gap_summary": nil,
				"incoming_industry_chain_graph_edge_id": nil, "incoming_transmission_title": nil,
				"incoming_transmission_mechanism": nil, "incoming_condition_summary": nil,
				"incoming_lineage": nil,
				"signals": []any{map[string]any{
					"variable_signal_key": "production_volume", "signal_role": "primary",
					"signal_direction": "decrease", "display_summary": "Production fell 10%",
					"display_order": 1, "lineage": formalSignalLineage,
				}},
			},
			map[string]any{
				"position": 2, "chain_node_entity_id": syntheticProductID,
				"state_summary": "Product supply tightens", "impact_direction": "positive",
				"impact_strength": "medium", "impact_summary": "Scarcity rises",
				"reasoning_basis_summary": "Formal one-hop impact", "evidence_gap_summary": nil,
				"incoming_industry_chain_graph_edge_id": nil,
				"incoming_transmission_title":           "Production to supply",
				"incoming_transmission_mechanism":       "Lower producer output reduces product supply.",
				"incoming_condition_summary":            "The production relation remains active.",
				"incoming_lineage":                      formalImpactLineage,
				"signals": []any{map[string]any{
					"variable_signal_key": "market_supply", "signal_role": "primary",
					"signal_direction": "decrease", "display_summary": "Product supply decreases",
					"display_order": 1, "lineage": inferenceLineage,
				}},
			},
		},
	}
	return syntheticResearchPublicationPayload{
		AnalysisBatchID:      "synthetic-theme-publication-1",
		AnalysisAsOf:         asOf.Format(time.RFC3339),
		DiscoveryWindowStart: "2026-07-28T00:00:00Z",
		DiscoveryWindowEnd:   asOf.Format(time.RFC3339),
		Theme:                theme, ReasoningTrees: []any{tree},
	}
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
	if evidenceID == syntheticForecastEvidenceID {
		return fmt.Sprintf(
			`{"mentions":[{"candidate_key":"company","mention":"upstream wafer capacity stage","predicted_entity_type":"chain_node","entity_role":"statement_source","evidence_ids":["%s"],"resolution_confidence":"0.99"}],"variable_signals":[{"candidate_key":"forecast-demand","subject_link_key":"company","variable_key":"market_demand","variable_version":1,"direction":"increase","assertion_modality":"source_forecast","evidence_ids":["%s"],"statement_at":"2026-07-28T10:00:00Z","valid_from":"2026-08-01T00:00:00Z","valid_until":"2026-09-30T23:59:59Z","forecast_period_start":"2026-08-01T00:00:00Z","forecast_period_end":"2026-09-30T23:59:59Z","measurements":[{"measurement_role":"relative_change","value_shape":"exact","raw_value":"12","raw_unit":"%%","canonical_value":"12","canonical_unit":"percent","raw_text":"forecasts wafer demand growth of 12 percent","is_approximate":false,"evidence_id":"%s"}],"extraction_confidence":"0.97"}]}`,
			evidenceID, evidenceID, evidenceID,
		)
	}
	return fmt.Sprintf(
		`{"mentions":[{"candidate_key":"company","mention":"upstream wafer capacity stage","predicted_entity_type":"chain_node","entity_role":"event_subject","evidence_ids":["%s"],"resolution_confidence":"0.99"}],"variable_signals":[{"candidate_key":"production","subject_link_key":"company","variable_key":"production_volume","variable_version":1,"direction":"decrease","assertion_modality":"actual","evidence_ids":["%s"],"measurements":[{"measurement_role":"relative_change","value_shape":"exact","raw_value":"-10","raw_unit":"%%","canonical_value":"-10","canonical_unit":"percent","raw_text":"production fell 10%%","is_approximate":false,"evidence_id":"%s"}],"extraction_confidence":"0.98"}]}`,
		evidenceID, evidenceID, evidenceID,
	)
}

func syntheticDirectImpact(evidenceID string) string {
	if evidenceID == syntheticForecastEvidenceID {
		return `{"direct_impacts":[]}`
	}
	return fmt.Sprintf(
		`{"direct_impacts":[{"candidate_key":"supply","source_signal_key":"production","target_entity_id":"%s","affected_variable_key":"market_supply","affected_variable_version":1,"affected_direction":"decrease","derivation_type":"rule_inferred","mechanism_summary":"Synthetic wafer production feeds Synthetic 8-inch Wafer Supply; lower production reduces directly available supply.","entity_relation_id":"%s","rule_key":"synthetic_production_decrease_reduces_chain_supply","rule_version":1,"evidence_ids":["%s"],"assertion_confidence":"0.96"}]}`,
		syntheticProductID, syntheticRelationID, evidenceID,
	)
}

func syntheticReview(decision string, evidenceID string) string {
	if evidenceID == syntheticForecastEvidenceID {
		return fmt.Sprintf(
			`{"items":[{"candidate_type":"entity_link","candidate_key":"company","decision":"%s","reason_codes":["synthetic_review"],"evidence_ids":["%s"]},{"candidate_type":"variable_signal","candidate_key":"forecast-demand","decision":"%s","reason_codes":["synthetic_review"],"evidence_ids":["%s"]}]}`,
			decision, evidenceID, decision, evidenceID,
		)
	}
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
		submission.AuditWorkPackage.EntityLinks[0].Mention != "upstream wafer capacity stage" ||
		submission.AuditWorkPackage.EntityLinks[0].EntityID != syntheticCompanyID ||
		len(submission.AuditWorkPackage.EntityLinks[0].EvidenceIDs) != 1 ||
		submission.AuditWorkPackage.EntityLinks[0].EvidenceIDs[0] != syntheticAcceptedEvidenceID ||
		submission.AuditWorkPackage.EntityLinks[0].ResolutionReceipt == nil ||
		submission.AuditWorkPackage.EntityLinks[0].ResolutionReceipt.AnchorEntityID != syntheticIndustryID ||
		submission.AuditWorkPackage.EntityLinks[0].ResolutionReceipt.TargetEntityID != syntheticCompanyID ||
		submission.AuditWorkPackage.VariableSignals[0].SubjectLinkKey !=
			submission.AuditWorkPackage.EntityLinks[0].CandidateKey ||
		len(submission.AuditWorkPackage.VariableSignals[0].EvidenceIDs) != 1 ||
		submission.AuditWorkPackage.VariableSignals[0].EvidenceIDs[0] != syntheticAcceptedEvidenceID ||
		submission.AuditWorkPackage.Evidence[0].RawDocumentID != syntheticAcceptedRawDocumentID ||
		submission.AuditWorkPackage.DirectImpacts[0].RuleKey !=
			"synthetic_production_decrease_reduces_chain_supply" ||
		submission.AuditWorkPackage.DirectImpacts[0].RuleVersion != 1 ||
		len(submission.ReviewSnapshots) != 1 {
		t.Fatalf("accepted Submission = %#v", submission)
	}
}

func assertSyntheticForecastSemantics(t *testing.T, semantics eventsemantic.EventSemantics) {
	t.Helper()
	if semantics.EventID != syntheticForecastEventID || len(semantics.Submissions) != 1 {
		t.Fatalf("forecast Event semantics = %#v", semantics)
	}
	submission := semantics.Submissions[0]
	if submission.Status != "accepted" ||
		len(submission.EntityLinks) != 1 ||
		submission.EntityLinks[0].Status != "accepted" ||
		len(submission.VariableSignals) != 1 ||
		submission.VariableSignals[0].Status != "accepted" ||
		submission.VariableSignals[0].RecordID == "" ||
		submission.DirectImpacts == nil ||
		len(submission.DirectImpacts) != 0 ||
		submission.AuditWorkPackage == nil ||
		len(submission.AuditWorkPackage.Evidence) != 1 ||
		submission.AuditWorkPackage.Evidence[0].RawDocumentID != syntheticForecastRawDocumentID ||
		submission.AuditWorkPackage.Evidence[0].PublishedAt == nil ||
		submission.AuditWorkPackage.Evidence[0].StatementSource != "Synthetic Wafer Fab" ||
		len(submission.AuditWorkPackage.VariableSignals) != 1 {
		t.Fatalf("forecast Submission = %#v", submission)
	}
	signal := submission.AuditWorkPackage.VariableSignals[0]
	if signal.AssertionModality != "source_forecast" ||
		signal.StatementAt == nil ||
		signal.ValidFrom == nil ||
		signal.ValidUntil == nil ||
		signal.ForecastPeriodStart == nil ||
		signal.ForecastPeriodEnd == nil ||
		signal.ExtractionConfidence != "0.97000" &&
			signal.ExtractionConfidence != "0.97" ||
		len(signal.Measurements) != 1 {
		t.Fatalf("forecast Variable Signal = %#v", signal)
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
