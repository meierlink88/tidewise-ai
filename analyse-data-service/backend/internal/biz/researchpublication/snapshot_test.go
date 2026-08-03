package researchpublication

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

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

func TestUATPreparedSnapshotFixtureIsPublishableWithoutFormalOntologyIDs(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "testdata", "research-theme-analyst-snapshot-v3", "01-uat-at01-prepared-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	var aggregate SnapshotAggregate
	if err := json.Unmarshal(payload, &aggregate); err != nil {
		t.Fatal(err)
	}
	if _, _, err := aggregate.Validate(); err != nil {
		t.Fatalf("prepared UAT fixture Validate() error = %v", err)
	}
	if len(aggregate.ReasoningTrees) != 2 || aggregate.ReasoningTrees[0].Nodes[2].DisplayName == aggregate.Theme.Impacts[0].DisplayName {
		t.Fatalf("fixture did not preserve distinct Theme/Tree presentation snapshots: %#v", aggregate)
	}
}

func TestPublishSnapshotUsesOnlyEventReferencesAndReplays(t *testing.T) {
	aggregate := validSnapshotAggregate()
	tx := &fakeTransaction{facts: ReferenceFacts{
		Events: map[string]EventFact{testEventID: {ID: testEventID}},
	}}
	service := NewService(fakeStore{tx: tx})

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
	tx := &fakeTransaction{facts: ReferenceFacts{Events: map[string]EventFact{
		"71000000-0000-5000-8000-000000000001": {ID: "71000000-0000-5000-8000-000000000001"},
		"71000000-0000-5000-8000-000000000002": {ID: "71000000-0000-5000-8000-000000000002"},
		"71000000-0000-5000-8000-000000000003": {ID: "71000000-0000-5000-8000-000000000003"},
	}}}
	service := NewService(fakeStore{tx: tx})

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
	tx := &fakeTransaction{facts: ReferenceFacts{Events: map[string]EventFact{
		"71000000-0000-5000-8000-000000000001": {ID: "71000000-0000-5000-8000-000000000001"},
		"71000000-0000-5000-8000-000000000003": {ID: "71000000-0000-5000-8000-000000000003"},
	}}}

	_, err := NewService(fakeStore{tx: tx}).PublishSnapshot(context.Background(), "theme-analyst", aggregate)
	var reference *ReferenceError
	if !errors.As(err, &reference) || reference.Path != "theme.events[2].event_id" ||
		reference.Reference != "71000000-0000-5000-8000-000000000002" {
		t.Fatalf("PublishSnapshot() error = %T %v, want missing third Event ReferenceError", err, err)
	}
	if tx.writes != 0 {
		t.Fatalf("writes = %d, want 0", tx.writes)
	}
}

func TestPublishSnapshotRejectsChangedPayloadForExistingBatch(t *testing.T) {
	aggregate := validSnapshotAggregate()
	tx := &fakeTransaction{facts: ReferenceFacts{
		Events: map[string]EventFact{testEventID: {ID: testEventID}},
	}}
	service := NewService(fakeStore{tx: tx})
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
	tx := &fakeTransaction{facts: ReferenceFacts{
		Events: map[string]EventFact{testEventID: {ID: testEventID}},
		Evidences: map[string]EvidenceFact{testEvidenceID: {
			ID: testEvidenceID, EventID: testImpactEventID,
		}},
	}}

	_, err := NewService(fakeStore{tx: tx}).PublishSnapshot(context.Background(), "theme-analyst", aggregate)
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
				EventID: "11111111-1111-4111-8111-111111111111", EvidenceRole: "driver",
			}},
		},
		ReasoningTrees: []SnapshotReasoningTree{{
			TreeKey: "tree:chip-commercialization", DisplayName: "先进芯片商业化路径",
			Title: "先进芯片商业化路径", DisplayOrder: 1,
			OneLineConclusion: "完成流片不等于商业兑现。",
			ImpactDirection:   "uncertain", ImpactStrength: "unknown",
			Events: []SnapshotTreeEvent{{
				EventID:      "11111111-1111-4111-8111-111111111111",
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
		{EventID: "71000000-0000-5000-8000-000000000001", EvidenceRole: "driver"},
		{EventID: "71000000-0000-5000-8000-000000000003", EvidenceRole: "supporting"},
		{EventID: "71000000-0000-5000-8000-000000000002", EvidenceRole: "context"},
	}
	aggregate.ReasoningTrees[0].Events = []SnapshotTreeEvent{
		{EventID: "71000000-0000-5000-8000-000000000001", EvidenceRole: "driver", DisplayOrder: 1},
		{EventID: "71000000-0000-5000-8000-000000000003", EvidenceRole: "supporting", DisplayOrder: 2},
		{EventID: "71000000-0000-5000-8000-000000000002", EvidenceRole: "context", DisplayOrder: 3},
	}
	return aggregate
}
