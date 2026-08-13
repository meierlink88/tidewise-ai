package eventsemantic

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeStore struct {
	listEligibleEvents    func(context.Context, int, *EligibleEventCursor) ([]EligibleEvent, error)
	context               func(context.Context, string) (Context, error)
	getEventSemantics     func(context.Context, string) (EventSemanticsResult, error)
	loadContextLeaseState func(context.Context, ContextLeaseRequest, time.Time) (ContextLeaseTransactionState, error)
	saveContextLease      func(context.Context, ContextLeaseWrite) error
	loadSubmissionState   func(context.Context, Submission) (SubmissionTransactionState, error)
	saveSubmission        func(context.Context, SubmissionWrite) error
	loadReviewState       func(context.Context, ReviewSubmission) (ReviewTransactionState, error)
	saveReview            func(context.Context, ReviewWrite) error
}

func (f fakeStore) ListEligibleEvents(ctx context.Context, limit int, cursor *EligibleEventCursor) ([]EligibleEvent, error) {
	return f.listEligibleEvents(ctx, limit, cursor)
}

func (f fakeStore) Context(ctx context.Context, id string) (Context, error) {
	return f.context(ctx, id)
}

func (f fakeStore) GetEventSemantics(ctx context.Context, eventID string) (EventSemanticsResult, error) {
	return f.getEventSemantics(ctx, eventID)
}

func (fakeStore) ListResearchSemantics(context.Context, ResearchSemanticQuery) ([]ResearchSemanticRecord, error) {
	return nil, nil
}

func (fakeStore) ResearchSemanticClosure(context.Context, ResearchSemanticClosureQuery) (ResearchSemanticDictionaries, error) {
	return ResearchSemanticDictionaries{}, nil
}

func (f fakeStore) InTransaction(ctx context.Context, fn func(Transaction) error) error {
	return fn(f)
}

func (f fakeStore) LoadContextLeaseState(ctx context.Context, request ContextLeaseRequest, observedAt time.Time) (ContextLeaseTransactionState, error) {
	return f.loadContextLeaseState(ctx, request, observedAt)
}

func (f fakeStore) SaveContextLease(ctx context.Context, write ContextLeaseWrite) error {
	return f.saveContextLease(ctx, write)
}

func (f fakeStore) LoadSubmissionState(ctx context.Context, submission Submission) (SubmissionTransactionState, error) {
	return f.loadSubmissionState(ctx, submission)
}

func (f fakeStore) SaveSubmission(ctx context.Context, write SubmissionWrite) error {
	return f.saveSubmission(ctx, write)
}

func (f fakeStore) LoadReviewState(ctx context.Context, submission ReviewSubmission) (ReviewTransactionState, error) {
	return f.loadReviewState(ctx, submission)
}

func (f fakeStore) SaveReview(ctx context.Context, write ReviewWrite) error {
	return f.saveReview(ctx, write)
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
	submission := validUseCaseSubmission()
	_, payloadHash, err := canonicalHash(submission)
	if err != nil {
		t.Fatal(err)
	}
	replayed := SubmissionResult{
		SubmissionID:         "20000000-0000-4000-8000-000000000001",
		EventID:              "10000000-0000-4000-8000-000000000001",
		Status:               StatusPendingReview,
		CanonicalPayloadHash: payloadHash,
	}
	store := fakeStore{
		loadSubmissionState: func(context.Context, Submission) (SubmissionTransactionState, error) {
			return SubmissionTransactionState{Existing: &replayed}, nil
		},
		saveSubmission: func(context.Context, SubmissionWrite) error {
			t.Fatal("replay wrote a second Submission")
			return nil
		},
	}
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	result, err := useCase.CreateSubmission(context.Background(), submission)
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
		loadSubmissionState: func(_ context.Context, received Submission) (SubmissionTransactionState, error) {
			if received.ContextLeaseID != submission.ContextLeaseID || received.SupersedesSubmissionID != submission.SupersedesSubmissionID {
				t.Fatalf("pinned submission = %#v", received)
			}
			return SubmissionTransactionState{
				Lease: SubmissionLeaseState{
					Found: true, EventID: submission.EventID, AgentExecutionID: submission.AgentExecutionID,
					Status: "active", LeaseExpiresAt: time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
					SupersedesSubmissionID: submission.SupersedesSubmissionID,
				},
				Context: Context{
					Event: Event{ID: submission.EventID}, OntologyVersion: submission.OntologyVersion,
					PolicyVersion: submission.AcceptancePolicyVersion,
				},
				SupersededSubmission: &SubmissionReference{
					SubmissionID: submission.SupersedesSubmissionID, EventID: submission.EventID,
					Status: StatusAccepted,
				},
			}, nil
		},
		saveSubmission: func(_ context.Context, write SubmissionWrite) error {
			if write.Submission.SupersedesSubmissionID != submission.SupersedesSubmissionID || len(write.Payload) == 0 || !validHash(write.PayloadHash) {
				t.Fatalf("write input = %#v hash=%q", write.Submission, write.PayloadHash)
			}
			if write.Status != StatusRejected || !write.ConsumeLease {
				t.Fatalf("submission decision = status %q consume=%v", write.Status, write.ConsumeLease)
			}
			return nil
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

func TestUseCaseSubmissionDecidesExpiredLeaseAsNotFound(t *testing.T) {
	fixedNow := time.Date(2026, 8, 12, 3, 4, 5, 0, time.UTC)
	submission := validUseCaseSubmission()
	saved := false
	store := fakeStore{
		loadSubmissionState: func(context.Context, Submission) (SubmissionTransactionState, error) {
			return SubmissionTransactionState{Lease: SubmissionLeaseState{
				Found: true, EventID: submission.EventID, AgentExecutionID: submission.AgentExecutionID,
				Status: "active", LeaseExpiresAt: fixedNow,
			}}, nil
		},
		saveSubmission: func(context.Context, SubmissionWrite) error {
			saved = true
			return nil
		},
	}
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	useCase.now = func() time.Time { return fixedNow }
	_, err = useCase.CreateSubmission(context.Background(), submission)
	var notFound *NotFoundError
	if !errors.As(err, &notFound) || notFound.Resource != "Event Semantic Context Lease" || saved {
		t.Fatalf("expired lease error = %v saved=%v", err, saved)
	}
}

func TestUseCaseContextLeaseOwnsTransactionDecision(t *testing.T) {
	fixedNow := time.Date(2026, 8, 12, 1, 2, 3, 0, time.UTC)
	request := ContextLeaseRequest{
		EventID:          "10000000-0000-4000-8000-000000000001",
		AgentExecutionID: "execution", WorkerID: "worker", Lease: 5 * time.Minute,
	}
	expiredLeaseID := "20000000-0000-4000-8000-000000000099"
	var saved ContextLeaseWrite
	store := fakeStore{
		loadContextLeaseState: func(_ context.Context, _ ContextLeaseRequest, observedAt time.Time) (ContextLeaseTransactionState, error) {
			if !observedAt.Equal(fixedNow) {
				t.Fatalf("observed at = %s", observedAt)
			}
			return ContextLeaseTransactionState{
				Event: LeaseEventState{
					Found: true, EventID: request.EventID, EventStatus: "confirmed",
					FactStatus: "verified", InputValid: true,
				},
				ExpiredLeaseIDs: []string{expiredLeaseID},
			}, nil
		},
		saveContextLease: func(_ context.Context, write ContextLeaseWrite) error {
			saved = write
			return nil
		},
	}
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	useCase.now = func() time.Time { return fixedNow }
	useCase.newUUID = func() string { return "20000000-0000-4000-8000-000000000001" }
	result, err := useCase.CreateContextLease(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.ID != "20000000-0000-4000-8000-000000000001" ||
		result.Status != "active" || !result.LeaseExpiresAt.Equal(fixedNow.Add(request.Lease)) {
		t.Fatalf("Context Lease result = %#v", result)
	}
	if saved.Lease != result || saved.AgentExecutionID != request.AgentExecutionID ||
		saved.WorkerID != request.WorkerID || !saved.TransitionedAt.Equal(fixedNow) ||
		len(saved.ExpireLeaseIDs) != 1 || saved.ExpireLeaseIDs[0] != expiredLeaseID {
		t.Fatalf("Context Lease write = %#v", saved)
	}
}

func TestUseCaseContextLeaseRejectsActiveLeaseBeforePersistence(t *testing.T) {
	saved := false
	store := fakeStore{
		loadContextLeaseState: func(context.Context, ContextLeaseRequest, time.Time) (ContextLeaseTransactionState, error) {
			return ContextLeaseTransactionState{
				Event:         LeaseEventState{Found: true, EventStatus: "confirmed", FactStatus: "verified", InputValid: true},
				ActiveLeaseID: "20000000-0000-4000-8000-000000000099",
			}, nil
		},
		saveContextLease: func(context.Context, ContextLeaseWrite) error {
			saved = true
			return nil
		},
	}
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	_, err = useCase.CreateContextLease(context.Background(), ContextLeaseRequest{
		EventID:          "10000000-0000-4000-8000-000000000001",
		AgentExecutionID: "execution", WorkerID: "worker", Lease: 5 * time.Minute,
	})
	var conflict *ConflictError
	if !errors.As(err, &conflict) || conflict.Reason != "Event already has an active Context Lease" || saved {
		t.Fatalf("Context Lease conflict = %v saved=%v", err, saved)
	}
}

func TestUseCaseContextLeaseAuthorsSupersededLeaseConsumption(t *testing.T) {
	const (
		eventID       = "10000000-0000-4000-8000-000000000001"
		priorLeaseID  = "20000000-0000-4000-8000-000000000001"
		priorSubmitID = "30000000-0000-4000-8000-000000000001"
	)
	var saved ContextLeaseWrite
	store := fakeStore{
		loadContextLeaseState: func(context.Context, ContextLeaseRequest, time.Time) (ContextLeaseTransactionState, error) {
			return ContextLeaseTransactionState{
				Event: LeaseEventState{
					Found: true, EventID: eventID, EventStatus: "confirmed", FactStatus: "verified", InputValid: true,
				},
				ActiveLeaseID: priorLeaseID,
				SupersededSubmission: &SubmissionReference{
					SubmissionID: priorSubmitID, EventID: eventID, ContextLeaseID: priorLeaseID, Status: StatusAccepted,
				},
			}, nil
		},
		saveContextLease: func(_ context.Context, write ContextLeaseWrite) error {
			saved = write
			return nil
		},
	}
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	_, err = useCase.CreateContextLease(context.Background(), ContextLeaseRequest{
		EventID: eventID, SupersedesSubmissionID: priorSubmitID,
		AgentExecutionID: "reanalysis", WorkerID: "worker", Lease: 5 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !saved.ConsumeSupersededLease || saved.Lease.SupersedesSubmissionID != priorSubmitID {
		t.Fatalf("Context Lease write = %#v", saved)
	}
}

func TestUseCaseReviewPreservesTypedConflict(t *testing.T) {
	want := &ConflictError{Reason: "frozen review identity changed"}
	const evidenceID = "30000000-0000-4000-8000-000000000001"
	store := fakeStore{
		loadReviewState: func(context.Context, ReviewSubmission) (ReviewTransactionState, error) {
			return ReviewTransactionState{
				Found: true,
				Identity: ReviewIdentity{
					AgentExecutionID: "execution", ReviewerPromptHash: strings.Repeat("a", 64),
					ReviewerModel: "reviewer",
				},
				Submission: &SubmissionResult{
					Status: StatusPendingReview,
					Precheck: PrecheckResult{
						EntityLinks: []CandidateDecision{{CandidateKey: "company", Status: StatusPendingReview}},
						ReviewerWorkPackage: ReviewerWorkPackage{EntityLinks: []EntityLinkCandidate{{
							Key: "company", EvidenceIDs: []string{evidenceID},
						}}},
					},
				},
			}, nil
		},
		saveReview: func(_ context.Context, write ReviewWrite) error {
			if write.Submission.Items[0].CandidateType != CandidateTypeEntityLink || len(write.Payload) == 0 || !validHash(write.PayloadHash) {
				t.Fatalf("review input = %#v hash=%q", write.Submission, write.PayloadHash)
			}
			return want
		},
	}
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	_, err = useCase.SubmitReview(context.Background(), ReviewSubmission{
		SubmissionID: "20000000-0000-4000-8000-000000000001", ReviewerExecutionKey: "execution:reviewer",
		PromptHash: strings.Repeat("a", 64), Model: "reviewer",
		Items: []ReviewItem{{
			CandidateType: CandidateTypeEntityLink, CandidateKey: "company", Decision: ReviewDecisionPass,
			EvidenceIDs: []string{evidenceID},
		}},
	})
	var conflict *ConflictError
	if !errors.As(err, &conflict) || conflict.Reason != want.Reason {
		t.Fatalf("review error = %T %v", err, err)
	}
}

func TestUseCaseReviewOwnsStatusAndSupersessionDecision(t *testing.T) {
	const evidenceID = "30000000-0000-4000-8000-000000000001"
	fixedNow := time.Date(2026, 8, 12, 2, 3, 4, 0, time.UTC)
	var saved ReviewWrite
	store := fakeStore{
		loadReviewState: func(context.Context, ReviewSubmission) (ReviewTransactionState, error) {
			return ReviewTransactionState{
				Found: true,
				Identity: ReviewIdentity{
					AgentExecutionID: "execution", ReviewerPromptHash: strings.Repeat("a", 64),
					ReviewerModel: "reviewer",
				},
				Submission: &SubmissionResult{
					SubmissionID: "20000000-0000-4000-8000-000000000001",
					Status:       StatusPendingReview,
					Precheck: PrecheckResult{
						EntityLinks: []CandidateDecision{{CandidateKey: "company", Status: StatusPendingReview}},
						ReviewerWorkPackage: ReviewerWorkPackage{EntityLinks: []EntityLinkCandidate{{
							Key: "company", EvidenceIDs: []string{evidenceID},
						}}},
					},
				},
				RetryBudget: 1,
			}, nil
		},
		saveReview: func(_ context.Context, write ReviewWrite) error {
			saved = write
			return nil
		},
	}
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	useCase.now = func() time.Time { return fixedNow }
	useCase.newUUID = func() string { return "40000000-0000-4000-8000-000000000001" }
	result, err := useCase.SubmitReview(context.Background(), ReviewSubmission{
		SubmissionID:         "20000000-0000-4000-8000-000000000001",
		ReviewerExecutionKey: "execution:reviewer", PromptHash: strings.Repeat("a", 64), Model: "reviewer",
		Items: []ReviewItem{{
			CandidateType: CandidateTypeEntityLink, CandidateKey: "company",
			Decision: ReviewDecisionPass, EvidenceIDs: []string{evidenceID},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != StatusAccepted || saved.Status != StatusAccepted ||
		!saved.SupersedePrior || !saved.ConsumeLease || saved.FinalizedAt == nil ||
		!saved.FinalizedAt.Equal(fixedNow) || saved.SnapshotID == "" {
		t.Fatalf("review result=%#v write=%#v", result, saved)
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
		AgentVersion:        "event-semantic-enricher.v4",
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

func validPrecheckContext() Context {
	return Context{
		Event:    Event{ID: "event", Title: "某公司收入上升", Status: "confirmed", FactStatus: "verified"},
		Evidence: []Evidence{{ID: "event-evidence", Statement: "某公司收入上升"}},
		Entities: []Entity{{ID: "company", Type: "company", Name: "某公司", CanonicalName: "某公司", Status: "active"}},
		Variables: []VariableDefinition{{
			Key: "revenue", Version: 1, Status: "active", AllowedDirections: []string{"increase"},
			ApplicableEntityTypes: []string{"company"},
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

func TestSemanticReviewIdentityIsBoundToFrozenRun(t *testing.T) {
	identity := ReviewIdentity{
		AgentExecutionID:   "execution",
		ReviewerPromptHash: "reviewer-hash", ReviewerModel: "reviewer-model",
		AdjudicatorPromptHash: "adjudicator-hash", AdjudicatorModel: "adjudicator-model",
	}
	if !identity.Matches(ReviewSubmission{
		ReviewerExecutionKey: "execution:reviewer",
		PromptHash:           "reviewer-hash", Model: "reviewer-model",
	}) {
		t.Fatal("expected frozen reviewer identity to match")
	}
	if identity.Matches(ReviewSubmission{
		ReviewerExecutionKey: "other-execution:reviewer",
		PromptHash:           "reviewer-hash", Model: "reviewer-model",
	}) {
		t.Fatal("review lineage from another execution was accepted")
	}
	if identity.Matches(ReviewSubmission{
		ReviewerExecutionKey: "execution:reviewer",
		PromptHash:           "adjudicator-hash", Model: "reviewer-model",
	}) {
		t.Fatal("mismatched prompt hash was accepted")
	}
}
