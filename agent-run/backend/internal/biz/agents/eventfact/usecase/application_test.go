package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact"
	eventworkflow "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact/workflow"
	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

type lifecycleLoggerStub struct {
	info  []agentrun.AgentLifecycleEvent
	warn  []agentrun.AgentLifecycleEvent
	error []agentrun.AgentLifecycleEvent
}

func (l *lifecycleLoggerStub) Info(event agentrun.AgentLifecycleEvent) {
	l.info = append(l.info, event)
}
func (l *lifecycleLoggerStub) Warn(event agentrun.AgentLifecycleEvent) {
	l.warn = append(l.warn, event)
}
func (l *lifecycleLoggerStub) Error(event agentrun.AgentLifecycleEvent) {
	l.error = append(l.error, event)
}

type memoryRepository struct {
	attempt         eventfact.ExecutionAttempt
	claimed         bool
	reclaim         bool
	journals        []eventfact.JournalEntry
	status          string
	completedStatus eventfact.WorkStatus
	completedResult eventfact.Result
	completionCalls int
	retryCalls      int
}

func (*memoryRepository) DispatchPendingSignals(context.Context, string, time.Time) (int, error) {
	return 0, nil
}
func (*memoryRepository) EnqueueWork(context.Context, []string, string, time.Time) (eventfact.WorkItem, bool, error) {
	return eventfact.WorkItem{}, true, nil
}
func (*memoryRepository) NextUnplannedWork(context.Context) (eventfact.WorkItem, bool, error) {
	return eventfact.WorkItem{}, false, nil
}
func (*memoryRepository) InitializeArtifactUnits(context.Context, eventfact.WorkItem, []eventfact.ArtifactSummary, time.Time) error {
	return nil
}
func (*memoryRepository) RejectUnplannedWork(context.Context, eventfact.WorkItem, string, time.Time) error {
	return nil
}
func (r *memoryRepository) ClaimNextWork(context.Context, eventfact.ExtractionSnapshot, time.Time) (eventfact.ExecutionAttempt, bool, error) {
	if r.claimed {
		return eventfact.ExecutionAttempt{}, false, nil
	}
	r.claimed = true
	if r.attempt.Unit.Key == "" {
		r.attempt.Unit = testArtifactUnit()
	}
	return r.attempt, true, nil
}
func (r *memoryRepository) SetAwaitingTagCatalog(
	_ context.Context,
	attempt eventfact.ExecutionAttempt,
	result eventfact.Result,
	_ string,
	_ time.Time,
) error {
	encoded, err := json.Marshal(result)
	if err != nil {
		return err
	}
	r.attempt = attempt
	r.attempt.WorkItem.Status = eventfact.WorkAwaitingTagCatalog
	r.attempt.Unit.Status = eventfact.WorkAwaitingTagCatalog
	r.attempt.Unit.ExtractionResult = encoded
	if r.reclaim {
		r.claimed = false
	}
	return nil
}
func (r *memoryRepository) RetryExtraction(context.Context, eventfact.ExecutionAttempt, eventfact.Result, string, time.Time) error {
	r.retryCalls++
	return nil
}
func (r *memoryRepository) CompleteExtraction(_ context.Context, _ eventfact.ExecutionAttempt, _ eventfact.Result, journals []eventfact.JournalEntry, _ time.Time) error {
	r.journals = append([]eventfact.JournalEntry(nil), journals...)
	return nil
}
func (r *memoryRepository) CompleteWithoutPublication(
	_ context.Context,
	_ eventfact.ExecutionAttempt,
	result eventfact.Result,
	status eventfact.WorkStatus,
	_ time.Time,
) error {
	r.completedStatus = status
	r.completedResult = result
	r.completionCalls++
	return nil
}
func (r *memoryRepository) ListDeliverableJournals(context.Context, time.Time) ([]eventfact.JournalEntry, error) {
	if r.status == "acknowledged" {
		return nil, nil
	}
	return append([]eventfact.JournalEntry(nil), r.journals...), nil
}
func (r *memoryRepository) MarkJournalSending(context.Context, eventfact.JournalEntry, time.Time) (bool, error) {
	r.status = "sending"
	return true, nil
}
func (r *memoryRepository) MarkJournalRetry(context.Context, eventfact.JournalEntry, string, string, time.Time) error {
	r.status = "retry_wait"
	return nil
}
func (r *memoryRepository) MarkJournalBlocked(context.Context, eventfact.JournalEntry, string, string, time.Time) error {
	r.status = "blocked"
	return nil
}
func (r *memoryRepository) AcknowledgeJournal(context.Context, eventfact.JournalEntry, string, []eventfact.CanonicalEvent, time.Time) error {
	r.status = "acknowledged"
	return nil
}

type lossThenSuccessData struct {
	published       [][]byte
	catalogFailures int
}

func (d *lossThenSuccessData) ActiveEventTags(context.Context) (eventfact.TagCatalog, error) {
	if d.catalogFailures > 0 {
		d.catalogFailures--
		return eventfact.TagCatalog{}, errors.New("catalog unavailable")
	}
	return eventfact.TagCatalog{
		Tags: []eventfact.Tag{{
			ID: "33333333-3333-4333-8333-333333333333", Kind: "news_category",
			Code: "technology", Name: "科技", IsActive: true,
		}},
	}, nil
}

func TestTagCatalogRecoveryResumesPersistedFactsWithoutFactExtractionRerun(t *testing.T) {
	repository := &memoryRepository{
		reclaim: true,
		attempt: eventfact.ExecutionAttempt{
			ID: "22222222-2222-4222-8222-222222222222",
			WorkItem: eventfact.WorkItem{
				Key: strings.Repeat("b", 64), ExtractorAgentVersion: eventfact.AgentVersion,
				CollectorExecutionIDs: []string{"11111111-1111-4111-8111-111111111111"},
			},
		},
	}
	data := &lossThenSuccessData{catalogFailures: 1}
	logger := &lifecycleLoggerStub{}
	factCalls := 0
	resumeCalls := 0
	runtime := func(context.Context) (Runtime, error) {
		return Runtime{
			Snapshot:      eventFactTestSnapshot(),
			ReadArtifacts: testReadArtifacts,
			ExtractFacts: func(
				_ context.Context,
				attempt *eventfact.ExecutionAttempt,
			) (*eventfact.Result, error) {
				factCalls++
				result := approvedResult()
				result.ExecutionID = attempt.ID
				result.Candidates[0].TagCodes = nil
				result.Candidates[0].Tags = nil
				result.Candidates[0].Review = eventfact.Review{}
				result.Candidates[0].ReviewState = ""
				result.ReviewModelCalls = 0
				return result, nil
			},
			Run: func(_ context.Context, input *eventworkflow.Input) (*eventfact.Result, error) {
				resumeCalls++
				if input.ResumeResult == nil || len(input.ResumeResult.Candidates) != 1 {
					t.Fatal("Catalog recovery did not pass persisted Candidate to workflow")
				}
				result := approvedResult()
				result.ExecutionID = input.Attempt.ID
				return result, nil
			},
		}, nil
	}
	application, err := New(
		repository, data, runtime, time.Minute, WithEventLogger(logger),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := application.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.attempt.WorkItem.Status != eventfact.WorkAwaitingTagCatalog ||
		factCalls != 1 || resumeCalls != 0 {
		t.Fatalf(
			"awaiting Catalog status=%q factCalls=%d resumeCalls=%d",
			repository.attempt.WorkItem.Status, factCalls, resumeCalls,
		)
	}
	if len(logger.info) != 1 ||
		logger.info[0].Code != "agent_execution_started" ||
		len(logger.warn) != 1 ||
		logger.warn[0].Code != "agent_execution_retry_scheduled" ||
		logger.warn[0].Stage != "tag_catalog" {
		t.Fatalf("first lifecycle info=%#v warn=%#v", logger.info, logger.warn)
	}
	if err := application.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if factCalls != 1 || resumeCalls != 1 {
		t.Fatalf(
			"Catalog recovery factCalls=%d resumeCalls=%d",
			factCalls, resumeCalls,
		)
	}
	if len(logger.info) != 3 ||
		logger.info[1].Code != "agent_execution_started" ||
		logger.info[2].Code != "agent_execution_completed" ||
		logger.info[2].Status != "succeeded" ||
		logger.info[2].Counts["candidate_events"] != 1 ||
		len(logger.error) != 0 {
		t.Fatalf(
			"terminal lifecycle info=%#v warn=%#v error=%#v",
			logger.info, logger.warn, logger.error,
		)
	}
}
func (d *lossThenSuccessData) PublishReviewedEvents(_ context.Context, payload []byte) (string, error) {
	d.published = append(d.published, append([]byte(nil), payload...))
	if len(d.published) == 1 {
		return "", &eventfact.RemoteError{
			Code: "data_transport_unavailable", Summary: "response lost", Retryable: true,
		}
	}
	return "receipt-2", nil
}

func TestUnknownPublicationResultReplaysExactPayloadWithoutModelRerun(t *testing.T) {
	repository := &memoryRepository{attempt: eventfact.ExecutionAttempt{
		ID: "22222222-2222-4222-8222-222222222222",
		WorkItem: eventfact.WorkItem{
			Key: strings.Repeat("b", 64), ExtractorAgentVersion: eventfact.AgentVersion,
			CollectorExecutionIDs: []string{"11111111-1111-4111-8111-111111111111"},
		},
	}}
	data := &lossThenSuccessData{}
	modelCalls := 0
	run := func(context.Context, *eventworkflow.Input) (*eventfact.Result, error) {
		modelCalls++
		return approvedResult(), nil
	}
	snapshot := eventfact.ExtractionSnapshot{
		PromptSHA256: strings.Repeat("c", 64), SchemaSHA256: strings.Repeat("d", 64),
		ProviderKey: "deepseek", Model: "deepseek-chat",
	}
	runtime := func(context.Context) (Runtime, error) {
		return Runtime{Snapshot: snapshot, Run: run, ReadArtifacts: testReadArtifacts}, nil
	}
	first, err := New(repository, data, runtime, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.status != "retry_wait" || modelCalls != 1 || len(data.published) != 1 {
		t.Fatalf("first attempt status=%s modelCalls=%d publications=%d", repository.status, modelCalls, len(data.published))
	}

	restarted, err := New(repository, data, runtime, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := restarted.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.status != "acknowledged" || modelCalls != 1 || len(data.published) != 2 {
		t.Fatalf("replay status=%s modelCalls=%d publications=%d", repository.status, modelCalls, len(data.published))
	}
	if string(data.published[0]) != string(data.published[1]) {
		t.Fatal("unknown result retry changed publication bytes")
	}
}

func TestManualReviewResultIsRejectedInsteadOfWaitingForHuman(t *testing.T) {
	repository := &memoryRepository{attempt: eventfact.ExecutionAttempt{
		ID: "22222222-2222-4222-8222-222222222222",
		WorkItem: eventfact.WorkItem{
			Key: strings.Repeat("b", 64), ExtractorAgentVersion: eventfact.AgentVersion,
			CollectorExecutionIDs: []string{"11111111-1111-4111-8111-111111111111"},
		},
	}}
	data := &lossThenSuccessData{}
	runtime := func(context.Context) (Runtime, error) {
		return Runtime{
			Snapshot:      eventFactTestSnapshot(),
			ReadArtifacts: testReadArtifacts,
			Run: func(_ context.Context, input *eventworkflow.Input) (*eventfact.Result, error) {
				result := approvedResult()
				result.ExecutionID = input.Attempt.ID
				result.Candidates[0].ReviewState = eventfact.ReviewManual
				return result, nil
			},
		}, nil
	}
	application, err := New(repository, data, runtime, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := application.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.completionCalls != 1 ||
		repository.completedStatus != eventfact.WorkRejected ||
		len(data.published) != 0 {
		t.Fatalf(
			"manual result completionCalls=%d status=%q publications=%d",
			repository.completionCalls, repository.completedStatus, len(data.published),
		)
	}
}

func TestFunctionCallContractExhaustionIsTerminalAndStageClassified(t *testing.T) {
	repository := &memoryRepository{attempt: eventfact.ExecutionAttempt{
		ID: "22222222-2222-4222-8222-222222222222",
		WorkItem: eventfact.WorkItem{
			Key: strings.Repeat("b", 64), ExtractorAgentVersion: eventfact.AgentVersion,
			CollectorExecutionIDs: []string{"11111111-1111-4111-8111-111111111111"},
		},
	}}
	data := &lossThenSuccessData{}
	runtime := func(context.Context) (Runtime, error) {
		return Runtime{
			Snapshot:      eventFactTestSnapshot(),
			ReadArtifacts: testReadArtifacts,
			Run: func(context.Context, *eventworkflow.Input) (*eventfact.Result, error) {
				return nil, errors.Join(
					eventworkflow.ErrReviewModel,
					&eventworkflow.ModelContractFailure{
						Stage: "review", Violation: "missing_tool_call",
						Observations: []eventfact.FunctionCallObservation{
							{Stage: "extraction", CallCount: 1, FinishReason: "tool_calls", ArgumentBytes: 256},
							{Stage: "tag_assignment", CallCount: 1, FinishReason: "tool_calls", ArgumentBytes: 128},
							{Stage: "review", CallCount: 2, FinishReason: "stop", Violation: "missing_tool_call"},
						},
					},
				)
			},
		}, nil
	}
	application, err := New(repository, data, runtime, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := application.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.retryCalls != 0 || repository.completionCalls != 1 ||
		repository.completedStatus != eventfact.WorkRejected {
		t.Fatalf(
			"retry=%d completion=%d status=%q",
			repository.retryCalls, repository.completionCalls, repository.completedStatus,
		)
	}
	if repository.completedResult.FailureCode != "event_fact_review_contract_missing_tool_call" ||
		repository.completedResult.FailureStage != "review" ||
		repository.completedResult.FailureViolation != "missing_tool_call" ||
		repository.completedResult.ExtractionModelCalls != 2 ||
		repository.completedResult.ReviewModelCalls != 2 ||
		len(repository.completedResult.FunctionCalls) != 3 {
		t.Fatalf("terminal result = %#v", repository.completedResult)
	}
}

func approvedResult() *eventfact.Result {
	artifact := eventfact.Artifact{
		ArtifactID: "sha256:artifact", DocumentID: "sha256:artifact",
		CollectorExecutionID: "11111111-1111-4111-8111-111111111111",
		Title:                "原始文档", SourceName: "来源", SourceType: "official",
		SourceURL: "https://example.com/1", ContentLevel: "full_text",
		CollectedAt:   time.Date(2026, 7, 26, 1, 0, 0, 0, time.UTC),
		ContentSHA256: strings.Repeat("e", 64),
	}
	result := &eventfact.Result{
		ExecutionID:          "22222222-2222-4222-8222-222222222222",
		PublicationArtifacts: []eventfact.Artifact{artifact},
		Candidates: []eventfact.Candidate{{
			CandidateID: "candidate:1", ArtifactID: artifact.ArtifactID,
			Title: "某公司宣布扩产", FactualSummary: "某公司宣布扩产。",
			FactPayload: map[string]any{"action": "扩产"}, EvidenceStatement: "某公司宣布扩产",
			SupportsFields: []string{"title", "factual_summary"}, SourceLevel: "primary",
			DedupeKey: "event-fact:" + strings.Repeat("f", 64), IdentityHash: strings.Repeat("f", 64),
			Tags: []eventfact.AssignedTag{{
				ID: "33333333-3333-4333-8333-333333333333", Kind: "news_category",
				Code: "technology", Confidence: 1, AssignmentReason: "Catalog 分类",
			}},
			Review: eventfact.Review{
				SemanticPass: true, Reasons: []string{"证据支持事实"}, Confidence: 1,
			},
			ReviewState: eventfact.ReviewAutoApproved,
		}},
	}
	_, _ = json.Marshal(result)
	return result
}

func testArtifactUnit() eventfact.ArtifactUnit {
	result := approvedResult()
	return eventfact.ArtifactUnit{
		Key:                  strings.Repeat("a", 64),
		WorkItemKey:          strings.Repeat("b", 64),
		ArtifactOrdinal:      1,
		ArtifactID:           result.PublicationArtifacts[0].ArtifactID,
		CollectorExecutionID: result.PublicationArtifacts[0].CollectorExecutionID,
		ContentSHA256:        result.PublicationArtifacts[0].ContentSHA256,
		Status:               eventfact.WorkPending,
	}
}

func testReadArtifacts(context.Context, []string) ([]eventfact.Artifact, error) {
	return append([]eventfact.Artifact(nil), approvedResult().PublicationArtifacts...), nil
}
