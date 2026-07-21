package researchanchorimport

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	domainimport "github.com/meierlink88/tidewise-ai/backend/services/data/domain/researchanchorimport"
	"github.com/meierlink88/tidewise-ai/backend/services/data/repositories"
)

func TestServicePublishesCompleteThemeAtomically(t *testing.T) {
	publication := sharedPublication(t)
	store := newFakeStore(publication)
	now := time.Date(2026, 7, 20, 9, 0, 0, 123456789, time.UTC)
	wantPersistedTime := now.Truncate(time.Microsecond)
	service := NewService(store)
	service.now = func() time.Time { return now }

	result, err := service.Import(context.Background(), "service:ai-research-analyst", publication)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.ThemeID != publication.ThemeID || result.Replayed {
		t.Fatalf("result = %#v", result)
	}
	if result.PublishedAt != wantPersistedTime || result.ImportedAt != wantPersistedTime {
		t.Fatalf("published/imported = %s/%s, want %s", result.PublishedAt, result.ImportedAt, wantPersistedTime)
	}
	wantCounts := Counts{Anchors: 2, EventAssociations: 4, PathNodes: 4, Receipts: 1}
	if result.Counts != wantCounts {
		t.Fatalf("counts = %#v, want %#v", result.Counts, wantCounts)
	}
	if len(store.tx.anchors) != 2 || len(store.tx.events) != 4 || len(store.tx.pathNodes) != 4 || store.tx.receipt == nil {
		t.Fatalf("persisted anchors/events/path/receipt = %d/%d/%d/%v", len(store.tx.anchors), len(store.tx.events), len(store.tx.pathNodes), store.tx.receipt != nil)
	}
	if got := store.tx.anchors[0].SupportSummary; got != publication.Anchors[0].SupportSummary {
		t.Fatalf("persisted support summary = %q, want %q", got, publication.Anchors[0].SupportSummary)
	}
	if got := store.tx.anchors[0].CounterSummary; !reflect.DeepEqual(got, publication.Anchors[0].CounterSummary) {
		t.Fatalf("persisted counter summary = %#v, want %#v", got, publication.Anchors[0].CounterSummary)
	}
	if got := store.tx.anchors[1].CounterSummary; got != nil {
		t.Fatalf("persisted no-contradiction counter summary = %#v, want nil", got)
	}
	if store.commits != 1 || store.rollbacks != 0 {
		t.Fatalf("commit/rollback = %d/%d", store.commits, store.rollbacks)
	}
}

func TestServiceReplaysSamePublisherAndPayloadWithoutDuplicateWrites(t *testing.T) {
	publication := sharedPublication(t)
	store := newFakeStore(publication)
	service := NewService(store)

	first, err := service.Import(context.Background(), "service:ai-research-analyst", publication)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Import(context.Background(), "service:ai-research-analyst", publication)
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
	if len(store.tx.anchors) != 2 || len(store.tx.events) != 4 || len(store.tx.pathNodes) != 4 {
		t.Fatal("replay wrote duplicate business rows")
	}
}

func TestServiceRejectsPayloadAndPublisherConflicts(t *testing.T) {
	publication := sharedPublication(t)
	store := newFakeStore(publication)
	service := NewService(store)
	if _, err := service.Import(context.Background(), "service:ai-research-analyst", publication); err != nil {
		t.Fatal(err)
	}

	if _, err := service.Import(context.Background(), "service:other", publication); !errors.Is(err, ErrPublisherConflict) {
		t.Fatalf("publisher conflict = %v, want ErrPublisherConflict", err)
	}
	changed := publication
	changed.Anchors = append([]domainimport.Anchor(nil), publication.Anchors...)
	changed.Anchors[0].FactSummary = "changed"
	if _, err := service.Import(context.Background(), "service:ai-research-analyst", changed); !errors.Is(err, ErrPayloadConflict) {
		t.Fatalf("payload conflict = %v, want ErrPayloadConflict", err)
	}
}

func TestServiceRejectsLegacyThemeWithoutPublicationReceipt(t *testing.T) {
	publication := sharedPublication(t)
	store := newFakeStore(publication)
	store.tx.theme.ThemeImportReceiptID = ""

	_, err := NewService(store).Import(context.Background(), "service:ai-research-analyst", publication)
	var referenceError *ReferenceError
	if !errors.As(err, &referenceError) || referenceError.Kind != ReferenceInvalid || referenceError.Path != "theme_id" {
		t.Fatalf("Import() error = %#v, want invalid theme_id reference", err)
	}
	if store.commits != 0 || store.rollbacks != 1 || store.tx.receipt != nil {
		t.Fatalf("commit/rollback/receipt = %d/%d/%v", store.commits, store.rollbacks, store.tx.receipt)
	}
}

func TestServiceReturnsOnlyFirstDeterministicReferenceErrorAndRollsBack(t *testing.T) {
	publication := sharedPublication(t)
	store := newFakeStore(publication)
	firstCenter := publication.Anchors[0].CenterChainNodeID
	firstEvent := publication.Anchors[0].Events[0].EventID
	delete(store.tx.existingNodeIDs, firstCenter)
	delete(store.tx.existingEventIDs, firstEvent)

	_, err := NewService(store).Import(context.Background(), "service:ai-research-analyst", publication)
	var referenceError *ReferenceError
	if !errors.As(err, &referenceError) {
		t.Fatalf("Import() error = %T %v, want ReferenceError", err, err)
	}
	if referenceError.Kind != ReferenceNotFound || referenceError.Path != "anchors[0].center_chain_node_id" || referenceError.Reference != firstCenter {
		t.Fatalf("first reference error = %#v", referenceError)
	}
	if len(store.tx.anchors) != 0 || store.tx.receipt != nil || store.commits != 0 || store.rollbacks != 1 {
		t.Fatalf("partial state anchors/receipt/commit/rollback = %d/%v/%d/%d", len(store.tx.anchors), store.tx.receipt != nil, store.commits, store.rollbacks)
	}
}

func TestServiceRejectsIncompleteCenterCoverage(t *testing.T) {
	publication := sharedPublication(t)
	store := newFakeStore(publication)
	publication.Anchors = publication.Anchors[:1]

	_, err := NewService(store).Import(context.Background(), "service:ai-research-analyst", publication)
	var contractError *ContractError
	if !errors.As(err, &contractError) || contractError.Path != "anchors" || contractError.Reference != "33333333-3333-4333-8333-333333333333" {
		t.Fatalf("Import() error = %#v, want missing center coverage", err)
	}
}

func TestServiceRollsBackReceiptAndFirstAnchorWhenLaterInsertFails(t *testing.T) {
	publication := sharedPublication(t)
	store := newFakeStore(publication)
	store.tx.failAnchorAfter = 1

	if _, err := NewService(store).Import(context.Background(), "service:ai-research-analyst", publication); err == nil {
		t.Fatal("Import() error = nil, want synthetic second Anchor failure")
	}
	if store.commits != 0 || store.rollbacks != 1 || store.tx.receipt != nil || len(store.tx.anchors) != 0 || len(store.tx.events) != 0 || len(store.tx.pathNodes) != 0 {
		t.Fatalf("partial state commit/rollback/receipt/anchors/events/path = %d/%d/%v/%d/%d/%d", store.commits, store.rollbacks, store.tx.receipt != nil, len(store.tx.anchors), len(store.tx.events), len(store.tx.pathNodes))
	}
}

func sharedPublication(t *testing.T) domainimport.Publication {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "..", "testdata", "reasoning-tree-v1", "01-multi-anchor-import-request.json")
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	publication, err := domainimport.DecodeStrict(file)
	if err != nil {
		t.Fatal(err)
	}
	return publication
}

type fakeStore struct {
	tx        *fakeTx
	commits   int
	rollbacks int
}

func newFakeStore(publication domainimport.Publication) *fakeStore {
	nodes := make(map[string]struct{})
	events := make(map[string]struct{})
	centers := make(map[string]struct{})
	themeEvents := make(map[string]struct{})
	for _, anchor := range publication.Anchors {
		centers[anchor.CenterChainNodeID] = struct{}{}
		for _, event := range anchor.Events {
			events[event.EventID] = struct{}{}
			themeEvents[event.EventID] = struct{}{}
		}
		for _, node := range anchor.PathNodes {
			nodes[node.ChainNodeID] = struct{}{}
		}
	}
	return &fakeStore{tx: &fakeTx{
		theme: &repositories.ResearchAnchorImportThemePublication{
			ThemeID: publication.ThemeID, ThemeImportReceiptID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
			PublisherSubject: "service:ai-research-analyst",
		},
		centerIDs: centers, themeEventIDs: themeEvents, existingNodeIDs: nodes, existingEventIDs: events,
	}}
}

func (s *fakeStore) InResearchAnchorImportTransaction(_ context.Context, fn func(Transaction) error) error {
	beforeAnchors, beforeEvents, beforePath := len(s.tx.anchors), len(s.tx.events), len(s.tx.pathNodes)
	beforeReceipt := s.tx.receipt
	if err := fn(s.tx); err != nil {
		s.tx.anchors = s.tx.anchors[:beforeAnchors]
		s.tx.events = s.tx.events[:beforeEvents]
		s.tx.pathNodes = s.tx.pathNodes[:beforePath]
		s.tx.receipt = beforeReceipt
		s.rollbacks++
		return err
	}
	s.commits++
	return nil
}

type fakeTx struct {
	theme            *repositories.ResearchAnchorImportThemePublication
	centerIDs        map[string]struct{}
	themeEventIDs    map[string]struct{}
	existingNodeIDs  map[string]struct{}
	existingEventIDs map[string]struct{}
	receipt          *Receipt
	anchors          []repositories.ResearchAnchorImportAnchor
	events           []repositories.ResearchAnchorImportEvent
	pathNodes        []repositories.ResearchAnchorImportPathNode
	failAnchorAfter  int
}

func (f *fakeTx) LockResearchAnchorImportTheme(context.Context, string) error { return nil }
func (f *fakeTx) ResearchAnchorImportReceipt(context.Context, string) (*Receipt, error) {
	return f.receipt, nil
}
func (f *fakeTx) ResearchAnchorImportThemePublication(context.Context, string) (*repositories.ResearchAnchorImportThemePublication, error) {
	return f.theme, nil
}
func (f *fakeTx) ResearchAnchorImportThemeChainNodes(context.Context, string) (map[string]struct{}, error) {
	return f.centerIDs, nil
}
func (f *fakeTx) ResearchAnchorImportThemeEvents(context.Context, string) (map[string]struct{}, error) {
	return f.themeEventIDs, nil
}
func (f *fakeTx) ExistingResearchAnchorChainNodes(context.Context, []string) (map[string]struct{}, error) {
	return f.existingNodeIDs, nil
}
func (f *fakeTx) ExistingResearchAnchorEvents(context.Context, []string) (map[string]struct{}, error) {
	return f.existingEventIDs, nil
}
func (f *fakeTx) InsertResearchAnchorImportReceipt(_ context.Context, receipt Receipt) error {
	copy := receipt
	f.receipt = &copy
	return nil
}
func (f *fakeTx) InsertResearchAnchor(_ context.Context, anchor repositories.ResearchAnchorImportAnchor) error {
	if f.failAnchorAfter > 0 && len(f.anchors) >= f.failAnchorAfter {
		return errors.New("synthetic Anchor insert failure")
	}
	f.anchors = append(f.anchors, anchor)
	return nil
}
func (f *fakeTx) InsertResearchAnchorEvent(_ context.Context, event repositories.ResearchAnchorImportEvent) error {
	f.events = append(f.events, event)
	return nil
}
func (f *fakeTx) InsertResearchAnchorPathNode(_ context.Context, node repositories.ResearchAnchorImportPathNode) error {
	f.pathNodes = append(f.pathNodes, node)
	return nil
}
func (f *fakeTx) VerifyResearchAnchorImportReceipt(context.Context, Receipt) error { return nil }
