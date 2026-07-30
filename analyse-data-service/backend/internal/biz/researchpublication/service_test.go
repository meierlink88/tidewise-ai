package researchpublication

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchreasoningtreeimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
)

const (
	testEventID              = "11111111-1111-4111-8111-111111111111"
	testChainID              = "22222222-2222-4222-8222-222222222222"
	testNodeID               = "33333333-3333-4333-8333-333333333333"
	testSignalID             = "44444444-4444-4444-8444-444444444444"
	testSubmissionID         = "55555555-5555-4555-8555-555555555555"
	testEvidenceID           = "66666666-6666-4666-8666-666666666666"
	testTargetNodeID         = "77777777-7777-4777-8777-777777777777"
	testImpactID             = "88888888-8888-4888-8888-888888888888"
	testImpactEventID        = "99999999-9999-4999-8999-999999999999"
	testImpactEvidenceID     = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	testTargetSignalID       = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	testImpactSourceSignalID = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
)

func TestAggregateRequiresFormalLineageAndOneAtomicTheme(t *testing.T) {
	aggregate := validAggregate()
	if _, _, err := aggregate.Validate(); err != nil {
		t.Fatalf("valid aggregate: %v", err)
	}
	payload, err := json.Marshal(aggregate)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), `"nodes"`) || !strings.Contains(string(payload), `"formal_signal"`) {
		t.Fatalf("canonical payload lost aggregate lineage: %s", payload)
	}

	aggregate.ReasoningTrees[0].Nodes[0].Signals[0].Lineage.EvidenceHash = nil
	if _, _, err := aggregate.Validate(); err == nil {
		t.Fatal("formal Signal without Evidence hash was accepted")
	}
}

func TestAggregateRejectsAnalystInferenceMasqueradingAsFormalFact(t *testing.T) {
	aggregate := validAggregate()
	lineage := &aggregate.ReasoningTrees[0].Nodes[0].Signals[0].Lineage
	lineage.SourceKind = "analyst_inference"
	upstream := testSignalID
	relation := "77777777-7777-4777-8777-777777777777"
	lineage.UpstreamVariableSignalID = &upstream
	lineage.EntityRelationID = &relation
	if _, _, err := aggregate.Validate(); err == nil {
		t.Fatal("analyst inference carrying formal Signal/Evidence claims was accepted")
	}
}

func TestAggregateRejectsUntrimmedPrimarySignalDisplaySummary(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "leading whitespace", value: " Supply decreases"},
		{
			name:  "trimmed value at limit but original over limit",
			value: " " + strings.Repeat("x", 200),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			aggregate := validAggregate()
			aggregate.Theme.Impacts[0].PrimarySignalDisplaySummary = test.value

			_, _, err := aggregate.Validate()
			var validation *researchthemeimport.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("Validate() error = %T %v, want Theme ValidationError", err, err)
			}
			if validation.Path != "themes[0].impacts[0].primary_signal_display_summary" {
				t.Fatalf("validation path = %q", validation.Path)
			}
		})
	}
}

func TestAggregatePreservesExistingThemeRequiredTextWhitespaceContract(t *testing.T) {
	aggregate := validAggregate()
	aggregate.Theme.Title = " Wafer supply "

	if _, _, err := aggregate.Validate(); err != nil {
		t.Fatalf("Validate() rejected an existing Theme required-text value: %v", err)
	}
}

func TestPublishPersistsThemeImpactPrimarySignalDisplaySummary(t *testing.T) {
	aggregate := validAggregate()
	tx := &fakeTransaction{facts: validFacts()}
	service := NewService(fakeStore{tx: tx})

	if _, err := service.Publish(context.Background(), "codex", aggregate); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if len(tx.impacts) != 1 {
		t.Fatalf("persisted impacts = %d, want 1", len(tx.impacts))
	}
	if got, want := tx.impacts[0].PrimarySignalDisplaySummary, aggregate.Theme.Impacts[0].PrimarySignalDisplaySummary; got != want {
		t.Fatalf("primary signal display summary = %q, want %q", got, want)
	}
}

func TestPublishRejectsReferenceMismatchBeforeAnyWrite(t *testing.T) {
	aggregate := validAggregate()
	tx := &fakeTransaction{facts: validFacts()}
	signal := tx.facts.Signals[testSignalID]
	signal.SubjectEntityID = "77777777-7777-4777-8777-777777777777"
	tx.facts.Signals[testSignalID] = signal
	service := NewService(fakeStore{tx: tx})

	_, err := service.Publish(context.Background(), "codex", aggregate)
	var reference *ReferenceError
	if !errors.As(err, &reference) {
		t.Fatalf("error = %T %v, want ReferenceError", err, err)
	}
	if tx.writes != 0 {
		t.Fatalf("writes = %d, want no writes before reference validation passes", tx.writes)
	}
}

func TestPublishRejectsEventOutsideDeclaredDiscoveryWindow(t *testing.T) {
	aggregate := validAggregate()
	tx := &fakeTransaction{facts: validFacts()}
	event := tx.facts.Events[testEventID]
	event.KnowledgeAvailableAt = time.Date(2026, 7, 29, 11, 0, 0, 0, time.UTC)
	tx.facts.Events[testEventID] = event
	service := NewService(fakeStore{tx: tx})

	_, err := service.Publish(context.Background(), "codex", aggregate)
	var reference *ReferenceError
	if !errors.As(err, &reference) || reference.Reference != testEventID {
		t.Fatalf("error = %T %v, want Event ReferenceError", err, err)
	}
	if tx.writes != 0 {
		t.Fatalf("writes = %d, want no writes for future Event", tx.writes)
	}
}

func TestPublishRejectsStructuralFactCreatedAfterAnalysisAsOf(t *testing.T) {
	aggregate := validAggregate()
	tx := &fakeTransaction{facts: validFacts()}
	future := time.Date(2026, 7, 29, 11, 0, 0, 0, time.UTC)
	tx.facts.Memberships[testChainID][testNodeID] = TemporalFact{
		CreatedAt: future,
		UpdatedAt: future,
	}
	service := NewService(fakeStore{tx: tx})

	_, err := service.Publish(context.Background(), "codex", aggregate)
	var reference *ReferenceError
	if !errors.As(err, &reference) || reference.Reference != testNodeID {
		t.Fatalf("error = %T %v, want future Membership ReferenceError", err, err)
	}
	if tx.writes != 0 {
		t.Fatalf("writes = %d, want no writes for future structural fact", tx.writes)
	}
}

func TestPublishRejectsInferenceWithUnrelatedRelation(t *testing.T) {
	aggregate := validAggregate()
	lineage := &aggregate.ReasoningTrees[0].Nodes[0].Signals[0].Lineage
	lineage.SourceKind = "analyst_inference"
	lineage.VariableSignalID = nil
	lineage.SemanticSubmissionID = nil
	lineage.EvidenceID = nil
	lineage.EvidenceHash = nil
	lineage.UpstreamVariableSignalID = stringPointer(testSignalID)
	relationID := "77777777-7777-4777-8777-777777777777"
	lineage.EntityRelationID = &relationID
	tx := &fakeTransaction{facts: validFacts()}
	tx.facts.EntityRelations[relationID] = EntityRelationFact{
		ID: relationID, FromEntityID: "88888888-8888-4888-8888-888888888888",
		ToEntityID:   testNodeID,
		TemporalFact: testTemporalFact(),
	}
	service := NewService(fakeStore{tx: tx})

	_, err := service.Publish(context.Background(), "codex", aggregate)
	var reference *ReferenceError
	if !errors.As(err, &reference) || reference.Reference != relationID {
		t.Fatalf("error = %T %v, want relation ReferenceError", err, err)
	}
	if tx.writes != 0 {
		t.Fatalf("writes = %d, want no writes for unrelated relation", tx.writes)
	}
}

func TestPublishRejectsDirectImpactWhoseSourceIsNotThePreviousNode(t *testing.T) {
	aggregate := validAggregateWithDirectImpact()
	tx := &fakeTransaction{facts: validFactsWithDirectImpact()}
	impact := tx.facts.Impacts[testImpactID]
	impact.SourceEntityID = "dddddddd-dddd-4ddd-8ddd-dddddddddddd"
	tx.facts.Impacts[testImpactID] = impact
	service := NewService(fakeStore{tx: tx})

	_, err := service.Publish(context.Background(), "codex", aggregate)
	var reference *ReferenceError
	if !errors.As(err, &reference) || reference.Reference != testImpactID {
		t.Fatalf("error = %T %v, want Direct Impact ReferenceError", err, err)
	}
	if tx.writes != 0 {
		t.Fatalf("writes = %d, want no writes for Direct Impact from an unrelated upstream Node", tx.writes)
	}
}

func TestPublishRequiresEachTreeToCoverItsFormalFactEvents(t *testing.T) {
	tests := []struct {
		name      string
		aggregate Aggregate
		facts     ReferenceFacts
		reference string
	}{
		{
			name: "formal Signal",
			aggregate: func() Aggregate {
				aggregate := validAggregate()
				aggregate.ReasoningTrees[0].Events = nil
				return aggregate
			}(),
			facts:     validFacts(),
			reference: testSignalID,
		},
		{
			name: "formal Direct Impact",
			aggregate: func() Aggregate {
				aggregate := validAggregateWithDirectImpact()
				aggregate.ReasoningTrees[0].Events = aggregate.ReasoningTrees[0].Events[:1]
				return aggregate
			}(),
			facts:     validFactsWithDirectImpact(),
			reference: testImpactID,
		},
		{
			name: "analyst inference from upstream Signal",
			aggregate: func() Aggregate {
				aggregate := validAggregate()
				aggregate.ReasoningTrees[0].Events = nil
				lineage := &aggregate.ReasoningTrees[0].Nodes[0].Signals[0].Lineage
				lineage.SourceKind = "analyst_inference"
				lineage.VariableSignalID = nil
				lineage.SemanticSubmissionID = nil
				lineage.EvidenceID = nil
				lineage.EvidenceHash = nil
				lineage.UpstreamVariableSignalID = stringPointer(testSignalID)
				lineage.EntityRelationID = stringPointer("eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee")
				return aggregate
			}(),
			facts: func() ReferenceFacts {
				facts := validFacts()
				signal := facts.Signals[testSignalID]
				signal.SubjectEntityID = "ffffffff-ffff-4fff-8fff-ffffffffffff"
				facts.Signals[testSignalID] = signal
				facts.EntityRelations["eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee"] = EntityRelationFact{
					ID:           "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee",
					FromEntityID: "ffffffff-ffff-4fff-8fff-ffffffffffff",
					ToEntityID:   testNodeID,
					TemporalFact: testTemporalFact(),
				}
				return facts
			}(),
			reference: testSignalID,
		},
		{
			name: "analyst inference from upstream Direct Impact",
			aggregate: func() Aggregate {
				aggregate := validAggregate()
				aggregate.Theme.Events = append(aggregate.Theme.Events, researchthemeimport.Event{
					EventID: testImpactEventID, EvidenceRole: "supporting",
				})
				aggregate.ReasoningTrees[0].Events = nil
				lineage := &aggregate.ReasoningTrees[0].Nodes[0].Signals[0].Lineage
				lineage.SourceKind = "analyst_inference"
				lineage.VariableSignalID = nil
				lineage.SemanticSubmissionID = nil
				lineage.EvidenceID = nil
				lineage.EvidenceHash = nil
				lineage.UpstreamDirectImpactAssertionID = stringPointer(testImpactID)
				lineage.EntityRelationID = stringPointer("eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee")
				return aggregate
			}(),
			facts: func() ReferenceFacts {
				facts := validFactsWithDirectImpact()
				facts.EntityRelations["eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee"] = EntityRelationFact{
					ID:           "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee",
					FromEntityID: testTargetNodeID,
					ToEntityID:   testNodeID,
					TemporalFact: testTemporalFact(),
				}
				return facts
			}(),
			reference: testImpactID,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tx := &fakeTransaction{facts: test.facts}
			service := NewService(fakeStore{tx: tx})

			_, err := service.Publish(context.Background(), "codex", test.aggregate)
			var reference *ReferenceError
			if !errors.As(err, &reference) || reference.Reference != test.reference {
				t.Fatalf("error = %T %v, want lineage ReferenceError for %s", err, err, test.reference)
			}
			if tx.writes != 0 {
				t.Fatalf("writes = %d, want no writes when Tree Event lineage is incomplete", tx.writes)
			}
		})
	}
}

func validAggregate() Aggregate {
	hash := strings.Repeat("a", 64)
	signalID, submissionID, evidenceID := testSignalID, testSubmissionID, testEvidenceID
	return Aggregate{
		AnalysisBatchID:      "theme-publication-1",
		AnalysisAsOf:         "2026-07-29T10:00:00Z",
		DiscoveryWindowStart: "2026-07-28T00:00:00Z",
		DiscoveryWindowEnd:   "2026-07-29T10:00:00Z",
		Theme: researchthemeimport.Theme{
			ThemeKey: "wafer-supply", Title: "Wafer supply", OneLineConclusion: "Supply tightens",
			ConclusionDirection: "positive", ImpactStrength: "medium",
			TransmissionStage: "validation", InvestmentGuidanceAction: "focus",
			InvestmentGuidanceSummary: "Watch supply", TimeHorizonCategory: "short_term",
			Impacts: []researchthemeimport.Impact{{
				ChainNodeEntityID: testNodeID, RelationRole: "driver",
				ImpactDirection: "positive", PrimarySignalDisplaySummary: "Supply decreases",
				DisplayOrder: 1,
			}},
			Events: []researchthemeimport.Event{{EventID: testEventID, EvidenceRole: "driver"}},
		},
		ReasoningTrees: []ReasoningTree{{
			ReasoningTree: researchreasoningtreeimport.ReasoningTree{
				IndustryChainEntityID: testChainID, Title: "Wafer chain", DisplayOrder: 1,
				OneLineConclusion: "Supply tightens", ImpactDirection: "positive",
				ImpactStrength: "medium",
				Events: []researchreasoningtreeimport.Event{{
					EventID: testEventID, EvidenceRole: "driver", DisplayOrder: 1,
				}},
			},
			Nodes: []Node{{
				Position: 1, ChainNodeEntityID: testNodeID,
				ImpactDirection: "positive", ImpactStrength: "medium",
				Signals: []Signal{{
					VariableSignalKey: "market_supply", SignalRole: "primary",
					SignalDirection: "decrease", DisplaySummary: "Supply decreases", DisplayOrder: 1,
					Lineage: SignalLineage{
						SourceKind: "formal_signal", VariableSignalID: &signalID,
						SemanticSubmissionID: &submissionID, EvidenceID: &evidenceID,
						EvidenceHash: &hash,
					},
				}},
			}},
		}},
	}
}

func validAggregateWithDirectImpact() Aggregate {
	aggregate := validAggregate()
	impactHash := strings.Repeat("b", 64)
	targetSignalID := testTargetSignalID
	submissionID := testSubmissionID
	evidenceID := testEvidenceID
	impactID := testImpactID
	impactEvidenceID := testImpactEvidenceID
	affectedVariableKey := "market_supply"
	affectedDirection := "decrease"
	incomingTitle := "Producer output to product supply"
	incomingMechanism := "Lower producer output reduces product supply"
	incomingCondition := "The producer-to-product relation remains active"
	aggregate.Theme.Events = append(aggregate.Theme.Events, researchthemeimport.Event{
		EventID: testImpactEventID, EvidenceRole: "supporting",
	})
	aggregate.ReasoningTrees[0].Events = append(
		aggregate.ReasoningTrees[0].Events,
		researchreasoningtreeimport.Event{
			EventID: testImpactEventID, EvidenceRole: "supporting", DisplayOrder: 2,
		},
	)
	aggregate.ReasoningTrees[0].Nodes = append(
		aggregate.ReasoningTrees[0].Nodes,
		Node{
			Position:                      2,
			ChainNodeEntityID:             testTargetNodeID,
			ImpactDirection:               "negative",
			ImpactStrength:                "medium",
			IncomingTransmissionTitle:     &incomingTitle,
			IncomingTransmissionMechanism: &incomingMechanism,
			IncomingConditionSummary:      &incomingCondition,
			IncomingLineage: &IncomingLineage{
				SourceKind:              "formal_direct_impact",
				DirectImpactAssertionID: &impactID,
				SemanticSubmissionID:    &submissionID,
				EvidenceID:              &impactEvidenceID,
				EvidenceHash:            &impactHash,
				AffectedVariableKey:     &affectedVariableKey,
				AffectedDirection:       &affectedDirection,
			},
			Signals: []Signal{{
				VariableSignalKey: "market_price",
				SignalRole:        "primary",
				SignalDirection:   "increase",
				DisplaySummary:    "Product price increases",
				DisplayOrder:      1,
				Lineage: SignalLineage{
					SourceKind:           "formal_signal",
					VariableSignalID:     &targetSignalID,
					SemanticSubmissionID: &submissionID,
					EvidenceID:           &evidenceID,
					EvidenceHash:         stringPointer(strings.Repeat("a", 64)),
				},
			}},
		},
	)
	return aggregate
}

func validFacts() ReferenceFacts {
	accepted := time.Date(2026, 7, 29, 9, 0, 0, 0, time.UTC)
	return ReferenceFacts{
		ChainNodeIDs: map[string]TemporalFact{testNodeID: testTemporalFact()},
		Events: map[string]EventFact{
			testEventID: {
				ID:                   testEventID,
				KnowledgeAvailableAt: time.Date(2026, 7, 29, 8, 0, 0, 0, time.UTC),
			},
		},
		IndustryChainIDs: map[string]TemporalFact{testChainID: testTemporalFact()},
		Memberships: map[string]map[string]TemporalFact{
			testChainID: {testNodeID: testTemporalFact()},
		},
		GraphEdges: map[string]GraphEdgeFact{},
		Signals: map[string]SignalFact{
			testSignalID: {
				ID: testSignalID, SemanticSubmissionID: testSubmissionID,
				EventID: testEventID, SubjectEntityID: testNodeID,
				VariableKey: "market_supply", Direction: "decrease",
				EvidenceIDs: map[string]struct{}{testEvidenceID: {}}, AcceptedAt: accepted,
			},
		},
		Impacts: map[string]ImpactFact{},
		Evidences: map[string]EvidenceFact{
			testEvidenceID: {
				ID: testEvidenceID, EventID: testEventID, Hash: strings.Repeat("a", 64),
				KnowledgeAvailableAt: accepted,
			},
		},
		EntityRelations: map[string]EntityRelationFact{},
	}
}

func validFactsWithDirectImpact() ReferenceFacts {
	facts := validFacts()
	accepted := time.Date(2026, 7, 29, 9, 0, 0, 0, time.UTC)
	facts.ChainNodeIDs[testTargetNodeID] = testTemporalFact()
	facts.Memberships[testChainID][testTargetNodeID] = testTemporalFact()
	facts.Events[testImpactEventID] = EventFact{
		ID:                   testImpactEventID,
		KnowledgeAvailableAt: time.Date(2026, 7, 29, 8, 30, 0, 0, time.UTC),
	}
	facts.Signals[testTargetSignalID] = SignalFact{
		ID: testTargetSignalID, SemanticSubmissionID: testSubmissionID,
		EventID: testEventID, SubjectEntityID: testTargetNodeID,
		VariableKey: "market_price", Direction: "increase",
		EvidenceIDs: map[string]struct{}{testEvidenceID: {}}, AcceptedAt: accepted,
	}
	facts.Impacts[testImpactID] = ImpactFact{
		ID: testImpactID, SemanticSubmissionID: testSubmissionID,
		SourceVariableSignalID: testImpactSourceSignalID,
		TargetEntityID:         testTargetNodeID,
		AffectedVariableKey:    "market_supply",
		AffectedDirection:      "decrease",
		SourceEventID:          testImpactEventID,
		SourceEntityID:         testNodeID,
		EvidenceIDs:            map[string]struct{}{testImpactEvidenceID: {}},
		AcceptedAt:             accepted,
	}
	facts.Evidences[testImpactEvidenceID] = EvidenceFact{
		ID: testImpactEvidenceID, EventID: testImpactEventID, Hash: strings.Repeat("b", 64),
		KnowledgeAvailableAt: accepted,
	}
	return facts
}

func testTemporalFact() TemporalFact {
	value := time.Date(2026, 7, 29, 7, 0, 0, 0, time.UTC)
	return TemporalFact{CreatedAt: value, UpdatedAt: value}
}

type fakeStore struct{ tx *fakeTransaction }

func (s fakeStore) InResearchPublicationTransaction(ctx context.Context, fn func(Transaction) error) error {
	return fn(s.tx)
}

type fakeTransaction struct {
	facts   ReferenceFacts
	writes  int
	impacts []researchthemeimport.ImpactRecord
}

func (*fakeTransaction) Lock(context.Context, string) error                { return nil }
func (*fakeTransaction) Receipt(context.Context, string) (*Receipt, error) { return nil, nil }
func (f *fakeTransaction) ReferenceFacts(context.Context, ReferenceQuery) (ReferenceFacts, error) {
	return f.facts, nil
}
func (f *fakeTransaction) InsertThemeReceipt(context.Context, Receipt) error { f.writes++; return nil }
func (f *fakeTransaction) InsertTheme(context.Context, researchthemeimport.ThemeRecord) error {
	f.writes++
	return nil
}
func (f *fakeTransaction) InsertThemeImpact(_ context.Context, value researchthemeimport.ImpactRecord) error {
	f.writes++
	f.impacts = append(f.impacts, value)
	return nil
}
func (f *fakeTransaction) InsertThemeEvent(context.Context, researchthemeimport.EventRecord) error {
	f.writes++
	return nil
}
func (f *fakeTransaction) InsertTreeReceipt(context.Context, researchreasoningtreeimport.Receipt) error {
	f.writes++
	return nil
}
func (f *fakeTransaction) InsertTree(context.Context, researchreasoningtreeimport.ReasoningTreeRecord) error {
	f.writes++
	return nil
}
func (f *fakeTransaction) InsertTreeEvent(context.Context, researchreasoningtreeimport.EventRecord) error {
	f.writes++
	return nil
}
func (f *fakeTransaction) InsertNode(context.Context, NodeRecord) error     { f.writes++; return nil }
func (f *fakeTransaction) InsertSignal(context.Context, SignalRecord) error { f.writes++; return nil }
func (*fakeTransaction) Verify(context.Context, Receipt) error              { return nil }

func stringPointer(value string) *string { return &value }
