package researchthemeimport

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeThemeStore struct{ tx *fakeThemeTransaction }

func (s *fakeThemeStore) InResearchThemeImportTransaction(ctx context.Context, fn func(Transaction) error) error {
	return fn(s.tx)
}

type fakeThemeTransaction struct {
	receipt         *Receipt
	nodes           map[string]struct{}
	events          map[string]struct{}
	inserted        Receipt
	insertedThemes  []ThemeRecord
	insertedImpacts []ImpactRecord
	verifyCalls     int
}

func (f *fakeThemeTransaction) LockResearchThemeImportBatch(context.Context, string) error {
	return nil
}
func (f *fakeThemeTransaction) ResearchThemeImportReceipt(context.Context, string) (*Receipt, error) {
	return f.receipt, nil
}
func (f *fakeThemeTransaction) ExistingResearchThemeImpactNodes(context.Context, []string) (map[string]struct{}, error) {
	return f.nodes, nil
}
func (f *fakeThemeTransaction) ExistingResearchThemeEvents(context.Context, []string) (map[string]struct{}, error) {
	return f.events, nil
}
func (f *fakeThemeTransaction) InsertResearchTheme(_ context.Context, value ThemeRecord) error {
	f.insertedThemes = append(f.insertedThemes, value)
	return nil
}
func (f *fakeThemeTransaction) InsertResearchThemeImpact(_ context.Context, value ImpactRecord) error {
	f.insertedImpacts = append(f.insertedImpacts, value)
	return nil
}
func (f *fakeThemeTransaction) InsertResearchThemeEvent(context.Context, EventRecord) error {
	return nil
}
func (f *fakeThemeTransaction) InsertResearchThemeImportReceipt(_ context.Context, value Receipt) error {
	f.inserted = value
	return nil
}
func (f *fakeThemeTransaction) VerifyResearchThemeImportReceipt(context.Context, Receipt) error {
	f.verifyCalls++
	return nil
}

func TestServicePublishesReplaysAndProtectsThemeBatchIdentity(t *testing.T) {
	batch := validThemeBatch()
	nodeID := batch.Themes[0].Impacts[0].ChainNodeEntityID
	tx := &fakeThemeTransaction{
		nodes:  map[string]struct{}{nodeID: {}},
		events: map[string]struct{}{},
	}
	service := NewService(&fakeThemeStore{tx: tx})
	now := time.Date(2026, 7, 28, 8, 0, 0, 123456789, time.UTC)
	persistedTime := now.Truncate(time.Microsecond)
	service.now = func() time.Time { return now }

	first, err := service.Import(context.Background(), "analysis-publisher", batch)
	if err != nil {
		t.Fatal(err)
	}
	if first.Replayed || first.Counts != (Counts{Themes: 1, Impacts: 1, Receipts: 1}) {
		t.Fatalf("first result = %#v", first)
	}
	if !first.PublishedAt.Equal(persistedTime) {
		t.Fatalf("first published_at = %s, want %s", first.PublishedAt, persistedTime)
	}
	if len(tx.insertedThemes) != 1 || len(tx.insertedImpacts) != 1 {
		t.Fatalf("inserted themes=%d impacts=%d", len(tx.insertedThemes), len(tx.insertedImpacts))
	}
	if tx.verifyCalls != 1 {
		t.Fatalf("first publication verify calls = %d", tx.verifyCalls)
	}

	tx.receipt = &tx.inserted
	replay, err := service.Import(context.Background(), "analysis-publisher", batch)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replayed || replay.ReceiptID != first.ReceiptID || !replay.PublishedAt.Equal(persistedTime) {
		t.Fatalf("replay result = %#v", replay)
	}
	if tx.verifyCalls != 2 {
		t.Fatalf("replay verify calls = %d", tx.verifyCalls)
	}

	changed := batch
	changed.Themes = append([]Theme(nil), batch.Themes...)
	changed.Themes[0].Title = "changed"
	if _, err := service.Import(context.Background(), "analysis-publisher", changed); !errors.Is(err, ErrPayloadConflict) {
		t.Fatalf("changed payload error = %v", err)
	}
	if _, err := service.Import(context.Background(), "another-publisher", batch); !errors.Is(err, ErrPublisherConflict) {
		t.Fatalf("publisher error = %v", err)
	}
}

func TestServiceRejectsMissingFormalThemeImpact(t *testing.T) {
	batch := validThemeBatch()
	service := NewService(&fakeThemeStore{tx: &fakeThemeTransaction{
		nodes:  map[string]struct{}{},
		events: map[string]struct{}{},
	}})

	_, err := service.Import(context.Background(), "analysis-publisher", batch)
	var referenceError *ReferenceError
	if !errors.As(err, &referenceError) {
		t.Fatalf("error = %v, want ReferenceError", err)
	}
}

func validThemeBatch() Batch {
	return Batch{
		AnalysisBatchID: "analysis-20260728",
		AnalysisAsOf:    "2026-07-28T08:00:00Z",
		WindowStart:     "2026-07-27T00:00:00Z",
		WindowEnd:       "2026-07-28T00:00:00Z",
		Themes: []Theme{{
			ThemeKey:                  "theme:optical-demand",
			Title:                     "高速光模块需求验证",
			OneLineConclusion:         "端口计划上调可能增强需求",
			ConclusionDirection:       "positive",
			ImpactStrength:            "medium",
			TransmissionStage:         "validation",
			InvestmentGuidanceAction:  "focus",
			InvestmentGuidanceSummary: "关注采购订单",
			TimeHorizonCategory:       "medium_term",
			Impacts: []Impact{{
				ChainNodeEntityID: "11111111-1111-4111-8111-111111111111",
				RelationRole:      "beneficiary",
				ImpactDirection:   "positive",
				DisplayOrder:      1,
			}},
			Events: []Event{},
		}},
	}
}

var _ Store = (*fakeThemeStore)(nil)
var _ Transaction = (*fakeThemeTransaction)(nil)
