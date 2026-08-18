package research

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	testEventID       = "EVT11111111-1111-4111-8111-111111111111"
	testEvidenceID    = "EEL66666666-6666-4666-8666-666666666666"
	testImpactEventID = "EVT99999999-9999-4999-8999-999999999999"
)

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
func (f *publicationTransactionStub) InsertSnapshotThemeImpact(context.Context, SnapshotImpactRecord) error {
	f.writes++
	return nil
}
func (f *publicationTransactionStub) InsertThemeEvent(context.Context, PublicationThemeEventRecord) error {
	f.writes++
	return nil
}
func (f *publicationTransactionStub) InsertSnapshotTreeReceipt(context.Context, SnapshotTreeReceipt) error {
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
func (f *publicationTransactionStub) InsertSnapshotNode(context.Context, SnapshotNodeRecord) error {
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
	if len(tx.lastReferenceQuery.EventIDs) == 0 || len(tx.lastReferenceQuery.EvidenceIDs) != 0 {
		t.Fatalf("snapshot queried unexpected references: %#v", tx.lastReferenceQuery)
	}
	for _, eventID := range tx.lastReferenceQuery.EventIDs {
		if eventID != testEventID {
			t.Fatalf("snapshot queried unexpected Event %q", eventID)
		}
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

func TestServiceMapsSnapshotReasoningTreeSignalsWithoutChoosingImpactPriority(t *testing.T) {
	themeID := "RTH11111111-1111-4111-8111-111111111111"
	treeID := "RRT22222222-2222-4222-8222-222222222222"
	nodeID := "RRN33333333-3333-4333-8333-333333333333"
	repository := &fakeRepository{reasoningTree: ReasoningTreeDetailRecord{
		ThemeID:       themeID,
		ImpactNodeIDs: []string{nodeID},
		ReasoningTree: ReasoningTreeRecord{
			ReasoningTreeID: treeID, ThemeID: themeID,
			TreeKey: "tree", DisplayName: "产业链", Title: "Tree", DisplayOrder: 1,
			OneLineConclusion: "结论", ImpactDirection: "positive", ImpactStrength: "medium",
			PublishedAt: time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC),
			Nodes: []ReasoningTreeNodeRecord{{
				ID: nodeID, NodeKey: "node", DisplayName: "节点", Position: 1,
				ImpactDirection: "positive", ImpactStrength: "medium",
				Signals: []SignalRecord{
					{SignalKey: "primary", SignalRole: "primary", Direction: stringPointer("increase"), DisplaySummary: "主信号", DisplayOrder: 1},
					{SignalKey: "support", SignalRole: "supporting", Direction: stringPointer("uncertain"), DisplaySummary: "支持信号", DisplayOrder: 2},
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
	if node.PrimarySignal.SignalKey != "primary" || node.SignalDisplaySummary != "支持信号" {
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
