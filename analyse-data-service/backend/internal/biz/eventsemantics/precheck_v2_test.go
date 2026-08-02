package eventsemantics

import "testing"

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
