package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
)

const testEvidenceID = "22222222-2222-4222-8222-222222222222"

type queuedModel struct {
	responses []string
	calls     int
}

func (m *queuedModel) Generate(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error) {
	m.calls++
	if len(m.responses) == 0 {
		return nil, errors.New("unexpected model call")
	}
	response := m.responses[0]
	m.responses = m.responses[1:]
	return schema.AssistantMessage(response, nil), nil
}

func (*queuedModel) Stream(context.Context, []*schema.Message, ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("stream is not used")
}

type retrieverStub struct {
	exactCalls  int
	searchCalls int
	exact       []eventsemantic.EntityCandidateSet
	search      []eventsemantic.EntityCandidateSet
}

func (s *retrieverStub) ExactEntities(_ context.Context, lookups []eventsemantic.EntityLookup) ([]eventsemantic.EntityCandidateSet, error) {
	s.exactCalls++
	if s.exact != nil {
		return s.exact, nil
	}
	result := make([]eventsemantic.EntityCandidateSet, 0, len(lookups))
	for _, lookup := range lookups {
		result = append(result, eventsemantic.EntityCandidateSet{CandidateKey: lookup.CandidateKey})
	}
	return result, nil
}

func (s *retrieverStub) SearchEntities(_ context.Context, lookups []eventsemantic.EntityLookup, topK int) ([]eventsemantic.EntityCandidateSet, error) {
	s.searchCalls++
	if topK != entityTopK {
		return nil, errors.New("unexpected topK")
	}
	if s.search != nil {
		return s.search, nil
	}
	result := make([]eventsemantic.EntityCandidateSet, 0, len(lookups))
	for _, lookup := range lookups {
		result = append(result, eventsemantic.EntityCandidateSet{CandidateKey: lookup.CandidateKey})
	}
	return result, nil
}

type dataStub struct {
	submission eventsemantic.SubmissionRequest
	review     eventsemantic.ReviewRequest
	reviewKeys []string
	submitted  bool
}

func (*dataStub) ListEligibleEvents(context.Context, int, string) (eventsemantic.EligibleEventPage, error) {
	return eventsemantic.EligibleEventPage{}, nil
}
func (*dataStub) CreateContextLease(context.Context, eventsemantic.ContextLeaseRequest) (eventsemantic.ContextLease, error) {
	return eventsemantic.ContextLease{}, nil
}
func (*dataStub) Context(context.Context, string) (eventsemantic.Context, error) {
	return eventsemantic.Context{}, nil
}
func (s *dataStub) CreateSubmission(_ context.Context, request eventsemantic.SubmissionRequest) (eventsemantic.SubmissionResult, error) {
	s.submitted = true
	s.submission = request
	return eventsemantic.SubmissionResult{
		SubmissionID: "66666666-6666-4666-8666-666666666666", EventID: request.EventID,
		Status: "pending_review", AgentExecutionID: request.AgentExecutionID,
		ReviewerWorkPackage: &eventsemantic.ReviewerWorkPackage{
			Event: testContext().Event, Evidence: testContext().Evidence,
			EntityLinks: request.EntityLinks, VariableSignals: request.VariableSignals,
		},
	}, nil
}
func (s *dataStub) SubmitReview(_ context.Context, _ string, request eventsemantic.ReviewRequest) (eventsemantic.SubmissionResult, error) {
	s.review = request
	s.reviewKeys = append(s.reviewKeys, request.ReviewerExecutionKey)
	for _, item := range request.Items {
		if item.Decision == "indeterminate" {
			return eventsemantic.SubmissionResult{
				SubmissionID: "66666666-6666-4666-8666-666666666666", EventID: s.submission.EventID,
				Status: "needs_reanalysis", AgentExecutionID: s.submission.AgentExecutionID,
				ReviewerWorkPackage: &eventsemantic.ReviewerWorkPackage{
					Event: testContext().Event, Evidence: testContext().Evidence,
					EntityLinks: s.submission.EntityLinks, VariableSignals: s.submission.VariableSignals,
				},
			}, nil
		}
	}
	links := make([]eventsemantic.CandidateDecision, 0, len(s.submission.EntityLinks))
	for _, item := range s.submission.EntityLinks {
		links = append(links, eventsemantic.CandidateDecision{CandidateKey: item.CandidateKey, Status: "accepted"})
	}
	signals := make([]eventsemantic.CandidateDecision, 0, len(s.submission.VariableSignals))
	for _, item := range s.submission.VariableSignals {
		signals = append(signals, eventsemantic.CandidateDecision{CandidateKey: item.CandidateKey, Status: "accepted"})
	}
	return eventsemantic.SubmissionResult{
		SubmissionID: "66666666-6666-4666-8666-666666666666", EventID: s.submission.EventID,
		Status: "accepted", EntityLinks: links, VariableSignals: signals,
	}, nil
}
func (*dataStub) GetEventSemantics(context.Context, string) (eventsemantic.EventSemantics, error) {
	return eventsemantic.EventSemantics{}, nil
}

func TestWorkflowUsesEventBatchedQdrantAndPublishesOnlyObjectiveV2Facts(t *testing.T) {
	generator := &queuedModel{responses: []string{
		`{"mentions":[{"candidate_key":"nvidia","mention":"英伟达","predicted_entity_type":"company","entity_role":"actor","evidence_ids":["` + testEvidenceID + `"]},{"candidate_key":"amkor","mention":"安靠科技","predicted_entity_type":"company","entity_role":"actor","evidence_ids":["` + testEvidenceID + `"]}],"variable_signals":[{"candidate_key":"nvidia-capacity","subject_link_key":"nvidia","variable_key":"capacity_commitment","variable_version":1,"direction":"increase","assertion_modality":"stated_intent","evidence_ids":["` + testEvidenceID + `"],"measurements":[{"measurement_text":"合作价值15亿美元","evidence_ids":["` + testEvidenceID + `"]},{"measurement_text":"首次把预付款锁定产能延伸至第三方封测厂","evidence_ids":["` + testEvidenceID + `"]}]}]}`,
		`{"selections":[{"candidate_key":"amkor","entity_id":"44444444-4444-4444-8444-444444444444","no_match":false}]}`,
	}}
	reviewer := &queuedModel{responses: []string{
		`{"items":[{"candidate_type":"entity_link","candidate_key":"nvidia","decision":"pass","reason_codes":[],"evidence_ids":["` + testEvidenceID + `"]},{"candidate_type":"entity_link","candidate_key":"amkor","decision":"pass","reason_codes":[],"evidence_ids":["` + testEvidenceID + `"]},{"candidate_type":"variable_signal","candidate_key":"nvidia-capacity","decision":"pass","reason_codes":[],"evidence_ids":["` + testEvidenceID + `"]}]}`,
	}}
	retriever := &retrieverStub{
		exact: []eventsemantic.EntityCandidateSet{
			{CandidateKey: "nvidia", Candidates: []eventsemantic.EntityCandidate{{Entity: eventsemantic.Entity{EntityID: "33333333-3333-4333-8333-333333333333", EntityType: "company", CanonicalName: "英伟达", Status: "active"}}}},
			{CandidateKey: "amkor"},
		},
		search: []eventsemantic.EntityCandidateSet{{CandidateKey: "amkor", Candidates: []eventsemantic.EntityCandidate{{
			Entity: eventsemantic.Entity{EntityID: "44444444-4444-4444-8444-444444444444", EntityType: "company", CanonicalName: "安靠科技", Status: "active"}, Score: 0.81,
		}}}},
	}
	data := &dataStub{}
	runnable, err := New(context.Background(), data, retriever, generator, reviewer)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runnable.Invoke(context.Background(), testInput())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "accepted" || result.AcceptedCandidates != 3 {
		t.Fatalf("result = %#v", result)
	}
	if retriever.exactCalls != 1 || retriever.searchCalls != 1 {
		t.Fatalf("Qdrant calls exact=%d search=%d", retriever.exactCalls, retriever.searchCalls)
	}
	if data.submission.AgentVersion != eventsemantic.AgentVersion || len(data.submission.EntityLinks) != 2 ||
		len(data.submission.VariableSignals) != 1 || len(data.submission.VariableSignals[0].Measurements) != 2 {
		t.Fatalf("submission = %#v", data.submission)
	}
}

func TestWorkflowPublishesOneEntityLinkWhenTwoMentionsResolveToSameEntity(t *testing.T) {
	generator := &queuedModel{responses: []string{
		`{"mentions":[{"candidate_key":"nvidia-name","mention":"英伟达","predicted_entity_type":"company","entity_role":"actor","evidence_ids":["` + testEvidenceID + `"]},{"candidate_key":"nvidia-alias","mention":"英伟达","predicted_entity_type":"company","entity_role":"actor","evidence_ids":["` + testEvidenceID + `"]}],"variable_signals":[]}`,
	}}
	reviewer := &queuedModel{responses: []string{
		`{"items":[{"candidate_type":"entity_link","candidate_key":"nvidia-name","decision":"pass","reason_codes":[],"evidence_ids":["` + testEvidenceID + `"]}]}`,
	}}
	entity := eventsemantic.Entity{
		EntityID: "33333333-3333-4333-8333-333333333333", EntityType: "company",
		CanonicalName: "英伟达", Status: "active",
	}
	retriever := &retrieverStub{exact: []eventsemantic.EntityCandidateSet{
		{CandidateKey: "nvidia-name", Candidates: []eventsemantic.EntityCandidate{{Entity: entity}}},
		{CandidateKey: "nvidia-alias", Candidates: []eventsemantic.EntityCandidate{{Entity: entity}}},
	}}
	data := &dataStub{}
	runnable, err := New(context.Background(), data, retriever, generator, reviewer)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runnable.Invoke(context.Background(), testInput())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "accepted" || len(data.submission.EntityLinks) != 1 ||
		data.submission.EntityLinks[0].CandidateKey != "nvidia-name" {
		t.Fatalf("result=%#v links=%#v", result, data.submission.EntityLinks)
	}
}

func TestWorkflowRejectsOmittedSelectionAfterOneRepair(t *testing.T) {
	generator := &queuedModel{responses: []string{
		`{"mentions":[{"candidate_key":"nvidia","mention":"英伟达","predicted_entity_type":"company","entity_role":"actor","evidence_ids":["` + testEvidenceID + `"]},{"candidate_key":"amkor","mention":"安靠科技","predicted_entity_type":"company","entity_role":"actor","evidence_ids":["` + testEvidenceID + `"]}],"variable_signals":[]}`,
		`{"selections":[{"candidate_key":"nvidia","entity_id":"33333333-3333-4333-8333-333333333333","no_match":false}]}`,
		`{"selections":[{"candidate_key":"nvidia","entity_id":"33333333-3333-4333-8333-333333333333","no_match":false}]}`,
	}}
	retriever := &retrieverStub{search: []eventsemantic.EntityCandidateSet{
		{CandidateKey: "nvidia", Candidates: []eventsemantic.EntityCandidate{{Entity: eventsemantic.Entity{
			EntityID: "33333333-3333-4333-8333-333333333333", EntityType: "company", Status: "active",
		}}}},
		{CandidateKey: "amkor", Candidates: []eventsemantic.EntityCandidate{{Entity: eventsemantic.Entity{
			EntityID: "44444444-4444-4444-8444-444444444444", EntityType: "company", Status: "active",
		}}}},
	}}
	data := &dataStub{}
	runnable, err := New(context.Background(), data, retriever, generator, &queuedModel{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := runnable.Invoke(context.Background(), testInput())
	var remote *eventsemantic.RemoteError
	if result != nil || !errors.As(err, &remote) ||
		remote.Code != "event_semantic_model_contract_invalid" || generator.calls != 3 || data.submitted {
		t.Fatalf("result=%#v err=%v calls=%d submitted=%t", result, err, generator.calls, data.submitted)
	}
}

func TestWorkflowRejectsInventedQdrantIDAfterOneRepair(t *testing.T) {
	generator := &queuedModel{responses: []string{
		`{"mentions":[{"candidate_key":"amkor","mention":"安靠科技","predicted_entity_type":"company","entity_role":"actor","evidence_ids":["` + testEvidenceID + `"]}],"variable_signals":[]}`,
		`{"selections":[{"candidate_key":"amkor","entity_id":"99999999-9999-4999-8999-999999999999","no_match":false}]}`,
		`{"selections":[{"candidate_key":"amkor","entity_id":"88888888-8888-4888-8888-888888888888","no_match":false}]}`,
	}}
	retriever := &retrieverStub{search: []eventsemantic.EntityCandidateSet{{CandidateKey: "amkor", Candidates: []eventsemantic.EntityCandidate{{
		Entity: eventsemantic.Entity{EntityID: "44444444-4444-4444-8444-444444444444", EntityType: "company", Status: "active"},
	}}}}}
	data := &dataStub{}
	runnable, err := New(context.Background(), data, retriever, generator, &queuedModel{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := runnable.Invoke(context.Background(), testInput())
	var remote *eventsemantic.RemoteError
	if result != nil || !errors.As(err, &remote) || remote.Code != "event_semantic_model_contract_invalid" || data.submitted {
		t.Fatalf("result=%#v err=%#v submitted=%v", result, err, data.submitted)
	}
}

func TestWorkflowRejectsMentionGroundedOnlyInEventSummaryAfterOneRepair(t *testing.T) {
	generator := &queuedModel{responses: []string{
		`{"mentions":[{"candidate_key":"summary-only","mention":"摘要专有词","predicted_entity_type":"company","entity_role":"actor","evidence_ids":["` + testEvidenceID + `"]}],"variable_signals":[]}`,
		`{"mentions":[{"candidate_key":"summary-only","mention":"摘要专有词","predicted_entity_type":"company","entity_role":"actor","evidence_ids":["` + testEvidenceID + `"]}],"variable_signals":[]}`,
	}}
	input := testInput()
	input.Context.Event.Summary += "摘要专有词"
	data := &dataStub{}
	runnable, err := New(context.Background(), data, &retrieverStub{}, generator, &queuedModel{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := runnable.Invoke(context.Background(), input)
	var remote *eventsemantic.RemoteError
	if result != nil || !errors.As(err, &remote) ||
		remote.Code != "event_semantic_model_contract_invalid" || generator.calls != 2 || data.submitted {
		t.Fatalf("result=%#v err=%#v calls=%d submitted=%v", result, err, generator.calls, data.submitted)
	}
}

func TestWorkflowRunsFrozenAdjudicatorAfterIndeterminateReview(t *testing.T) {
	generator := &queuedModel{responses: []string{
		`{"mentions":[{"candidate_key":"nvidia","mention":"英伟达","predicted_entity_type":"company","entity_role":"actor","evidence_ids":["` + testEvidenceID + `"]}],"variable_signals":[]}`,
	}}
	reviewer := &queuedModel{responses: []string{
		`{"items":[{"candidate_type":"entity_link","candidate_key":"nvidia","decision":"indeterminate","reason_codes":["ambiguous"],"evidence_ids":["` + testEvidenceID + `"]}]}`,
		`{"items":[{"candidate_type":"entity_link","candidate_key":"nvidia","decision":"pass","reason_codes":[],"evidence_ids":["` + testEvidenceID + `"]}]}`,
	}}
	entity := eventsemantic.Entity{
		EntityID: "33333333-3333-4333-8333-333333333333", EntityType: "company",
		CanonicalName: "英伟达", Status: "active",
	}
	data := &dataStub{}
	runnable, err := New(context.Background(), data, &retrieverStub{exact: []eventsemantic.EntityCandidateSet{{
		CandidateKey: "nvidia", Candidates: []eventsemantic.EntityCandidate{{Entity: entity}},
	}}}, generator, reviewer)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runnable.Invoke(context.Background(), testInput())
	if err != nil {
		t.Fatal(err)
	}
	wantReviewer := testInput().Attempt.ID + ":reviewer"
	wantAdjudicator := testInput().Attempt.ID + ":adjudicator"
	if result.Status != "accepted" || reviewer.calls != 2 || len(data.reviewKeys) != 2 ||
		data.reviewKeys[0] != wantReviewer || data.reviewKeys[1] != wantAdjudicator ||
		data.submission.AdjudicatorPromptHash != ReviewerPromptHash() ||
		data.submission.AdjudicatorModel != testInput().ReviewerModel {
		t.Fatalf("result=%#v calls=%d keys=%#v submission=%#v", result, reviewer.calls, data.reviewKeys, data.submission)
	}
}

func TestWorkflowResumesUnknownReviewerOutcomeAtAdjudicator(t *testing.T) {
	input := testInput()
	input.ExistingSubmission = &eventsemantic.SubmissionResult{
		SubmissionID: "66666666-6666-4666-8666-666666666666", EventID: input.Context.Event.ID,
		Status: "needs_reanalysis", AgentExecutionID: input.Attempt.ID,
		ReviewerWorkPackage: &eventsemantic.ReviewerWorkPackage{
			Event: input.Context.Event, Evidence: input.Context.Evidence,
			EntityLinks: []eventsemantic.EntityLinkCandidate{{
				CandidateKey: "nvidia", Mention: "英伟达",
				EntityID: "33333333-3333-4333-8333-333333333333", EntityRole: "actor",
				EvidenceIDs: []string{testEvidenceID}, ResolutionMethod: "qdrant_exact",
			}},
		},
		ReviewSnapshots: []eventsemantic.ReviewSnapshot{{ReviewerExecutionKey: input.Attempt.ID + ":reviewer"}},
	}
	reviewer := &queuedModel{responses: []string{
		`{"items":[{"candidate_type":"entity_link","candidate_key":"nvidia","decision":"pass","reason_codes":[],"evidence_ids":["` + testEvidenceID + `"]}]}`,
	}}
	data := &dataStub{submission: eventsemantic.SubmissionRequest{
		EventID: input.Context.Event.ID, AgentExecutionID: input.Attempt.ID,
		EntityLinks: input.ExistingSubmission.ReviewerWorkPackage.EntityLinks,
	}}
	generator := &queuedModel{}
	runnable, err := New(context.Background(), data, &retrieverStub{}, generator, reviewer)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runnable.Invoke(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "accepted" || generator.calls != 0 || reviewer.calls != 1 ||
		len(data.reviewKeys) != 1 || data.reviewKeys[0] != input.Attempt.ID+":adjudicator" {
		t.Fatalf("result=%#v generator=%d reviewer=%d keys=%#v", result, generator.calls, reviewer.calls, data.reviewKeys)
	}
}

func testInput() *Input {
	contextValue := testContext()
	return &Input{
		Attempt: eventsemantic.ExecutionAttempt{
			ID:           "77777777-7777-4777-8777-777777777777",
			ContextLease: eventsemantic.ContextLease{ContextLeaseID: contextValue.ContextLeaseID, EventID: contextValue.Event.ID},
		},
		Context: contextValue, GeneratorModel: "deepseek", ReviewerModel: "deepseek",
	}
}

func testContext() eventsemantic.Context {
	return eventsemantic.Context{
		ContextLeaseID:          "11111111-1111-4111-8111-111111111111",
		ManifestContractVersion: "event-semantic-context-manifest.v2",
		OntologyVersion:         "event-semantics.objective-v2@1", AcceptancePolicyVersion: "event-semantics.objective-v2@1",
		Event: eventsemantic.Event{
			ID:      "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
			Title:   "英伟达与安靠科技达成15亿美元战略合作",
			Summary: "首次把预付款锁定产能延伸至第三方封测厂。",
		},
		Evidence: []eventsemantic.Evidence{{
			EvidenceID: testEvidenceID,
			Excerpt:    "英伟达与安靠科技达成价值15亿美元战略合作，首次把预付款锁定产能延伸至第三方封测厂。",
		}},
		EntityTypeDefinitions: []eventsemantic.EntityTypeDefinition{{
			TypeKey: "company", Version: 1, Status: "active", SignalSubjectAllowed: true,
			AllowedEventRoles: []string{"actor", "event_subject", "affected_entity"},
		}},
		VariableDefinitions: []eventsemantic.VariableDefinition{{
			Key: "capacity_commitment", Version: 1, Status: "active",
			AllowedDirections: []string{"increase"}, ApplicableEntityTypes: []string{"company"},
		}},
		AssertionModalities: []string{"actual", "stated_intent", "source_forecast"},
		MeasurementContract: eventsemantic.MeasurementContract{
			Representation: "evidence_grounded_narrative", MaxItemsPerSignal: 8,
			MaxTextCharacters: 2000, RequiresEvidenceIDs: true, NumericValidation: false,
		},
	}
}
