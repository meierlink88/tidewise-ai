package research

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	entitybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity"
	eventbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/event"
	eventsemanticbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/eventsemantic"
)

const (
	testEventID              = "EVT11111111-1111-4111-8111-111111111111"
	testChainID              = "ICH22222222-2222-4222-8222-222222222222"
	testNodeID               = "CND33333333-3333-4333-8333-333333333333"
	testSignalID             = "VSG44444444-4444-4444-8444-444444444444"
	testSubmissionID         = "ESS55555555-5555-4555-8555-555555555555"
	testEvidenceID           = "EEL66666666-6666-4666-8666-666666666666"
	testTargetNodeID         = "CND77777777-7777-4777-8777-777777777777"
	testImpactID             = "DIA88888888-8888-4888-8888-888888888888"
	testImpactEventID        = "EVT99999999-9999-4999-8999-999999999999"
	testImpactEvidenceID     = "EELaaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	testTargetSignalID       = "VSGbbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	testImpactSourceSignalID = "VSGcccccccc-cccc-4ccc-8ccc-cccccccccccc"
	testBCIChainID           = "ICH822a8ddc-5ebc-5f03-8ef8-ba9bfba192d9"
	testBCISystemNodeID      = "CNDc38d2f7b-9900-5e81-af06-76393bcc2617"
	testBCITerminalNodeID    = "CND96336148-76c0-504e-b82e-ac395f8fe268"
	testBCIElectrodeNodeID   = "CNDd3882237-d639-5660-b7d8-aa3563706113"
	testBCITerminalEdgeID    = "IGE300188b0-d01c-5987-ad8a-646067edc7cd"
	testBCIElectrodeEdgeID   = "IGEdc00a16e-0d8e-5db9-9a5d-fbc1fd9a84cf"
)

func TestPublicationAggregateRequiresFormalLineageAndOneAtomicTheme(t *testing.T) {
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

func TestPublicationAggregateRejectsAnalystInferenceMasqueradingAsFormalFact(t *testing.T) {
	aggregate := validAggregate()
	lineage := &aggregate.ReasoningTrees[0].Nodes[0].Signals[0].Lineage
	lineage.SourceKind = "analyst_inference"
	upstream := testSignalID
	relation := "ERL77777777-7777-4777-8777-777777777777"
	lineage.UpstreamVariableSignalID = &upstream
	lineage.EntityRelationID = &relation
	if _, _, err := aggregate.Validate(); err == nil {
		t.Fatal("analyst inference carrying formal Signal/Evidence claims was accepted")
	}
}

func TestPublishRejectsReferenceMismatchBeforeAnyWrite(t *testing.T) {
	aggregate := validAggregate()
	tx := &publicationTransactionStub{facts: validFacts()}
	signal := tx.facts.Signals[testSignalID]
	signal.SubjectEntityID = "CND77777777-7777-4777-8777-777777777777"
	tx.facts.Signals[testSignalID] = signal
	service := (&UseCase{publicationStore: publicationStoreStub{tx: tx}, now: time.Now})

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
	tx := &publicationTransactionStub{facts: validFacts()}
	event := tx.facts.Events[testEventID]
	event.KnowledgeAvailableAt = time.Date(2026, 7, 29, 11, 0, 0, 0, time.UTC)
	tx.facts.Events[testEventID] = event
	service := (&UseCase{publicationStore: publicationStoreStub{tx: tx}, now: time.Now})

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
	tx := &publicationTransactionStub{facts: validFacts()}
	future := time.Date(2026, 7, 29, 11, 0, 0, 0, time.UTC)
	tx.facts.Memberships[testChainID][testNodeID] = TemporalFact{
		CreatedAt: future,
		UpdatedAt: future,
	}
	service := (&UseCase{publicationStore: publicationStoreStub{tx: tx}, now: time.Now})

	_, err := service.Publish(context.Background(), "codex", aggregate)
	var reference *ReferenceError
	if !errors.As(err, &reference) || reference.Reference != testNodeID {
		t.Fatalf("error = %T %v, want future Membership ReferenceError", err, err)
	}
	if tx.writes != 0 {
		t.Fatalf("writes = %d, want no writes for future structural fact", tx.writes)
	}
}

func TestPublishAcceptsReverseMultiHopAnalystInference(t *testing.T) {
	aggregate, facts := validBCIAnalystInferenceAggregate(true, 3)
	tx := &publicationTransactionStub{facts: facts}
	service := (&UseCase{publicationStore: publicationStoreStub{tx: tx}, now: time.Now})

	result, err := service.Publish(context.Background(), "codex", aggregate)
	if err != nil {
		t.Fatalf("publish reverse multi-hop analyst inference: %v", err)
	}
	if result.Counts.Nodes != 3 || result.Counts.SignalAssociations != 3 {
		t.Fatalf("publication counts = %#v, want three-node BCI Tree", result.Counts)
	}
	if tx.writes == 0 {
		t.Fatal("reverse multi-hop publication completed without atomic writes")
	}
}

func TestPublishKeepsForwardOneHopAnalystInference(t *testing.T) {
	aggregate, facts := validBCIAnalystInferenceAggregate(false, 2)
	tx := &publicationTransactionStub{facts: facts}
	service := (&UseCase{publicationStore: publicationStoreStub{tx: tx}, now: time.Now})

	result, err := service.Publish(context.Background(), "codex", aggregate)
	if err != nil {
		t.Fatalf("publish forward one-hop analyst inference: %v", err)
	}
	if result.Counts.Nodes != 2 {
		t.Fatalf("publication counts = %#v, want forward two-node Tree", result.Counts)
	}
}

func TestPublishRejectsExistingReceiptIdentityConflictsBeforeAnyWrite(t *testing.T) {
	aggregate, facts := validBCIAnalystInferenceAggregate(true, 3)
	_, themeID, err := aggregate.Validate()
	if err != nil {
		t.Fatalf("validate fixture: %v", err)
	}
	payloadHash, err := CanonicalHash(aggregate)
	if err != nil {
		t.Fatalf("hash fixture: %v", err)
	}
	receipt := publicationPlan(aggregate, themeID, payloadHash)
	receipt.PublisherSubject = "codex"

	tests := []struct {
		name      string
		publisher string
		aggregate Aggregate
		want      error
	}{
		{
			name:      "payload conflict",
			publisher: "codex",
			aggregate: func() Aggregate {
				changed := aggregate
				changed.Theme.Title = "Changed BCI demand"
				return changed
			}(),
			want: ErrPayloadConflict,
		},
		{
			name:      "publisher mismatch",
			publisher: "another-analyst",
			aggregate: aggregate,
			want:      ErrPublisherConflict,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tx := &publicationTransactionStub{facts: facts, receipt: &receipt}
			service := (&UseCase{publicationStore: publicationStoreStub{tx: tx}, now: time.Now})

			_, err := service.Publish(context.Background(), test.publisher, test.aggregate)
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %T %v, want %v", err, err, test.want)
			}
			if tx.writes != 0 {
				t.Fatalf("writes = %d, want no writes for receipt identity conflict", tx.writes)
			}
		})
	}
}

func TestPublishAcceptsReverseEntityRelationBetweenAdjacentNodes(t *testing.T) {
	aggregate, facts := validBCIAnalystInferenceAggregate(true, 2)
	relationID := "ERLeeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee"
	node := &aggregate.ReasoningTrees[0].Nodes[1]
	node.IncomingIndustryChainGraphEdgeID = nil
	node.IncomingLineage.EntityRelationID = &relationID
	node.Signals[0].Lineage.IndustryChainGraphEdgeID = nil
	node.Signals[0].Lineage.EntityRelationID = &relationID
	facts.EntityRelations[relationID] = EntityRelationFact{
		ID: relationID, FromEntityID: testBCITerminalNodeID, ToEntityID: testBCISystemNodeID,
		TemporalFact: testTemporalFact(),
	}
	tx := &publicationTransactionStub{facts: facts}
	service := (&UseCase{publicationStore: publicationStoreStub{tx: tx}, now: time.Now})

	if _, err := service.Publish(context.Background(), "codex", aggregate); err != nil {
		t.Fatalf("publish reverse adjacent Entity Relation inference: %v", err)
	}
}

func TestPublishRejectsAnalystInferenceRelationOutsideAdjacentNodes(t *testing.T) {
	aggregate, facts := validBCIAnalystInferenceAggregate(true, 3)
	edge := facts.GraphEdges[testBCIElectrodeEdgeID]
	edge.FromChainNodeID = testBCISystemNodeID
	facts.GraphEdges[testBCIElectrodeEdgeID] = edge
	tx := &publicationTransactionStub{facts: facts}
	service := (&UseCase{publicationStore: publicationStoreStub{tx: tx}, now: time.Now})

	_, err := service.Publish(context.Background(), "codex", aggregate)
	var reference *ReferenceError
	if !errors.As(err, &reference) || reference.Reference != testBCIElectrodeEdgeID {
		t.Fatalf("error = %T %v, want non-adjacent Graph Edge ReferenceError", err, err)
	}
	if tx.writes != 0 {
		t.Fatalf("writes = %d, want no writes for a non-adjacent relation", tx.writes)
	}
}

func TestPublishRejectsAnalystInferenceGraphEdgeFromAnotherIndustryChain(t *testing.T) {
	aggregate, facts := validBCIAnalystInferenceAggregate(true, 3)
	edge := facts.GraphEdges[testBCITerminalEdgeID]
	edge.IndustryChainID = testChainID
	facts.GraphEdges[testBCITerminalEdgeID] = edge
	tx := &publicationTransactionStub{facts: facts}
	service := (&UseCase{publicationStore: publicationStoreStub{tx: tx}, now: time.Now})

	_, err := service.Publish(context.Background(), "codex", aggregate)
	var reference *ReferenceError
	if !errors.As(err, &reference) || reference.Reference != testBCITerminalEdgeID {
		t.Fatalf("error = %T %v, want wrong-chain Graph Edge ReferenceError", err, err)
	}
	if tx.writes != 0 {
		t.Fatalf("writes = %d, want no writes for a wrong-chain Graph Edge", tx.writes)
	}
}

func TestPublishRejectsInactiveAnalystInferenceGraphEdge(t *testing.T) {
	aggregate, facts := validBCIAnalystInferenceAggregate(true, 3)
	edge := facts.GraphEdges[testBCITerminalEdgeID]
	edge.UpdatedAt = time.Date(2026, 7, 29, 11, 0, 0, 0, time.UTC)
	facts.GraphEdges[testBCITerminalEdgeID] = edge
	tx := &publicationTransactionStub{facts: facts}
	service := (&UseCase{publicationStore: publicationStoreStub{tx: tx}, now: time.Now})

	_, err := service.Publish(context.Background(), "codex", aggregate)
	var reference *ReferenceError
	if !errors.As(err, &reference) || reference.Reference != testBCITerminalEdgeID {
		t.Fatalf("error = %T %v, want inactive Graph Edge ReferenceError", err, err)
	}
	if tx.writes != 0 {
		t.Fatalf("writes = %d, want no writes for an inactive Graph Edge", tx.writes)
	}
}

func TestPublishRejectsInvalidBCIIndustryChainMembership(t *testing.T) {
	tests := []struct {
		name  string
		setup func(ReferenceFacts)
	}{
		{
			name: "missing inactive or unapproved membership is absent from active approved facts",
			setup: func(facts ReferenceFacts) {
				delete(facts.Memberships[testBCIChainID], testBCITerminalNodeID)
			},
		},
		{
			name: "membership belongs to another Industry Chain",
			setup: func(facts ReferenceFacts) {
				delete(facts.Memberships[testBCIChainID], testBCITerminalNodeID)
				facts.Memberships[testChainID] = map[string]TemporalFact{
					testBCITerminalNodeID: testTemporalFact(),
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			aggregate, facts := validBCIAnalystInferenceAggregate(true, 3)
			test.setup(facts)
			tx := &publicationTransactionStub{facts: facts}
			service := (&UseCase{publicationStore: publicationStoreStub{tx: tx}, now: time.Now})

			_, err := service.Publish(context.Background(), "codex", aggregate)
			var reference *ReferenceError
			if !errors.As(err, &reference) || reference.Reference != testBCITerminalNodeID {
				t.Fatalf("error = %T %v, want invalid BCI membership ReferenceError", err, err)
			}
			if tx.writes != 0 {
				t.Fatalf("writes = %d, want no writes for invalid BCI membership", tx.writes)
			}
		})
	}
}

func TestPublishRejectsMissingAcceptedRootSignal(t *testing.T) {
	aggregate, facts := validBCIAnalystInferenceAggregate(true, 3)
	delete(facts.Signals, testSignalID)
	tx := &publicationTransactionStub{facts: facts}
	service := (&UseCase{publicationStore: publicationStoreStub{tx: tx}, now: time.Now})

	_, err := service.Publish(context.Background(), "codex", aggregate)
	var reference *ReferenceError
	if !errors.As(err, &reference) || reference.Reference != testSignalID {
		t.Fatalf("error = %T %v, want missing root Signal ReferenceError", err, err)
	}
	if tx.writes != 0 {
		t.Fatalf("writes = %d, want no writes for a missing root Signal", tx.writes)
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
	relationID := "ERL77777777-7777-4777-8777-777777777777"
	lineage.EntityRelationID = &relationID
	tx := &publicationTransactionStub{facts: validFacts()}
	tx.facts.EntityRelations[relationID] = EntityRelationFact{
		ID: relationID, FromEntityID: "ENT88888888-8888-4888-8888-888888888888",
		ToEntityID:   testNodeID,
		TemporalFact: testTemporalFact(),
	}
	service := (&UseCase{publicationStore: publicationStoreStub{tx: tx}, now: time.Now})

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
	tx := &publicationTransactionStub{facts: validFactsWithDirectImpact()}
	impact := tx.facts.Impacts[testImpactID]
	impact.SourceEntityID = "ENTdddddddd-dddd-4ddd-8ddd-dddddddddddd"
	tx.facts.Impacts[testImpactID] = impact
	service := (&UseCase{publicationStore: publicationStoreStub{tx: tx}, now: time.Now})

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
				lineage.EntityRelationID = stringPointer("ERLeeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee")
				return aggregate
			}(),
			facts: func() ReferenceFacts {
				facts := validFacts()
				signal := facts.Signals[testSignalID]
				signal.SubjectEntityID = "ENTffffffff-ffff-4fff-8fff-ffffffffffff"
				facts.Signals[testSignalID] = signal
				facts.EntityRelations["ERLeeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee"] = EntityRelationFact{
					ID:           "ERLeeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee",
					FromEntityID: "ENTffffffff-ffff-4fff-8fff-ffffffffffff",
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
				aggregate.Theme.Events = append(aggregate.Theme.Events, ThemeEventInput{
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
				lineage.EntityRelationID = stringPointer("ERLeeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee")
				return aggregate
			}(),
			facts: func() ReferenceFacts {
				facts := validFactsWithDirectImpact()
				facts.EntityRelations["ERLeeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee"] = EntityRelationFact{
					ID:           "ERLeeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee",
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
			tx := &publicationTransactionStub{facts: test.facts}
			service := (&UseCase{publicationStore: publicationStoreStub{tx: tx}, now: time.Now})

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
		Theme: ThemeInput{
			ThemeKey: "wafer-supply", Title: "Wafer supply", OneLineConclusion: "Supply tightens",
			ConclusionDirection: "positive", ImpactStrength: "medium",
			TransmissionStage: "validation", InvestmentGuidanceAction: "focus",
			InvestmentGuidanceSummary: "Watch supply", TimeHorizonCategory: "short_term",
			Impacts: []ThemeImpactInput{{
				ChainNodeID: testNodeID, RelationRole: "driver",
				ImpactDirection: "positive", DisplayOrder: 1,
			}},
			Events: []ThemeEventInput{{EventID: testEventID, EvidenceRole: "driver"}},
		},
		ReasoningTrees: []ReasoningTree{{
			ReasonTreeInput: ReasonTreeInput{
				IndustryChainID: testChainID, Title: "Wafer chain", DisplayOrder: 1,
				OneLineConclusion: "Supply tightens", ImpactDirection: "positive",
				ImpactStrength: "medium",
				Events: []ReasonTreeEventInput{{
					EventID: testEventID, EvidenceRole: "driver", DisplayOrder: 1,
				}},
			},
			Nodes: []Node{{
				Position: 1, ChainNodeID: testNodeID,
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
	aggregate.Theme.Events = append(aggregate.Theme.Events, ThemeEventInput{
		EventID: testImpactEventID, EvidenceRole: "supporting",
	})
	aggregate.ReasoningTrees[0].Events = append(
		aggregate.ReasoningTrees[0].Events,
		ReasonTreeEventInput{
			EventID: testImpactEventID, EvidenceRole: "supporting", DisplayOrder: 2,
		},
	)
	aggregate.ReasoningTrees[0].Nodes = append(
		aggregate.ReasoningTrees[0].Nodes,
		Node{
			Position:                      2,
			ChainNodeID:                   testTargetNodeID,
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

func validBCIAnalystInferenceAggregate(reverseEdges bool, nodeCount int) (Aggregate, ReferenceFacts) {
	if nodeCount < 2 || nodeCount > 3 {
		panic("BCI analyst inference fixture requires two or three nodes")
	}
	aggregate := validAggregate()
	aggregate.AnalysisBatchID = "bci-reverse-multi-hop"
	aggregate.Theme.ThemeKey = "bci-demand"
	aggregate.Theme.Title = "BCI demand"
	aggregate.Theme.Impacts = nil
	aggregate.ReasoningTrees[0].IndustryChainID = testBCIChainID
	aggregate.ReasoningTrees[0].Title = "BCI system chain"
	aggregate.ReasoningTrees[0].Nodes = nil

	nodeIDs := []string{testBCISystemNodeID, testBCITerminalNodeID, testBCIElectrodeNodeID}
	edgeIDs := []string{testBCITerminalEdgeID, testBCIElectrodeEdgeID}
	signalKeys := []string{"market_demand", "terminal_market_demand", "electrode_market_demand"}
	for position, nodeID := range nodeIDs[:nodeCount] {
		aggregate.Theme.Impacts = append(aggregate.Theme.Impacts, ThemeImpactInput{
			ChainNodeID:     nodeID,
			RelationRole:    "exposure",
			ImpactDirection: "uncertain",
			DisplayOrder:    position + 1,
		})
		node := Node{
			Position:        position + 1,
			ChainNodeID:     nodeID,
			ImpactDirection: "uncertain",
			ImpactStrength:  "unknown",
		}
		if position == 0 {
			node.Signals = []Signal{validAggregate().ReasoningTrees[0].Nodes[0].Signals[0]}
			node.Signals[0].VariableSignalKey = "market_demand"
			node.Signals[0].SignalDirection = "increase"
			node.Signals[0].DisplaySummary = "System market demand increases"
		} else {
			edgeID := edgeIDs[position-1]
			title := "Demand propagates to the adjacent component"
			mechanism := "The adjacent component is required by the previous BCI node"
			condition := "The previous-node demand is realized"
			node.IncomingIndustryChainGraphEdgeID = &edgeID
			node.IncomingTransmissionTitle = &title
			node.IncomingTransmissionMechanism = &mechanism
			node.IncomingConditionSummary = &condition
			node.IncomingLineage = &IncomingLineage{
				SourceKind:               "analyst_inference",
				UpstreamVariableSignalID: stringPointer(testSignalID),
			}
			node.Signals = []Signal{{
				VariableSignalKey: signalKeys[position],
				SignalRole:        "primary",
				SignalDirection:   "increase",
				DisplaySummary:    "Adjacent component market demand increases",
				DisplayOrder:      1,
				Lineage: SignalLineage{
					SourceKind:               "analyst_inference",
					UpstreamVariableSignalID: stringPointer(testSignalID),
					IndustryChainGraphEdgeID: &edgeID,
				},
			}}
		}
		aggregate.ReasoningTrees[0].Nodes = append(aggregate.ReasoningTrees[0].Nodes, node)
	}

	facts := validFacts()
	facts.ChainNodeIDs = map[string]TemporalFact{}
	facts.IndustryChainIDs = map[string]TemporalFact{testBCIChainID: testTemporalFact()}
	facts.Memberships = map[string]map[string]TemporalFact{testBCIChainID: {}}
	for _, nodeID := range nodeIDs[:nodeCount] {
		facts.ChainNodeIDs[nodeID] = testTemporalFact()
		facts.Memberships[testBCIChainID][nodeID] = testTemporalFact()
	}
	rootSignal := facts.Signals[testSignalID]
	rootSignal.SubjectEntityID = testBCISystemNodeID
	rootSignal.VariableKey = "market_demand"
	rootSignal.Direction = "increase"
	facts.Signals[testSignalID] = rootSignal
	facts.GraphEdges = map[string]GraphEdgeFact{}
	for index, edgeID := range edgeIDs[:nodeCount-1] {
		fromNodeID, toNodeID := nodeIDs[index], nodeIDs[index+1]
		if reverseEdges {
			fromNodeID, toNodeID = toNodeID, fromNodeID
		}
		facts.GraphEdges[edgeID] = GraphEdgeFact{
			ReasonTreeGraphEdgeReference: ReasonTreeGraphEdgeReference{
				ID: edgeID, IndustryChainID: testBCIChainID,
				FromChainNodeID: fromNodeID, ToChainNodeID: toNodeID,
			},
			TemporalFact: testTemporalFact(),
		}
	}
	return aggregate, facts
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

type publicationStoreStub struct{ tx *publicationTransactionStub }

func (s publicationStoreStub) InResearchPublicationTransaction(ctx context.Context, fn func(PublicationTransaction) error) error {
	return fn(s.tx)
}

type publicationTransactionStub struct {
	facts              ReferenceFacts
	receipt            *Receipt
	lastReferenceQuery ReferenceQuery
	writes             int
}

func (*publicationTransactionStub) Lock(context.Context, string) error { return nil }
func (f *publicationTransactionStub) Receipt(context.Context, string) (*Receipt, error) {
	return f.receipt, nil
}
func (f *publicationTransactionStub) ReferenceFacts(_ context.Context, query ReferenceQuery) (ReferenceFacts, error) {
	f.lastReferenceQuery = query
	return f.facts, nil
}
func (f *publicationTransactionStub) InsertThemeReceipt(context.Context, Receipt) error {
	f.writes++
	return nil
}
func (f *publicationTransactionStub) InsertTheme(context.Context, PublicationThemeRecord) error {
	f.writes++
	return nil
}
func (f *publicationTransactionStub) InsertThemeImpact(context.Context, PublicationThemeImpactRecord) error {
	f.writes++
	return nil
}
func (f *publicationTransactionStub) InsertSnapshotThemeImpact(context.Context, SnapshotImpactRecord) error {
	f.writes++
	return nil
}
func (f *publicationTransactionStub) InsertThemeEvent(context.Context, PublicationThemeEventRecord) error {
	f.writes++
	return nil
}
func (f *publicationTransactionStub) InsertTreeReceipt(context.Context, ReasonTreeReceipt) error {
	f.writes++
	return nil
}
func (f *publicationTransactionStub) InsertSnapshotTreeReceipt(context.Context, SnapshotTreeReceipt) error {
	f.writes++
	return nil
}
func (f *publicationTransactionStub) InsertTree(context.Context, ReasonTreeRecord) error {
	f.writes++
	return nil
}
func (f *publicationTransactionStub) InsertSnapshotTree(context.Context, SnapshotTreeRecord) error {
	f.writes++
	return nil
}
func (f *publicationTransactionStub) InsertTreeEvent(context.Context, ReasonTreeEventRecord) error {
	f.writes++
	return nil
}
func (f *publicationTransactionStub) InsertNode(context.Context, NodeRecord) error {
	f.writes++
	return nil
}
func (f *publicationTransactionStub) InsertSnapshotNode(context.Context, SnapshotNodeRecord) error {
	f.writes++
	return nil
}
func (f *publicationTransactionStub) InsertSignal(context.Context, PublicationSignalRecord) error {
	f.writes++
	return nil
}
func (f *publicationTransactionStub) InsertSnapshotSignal(context.Context, SnapshotSignalRecord) error {
	f.writes++
	return nil
}
func (*publicationTransactionStub) Verify(context.Context, Receipt) error { return nil }

func stringPointer(value string) *string { return &value }
func TestSnapshotAggregateValidateAcceptsAnalystDisplayContentWithoutFormalIDs(t *testing.T) {
	aggregate := validSnapshotAggregate()

	_, themeID, err := aggregate.Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if themeID == "" {
		t.Fatal("Validate() themeID is empty")
	}
}

func TestPublishSnapshotUsesOnlyEventReferencesAndReplays(t *testing.T) {
	aggregate := validSnapshotAggregate()
	tx := &publicationTransactionStub{facts: ReferenceFacts{
		Events: map[string]EventFact{testEventID: {ID: testEventID}},
	}}
	service := (&UseCase{publicationStore: publicationStoreStub{tx: tx}, now: time.Now})

	result, err := service.PublishSnapshot(context.Background(), "theme-analyst", aggregate)
	if err != nil {
		t.Fatalf("PublishSnapshot() error = %v", err)
	}
	if result.PublicationMode != SnapshotPublicationMode || result.Counts.ReasoningTrees != 1 {
		t.Fatalf("PublishSnapshot() result = %#v", result)
	}
	if len(tx.lastReferenceQuery.ChainNodeIDs) != 0 || len(tx.lastReferenceQuery.SignalIDs) != 0 || len(tx.lastReferenceQuery.IndustryChainIDs) != 0 {
		t.Fatalf("snapshot queried formal ontology references: %#v", tx.lastReferenceQuery)
	}
	if !tx.lastReferenceQuery.SnapshotEventExistenceOnly {
		t.Fatalf("snapshot imposed formal Event status gates: %#v", tx.lastReferenceQuery)
	}

	receipt := snapshotPublicationPlan(aggregate, result.ThemeID, result.PayloadHash)
	receipt.PublisherSubject = "theme-analyst"
	receipt.PublishedAt, receipt.ImportedAt = result.PublishedAt, result.ImportedAt
	tx.receipt = &receipt
	writes := tx.writes
	replayed, err := service.PublishSnapshot(context.Background(), "theme-analyst", aggregate)
	if err != nil {
		t.Fatalf("replay PublishSnapshot() error = %v", err)
	}
	if !replayed.Replayed || tx.writes != writes {
		t.Fatalf("replay = %#v, writes = %d want %d", replayed, tx.writes, writes)
	}
}

func TestPublishSnapshotCanonicalizesUnorderedThemeEventSetForReplay(t *testing.T) {
	aggregate := snapshotAggregateWithThreeEvents()
	tx := &publicationTransactionStub{facts: ReferenceFacts{Events: map[string]EventFact{
		"EVT71000000-0000-5000-8000-000000000001": {ID: "EVT71000000-0000-5000-8000-000000000001"},
		"EVT71000000-0000-5000-8000-000000000002": {ID: "EVT71000000-0000-5000-8000-000000000002"},
		"EVT71000000-0000-5000-8000-000000000003": {ID: "EVT71000000-0000-5000-8000-000000000003"},
	}}}
	service := (&UseCase{publicationStore: publicationStoreStub{tx: tx}, now: time.Now})

	result, err := service.PublishSnapshot(context.Background(), "theme-analyst", aggregate)
	if err != nil {
		t.Fatalf("PublishSnapshot() unordered Theme Events error = %v", err)
	}
	receipt := snapshotPublicationPlan(aggregate, result.ThemeID, result.PayloadHash)
	receipt.PublisherSubject = "theme-analyst"
	receipt.PublishedAt, receipt.ImportedAt = result.PublishedAt, result.ImportedAt
	tx.receipt = &receipt
	writes := tx.writes

	reordered := aggregate
	reordered.Theme.Events = []SnapshotEvent{
		aggregate.Theme.Events[2], aggregate.Theme.Events[0], aggregate.Theme.Events[1],
	}
	replayed, err := service.PublishSnapshot(context.Background(), "theme-analyst", reordered)
	if err != nil {
		t.Fatalf("PublishSnapshot() reordered replay error = %v", err)
	}
	if !replayed.Replayed || replayed.PayloadHash != result.PayloadHash || tx.writes != writes {
		t.Fatalf("replay = %#v writes = %d, want same hash and %d writes", replayed, tx.writes, writes)
	}
}

func TestPublishSnapshotRejectsMissingThirdEventWithoutWrites(t *testing.T) {
	aggregate := snapshotAggregateWithThreeEvents()
	tx := &publicationTransactionStub{facts: ReferenceFacts{Events: map[string]EventFact{
		"EVT71000000-0000-5000-8000-000000000001": {ID: "EVT71000000-0000-5000-8000-000000000001"},
		"EVT71000000-0000-5000-8000-000000000003": {ID: "EVT71000000-0000-5000-8000-000000000003"},
	}}}

	_, err := (&UseCase{publicationStore: publicationStoreStub{tx: tx}, now: time.Now}).PublishSnapshot(context.Background(), "theme-analyst", aggregate)
	var reference *ReferenceError
	if !errors.As(err, &reference) || reference.Path != "theme.events[2].event_id" ||
		reference.Reference != "EVT71000000-0000-5000-8000-000000000002" {
		t.Fatalf("PublishSnapshot() error = %T %v, want missing third Event ReferenceError", err, err)
	}
	if tx.writes != 0 {
		t.Fatalf("writes = %d, want 0", tx.writes)
	}
}

func TestPublishSnapshotRejectsChangedPayloadForExistingBatch(t *testing.T) {
	aggregate := validSnapshotAggregate()
	tx := &publicationTransactionStub{facts: ReferenceFacts{
		Events: map[string]EventFact{testEventID: {ID: testEventID}},
	}}
	service := (&UseCase{publicationStore: publicationStoreStub{tx: tx}, now: time.Now})
	result, err := service.PublishSnapshot(context.Background(), "theme-analyst", aggregate)
	if err != nil {
		t.Fatalf("initial PublishSnapshot() error = %v", err)
	}
	receipt := snapshotPublicationPlan(aggregate, result.ThemeID, result.PayloadHash)
	receipt.PublisherSubject = "theme-analyst"
	receipt.PublishedAt, receipt.ImportedAt = result.PublishedAt, result.ImportedAt
	tx.receipt = &receipt
	writes := tx.writes

	aggregate.Theme.Title = "changed analyst snapshot"
	_, err = service.PublishSnapshot(context.Background(), "theme-analyst", aggregate)
	if !errors.Is(err, ErrPayloadConflict) {
		t.Fatalf("changed PublishSnapshot() error = %v, want ErrPayloadConflict", err)
	}
	if tx.writes != writes {
		t.Fatalf("writes = %d, want unchanged %d", tx.writes, writes)
	}
}

func TestPublishSnapshotValidatesOptionalEvidenceOwnership(t *testing.T) {
	aggregate := validSnapshotAggregate()
	aggregate.Theme.Events[0].EvidenceIDs = []string{testEvidenceID}
	tx := &publicationTransactionStub{facts: ReferenceFacts{
		Events: map[string]EventFact{testEventID: {ID: testEventID}},
		Evidences: map[string]EvidenceFact{testEvidenceID: {
			ID: testEvidenceID, EventID: testImpactEventID,
		}},
	}}

	_, err := (&UseCase{publicationStore: publicationStoreStub{tx: tx}, now: time.Now}).PublishSnapshot(context.Background(), "theme-analyst", aggregate)
	var reference *ReferenceError
	if !errors.As(err, &reference) || reference.Reference != testEvidenceID {
		t.Fatalf("PublishSnapshot() error = %T %v, want Evidence ReferenceError", err, err)
	}
	if tx.writes != 0 {
		t.Fatalf("writes = %d, want 0", tx.writes)
	}
}

func validSnapshotAggregate() SnapshotAggregate {
	return SnapshotAggregate{
		PublicationMode:      "analyst_snapshot",
		AnalysisBatchID:      "uat-analyst-snapshot-001",
		AnalysisAsOf:         "2026-08-03T11:00:00Z",
		DiscoveryWindowStart: "2026-08-03T03:00:00Z",
		DiscoveryWindowEnd:   "2026-08-03T07:00:00Z",
		Theme: SnapshotTheme{
			ThemeKey:                  "theme:chip-commercialization",
			Title:                     "先进芯片进入商业化验证阶段",
			OneLineConclusion:         "完成流片后仍需验证终端采用和商业兑现。",
			ConclusionDirection:       "uncertain",
			ImpactStrength:            "unknown",
			TransmissionStage:         "validation",
			InvestmentGuidanceAction:  "observe",
			InvestmentGuidanceSummary: "观察终端采用率与收入兑现。",
			TimeHorizonCategory:       "medium_term",
			Impacts: []SnapshotImpact{{
				NodeKey: "analysis-node-chip", DisplayName: "先进芯片机会",
				RelationRole: "beneficiary", ImpactDirection: "uncertain", DisplayOrder: 1,
			}},
			Events: []SnapshotEvent{{
				EventID: testEventID, EvidenceRole: "driver",
			}},
		},
		ReasoningTrees: []SnapshotReasoningTree{{
			TreeKey: "tree:chip-commercialization", DisplayName: "先进芯片商业化路径",
			Title: "先进芯片商业化路径", DisplayOrder: 1,
			OneLineConclusion: "完成流片不等于商业兑现。",
			ImpactDirection:   "uncertain", ImpactStrength: "unknown",
			Events: []SnapshotTreeEvent{{
				EventID:      testEventID,
				EvidenceRole: "driver", DisplayOrder: 1,
			}},
			Nodes: []SnapshotNode{{
				NodeKey: "analysis-node-chip", DisplayName: "先进芯片完成流片",
				Position: 1, ImpactDirection: "uncertain", ImpactStrength: "unknown",
				Signals: []SnapshotSignal{{
					SignalKey: "signal:tapeout", DisplaySummary: "完成流片",
					Role: "primary", DisplayOrder: 1,
				}},
			}},
		}},
	}
}

func snapshotAggregateWithThreeEvents() SnapshotAggregate {
	aggregate := validSnapshotAggregate()
	aggregate.AnalysisBatchID = "uat-analyst-snapshot-three-events"
	aggregate.Theme.Events = []SnapshotEvent{
		{EventID: "EVT71000000-0000-5000-8000-000000000001", EvidenceRole: "driver"},
		{EventID: "EVT71000000-0000-5000-8000-000000000003", EvidenceRole: "supporting"},
		{EventID: "EVT71000000-0000-5000-8000-000000000002", EvidenceRole: "context"},
	}
	aggregate.ReasoningTrees[0].Events = []SnapshotTreeEvent{
		{EventID: "EVT71000000-0000-5000-8000-000000000001", EvidenceRole: "driver", DisplayOrder: 1},
		{EventID: "EVT71000000-0000-5000-8000-000000000003", EvidenceRole: "supporting", DisplayOrder: 2},
		{EventID: "EVT71000000-0000-5000-8000-000000000002", EvidenceRole: "context", DisplayOrder: 3},
	}
	return aggregate
}

type contextStoreStub struct {
	query        AnalysisContextStoreQuery
	page         AnalysisContextStorePage
	dictionaries Dictionaries
}

func newAnalysisContextTestUseCase(store *contextStoreStub) *UseCase {
	return &UseCase{eventProvider: store, semanticProvider: store, entityProvider: store}
}

func (s *contextStoreStub) ListResearchEvents(
	_ context.Context,
	query eventbiz.ResearchEventQuery,
) (eventbiz.ResearchEventPage, error) {
	s.query = AnalysisContextStoreQuery{
		DiscoveryWindowStart: query.DiscoveryWindowStart, DiscoveryWindowEnd: query.DiscoveryWindowEnd,
		AnalysisAsOf: query.AnalysisAsOf, PageSize: query.PageSize,
		AfterKnowledgeAvailableAt: query.AfterKnowledgeAvailableAt, AfterEventID: query.AfterEventID,
	}
	page := eventbiz.ResearchEventPage{Events: make([]eventbiz.ResearchEventRecord, 0, len(s.page.Bundles)), HasMore: s.page.HasMore}
	for _, bundle := range s.page.Bundles {
		page.Events = append(page.Events, eventbiz.ResearchEventRecord{
			Event: bundle.Bundle.Event, Evidence: bundle.Bundle.Evidence,
			KnowledgeAvailableAt: bundle.KnowledgeAvailableAt,
		})
	}
	return page, nil
}

func (s *contextStoreStub) ListResearchSemantics(context.Context, eventsemanticbiz.ResearchSemanticQuery) ([]eventsemanticbiz.ResearchSemanticRecord, error) {
	result := make([]eventsemanticbiz.ResearchSemanticRecord, 0, len(s.page.Bundles))
	for _, bundle := range s.page.Bundles {
		result = append(result, eventsemanticbiz.ResearchSemanticRecord{
			EventID: bundle.EventID, EntityLinks: bundle.Bundle.EntityLinks,
			VariableSignals: bundle.Bundle.VariableSignals,
		})
	}
	return result, nil
}

func (s *contextStoreStub) ResearchSemanticClosure(context.Context, eventsemanticbiz.ResearchSemanticClosureQuery) (eventsemanticbiz.ResearchSemanticDictionaries, error) {
	return eventsemanticbiz.ResearchSemanticDictionaries{
		VariableDefinitions:     s.dictionaries.VariableDefinitions,
		DirectTransmissionRules: s.dictionaries.DirectTransmissionRules,
		AcceptancePolicies:      s.dictionaries.AcceptancePolicies,
	}, nil
}

func (s *contextStoreStub) ResearchReferenceClosure(context.Context, entitybiz.ResearchReferenceQuery) (entitybiz.ResearchReferenceDictionaries, error) {
	return entitybiz.ResearchReferenceDictionaries{
		Entities: s.dictionaries.Entities, RelationDefinitions: s.dictionaries.RelationDefinitions,
		EntityRelations: s.dictionaries.EntityRelations, IndustryChains: s.dictionaries.IndustryChains,
		IndustryChainMemberships: s.dictionaries.IndustryChainMemberships,
		IndustryChainGraphEdges:  s.dictionaries.IndustryChainGraphEdges,
	}, nil
}

func (*contextStoreStub) SearchResearchGraph(context.Context, GraphQuery) (GraphSubgraph, error) {
	return GraphSubgraph{}, nil
}

func TestAnalysisContextReturnsStableCursorBoundToTheResearchWindow(t *testing.T) {
	store := &contextStoreStub{page: AnalysisContextStorePage{
		Bundles: []BundleRecord{
			{
				KnowledgeAvailableAt: time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC),
				EventID:              "EVT11111111-1111-4111-8111-111111111111",
				Bundle: EventSemanticBundle{Event: Event{
					ID: "EVT11111111-1111-4111-8111-111111111111",
				}},
			},
			{
				KnowledgeAvailableAt: time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC),
				EventID:              "EVT22222222-2222-4222-8222-222222222222",
				Bundle: EventSemanticBundle{Event: Event{
					ID: "EVT22222222-2222-4222-8222-222222222222",
				}},
			},
		},
		HasMore: true,
	}}
	service := newAnalysisContextTestUseCase(store)
	request := AnalysisContextRequest{
		DiscoveryWindowStart: "2026-07-28T00:00:00Z",
		DiscoveryWindowEnd:   "2026-07-29T00:00:00Z",
		AnalysisAsOf:         "2026-07-29T00:00:00Z",
		PageSize:             2,
	}

	first, err := service.List(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !first.HasMore || first.NextCursor == "" || len(first.EventSemanticBundles) != 2 {
		t.Fatalf("first page = %#v", first)
	}

	store.page = AnalysisContextStorePage{}
	request.Cursor = first.NextCursor
	second, err := service.List(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if store.query.AfterEventID != "EVT22222222-2222-4222-8222-222222222222" ||
		store.query.AfterKnowledgeAvailableAt == nil ||
		!store.query.AfterKnowledgeAvailableAt.Equal(time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("continuation query = %#v", store.query)
	}
	if second.EventSemanticBundles == nil || second.HasMore || second.NextCursor != "" {
		t.Fatalf("empty continuation = %#v", second)
	}
}

func TestAnalysisContextFiltersSemanticsWhenLegacyEvidenceIsUnavailable(t *testing.T) {
	eventID := "EVT11111111-1111-4111-8111-111111111111"
	evidenceID := "EEL22222222-2222-4222-8222-222222222222"
	linkID := "ENL33333333-3333-4333-8333-333333333333"
	store := &contextStoreStub{page: AnalysisContextStorePage{Bundles: []BundleRecord{{
		KnowledgeAvailableAt: time.Date(2026, 7, 28, 8, 0, 0, 0, time.UTC),
		EventID:              eventID,
		Bundle: EventSemanticBundle{
			Event: Event{ID: eventID},
			EntityLinks: []EntityLink{{
				EventEntityLinkID: linkID,
				EvidenceIDs:       []string{evidenceID},
			}},
			VariableSignals: []VariableSignal{{
				VariableSignalID:         "VSG44444444-4444-4444-8444-444444444444",
				SubjectEventEntityLinkID: linkID,
				EvidenceIDs:              []string{evidenceID},
			}},
		},
	}}}}
	result, err := newAnalysisContextTestUseCase(store).List(context.Background(), AnalysisContextRequest{
		DiscoveryWindowStart: "2026-07-28T00:00:00Z",
		DiscoveryWindowEnd:   "2026-07-29T00:00:00Z",
		AnalysisAsOf:         "2026-07-29T00:00:00Z",
		PageSize:             1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.EventSemanticBundles) != 1 ||
		len(result.EventSemanticBundles[0].Evidence) != 0 ||
		len(result.EventSemanticBundles[0].EntityLinks) != 0 ||
		len(result.EventSemanticBundles[0].VariableSignals) != 0 {
		t.Fatalf("Analysis Context Event-only bundle = %#v", result.EventSemanticBundles)
	}
}

func TestAnalysisContextReturnsVersionedPageAndReferenceClosureFingerprints(t *testing.T) {
	store := &contextStoreStub{page: AnalysisContextStorePage{
		Bundles: []BundleRecord{},
	}}
	result, err := newAnalysisContextTestUseCase(store).List(context.Background(), AnalysisContextRequest{
		DiscoveryWindowStart: "2026-07-28T00:00:00Z",
		DiscoveryWindowEnd:   "2026-07-29T00:00:00Z",
		AnalysisAsOf:         "2026-07-29T00:00:00Z",
		PageSize:             20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ContractVersion != "research-analysis-context.v1" ||
		result.TBoxContractVersion != "event-semantics.phase-one@1" {
		t.Fatalf(
			"versions = contract %q TBox %q",
			result.ContractVersion,
			result.TBoxContractVersion,
		)
	}
	if !testAnalysisHashPattern(result.EventPageFingerprint) ||
		!testAnalysisHashPattern(result.ReferenceClosureFingerprint) {
		t.Fatalf(
			"fingerprints = event %q closure %q",
			result.EventPageFingerprint,
			result.ReferenceClosureFingerprint,
		)
	}
	if result.Dictionaries.Entities == nil ||
		result.EventSemanticBundles == nil {
		t.Fatalf("empty page must preserve empty arrays: %#v", result)
	}
}

func TestAnalysisContextRejectsInvalidResearchTimeBoundariesAndCursorMismatch(t *testing.T) {
	service := newAnalysisContextTestUseCase(&contextStoreStub{})
	valid := AnalysisContextRequest{
		DiscoveryWindowStart: "2026-07-28T00:00:00Z",
		DiscoveryWindowEnd:   "2026-07-29T00:00:00Z",
		AnalysisAsOf:         "2026-07-29T00:00:00Z",
		PageSize:             20,
	}
	cases := []AnalysisContextRequest{
		{
			DiscoveryWindowStart: valid.DiscoveryWindowStart,
			DiscoveryWindowEnd:   "2026-07-30T00:00:00Z",
			AnalysisAsOf:         valid.AnalysisAsOf,
			PageSize:             valid.PageSize,
		},
		{
			DiscoveryWindowStart:   valid.DiscoveryWindowStart,
			DiscoveryWindowEnd:     valid.DiscoveryWindowEnd,
			AnalysisAsOf:           valid.AnalysisAsOf,
			PredictionHorizonStart: stringPointer("2026-07-28T12:00:00Z"),
			PredictionHorizonEnd:   stringPointer("2026-07-30T00:00:00Z"),
			PageSize:               valid.PageSize,
		},
		{
			DiscoveryWindowStart: "2025-07-27T00:00:00Z",
			DiscoveryWindowEnd:   valid.DiscoveryWindowEnd,
			AnalysisAsOf:         valid.AnalysisAsOf,
			PageSize:             valid.PageSize,
		},
	}
	for _, request := range cases {
		if _, err := service.List(context.Background(), request); err == nil {
			t.Fatalf("request unexpectedly accepted: %#v", request)
		}
	}

	store := &contextStoreStub{page: AnalysisContextStorePage{Bundles: []BundleRecord{{
		KnowledgeAvailableAt: time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC),
		EventID:              "EVT11111111-1111-4111-8111-111111111111",
		Bundle: EventSemanticBundle{Event: Event{
			ID: "EVT11111111-1111-4111-8111-111111111111",
		}},
	}}, HasMore: true}}
	service = newAnalysisContextTestUseCase(store)
	first, err := service.List(context.Background(), valid)
	if err != nil {
		t.Fatal(err)
	}
	valid.Cursor = first.NextCursor
	valid.DiscoveryWindowStart = "2026-07-27T00:00:00Z"
	if _, err := service.List(context.Background(), valid); err == nil {
		t.Fatal("cursor accepted with a changed discovery window")
	}
}

func TestAnalysisContextKeepsCursorValidAfterUnrelatedDictionaryChanges(t *testing.T) {
	store := &contextStoreStub{page: AnalysisContextStorePage{
		Bundles: []BundleRecord{{
			KnowledgeAvailableAt: time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC),
			EventID:              "EVT11111111-1111-4111-8111-111111111111",
			Bundle: EventSemanticBundle{Event: Event{
				ID: "EVT11111111-1111-4111-8111-111111111111",
			}},
		}},
		HasMore: true,
	}}
	service := newAnalysisContextTestUseCase(store)
	request := AnalysisContextRequest{
		DiscoveryWindowStart: "2026-07-28T00:00:00Z",
		DiscoveryWindowEnd:   "2026-07-29T00:00:00Z",
		AnalysisAsOf:         "2026-07-29T00:00:00Z",
		PageSize:             20,
	}
	first, err := service.List(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	request.Cursor = first.NextCursor
	store.page = AnalysisContextStorePage{}
	store.dictionaries = Dictionaries{Entities: []Entity{{
		EntityID:   "CND33333333-3333-4333-8333-333333333333",
		EntityType: "chain_node",
	}}}
	if _, err := service.List(context.Background(), request); err != nil {
		t.Fatalf("cursor was invalidated by an unrelated dictionary change: %v", err)
	}
}

func TestAnalysisContextRejectsCursorWithInvalidTerminalEventID(t *testing.T) {
	request := AnalysisContextRequest{
		DiscoveryWindowStart: "2026-07-28T00:00:00Z",
		DiscoveryWindowEnd:   "2026-07-29T00:00:00Z",
		AnalysisAsOf:         "2026-07-29T00:00:00Z",
		PageSize:             20,
	}
	_, _, fingerprint, err := validateAnalysisContextRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	request.Cursor, err = encodeCursor(contextCursor{
		Version:              1,
		Fingerprint:          fingerprint,
		KnowledgeAvailableAt: "2026-07-28T09:00:00Z",
		EventID:              "not-a-uuid",
	})
	if err != nil {
		t.Fatal(err)
	}

	store := &contextStoreStub{page: AnalysisContextStorePage{}}
	if _, err := newAnalysisContextTestUseCase(store).List(context.Background(), request); err == nil {
		t.Fatal("cursor with an invalid terminal Event ID was accepted")
	}
}

func TestAnalysisContextRequiresRestartWhenPageReferenceClosureIsInconsistent(t *testing.T) {
	eventID := "EVT11111111-1111-4111-8111-111111111111"
	store := &contextStoreStub{page: AnalysisContextStorePage{
		Bundles: []BundleRecord{{
			KnowledgeAvailableAt: time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC),
			EventID:              eventID,
			Bundle: EventSemanticBundle{
				Event: Event{ID: eventID},
				EntityLinks: []EntityLink{{
					EventEntityLinkID: "ENL22222222-2222-4222-8222-222222222222",
					EntityID:          "CND33333333-3333-4333-8333-333333333333",
				}},
			},
		}},
	}}
	request := AnalysisContextRequest{
		DiscoveryWindowStart: "2026-07-28T00:00:00Z",
		DiscoveryWindowEnd:   "2026-07-29T00:00:00Z",
		AnalysisAsOf:         "2026-07-29T00:00:00Z",
		PageSize:             20,
	}

	if _, err := newAnalysisContextTestUseCase(store).List(context.Background(), request); !errors.Is(
		err, ErrReferenceClosureInconsistent,
	) {
		t.Fatalf("error = %v, want reference closure inconsistency", err)
	}
}

func TestAnalysisContextFailsClosedForBundleClosureAndPageBudgets(t *testing.T) {
	request := AnalysisContextRequest{
		DiscoveryWindowStart: "2026-07-28T00:00:00Z",
		DiscoveryWindowEnd:   "2026-07-29T00:00:00Z",
		AnalysisAsOf:         "2026-07-29T00:00:00Z",
		PageSize:             20,
	}
	tests := []struct {
		name          string
		store         *contextStoreStub
		wantComponent string
	}{
		{
			name: "complete Event Bundle",
			store: &contextStoreStub{page: AnalysisContextStorePage{Bundles: []BundleRecord{{
				KnowledgeAvailableAt: time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC),
				EventID:              "EVT11111111-1111-4111-8111-111111111111",
				Bundle: EventSemanticBundle{Event: Event{
					ID:      "EVT11111111-1111-4111-8111-111111111111",
					Summary: strings.Repeat("x", MaxEventSemanticBundleBytes),
				}},
			}}}},
			wantComponent: "event_semantic_bundle",
		},
		{
			name:          "complete Event Bundle row count",
			store:         bundleRowBudgetStore(),
			wantComponent: "event_semantic_bundle",
		},
		{
			name: "page reference closure",
			store: &contextStoreStub{dictionaries: Dictionaries{
				Entities: []Entity{{
					EntityID:   "CND33333333-3333-4333-8333-333333333333",
					EntityType: "chain_node",
					Name:       strings.Repeat("x", MaxDictionaryBytes),
				}},
			}},
			wantComponent: "reference_closure",
		},
		{
			name:          "combined reference closure row count",
			store:         dictionaryRowBudgetStore(),
			wantComponent: "reference_closure",
		},
		{
			name:          "encoded Analysis Context page",
			store:         pageBudgetStore(),
			wantComponent: "analysis_context_page",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := newAnalysisContextTestUseCase(test.store).List(context.Background(), request)
			var resourceLimit *ResearchResourceLimitError
			if !errors.As(err, &resourceLimit) ||
				resourceLimit.Component != test.wantComponent ||
				resourceLimit.ActualBytes == nil ||
				resourceLimit.MaxBytes == nil {
				t.Fatalf("error = %#v", err)
			}
		})
	}
}

func TestAnalysisContextMapsSemanticProviderResourceLimit(t *testing.T) {
	actual := int64(50_001)
	maximum := int64(50_000)
	err := mapResearchSemanticProviderError(&eventsemanticbiz.ResearchResourceLimitError{
		Reason: "semantic dictionary is too large", Component: "reference_closure",
		ActualRows: &actual, MaxRows: &maximum, RetryGuidance: "reduce_page_size",
	})
	var limit *ResearchResourceLimitError
	if !errors.As(err, &limit) || limit.Component != "reference_closure" ||
		limit.ActualRows == nil || *limit.ActualRows != actual {
		t.Fatalf("mapped provider limit = %#v", err)
	}
}

func TestAnalysisContextDictionaryBudgetCountsApplicableEntityTypeMappings(t *testing.T) {
	value := Dictionaries{VariableDefinitions: []VariableDefinition{{
		Key: "metric", Version: 1, ApplicableEntityTypes: []string{"company", "industry_chain"},
	}}}
	if got, want := dictionaryRows(value), 3; got != want {
		t.Fatalf("dictionaryRows() = %d, want %d", got, want)
	}
}

func bundleRowBudgetStore() *contextStoreStub {
	evidence := make([]Evidence, 0, MaxEventSemanticBundleRows+1)
	for index := 1; index <= MaxEventSemanticBundleRows+1; index++ {
		evidence = append(evidence, Evidence{EvidenceID: fmt.Sprintf("EEL50000000-0000-4000-8000-%012d", index)})
	}
	eventID := "EVT51111111-1111-4111-8111-111111111111"
	return &contextStoreStub{page: AnalysisContextStorePage{Bundles: []BundleRecord{{
		KnowledgeAvailableAt: time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC), EventID: eventID,
		Bundle: EventSemanticBundle{Event: Event{ID: eventID}, Evidence: evidence},
	}}}}
}

func dictionaryRowBudgetStore() *contextStoreStub {
	relations := make([]RelationDefinition, 0, MaxDictionaryRows+1)
	for index := 1; index <= MaxDictionaryRows+1; index++ {
		relations = append(relations, RelationDefinition{RelationType: fmt.Sprintf("r%d", index), Direction: "directed"})
	}
	return &contextStoreStub{dictionaries: Dictionaries{RelationDefinitions: relations}}
}

func pageBudgetStore() *contextStoreStub {
	bundles := make([]BundleRecord, 0, 10)
	for index := 1; index <= 10; index++ {
		eventID := fmt.Sprintf("EVT40000000-0000-4000-8000-%012d", index)
		bundles = append(bundles, BundleRecord{
			KnowledgeAvailableAt: time.Date(2026, 7, 28, index, 0, 0, 0, time.UTC),
			EventID:              eventID,
			Bundle: EventSemanticBundle{Event: Event{
				ID: eventID, Summary: strings.Repeat("x", 450*1024),
			}},
		})
	}
	return &contextStoreStub{
		page: AnalysisContextStorePage{Bundles: bundles},
		dictionaries: Dictionaries{
			Entities: []Entity{{
				EntityID:   "CND33333333-3333-4333-8333-333333333333",
				EntityType: "chain_node",
				Name:       strings.Repeat("x", 3900*1024),
			}},
		},
	}
}

func testAnalysisHashPattern(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') &&
			(character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

type graphStoreStub struct {
	query GraphQuery
	graph GraphSubgraph
}

func (s *graphStoreStub) SearchResearchGraph(_ context.Context, query GraphQuery) (GraphSubgraph, error) {
	s.query = query
	return s.graph, nil
}

func TestGraphReturnsDeterministicReferenceCompleteGraph(t *testing.T) {
	store := &graphStoreStub{graph: GraphSubgraph{
		ActualDepth: 1,
		Entities: []GraphEntity{
			{
				EntityID:   "ENT11111111-1111-4111-8111-111111111111",
				EntityType: "company",
				Name:       "Producer", CanonicalName: "producer", Status: "active",
			},
			{
				EntityID:   "ICH22222222-2222-4222-8222-222222222222",
				EntityType: "industry_chain",
				Name:       "Product", CanonicalName: "product", Status: "active",
			},
		},
		RelationDefinitions: []GraphRelationDefinition{{
			RelationType: "produces", Direction: "directed",
		}},
		EntityRelations: []GraphEntityRelation{{
			EntityRelationID: "ERL33333333-3333-4333-8333-333333333333",
			FromEntityID:     "ENT11111111-1111-4111-8111-111111111111",
			ToEntityID:       "ICH22222222-2222-4222-8222-222222222222",
			RelationType:     "produces",
			Status:           "active",
		}},
		IndustryChains:           []GraphIndustryChain{},
		IndustryChainMemberships: []GraphIndustryChainMembership{},
		IndustryChainGraphEdges:  []GraphIndustryChainEdge{},
	}}
	result, err := (&UseCase{graphStore: store}).Search(context.Background(), GraphSearchRequest{
		AnalysisAsOf:  "2026-07-30T00:00:00Z",
		SeedEntityIDs: []string{"ENT11111111-1111-4111-8111-111111111111"},
		RelationFilters: []RelationFilter{{
			RelationType: "produces",
			Direction:    DirectionOutgoing,
		}},
		MaxDepth:   1,
		NodeBudget: 10,
		EdgeBudget: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ContractVersion != "research-graph-search.v1" ||
		result.ActualDepth != 1 ||
		!testGraphHashPattern(result.QueryFingerprint) ||
		!testGraphHashPattern(result.GraphFingerprint) {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Entities) != 2 ||
		len(result.EntityRelations) != 1 ||
		store.query.RelationFilters[0].Direction != DirectionOutgoing {
		t.Fatalf("result = %#v query = %#v", result, store.query)
	}
}

func testGraphHashPattern(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') &&
			(character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func TestGraphAcceptsIndependentCountryAndRegionObjects(t *testing.T) {
	store := &graphStoreStub{graph: GraphSubgraph{
		ActualDepth: 1,
		Entities: []GraphEntity{
			{EntityID: "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b", EntityType: "country", Name: "中国", CanonicalName: "中国", Status: "active"},
			{EntityID: "REG88d53cc8-1c75-57e6-a02c-56f9a4bc13c4", EntityType: "region", Name: "亚太地区", CanonicalName: "亚太地区", Status: "active"},
		},
		RelationDefinitions: []GraphRelationDefinition{{RelationType: "belongs_to_region", Direction: "directed"}},
		EntityRelations: []GraphEntityRelation{{
			EntityRelationID: "ERL33333333-3333-4333-8333-333333333333",
			FromEntityID:     "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b", ToEntityID: "REG88d53cc8-1c75-57e6-a02c-56f9a4bc13c4", RelationType: "belongs_to_region", Status: "active",
		}},
		IndustryChains: []GraphIndustryChain{}, IndustryChainMemberships: []GraphIndustryChainMembership{}, IndustryChainGraphEdges: []GraphIndustryChainEdge{},
	}}
	result, err := (&UseCase{graphStore: store}).Search(context.Background(), GraphSearchRequest{
		AnalysisAsOf: "2026-08-14T00:00:00Z", SeedEntityIDs: []string{"COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b"},
		RelationFilters: []RelationFilter{{RelationType: "belongs_to_region", Direction: DirectionOutgoing}},
		MaxDepth:        1, NodeBudget: 10, EdgeBudget: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entities) != 2 || result.Entities[0].EntityID != "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b" || store.query.SeedEntityIDs[0] != "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b" {
		t.Fatalf("Country graph result=%#v query=%#v", result, store.query)
	}
}

func TestGraphRejectsInvalidOrOrphanedGraphRequests(t *testing.T) {
	valid := GraphSearchRequest{
		AnalysisAsOf:  "2026-07-30T00:00:00Z",
		SeedEntityIDs: []string{"ENT11111111-1111-4111-8111-111111111111"},
		RelationFilters: []RelationFilter{{
			RelationType: "produces",
			Direction:    DirectionOutgoing,
		}},
		MaxDepth:   1,
		NodeBudget: 10,
		EdgeBudget: 10,
	}
	for _, mutate := range []func(*GraphSearchRequest){
		func(request *GraphSearchRequest) { request.SeedEntityIDs = nil },
		func(request *GraphSearchRequest) {
			request.SeedEntityIDs = append(request.SeedEntityIDs, request.SeedEntityIDs[0])
		},
		func(request *GraphSearchRequest) { request.RelationFilters[0].Direction = "sideways" },
		func(request *GraphSearchRequest) {
			request.RelationFilters = append(
				request.RelationFilters,
				RelationFilter{
					RelationType: "produces",
					Direction:    DirectionIncoming,
				},
			)
		},
		func(request *GraphSearchRequest) { request.MaxDepth = GraphMaxDepth + 1 },
		func(request *GraphSearchRequest) { request.NodeBudget = 0 },
	} {
		request := valid
		request.SeedEntityIDs = append([]string(nil), valid.SeedEntityIDs...)
		request.RelationFilters = append([]RelationFilter(nil), valid.RelationFilters...)
		mutate(&request)
		if _, err := (&UseCase{graphStore: &graphStoreStub{}}).Search(
			context.Background(),
			request,
		); err == nil {
			t.Fatalf("request unexpectedly accepted: %#v", request)
		}
	}

	store := &graphStoreStub{graph: GraphSubgraph{
		Entities: []GraphEntity{{
			EntityID:   "ENT11111111-1111-4111-8111-111111111111",
			EntityType: "company",
		}},
		RelationDefinitions: []GraphRelationDefinition{{
			RelationType: "produces", Direction: "directed",
		}},
		EntityRelations: []GraphEntityRelation{{
			EntityRelationID: "ERL33333333-3333-4333-8333-333333333333",
			FromEntityID:     "ENT11111111-1111-4111-8111-111111111111",
			ToEntityID:       "ICH22222222-2222-4222-8222-222222222222",
			RelationType:     "produces",
		}},
	}}
	if _, err := (&UseCase{graphStore: store}).Search(context.Background(), valid); err == nil {
		t.Fatal("orphaned graph edge was accepted")
	}
}

func TestGraphReportsTheExceededGraphBudgetDimension(t *testing.T) {
	valid := GraphSearchRequest{
		AnalysisAsOf:  "2026-07-30T00:00:00Z",
		SeedEntityIDs: []string{"ENT11111111-1111-4111-8111-111111111111"},
		RelationFilters: []RelationFilter{{
			RelationType: "produces",
			Direction:    DirectionOutgoing,
		}},
		MaxDepth: 1, NodeBudget: 10, EdgeBudget: 1,
	}
	entities := []GraphEntity{
		{EntityID: "ENT11111111-1111-4111-8111-111111111111"},
		{EntityID: "ICH22222222-2222-4222-8222-222222222222"},
	}
	relations := []GraphEntityRelation{
		{
			EntityRelationID: "ERL33333333-3333-4333-8333-333333333333",
			FromEntityID:     entities[0].EntityID,
			ToEntityID:       entities[1].EntityID,
			RelationType:     "produces",
		},
		{
			EntityRelationID: "ERL44444444-4444-4444-8444-444444444444",
			FromEntityID:     entities[0].EntityID,
			ToEntityID:       entities[1].EntityID,
			RelationType:     "produces",
		},
	}
	_, err := (&UseCase{graphStore: &graphStoreStub{graph: GraphSubgraph{
		Entities:            entities,
		RelationDefinitions: []GraphRelationDefinition{{RelationType: "produces"}},
		EntityRelations:     relations,
	}}}).Search(context.Background(), valid)
	var resourceLimit *ResearchResourceLimitError
	if !errors.As(err, &resourceLimit) ||
		resourceLimit.Component != "research_graph_edges" ||
		resourceLimit.ActualRows == nil ||
		resourceLimit.MaxRows == nil ||
		*resourceLimit.ActualRows != 2 ||
		*resourceLimit.MaxRows != 1 {
		t.Fatalf("edge budget error = %#v", err)
	}

	valid.EdgeBudget = 10
	_, err = (&UseCase{graphStore: &graphStoreStub{graph: GraphSubgraph{
		Entities: []GraphEntity{{
			EntityID: "ENT11111111-1111-4111-8111-111111111111",
			Name:     strings.Repeat("x", GraphMaxResultBytes),
		}},
	}}}).Search(context.Background(), valid)
	if !errors.As(err, &resourceLimit) ||
		resourceLimit.Component != "research_graph_result" ||
		resourceLimit.ActualBytes == nil ||
		resourceLimit.MaxBytes == nil {
		t.Fatalf("response budget error = %#v", err)
	}
}

type fakeRepository struct {
	themePage      ThemeStorePage
	themeDetail    ThemeDetailRecord
	reasoningTrees ReasoningTreeListRecord
	reasoningTree  ReasoningTreeDetailRecord
	err            error
	themeFilter    ThemeListFilter
	themeID        string
	treeID         string
}

func newReadTestUseCase(repository Repository, now func() time.Time) *UseCase {
	return &UseCase{repository: repository, now: now}
}

func (f *fakeRepository) ListResearchThemes(_ context.Context, filter ThemeListFilter) (ThemeStorePage, error) {
	f.themeFilter = filter
	return f.themePage, f.err
}
func (f *fakeRepository) GetResearchTheme(context.Context, string) (ThemeDetailRecord, error) {
	return f.themeDetail, f.err
}
func (f *fakeRepository) ListResearchThemeReasoningTrees(_ context.Context, themeID string) (ReasoningTreeListRecord, error) {
	f.themeID = themeID
	return f.reasoningTrees, f.err
}
func (f *fakeRepository) GetResearchThemeReasoningTree(_ context.Context, themeID, treeID string) (ReasoningTreeDetailRecord, error) {
	f.themeID, f.treeID = themeID, treeID
	return f.reasoningTree, f.err
}

func TestServiceUsesPublishedAtCursorForThemeOrdering(t *testing.T) {
	now := time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC)
	repository := &fakeRepository{themePage: ThemeStorePage{
		AsOf: now, WindowStart: now.Add(-24 * time.Hour), WindowEnd: now,
		ThemeCount: 1, EventCount: 2, HasMore: true,
		Items: []ThemeSummaryRecord{{
			ID: "11111111-1111-4111-8111-111111111111", AnalysisBatchID: "batch",
			Title: "Theme", OneLineConclusion: "结论", ConclusionDirection: "positive",
			ImpactStrength: "medium", TransmissionStage: "validation",
			InvestmentGuidanceAction: "focus", InvestmentGuidanceSummary: "关注订单",
			TimeHorizonCategory: "short_term", AnalysisAsOf: now, WindowStart: now.Add(-time.Hour),
			WindowEnd: now, PublishedAt: now, Impacts: []ThemeImpactRecord{},
		}},
	}}
	service := newReadTestUseCase(repository, func() time.Time { return now })

	page, err := service.ListThemes(context.Background(), ResearchListRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if page.NextCursor == nil {
		t.Fatal("next cursor is nil")
	}
	cursor, err := decodeResearchCursor(*page.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	if cursor.Kind != "themes" || cursor.ID != page.Items[0].ID || !cursor.PublishedAt.Equal(now) {
		t.Fatalf("cursor = %#v", cursor)
	}
}

func TestServiceUsesExplicitPublicationRangeForThemeListing(t *testing.T) {
	now := time.Date(2026, 8, 4, 8, 30, 0, 0, time.UTC)
	publishedFrom := time.Date(2026, 8, 3, 16, 0, 0, 0, time.UTC)
	publishedTo := time.Date(2026, 8, 4, 16, 0, 0, 0, time.UTC)
	repository := &fakeRepository{themePage: ThemeStorePage{
		AsOf: now, WindowStart: publishedFrom, WindowEnd: publishedTo,
	}}
	service := newReadTestUseCase(repository, func() time.Time { return now })

	page, err := service.ListThemes(context.Background(), ResearchListRequest{
		PublishedFrom: &publishedFrom,
		PublishedTo:   &publishedTo,
		Limit:         5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !repository.themeFilter.WindowStart.Equal(publishedFrom) || !repository.themeFilter.WindowEnd.Equal(publishedTo) {
		t.Fatalf("repository range = [%s, %s), want [%s, %s)", repository.themeFilter.WindowStart, repository.themeFilter.WindowEnd, publishedFrom, publishedTo)
	}
	if !page.WindowStart.Equal(publishedFrom) || !page.WindowEnd.Equal(publishedTo) {
		t.Fatalf("response range = [%s, %s), want [%s, %s)", page.WindowStart, page.WindowEnd, publishedFrom, publishedTo)
	}
}

func TestServiceRejectsMixedLegacyAndExplicitPublicationRange(t *testing.T) {
	publishedFrom := time.Date(2026, 8, 3, 16, 0, 0, 0, time.UTC)
	publishedTo := time.Date(2026, 8, 4, 16, 0, 0, 0, time.UTC)
	service := newReadTestUseCase(&fakeRepository{}, time.Now)

	_, err := service.ListThemes(context.Background(), ResearchListRequest{
		WindowHours:   24,
		PublishedFrom: &publishedFrom,
		PublishedTo:   &publishedTo,
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidRequest)
	}
}

func TestServiceRejectsExplicitRangeCursorWithDifferentBounds(t *testing.T) {
	now := time.Date(2026, 8, 4, 8, 30, 0, 0, time.UTC)
	publishedFrom := time.Date(2026, 8, 3, 16, 0, 0, 0, time.UTC)
	publishedTo := time.Date(2026, 8, 4, 16, 0, 0, 0, time.UTC)
	repository := &fakeRepository{themePage: ThemeStorePage{
		AsOf: now, WindowStart: publishedFrom, WindowEnd: publishedTo, HasMore: true,
		Items: []ThemeSummaryRecord{{ID: "11111111-1111-4111-8111-111111111111", PublishedAt: now}},
	}}
	service := newReadTestUseCase(repository, func() time.Time { return now })
	first, err := service.ListThemes(context.Background(), ResearchListRequest{
		PublishedFrom: &publishedFrom, PublishedTo: &publishedTo, Limit: 5,
	})
	if err != nil || first.NextCursor == nil {
		t.Fatalf("cursor/error = %v/%v", first.NextCursor, err)
	}
	differentTo := publishedTo.Add(time.Hour)

	_, err = service.ListThemes(context.Background(), ResearchListRequest{
		PublishedFrom: &publishedFrom, PublishedTo: &differentTo, Limit: 5, Cursor: *first.NextCursor,
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidRequest)
	}
}

func TestServiceReadsHistoricalThemeDetailWithoutListWindowMembership(t *testing.T) {
	themeID := "RTH11111111-1111-4111-8111-111111111111"
	oldPublication := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repository := &fakeRepository{themeDetail: ThemeDetailRecord{ThemeSummaryRecord: ThemeSummaryRecord{
		ID: themeID, PublishedAt: oldPublication,
	}}}
	service := newReadTestUseCase(repository, func() time.Time {
		return time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	})

	detail, err := service.GetTheme(context.Background(), themeID, ResearchDetailRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if detail.Theme.ID != themeID || !detail.Theme.PublishedAt.Equal(oldPublication) {
		t.Fatalf("detail = %#v", detail)
	}
}

func TestServiceMapsReasoningTreeSignalsWithoutChoosingImpactPriority(t *testing.T) {
	themeID := "RTH11111111-1111-4111-8111-111111111111"
	treeID := "RRT22222222-2222-4222-8222-222222222222"
	nodeID := "RRN33333333-3333-4333-8333-333333333333"
	repository := &fakeRepository{reasoningTree: ReasoningTreeDetailRecord{
		ThemeID:       themeID,
		ImpactNodeIDs: []string{nodeID},
		ReasoningTree: ReasoningTreeRecord{
			ReasoningTreeID: treeID, ThemeID: themeID,
			IndustryChainID:   "ICH44444444-4444-4444-8444-444444444444",
			IndustryChainName: "产业链", Title: "Tree", DisplayOrder: 1,
			OneLineConclusion: "结论", ImpactDirection: "positive", ImpactStrength: "medium",
			PublishedAt: time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC),
			Nodes: []ReasoningTreeNodeRecord{{
				ID: nodeID, Position: 1, ChainNodeID: nodeID, Name: "节点",
				ImpactDirection: "positive", ImpactStrength: "medium",
				Signals: []SignalRecord{
					{VariableSignalKey: "primary", SignalRole: "primary", SignalDirection: "increase", DisplaySummary: "主信号", DisplayOrder: 1},
					{VariableSignalKey: "support", SignalRole: "supporting", SignalDirection: "uncertain", DisplaySummary: "支持信号", DisplayOrder: 2},
				},
			}},
		},
	}}
	service := newReadTestUseCase(repository, time.Now)

	detail, err := service.GetReasoningTree(context.Background(), themeID, treeID)
	if err != nil {
		t.Fatal(err)
	}
	node := detail.ReasoningTree.Nodes[0]
	if node.PrimarySignal.VariableSignalKey != "primary" || node.SignalDisplaySummary != "支持信号" {
		t.Fatalf("node signal projection = %#v", node)
	}
	if len(detail.ImpactNodeIDs) != 1 || detail.ImpactNodeIDs[0] != nodeID {
		t.Fatalf("impact IDs = %#v", detail.ImpactNodeIDs)
	}
}

func TestServiceKeepsStableReasoningTreeErrors(t *testing.T) {
	themeID := "RTH11111111-1111-4111-8111-111111111111"
	treeID := "RRT22222222-2222-4222-8222-222222222222"
	for _, test := range []struct {
		repositoryError error
		want            error
	}{
		{ErrResearchThemeNotFound, ErrThemeNotFound},
		{ErrResearchReasoningTreesNotFound, ErrReasoningTreesNotFound},
		{ErrResearchReasoningTreeNotFound, ErrReasoningTreeNotFound},
		{ErrResearchReasoningTreeInvariant, ErrReasoningTreeInvariantViolation},
		{errors.New("database unavailable"), ErrRepository},
	} {
		service := newReadTestUseCase(&fakeRepository{err: test.repositoryError}, time.Now)
		_, err := service.GetReasoningTree(context.Background(), themeID, treeID)
		if !errors.Is(err, test.want) {
			t.Fatalf("error = %v, want %v", err, test.want)
		}
	}
}

var _ Repository = (*fakeRepository)(nil)
