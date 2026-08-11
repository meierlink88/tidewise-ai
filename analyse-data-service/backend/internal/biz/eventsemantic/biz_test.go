package eventsemantic

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeStore struct {
	listEligibleEvents func(context.Context, int, *EligibleEventCursor) ([]EligibleEvent, error)
	context            func(context.Context, string) (Context, error)
	submissionContext  func(context.Context, string, Submission) (Context, error)
	replaySubmission   func(context.Context, string, string) (SubmissionResult, bool, error)
	createContextLease func(context.Context, ContextLeaseRequest) (ContextLease, error)
	createSubmission   func(context.Context, Submission, PrecheckResult, []byte, string) (SubmissionResult, error)
	submitReview       func(context.Context, ReviewSubmission, []byte, string) (SubmissionResult, error)
	getEventSemantics  func(context.Context, string) (EventSemanticsResult, error)
}

func (f fakeStore) ListEligibleEvents(ctx context.Context, limit int, cursor *EligibleEventCursor) ([]EligibleEvent, error) {
	return f.listEligibleEvents(ctx, limit, cursor)
}

func (f fakeStore) Context(ctx context.Context, id string) (Context, error) {
	return f.context(ctx, id)
}

func (f fakeStore) SubmissionContext(ctx context.Context, id string, submission Submission) (Context, error) {
	return f.submissionContext(ctx, id, submission)
}

func (f fakeStore) ReplaySubmission(ctx context.Context, executionID, hash string) (SubmissionResult, bool, error) {
	return f.replaySubmission(ctx, executionID, hash)
}

func (f fakeStore) CreateContextLease(ctx context.Context, request ContextLeaseRequest) (ContextLease, error) {
	return f.createContextLease(ctx, request)
}

func (f fakeStore) CreateSubmission(
	ctx context.Context,
	submission Submission,
	precheck PrecheckResult,
	payload []byte,
	hash string,
) (SubmissionResult, error) {
	return f.createSubmission(ctx, submission, precheck, payload, hash)
}

func (f fakeStore) SubmitReview(
	ctx context.Context,
	submission ReviewSubmission,
	payload []byte,
	hash string,
) (SubmissionResult, error) {
	return f.submitReview(ctx, submission, payload, hash)
}

func (f fakeStore) GetEventSemantics(ctx context.Context, eventID string) (EventSemanticsResult, error) {
	return f.getEventSemantics(ctx, eventID)
}

func TestUseCaseEligibilityCursorPreservesStableKeyset(t *testing.T) {
	firstSeen := time.Date(2026, 8, 11, 1, 2, 3, 0, time.UTC)
	items := []EligibleEvent{
		{EventID: "10000000-0000-4000-8000-000000000001", FirstSeenAt: firstSeen},
		{EventID: "10000000-0000-4000-8000-000000000002", FirstSeenAt: firstSeen.Add(time.Second)},
		{EventID: "10000000-0000-4000-8000-000000000003", FirstSeenAt: firstSeen.Add(2 * time.Second)},
	}
	var receivedCursor *EligibleEventCursor
	store := fakeStore{listEligibleEvents: func(_ context.Context, limit int, cursor *EligibleEventCursor) ([]EligibleEvent, error) {
		if limit != 3 {
			t.Fatalf("repository limit = %d", limit)
		}
		receivedCursor = cursor
		if cursor == nil {
			return items, nil
		}
		return nil, nil
	}}
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	page, err := useCase.ListEligibleEvents(context.Background(), 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Events) != 2 || page.NextCursor == "" {
		t.Fatalf("page = %#v", page)
	}
	if _, err := useCase.ListEligibleEvents(context.Background(), 2, page.NextCursor); err != nil {
		t.Fatal(err)
	}
	if receivedCursor == nil || receivedCursor.EventID != items[1].EventID || !receivedCursor.FirstSeenAt.Equal(items[1].FirstSeenAt) {
		t.Fatalf("decoded cursor = %#v", receivedCursor)
	}
}

func TestUseCaseSubmissionReplayBypassesPinnedContextAndWrite(t *testing.T) {
	replayed := SubmissionResult{
		SubmissionID: "20000000-0000-4000-8000-000000000001",
		EventID:      "10000000-0000-4000-8000-000000000001",
		Status:       StatusPendingReview,
	}
	store := fakeStore{
		replaySubmission: func(context.Context, string, string) (SubmissionResult, bool, error) {
			return replayed, true, nil
		},
		submissionContext: func(context.Context, string, Submission) (Context, error) {
			t.Fatal("replay read pinned Context")
			return Context{}, nil
		},
		createSubmission: func(context.Context, Submission, PrecheckResult, []byte, string) (SubmissionResult, error) {
			t.Fatal("replay wrote a second Submission")
			return SubmissionResult{}, nil
		},
	}
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	result, err := useCase.CreateSubmission(context.Background(), validUseCaseSubmission())
	if err != nil {
		t.Fatal(err)
	}
	if result.SubmissionID != replayed.SubmissionID {
		t.Fatalf("replay = %#v", result)
	}
}

func TestUseCaseSubmissionPinsContextAndCarriesSupersession(t *testing.T) {
	submission := validUseCaseSubmission()
	submission.SupersedesSubmissionID = "20000000-0000-4000-8000-000000000002"
	store := fakeStore{
		replaySubmission: func(context.Context, string, string) (SubmissionResult, bool, error) {
			return SubmissionResult{}, false, nil
		},
		submissionContext: func(_ context.Context, leaseID string, received Submission) (Context, error) {
			if leaseID != submission.ContextLeaseID || received.SupersedesSubmissionID != submission.SupersedesSubmissionID {
				t.Fatalf("pinned submission = %#v", received)
			}
			return Context{
				Event: Event{ID: submission.EventID}, OntologyVersion: submission.OntologyVersion,
				PolicyVersion: submission.AcceptancePolicyVersion,
			}, nil
		},
		createSubmission: func(_ context.Context, received Submission, _ PrecheckResult, payload []byte, hash string) (SubmissionResult, error) {
			if received.SupersedesSubmissionID != submission.SupersedesSubmissionID || len(payload) == 0 || !validHash(hash) {
				t.Fatalf("write input = %#v hash=%q", received, hash)
			}
			return SubmissionResult{SubmissionID: "20000000-0000-4000-8000-000000000003"}, nil
		},
	}
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := useCase.CreateSubmission(context.Background(), submission); err != nil {
		t.Fatal(err)
	}
}

func TestUseCaseReviewPreservesTypedConflict(t *testing.T) {
	want := &ConflictError{Reason: "frozen review identity changed"}
	store := fakeStore{submitReview: func(_ context.Context, received ReviewSubmission, payload []byte, hash string) (SubmissionResult, error) {
		if received.Items[0].CandidateType != CandidateTypeEntityLink || len(payload) == 0 || !validHash(hash) {
			t.Fatalf("review input = %#v hash=%q", received, hash)
		}
		return SubmissionResult{}, want
	}}
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	_, err = useCase.SubmitReview(context.Background(), ReviewSubmission{
		SubmissionID: "20000000-0000-4000-8000-000000000001", ReviewerExecutionKey: "execution:reviewer",
		PromptHash: strings.Repeat("a", 64), Model: "reviewer",
		Items: []ReviewItem{{
			CandidateType: CandidateTypeEntityLink, CandidateKey: "company", Decision: ReviewDecisionPass,
			EvidenceIDs: []string{"30000000-0000-4000-8000-000000000001"},
		}},
	})
	var conflict *ConflictError
	if !errors.As(err, &conflict) || conflict.Reason != want.Reason {
		t.Fatalf("review error = %T %v", err, err)
	}
}

func TestApplyReviewPropagatesUpstreamRejection(t *testing.T) {
	const evidenceID = "30000000-0000-4000-8000-000000000001"
	precheck := PrecheckResult{
		EntityLinks:     []CandidateDecision{{CandidateKey: "company", Status: StatusPendingReview}},
		VariableSignals: []CandidateDecision{{CandidateKey: "revenue", Status: StatusPendingReview}},
		ReviewerWorkPackage: ReviewerWorkPackage{
			EntityLinks: []EntityLinkCandidate{{Key: "company", EvidenceIDs: []string{evidenceID}}},
			VariableSignals: []VariableSignalCandidate{{
				Key: "revenue", SubjectLinkKey: "company", EvidenceIDs: []string{evidenceID},
			}},
		},
	}
	err := ApplyReview(&precheck, ReviewSubmission{Items: []ReviewItem{
		{CandidateType: CandidateTypeEntityLink, CandidateKey: "company", Decision: ReviewDecisionFail,
			ReasonCodes: []string{"unsupported"}, EvidenceIDs: []string{evidenceID}},
		{CandidateType: CandidateTypeVariableSignal, CandidateKey: "revenue", Decision: ReviewDecisionPass,
			EvidenceIDs: []string{evidenceID}},
	}}, false)
	if err != nil {
		t.Fatal(err)
	}
	assertCandidateStatus(t, precheck.EntityLinks, "company", StatusRejected, "unsupported")
	assertCandidateStatus(t, precheck.VariableSignals, "revenue", StatusRejected, "upstream_rejected")
	if status := SummarizeSubmission(precheck); status != StatusRejected {
		t.Fatalf("submission status = %q", status)
	}
}

func TestApplyReviewRejectsEvidenceOutsideFrozenCandidate(t *testing.T) {
	precheck := PrecheckResult{
		EntityLinks: []CandidateDecision{{CandidateKey: "company", Status: StatusPendingReview}},
		ReviewerWorkPackage: ReviewerWorkPackage{EntityLinks: []EntityLinkCandidate{{
			Key: "company", EvidenceIDs: []string{"30000000-0000-4000-8000-000000000001"},
		}}},
	}
	err := ApplyReview(&precheck, ReviewSubmission{Items: []ReviewItem{{
		CandidateType: CandidateTypeEntityLink, CandidateKey: "company", Decision: ReviewDecisionPass,
		EvidenceIDs: []string{"30000000-0000-4000-8000-000000000099"},
	}}}, false)
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("review error = %T %v", err, err)
	}
}

func validUseCaseSubmission() Submission {
	return Submission{
		ContextLeaseID:   "10000000-0000-4000-8000-000000000002",
		EventID:          "10000000-0000-4000-8000-000000000001",
		AgentExecutionID: "execution", AgentKey: "event-semantic-enricher",
		AgentVersion:        "event-semantic-enricher.v3",
		GeneratorPromptHash: strings.Repeat("a", 64), GeneratorModel: "generator",
		ReviewerPromptHash: strings.Repeat("b", 64), ReviewerModel: "reviewer",
		AdjudicatorPromptHash: strings.Repeat("c", 64), AdjudicatorModel: "adjudicator",
		OntologyVersion: "event-semantics.objective-v3@1", AcceptancePolicyVersion: "event-semantics.objective-v2@1",
	}
}

func TestValidConfidenceMatchesOpenAPIDecimalStringContract(t *testing.T) {
	for _, value := range []string{"", "0", "1", "0.5", "0.00001", "1.0", "1.00000"} {
		if !validConfidence(value) {
			t.Errorf("validConfidence(%q) = false, want true", value)
		}
	}
	for _, value := range []string{".5", "1e-3", "+0.5", "00.5", "0.", "0.000001", "1.00001", " 0.5 ", "-0.1", "2"} {
		if validConfidence(value) {
			t.Errorf("validConfidence(%q) = true, want false", value)
		}
	}
}

func TestPrecheckAcceptsNarrativeMeasurementsWithoutDirectImpact(t *testing.T) {
	semanticContext := Context{
		Event: Event{
			ID:         "11111111-1111-4111-8111-111111111111",
			Title:      "某公司公布净利润为17至18.3亿元",
			Summary:    "同比增长41.5%至52.32%",
			Status:     "confirmed",
			FactStatus: "verified",
		},
		Evidence: []Evidence{{
			ID:        "22222222-2222-4222-8222-222222222222",
			Statement: "某公司公布净利润为17至18.3亿元，同比增长41.5%至52.32%",
		}},
		Entities: []Entity{{
			ID: "33333333-3333-4333-8333-333333333333", Type: "company",
			Name: "某公司", CanonicalName: "某公司", Status: "active",
		}},
		Variables: []VariableDefinition{{
			Key: "net_profit", Version: 1, Status: "active",
			AllowedDirections: []string{"increase"}, ApplicableEntityTypes: []string{"company"},
		}},
		EntityTypes: []EntityTypeDefinition{{
			TypeKey: "company", Version: 1, Status: "active", EventLinkAllowed: true, SignalSubjectAllowed: true,
			AllowedEventRoles: []string{"event_subject"},
		}},
		AssertionModalities: []string{"actual", "stated_intent", "source_forecast"},
		MeasurementContract: MeasurementContract{
			Representation: "evidence_grounded_narrative", MaxItemsPerSignal: 8,
			MaxTextCharacters: 2000, RequiresEvidenceIDs: true,
		},
	}

	result := Precheck(semanticContext, Submission{
		EventID: semanticContext.Event.ID,
		EntityLinks: []EntityLinkCandidate{{
			Key: "company", Mention: "某公司", EntityID: semanticContext.Entities[0].ID,
			ProjectedEntityType: "company",
			EntityRole:          "event_subject", EvidenceIDs: []string{semanticContext.Evidence[0].ID},
			ResolutionMethod: "qdrant_exact",
		}, {
			Key: "company-duplicate", Mention: "某公司", EntityID: semanticContext.Entities[0].ID,
			ProjectedEntityType: "company",
			EntityRole:          "event_subject", EvidenceIDs: []string{semanticContext.Evidence[0].ID},
			ResolutionMethod: "qdrant_vector",
		}},
		VariableSignals: []VariableSignalCandidate{{
			Key: "net-profit", SubjectLinkKey: "company", VariableKey: "net_profit",
			VariableVersion: 1, Direction: "increase", AssertionModality: "actual",
			EvidenceIDs: []string{semanticContext.Evidence[0].ID},
			Measurements: []MeasurementValue{
				{Text: "净利润为17至18.3亿元", EvidenceIDs: []string{semanticContext.Evidence[0].ID}},
				{Text: "同比增长41.5%至52.32%", EvidenceIDs: []string{semanticContext.Evidence[0].ID}},
			},
		}, {
			Key: "net-profit-intent", SubjectLinkKey: "company", VariableKey: "net_profit",
			VariableVersion: 1, Direction: "increase", AssertionModality: "stated_intent",
			EvidenceIDs: []string{semanticContext.Evidence[0].ID},
		}},
	})

	assertCandidateStatus(t, result.EntityLinks, "company", StatusPendingReview, "")
	assertCandidateStatus(t, result.EntityLinks, "company-duplicate", StatusRejected, "duplicate_entity_link")
	assertCandidateStatus(t, result.VariableSignals, "net-profit", StatusPendingReview, "")
	assertCandidateStatus(t, result.VariableSignals, "net-profit-intent", StatusPendingReview, "")
	if len(result.ReviewerWorkPackage.EntityLinks) != 1 ||
		len(result.ReviewerWorkPackage.ResolvedEntities) != 1 ||
		result.ReviewerWorkPackage.ResolvedEntities[0].ID != semanticContext.Entities[0].ID ||
		len(result.ReviewerWorkPackage.VariableSignals) != 2 {
		t.Fatalf("reviewer work package = %#v", result.ReviewerWorkPackage)
	}
}

func TestPrecheckRejectsNarrativeMeasurementWithoutEventEvidence(t *testing.T) {
	semanticContext := Context{
		Event:    Event{ID: "event", Title: "公司收入上升", Status: "confirmed", FactStatus: "verified"},
		Evidence: []Evidence{{ID: "event-evidence", Statement: "公司收入上升"}},
		Entities: []Entity{{ID: "company", Type: "company", Name: "公司", CanonicalName: "公司", Status: "active"}},
		Variables: []VariableDefinition{{
			Key: "revenue", Version: 1, Status: "active", AllowedDirections: []string{"increase"},
			ApplicableEntityTypes: []string{"company"},
		}},
		EntityTypes: []EntityTypeDefinition{{
			TypeKey: "company", Version: 1, Status: "active", EventLinkAllowed: true, SignalSubjectAllowed: true,
			AllowedEventRoles: []string{"event_subject"},
		}},
		AssertionModalities: []string{"actual", "stated_intent", "source_forecast"},
		MeasurementContract: MeasurementContract{
			Representation: "evidence_grounded_narrative", MaxItemsPerSignal: 8,
			MaxTextCharacters: 2000, RequiresEvidenceIDs: true,
		},
	}
	result := Precheck(semanticContext, Submission{
		EntityLinks: []EntityLinkCandidate{{
			Key: "company", Mention: "公司", EntityID: "company", EntityRole: "event_subject",
			ProjectedEntityType: "company",
			EvidenceIDs:         []string{"event-evidence"}, ResolutionMethod: "qdrant_vector",
		}},
		VariableSignals: []VariableSignalCandidate{{
			Key: "revenue", SubjectLinkKey: "company", VariableKey: "revenue", VariableVersion: 1,
			Direction: "increase", AssertionModality: "actual", EvidenceIDs: []string{"event-evidence"},
			Measurements: []MeasurementValue{{Text: "收入增长30%", EvidenceIDs: []string{"other-event-evidence"}}},
		}},
	})

	assertCandidateStatus(t, result.VariableSignals, "revenue", StatusRejected, "evidence_not_in_event")
}

func TestPrecheckRejectsDuplicateEvidenceLineage(t *testing.T) {
	semanticContext := validPrecheckContext()
	result := Precheck(semanticContext, Submission{EntityLinks: []EntityLinkCandidate{{
		Key: "company", Mention: "某公司", EntityID: "company", ProjectedEntityType: "company",
		EntityRole: "event_subject", EvidenceIDs: []string{"event-evidence", "event-evidence"}, ResolutionMethod: "qdrant_exact",
	}}})
	assertCandidateStatus(t, result.EntityLinks, "company", StatusRejected, "evidence_not_in_event")
}

func TestPrecheckAcceptsMentionWithValidEventEvidenceMembership(t *testing.T) {
	semanticContext := Context{
		Event: Event{
			ID: "event", Summary: "摘要专有词", Status: "confirmed", FactStatus: "verified",
		},
		Evidence: []Evidence{{ID: "event-evidence", Statement: "证据没有该实体称谓"}},
		Entities: []Entity{{
			ID: "company", Type: "company", Name: "摘要专有词", CanonicalName: "摘要专有词", Status: "active",
		}},
		EntityTypes: []EntityTypeDefinition{{
			TypeKey: "company", Version: 1, Status: "active", EventLinkAllowed: true, AllowedEventRoles: []string{"event_subject"},
		}},
	}
	result := Precheck(semanticContext, Submission{EntityLinks: []EntityLinkCandidate{{
		Key: "company", Mention: "摘要专有词", EntityID: "company", EntityRole: "event_subject",
		ProjectedEntityType: "company",
		EvidenceIDs:         []string{"event-evidence"}, ResolutionMethod: "qdrant_exact",
	}}})

	assertCandidateStatus(t, result.EntityLinks, "company", StatusPendingReview, "")
}

func TestPrecheckDoesNotRequirePrimaryEvidenceDesignation(t *testing.T) {
	semanticContext := Context{
		Event:    Event{ID: "event", Summary: "摘要专有词", Status: "confirmed", FactStatus: "verified"},
		Evidence: []Evidence{{ID: "event-evidence", Statement: "证据没有该实体称谓", Relation: "supports"}},
		Entities: []Entity{{ID: "company", Type: "company", Name: "摘要专有词", CanonicalName: "摘要专有词", Status: "active"}},
		EntityTypes: []EntityTypeDefinition{{
			TypeKey: "company", Version: 1, Status: "active", EventLinkAllowed: true, AllowedEventRoles: []string{"event_subject"},
		}},
	}
	result := Precheck(semanticContext, Submission{EntityLinks: []EntityLinkCandidate{{
		Key: "company", Mention: "摘要专有词", EntityID: "company", EntityRole: "event_subject",
		ProjectedEntityType: "company",
		EvidenceIDs:         []string{"event-evidence"}, ResolutionMethod: "qdrant_exact",
	}}})

	assertCandidateStatus(t, result.EntityLinks, "company", StatusPendingReview, "")
}

func TestPrecheckRejectsProjectedEntityTypeDrift(t *testing.T) {
	semanticContext := validPrecheckContext()
	result := Precheck(semanticContext, Submission{EntityLinks: []EntityLinkCandidate{{
		Key: "company", Mention: "某公司", EntityID: "company", ProjectedEntityType: "product",
		EntityRole: "event_subject", EvidenceIDs: []string{"event-evidence"}, ResolutionMethod: "qdrant_exact",
	}}})
	assertCandidateStatus(t, result.EntityLinks, "company", StatusRejected, "entity_projection_type_mismatch")
}

func TestPrecheckRejectsSignalWhenFormalEntityTypeCannotBeSignalSubject(t *testing.T) {
	semanticContext := validPrecheckContext()
	semanticContext.EntityTypes[0].SignalSubjectAllowed = false
	result := Precheck(semanticContext, Submission{
		EntityLinks: []EntityLinkCandidate{{
			Key: "company", Mention: "某公司", EntityID: "company", ProjectedEntityType: "company",
			EntityRole: "event_subject", EvidenceIDs: []string{"event-evidence"}, ResolutionMethod: "qdrant_exact",
		}},
		VariableSignals: []VariableSignalCandidate{{
			Key: "revenue", SubjectLinkKey: "company", VariableKey: "revenue", VariableVersion: 1,
			Direction: "increase", AssertionModality: "actual", EvidenceIDs: []string{"event-evidence"},
		}},
	})
	assertCandidateStatus(t, result.EntityLinks, "company", StatusPendingReview, "")
	assertCandidateStatus(t, result.VariableSignals, "revenue", StatusRejected, "signal_subject_not_allowed")
}

func validPrecheckContext() Context {
	return Context{
		Event:    Event{ID: "event", Title: "某公司收入上升", Status: "confirmed", FactStatus: "verified"},
		Evidence: []Evidence{{ID: "event-evidence", Statement: "某公司收入上升"}},
		Entities: []Entity{{ID: "company", Type: "company", Name: "某公司", CanonicalName: "某公司", Status: "active"}},
		Variables: []VariableDefinition{{
			Key: "revenue", Version: 1, Status: "active", AllowedDirections: []string{"increase"},
			ApplicableEntityTypes: []string{"company"},
		}},
		EntityTypes: []EntityTypeDefinition{{
			TypeKey: "company", Version: 1, Status: "active", EventLinkAllowed: true, SignalSubjectAllowed: true,
			AllowedEventRoles: []string{"event_subject"},
		}},
		AssertionModalities: []string{"actual"},
		MeasurementContract: MeasurementContract{
			Representation: "evidence_grounded_narrative", MaxItemsPerSignal: 8, MaxTextCharacters: 2000,
			RequiresEvidenceIDs: true,
		},
	}
}

func assertCandidateStatus(
	t *testing.T,
	items []CandidateDecision,
	key string,
	status ReviewStatus,
	reason string,
) {
	t.Helper()
	for _, item := range items {
		if item.CandidateKey == key {
			if item.Status != status || item.ReasonCode != reason {
				t.Fatalf("candidate %q = %#v, want status=%q reason=%q", key, item, status, reason)
			}
			return
		}
	}
	t.Fatalf("candidate %q not found in %#v", key, items)
}

func TestNoSemanticCandidatesProduceARealRejectedSubmissionOutcome(t *testing.T) {
	status := SummarizeSubmission(PrecheckResult{})
	if status != StatusRejected {
		t.Fatalf("status = %q, want rejected", status)
	}
}
