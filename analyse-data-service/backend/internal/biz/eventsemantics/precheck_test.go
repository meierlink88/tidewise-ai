package eventsemantics

import (
	"strings"
	"testing"
	"time"
)

func TestPrecheckKeepsValidCandidatesReviewableAndRejectsInvalidItemsIndependently(t *testing.T) {
	context := Context{
		Event: Event{
			ID: "11111111-1111-4111-8111-111111111111", Status: "confirmed",
			FactStatus: "verified", OccurredAt: timePtr(time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)),
		},
		Evidence: []Evidence{{ID: "22222222-2222-4222-8222-222222222222"}},
		Entities: []Entity{
			{ID: "33333333-3333-4333-8333-333333333333", Type: "company", Name: "某晶圆厂", CanonicalName: "某晶圆厂", Status: "active"},
			{ID: "44444444-4444-4444-8444-444444444444", Type: "product", Name: "8英寸晶圆", CanonicalName: "8英寸晶圆", Status: "active"},
		},
		Variables: []VariableDefinition{
			{Key: "production_volume", Version: 1, Status: "active", AllowedDirections: []string{"increase", "decrease"}, ApplicableEntityTypes: []string{"company"}},
			{Key: "market_supply", Version: 1, Status: "active", AllowedDirections: []string{"increase", "decrease"}, ApplicableEntityTypes: []string{"product"}},
		},
		Relations: []EntityRelation{{
			ID:           "55555555-5555-4555-8555-555555555555",
			FromEntityID: "33333333-3333-4333-8333-333333333333",
			ToEntityID:   "44444444-4444-4444-8444-444444444444",
			Type:         "produces", Status: "active",
		}},
		Rules: []DirectTransmissionRule{{
			Key: "production_decrease_reduces_product_supply", Version: 1, Status: "approved",
			SourceEntityType: "company", SourceVariableKey: "production_volume", SourceVariableVersion: 1,
			SourceDirection: "decrease", RelationType: "produces", TargetEntityType: "product",
			AffectedVariableKey: "market_supply", AffectedVariableVersion: 1, AffectedDirection: "decrease",
		}},
	}
	submission := Submission{
		EventID: context.Event.ID,
		EntityLinks: []EntityLinkCandidate{{
			Key: "company", Mention: "某晶圆厂", EntityID: context.Entities[0].ID,
			EntityRole: "actor", EvidenceIDs: []string{context.Evidence[0].ID},
			ResolutionMethod: "data_service_resolution",
		}},
		VariableSignals: []VariableSignalCandidate{
			{
				Key: "production", SubjectLinkKey: "company", VariableKey: "production_volume",
				VariableVersion: 1, Direction: "decrease", AssertionModality: "actual",
				EvidenceIDs: []string{context.Evidence[0].ID},
			},
			{
				Key: "invented", SubjectLinkKey: "company", VariableKey: "capacity",
				VariableVersion: 1, Direction: "decrease", AssertionModality: "actual",
				EvidenceIDs: []string{context.Evidence[0].ID},
			},
		},
		DirectImpacts: []DirectImpactCandidate{
			{
				Key: "supply", SourceSignalKey: "production", TargetEntityID: context.Entities[1].ID,
				AffectedVariableKey: "market_supply", AffectedVariableVersion: 1,
				AffectedDirection: "decrease", DerivationType: "rule_inferred",
				EntityRelationID: context.Relations[0].ID,
				RuleKey:          context.Rules[0].Key, RuleVersion: 1,
				EvidenceIDs: []string{context.Evidence[0].ID},
			},
			{
				Key: "self", SourceSignalKey: "production", TargetEntityID: context.Entities[0].ID,
				AffectedVariableKey: "market_supply", AffectedVariableVersion: 1,
				AffectedDirection: "decrease", DerivationType: "rule_inferred",
				EntityRelationID: context.Relations[0].ID,
				RuleKey:          context.Rules[0].Key, RuleVersion: 1,
				EvidenceIDs: []string{context.Evidence[0].ID},
			},
		},
	}

	result := Precheck(context, submission)

	assertCandidateStatus(t, result.EntityLinks, "company", StatusPendingReview, "")
	assertCandidateStatus(t, result.VariableSignals, "production", StatusPendingReview, "")
	assertCandidateStatus(t, result.VariableSignals, "invented", StatusRejected, "variable_not_found")
	assertCandidateStatus(t, result.DirectImpacts, "supply", StatusPendingReview, "")
	assertCandidateStatus(t, result.DirectImpacts, "self", StatusRejected, "target_equals_subject")
	if len(result.ReviewerWorkPackage.EntityLinks) != 1 ||
		len(result.ReviewerWorkPackage.VariableSignals) != 1 ||
		len(result.ReviewerWorkPackage.DirectImpacts) != 1 {
		t.Fatalf("reviewer work package = %#v", result.ReviewerWorkPackage)
	}
}

func TestPrecheckRejectsEntityRoleOutsideControlledVocabulary(t *testing.T) {
	context := Context{
		Event:    Event{ID: "event", Status: "confirmed", FactStatus: "verified"},
		Evidence: []Evidence{{ID: "evidence"}},
		Entities: []Entity{{
			ID: "company", Type: "company", Name: "company",
			CanonicalName: "company", Status: "active",
		}},
	}
	result := Precheck(context, Submission{
		EventID: "event",
		EntityLinks: []EntityLinkCandidate{{
			Key: "company", Mention: "company", EntityID: "company",
			EntityRole: "beneficiary", EvidenceIDs: []string{"evidence"},
			ResolutionMethod: "data_service_resolution",
		}},
	})

	assertCandidateStatus(
		t, result.EntityLinks, "company", StatusRejected, "entity_role_invalid",
	)
}

func TestPrecheckAcceptsDataOwnedAnchorReceiptWithoutExactMentionMatch(t *testing.T) {
	context := Context{
		Event:    Event{ID: "event", Status: "confirmed", FactStatus: "verified"},
		Evidence: []Evidence{{ID: "evidence"}},
		Entities: []Entity{{
			ID: "chain-node", Type: "chain_node", Name: "Formal Node",
			CanonicalName: "Formal Node", Status: "active",
		}},
	}
	result := Precheck(context, Submission{EntityLinks: []EntityLinkCandidate{{
		Key: "node", Mention: "non exact event phrase", EntityID: "chain-node",
		EntityRole: "event_subject", EvidenceIDs: []string{"evidence"},
		ResolutionMethod: "data_service_anchor_resolution",
		ResolutionReceipt: &ResolutionReceipt{
			TargetEntityID: "chain-node", PathFingerprint: strings.Repeat("a", 64),
		},
	}}})

	assertCandidateStatus(t, result.EntityLinks, "node", StatusPendingReview, "")
}

func TestPrecheckPropagatesRejectedUpstreamWithoutRejectingTheWholeSnapshot(t *testing.T) {
	context := Context{
		Event:    Event{ID: "event", Status: "confirmed", FactStatus: "verified", OccurredAt: timePtr(time.Now())},
		Evidence: []Evidence{{ID: "evidence"}},
		Entities: []Entity{{ID: "company", Type: "company", Name: "company", CanonicalName: "company", Status: "active"}},
		Variables: []VariableDefinition{{
			Key: "revenue", Version: 1, Status: "active",
			AllowedDirections: []string{"increase"}, ApplicableEntityTypes: []string{"company"},
		}},
	}
	result := Precheck(context, Submission{
		EventID: "event",
		EntityLinks: []EntityLinkCandidate{{
			Key: "bad-link", Mention: "company", EntityID: "company", EntityRole: "actor",
			EvidenceIDs: []string{"not-this-event"}, ResolutionMethod: "data_service_resolution",
		}},
		VariableSignals: []VariableSignalCandidate{{
			Key: "revenue", SubjectLinkKey: "bad-link", VariableKey: "revenue",
			VariableVersion: 1, Direction: "increase", AssertionModality: "actual",
			EvidenceIDs: []string{"evidence"},
		}},
	})

	assertCandidateStatus(t, result.EntityLinks, "bad-link", StatusRejected, "evidence_not_in_event")
	assertCandidateStatus(t, result.VariableSignals, "revenue", StatusRejected, "upstream_rejected")
}

func TestPrecheckRejectsRuleWhenEntityRelationOnlyConnectsInReverse(t *testing.T) {
	context := Context{
		Event:    Event{ID: "event", Status: "confirmed", FactStatus: "verified", OccurredAt: timePtr(time.Now())},
		Evidence: []Evidence{{ID: "evidence"}},
		Entities: []Entity{
			{ID: "company", Type: "company", Name: "company", CanonicalName: "company", Status: "active"},
			{ID: "product", Type: "product", Name: "product", CanonicalName: "product", Status: "active"},
		},
		Variables: []VariableDefinition{
			{Key: "production_volume", Version: 1, Status: "active", AllowedDirections: []string{"decrease"}, ApplicableEntityTypes: []string{"company"}},
			{Key: "market_supply", Version: 1, Status: "active", AllowedDirections: []string{"decrease"}, ApplicableEntityTypes: []string{"product"}},
		},
		Relations: []EntityRelation{{
			ID: "reverse-produces", FromEntityID: "product", ToEntityID: "company",
			Type: "produces", Status: "active",
		}},
		Rules: []DirectTransmissionRule{{
			Key: "production_decrease_reduces_product_supply", Version: 1, Status: "approved",
			SourceEntityType: "company", SourceVariableKey: "production_volume", SourceVariableVersion: 1,
			SourceDirection: "decrease", RelationType: "produces", TargetEntityType: "product",
			AffectedVariableKey: "market_supply", AffectedVariableVersion: 1, AffectedDirection: "decrease",
		}},
	}
	result := Precheck(context, Submission{
		EntityLinks: []EntityLinkCandidate{{
			Key: "company", Mention: "company", EntityID: "company", EntityRole: "actor",
			EvidenceIDs: []string{"evidence"}, ResolutionMethod: "data_service_resolution",
		}},
		VariableSignals: []VariableSignalCandidate{{
			Key: "production", SubjectLinkKey: "company", VariableKey: "production_volume",
			VariableVersion: 1, Direction: "decrease", AssertionModality: "actual",
			EvidenceIDs: []string{"evidence"},
		}},
		DirectImpacts: []DirectImpactCandidate{{
			Key: "impact", SourceSignalKey: "production", TargetEntityID: "product",
			AffectedVariableKey: "market_supply", AffectedVariableVersion: 1,
			AffectedDirection: "decrease", DerivationType: "rule_inferred",
			EntityRelationID: "reverse-produces",
			RuleKey:          "production_decrease_reduces_product_supply", RuleVersion: 1,
			EvidenceIDs: []string{"evidence"},
		}},
	})

	assertCandidateStatus(t, result.DirectImpacts, "impact", StatusRejected, "relation_not_found")
}

func TestPrecheckRejectsInvalidConfidenceTimeAndMeasurementBeforePersistence(t *testing.T) {
	start := time.Date(2026, 7, 29, 10, 0, 0, 0, time.UTC)
	end := start.Add(-time.Hour)
	context := Context{
		Event:    Event{ID: "event", Status: "confirmed", FactStatus: "verified", OccurredAt: timePtr(start)},
		Evidence: []Evidence{{ID: "evidence"}},
		Entities: []Entity{
			{ID: "company", Type: "company", Name: "company", CanonicalName: "company", Status: "active"},
			{ID: "product", Type: "product", Name: "product", CanonicalName: "product", Status: "active"},
		},
		Variables: []VariableDefinition{
			{Key: "production_volume", Version: 1, Status: "active", AllowedDirections: []string{"decrease"}, ApplicableEntityTypes: []string{"company"}},
			{Key: "market_supply", Version: 1, Status: "active", AllowedDirections: []string{"decrease"}, ApplicableEntityTypes: []string{"product"}},
		},
	}
	result := Precheck(context, Submission{
		EntityLinks: []EntityLinkCandidate{
			{Key: "company", Mention: "company", EntityID: "company", EntityRole: "actor", EvidenceIDs: []string{"evidence"}, ResolutionMethod: "data_service_resolution"},
			{Key: "bad-confidence", Mention: "product", EntityID: "product", EntityRole: "event_object", EvidenceIDs: []string{"evidence"}, ResolutionMethod: "data_service_resolution", ResolutionConfidence: "1.1"},
		},
		VariableSignals: []VariableSignalCandidate{
			{Key: "bad-time", SubjectLinkKey: "company", VariableKey: "production_volume", VariableVersion: 1, Direction: "decrease", AssertionModality: "actual", EvidenceIDs: []string{"evidence"}, ValidFrom: &start, ValidUntil: &end},
			{Key: "bad-shape", SubjectLinkKey: "company", VariableKey: "production_volume", VariableVersion: 1, Direction: "decrease", AssertionModality: "actual", EvidenceIDs: []string{"evidence"}, Measurements: []MeasurementValue{{Role: "absolute_change", Shape: "range", RawLower: stringPtr("10"), RawText: "10%", EvidenceID: "evidence"}}},
			{Key: "bad-decimal", SubjectLinkKey: "company", VariableKey: "production_volume", VariableVersion: 1, Direction: "decrease", AssertionModality: "actual", EvidenceIDs: []string{"evidence"}, Measurements: []MeasurementValue{{Role: "relative_change", Shape: "exact", RawValue: stringPtr("not-a-number"), CanonicalValue: stringPtr("10"), CanonicalUnit: "percent", RawText: "下降10%", EvidenceID: "evidence"}}},
			{Key: "bad-confidence", SubjectLinkKey: "company", VariableKey: "production_volume", VariableVersion: 1, Direction: "decrease", AssertionModality: "actual", EvidenceIDs: []string{"evidence"}, ExtractionConfidence: "-0.1"},
		},
	})

	assertCandidateStatus(t, result.EntityLinks, "bad-confidence", StatusRejected, "confidence_invalid")
	assertCandidateStatus(t, result.VariableSignals, "bad-time", StatusRejected, "signal_time_invalid")
	assertCandidateStatus(t, result.VariableSignals, "bad-shape", StatusRejected, "measurement_value_invalid")
	assertCandidateStatus(t, result.VariableSignals, "bad-decimal", StatusRejected, "measurement_value_invalid")
	assertCandidateStatus(t, result.VariableSignals, "bad-confidence", StatusRejected, "confidence_invalid")
}

func TestSubmissionMetadataRejectsDuplicateCandidateKeys(t *testing.T) {
	hash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	err := validateSubmissionMetadata(Submission{
		GeneratorPromptHash: hash, ReviewerPromptHash: hash,
		GeneratorModel: "model", ReviewerModel: "model",
		OntologyVersion: "ontology@1", AcceptancePolicyVersion: "policy@1",
		EntityLinks: []EntityLinkCandidate{{Key: "duplicate"}, {Key: "duplicate"}},
	})
	if err == nil {
		t.Fatal("expected duplicate candidate keys to be rejected")
	}
}

func TestPrecheckRejectsMissingModalityPeriodsAndMeasurementDirectionConflict(t *testing.T) {
	now := time.Date(2026, 7, 29, 10, 0, 0, 0, time.UTC)
	context := Context{
		Event:    Event{ID: "event", Status: "confirmed", FactStatus: "verified", OccurredAt: &now},
		Evidence: []Evidence{{ID: "evidence"}},
		Entities: []Entity{{
			ID: "company", Type: "company", Name: "company", CanonicalName: "company", Status: "active",
		}},
		Variables: []VariableDefinition{{
			Key: "revenue", Version: 1, Status: "active",
			AllowedDirections: []string{"increase"}, ApplicableEntityTypes: []string{"company"},
		}},
	}
	result := Precheck(context, Submission{
		EntityLinks: []EntityLinkCandidate{{
			Key: "company", Mention: "company", EntityID: "company", EntityRole: "actor",
			EvidenceIDs: []string{"evidence"}, ResolutionMethod: "data_service_resolution",
		}},
		VariableSignals: []VariableSignalCandidate{
			{
				Key: "intent", SubjectLinkKey: "company", VariableKey: "revenue", VariableVersion: 1,
				Direction: "increase", AssertionModality: "stated_intent", StatementAt: &now,
				EvidenceIDs: []string{"evidence"},
			},
			{
				Key: "forecast", SubjectLinkKey: "company", VariableKey: "revenue", VariableVersion: 1,
				Direction: "increase", AssertionModality: "source_forecast", StatementAt: &now,
				EvidenceIDs: []string{"evidence"},
			},
			{
				Key: "conflict", SubjectLinkKey: "company", VariableKey: "revenue", VariableVersion: 1,
				Direction: "increase", AssertionModality: "actual", EvidenceIDs: []string{"evidence"},
				Measurements: []MeasurementValue{{
					Role: "relative_change", Shape: "exact", RawValue: stringPtr("-10.00"),
					CanonicalValue: stringPtr("-10.00"), RawUnit: "%", CanonicalUnit: "percent",
					RawText: "下降10.00%", EvidenceID: "evidence",
				}},
			},
		},
	})
	assertCandidateStatus(t, result.VariableSignals, "intent", StatusRejected, "effective_period_missing")
	assertCandidateStatus(t, result.VariableSignals, "forecast", StatusRejected, "forecast_period_missing")
	assertCandidateStatus(t, result.VariableSignals, "conflict", StatusRejected, "measurement_direction_conflict")
}

func TestPrecheckRequiresTransmissionRuleVariableVersions(t *testing.T) {
	context := Context{
		Event:    Event{ID: "event", Status: "confirmed", FactStatus: "verified", OccurredAt: timePtr(time.Now())},
		Evidence: []Evidence{{ID: "evidence"}},
		Entities: []Entity{
			{ID: "company", Type: "company", Name: "company", CanonicalName: "company", Status: "active"},
			{ID: "product", Type: "product", Name: "product", CanonicalName: "product", Status: "active"},
		},
		Variables: []VariableDefinition{
			{Key: "production_volume", Version: 2, Status: "active", AllowedDirections: []string{"decrease"}, ApplicableEntityTypes: []string{"company"}},
			{Key: "market_supply", Version: 2, Status: "active", AllowedDirections: []string{"decrease"}, ApplicableEntityTypes: []string{"product"}},
		},
		Relations: []EntityRelation{{ID: "relation", FromEntityID: "company", ToEntityID: "product", Type: "produces", Status: "active"}},
		Rules: []DirectTransmissionRule{{
			Key: "rule", Version: 1, Status: "approved", SourceEntityType: "company",
			SourceVariableKey: "production_volume", SourceVariableVersion: 1,
			SourceDirection: "decrease", RelationType: "produces", TargetEntityType: "product",
			AffectedVariableKey: "market_supply", AffectedVariableVersion: 1, AffectedDirection: "decrease",
		}},
	}
	result := Precheck(context, Submission{
		EntityLinks:     []EntityLinkCandidate{{Key: "company", Mention: "company", EntityID: "company", EntityRole: "actor", EvidenceIDs: []string{"evidence"}, ResolutionMethod: "data_service_resolution"}},
		VariableSignals: []VariableSignalCandidate{{Key: "signal", SubjectLinkKey: "company", VariableKey: "production_volume", VariableVersion: 2, Direction: "decrease", AssertionModality: "actual", EvidenceIDs: []string{"evidence"}}},
		DirectImpacts:   []DirectImpactCandidate{{Key: "impact", SourceSignalKey: "signal", TargetEntityID: "product", AffectedVariableKey: "market_supply", AffectedVariableVersion: 2, AffectedDirection: "decrease", DerivationType: "rule_inferred", EntityRelationID: "relation", RuleKey: "rule", RuleVersion: 1, EvidenceIDs: []string{"evidence"}}},
	})
	assertCandidateStatus(t, result.DirectImpacts, "impact", StatusRejected, "rule_not_matched")
}

func TestMeasurementDirectionMappingIsDeterministic(t *testing.T) {
	tests := []struct {
		name         string
		direction    string
		measurements []MeasurementValue
		conflict     bool
	}{
		{name: "evidence-only direction remains allowed", direction: "increase"},
		{
			name: "absolute level without comparison is uncertain", direction: "increase",
			measurements: []MeasurementValue{{Role: "absolute_level", CanonicalValue: stringPtr("10")}},
			conflict:     true,
		},
		{
			name: "absolute level accepts uncertain", direction: "uncertain",
			measurements: []MeasurementValue{{Role: "absolute_level", CanonicalValue: stringPtr("10")}},
		},
		{
			name: "zero change must be unchanged", direction: "increase",
			measurements: []MeasurementValue{{Role: "relative_change", CanonicalValue: stringPtr("0")}},
			conflict:     true,
		},
		{
			name: "positive change cannot be uncertain", direction: "uncertain",
			measurements: []MeasurementValue{{Role: "relative_change", CanonicalValue: stringPtr("10")}},
			conflict:     true,
		},
		{
			name: "range crossing zero is uncertain", direction: "uncertain",
			measurements: []MeasurementValue{{
				Role: "relative_change", CanonicalLower: stringPtr("-1"), CanonicalUpper: stringPtr("1"),
			}},
		},
		{
			name: "opposite periods can be mixed", direction: "mixed",
			measurements: []MeasurementValue{
				{Role: "relative_change", CanonicalValue: stringPtr("10"), ComparisonPeriod: "2026Q1"},
				{Role: "relative_change", CanonicalValue: stringPtr("-5"), ComparisonPeriod: "2026Q2"},
			},
		},
		{
			name: "opposite values without distinct context are uncertain", direction: "mixed",
			measurements: []MeasurementValue{
				{Role: "relative_change", CanonicalValue: stringPtr("10")},
				{Role: "relative_change", CanonicalValue: stringPtr("-5")},
			},
			conflict: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := measurementDirectionConflicts(test.direction, test.measurements); got != test.conflict {
				t.Fatalf("measurementDirectionConflicts() = %v, want %v", got, test.conflict)
			}
		})
	}
}

func assertCandidateStatus(t *testing.T, items []CandidateDecision, key string, status ReviewStatus, reason string) {
	t.Helper()
	for _, item := range items {
		if item.CandidateKey == key {
			if item.Status != status || item.ReasonCode != reason {
				t.Fatalf("%s decision = %#v, want status %q reason %q", key, item, status, reason)
			}
			return
		}
	}
	t.Fatalf("candidate %q not found in %#v", key, items)
}

func timePtr(value time.Time) *time.Time { return &value }
func stringPtr(value string) *string     { return &value }
