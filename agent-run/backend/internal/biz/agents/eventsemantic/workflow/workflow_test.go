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
	resolved       bool
	targetSearched bool
	targetSubject  string
	runRequest     eventsemantic.SubmissionRequest
	reviewRequest  eventsemantic.ReviewRequest
}

func (*semanticDataStub) ListEligibleEvents(context.Context, int) ([]eventsemantic.EligibleEvent, error) {
	return nil, nil
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
	return []eventsemantic.EntityResolution{{
		Mention: mentions[0].Mention,
		Candidates: []eventsemantic.Entity{{
			EntityID:   "33333333-3333-4333-8333-333333333333",
			EntityType: "company", Status: "active",
		}},
	}}, nil
}
func (s *semanticDataStub) SearchDirectTargets(
	_ context.Context,
	_ string,
	subjectEntityID string,
	_ []string,
) ([]eventsemantic.DirectTarget, error) {
	s.targetSearched = true
	s.targetSubject = subjectEntityID
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
func (s *semanticDataStub) CreateSubmission(
	_ context.Context,
	request eventsemantic.SubmissionRequest,
) (eventsemantic.SubmissionResult, error) {
	s.runRequest = request
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
func (s *semanticDataStub) SubmitReview(
	_ context.Context,
	runID string,
	request eventsemantic.ReviewRequest,
) (eventsemantic.SubmissionResult, error) {
	s.reviewRequest = request
	return eventsemantic.SubmissionResult{SubmissionID: runID, Status: "accepted"}, nil
}
func (*semanticDataStub) GetEventSemantics(
	context.Context,
	string,
) (eventsemantic.EventSemantics, error) {
	return eventsemantic.EventSemantics{}, nil
}
func TestWorkflowResolvesDataOwnedIdentitiesAndUsesIndependentReviewer(t *testing.T) {
	generator := &queuedModel{responses: []string{
		`{"entity_links":[{"candidate_key":"company","mention":"某晶圆厂","entity_id":"99999999-9999-4999-8999-999999999999","entity_role":"statement_source","evidence_ids":["22222222-2222-4222-8222-222222222222"],"resolution_method":"model_guess","resolution_confidence":"0.9"}],"variable_signals":[{"candidate_key":"production","subject_link_key":"company","variable_key":"production_volume","variable_version":1,"direction":"decrease","assertion_modality":"source_forecast","evidence_ids":["22222222-2222-4222-8222-222222222222"],"measurements":[{"measurement_role":"relative_change","value_shape":"exact","raw_value":"-10","raw_unit":"%","canonical_value":"-10","canonical_unit":"percent","raw_text":"产量预计下降10%","is_approximate":false,"evidence_id":"22222222-2222-4222-8222-222222222222"}],"statement_at":"2026-07-28T08:00:00Z","valid_from":"2026-08-01T00:00:00Z","valid_until":"2026-12-31T23:59:59Z","forecast_period_start":"2026-08-01T00:00:00Z","forecast_period_end":"2026-12-31T23:59:59Z","extraction_confidence":"0.91"}]}`,
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
	if result.Status != "accepted" || !data.resolved || !data.targetSearched {
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
			Excerpt:              "某晶圆厂预计下半年8英寸晶圆产量下降10%",
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
			{Key: "production_volume", Version: 1, Status: "active"},
			{Key: "market_supply", Version: 1, Status: "active"},
		},
		DirectTransmissionRules: []eventsemantic.TransmissionRule{{
			RuleKey: "production_decrease_reduces_product_supply", Version: 1, Status: "approved",
		}},
	}
}

func stringPointer(value string) *string {
	return &value
}
