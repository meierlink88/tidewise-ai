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
	testBCIChainID           = "822a8ddc-5ebc-5f03-8ef8-ba9bfba192d9"
	testBCISystemNodeID      = "c38d2f7b-9900-5e81-af06-76393bcc2617"
	testBCITerminalNodeID    = "96336148-76c0-504e-b82e-ac395f8fe268"
	testBCIElectrodeNodeID   = "d3882237-d639-5660-b7d8-aa3563706113"
	testBCITerminalEdgeID    = "300188b0-d01c-5987-ad8a-646067edc7cd"
	testBCIElectrodeEdgeID   = "dc00a16e-0d8e-5db9-9a5d-fbc1fd9a84cf"
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

func TestPublishAcceptsReverseMultiHopAnalystInference(t *testing.T) {
	aggregate, facts := validBCIAnalystInferenceAggregate(true, 3)
	tx := &fakeTransaction{facts: facts}
	service := NewService(fakeStore{tx: tx})

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
	tx := &fakeTransaction{facts: facts}
	service := NewService(fakeStore{tx: tx})

	result, err := service.Publish(context.Background(), "codex", aggregate)
	if err != nil {
		t.Fatalf("publish forward one-hop analyst inference: %v", err)
	}
	if result.Counts.Nodes != 2 {
		t.Fatalf("publication counts = %#v, want forward two-node Tree", result.Counts)
	}
}

func TestPublishAcceptsReverseEntityRelationBetweenAdjacentNodes(t *testing.T) {
	aggregate, facts := validBCIAnalystInferenceAggregate(true, 2)
	relationID := "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee"
	node := &aggregate.ReasoningTrees[0].Nodes[1]
	node.IncomingIndustryChainGraphEdgeID = nil
	node.IncomingLineage.EntityRelationID = &relationID
	node.Signals[0].Lineage.IndustryChainGraphEdgeID = nil
	node.Signals[0].Lineage.EntityRelationID = &relationID
	facts.EntityRelations[relationID] = EntityRelationFact{
		ID: relationID, FromEntityID: testBCITerminalNodeID, ToEntityID: testBCISystemNodeID,
		TemporalFact: testTemporalFact(),
	}
	tx := &fakeTransaction{facts: facts}
	service := NewService(fakeStore{tx: tx})

	if _, err := service.Publish(context.Background(), "codex", aggregate); err != nil {
		t.Fatalf("publish reverse adjacent Entity Relation inference: %v", err)
	}
}

func TestPublishRejectsAnalystInferenceRelationOutsideAdjacentNodes(t *testing.T) {
	aggregate, facts := validBCIAnalystInferenceAggregate(true, 3)
	edge := facts.GraphEdges[testBCIElectrodeEdgeID]
	edge.FromChainNodeEntityID = testBCISystemNodeID
	facts.GraphEdges[testBCIElectrodeEdgeID] = edge
	tx := &fakeTransaction{facts: facts}
	service := NewService(fakeStore{tx: tx})

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
	edge.IndustryChainEntityID = testChainID
	facts.GraphEdges[testBCITerminalEdgeID] = edge
	tx := &fakeTransaction{facts: facts}
	service := NewService(fakeStore{tx: tx})

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
	tx := &fakeTransaction{facts: facts}
	service := NewService(fakeStore{tx: tx})

	_, err := service.Publish(context.Background(), "codex", aggregate)
	var reference *ReferenceError
	if !errors.As(err, &reference) || reference.Reference != testBCITerminalEdgeID {
		t.Fatalf("error = %T %v, want inactive Graph Edge ReferenceError", err, err)
	}
	if tx.writes != 0 {
		t.Fatalf("writes = %d, want no writes for an inactive Graph Edge", tx.writes)
	}
}

func TestPublishRejectsMissingAcceptedRootSignal(t *testing.T) {
	aggregate, facts := validBCIAnalystInferenceAggregate(true, 3)
	delete(facts.Signals, testSignalID)
	tx := &fakeTransaction{facts: facts}
	service := NewService(fakeStore{tx: tx})

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
				ImpactDirection: "positive", DisplayOrder: 1,
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

func validBCIAnalystInferenceAggregate(reverseEdges bool, nodeCount int) (Aggregate, ReferenceFacts) {
	if nodeCount < 2 || nodeCount > 3 {
		panic("BCI analyst inference fixture requires two or three nodes")
	}
	aggregate := validAggregate()
	aggregate.AnalysisBatchID = "bci-reverse-multi-hop"
	aggregate.Theme.ThemeKey = "bci-demand"
	aggregate.Theme.Title = "BCI demand"
	aggregate.Theme.Impacts = nil
	aggregate.ReasoningTrees[0].IndustryChainEntityID = testBCIChainID
	aggregate.ReasoningTrees[0].Title = "BCI system chain"
	aggregate.ReasoningTrees[0].Nodes = nil

	nodeIDs := []string{testBCISystemNodeID, testBCITerminalNodeID, testBCIElectrodeNodeID}
	edgeIDs := []string{testBCITerminalEdgeID, testBCIElectrodeEdgeID}
	signalKeys := []string{"market_demand", "terminal_market_demand", "electrode_market_demand"}
	for position, nodeID := range nodeIDs[:nodeCount] {
		aggregate.Theme.Impacts = append(aggregate.Theme.Impacts, researchthemeimport.Impact{
			ChainNodeEntityID: nodeID,
			RelationRole:      "exposure",
			ImpactDirection:   "uncertain",
			DisplayOrder:      position + 1,
		})
		node := Node{
			Position:          position + 1,
			ChainNodeEntityID: nodeID,
			ImpactDirection:   "uncertain",
			ImpactStrength:    "unknown",
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
			GraphEdgeReference: researchreasoningtreeimport.GraphEdgeReference{
				ID: edgeID, IndustryChainEntityID: testBCIChainID,
				FromChainNodeEntityID: fromNodeID, ToChainNodeEntityID: toNodeID,
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

type fakeStore struct{ tx *fakeTransaction }

func (s fakeStore) InResearchPublicationTransaction(ctx context.Context, fn func(Transaction) error) error {
	return fn(s.tx)
}

type fakeTransaction struct {
	facts  ReferenceFacts
	writes int
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
func (f *fakeTransaction) InsertThemeImpact(context.Context, researchthemeimport.ImpactRecord) error {
	f.writes++
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
