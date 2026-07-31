package workflow

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
)

type queuedModel struct {
	responses []string
	calls     [][]*schema.Message
}

type scriptedModelResult struct {
	content string
	err     error
}

type scriptedModel struct {
	results []scriptedModelResult
	calls   [][]*schema.Message
}

func (m *scriptedModel) Generate(
	_ context.Context,
	input []*schema.Message,
	_ ...model.Option,
) (*schema.Message, error) {
	m.calls = append(m.calls, input)
	if len(m.results) == 0 {
		return nil, errors.New("unexpected model call")
	}
	result := m.results[0]
	m.results = m.results[1:]
	if result.err != nil {
		return nil, result.err
	}
	return schema.AssistantMessage(result.content, nil), nil
}

func (*scriptedModel) Stream(
	context.Context,
	[]*schema.Message,
	...model.Option,
) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("not implemented")
}

func (m *queuedModel) Generate(
	_ context.Context,
	input []*schema.Message,
	_ ...model.Option,
) (*schema.Message, error) {
	m.calls = append(m.calls, input)
	if len(m.responses) == 0 {
		return nil, errors.New("unexpected model call")
	}
	response := m.responses[0]
	m.responses = m.responses[1:]
	return schema.AssistantMessage(response, nil), nil
}

func (*queuedModel) Stream(
	context.Context,
	[]*schema.Message,
	...model.Option,
) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("not implemented")
}

type semanticDataStub struct {
	resolved          bool
	targetSearched    bool
	targetSubject     string
	runRequest        eventsemantic.SubmissionRequest
	reviewRequest     eventsemantic.ReviewRequest
	rejectEmpty       bool
	routeCalls        int
	anchorCalls       int
	candidateCalls    int
	submissionCalls   int
	emptyCandidates   bool
	resolutionMention string
	directTargets     []eventsemantic.DirectTarget
}

func (*semanticDataStub) ListEligibleEvents(context.Context, int, string) (eventsemantic.EligibleEventPage, error) {
	return eventsemantic.EligibleEventPage{}, nil
}
func (*semanticDataStub) CreateContextLease(
	context.Context,
	eventsemantic.ContextLeaseRequest,
) (eventsemantic.ContextLease, error) {
	return eventsemantic.ContextLease{}, nil
}
func (*semanticDataStub) Context(context.Context, string) (eventsemantic.Context, error) {
	return eventsemantic.Context{}, nil
}
func (s *semanticDataStub) Resolve(
	_ context.Context,
	_ string,
	mentions []eventsemantic.EntityMention,
) ([]eventsemantic.EntityResolution, error) {
	s.resolved = true
	resolvedMention := mentions[0].Mention
	if s.resolutionMention != "" {
		resolvedMention = s.resolutionMention
	}
	return []eventsemantic.EntityResolution{{
		Mention: resolvedMention,
		Candidates: []eventsemantic.Entity{{
			EntityID:   "33333333-3333-4333-8333-333333333333",
			EntityType: "company", Status: "active",
		}},
	}}, nil
}

func TestWorkflowRejectsEntityResolutionForAnotherMention(t *testing.T) {
	generator := &queuedModel{responses: []string{
		`{"mentions":[{"candidate_key":"company","mention":"某晶圆厂","predicted_entity_type":"company","entity_role":"statement_source","evidence_ids":["22222222-2222-4222-8222-222222222222"],"resolution_confidence":"0.9"}],"variable_signals":[]}`,
	}}
	data := &semanticDataStub{resolutionMention: "另一家公司"}
	runnable, err := New(context.Background(), data, generator, &queuedModel{})
	if err != nil {
		t.Fatal(err)
	}
	contextValue := requestContext()
	result, err := runnable.Invoke(context.Background(), &Input{
		Attempt: eventsemantic.ExecutionAttempt{
			ID: "77777777-7777-4777-8777-777777777777",
			ContextLease: eventsemantic.ContextLease{
				ContextLeaseID: contextValue.ContextLeaseID, EventID: contextValue.Event.ID,
			},
		},
		Context: contextValue, GeneratorModel: "deepseek", ReviewerModel: "deepseek",
	})

	var remote *eventsemantic.RemoteError
	if result != nil || !errors.As(err, &remote) || remote.Code != "data_response_invalid" || remote.Retryable {
		t.Fatalf("result=%#v error=%T %#v", result, err, err)
	}
	if data.runRequest.EventID != "" || data.targetSearched {
		t.Fatalf("invalid resolution was consumed: submission=%#v target_searched=%v", data.runRequest, data.targetSearched)
	}
}
func (s *semanticDataStub) SearchDirectTargets(
	_ context.Context,
	_ string,
	subjectEntityID string,
	_ []string,
) ([]eventsemantic.DirectTarget, error) {
	s.targetSearched = true
	s.targetSubject = subjectEntityID
	if s.directTargets != nil {
		return append([]eventsemantic.DirectTarget(nil), s.directTargets...), nil
	}
	return []eventsemantic.DirectTarget{{
		Entity: eventsemantic.Entity{
			EntityID:   "44444444-4444-4444-8444-444444444444",
			EntityType: "product", Status: "active",
		},
		Relation: eventsemantic.EntityRelation{
			EntityRelationID: "55555555-5555-4555-8555-555555555555",
			FromEntityID:     "33333333-3333-4333-8333-333333333333",
			ToEntityID:       "44444444-4444-4444-8444-444444444444",
			RelationType:     "produces", Status: "active",
		},
	}}, nil
}
func (s *semanticDataStub) ListResolutionRoutes(context.Context, string, string) ([]eventsemantic.ResolutionRoute, error) {
	s.routeCalls++
	return []eventsemantic.ResolutionRoute{{
		RouteID: "chain-node-via-industry.v1", RouteContractVersion: "event-semantic-anchor-routes.v1",
		TargetEntityType: "chain_node", AnchorEntityType: "industry",
		MappingRelationType: "mapped_to_industry", Partitions: []string{"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"},
		PartitionLabels: map[string]string{"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa": "Formal Industry"},
		Direction:       "industry_to_industry_chain_to_chain_node", Purpose: "Resolve through Industry",
	}}, nil
}
func (s *semanticDataStub) ListResolutionAnchors(context.Context, string, string, string, []string, int, string) (eventsemantic.ResolutionAnchorPage, error) {
	s.anchorCalls++
	return eventsemantic.ResolutionAnchorPage{Anchors: []eventsemantic.ResolutionAnchor{{
		Entity: eventsemantic.Entity{
			EntityID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", EntityType: "industry",
			CanonicalName: "Formal Industry", Status: "active",
		},
		Partition: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
	}}}, nil
}
func (s *semanticDataStub) ResolveChainNodeCandidates(context.Context, string, string, []string, int, string) (eventsemantic.ResolutionCandidatePage, error) {
	s.candidateCalls++
	if s.emptyCandidates {
		return eventsemantic.ResolutionCandidatePage{}, nil
	}
	return eventsemantic.ResolutionCandidatePage{Candidates: []eventsemantic.ResolutionCandidate{{
		Entity: eventsemantic.Entity{
			EntityID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", EntityType: "chain_node",
			CanonicalName: "Formal Chain Node", Status: "active",
		},
		ResolutionReceipt: eventsemantic.ResolutionReceipt{
			RouteID: "chain-node-via-industry.v1", RouteContractVersion: "event-semantic-anchor-routes.v1",
			AnchorEntityID:        "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
			IndustryChainEntityID: "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
			MappingRelationID:     "dddddddd-dddd-4ddd-8ddd-dddddddddddd",
			TargetEntityID:        "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
			MembershipPosition:    2, MembershipUpdatedAt: "2026-07-31T08:00:00Z",
			PathFingerprint: strings.Repeat("a", 64),
		},
	}}}, nil
}
func (s *semanticDataStub) CreateSubmission(
	_ context.Context,
	request eventsemantic.SubmissionRequest,
) (eventsemantic.SubmissionResult, error) {
	s.submissionCalls++
	s.runRequest = request
	if s.rejectEmpty &&
		len(request.EntityLinks) == 0 &&
		len(request.VariableSignals) == 0 &&
		len(request.DirectImpacts) == 0 {
		return eventsemantic.SubmissionResult{
			SubmissionID: "66666666-6666-4666-8666-666666666666",
			EventID:      request.EventID,
			Status:       "rejected",
		}, nil
	}
	return eventsemantic.SubmissionResult{
		SubmissionID: "66666666-6666-4666-8666-666666666666",
		EventID:      request.EventID, Status: "pending_review",
		ReviewerWorkPackage: &eventsemantic.ReviewerWorkPackage{
			Event: requestContext().Event, Evidence: requestContext().Evidence,
			EntityLinks: request.EntityLinks, VariableSignals: request.VariableSignals,
			DirectImpacts: request.DirectImpacts,
		},
	}, nil
}

func TestWorkflowRepairsCorrectableGeneratorConfidenceOnce(t *testing.T) {
	invalid := `{"mentions":[{"candidate_key":"company","mention":"某晶圆厂","predicted_entity_type":"company","entity_role":"statement_source","evidence_ids":["22222222-2222-4222-8222-222222222222"],"resolution_confidence":"high"}],"variable_signals":[{"candidate_key":"production","subject_link_key":"company","variable_key":"production_volume","variable_version":1,"direction":"decrease","assertion_modality":"actual","evidence_ids":["22222222-2222-4222-8222-222222222222"],"measurements":[],"extraction_confidence":"medium"}]}`
	corrected := `{"mentions":[{"candidate_key":"company","mention":"某晶圆厂","predicted_entity_type":"company","entity_role":"statement_source","evidence_ids":["22222222-2222-4222-8222-222222222222"],"resolution_confidence":"0.87"}],"variable_signals":[{"candidate_key":"production","subject_link_key":"company","variable_key":"production_volume","variable_version":1,"direction":"decrease","assertion_modality":"actual","evidence_ids":["22222222-2222-4222-8222-222222222222"],"measurements":[],"extraction_confidence":"0.74"}]}`
	generator := &queuedModel{responses: []string{invalid, corrected, `{"direct_impacts":[]}`}}
	reviewer := &queuedModel{responses: []string{
		`{"items":[{"candidate_type":"entity_link","candidate_key":"company","decision":"pass","reason_codes":[],"evidence_ids":["22222222-2222-4222-8222-222222222222"]},{"candidate_type":"variable_signal","candidate_key":"production","decision":"pass","reason_codes":[],"evidence_ids":["22222222-2222-4222-8222-222222222222"]}]}`,
	}}
	data := &semanticDataStub{}
	runnable, err := New(context.Background(), data, generator, reviewer)
	if err != nil {
		t.Fatal(err)
	}
	contextValue := requestContext()
	contextValue.DirectTransmissionRules = nil
	result, err := runnable.Invoke(context.Background(), &Input{
		Attempt: eventsemantic.ExecutionAttempt{
			ID: "77777777-7777-4777-8777-777777777777",
			ContextLease: eventsemantic.ContextLease{
				ContextLeaseID: contextValue.ContextLeaseID, EventID: contextValue.Event.ID,
			},
		},
		Context: contextValue, GeneratorModel: "deepseek", ReviewerModel: "deepseek",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "accepted" || len(generator.calls) != 3 || data.submissionCalls != 1 {
		t.Fatalf("result=%#v generator_calls=%d submission_calls=%d", result, len(generator.calls), data.submissionCalls)
	}
	if got := data.runRequest.EntityLinks[0].ResolutionConfidence; got != "0.87" {
		t.Fatalf("resolution confidence = %q", got)
	}
	if got := data.runRequest.VariableSignals[0].ExtractionConfidence; got != "0.74" {
		t.Fatalf("extraction confidence = %q", got)
	}
	repairCall := generator.calls[1]
	if len(repairCall) != 2 || repairCall[0].Role != schema.System ||
		!strings.Contains(repairCall[0].Content, "original_output") ||
		!strings.Contains(repairCall[1].Content, `"contract_version":"event-semantic-model-contract-repair-envelope.v1"`) ||
		!strings.Contains(repairCall[1].Content, `"operation":"repair_model_contract"`) ||
		!strings.Contains(repairCall[1].Content, `"stage":"event_native_candidates"`) ||
		!strings.Contains(repairCall[1].Content, `"original_output":`) ||
		!strings.Contains(repairCall[1].Content, `"output_schema":`) ||
		!strings.Contains(repairCall[1].Content, `"field_contracts":`) ||
		!strings.Contains(repairCall[1].Content, "resolution_confidence_invalid") ||
		!strings.Contains(repairCall[1].Content, "extraction_confidence_invalid") ||
		strings.Contains(repairCall[1].Content, contextValue.Event.Title) {
		t.Fatalf("repair call = %#v", repairCall)
	}
	firstPrompt := generator.calls[0][1].Content
	for _, required := range []string{
		`"valid_examples":["0","0.73","1.0"]`,
		`"invalid_examples":["high","medium",0.8,"-0.1","1.1",""]`,
		"RFC3339", "positive_integer", "exact_input_id", "decimal_string",
	} {
		if !strings.Contains(firstPrompt, required) {
			t.Fatalf("generator machine contract is missing %q", required)
		}
	}
}

func TestWorkflowFailsAfterOneModelContractRepair(t *testing.T) {
	invalid := `{"mentions":[{"candidate_key":"company","mention":"某晶圆厂","predicted_entity_type":"company","entity_role":"statement_source","evidence_ids":["22222222-2222-4222-8222-222222222222"],"resolution_confidence":"high"}],"variable_signals":[]}`
	generator := &queuedModel{responses: []string{invalid, invalid}}
	data := &semanticDataStub{}
	runnable, err := New(context.Background(), data, generator, &queuedModel{})
	if err != nil {
		t.Fatal(err)
	}
	contextValue := requestContext()
	result, err := runnable.Invoke(context.Background(), &Input{
		Attempt: eventsemantic.ExecutionAttempt{
			ID: "77777777-7777-4777-8777-777777777777",
			ContextLease: eventsemantic.ContextLease{
				ContextLeaseID: contextValue.ContextLeaseID, EventID: contextValue.Event.ID,
			},
		},
		Context: contextValue, GeneratorModel: "deepseek", ReviewerModel: "deepseek",
	})
	var remote *eventsemantic.RemoteError
	if result != nil || !errors.As(err, &remote) || remote.Code != "event_semantic_model_contract_invalid" || remote.Retryable {
		t.Fatalf("result=%#v error=%T %#v", result, err, err)
	}
	if len(generator.calls) != 2 || data.submissionCalls != 0 || data.resolved || data.targetSearched {
		t.Fatalf("calls=%d submission=%d resolved=%v targets=%v", len(generator.calls), data.submissionCalls, data.resolved, data.targetSearched)
	}
}

func TestWorkflowDoesNotRepairProviderFailure(t *testing.T) {
	generator := &scriptedModel{results: []scriptedModelResult{{err: errors.New("provider unavailable")}}}
	data := &semanticDataStub{}
	runnable, err := New(context.Background(), data, generator, &queuedModel{})
	if err != nil {
		t.Fatal(err)
	}
	contextValue := requestContext()
	result, err := runnable.Invoke(context.Background(), &Input{
		Attempt: eventsemantic.ExecutionAttempt{
			ID: "77777777-7777-4777-8777-777777777777",
			ContextLease: eventsemantic.ContextLease{
				ContextLeaseID: contextValue.ContextLeaseID, EventID: contextValue.Event.ID,
			},
		},
		Context: contextValue, GeneratorModel: "deepseek", ReviewerModel: "deepseek",
	})
	if result != nil || !errors.Is(err, eventsemantic.ErrModelUnavailable) || len(generator.calls) != 1 || data.submissionCalls != 0 {
		t.Fatalf("result=%#v error=%v calls=%d submission=%d", result, err, len(generator.calls), data.submissionCalls)
	}
}

func TestWorkflowDoesNotRepairUnsupportedEvidenceSemantics(t *testing.T) {
	unsupported := `{"mentions":[{"candidate_key":"company","mention":"原证据中不存在的实体","predicted_entity_type":"company","entity_role":"statement_source","evidence_ids":["22222222-2222-4222-8222-222222222222"],"resolution_confidence":"0.81"}],"variable_signals":[]}`
	generator := &scriptedModel{results: []scriptedModelResult{{content: unsupported}}}
	data := &semanticDataStub{}
	runnable, err := New(context.Background(), data, generator, &queuedModel{})
	if err != nil {
		t.Fatal(err)
	}
	contextValue := requestContext()
	result, err := runnable.Invoke(context.Background(), &Input{
		Attempt: eventsemantic.ExecutionAttempt{
			ID: "77777777-7777-4777-8777-777777777777",
			ContextLease: eventsemantic.ContextLease{
				ContextLeaseID: contextValue.ContextLeaseID, EventID: contextValue.Event.ID,
			},
		},
		Context: contextValue, GeneratorModel: "deepseek", ReviewerModel: "deepseek",
	})
	var remote *eventsemantic.RemoteError
	if result != nil || !errors.As(err, &remote) || remote.Code != "event_semantic_model_contract_invalid" ||
		len(generator.calls) != 1 || data.submissionCalls != 0 {
		t.Fatalf("result=%#v error=%v calls=%d submission=%d", result, err, len(generator.calls), data.submissionCalls)
	}
}

func TestWorkflowPreservesCallerCancellationWithoutRepair(t *testing.T) {
	generator := &scriptedModel{results: []scriptedModelResult{{err: context.Canceled}}}
	data := &semanticDataStub{}
	runnable, err := New(context.Background(), data, generator, &queuedModel{})
	if err != nil {
		t.Fatal(err)
	}
	contextValue := requestContext()
	result, err := runnable.Invoke(context.Background(), &Input{
		Attempt: eventsemantic.ExecutionAttempt{
			ID: "77777777-7777-4777-8777-777777777777",
			ContextLease: eventsemantic.ContextLease{
				ContextLeaseID: contextValue.ContextLeaseID, EventID: contextValue.Event.ID,
			},
		},
		Context: contextValue, GeneratorModel: "deepseek", ReviewerModel: "deepseek",
	})
	if result != nil || !errors.Is(err, context.Canceled) || len(generator.calls) != 1 || data.submissionCalls != 0 {
		t.Fatalf("result=%#v error=%v calls=%d submission=%d", result, err, len(generator.calls), data.submissionCalls)
	}
}

func TestWorkflowPreservesRepairDeadlineWithoutThirdCall(t *testing.T) {
	invalid := `{"mentions":[{"candidate_key":"company","mention":"某晶圆厂","predicted_entity_type":"company","entity_role":"statement_source","evidence_ids":["22222222-2222-4222-8222-222222222222"],"resolution_confidence":"high"}],"variable_signals":[]}`
	generator := &scriptedModel{results: []scriptedModelResult{
		{content: invalid},
		{err: context.DeadlineExceeded},
	}}
	data := &semanticDataStub{}
	runnable, err := New(context.Background(), data, generator, &queuedModel{})
	if err != nil {
		t.Fatal(err)
	}
	contextValue := requestContext()
	result, err := runnable.Invoke(context.Background(), &Input{
		Attempt: eventsemantic.ExecutionAttempt{
			ID: "77777777-7777-4777-8777-777777777777",
			ContextLease: eventsemantic.ContextLease{
				ContextLeaseID: contextValue.ContextLeaseID, EventID: contextValue.Event.ID,
			},
		},
		Context: contextValue, GeneratorModel: "deepseek", ReviewerModel: "deepseek",
	})
	if result != nil || !errors.Is(err, context.DeadlineExceeded) || len(generator.calls) != 2 || data.submissionCalls != 0 {
		t.Fatalf("result=%#v error=%v calls=%d submission=%d", result, err, len(generator.calls), data.submissionCalls)
	}
}

func TestRepairTraceContractIncludesOperationAndAllModelStages(t *testing.T) {
	generatorTrace := repairContractHashMaterial(
		modelStageEventNativeCandidates,
		modelStageChainNodeRoute,
		modelStageChainNodeAnchor,
		modelStageChainNodeCandidate,
		modelStageDirectImpacts,
	)
	for _, required := range []string{
		modelContractRepairOperation,
		modelContractRepairPolicyVersion,
		modelContractRepairEnvelopeVersion,
		"message_roles",
		"system",
		"user",
		"original_output",
		"output_schema",
		"field_contracts",
		modelStageEventNativeCandidates,
		modelStageChainNodeRoute,
		modelStageChainNodeAnchor,
		modelStageChainNodeCandidate,
		modelStageDirectImpacts,
	} {
		if !strings.Contains(generatorTrace, required) {
			t.Fatalf("generator repair trace contract is missing %q: %s", required, generatorTrace)
		}
	}
	if !strings.Contains(repairContractHashMaterial(modelStageReviewer), modelStageReviewer) ||
		!strings.Contains(repairContractHashMaterial(modelStageAdjudicator), modelStageAdjudicator) {
		t.Fatal("review repair trace contracts are missing their stage identities")
	}
}

func TestWorkflowRepairsImpactConfidenceOnce(t *testing.T) {
	generator := &queuedModel{responses: []string{
		`{"mentions":[{"candidate_key":"company","mention":"某晶圆厂","predicted_entity_type":"company","entity_role":"statement_source","evidence_ids":["22222222-2222-4222-8222-222222222222"],"resolution_confidence":"0.9"}],"variable_signals":[{"candidate_key":"production","subject_link_key":"company","variable_key":"production_volume","variable_version":1,"direction":"decrease","assertion_modality":"actual","evidence_ids":["22222222-2222-4222-8222-222222222222"],"measurements":[],"extraction_confidence":"0.91"}]}`,
		`{"direct_impacts":[{"candidate_key":"supply","source_signal_key":"production","target_entity_id":"44444444-4444-4444-8444-444444444444","affected_variable_key":"market_supply","affected_variable_version":1,"affected_direction":"decrease","derivation_type":"rule_inferred","mechanism_summary":"产量下降减少产品供给","entity_relation_id":"55555555-5555-4555-8555-555555555555","rule_key":"production_decrease_reduces_product_supply","rule_version":1,"evidence_ids":["22222222-2222-4222-8222-222222222222"],"assertion_confidence":"high"}]}`,
		`{"direct_impacts":[{"candidate_key":"supply","source_signal_key":"production","target_entity_id":"44444444-4444-4444-8444-444444444444","affected_variable_key":"market_supply","affected_variable_version":1,"affected_direction":"decrease","derivation_type":"rule_inferred","mechanism_summary":"产量下降减少产品供给","entity_relation_id":"55555555-5555-4555-8555-555555555555","rule_key":"production_decrease_reduces_product_supply","rule_version":1,"evidence_ids":["22222222-2222-4222-8222-222222222222"],"assertion_confidence":"0.86"}]}`,
	}}
	reviewer := &queuedModel{responses: []string{
		`{"items":[{"candidate_type":"entity_link","candidate_key":"company","decision":"pass","reason_codes":[],"evidence_ids":["22222222-2222-4222-8222-222222222222"]},{"candidate_type":"variable_signal","candidate_key":"production","decision":"pass","reason_codes":[],"evidence_ids":["22222222-2222-4222-8222-222222222222"]},{"candidate_type":"direct_impact","candidate_key":"supply","decision":"pass","reason_codes":[],"evidence_ids":["22222222-2222-4222-8222-222222222222"]}]}`,
	}}
	data := &semanticDataStub{}
	runnable, err := New(context.Background(), data, generator, reviewer)
	if err != nil {
		t.Fatal(err)
	}
	contextValue := requestContext()
	result, err := runnable.Invoke(context.Background(), &Input{
		Attempt: eventsemantic.ExecutionAttempt{
			ID: "77777777-7777-4777-8777-777777777777",
			ContextLease: eventsemantic.ContextLease{
				ContextLeaseID: contextValue.ContextLeaseID, EventID: contextValue.Event.ID,
			},
		},
		Context: contextValue, GeneratorModel: "deepseek", ReviewerModel: "deepseek",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "accepted" || len(generator.calls) != 3 || data.submissionCalls != 1 ||
		data.runRequest.DirectImpacts[0].AssertionConfidence != "0.86" {
		t.Fatalf("result=%#v calls=%d submission=%#v", result, len(generator.calls), data.runRequest)
	}
	if !strings.Contains(generator.calls[2][1].Content, "assertion_confidence_invalid") {
		t.Fatalf("impact repair request = %#v", generator.calls[2])
	}
}

func TestWorkflowRejectsMissingApplicableRuleImpact(t *testing.T) {
	generator := &queuedModel{responses: []string{
		`{"mentions":[{"candidate_key":"company","mention":"某晶圆厂","predicted_entity_type":"company","entity_role":"statement_source","evidence_ids":["22222222-2222-4222-8222-222222222222"],"resolution_confidence":"0.9"}],"variable_signals":[{"candidate_key":"production","subject_link_key":"company","variable_key":"production_volume","variable_version":1,"direction":"decrease","assertion_modality":"actual","evidence_ids":["22222222-2222-4222-8222-222222222222"],"measurements":[],"extraction_confidence":"0.91"}]}`,
		`{"direct_impacts":[]}`,
	}}
	data := &semanticDataStub{}
	runnable, err := New(context.Background(), data, generator, &queuedModel{})
	if err != nil {
		t.Fatal(err)
	}
	contextValue := requestContext()
	result, err := runnable.Invoke(context.Background(), &Input{
		Attempt: eventsemantic.ExecutionAttempt{
			ID: "77777777-7777-4777-8777-777777777777",
			ContextLease: eventsemantic.ContextLease{
				ContextLeaseID: contextValue.ContextLeaseID, EventID: contextValue.Event.ID,
			},
		},
		Context: contextValue, GeneratorModel: "deepseek", ReviewerModel: "deepseek",
	})
	var remote *eventsemantic.RemoteError
	if result != nil || !errors.As(err, &remote) || remote.Code != "event_semantic_model_contract_invalid" ||
		len(generator.calls) != 2 || data.submissionCalls != 0 {
		t.Fatalf("result=%#v error=%v calls=%d submission=%d", result, err, len(generator.calls), data.submissionCalls)
	}
}

func TestWorkflowRejectsImpactBindingBudgetBeforeModelCall(t *testing.T) {
	targets := make([]eventsemantic.DirectTarget, 0, 26)
	for index := 0; index < 26; index++ {
		suffix := string(rune('a' + index))
		targets = append(targets, eventsemantic.DirectTarget{
			Entity: eventsemantic.Entity{
				EntityID: "target-" + suffix, EntityType: "product", Status: "active",
			},
			Relation: eventsemantic.EntityRelation{
				EntityRelationID: "relation-" + suffix,
				FromEntityID:     "33333333-3333-4333-8333-333333333333",
				ToEntityID:       "target-" + suffix, RelationType: "produces", Status: "active",
			},
		})
	}
	generator := &queuedModel{responses: []string{
		`{"mentions":[{"candidate_key":"company","mention":"某晶圆厂","predicted_entity_type":"company","entity_role":"statement_source","evidence_ids":["22222222-2222-4222-8222-222222222222"],"resolution_confidence":"0.9"}],"variable_signals":[{"candidate_key":"production-a","subject_link_key":"company","variable_key":"production_volume","variable_version":1,"direction":"decrease","assertion_modality":"actual","evidence_ids":["22222222-2222-4222-8222-222222222222"],"measurements":[],"extraction_confidence":"0.91"},{"candidate_key":"production-b","subject_link_key":"company","variable_key":"production_volume","variable_version":1,"direction":"decrease","assertion_modality":"actual","evidence_ids":["22222222-2222-4222-8222-222222222222"],"measurements":[],"extraction_confidence":"0.92"}]}`,
	}}
	data := &semanticDataStub{directTargets: targets}
	runnable, err := New(context.Background(), data, generator, &queuedModel{})
	if err != nil {
		t.Fatal(err)
	}
	contextValue := requestContext()
	result, err := runnable.Invoke(context.Background(), &Input{
		Attempt: eventsemantic.ExecutionAttempt{
			ID: "77777777-7777-4777-8777-777777777777",
			ContextLease: eventsemantic.ContextLease{
				ContextLeaseID: contextValue.ContextLeaseID, EventID: contextValue.Event.ID,
			},
		},
		Context: contextValue, GeneratorModel: "deepseek", ReviewerModel: "deepseek",
	})
	var remote *eventsemantic.RemoteError
	if result != nil || !errors.As(err, &remote) ||
		remote.Code != "event_semantic_impact_budget_exceeded" || remote.Retryable ||
		len(generator.calls) != 1 || data.submissionCalls != 0 {
		t.Fatalf("result=%#v error=%v calls=%d submission=%d", result, err, len(generator.calls), data.submissionCalls)
	}
}

func TestWorkflowRepairsReviewerCandidateCoverageOnce(t *testing.T) {
	generator := &queuedModel{responses: []string{
		`{"mentions":[{"candidate_key":"company","mention":"某晶圆厂","predicted_entity_type":"company","entity_role":"statement_source","evidence_ids":["22222222-2222-4222-8222-222222222222"],"resolution_confidence":"0.9"}],"variable_signals":[{"candidate_key":"production","subject_link_key":"company","variable_key":"production_volume","variable_version":1,"direction":"decrease","assertion_modality":"actual","evidence_ids":["22222222-2222-4222-8222-222222222222"],"measurements":[],"extraction_confidence":"0.91"}]}`,
		`{"direct_impacts":[{"candidate_key":"supply","source_signal_key":"production","target_entity_id":"44444444-4444-4444-8444-444444444444","affected_variable_key":"market_supply","affected_variable_version":1,"affected_direction":"decrease","derivation_type":"rule_inferred","mechanism_summary":"产量下降减少产品供给","entity_relation_id":"55555555-5555-4555-8555-555555555555","rule_key":"production_decrease_reduces_product_supply","rule_version":1,"evidence_ids":["22222222-2222-4222-8222-222222222222"],"assertion_confidence":"0.86"}]}`,
	}}
	reviewer := &queuedModel{responses: []string{
		`{"items":[{"candidate_type":"entity_link","candidate_key":"company","decision":"pass","reason_codes":[],"evidence_ids":["22222222-2222-4222-8222-222222222222"]}]}`,
		`{"items":[{"candidate_type":"entity_link","candidate_key":"company","decision":"pass","reason_codes":[],"evidence_ids":["22222222-2222-4222-8222-222222222222"]},{"candidate_type":"variable_signal","candidate_key":"production","decision":"pass","reason_codes":[],"evidence_ids":["22222222-2222-4222-8222-222222222222"]},{"candidate_type":"direct_impact","candidate_key":"supply","decision":"pass","reason_codes":[],"evidence_ids":["22222222-2222-4222-8222-222222222222"]}]}`,
	}}
	data := &semanticDataStub{}
	runnable, err := New(context.Background(), data, generator, reviewer)
	if err != nil {
		t.Fatal(err)
	}
	contextValue := requestContext()
	result, err := runnable.Invoke(context.Background(), &Input{
		Attempt: eventsemantic.ExecutionAttempt{
			ID: "77777777-7777-4777-8777-777777777777",
			ContextLease: eventsemantic.ContextLease{
				ContextLeaseID: contextValue.ContextLeaseID, EventID: contextValue.Event.ID,
			},
		},
		Context: contextValue, GeneratorModel: "deepseek", ReviewerModel: "deepseek",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "accepted" || len(reviewer.calls) != 2 || data.submissionCalls != 1 {
		t.Fatalf("result=%#v reviewer_calls=%d submission_calls=%d", result, len(reviewer.calls), data.submissionCalls)
	}
	repair := reviewer.calls[1]
	if len(repair) != 2 || !strings.Contains(repair[1].Content, "review_candidate_coverage_invalid") ||
		!strings.Contains(repair[1].Content, `"expected_item_count":3`) ||
		strings.Contains(repair[1].Content, contextValue.Event.Title) {
		t.Fatalf("review repair call=%#v", repair)
	}
}

func TestWorkflowPersistsNoCandidateSubmissionAsRejected(t *testing.T) {
	generator := &queuedModel{responses: []string{
		`{"mentions":[],"variable_signals":[]}`,
		`{"direct_impacts":[]}`,
	}}
	reviewer := &queuedModel{}
	data := &semanticDataStub{rejectEmpty: true}
	runnable, err := New(context.Background(), data, generator, reviewer)
	if err != nil {
		t.Fatal(err)
	}
	contextSnapshot := requestContext()
	result, err := runnable.Invoke(context.Background(), &Input{
		Attempt: eventsemantic.ExecutionAttempt{
			ID: "77777777-7777-4777-8777-777777777777",
			ContextLease: eventsemantic.ContextLease{
				ContextLeaseID: contextSnapshot.ContextLeaseID,
				EventID:        contextSnapshot.Event.ID,
			},
		},
		Context:        contextSnapshot,
		GeneratorModel: "deepseek",
		ReviewerModel:  "deepseek",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "rejected" ||
		result.SubmissionID == "" ||
		result.AcceptedCandidates != 0 ||
		result.RejectedCandidates != 0 {
		t.Fatalf("result = %#v", result)
	}
	if len(data.runRequest.EntityLinks) != 0 ||
		len(data.runRequest.VariableSignals) != 0 ||
		len(data.runRequest.DirectImpacts) != 0 ||
		len(generator.calls) != 2 ||
		len(reviewer.calls) != 0 {
		t.Fatalf(
			"request=%#v generatorCalls=%d reviewerCalls=%d",
			data.runRequest, len(generator.calls), len(reviewer.calls),
		)
	}
}
func (s *semanticDataStub) SubmitReview(
	_ context.Context,
	runID string,
	request eventsemantic.ReviewRequest,
) (eventsemantic.SubmissionResult, error) {
	s.reviewRequest = request
	result := eventsemantic.SubmissionResult{
		SubmissionID: runID, Status: "accepted",
	}
	for _, item := range request.Items {
		decision := eventsemantic.CandidateDecision{
			CandidateKey: item.CandidateKey, Status: "accepted",
		}
		switch item.CandidateType {
		case "entity_link":
			result.EntityLinks = append(result.EntityLinks, decision)
		case "variable_signal":
			result.VariableSignals = append(result.VariableSignals, decision)
		case "direct_impact":
			result.DirectImpacts = append(result.DirectImpacts, decision)
		}
	}
	return result, nil
}
func (*semanticDataStub) GetEventSemantics(
	context.Context,
	string,
) (eventsemantic.EventSemantics, error) {
	return eventsemantic.EventSemantics{}, nil
}
func TestWorkflowResolvesDataOwnedIdentitiesAndUsesIndependentReviewer(t *testing.T) {
	generator := &queuedModel{responses: []string{
		`{"mentions":[{"candidate_key":"company","mention":"某晶圆厂","predicted_entity_type":"company","entity_role":"statement_source","evidence_ids":["22222222-2222-4222-8222-222222222222"],"resolution_confidence":"0.9"}],"variable_signals":[{"candidate_key":"production","subject_link_key":"company","variable_key":"production_volume","variable_version":1,"direction":"decrease","assertion_modality":"source_forecast","evidence_ids":["22222222-2222-4222-8222-222222222222"],"measurements":[{"measurement_role":"relative_change","value_shape":"exact","raw_value":"-10","raw_unit":"%","canonical_value":"-10","canonical_unit":"percent","raw_text":"产量预计下降10%","is_approximate":false,"evidence_id":"22222222-2222-4222-8222-222222222222"}],"statement_at":"2026-07-28T08:00:00Z","valid_from":"2026-08-01T00:00:00Z","valid_until":"2026-12-31T23:59:59Z","forecast_period_start":"2026-08-01T00:00:00Z","forecast_period_end":"2026-12-31T23:59:59Z","extraction_confidence":"0.91"}]}`,
		`{"direct_impacts":[{"candidate_key":"supply","source_signal_key":"production","target_entity_id":"44444444-4444-4444-8444-444444444444","affected_variable_key":"market_supply","affected_variable_version":1,"affected_direction":"decrease","derivation_type":"rule_inferred","mechanism_summary":"产量下降减少产品供给","entity_relation_id":"55555555-5555-4555-8555-555555555555","rule_key":"production_decrease_reduces_product_supply","rule_version":1,"evidence_ids":["22222222-2222-4222-8222-222222222222"],"assertion_confidence":"0.9"}]}`,
	}}
	reviewer := &queuedModel{responses: []string{
		`{"items":[{"candidate_type":"entity_link","candidate_key":"company","decision":"pass","reason_codes":[],"evidence_ids":["22222222-2222-4222-8222-222222222222"]},{"candidate_type":"variable_signal","candidate_key":"production","decision":"pass","reason_codes":[],"evidence_ids":["22222222-2222-4222-8222-222222222222"]},{"candidate_type":"direct_impact","candidate_key":"supply","decision":"pass","reason_codes":[],"evidence_ids":["22222222-2222-4222-8222-222222222222"]}]}`,
	}}
	data := &semanticDataStub{}
	runnable, err := New(context.Background(), data, generator, reviewer)
	if err != nil {
		t.Fatal(err)
	}
	contextSnapshot := requestContext()
	result, err := runnable.Invoke(context.Background(), &Input{
		Attempt: eventsemantic.ExecutionAttempt{
			ID: "77777777-7777-4777-8777-777777777777",
			WorkItem: eventsemantic.WorkItem{
				ID:                     "99999999-9999-4999-8999-999999999999",
				SupersedesSubmissionID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
			},
			ContextLease: eventsemantic.ContextLease{
				ContextLeaseID: contextSnapshot.ContextLeaseID, EventID: contextSnapshot.Event.ID,
				LeaseExpiresAt: time.Now().Add(time.Minute),
			},
		},
		Context: contextSnapshot, GeneratorModel: "deepseek", ReviewerModel: "deepseek",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "accepted" ||
		result.AcceptedCandidates != 3 ||
		result.RejectedCandidates != 0 ||
		!data.resolved || !data.targetSearched {
		t.Fatalf("result=%#v resolved=%v targetSearched=%v", result, data.resolved, data.targetSearched)
	}
	if data.runRequest.AgentKey != eventsemantic.AgentKey ||
		data.targetSubject != "33333333-3333-4333-8333-333333333333" ||
		data.runRequest.EntityLinks[0].EntityID != "33333333-3333-4333-8333-333333333333" ||
		data.runRequest.EntityLinks[0].ResolutionMethod != "data_service_resolution" ||
		data.runRequest.DirectImpacts[0].AffectedDirection != "decrease" ||
		data.runRequest.VariableSignals[0].StatementAt == nil ||
		data.runRequest.VariableSignals[0].ForecastPeriodEnd == nil ||
		data.runRequest.VariableSignals[0].ExtractionConfidence != "0.91" ||
		len(data.runRequest.VariableSignals[0].Measurements) != 1 ||
		data.runRequest.VariableSignals[0].Measurements[0].CanonicalUnit != "percent" ||
		data.runRequest.SupersedesSubmissionID != "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" ||
		data.reviewRequest.ReviewerExecutionKey != "77777777-7777-4777-8777-777777777777:reviewer" {
		t.Fatalf("run=%#v review=%#v", data.runRequest, data.reviewRequest)
	}
	if len(generator.calls) != 2 || len(reviewer.calls) != 1 ||
		reviewer.calls[0][0].Content != reviewerProtocol {
		t.Fatalf("generator calls=%d reviewer calls=%d", len(generator.calls), len(reviewer.calls))
	}
	for _, required := range []string{
		"全部匹配时必须输出 rule_inferred 候选",
		"不得省略已满足前提的 approved Rule",
		"匹配结果非空时 direct_impacts 不得为空",
		"不得要求 Evidence 直接陈述下游影响",
	} {
		if !strings.Contains(generator.calls[1][0].Content, required) {
			t.Fatalf("impact prompt is missing %q", required)
		}
	}
	for _, required := range []string{
		"每个结构化候选必须且只能返回一个 review item",
		"即使判定 fail 或 indeterminate 也不得省略",
		"rule_inferred 不要求 Evidence 直接陈述下游影响",
		"正式 relation 和 approved Rule",
	} {
		if !strings.Contains(reviewer.calls[0][0].Content, required) {
			t.Fatalf("reviewer prompt is missing %q", required)
		}
	}
	reviewerPayload := reviewer.calls[0][1].Content
	for _, required := range []string{
		`"expected_item_count":3`,
		`{"candidate_type":"entity_link","candidate_key":"company","evidence_ids":`,
		`{"candidate_type":"variable_signal","candidate_key":"production","evidence_ids":`,
		`{"candidate_type":"direct_impact","candidate_key":"supply","evidence_ids":`,
	} {
		if !strings.Contains(reviewerPayload, required) {
			t.Fatalf("reviewer payload is missing %q: %s", required, reviewerPayload)
		}
	}
	nativePrompt := generator.calls[0][1].Content
	for _, required := range []string{
		"statement_at", "valid_from", "valid_until",
		"forecast_period_start", "forecast_period_end", "extraction_confidence",
		"measurement_role", "canonical_unit", "evidence_id",
	} {
		if !strings.Contains(nativePrompt, required) {
			t.Fatalf("Generator output contract is missing %s: %s", required, nativePrompt)
		}
	}
}

func TestWorkflowResolvesNonExactChainNodeThroughBoundedFormalAnchors(t *testing.T) {
	generator := &queuedModel{responses: []string{
		`{"mentions":[{"candidate_key":"node","mention":"上游关键制造环节","predicted_entity_type":"chain_node","entity_role":"event_subject","evidence_ids":["22222222-2222-4222-8222-222222222222"],"resolution_confidence":"0.88"}],"variable_signals":[{"candidate_key":"signal","subject_link_key":"node","variable_key":"production_volume","variable_version":1,"direction":"decrease","assertion_modality":"actual","evidence_ids":["22222222-2222-4222-8222-222222222222"],"measurements":[],"extraction_confidence":"0.9"}]}`,
		`{"route_id":"chain-node-via-industry.v1","partition":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","unresolved":false}`,
		`{"anchor_entity_id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","unresolved":false}`,
		`{"target_entity_id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb","unresolved":false}`,
		`{"direct_impacts":[]}`,
	}}
	reviewer := &queuedModel{responses: []string{
		`{"items":[{"candidate_type":"entity_link","candidate_key":"node","decision":"pass","reason_codes":[],"evidence_ids":["22222222-2222-4222-8222-222222222222"]},{"candidate_type":"variable_signal","candidate_key":"signal","decision":"pass","reason_codes":[],"evidence_ids":["22222222-2222-4222-8222-222222222222"]}]}`,
	}}
	data := &semanticDataStub{}
	runnable, err := New(context.Background(), data, generator, reviewer)
	if err != nil {
		t.Fatal(err)
	}
	contextManifest := requestContext()
	result, err := runnable.Invoke(context.Background(), &Input{
		Attempt: eventsemantic.ExecutionAttempt{
			ID: "77777777-7777-4777-8777-777777777777",
			ContextLease: eventsemantic.ContextLease{
				ContextLeaseID: contextManifest.ContextLeaseID, EventID: contextManifest.Event.ID,
			},
		},
		Context: contextManifest, GeneratorModel: "deepseek", ReviewerModel: "deepseek",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "accepted" || data.routeCalls != 1 || data.anchorCalls != 1 || data.candidateCalls != 1 {
		t.Fatalf("result=%#v route=%d anchor=%d candidate=%d", result, data.routeCalls, data.anchorCalls, data.candidateCalls)
	}
	if len(data.runRequest.EntityLinks) != 1 || data.runRequest.EntityLinks[0].EntityID != "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb" ||
		data.runRequest.EntityLinks[0].ResolutionReceipt == nil ||
		data.runRequest.EntityLinks[0].ResolutionReceipt.AnchorEntityID != "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" {
		t.Fatalf("submitted links = %#v", data.runRequest.EntityLinks)
	}
	if len(generator.calls) != 5 {
		t.Fatalf("generator calls = %d, want bounded 5", len(generator.calls))
	}
	candidatePrompt := generator.calls[3][1].Content
	if !strings.Contains(candidatePrompt, `"variable_signals"`) ||
		!strings.Contains(candidatePrompt, `"variable_key":"production_volume"`) ||
		!strings.Contains(generator.calls[3][0].Content, "候选职责能够承载该变量") {
		t.Fatalf("candidate disambiguation is missing the mention-bound Signal: %s", candidatePrompt)
	}
	firstPrompt := generator.calls[0][1].Content
	for _, forbidden := range []string{`"entities"`, `"relations"`} {
		if strings.Contains(firstPrompt, forbidden) {
			t.Fatalf("compact extraction prompt contains %s: %s", forbidden, firstPrompt)
		}
	}
}

func TestWorkflowTreatsEmptyChainNodeCandidatesAsUnresolvedWithoutRetryFailure(t *testing.T) {
	generator := &queuedModel{responses: []string{
		`{"mentions":[{"candidate_key":"node","mention":"unsupported stage","predicted_entity_type":"chain_node","entity_role":"event_subject","evidence_ids":["22222222-2222-4222-8222-222222222222"],"resolution_confidence":"0.8"}],"variable_signals":[{"candidate_key":"signal","subject_link_key":"node","variable_key":"production_volume","variable_version":1,"direction":"decrease","assertion_modality":"actual","evidence_ids":["22222222-2222-4222-8222-222222222222"],"measurements":[],"extraction_confidence":"0.8"}]}`,
		`{"route_id":"chain-node-via-industry.v1","partition":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","unresolved":false}`,
		`{"anchor_entity_id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","unresolved":false}`,
		`{"direct_impacts":[]}`,
	}}
	data := &semanticDataStub{emptyCandidates: true, rejectEmpty: true}
	runnable, err := New(context.Background(), data, generator, &queuedModel{})
	if err != nil {
		t.Fatal(err)
	}
	contextManifest := requestContext()
	result, err := runnable.Invoke(context.Background(), &Input{
		Attempt: eventsemantic.ExecutionAttempt{
			ID: "77777777-7777-4777-8777-777777777777",
			ContextLease: eventsemantic.ContextLease{
				ContextLeaseID: contextManifest.ContextLeaseID, EventID: contextManifest.Event.ID,
			},
		},
		Context: contextManifest, GeneratorModel: "deepseek", ReviewerModel: "deepseek",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "rejected" || len(data.runRequest.EntityLinks) != 0 ||
		len(data.runRequest.VariableSignals) != 0 || data.candidateCalls != 1 {
		t.Fatalf("result=%#v submission=%#v candidates=%d", result, data.runRequest, data.candidateCalls)
	}
}

func TestStrictModelOutputRejectsDuplicateKeysAndUnboundedCandidates(t *testing.T) {
	var selection routeSelection
	if err := decodeStrict(
		`{"route_id":"one","route_id":"two","partition":"p","unresolved":false}`,
		&selection,
	); err == nil {
		t.Fatal("duplicate JSON key was accepted")
	}
	output := nativeOutput{Mentions: make([]mentionCandidate, 21)}
	if err := validateNativeOutput(output, requestContext()); err == nil {
		t.Fatal("unbounded mention output was accepted")
	}
	if err := validateImpactOutput(
		impactOutput{DirectImpacts: make([]eventsemantic.DirectImpactCandidate, 51)},
		&state{},
	); err == nil {
		t.Fatal("unbounded Direct Impact output was accepted")
	}
	if err := validateReviewOutput(reviewOutput{Items: []eventsemantic.ReviewItem{{
		CandidateType: "entity_link", CandidateKey: "link", Decision: "invented",
		EvidenceIDs: []string{"22222222-2222-4222-8222-222222222222"},
	}}}, eventsemantic.ReviewerWorkPackage{
		Evidence:    requestContext().Evidence,
		EntityLinks: []eventsemantic.EntityLinkCandidate{{CandidateKey: "link"}},
	}); err == nil {
		t.Fatal("invalid review enum was accepted")
	}
}

func TestNativeMentionMustBePresentInItsCitedEvidenceText(t *testing.T) {
	contextValue := requestContext()
	output := nativeOutput{Mentions: []mentionCandidate{{
		CandidateKey: "node", Mention: "invented upstream stage", PredictedEntityType: "chain_node",
		EntityRole: "event_subject", EvidenceIDs: []string{"22222222-2222-4222-8222-222222222222"},
		ResolutionConfidence: "0.8",
	}}}
	if err := validateNativeOutput(output, contextValue); err == nil {
		t.Fatal("mention text unsupported by the cited Evidence was accepted")
	}
}

func TestNativeOutputRejectsInvalidTimestampAndDecimalMachineFormats(t *testing.T) {
	contextValue := requestContext()
	output := nativeOutput{
		Mentions: []mentionCandidate{{
			CandidateKey: "company", Mention: "某晶圆厂", PredictedEntityType: "company",
			EntityRole: "event_subject", EvidenceIDs: []string{"22222222-2222-4222-8222-222222222222"},
			ResolutionConfidence: "0.9",
		}},
		VariableSignals: []eventsemantic.VariableSignalCandidate{{
			CandidateKey: "production", SubjectLinkKey: "company",
			VariableKey: "production_volume", VariableVersion: 1,
			Direction: "decrease", AssertionModality: "actual",
			EvidenceIDs: []string{"22222222-2222-4222-8222-222222222222"},
			StatementAt: stringPointer("2026/08/01"), ExtractionConfidence: "0.9",
			Measurements: []eventsemantic.Measurement{{
				MeasurementRole: "relative_change", ValueShape: "exact",
				RawValue: stringPointer("ten"), CanonicalValue: stringPointer("-10"),
				RawUnit: "%", CanonicalUnit: "percent", RawText: "产量下降10%",
				EvidenceID: "22222222-2222-4222-8222-222222222222",
			}},
		}},
	}
	err := validateNativeOutput(output, contextValue)
	if err == nil || !strings.Contains(err.Error(), "signal_timestamp_invalid") ||
		!strings.Contains(err.Error(), "measurement_decimal_invalid") {
		t.Fatalf("machine format error = %v", err)
	}
}

func TestModelContractErrorIsDeterministicAndNonRetryable(t *testing.T) {
	err := modelContractError("safe summary")
	var remote *eventsemantic.RemoteError
	if !errors.As(err, &remote) || remote.Retryable ||
		remote.Code != "event_semantic_model_contract_invalid" {
		t.Fatalf("model contract error = %#v", err)
	}
}

func TestWorkflowResumesPersistedSubmissionWithoutRerunningGenerator(t *testing.T) {
	generator := &queuedModel{}
	reviewer := &queuedModel{responses: []string{
		`{"items":[{"candidate_type":"entity_link","candidate_key":"company","decision":"pass","reason_codes":[],"evidence_ids":["22222222-2222-4222-8222-222222222222"]}]}`,
	}}
	data := &semanticDataStub{}
	runnable, err := New(context.Background(), data, generator, reviewer)
	if err != nil {
		t.Fatal(err)
	}
	contextSnapshot := requestContext()
	executionID := "77777777-7777-4777-8777-777777777777"
	existing := &eventsemantic.SubmissionResult{
		SubmissionID:     "66666666-6666-4666-8666-666666666666",
		EventID:          contextSnapshot.Event.ID,
		AgentExecutionID: executionID,
		Status:           "pending_review",
		ReviewerWorkPackage: &eventsemantic.ReviewerWorkPackage{
			Event: contextSnapshot.Event, Evidence: contextSnapshot.Evidence,
			EntityLinks: []eventsemantic.EntityLinkCandidate{{
				CandidateKey: "company", Mention: "某晶圆厂",
				EntityID:   "33333333-3333-4333-8333-333333333333",
				EntityRole: "actor", EvidenceIDs: []string{"22222222-2222-4222-8222-222222222222"},
			}},
		},
	}
	result, err := runnable.Invoke(context.Background(), &Input{
		Attempt: eventsemantic.ExecutionAttempt{
			ID: executionID,
			ContextLease: eventsemantic.ContextLease{
				ContextLeaseID: contextSnapshot.ContextLeaseID,
				EventID:        contextSnapshot.Event.ID,
			},
		},
		Context: contextSnapshot, ExistingSubmission: existing,
		GeneratorModel: "deepseek", ReviewerModel: "deepseek",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "accepted" || len(generator.calls) != 0 || len(reviewer.calls) != 1 {
		t.Fatalf(
			"result=%#v generatorCalls=%d reviewerCalls=%d",
			result, len(generator.calls), len(reviewer.calls),
		)
	}
	if data.runRequest.AgentExecutionID != "" ||
		data.reviewRequest.ReviewerExecutionKey != executionID+":reviewer" {
		t.Fatalf("submission=%#v review=%#v", data.runRequest, data.reviewRequest)
	}
}

func requestContext() eventsemantic.Context {
	return eventsemantic.Context{
		ContextLeaseID:          "11111111-1111-4111-8111-111111111111",
		OntologyVersion:         "event-semantics.phase-one@1",
		AcceptancePolicyVersion: "event-semantics.phase-one@1",
		Event: eventsemantic.Event{
			ID:    "88888888-8888-4888-8888-888888888888",
			Title: "某晶圆厂产量下降", EventStatus: "confirmed", FactStatus: "verified",
		},
		Evidence: []eventsemantic.Evidence{{
			EvidenceID:           "22222222-2222-4222-8222-222222222222",
			Excerpt:              "某晶圆厂预计下半年8英寸晶圆产量下降10%，影响上游关键制造环节；unsupported stage 暂无候选。",
			RawDocumentID:        "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
			SourceName:           "某晶圆厂",
			SourceType:           "company_announcement",
			SourceURL:            "https://example.com/forecast",
			Title:                "某晶圆厂产量预测",
			PublishedAt:          stringPointer("2026-07-28T08:00:00Z"),
			FirstSeenAt:          "2026-07-28T08:05:00Z",
			KnowledgeAvailableAt: "2026-07-28T08:05:00Z",
			StatementSource:      "某晶圆厂",
		}},
		VariableDefinitions: []eventsemantic.VariableDefinition{
			{Key: "production_volume", Version: 1, Status: "active",
				AllowedDirections: []string{"decrease"}, ApplicableEntityTypes: []string{"company", "chain_node"}},
			{Key: "market_supply", Version: 1, Status: "active",
				AllowedDirections: []string{"decrease"}, ApplicableEntityTypes: []string{"product"}},
		},
		DirectTransmissionRules: []eventsemantic.TransmissionRule{{
			RuleKey: "production_decrease_reduces_product_supply", Version: 1, Status: "approved",
			SourceEntityType: "company", TargetEntityType: "product",
			SourceVariableKey: "production_volume", SourceVariableVersion: 1,
			SourceDirection: "decrease", RelationType: "produces",
			AffectedVariableKey: "market_supply", AffectedVariableVersion: 1,
			AffectedDirection: "decrease",
		}},
	}
}

func stringPointer(value string) *string {
	return &value
}
