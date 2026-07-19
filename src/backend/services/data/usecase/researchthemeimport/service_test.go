package researchthemeimport

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	domainimport "github.com/meierlink88/tidewise-ai/backend/services/data/domain/researchthemeimport"
	"github.com/meierlink88/tidewise-ai/backend/services/data/repositories"
)

func TestServiceRejectsInvalidPublisherSubjectBeforeTransaction(t *testing.T) {
	for _, publisher := range []string{"", strings.Repeat("a", 201)} {
		store := newFakeStore()
		if _, err := NewService(store).Import(context.Background(), publisher, validBatch()); err == nil {
			t.Fatalf("publisher length %d error = nil", len(publisher))
		}
		if store.commits != 0 || store.rollbacks != 0 {
			t.Fatalf("invalid publisher reached transaction: commits=%d rollbacks=%d", store.commits, store.rollbacks)
		}
	}
}

func TestServicePublishesWholeBatchWithOneTimestamp(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 7, 19, 9, 30, 0, 0, time.UTC)
	service := NewService(store)
	service.now = func() time.Time { return now }

	result, err := service.Import(context.Background(), "agent-run", validBatch())
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.AnalysisBatchID != validBatch().AnalysisBatchID || result.Replayed {
		t.Fatalf("result = %#v", result)
	}
	if result.PublishedAt != now || result.ImportedAt != now {
		t.Fatalf("published/imported = %s/%s, want %s", result.PublishedAt, result.ImportedAt, now)
	}
	if got := result.ThemeIDsByKey[validBatch().Themes[0].ThemeKey]; got != domainimport.ThemeID(validBatch().AnalysisBatchID, validBatch().Themes[0].ThemeKey) {
		t.Fatalf("theme ID = %q", got)
	}
	wantCounts := Counts{Themes: 1, ChainNodeAssociations: 1, EventAssociations: 1, Receipts: 1}
	if result.Counts != wantCounts {
		t.Fatalf("counts = %#v, want %#v", result.Counts, wantCounts)
	}
	if len(store.tx.themes) != 1 || len(store.tx.chainNodes) != 1 || len(store.tx.events) != 1 || store.tx.receipt == nil {
		t.Fatalf("persisted themes/nodes/events/receipt = %d/%d/%d/%v", len(store.tx.themes), len(store.tx.chainNodes), len(store.tx.events), store.tx.receipt != nil)
	}
	if store.commits != 1 || store.rollbacks != 0 {
		t.Fatalf("commit/rollback = %d/%d", store.commits, store.rollbacks)
	}
}

func TestServiceReplaysSamePublisherAndPayload(t *testing.T) {
	store := newFakeStore()
	service := NewService(store)
	first, err := service.Import(context.Background(), "agent-run", validBatch())
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Import(context.Background(), "agent-run", validBatch())
	if err != nil {
		t.Fatal(err)
	}
	if !second.Replayed {
		t.Fatal("replay result is not marked replayed")
	}
	second.Replayed = false
	if !reflect.DeepEqual(second, first) {
		t.Fatalf("replay = %#v, want %#v", second, first)
	}
	if len(store.tx.themes) != 1 || len(store.tx.chainNodes) != 1 || len(store.tx.events) != 1 {
		t.Fatal("replay wrote duplicate business rows")
	}
}

func TestServiceRejectsBatchOwnershipAndPayloadConflicts(t *testing.T) {
	store := newFakeStore()
	service := NewService(store)
	if _, err := service.Import(context.Background(), "agent-run", validBatch()); err != nil {
		t.Fatal(err)
	}

	if _, err := service.Import(context.Background(), "other-publisher", validBatch()); !errors.Is(err, ErrPublisherConflict) {
		t.Fatalf("publisher conflict = %v, want ErrPublisherConflict", err)
	}
	changed := validBatch()
	changed.Themes[0].Name = "changed"
	if _, err := service.Import(context.Background(), "agent-run", changed); !errors.Is(err, ErrPayloadConflict) {
		t.Fatalf("payload conflict = %v, want ErrPayloadConflict", err)
	}
}

func TestServiceRejectsMissingReferencesAndRollsBackWholeBatch(t *testing.T) {
	store := newFakeStore()
	delete(store.tx.chainNodeIDs, validBatch().Themes[0].ChainNodes[0].ChainNodeID)
	service := NewService(store)

	_, err := service.Import(context.Background(), "agent-run", validBatch())
	var referenceError *ReferenceError
	if !errors.As(err, &referenceError) {
		t.Fatalf("Import() error = %T %v, want ReferenceError", err, err)
	}
	if referenceError.ThemeKey != validBatch().Themes[0].ThemeKey || referenceError.Path != "themes[0].chain_nodes[0].chain_node_id" {
		t.Fatalf("reference error = %#v", referenceError)
	}
	if len(store.tx.themes) != 0 || store.tx.receipt != nil || store.commits != 0 || store.rollbacks != 1 {
		t.Fatalf("partial state themes/receipt/commit/rollback = %d/%v/%d/%d", len(store.tx.themes), store.tx.receipt != nil, store.commits, store.rollbacks)
	}
}

func validBatch() domainimport.Batch {
	return domainimport.Batch{
		AnalysisBatchID: "20260718T-v6-72h-validation",
		WindowStart:     "2026-07-15T00:00:00Z",
		WindowEnd:       "2026-07-18T00:00:00Z",
		Themes: []domainimport.Theme{{
			ThemeKey:                  "theme:ai-semiconductor-expansion",
			Name:                      "AI算力扩产与半导体",
			OneLineConclusion:         "晶圆扩产增强但卡点与价格背离",
			ImpactLevel:               "high",
			TransmissionPath:          "AI芯片采购 → 晶圆扩产",
			TradingDirection:          "优先研究设备和材料",
			TransmissionStage:         "validation",
			NextCheckpoint:            "重点跟踪订单和交期",
			MarketConfirmationSummary: "当前没有可归属的正式市场观测",
			ChainNodes: []domainimport.ChainNode{{
				ChainNodeID: "11111111-1111-4111-8111-111111111111", RelationRole: "driver", ImpactSummary: "需求驱动",
			}},
			Events: []domainimport.Event{{
				EventID: "22222222-2222-4222-8222-222222222222", EvidenceRole: "driver", SupportedClaim: "支持扩产判断",
			}},
		}},
	}
}

type fakeStore struct {
	tx        *fakeTx
	commits   int
	rollbacks int
}

func newFakeStore() *fakeStore {
	return &fakeStore{tx: &fakeTx{
		chainNodeIDs: map[string]struct{}{validBatch().Themes[0].ChainNodes[0].ChainNodeID: {}},
		eventIDs:     map[string]struct{}{validBatch().Themes[0].Events[0].EventID: {}},
	}}
}

func (s *fakeStore) InResearchThemeImportTransaction(_ context.Context, fn func(Transaction) error) error {
	beforeThemes := len(s.tx.themes)
	beforeNodes := len(s.tx.chainNodes)
	beforeEvents := len(s.tx.events)
	beforeReceipt := s.tx.receipt
	if err := fn(s.tx); err != nil {
		s.tx.themes = s.tx.themes[:beforeThemes]
		s.tx.chainNodes = s.tx.chainNodes[:beforeNodes]
		s.tx.events = s.tx.events[:beforeEvents]
		s.tx.receipt = beforeReceipt
		s.rollbacks++
		return err
	}
	s.commits++
	return nil
}

type fakeTx struct {
	chainNodeIDs map[string]struct{}
	eventIDs     map[string]struct{}
	themes       []repositories.ResearchThemeImportTheme
	chainNodes   []repositories.ResearchThemeImportChainNode
	events       []repositories.ResearchThemeImportEvent
	receipt      *Receipt
}

func (f *fakeTx) LockResearchThemeImportBatch(context.Context, string) error { return nil }

func (f *fakeTx) ResearchThemeImportReceipt(_ context.Context, batchID string) (*Receipt, error) {
	if f.receipt == nil || f.receipt.AnalysisBatchID != batchID {
		return nil, nil
	}
	copy := *f.receipt
	copy.ThemeIDsByKey = cloneStringMap(f.receipt.ThemeIDsByKey)
	return &copy, nil
}

func (f *fakeTx) ExistingResearchThemeChainNodes(_ context.Context, _ []string) (map[string]struct{}, error) {
	return f.chainNodeIDs, nil
}

func (f *fakeTx) ExistingResearchThemeEvents(_ context.Context, _ []string) (map[string]struct{}, error) {
	return f.eventIDs, nil
}

func (f *fakeTx) InsertResearchTheme(_ context.Context, theme repositories.ResearchThemeImportTheme) error {
	f.themes = append(f.themes, theme)
	return nil
}

func (f *fakeTx) InsertResearchThemeChainNode(_ context.Context, node repositories.ResearchThemeImportChainNode) error {
	f.chainNodes = append(f.chainNodes, node)
	return nil
}

func (f *fakeTx) InsertResearchThemeEvent(_ context.Context, event repositories.ResearchThemeImportEvent) error {
	f.events = append(f.events, event)
	return nil
}

func (f *fakeTx) InsertResearchThemeImportReceipt(_ context.Context, receipt Receipt) error {
	copy := receipt
	copy.ThemeIDsByKey = cloneStringMap(receipt.ThemeIDsByKey)
	f.receipt = &copy
	return nil
}

func (f *fakeTx) VerifyResearchThemeImportReceipt(context.Context, Receipt) error { return nil }
