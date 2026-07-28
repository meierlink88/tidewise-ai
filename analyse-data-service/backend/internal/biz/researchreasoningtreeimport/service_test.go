package researchreasoningtreeimport

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStore struct{ tx *fakeTransaction }

func (s *fakeStore) InResearchReasoningTreeImportTransaction(ctx context.Context, fn func(Transaction) error) error {
	return fn(s.tx)
}

type fakeTransaction struct {
	receipt         *Receipt
	parent          *ThemePublication
	chains          map[string]struct{}
	memberships     map[string]map[string]struct{}
	graphEdges      map[string]GraphEdgeReference
	snapshots       map[string]SignalSnapshot
	inserted        Receipt
	insertedTrees   []ReasoningTreeRecord
	insertedNodes   []NodeRecord
	insertedSignals []SignalRecord
}

func (f *fakeTransaction) LockResearchReasoningTreeImportTheme(context.Context, string) error {
	return nil
}
func (f *fakeTransaction) LockResearchReasoningTreeAnalysisBatch(context.Context, string) error {
	return nil
}
func (f *fakeTransaction) ResearchReasoningTreeImportReceipt(context.Context, string) (*Receipt, error) {
	return f.receipt, nil
}
func (f *fakeTransaction) ResearchReasoningTreeImportThemePublication(context.Context, string) (*ThemePublication, error) {
	return f.parent, nil
}
func (f *fakeTransaction) ResearchReasoningTreeSignalSnapshots(context.Context, string, []string) (map[string]SignalSnapshot, error) {
	return f.snapshots, nil
}
func (f *fakeTransaction) ExistingResearchReasoningTreeIndustryChains(context.Context, []string) (map[string]struct{}, error) {
	return f.chains, nil
}
func (f *fakeTransaction) ResearchReasoningTreeChainMemberships(context.Context, []string) (map[string]map[string]struct{}, error) {
	return f.memberships, nil
}
func (f *fakeTransaction) ResearchReasoningTreeGraphEdges(context.Context, []string) (map[string]GraphEdgeReference, error) {
	return f.graphEdges, nil
}
func (f *fakeTransaction) InsertResearchReasoningTreeImportReceipt(_ context.Context, value Receipt) error {
	f.inserted = value
	return nil
}
func (f *fakeTransaction) InsertResearchReasoningTree(_ context.Context, value ReasoningTreeRecord) error {
	f.insertedTrees = append(f.insertedTrees, value)
	return nil
}
func (f *fakeTransaction) InsertResearchReasoningTreeEvent(context.Context, EventRecord) error {
	return nil
}
func (f *fakeTransaction) InsertResearchReasoningTreeNode(_ context.Context, value NodeRecord) error {
	f.insertedNodes = append(f.insertedNodes, value)
	return nil
}
func (f *fakeTransaction) InsertResearchReasoningTreeNodeSignal(_ context.Context, value SignalRecord) error {
	f.insertedSignals = append(f.insertedSignals, value)
	return nil
}
func (f *fakeTransaction) VerifyResearchReasoningTreeImportReceipt(context.Context, Receipt) error {
	return nil
}

func TestServicePublishesAndReplaysCompleteReasoningTreeSet(t *testing.T) {
	publication := validPublication()
	tx := validFakeTransaction(publication)
	service := NewService(&fakeStore{tx: tx})
	now := time.Date(2026, 7, 28, 8, 30, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	first, err := service.Import(context.Background(), "analysis-publisher", publication)
	if err != nil {
		t.Fatal(err)
	}
	if first.Replayed || first.Counts != (Counts{ReasoningTrees: 1, Nodes: 1, SignalAssociations: 1, Receipts: 1}) {
		t.Fatalf("first result = %#v", first)
	}
	if len(tx.insertedTrees) != 1 || len(tx.insertedNodes) != 1 || len(tx.insertedSignals) != 1 {
		t.Fatalf("inserted trees=%d nodes=%d signals=%d", len(tx.insertedTrees), len(tx.insertedNodes), len(tx.insertedSignals))
	}

	tx.receipt = &tx.inserted
	replay, err := service.Import(context.Background(), "analysis-publisher", publication)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replayed || replay.ReceiptID != first.ReceiptID || !replay.PublishedAt.Equal(now) {
		t.Fatalf("replay result = %#v", replay)
	}

	changed := publication
	changed.ReasoningTrees = append([]ReasoningTree(nil), publication.ReasoningTrees...)
	changed.ReasoningTrees[0].Title = "changed"
	if _, err := service.Import(context.Background(), "analysis-publisher", changed); !errors.Is(err, ErrPayloadConflict) {
		t.Fatalf("changed payload error = %v", err)
	}
	if _, err := service.Import(context.Background(), "another-publisher", publication); !errors.Is(err, ErrPublisherConflict) {
		t.Fatalf("publisher error = %v", err)
	}
}

func TestServiceRejectsIncompleteImpactCoverageAndSignalDrift(t *testing.T) {
	publication := validPublication()

	t.Run("impact coverage", func(t *testing.T) {
		tx := validFakeTransaction(publication)
		tx.parent.ImpactNodeIDs["cccccccc-cccc-4ccc-8ccc-cccccccccccc"] = struct{}{}
		_, err := NewService(&fakeStore{tx: tx}).Import(context.Background(), "analysis-publisher", publication)
		var contractError *ContractError
		if !errors.As(err, &contractError) {
			t.Fatalf("error = %v, want ContractError", err)
		}
	})

	t.Run("same batch signal snapshot", func(t *testing.T) {
		tx := validFakeTransaction(publication)
		tx.snapshots["signal:port-plan"] = SignalSnapshot{
			SignalDirection: "decrease",
			DisplaySummary:  "端口计划下降",
		}
		_, err := NewService(&fakeStore{tx: tx}).Import(context.Background(), "analysis-publisher", publication)
		var contractError *ContractError
		if !errors.As(err, &contractError) {
			t.Fatalf("error = %v, want ContractError", err)
		}
	})
}

func validFakeTransaction(publication Publication) *fakeTransaction {
	chainID := publication.ReasoningTrees[0].IndustryChainEntityID
	nodeID := publication.ReasoningTrees[0].Nodes[0].ChainNodeEntityID
	return &fakeTransaction{
		parent: &ThemePublication{
			ThemeID: publication.ThemeID, AnalysisBatchID: "analysis-20260728",
			ThemeImportReceiptID: "dddddddd-dddd-4ddd-8ddd-dddddddddddd",
			PublisherSubject:     "analysis-publisher",
			ImpactNodeIDs:        map[string]struct{}{nodeID: {}},
			EventIDs:             map[string]struct{}{},
		},
		chains:      map[string]struct{}{chainID: {}},
		memberships: map[string]map[string]struct{}{chainID: {nodeID: {}}},
		graphEdges:  map[string]GraphEdgeReference{},
		snapshots:   map[string]SignalSnapshot{},
	}
}

var _ Store = (*fakeStore)(nil)
var _ Transaction = (*fakeTransaction)(nil)
