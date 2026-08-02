package workflow

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact"
)

type fakeModel struct {
	responses []string
	calls     int
	toolName  string
}

type scriptedToolModel struct {
	responses []*schema.Message
	errors    []error
	calls     int
}

func (m *scriptedToolModel) Generate(
	context.Context, []*schema.Message, ...model.Option,
) (*schema.Message, error) {
	if m.calls >= len(m.responses) {
		if m.calls < len(m.errors) {
			err := m.errors[m.calls]
			m.calls++
			return nil, err
		}
		return nil, errors.New("unexpected model call")
	}
	response := m.responses[m.calls]
	m.calls++
	return response, nil
}

func TestForcedFunctionCallPreservesContextCancellation(t *testing.T) {
	chatModel := &scriptedToolModel{errors: []error{context.Canceled}}
	var output extractionOutput
	_, err := generateToolResult(
		context.Background(), chatModel, extractionFunctionName, "test",
		[]*schema.Message{schema.UserMessage("input")}, &output, nil,
	)
	if !errors.Is(err, context.Canceled) || chatModel.calls != 1 {
		t.Fatalf("error = %v, calls = %d", err, chatModel.calls)
	}
}

func (*scriptedToolModel) Stream(
	context.Context, []*schema.Message, ...model.Option,
) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("unexpected stream call")
}

func (m *scriptedToolModel) WithTools([]*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

func functionResponse(name, arguments string, extraCalls ...schema.ToolCall) *schema.Message {
	calls := []schema.ToolCall{{
		ID: "call-1", Type: "function",
		Function: schema.FunctionCall{Name: name, Arguments: arguments},
	}}
	calls = append(calls, extraCalls...)
	return schema.AssistantMessage("", calls)
}

func TestForcedFunctionCallUsesOneBoundedCorrection(t *testing.T) {
	valid := functionResponse(extractionFunctionName, `{"documents":[]}`)
	valid.ResponseMeta = &schema.ResponseMeta{FinishReason: "tool_calls"}
	wrong := functionResponse("wrong_function", `{"documents":[]}`)
	multiple := functionResponse(extractionFunctionName, `{"documents":[]}`, schema.ToolCall{
		ID: "call-2", Type: "function",
		Function: schema.FunctionCall{Name: extractionFunctionName, Arguments: `{"documents":[]}`},
	})
	tests := []struct {
		name      string
		responses []*schema.Message
		wantError bool
	}{
		{name: "expected call", responses: []*schema.Message{valid}},
		{name: "missing call repaired", responses: []*schema.Message{schema.AssistantMessage("prose", nil), valid}},
		{name: "wrong function", responses: []*schema.Message{wrong, wrong}, wantError: true},
		{name: "multiple functions", responses: []*schema.Message{multiple, multiple}, wantError: true},
		{name: "malformed arguments", responses: []*schema.Message{
			functionResponse(extractionFunctionName, `{`),
			functionResponse(extractionFunctionName, `{}`),
		}, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chatModel := &scriptedToolModel{responses: test.responses}
			var output extractionOutput
			observation, err := generateToolResult(
				context.Background(), chatModel, extractionFunctionName, "test",
				[]*schema.Message{schema.UserMessage("input")}, &output, nil,
			)
			if (err != nil) != test.wantError {
				t.Fatalf("error = %v, wantError %v", err, test.wantError)
			}
			if test.wantError {
				var contractFailure *ModelContractFailure
				if !errors.As(err, &contractFailure) || contractFailure.Stage != "extraction" ||
					contractFailure.Violation == "" {
					t.Fatalf("contract failure = %#v, error = %v", contractFailure, err)
				}
			}
			wantCalls := 1
			if test.wantError || len(test.responses) == 2 {
				wantCalls = 2
			}
			if chatModel.calls != wantCalls {
				t.Fatalf("calls = %d, want %d", chatModel.calls, wantCalls)
			}
			if !test.wantError && (observation.Stage != "extraction" ||
				observation.CallCount != wantCalls || observation.ArgumentBytes == 0) {
				t.Fatalf("observation = %#v", observation)
			}
		})
	}
}

func TestForcedFunctionCallRepairsInputCoverageViolation(t *testing.T) {
	chatModel := &scriptedToolModel{responses: []*schema.Message{
		functionResponse(extractionFunctionName, `{"documents":[]}`),
		functionResponse(extractionFunctionName, `{"documents":[{"artifact_id":"artifact-1","events":[],"no_event_reason":"无可发布事件"}]}`),
	}}
	var output extractionOutput
	observation, err := generateToolResult(
		context.Background(), chatModel, extractionFunctionName, "test",
		[]*schema.Message{schema.UserMessage("input")}, &output,
		func(candidateOutput *extractionOutput) error {
			_, _, validationErr := convertExtraction(
				[]eventfact.Artifact{{ArtifactID: "artifact-1"}}, *candidateOutput,
			)
			return validationErr
		},
	)
	if err != nil {
		t.Fatalf("generate tool result: %v", err)
	}
	if chatModel.calls != 2 || observation.CallCount != 2 || len(output.Documents) != 1 {
		t.Fatalf("calls = %d, observation = %#v, output = %#v", chatModel.calls, observation, output)
	}
}

func (f *fakeModel) Generate(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error) {
	if f.calls >= len(f.responses) {
		return nil, errors.New("unexpected model call")
	}
	response := f.responses[f.calls]
	f.calls++
	return schema.AssistantMessage("", []schema.ToolCall{{
		ID: "call-1", Type: "function",
		Function: schema.FunctionCall{Name: f.toolName, Arguments: response},
	}}), nil
}

func (*fakeModel) Stream(context.Context, []*schema.Message, ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("unexpected stream call")
}

func (f *fakeModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	if len(tools) != 1 {
		return nil, errors.New("expected one bound tool")
	}
	f.toolName = tools[0].Name
	return f, nil
}

type fakeReader struct {
	artifacts []eventfact.Artifact
}

type fakeCanonicalReader struct {
	events []eventfact.CanonicalEvent
}

func (f fakeCanonicalReader) FindCanonicalEvents(context.Context, []string) ([]eventfact.CanonicalEvent, error) {
	return append([]eventfact.CanonicalEvent(nil), f.events...), nil
}

func (f fakeReader) Read(context.Context, []string) ([]eventfact.Artifact, error) {
	return append([]eventfact.Artifact(nil), f.artifacts...), nil
}

func TestWorkflowAutoApprovesSingleVerifiedSource(t *testing.T) {
	extractor := &fakeModel{responses: []string{`{
		"documents":[{
			"artifact_id":"sha256:artifact",
			"no_event_reason":"",
			"events":[{
				"title":"某公司宣布新增产线",
				"factual_summary":"某公司于2026年7月26日宣布新增一条产线。",
				"occurred_at":"2026-07-26T00:00:00Z",
				"evidence_statement":"某公司于2026年7月26日宣布新增一条产线。",
				"actor_mentions":["某公司"],
				"action":"新增",
				"object_mentions":["产线"],
				"change":{},
				"lifecycle_status":"announced",
				"time_precision":"day",
				"location_mentions":[],
				"reference_period":"",
				"quantities":["一条"]
			}]
		}]
	}`, tagClassificationJSON()}}
	reviewer := &fakeModel{responses: []string{`{
		"reviews":[{
			"candidate_id":"candidate:1",
			"semantic_pass":true,
			"conflict":false,
			"reasons":["事实表达与逐字证据一致"],
			"confidence":0.98
		}]
	}`}}
	artifact := eventfact.Artifact{
		ArtifactID: "sha256:artifact", CollectorExecutionID: "11111111-1111-4111-8111-111111111111",
		DocumentID: "sha256:artifact", Title: "某公司宣布新增产线",
		SourceName: "公司公告", SourceType: "official", SourceURL: "https://example.com/1",
		ContentLevel: "full_text", CollectedAt: time.Date(2026, 7, 26, 1, 0, 0, 0, time.UTC),
		ContentSHA256: strings.Repeat("a", 64),
		Body:          "# 某公司宣布新增产线\n\n某公司于2026年7月26日宣布新增一条产线。",
	}
	runnable, err := New(context.Background(), fakeReader{artifacts: []eventfact.Artifact{artifact}}, fakeCanonicalReader{}, extractor, reviewer)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runnable.Invoke(context.Background(), &Input{
		ArtifactID: artifact.ArtifactID,
		Attempt: eventfact.ExecutionAttempt{
			ID: "22222222-2222-4222-8222-222222222222",
			WorkItem: eventfact.WorkItem{
				CollectorExecutionIDs: []string{"11111111-1111-4111-8111-111111111111"},
			},
		},
		Catalog: eventfact.TagCatalog{
			Revision: "event-tags:r", Hash: strings.Repeat("b", 64),
			Tags: []eventfact.Tag{{
				ID: "33333333-3333-4333-8333-333333333333", Kind: "news_category",
				Code: "technology", Name: "科技", IsActive: true,
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if extractor.calls != 2 || reviewer.calls != 1 ||
		result.ExtractionModelCalls != 2 || result.ReviewModelCalls != 1 {
		t.Fatalf("model calls extraction=%d review=%d result=%#v", extractor.calls, reviewer.calls, result)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].ReviewState != eventfact.ReviewAutoApproved ||
		result.Candidates[0].Tags[0].ID != "33333333-3333-4333-8333-333333333333" ||
		result.Candidates[0].DedupeKey == "" || result.Candidates[0].IdentityHash == "" ||
		result.Candidates[0].FactPayload["action"] != "新增" ||
		result.Candidates[0].FactPayload["lifecycle_status"] != "announced" ||
		result.Candidates[0].SourceLevel != "primary" ||
		!containsString(result.Candidates[0].SupportsFields, "occurred_at") {
		t.Fatalf("candidate = %#v", result.Candidates)
	}
}

func TestFailureCodeClassifiesSafeDeterministicModelFailures(t *testing.T) {
	for _, test := range []struct {
		err  error
		code string
	}{
		{
			err:  errors.New("[NodeRunError] Event Fact extraction response is invalid"),
			code: "event_fact_model_response_invalid",
		},
		{
			err:  errors.New("[NodeRunError] Event extraction did not account for every Artifact"),
			code: "event_fact_model_output_incomplete",
		},
		{
			err:  errors.New("unexpected internal failure"),
			code: "event_fact_workflow_rejected",
		},
		{
			err: &ModelContractFailure{
				Stage: "tag_assignment", Violation: "argument_type_invalid_assignments",
			},
			code: "event_fact_tag_assignment_contract_argument_type_invalid_assignments",
		},
	} {
		if got := FailureCode(test.err); got != test.code {
			t.Fatalf("FailureCode(%v) = %q, want %q", test.err, got, test.code)
		}
	}
}

func TestArtifactUnitCollapsesDuplicateDeterministicEventCandidates(t *testing.T) {
	candidate := eventfact.Candidate{
		CandidateID: "candidate:1", ArtifactID: "sha256:artifact",
		Title: "某公司宣布扩产", FactualSummary: "某公司宣布扩产。",
		FactPayload:   map[string]any{"action": "扩产"},
		ActorMentions: []string{"某公司"}, Action: "扩产",
		ObjectMentions: []string{"产线"}, LifecycleStatus: "announced",
		TimePrecision: "unknown",
	}
	duplicate := candidate
	duplicate.CandidateID = "candidate:2"
	candidates := []eventfact.Candidate{candidate, duplicate}
	applyDeterministicIdentities(candidates)
	candidates = dedupeExactUnitCandidates(candidates)
	if len(candidates) != 1 || candidates[0].CandidateID != "candidate:1" ||
		candidates[0].DedupeKey == "" {
		t.Fatalf("deduplicated Artifact Unit candidates = %#v", candidates)
	}
}

func TestWorkflowResumesPersistedFactsAtCatalogClassificationBoundary(t *testing.T) {
	extractor := &fakeModel{responses: []string{
		`{"assignments":[{"candidate_id":"candidate:1","tag_codes":["technology"]}]}`,
	}}
	reviewer := &fakeModel{responses: []string{
		`{"reviews":[{"candidate_id":"candidate:1","semantic_pass":true,"conflict":false,"reasons":["事实表达与逐字证据一致"],"confidence":0.98}]}`,
	}}
	runnable, err := New(
		context.Background(),
		fakeReader{artifacts: []eventfact.Artifact{testArtifact()}},
		fakeCanonicalReader{},
		extractor,
		reviewer,
	)
	if err != nil {
		t.Fatal(err)
	}
	partialCandidate := eventfact.Candidate{
		CandidateID: "candidate:1", ArtifactID: "sha256:artifact",
		Title: "某公司宣布新增产线", FactualSummary: "某公司于2026年7月26日宣布新增一条产线。",
		FactPayload:       map[string]any{"action": "新增产线"},
		EvidenceStatement: "某公司于2026年7月26日宣布新增一条产线。",
		SupportsFields:    []string{"title", "factual_summary", "fact_payload"},
		SourceLevel:       "primary", ActorMentions: []string{"某公司"}, Action: "新增",
		ObjectMentions: []string{"产线"}, LifecycleStatus: "announced",
		TimePrecision: "unknown",
	}
	result, err := runnable.Invoke(context.Background(), &Input{
		ArtifactID: "sha256:artifact",
		Attempt:    testInput().Attempt,
		Catalog:    testInput().Catalog,
		ResumeResult: &eventfact.Result{
			Candidates:           []eventfact.Candidate{partialCandidate},
			ExtractionModelCalls: 1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if extractor.calls != 1 || reviewer.calls != 1 ||
		result.ExtractionModelCalls != 1 || result.ReviewModelCalls != 1 ||
		result.Candidates[0].ReviewState != eventfact.ReviewAutoApproved {
		t.Fatalf(
			"resumed workflow extractor=%d reviewer=%d result=%#v",
			extractor.calls, reviewer.calls, result,
		)
	}
}

func TestWorkflowRejectsSemanticConflictWithoutHumanReview(t *testing.T) {
	extractor := &fakeModel{responses: []string{validExtractionJSON(), tagClassificationJSON()}}
	reviewer := &fakeModel{responses: []string{`{"reviews":[{"candidate_id":"candidate:1","semantic_pass":false,"conflict":true,"reasons":["证据存在语义冲突"],"confidence":0.7}]}`}}
	runnable, err := New(context.Background(), fakeReader{artifacts: []eventfact.Artifact{testArtifact()}}, fakeCanonicalReader{}, extractor, reviewer)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runnable.Invoke(context.Background(), testInput())
	if err != nil {
		t.Fatal(err)
	}
	if result.Candidates[0].ReviewState != eventfact.ReviewRejected {
		t.Fatalf("review state = %q", result.Candidates[0].ReviewState)
	}
}

func TestWorkflowReusesAgentRunCanonicalCoreFacts(t *testing.T) {
	identityCandidate := []eventfact.Candidate{{
		ActorMentions: []string{"某公司"}, Action: "新增", ObjectMentions: []string{"产线"},
		LifecycleStatus: "announced", TimePrecision: "day", LocationMentions: []string{},
	}}
	occurredAt := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	identityCandidate[0].OccurredAt = &occurredAt
	applyDeterministicIdentities(identityCandidate)
	canonicalCore := []byte(`{"title":"已发布规范标题","factual_summary":"已发布且不可变的事实摘要。","occurred_at":"2026-07-26T00:00:00Z","fact_payload":{"action":"既有核心事实"}}`)
	extractor := &fakeModel{responses: []string{validExtractionJSON(), tagClassificationJSON()}}
	reviewer := &fakeModel{responses: []string{`{"reviews":[{"candidate_id":"candidate:1","semantic_pass":true,"conflict":false,"reasons":["新增证据支持既有核心事实"],"confidence":0.95}]}`}}
	runnable, err := New(
		context.Background(),
		fakeReader{artifacts: []eventfact.Artifact{testArtifact()}},
		fakeCanonicalReader{events: []eventfact.CanonicalEvent{{
			DedupeKey: "canonical:event", IdentityHash: identityCandidate[0].IdentityHash,
			CoreFacts: canonicalCore,
		}}},
		extractor,
		reviewer,
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runnable.Invoke(context.Background(), testInput())
	if err != nil {
		t.Fatal(err)
	}
	candidate := result.Candidates[0]
	if candidate.DedupeKey != "canonical:event" ||
		candidate.Title != "已发布规范标题" ||
		candidate.FactPayload["action"] != "既有核心事实" {
		t.Fatalf("canonical Candidate = %#v", candidate)
	}
}

func TestWorkflowLetsModelJudgeProgrammaticallyRecalledSemanticDuplicate(t *testing.T) {
	extractor := &fakeModel{responses: []string{validExtractionJSON(), tagClassificationJSON()}}
	reviewer := &fakeModel{responses: []string{
		`{"judgments":[{"candidate_id":"candidate:1","dedupe_key":"canonical:semantic","same_event":true}]}`,
		`{"reviews":[{"candidate_id":"candidate:1","semantic_pass":true,"conflict":false,"reasons":["新增证据支持既有核心事实"],"confidence":0.95}]}`,
	}}
	runnable, err := New(
		context.Background(),
		fakeReader{artifacts: []eventfact.Artifact{testArtifact()}},
		fakeCanonicalReader{events: []eventfact.CanonicalEvent{{
			DedupeKey:    "canonical:semantic",
			IdentityHash: strings.Repeat("f", 64),
			CoreFacts: []byte(
				`{"title":"某公司扩建产线","factual_summary":"某公司宣布新增产线。","occurred_at":"2026-07-26T00:00:00Z","fact_payload":{"action":"扩建产线","lifecycle_status":"announced"}}`,
			),
		}}},
		extractor,
		reviewer,
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runnable.Invoke(context.Background(), testInput())
	if err != nil {
		t.Fatal(err)
	}
	if result.Candidates[0].DedupeKey != "canonical:semantic" ||
		result.Candidates[0].Title != "某公司扩建产线" ||
		reviewer.calls != 2 || result.ReviewModelCalls != 2 {
		t.Fatalf("semantic duplicate result = %#v reviewerCalls=%d", result, reviewer.calls)
	}
}

func TestWorkflowAllowsModelAuthoredEvidenceStatementWithoutVerbatimMatch(t *testing.T) {
	extractor := &fakeModel{responses: []string{strings.Replace(
		validExtractionJSON(),
		`"evidence_statement":"某公司于2026年7月26日宣布新增一条产线。"`,
		`"evidence_statement":"正文中不存在的句子"`,
		1,
	), tagClassificationJSON()}}
	reviewer := &fakeModel{responses: []string{`{"reviews":[{"candidate_id":"candidate:1","semantic_pass":true,"conflict":false,"reasons":["证据陈述在语义上受 Artifact 支持"],"confidence":0.95}]}`}}
	runnable, err := New(context.Background(), fakeReader{artifacts: []eventfact.Artifact{testArtifact()}}, fakeCanonicalReader{}, extractor, reviewer)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runnable.Invoke(context.Background(), testInput())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Candidates) != 1 ||
		result.Candidates[0].ReviewState != eventfact.ReviewAutoApproved {
		t.Fatalf("reviewed Candidate = %#v", result.Candidates)
	}
	if extractor.calls != 2 || reviewer.calls != 1 ||
		result.ExtractionModelCalls != 2 || result.ReviewModelCalls != 1 {
		t.Fatalf("model calls extraction=%d review=%d result=%#v", extractor.calls, reviewer.calls, result)
	}
}

func TestWorkflowRejectsCamelCaseSemanticRelationInFactPayload(t *testing.T) {
	extractor := &fakeModel{responses: []string{strings.Replace(
		validExtractionJSON(),
		`"change":{}`,
		`"change":{"eventToEntity":{"entityId":"entity-1"}}`,
		1,
	)}}
	reviewer := &fakeModel{responses: []string{`{}`}}
	runnable, err := New(
		context.Background(),
		fakeReader{artifacts: []eventfact.Artifact{testArtifact()}},
		fakeCanonicalReader{},
		extractor,
		reviewer,
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runnable.Invoke(context.Background(), testInput())
	if err != nil {
		t.Fatal(err)
	}
	if result.Candidates[0].ReviewState != eventfact.ReviewRejected ||
		!strings.Contains(result.Candidates[0].Review.Reasons[0], "forbidden semantic") {
		t.Fatalf("semantic relation Candidate = %#v", result.Candidates[0])
	}
	if reviewer.calls != 0 {
		t.Fatalf("review model calls = %d, want 0", reviewer.calls)
	}
}

func validExtractionJSON() string {
	return `{"documents":[{"artifact_id":"sha256:artifact","no_event_reason":"","events":[{"title":"某公司宣布新增产线","factual_summary":"某公司于2026年7月26日宣布新增一条产线。","occurred_at":"2026-07-26T00:00:00Z","evidence_statement":"某公司于2026年7月26日宣布新增一条产线。","actor_mentions":["某公司"],"action":"新增","object_mentions":["产线"],"change":{},"lifecycle_status":"announced","time_precision":"day","location_mentions":[],"reference_period":"","quantities":["一条"]}]}]}`
}

func tagClassificationJSON() string {
	return `{"assignments":[{"candidate_id":"candidate:1","tag_codes":["technology"]}]}`
}

func testArtifact() eventfact.Artifact {
	return eventfact.Artifact{
		ArtifactID: "sha256:artifact", CollectorExecutionID: "11111111-1111-4111-8111-111111111111",
		DocumentID: "sha256:artifact", Title: "某公司宣布新增产线",
		SourceName: "公司公告", SourceType: "official", SourceURL: "https://example.com/1",
		ContentLevel: "full_text", CollectedAt: time.Date(2026, 7, 26, 1, 0, 0, 0, time.UTC),
		ContentSHA256: strings.Repeat("a", 64),
		Body:          "# 某公司宣布新增产线\n\n某公司于2026年7月26日宣布新增一条产线。",
	}
}

func testInput() *Input {
	return &Input{
		ArtifactID: "sha256:artifact",
		Attempt: eventfact.ExecutionAttempt{
			ID: "22222222-2222-4222-8222-222222222222",
			WorkItem: eventfact.WorkItem{
				CollectorExecutionIDs: []string{"11111111-1111-4111-8111-111111111111"},
			},
		},
		Catalog: eventfact.TagCatalog{
			Revision: "event-tags:r", Hash: strings.Repeat("b", 64),
			Tags: []eventfact.Tag{{
				ID: "33333333-3333-4333-8333-333333333333", Kind: "news_category",
				Code: "technology", Name: "科技", IsActive: true,
			}},
		},
	}
}
