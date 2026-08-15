package workflow

import (
	"context"
	"errors"
	"strings"
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
	exactCalls, searchCalls int
	exact, search           []eventsemantic.EntityCandidateSet
	lookups                 [][]eventsemantic.EntityLookup
	topK                    int
}

func (s *retrieverStub) ExactEntities(_ context.Context, lookups []eventsemantic.EntityLookup) ([]eventsemantic.EntityCandidateSet, error) {
	s.exactCalls++
	s.lookups = append(s.lookups, append([]eventsemantic.EntityLookup(nil), lookups...))
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
	s.lookups = append(s.lookups, append([]eventsemantic.EntityLookup(nil), lookups...))
	if topK != s.topK {
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
	reviewKeys []string
	reviews    []eventsemantic.ReviewRequest
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
	s.reviewKeys = append(s.reviewKeys, request.ReviewerExecutionKey)
	s.reviews = append(s.reviews, request)
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

func TestWorkflowUsesCrossTypeEventBatchAndGeneratesSignalsAfterResolution(t *testing.T) {
	generator := &queuedModel{responses: []string{
		`{"mentions":[{"candidate_key":"nvidia","mention":"英伟达","evidence_ids":["` + testEvidenceID + `"]},{"candidate_key":"packaging","mention":"第三方封测厂","evidence_ids":["` + testEvidenceID + `"]}]}`,
		`{"selections":[{"candidate_key":"nvidia","entity_id":"ENT33333333-3333-4333-8333-333333333333","entity_role":"actor","no_match":false},{"candidate_key":"packaging","entity_id":"ENT44444444-4444-4444-8444-444444444444","entity_role":"event_subject","no_match":false}]}`,
		`{"variable_signals":[{"candidate_key":"nvidia-capacity","subject_link_key":"nvidia","variable_key":"capacity_commitment","variable_version":1,"direction":"increase","assertion_modality":"stated_intent","evidence_ids":["` + testEvidenceID + `"],"measurements":[{"measurement_text":"价值15亿美元","evidence_ids":["` + testEvidenceID + `"]}]}]}`,
	}}
	reviewer := &queuedModel{responses: []string{
		`{"items":[{"candidate_type":"entity_link","candidate_key":"nvidia","decision":"pass","reason_codes":[],"evidence_ids":["` + testEvidenceID + `"]},{"candidate_type":"entity_link","candidate_key":"packaging","decision":"pass","reason_codes":[],"evidence_ids":["` + testEvidenceID + `"]},{"candidate_type":"variable_signal","candidate_key":"nvidia-capacity","decision":"pass","reason_codes":[],"evidence_ids":["` + testEvidenceID + `"]}]}`,
	}}
	retriever := &retrieverStub{
		exact: []eventsemantic.EntityCandidateSet{
			{CandidateKey: "nvidia", Candidates: []eventsemantic.EntityCandidate{{Entity: companyEntity()}}},
			{CandidateKey: "packaging"},
		},
		search: []eventsemantic.EntityCandidateSet{{CandidateKey: "packaging", Candidates: []eventsemantic.EntityCandidate{{
			Entity: eventsemantic.Entity{EntityID: "ENT44444444-4444-4444-8444-444444444444", EntityType: "chain_node", CanonicalName: "第三方封测", Status: "active"}, Score: 0.81,
		}}}},
	}
	data := &dataStub{}
	result := invoke(t, data, retriever, generator, reviewer, testInput())
	if result.Status != "accepted" || result.AcceptedCandidates != 3 {
		t.Fatalf("result = %#v", result)
	}
	if retriever.exactCalls != 1 || retriever.searchCalls != 1 || len(retriever.lookups[0]) != 2 || len(retriever.lookups[1]) != 1 {
		t.Fatalf("retrieval calls=%d/%d lookups=%#v", retriever.exactCalls, retriever.searchCalls, retriever.lookups)
	}
	if len(data.submission.EntityLinks) != 2 || len(data.submission.VariableSignals) != 1 ||
		data.submission.EntityLinks[0].ProjectedEntityType != "company" ||
		data.submission.VariableSignals[0].Measurements[0].MeasurementText != "价值15亿美元" {
		t.Fatalf("submission = %#v", data.submission)
	}
	if strings.Contains(mentionSchema, "predicted_entity_type") || strings.Contains(mentionSchema, "entity_role") || strings.Contains(mentionSchema, "variable_signals") {
		t.Fatalf("Stage A schema leaked V2 fields: %s", mentionSchema)
	}
}

func TestWorkflowIsolatesInvalidSelectionWithoutDeletingValidLink(t *testing.T) {
	generator := &queuedModel{responses: []string{
		`{"mentions":[{"candidate_key":"nvidia","mention":"英伟达","evidence_ids":["` + testEvidenceID + `"]},{"candidate_key":"amkor","mention":"安靠科技","evidence_ids":["` + testEvidenceID + `"]}]}`,
		`{"selections":[{"candidate_key":"nvidia","entity_id":"ENT33333333-3333-4333-8333-333333333333","entity_role":"actor","no_match":false},{"candidate_key":"amkor","entity_id":"ENT99999999-9999-4999-8999-999999999999","entity_role":"actor","no_match":false}]}`,
		`{"variable_signals":[]}`,
	}}
	reviewer := &queuedModel{responses: []string{
		`{"selections":[{"candidate_key":"amkor","entity_id":"","entity_role":"","no_match":true,"no_match_reason":"no_candidate_same_entity"}]}`,
		`{"items":[{"candidate_type":"entity_link","candidate_key":"nvidia","decision":"pass","reason_codes":[],"evidence_ids":["` + testEvidenceID + `"]}]}`,
	}}
	retriever := &retrieverStub{exact: []eventsemantic.EntityCandidateSet{
		{CandidateKey: "nvidia", Candidates: []eventsemantic.EntityCandidate{{Entity: companyEntity()}}},
		{CandidateKey: "amkor", Candidates: []eventsemantic.EntityCandidate{{Entity: eventsemantic.Entity{EntityID: "ENT55555555-5555-4555-8555-555555555555", EntityType: "company", CanonicalName: "安靠科技", Status: "active"}}}},
	}}
	data := &dataStub{}
	input := testInput()
	result := invoke(t, data, retriever, generator, reviewer, input)
	if result.Status != "accepted" || len(data.submission.EntityLinks) != 1 || data.submission.EntityLinks[0].CandidateKey != "nvidia" {
		t.Fatalf("result=%#v links=%#v", result, data.submission.EntityLinks)
	}
	if !hasIsolation(result.Audit, "amkor", "selection_outside_qdrant_response") {
		t.Fatalf("audit = %#v", result.Audit)
	}
}

func TestRecordSelectionClassifiesNoMatchByExactCandidateAvailability(t *testing.T) {
	tests := []struct {
		name          string
		noMatchReason string
		hasExact      bool
		wantReason    string
		wantOwner     string
	}{
		{name: "Stage A non-entity", noMatchReason: "mention_not_entity", wantReason: "stage_a_non_entity_mention", wantOwner: "model_extraction"},
		{name: "identity projection gap", noMatchReason: "no_candidate_same_entity", wantReason: "identity_projection_gap", wantOwner: "abox_or_retrieval"},
		{name: "model rejects exact identity", noMatchReason: "no_candidate_same_entity", hasExact: true, wantReason: "selector_rejected_exact_candidates", wantOwner: "model_selection"},
		{name: "model lacks context", noMatchReason: "insufficient_context", wantReason: "selector_insufficient_context", wantOwner: "model_selection"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			audit := &eventsemantic.StageAudit{}
			recordSelection(audit, entitySelection{CandidateKey: "entity", NoMatch: true, NoMatchReason: test.noMatchReason}, eventsemantic.Entity{}, test.hasExact, "primary_selector")
			if len(audit.Selections) != 1 || audit.Selections[0].ReasonCode != test.wantReason || audit.Selections[0].Owner != test.wantOwner {
				t.Fatalf("selection audit = %#v", audit.Selections)
			}
		})
	}
}

func TestWorkflowAcceptsEventOnlyMentionWithPrimarySupportingLineageAndNoSignal(t *testing.T) {
	input := testInput()
	input.Context.Event.Summary += " 摘要专有词"
	generator := &queuedModel{responses: []string{
		`{"mentions":[{"candidate_key":"summary","mention":"摘要专有词","evidence_ids":["` + testEvidenceID + `"]}]}`,
		`{"selections":[{"candidate_key":"summary","entity_id":"ENT33333333-3333-4333-8333-333333333333","entity_role":"event_subject","no_match":false}]}`,
		`{"variable_signals":[]}`,
	}}
	reviewer := &queuedModel{responses: []string{`{"items":[{"candidate_type":"entity_link","candidate_key":"summary","decision":"pass","reason_codes":[],"evidence_ids":["` + testEvidenceID + `"]}]}`}}
	data := &dataStub{}
	result := invoke(t, data, &retrieverStub{exact: []eventsemantic.EntityCandidateSet{{CandidateKey: "summary", Candidates: []eventsemantic.EntityCandidate{{Entity: companyEntity()}}}}}, generator, reviewer, input)
	if result.Status != "accepted" || len(data.submission.EntityLinks) != 1 || len(data.submission.VariableSignals) != 0 {
		t.Fatalf("result=%#v submission=%#v", result, data.submission)
	}
}

func TestWorkflowVectorRecallsAndMergesNonUniqueExactCandidates(t *testing.T) {
	input := testInput()
	generator := &queuedModel{responses: []string{
		`{"mentions":[{"candidate_key":"nvidia","mention":"英伟达","evidence_ids":["` + testEvidenceID + `"]}]}`,
		`{"selections":[{"candidate_key":"nvidia","entity_id":"ENT33333333-3333-4333-8333-333333333333","entity_role":"actor","no_match":false}]}`,
		`{"variable_signals":[]}`,
	}}
	reviewer := &queuedModel{responses: []string{`{"items":[{"candidate_type":"entity_link","candidate_key":"nvidia","decision":"pass","reason_codes":[],"evidence_ids":["` + testEvidenceID + `"]}]}`}}
	retriever := &retrieverStub{
		exact: []eventsemantic.EntityCandidateSet{{CandidateKey: "nvidia", Candidates: []eventsemantic.EntityCandidate{
			{Entity: companyEntity()},
			{Entity: eventsemantic.Entity{EntityID: "ENT55555555-5555-4555-8555-555555555555", EntityType: "concept", CanonicalName: "英伟达生态", Status: "active"}},
		}}},
		search: []eventsemantic.EntityCandidateSet{{CandidateKey: "nvidia", Candidates: []eventsemantic.EntityCandidate{
			{Entity: companyEntity(), Score: 0.93},
			{Entity: eventsemantic.Entity{EntityID: "ENT44444444-4444-4444-8444-444444444444", EntityType: "chain_node", CanonicalName: "GPU", Status: "active"}, Score: 0.72},
		}}},
	}
	data := &dataStub{}
	result := invoke(t, data, retriever, generator, reviewer, input)
	if result.Status != "accepted" || retriever.searchCalls != 1 || len(retriever.lookups[1]) != 1 {
		t.Fatalf("result=%#v calls=%d lookups=%#v", result, retriever.searchCalls, retriever.lookups)
	}
	if len(data.submission.EntityLinks) != 1 || data.submission.EntityLinks[0].ResolutionMethod != "qdrant_exact" {
		t.Fatalf("submission=%#v", data.submission)
	}
	if len(result.Audit.CandidateSets) != 2 || result.Audit.CandidateSets[0].Method != "qdrant_exact" ||
		result.Audit.CandidateSets[1].Method != "qdrant_vector" {
		t.Fatalf("candidate audit=%#v", result.Audit.CandidateSets)
	}
}

func TestWorkflowRechecksUniqueExactCandidateRejectedByPrimarySelector(t *testing.T) {
	generator := &queuedModel{responses: []string{
		`{"mentions":[{"candidate_key":"nvidia","mention":"英伟达","evidence_ids":["` + testEvidenceID + `"]}]}`,
		`{"selections":[{"candidate_key":"nvidia","entity_id":"","entity_role":"","no_match":true,"no_match_reason":"mention_not_entity"}]}`,
		`{"variable_signals":[]}`,
	}}
	reviewer := &queuedModel{responses: []string{
		`{"selections":[{"candidate_key":"nvidia","entity_id":"ENT33333333-3333-4333-8333-333333333333","entity_role":"actor","no_match":false,"no_match_reason":""}]}`,
		`{"items":[{"candidate_type":"entity_link","candidate_key":"nvidia","decision":"pass","reason_codes":[],"evidence_ids":["` + testEvidenceID + `"]}]}`,
	}}
	data := &dataStub{}
	result := invoke(t, data, &retrieverStub{exact: []eventsemantic.EntityCandidateSet{{
		CandidateKey: "nvidia", Candidates: []eventsemantic.EntityCandidate{{Entity: companyEntity()}},
	}}}, generator, reviewer, testInput())
	if result.Status != "accepted" || len(data.submission.EntityLinks) != 1 || reviewer.calls != 2 {
		t.Fatalf("result=%#v links=%#v reviewer_calls=%d", result, data.submission.EntityLinks, reviewer.calls)
	}
	if len(result.Audit.Selections) != 2 || result.Audit.Selections[0].ResolutionRoute != "primary_selector" ||
		!result.Audit.Selections[0].NoMatch || result.Audit.Selections[1].ResolutionRoute != "secondary_review" ||
		result.Audit.Selections[1].NoMatch {
		t.Fatalf("selection audit=%#v", result.Audit.Selections)
	}
}

func TestWorkflowDoesNotRecheckVectorCandidateWithoutFormalAliasIdentity(t *testing.T) {
	input := testInput()
	input.Context.Event.Title = "非侵入式脑机接口取得进展"
	input.Context.Event.Summary = "非侵入式脑机接口已完成新一轮验证。"
	input.Context.Evidence[0].Statement = "非侵入式脑机接口已完成新一轮验证。"
	generator := &queuedModel{responses: []string{
		`{"mentions":[{"candidate_key":"bci","mention":"非侵入式脑机接口","evidence_ids":["` + testEvidenceID + `"]}]}`,
		`{"selections":[{"candidate_key":"bci","entity_id":"","entity_role":"","no_match":true,"no_match_reason":"no_candidate_same_entity"}]}`,
	}}
	reviewer := &queuedModel{responses: []string{`{"items":[]}`}}
	retriever := &retrieverStub{
		exact: []eventsemantic.EntityCandidateSet{{CandidateKey: "bci"}},
		search: []eventsemantic.EntityCandidateSet{{CandidateKey: "bci", Candidates: []eventsemantic.EntityCandidate{{
			Entity: eventsemantic.Entity{EntityID: "ENT88888888-8888-4888-8888-888888888888", EntityType: "technology", Name: "非侵入式脑机接口系统", CanonicalName: "非侵入式脑机接口系统", Status: "active"}, Score: 0.91,
		}}}},
	}
	data := &dataStub{}
	result := invoke(t, data, retriever, generator, reviewer, input)
	if result.Status != "accepted" || len(data.submission.EntityLinks) != 0 || reviewer.calls != 1 {
		t.Fatalf("result=%#v links=%#v", result, data.submission.EntityLinks)
	}
}

func TestSelectorProtocolUsesRecallFirstReviewFallback(t *testing.T) {
	for _, forbidden := range []string{"省略“系统、服务、设备、产品”", "允许的名称规范化", "广为确认的正式简称"} {
		if strings.Contains(selectorProtocol, forbidden) {
			t.Fatalf("selector protocol retained handwritten identity rule %q", forbidden)
		}
	}
	for _, required := range []string{"identity_locked_candidate_keys", "canonical_name、name 或正式 aliases", "不得根据手写简称", "statement_source", "event_object", "affected_entity", "国新办"} {
		if !strings.Contains(selectorProtocol, required) {
			t.Fatalf("selector protocol is missing %q", required)
		}
	}
	if !strings.Contains(reviewerProtocol, "国新办") || !strings.Contains(reviewerProtocol, "国务院") {
		t.Fatal("reviewer protocol must reject related-but-distinct institutions")
	}
}

func TestRoleProtocolsCoverActionTargetGoldenCases(t *testing.T) {
	for _, protocol := range []string{selectorProtocol, reviewerProtocol} {
		for _, required := range []string{
			"特朗普发布对伊朗48小时通牒", "美国暂停对伊朗军事打击",
			"巴西对原产于中国的钢瓶发起调查", "event_object", "affected_entity", "event_subject",
		} {
			if !strings.Contains(protocol, required) {
				t.Fatalf("role protocol is missing golden guidance %q", required)
			}
		}
	}
}

func TestWorkflowIsolatesInvalidMentionItemWithoutRepairingWholeEnvelope(t *testing.T) {
	generator := &queuedModel{responses: []string{
		`{"mentions":[{"candidate_key":"nvidia","mention":"英伟达","evidence_ids":["` + testEvidenceID + `"]},{"candidate_key":"broken","mention":42,"evidence_ids":["` + testEvidenceID + `"]}]}`,
		`{"selections":[{"candidate_key":"nvidia","entity_id":"ENT33333333-3333-4333-8333-333333333333","entity_role":"actor","no_match":false}]}`,
		`{"variable_signals":[]}`,
	}}
	reviewer := &queuedModel{responses: []string{`{"items":[{"candidate_type":"entity_link","candidate_key":"nvidia","decision":"pass","reason_codes":[],"evidence_ids":["` + testEvidenceID + `"]}]}`}}
	data := &dataStub{}
	result := invoke(t, data, &retrieverStub{exact: []eventsemantic.EntityCandidateSet{{
		CandidateKey: "nvidia", Candidates: []eventsemantic.EntityCandidate{{Entity: companyEntity()}},
	}}}, generator, reviewer, testInput())
	if result.Status != "accepted" || len(data.submission.EntityLinks) != 1 || generator.calls != 3 {
		t.Fatalf("result=%#v links=%#v generator_calls=%d", result, data.submission.EntityLinks, generator.calls)
	}
	if !hasIsolation(result.Audit, "broken", "mention_item_invalid") {
		t.Fatalf("audit=%#v", result.Audit)
	}
}

func TestMentionProtocolExtractsEntitySpanFromCompoundStatement(t *testing.T) {
	for _, required := range []string{"日本央行", "美联储", "央行报告", "只输出其中的机构 Mention"} {
		if !strings.Contains(mentionProtocol, required) {
			t.Fatalf("mention protocol is missing %q", required)
		}
	}
	for _, forbidden := range []string{"predicted_entity_type", "entity_role", "variable_signals"} {
		if strings.Contains(mentionProtocol, forbidden) {
			t.Fatalf("mention protocol leaked Stage A forbidden field %q", forbidden)
		}
	}
}

func TestMissingReviewItemUsesThatCandidatesEvidence(t *testing.T) {
	otherEvidenceID := "99999999-9999-4999-8999-999999999999"
	work := eventsemantic.ReviewerWorkPackage{
		Evidence: []eventsemantic.Evidence{{EvidenceID: testEvidenceID}, {EvidenceID: otherEvidenceID}},
		EntityLinks: []eventsemantic.EntityLinkCandidate{
			{CandidateKey: "first", EvidenceIDs: []string{testEvidenceID}},
			{CandidateKey: "second", EvidenceIDs: []string{otherEvidenceID}},
		},
	}
	expected := expectedReviewCandidates(work)
	items := isolateReview([]eventsemantic.ReviewItem{{
		CandidateType: "entity_link", CandidateKey: "first", Decision: "pass", EvidenceIDs: []string{testEvidenceID},
	}}, expected, work, &eventsemantic.StageAudit{})
	if len(items) != 2 || len(items[1].EvidenceIDs) != 1 || items[1].EvidenceIDs[0] != otherEvidenceID ||
		items[1].Decision != "fail" {
		t.Fatalf("review items=%#v", items)
	}
}

func TestWorkflowRunsAdjudicatorAfterIndeterminateReview(t *testing.T) {
	generator := &queuedModel{responses: []string{
		`{"mentions":[{"candidate_key":"nvidia","mention":"英伟达","evidence_ids":["` + testEvidenceID + `"]}]}`,
		`{"selections":[{"candidate_key":"nvidia","entity_id":"ENT33333333-3333-4333-8333-333333333333","entity_role":"actor","no_match":false}]}`,
		`{"variable_signals":[]}`,
	}}
	reviewer := &queuedModel{responses: []string{
		`{"items":[{"candidate_type":"entity_link","candidate_key":"nvidia","decision":"indeterminate","reason_codes":["ambiguous"],"evidence_ids":["` + testEvidenceID + `"]}]}`,
		`{"items":[{"candidate_type":"entity_link","candidate_key":"nvidia","decision":"pass","reason_codes":[],"evidence_ids":["` + testEvidenceID + `"]}]}`,
	}}
	data := &dataStub{}
	result := invoke(t, data, &retrieverStub{exact: []eventsemantic.EntityCandidateSet{{
		CandidateKey: "nvidia", Candidates: []eventsemantic.EntityCandidate{{Entity: companyEntity()}},
	}}}, generator, reviewer, testInput())
	if result.Status != "accepted" || reviewer.calls != 2 || len(data.reviewKeys) != 2 ||
		data.reviewKeys[0] != testInput().Attempt.ID+":reviewer" ||
		data.reviewKeys[1] != testInput().Attempt.ID+":adjudicator" {
		t.Fatalf("result=%#v calls=%d keys=%#v", result, reviewer.calls, data.reviewKeys)
	}
}

func TestWorkflowResumesUnknownReviewerOutcomeAtAdjudicator(t *testing.T) {
	input := testInput()
	link := eventsemantic.EntityLinkCandidate{
		CandidateKey: "nvidia", Mention: "英伟达", EntityID: companyEntity().EntityID,
		ProjectedEntityType: "company", EntityRole: "actor", EvidenceIDs: []string{testEvidenceID}, ResolutionMethod: "qdrant_exact",
	}
	input.ExistingSubmission = &eventsemantic.SubmissionResult{
		SubmissionID: "66666666-6666-4666-8666-666666666666", EventID: input.Context.Event.ID,
		Status: "needs_reanalysis", AgentExecutionID: input.Attempt.ID,
		ReviewerWorkPackage: &eventsemantic.ReviewerWorkPackage{
			Event: input.Context.Event, Evidence: input.Context.Evidence, EntityLinks: []eventsemantic.EntityLinkCandidate{link},
		},
		ReviewSnapshots: []eventsemantic.ReviewSnapshot{{ReviewerExecutionKey: input.Attempt.ID + ":reviewer"}},
	}
	reviewer := &queuedModel{responses: []string{`{"items":[{"candidate_type":"entity_link","candidate_key":"nvidia","decision":"pass","reason_codes":[],"evidence_ids":["` + testEvidenceID + `"]}]}`}}
	data := &dataStub{submission: eventsemantic.SubmissionRequest{
		EventID: input.Context.Event.ID, AgentExecutionID: input.Attempt.ID, EntityLinks: []eventsemantic.EntityLinkCandidate{link},
	}}
	generator := &queuedModel{}
	runnable, err := New(context.Background(), data, &retrieverStub{topK: 10}, generator, reviewer, 10)
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

func TestWorkflowOnlyTerminatesAfterJSONEnvelopeRepairStillFails(t *testing.T) {
	generator := &queuedModel{responses: []string{`not-json`, `{"mentions":{}}`}}
	runnable, err := New(context.Background(), &dataStub{}, &retrieverStub{topK: 10}, generator, &queuedModel{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runnable.Invoke(context.Background(), testInput())
	var remote *eventsemantic.RemoteError
	if result != nil || !errors.As(err, &remote) || remote.Code != "event_semantic_model_contract_invalid" || generator.calls != 2 {
		t.Fatalf("result=%#v err=%#v calls=%d", result, err, generator.calls)
	}
}

func TestStageEnvelopeRequiresExplicitTopLevelArray(t *testing.T) {
	tests := []struct {
		name, stage, schema, invalid, valid string
	}{
		{name: "mention null", stage: "mention_extraction", schema: mentionSchema, invalid: `null`, valid: `{"mentions":[]}`},
		{name: "mention empty object", stage: "mention_extraction", schema: mentionSchema, invalid: `{}`, valid: `{"mentions":[]}`},
		{name: "mention missing field", stage: "mention_extraction", schema: mentionSchema, invalid: `{"other":[]}`, valid: `{"mentions":[]}`},
		{name: "mention wrong type", stage: "mention_extraction", schema: mentionSchema, invalid: `{"mentions":{}}`, valid: `{"mentions":[]}`},
		{name: "mention null array", stage: "mention_extraction", schema: mentionSchema, invalid: `{"mentions":null}`, valid: `{"mentions":[]}`},
		{name: "mention unknown field", stage: "mention_extraction", schema: mentionSchema, invalid: `{"mentions":[],"extra":true}`, valid: `{"mentions":[]}`},
		{name: "mention duplicate field", stage: "mention_extraction", schema: mentionSchema, invalid: `{"mentions":[],"mentions":[]}`, valid: `{"mentions":[]}`},
		{name: "mention trailing value", stage: "mention_extraction", schema: mentionSchema, invalid: `{"mentions":[]} {}`, valid: `{"mentions":[]}`},
		{name: "selection missing field", stage: "entity_selection", schema: selectorSchema, invalid: `{}`, valid: `{"selections":[]}`},
		{name: "signal missing field", stage: "signal_extraction", schema: signalSchema, invalid: `{}`, valid: `{"variable_signals":[]}`},
		{name: "review missing field", stage: "independent_review", schema: reviewSchema, invalid: `{}`, valid: `{"items":[]}`},
	}
	for _, test := range tests {
		t.Run(test.name+" fails after repair", func(t *testing.T) {
			model := &queuedModel{responses: []string{test.invalid, test.invalid}}
			var err error
			switch test.stage {
			case "mention_extraction":
				_, err = generateEnvelope[mentionOutput](context.Background(), model, test.stage, "system", `{}`, test.schema, &eventsemantic.StageAudit{})
			case "entity_selection":
				_, err = generateEnvelope[selectionOutput](context.Background(), model, test.stage, "system", `{}`, test.schema, &eventsemantic.StageAudit{})
			case "signal_extraction":
				_, err = generateEnvelope[signalOutput](context.Background(), model, test.stage, "system", `{}`, test.schema, &eventsemantic.StageAudit{})
			case "independent_review":
				_, err = generateEnvelope[reviewOutput](context.Background(), model, test.stage, "system", `{}`, test.schema, &eventsemantic.StageAudit{})
			}
			var remote *eventsemantic.RemoteError
			if !errors.As(err, &remote) || remote.Code != "event_semantic_model_contract_invalid" || model.calls != 2 {
				t.Fatalf("err=%#v calls=%d", err, model.calls)
			}
		})
		t.Run(test.name+" accepts explicit empty array", func(t *testing.T) {
			model := &queuedModel{responses: []string{test.valid}}
			var err error
			switch test.stage {
			case "mention_extraction":
				_, err = generateEnvelope[mentionOutput](context.Background(), model, test.stage, "system", `{}`, test.schema, &eventsemantic.StageAudit{})
			case "entity_selection":
				_, err = generateEnvelope[selectionOutput](context.Background(), model, test.stage, "system", `{}`, test.schema, &eventsemantic.StageAudit{})
			case "signal_extraction":
				_, err = generateEnvelope[signalOutput](context.Background(), model, test.stage, "system", `{}`, test.schema, &eventsemantic.StageAudit{})
			case "independent_review":
				_, err = generateEnvelope[reviewOutput](context.Background(), model, test.stage, "system", `{}`, test.schema, &eventsemantic.StageAudit{})
			}
			if err != nil || model.calls != 1 {
				t.Fatalf("err=%v calls=%d", err, model.calls)
			}
		})
	}
}

func invoke(t *testing.T, data *dataStub, retriever *retrieverStub, generator, reviewer *queuedModel, input *Input) *eventsemantic.Result {
	t.Helper()
	retriever.topK = 10
	runnable, err := New(context.Background(), data, retriever, generator, reviewer, 10)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runnable.Invoke(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func hasIsolation(audit eventsemantic.StageAudit, key, reason string) bool {
	for _, item := range audit.Isolations {
		if item.CandidateKey == key && item.ReasonCode == reason {
			return true
		}
	}
	return false
}

func companyEntity() eventsemantic.Entity {
	return eventsemantic.Entity{EntityID: "ENT33333333-3333-4333-8333-333333333333", EntityType: "company", CanonicalName: "英伟达", Status: "active"}
}

func testInput() *Input {
	contextValue := testContext()
	audit := &eventsemantic.StageAudit{}
	return &Input{
		Attempt: eventsemantic.ExecutionAttempt{
			ID:           "77777777-7777-4777-8777-777777777777",
			ContextLease: eventsemantic.ContextLease{ContextLeaseID: contextValue.ContextLeaseID, EventID: contextValue.Event.ID},
		},
		Context: contextValue, GeneratorModel: "deepseek", ReviewerModel: "deepseek", Audit: audit,
	}
}

func testContext() eventsemantic.Context {
	return eventsemantic.Context{
		ContextLeaseID: "11111111-1111-4111-8111-111111111111", ManifestContractVersion: "event-semantic-context-manifest.v4",
		OntologyVersion: "event-semantics.objective-v3@1", AcceptancePolicyVersion: "event-semantics.objective-v2@1",
		Event: eventsemantic.Event{ID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", Title: "英伟达与安靠科技达成15亿美元战略合作", Summary: "首次把预付款锁定产能延伸至第三方封测厂。"},
		Evidence: []eventsemantic.Evidence{{
			EvidenceID: testEvidenceID, Statement: "英伟达与安靠科技达成价值15亿美元战略合作，首次把预付款锁定产能延伸至第三方封测厂。",
			Relation: "supports",
		}},
		VariableDefinitions: []eventsemantic.VariableDefinition{{
			Key: "capacity_commitment", Version: 1, NameZH: "产能承诺", NameEN: "Capacity commitment", Domain: "operations", BusinessDefinition: "正式产能承诺", ValueType: "narrative", Status: "active",
			AllowedDirections: []string{"increase"}, ApplicableEntityTypes: []string{"company"},
		}},
		AssertionModalities: []string{"actual", "stated_intent", "source_forecast"},
		MeasurementContract: eventsemantic.MeasurementContract{Representation: "evidence_grounded_narrative", MaxItemsPerSignal: 8, MaxTextCharacters: 2000, RequiresEvidenceIDs: true, NumericValidation: false},
	}
}
