package research

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	entitybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity"
	eventbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/event"
	eventsemanticbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/eventsemantic"
	researchbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/research"
	entitydata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity"
	eventdata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/event"
	eventsemanticdata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/eventsemantic"
	"github.com/pressly/goose/v3"
	"net/url"
)

func TestResearchThemeAdapterRejectsMalformedPersistedRows(t *testing.T) {
	now := time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC)
	valid := ResearchThemeSummary{
		ID: "RTH11111111-1111-4111-8111-111111111111", AnalysisBatchID: "batch:one",
		Title: "Theme", OneLineConclusion: "Conclusion", ConclusionDirection: "positive",
		ImpactStrength: "medium", TransmissionStage: "validation", InvestmentGuidanceAction: "observe",
		InvestmentGuidanceSummary: "Observe", TimeHorizonCategory: "medium_term",
		AnalysisAsOf: now, WindowStart: now.Add(-time.Hour), WindowEnd: now, PublishedAt: now,
		Impacts: []ResearchThemeImpact{{NodeKey: "node:one", DisplayName: "Node", RelationRole: "driver", ImpactDirection: "positive", DisplayOrder: 1}},
	}
	if err := validatePersistedResearchThemeSummary(valid); err != nil {
		t.Fatalf("valid persisted Theme rejected: %v", err)
	}
	invalid := valid
	invalid.ConclusionDirection = "invented"
	if err := validatePersistedResearchThemeSummary(invalid); err == nil {
		t.Fatal("malformed persisted Theme enum was accepted")
	}
	invalid = valid
	invalid.Impacts = append(append([]ResearchThemeImpact(nil), valid.Impacts...), valid.Impacts[0])
	invalid.Impacts[1].DisplayOrder = 2
	if err := validatePersistedResearchThemeSummary(invalid); err == nil {
		t.Fatal("duplicated persisted Impact identity was accepted")
	}
}

func TestResearchReasoningTreeAdapterRejectsMalformedPersistedRows(t *testing.T) {
	now := time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC)
	treeID := "RRT11111111-1111-4111-8111-111111111111"
	chainID := "ENT22222222-2222-4222-8222-222222222222"
	publication := researchReasoningTreePublication{
		ReceiptID: "RRI33333333-3333-4333-8333-333333333333",
		Mapping:   map[string]string{chainID: treeID},
		Counts:    researchbiz.ReasonTreeCounts{ReasoningTrees: 1, Nodes: 1, SignalAssociations: 1, Receipts: 1},
		Trees: []ResearchReasoningTreeSummary{{
			ReasoningTreeID: treeID, TreeKey: chainID, DisplayName: "Chain", IndustryChainEntityID: chainID,
			IndustryChainName: "Chain", Title: "Tree", DisplayOrder: 1, PublishedAt: now,
		}},
	}
	if !validReasoningTreePublication(publication, 1, 0, 1) {
		t.Fatal("valid persisted Reasoning Tree publication was rejected")
	}
	invalidPublication := publication
	invalidPublication.Trees = append([]ResearchReasoningTreeSummary(nil), publication.Trees...)
	invalidPublication.Trees[0].ReasoningTreeID = "not-a-uuid"
	if validReasoningTreePublication(invalidPublication, 1, 0, 1) {
		t.Fatal("malformed persisted Reasoning Tree identity was accepted")
	}
	detail := ResearchReasoningTreeDetail{
		ThemeKey: "theme:one", PublicationMode: "formal", PublicationContractVersion: 2,
	}
	tree := ResearchReasoningTree{
		ReasoningTreeID: treeID, ThemeID: "RTH44444444-4444-4444-8444-444444444444",
		TreeKey: chainID, DisplayName: "Chain", IndustryChainEntityID: chainID, IndustryChainName: "Chain",
		Title: "Tree", OneLineConclusion: "Conclusion", ImpactDirection: "positive", ImpactStrength: "medium",
		DisplayOrder: 1, PublishedAt: now,
		Nodes: []researchbiz.ReasoningTreeNodeRecord{{
			ID: "RRN55555555-5555-4555-8555-555555555555", NodeKey: chainID, DisplayName: "Node",
			ChainNodeEntityID: chainID, Name: "Node", Position: 1, ImpactDirection: "positive", ImpactStrength: "medium",
			Signals: []researchbiz.SignalRecord{{
				SignalKey: "signal:one", VariableSignalKey: "signal:one", SignalRole: "primary",
				SignalDirection: "increase", DisplaySummary: "Signal", DisplayOrder: 1,
			}},
		}},
	}
	if !validReasoningTreeDetail(detail, tree, []string{chainID}) {
		t.Fatal("valid persisted Reasoning Tree detail was rejected")
	}
	invalidTree := tree
	invalidTree.Nodes = append([]researchbiz.ReasoningTreeNodeRecord(nil), tree.Nodes...)
	invalidTree.Nodes[0].Signals = append([]researchbiz.SignalRecord(nil), tree.Nodes[0].Signals...)
	invalidTree.Nodes[0].Signals[0].SignalRole = "invented"
	if validReasoningTreeDetail(detail, invalidTree, []string{chainID}) {
		t.Fatal("malformed persisted Reasoning Tree Signal enum was accepted")
	}
	invalidTree = tree
	invalidTree.Nodes = append([]researchbiz.ReasoningTreeNodeRecord(nil), tree.Nodes...)
	incomingID := "IGE66666666-6666-4666-8666-666666666666"
	invalidTree.Nodes[0].IncomingIndustryChainGraphEdgeID = &incomingID
	invalidTree.Nodes[0].IncomingGraphEdge = &researchbiz.GraphEdgeRecord{
		ID: incomingID, RelationType: "input_to", ReviewStatus: "approved", Status: "active",
	}
	if validReasoningTreeDetail(detail, invalidTree, []string{chainID}) {
		t.Fatal("persisted Reasoning Tree first node with an Incoming Graph Edge was accepted")
	}
	secondNode := tree.Nodes[0]
	secondNode.ID = "RRN77777777-7777-4777-8777-777777777777"
	secondNode.NodeKey = "node:second"
	secondNode.DisplayName = "Second node"
	secondNode.ChainNodeEntityID = "ENT88888888-8888-4888-8888-888888888888"
	secondNode.Position = 2
	secondNode.IncomingIndustryChainGraphEdgeID = &incomingID
	secondNode.IncomingGraphEdge = &researchbiz.GraphEdgeRecord{
		ID: "IGE99999999-9999-4999-8999-999999999999", RelationType: "input_to", ReviewStatus: "approved", Status: "active",
	}
	invalidTree = tree
	invalidTree.Nodes = append(append([]researchbiz.ReasoningTreeNodeRecord(nil), tree.Nodes...), secondNode)
	if validReasoningTreeDetail(detail, invalidTree, []string{chainID, secondNode.NodeKey}) {
		t.Fatal("persisted Reasoning Tree mismatched Incoming Graph Edge identity was accepted")
	}
}

func TestPostgresResearchAggregateUsesDomainProviders(t *testing.T) {
	db := openResearchV1TestDatabase(t)
	seedResearchV1MasterData(t, db)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	now := time.Now().UTC().Add(time.Minute).Truncate(time.Second)
	seedTypedResearchSemanticFact(t, ctx, db, now)
	seedTypedForwardSemanticFact(t, ctx, db, now)
	if _, err := db.ExecContext(ctx, `
INSERT INTO variable_definitions (
    variable_key, version, name_zh, name_en, domain, business_definition,
    value_type, allowed_directions, status, created_at
) VALUES (
    'future_only_variable', 1, '未来变量', 'Future-only variable', 'test',
    'Must not cross analysis_as_of', 'index', ARRAY['increase'], 'active', $1
)`, now.Add(time.Hour)); err != nil {
		t.Fatalf("seed future TBox definition: %v", err)
	}

	researchStore, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	eventStore, err := eventdata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	eventUseCase, err := eventbiz.NewUseCase(eventStore)
	if err != nil {
		t.Fatal(err)
	}
	eventSemanticStore, err := eventsemanticdata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	eventSemanticUseCase, err := eventsemanticbiz.NewUseCase(eventSemanticStore)
	if err != nil {
		t.Fatal(err)
	}
	entityStore, err := entitydata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	entityUseCase, err := entitybiz.NewUseCase(entityStore)
	if err != nil {
		t.Fatal(err)
	}
	contextService, err := researchbiz.NewUseCase(
		researchStore,
		researchStore,
		eventUseCase,
		eventSemanticUseCase,
		entityUseCase,
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	eventOnlyAsOf := now.Add(-30*time.Minute - 30*time.Second)
	eventOnlyPage, err := contextService.List(ctx, researchbiz.AnalysisContextRequest{
		DiscoveryWindowStart: now.Add(-time.Hour).Format(time.RFC3339),
		DiscoveryWindowEnd:   eventOnlyAsOf.Format(time.RFC3339),
		AnalysisAsOf:         eventOnlyAsOf.Format(time.RFC3339),
		PageSize:             20,
	})
	if err != nil || len(eventOnlyPage.EventSemanticBundles) != 1 {
		t.Fatalf("list Event-only Research Analysis Context page: %#v, %v", eventOnlyPage, err)
	}
	eventOnlyBundle := eventOnlyPage.EventSemanticBundles[0]
	if eventOnlyBundle.Event.ID != testTypedEventID || len(eventOnlyBundle.Evidence) != 0 {
		t.Fatalf("Event-only Research Analysis Context bundle = %#v", eventOnlyBundle)
	}
	request := researchbiz.AnalysisContextRequest{
		DiscoveryWindowStart: now.Add(-time.Hour).Format(time.RFC3339),
		DiscoveryWindowEnd:   now.Format(time.RFC3339),
		AnalysisAsOf:         now.Format(time.RFC3339),
		PageSize:             1,
	}
	firstPage, err := contextService.List(ctx, request)
	if err != nil {
		t.Fatalf("list first typed Research Analysis Context page: %v", err)
	}
	if !firstPage.HasMore || firstPage.NextCursor == "" ||
		len(firstPage.EventSemanticBundles) != 1 {
		t.Fatalf("first typed Analysis Context page = %#v", firstPage)
	}
	request.Cursor = firstPage.NextCursor
	secondPage, err := contextService.List(ctx, request)
	if err != nil {
		t.Fatalf("list second typed Research Analysis Context page: %v", err)
	}
	if secondPage.HasMore || len(secondPage.EventSemanticBundles) != 1 ||
		len(firstPage.EventPageFingerprint) != 64 ||
		len(secondPage.EventPageFingerprint) != 64 ||
		len(firstPage.ReferenceClosureFingerprint) != 64 ||
		len(secondPage.ReferenceClosureFingerprint) != 64 {
		t.Fatalf("second typed Analysis Context page = %#v", secondPage)
	}
	bundles := append(firstPage.EventSemanticBundles, secondPage.EventSemanticBundles...)
	var actualFound, forecastFound bool
	for _, bundle := range bundles {
		if bundle.Event.ID == testTypedEventID {
			if len(bundle.Evidence) != 1 ||
				!bundle.Evidence[0].KnowledgeAvailableAt.After(bundle.Event.KnowledgeAvailableAt) {
				t.Fatalf("legacy Event Evidence availability = %#v, want later than Event availability", bundle)
			}
		}
		if len(bundle.VariableSignals) != 1 {
			t.Fatalf("typed Analysis Context bundle = %#v", bundle)
		}
		signal := bundle.VariableSignals[0]
		switch signal.VariableSignalID {
		case testTypedSignalID:
			actualFound = signal.AssertionModality == "actual" &&
				len(signal.DirectImpacts) == 0
		case testTypedForwardSignalID:
			forecastFound = signal.AssertionModality == "source_forecast" &&
				signal.StatementAt != nil && signal.ForecastPeriodStart != nil &&
				signal.ForecastPeriodEnd != nil && len(signal.Measurements) == 1 &&
				len(signal.DirectImpacts) == 0
		}
	}
	if !actualFound || !forecastFound ||
		firstPage.TemporalSemantics != researchbiz.AnalysisContextTemporalSemantics ||
		firstPage.TBoxContractVersion != researchbiz.AnalysisContextTBoxContractVersion ||
		len(firstPage.Dictionaries.VariableDefinitions) == 0 ||
		len(secondPage.Dictionaries.VariableDefinitions) == 0 {
		t.Fatalf("typed Analysis Context bundles = %#v", bundles)
	}
	for _, dictionaries := range []researchbiz.Dictionaries{
		firstPage.Dictionaries,
		secondPage.Dictionaries,
	} {
		for _, definition := range dictionaries.VariableDefinitions {
			if definition.Key == "future_only_variable" {
				t.Fatal("Analysis Context leaked a TBox definition created after analysis_as_of")
			}
		}
	}
	if _, err := db.ExecContext(
		ctx,
		`UPDATE entity_nodes SET updated_at = $2 WHERE id = $1::text`,
		testTypedNodeID,
		now.Add(time.Hour),
	); err != nil {
		t.Fatalf("move referenced Entity update after analysis_as_of: %v", err)
	}
	request.Cursor = ""
	if _, err := contextService.List(ctx, request); !errors.Is(
		err, researchbiz.ErrHistoricalSemanticsUnavailable,
	) {
		t.Fatalf("dangling historical Entity reference error = %v", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`UPDATE entity_nodes SET updated_at = $2 WHERE id = $1::text`,
		testTypedNodeID,
		now.Add(-time.Hour),
	); err != nil {
		t.Fatalf("restore referenced Entity update time: %v", err)
	}
	seedPostAsOfSupersession(t, ctx, db, now)
	request.Cursor = ""
	if _, err := contextService.List(ctx, request); !errors.Is(
		err, researchbiz.ErrHistoricalSemanticsUnavailable,
	) {
		t.Fatalf("historical Analysis Context error = %v", err)
	}

	publicationService := contextService
	aggregate := typedResearchAggregate(now)
	published, err := publicationService.Publish(ctx, "integration-analyst", aggregate)
	if err != nil {
		t.Fatalf("publish atomic Theme aggregate: %v", err)
	}
	if published.ThemeID == "" || published.Counts.Themes != 1 ||
		published.Counts.ReasoningTrees != 1 || published.Counts.Receipts != 2 {
		t.Fatalf("publication result = %#v", published)
	}
	replayed, err := publicationService.Publish(ctx, "integration-analyst", aggregate)
	if err != nil || !replayed.Replayed || replayed.ThemeID != published.ThemeID {
		t.Fatalf("publication replay = %#v err=%v", replayed, err)
	}
	var concurrent sync.WaitGroup
	replayResults := make(chan researchbiz.Result, 2)
	replayErrors := make(chan error, 2)
	for range 2 {
		concurrent.Add(1)
		go func() {
			defer concurrent.Done()
			result, replayErr := publicationService.Publish(ctx, "integration-analyst", aggregate)
			replayResults <- result
			replayErrors <- replayErr
		}()
	}
	concurrent.Wait()
	close(replayResults)
	close(replayErrors)
	for replayErr := range replayErrors {
		if replayErr != nil {
			t.Fatalf("concurrent publication replay error = %v", replayErr)
		}
	}
	for result := range replayResults {
		if !result.Replayed || result.ThemeID != published.ThemeID || result.ReceiptID != published.ReceiptID {
			t.Fatalf("concurrent publication replay = %#v", result)
		}
	}
	secondAggregate := typedResearchAggregate(now)
	secondAggregate.AnalysisBatchID = "typed-research-publication-2"
	secondAggregate.Theme.ThemeKey = "typed-supply-2"
	secondAggregate.Theme.Title = "Second typed supply context"
	secondPublished, err := publicationService.Publish(
		ctx,
		"integration-analyst",
		secondAggregate,
	)
	if err != nil {
		t.Fatalf("publish second atomic Theme aggregate: %v", err)
	}
	if !secondPublished.PublishedAt.After(published.PublishedAt) {
		t.Fatalf(
			"second publication time = %s, want after %s",
			secondPublished.PublishedAt,
			published.PublishedAt,
		)
	}
	readService := contextService
	firstThemePage, err := readService.ListThemes(
		ctx,
		researchbiz.ResearchListRequest{WindowHours: 24, Limit: 1},
	)
	if err != nil {
		t.Fatalf("list first independently published Theme page: %v", err)
	}
	if firstThemePage.ThemeCount != 2 ||
		len(firstThemePage.Items) != 1 ||
		firstThemePage.Items[0].ID != secondPublished.ThemeID ||
		firstThemePage.NextCursor == nil {
		t.Fatalf(
			"first independent Theme page = %#v, want latest Theme plus next cursor",
			firstThemePage,
		)
	}
	secondThemePage, err := readService.ListThemes(
		ctx,
		researchbiz.ResearchListRequest{
			WindowHours: 24,
			Limit:       1,
			Cursor:      *firstThemePage.NextCursor,
		},
	)
	if err != nil {
		t.Fatalf("list second independently published Theme page: %v", err)
	}
	if secondThemePage.ThemeCount != 2 ||
		len(secondThemePage.Items) != 1 ||
		secondThemePage.Items[0].ID != published.ThemeID ||
		secondThemePage.NextCursor != nil {
		t.Fatalf(
			"second independent Theme page = %#v, want earlier Theme and no cursor",
			secondThemePage,
		)
	}
	concurrentAggregate := typedResearchAggregate(now)
	concurrentAggregate.AnalysisBatchID = "typed-research-concurrent-first"
	concurrentAggregate.Theme.ThemeKey = "typed-supply-concurrent"
	concurrentAggregate.Theme.Title = "Concurrent typed supply context"
	startConcurrent := make(chan struct{})
	concurrentResults := make(chan researchbiz.Result, 2)
	concurrentErrors := make(chan error, 2)
	for range 2 {
		concurrent.Add(1)
		go func() {
			defer concurrent.Done()
			<-startConcurrent
			result, publishErr := publicationService.Publish(ctx, "integration-analyst", concurrentAggregate)
			concurrentResults <- result
			concurrentErrors <- publishErr
		}()
	}
	close(startConcurrent)
	concurrent.Wait()
	close(concurrentResults)
	close(concurrentErrors)
	for publishErr := range concurrentErrors {
		if publishErr != nil {
			t.Fatalf("concurrent first publication error = %v", publishErr)
		}
	}
	createdCount, replayCount := 0, 0
	for result := range concurrentResults {
		if result.Replayed {
			replayCount++
		} else {
			createdCount++
		}
	}
	if createdCount != 1 || replayCount != 1 {
		t.Fatalf("concurrent first publication created=%d replayed=%d", createdCount, replayCount)
	}

	conflictAggregate := typedResearchAggregate(now)
	conflictAggregate.AnalysisBatchID = "typed-research-concurrent-conflict"
	conflictAggregate.Theme.ThemeKey = "typed-supply-conflict"
	changedConflictAggregate := conflictAggregate
	changedConflictAggregate.Theme.Title = "Different payload for the same identity"
	startConflict := make(chan struct{})
	conflictErrors := make(chan error, 2)
	for _, candidate := range []researchbiz.Aggregate{conflictAggregate, changedConflictAggregate} {
		candidate := candidate
		concurrent.Add(1)
		go func() {
			defer concurrent.Done()
			<-startConflict
			_, publishErr := publicationService.Publish(ctx, "integration-analyst", candidate)
			conflictErrors <- publishErr
		}()
	}
	close(startConflict)
	concurrent.Wait()
	close(conflictErrors)
	successCount, conflictCount := 0, 0
	for publishErr := range conflictErrors {
		switch {
		case publishErr == nil:
			successCount++
		case errors.Is(publishErr, researchbiz.ErrPayloadConflict):
			conflictCount++
		default:
			t.Fatalf("concurrent conflicting publication error = %v", publishErr)
		}
	}
	if successCount != 1 || conflictCount != 1 {
		t.Fatalf("concurrent conflicting publication success=%d conflict=%d", successCount, conflictCount)
	}

	snapshotAggregate := integrationSnapshotAggregate(now)
	snapshotPublished, err := publicationService.PublishSnapshot(ctx, "integration-analyst", snapshotAggregate)
	if err != nil {
		t.Fatalf("publish analyst snapshot Theme aggregate: %v", err)
	}
	if snapshotPublished.PublicationMode != researchbiz.SnapshotPublicationMode ||
		len(snapshotPublished.ReasoningTreeIDsByTreeKey) != 1 {
		t.Fatalf("snapshot publication result = %#v", snapshotPublished)
	}
	snapshotTreeID := snapshotPublished.ReasoningTreeIDsByTreeKey["tree:typed-snapshot"]
	snapshotDetail, err := readService.GetReasoningTree(ctx, snapshotPublished.ThemeID, snapshotTreeID)
	if err != nil {
		t.Fatalf("read analyst snapshot Reason Tree: %v", err)
	}
	if snapshotDetail.ReasoningTree.IndustryChainEntityID != "" ||
		snapshotDetail.ReasoningTree.TreeKey != "tree:typed-snapshot" ||
		snapshotDetail.ReasoningTree.DisplayName != "利率—住房融资传导路径" ||
		snapshotDetail.ReasoningTree.Nodes[0].NodeKey != "node:housing-finance" ||
		snapshotDetail.ReasoningTree.Nodes[0].DisplayName != "美国住房融资条件" ||
		snapshotDetail.ReasoningTree.Nodes[0].PrimarySignal.DisplaySummary != "融资成本保持高位" ||
		snapshotDetail.ReasoningTree.Nodes[0].PrimarySignal.Direction != nil ||
		len(snapshotDetail.ReasoningTree.Events[0].EvidenceIDs) != 1 ||
		snapshotDetail.ReasoningTree.Events[0].EvidenceIDs[0] != testTypedEvidenceID {
		t.Fatalf("snapshot readback lost display contract: %#v", snapshotDetail)
	}
	assertAnalystSnapshotSignalConstraintsDoNotRelaxFormalRows(
		t,
		ctx,
		db,
		published.ReasoningTreeIDsByIndustryChainEntityID[testTypedChainID],
		snapshotTreeID,
	)

	seedBCIReverseGraph(t, ctx, db)
	if _, err := db.ExecContext(
		ctx,
		`UPDATE event_entity_links SET entity_id = $1::text WHERE id = 'ENL11000000-0000-4000-8000-000000000005'`,
		testBCISystemNodeID,
	); err != nil {
		t.Fatalf("move accepted root Signal to BCI system node: %v", err)
	}
	bciAggregate := bciReverseResearchAggregate(now)
	bciPublished, err := publicationService.Publish(ctx, "integration-analyst", bciAggregate)
	if err != nil {
		t.Fatalf("publish reverse multi-hop BCI Theme aggregate: %v", err)
	}
	bciTreeID := bciPublished.ReasoningTreeIDsByIndustryChainEntityID[testBCIChainID]
	bciDetail, err := readService.GetReasoningTree(ctx, bciPublished.ThemeID, bciTreeID)
	if err != nil {
		t.Fatalf("read reverse multi-hop BCI Reason Tree: %v", err)
	}
	expectedNodeIDs := []string{
		testBCISystemNodeID,
		testBCITerminalNodeID,
		testBCIElectrodeNodeID,
	}
	expectedEdgeIDs := []*string{
		nil,
		optionalString(testBCITerminalEdgeID),
		optionalString(testBCIElectrodeEdgeID),
	}
	if len(bciDetail.ReasoningTree.Nodes) != len(expectedNodeIDs) {
		t.Fatalf("BCI readback nodes = %#v, want three-node path", bciDetail.ReasoningTree.Nodes)
	}
	for index, node := range bciDetail.ReasoningTree.Nodes {
		if node.Position != index+1 || node.ChainNodeEntityID != expectedNodeIDs[index] ||
			!equalOptionalString(node.IncomingIndustryChainGraphEdgeID, expectedEdgeIDs[index]) {
			t.Fatalf("BCI readback node %d = %#v", index+1, node)
		}
	}
	assertBCIPersistedLineage(t, ctx, db, bciTreeID)

	store := researchStore
	rollbackBatch := "typed-research-rollback"
	rollbackErr := errors.New("synthetic late failure")
	err = store.InResearchPublicationTransaction(ctx, func(tx researchbiz.PublicationTransaction) error {
		if err := tx.InsertThemeReceipt(ctx, researchbiz.Receipt{
			ID:              "RTI11000000-0000-4000-8000-000000000010",
			AnalysisBatchID: rollbackBatch, PublisherSubject: "integration-analyst",
			PayloadHash:     strings.Repeat("d", 64),
			ThemeID:         "RTH11000000-0000-4000-8000-000000000011",
			ThemeKey:        "rollback",
			ContractVersion: 2,
			PublicationMode: "formal",
			ReasoningTreeIDsByIndustryChainEntityID: map[string]string{
				testTypedChainID: "RRT11000000-0000-4000-8000-000000000012",
			},
			Counts: researchbiz.Counts{
				Themes: 1, Impacts: 1, ThemeEventAssociations: 1,
				ReasoningTrees: 1, Receipts: 2,
			},
			PublishedAt: now, ImportedAt: now,
		}); err != nil {
			return err
		}
		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("rollback error = %v, want synthetic late failure", err)
	}
	var receiptCount int
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM research_theme_import_receipts WHERE analysis_batch_id = $1`,
		rollbackBatch,
	).Scan(&receiptCount); err != nil {
		t.Fatal(err)
	}
	if receiptCount != 0 {
		t.Fatalf("failed transaction persisted %d aggregate receipts", receiptCount)
	}
}

func assertAnalystSnapshotSignalConstraintsDoNotRelaxFormalRows(
	t *testing.T,
	ctx context.Context,
	db *sql.DB,
	formalTreeID string,
	snapshotTreeID string,
) {
	t.Helper()
	var formalNodeID, snapshotNodeID string
	for treeID, target := range map[string]*string{
		formalTreeID:   &formalNodeID,
		snapshotTreeID: &snapshotNodeID,
	} {
		if err := db.QueryRowContext(ctx, `SELECT id::text
FROM research_reasoning_tree_nodes
WHERE reasoning_tree_id = $1::text
ORDER BY position
LIMIT 1`, treeID).Scan(target); err != nil {
			t.Fatalf("read Reason Tree node for signal constraint test: %v", err)
		}
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO research_reasoning_tree_node_signals (
    reasoning_tree_node_id, signal_key, signal_role, signal_direction,
    display_summary, display_order, source_kind
) VALUES ($1::text, 'signal:formal-bypass', 'supporting', 'increase',
    'must be rejected', 2, 'legacy_snapshot')`, formalNodeID); err == nil {
		t.Fatal("formal/legacy signal row accepted analyst snapshot signal_key identity")
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO research_reasoning_tree_node_signals (
    reasoning_tree_node_id, variable_signal_key, signal_role, signal_direction,
    display_summary, display_order, source_kind
) VALUES ($1::text, 'formal_direction_required', 'supporting', NULL,
    'must be rejected', 2, 'legacy_snapshot')`, formalNodeID); err == nil {
		t.Fatal("formal/legacy signal row accepted nullable signal_direction")
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO research_reasoning_tree_node_signals (
    reasoning_tree_node_id, variable_signal_key, signal_role, signal_direction,
    display_summary, display_order, source_kind
) VALUES ($1::text, 'snapshot-formal-bypass', 'supporting', 'increase',
    'must be rejected', 2, 'analyst_snapshot')`, snapshotNodeID); err == nil {
		t.Fatal("analyst snapshot signal row accepted formal variable_signal_key identity")
	}
}

func integrationSnapshotAggregate(asOf time.Time) researchbiz.SnapshotAggregate {
	return researchbiz.SnapshotAggregate{
		PublicationMode:      researchbiz.SnapshotPublicationMode,
		AnalysisBatchID:      "integration-analyst-snapshot",
		AnalysisAsOf:         asOf.Format(time.RFC3339),
		DiscoveryWindowStart: asOf.Add(-time.Hour).Format(time.RFC3339),
		DiscoveryWindowEnd:   asOf.Format(time.RFC3339),
		Theme: researchbiz.SnapshotTheme{
			ThemeKey: "theme:typed-snapshot", Title: "高利率环境下的住房融资压力",
			OneLineConclusion:   "融资成本高位可能继续抑制住房需求。",
			ConclusionDirection: "negative", ImpactStrength: "medium", TransmissionStage: "validation",
			InvestmentGuidanceAction: "observe", InvestmentGuidanceSummary: "观察按揭利率与住房成交。",
			TimeHorizonCategory: "medium_term",
			Impacts: []researchbiz.SnapshotImpact{{
				NodeKey: "node:housing-finance", DisplayName: "住房融资压力",
				RelationRole: "constraint", ImpactDirection: "negative", DisplayOrder: 1,
			}},
			Events: []researchbiz.SnapshotEvent{{EventID: testTypedEventID, EvidenceIDs: []string{testTypedEvidenceID}, EvidenceRole: "driver"}},
		},
		ReasoningTrees: []researchbiz.SnapshotReasoningTree{{
			TreeKey: "tree:typed-snapshot", DisplayName: "利率—住房融资传导路径",
			Title: "住房融资", DisplayOrder: 1, OneLineConclusion: "高利率继续压制融资可得性。",
			ImpactDirection: "negative", ImpactStrength: "medium",
			Events: []researchbiz.SnapshotTreeEvent{{EventID: testTypedEventID, EvidenceIDs: []string{testTypedEvidenceID}, EvidenceRole: "driver", DisplayOrder: 1}},
			Nodes: []researchbiz.SnapshotNode{{
				NodeKey: "node:housing-finance", DisplayName: "美国住房融资条件", Position: 1,
				ImpactDirection: "negative", ImpactStrength: "medium",
				Signals: []researchbiz.SnapshotSignal{{
					SignalKey: "signal:mortgage-cost", DisplaySummary: "融资成本保持高位",
					Role: "primary", DisplayOrder: 1,
				}},
			}},
		}},
	}
}

func seedPostAsOfSupersession(
	t *testing.T,
	ctx context.Context,
	db *sql.DB,
	asOf time.Time,
) {
	t.Helper()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO event_semantic_context_leases (
    id, event_id, agent_execution_id, worker_id, status, lease_expires_at,
    context_snapshot, leased_at, consumed_at
) VALUES (
    'SCL13000000-0000-4000-8000-000000000003', $1,
    'post-as-of-supersession', 'integration-worker', 'consumed',
    $2, '{}'::jsonb, $3, $3
)`, testTypedEventID, asOf.Add(time.Hour), asOf.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO event_semantic_submissions (
    id, context_lease_id, event_id, agent_execution_id, agent_key, agent_version,
    generator_prompt_hash, generator_model, reviewer_prompt_hash, reviewer_model,
    ontology_version, acceptance_policy_key, acceptance_policy_version,
    canonical_payload_hash, status, candidate_counts, decision_summary,
    created_at, finalized_at
) VALUES (
    'ESS13000000-0000-4000-8000-000000000004',
    'SCL13000000-0000-4000-8000-000000000003', $1,
    'post-as-of-supersession', 'event-semantic', 'integration',
    $2, 'fake-generator', $3, 'fake-reviewer', 'integration-ontology',
    'event-semantics.phase-one', 1, $4, 'superseded',
    '{}'::jsonb, '{}'::jsonb, $5, $6
)`,
		testTypedEventID, strings.Repeat("4", 64), strings.Repeat("5", 64),
		strings.Repeat("6", 64), asOf.Add(-time.Minute), asOf.Add(time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

const (
	testTypedChainID             = "ENT10000000-0000-4000-8000-000000000001"
	testTypedNodeID              = "ENT10000000-0000-4000-8000-000000000002"
	testTypedEventID             = "EVT10000000-0000-4000-8000-000000000003"
	testTypedEvidenceID          = "EEL11000000-0000-4000-8000-000000000002"
	testTypedSubmissionID        = "ESS11000000-0000-4000-8000-000000000004"
	testTypedSignalID            = "VSG11000000-0000-4000-8000-000000000006"
	testTypedForwardEventID      = "EVT10000000-0000-4000-8000-000000000004"
	testTypedForwardEvidenceID   = "EEL12000000-0000-4000-8000-000000000002"
	testTypedForwardSubmissionID = "ESS12000000-0000-4000-8000-000000000004"
	testTypedForwardSignalID     = "VSG12000000-0000-4000-8000-000000000006"
	testBCIChainID               = "ENT822a8ddc-5ebc-5f03-8ef8-ba9bfba192d9"
	testBCISystemNodeID          = "ENTc38d2f7b-9900-5e81-af06-76393bcc2617"
	testBCITerminalNodeID        = "ENT96336148-76c0-504e-b82e-ac395f8fe268"
	testBCIElectrodeNodeID       = "ENTd3882237-d639-5660-b7d8-aa3563706113"
	testBCITerminalEdgeID        = "IGE300188b0-d01c-5987-ad8a-646067edc7cd"
	testBCIElectrodeEdgeID       = "IGEdc00a16e-0d8e-5db9-9a5d-fbc1fd9a84cf"
)

func seedBCIReverseGraph(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	statements := []struct {
		query string
		args  []any
	}{
		{
			`INSERT INTO entity_nodes (
    id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status
) VALUES
    ($1::text, 'industry-chain:bci-system', 'industry_chain', 'industry_chain',
     '脑机接口系统产业链', '脑机接口系统产业链', '{}', 'active'),
    ($2::text, 'chain-node:bci-system', 'chain_node', 'chain_node',
     '非侵入式脑机接口系统', '非侵入式脑机接口系统', '{}', 'active'),
    ($3::text, 'chain-node:bci-terminal', 'chain_node', 'chain_node',
     '脑机接口采集终端', '脑机接口采集终端', '{}', 'active'),
    ($4::text, 'chain-node:bci-electrode', 'chain_node', 'chain_node',
     '脑机接口采集电极', '脑机接口采集电极', '{}', 'active')`,
			[]any{testBCIChainID, testBCISystemNodeID, testBCITerminalNodeID, testBCIElectrodeNodeID},
		},
		{
			`INSERT INTO chain_node_profiles (entity_id, definition, boundary_note, review_status)
VALUES
    ($1::text, '非侵入式脑机接口系统节点', '系统边界', 'approved'),
    ($2::text, '脑机接口采集终端节点', '终端边界', 'approved'),
    ($3::text, '脑机接口采集电极节点', '电极边界', 'approved')`,
			[]any{testBCISystemNodeID, testBCITerminalNodeID, testBCIElectrodeNodeID},
		},
		{
			`INSERT INTO industry_chain_definitions (
    entity_id, scope, target_output, end_use, technology_route_qualifier,
    observable_variables, geography, as_of_date, review_status, review_note
) VALUES (
    $1::text, '非侵入式脑机接口系统与采集硬件', '脑机接口系统', '康复与人机交互', NULL,
    ARRAY['市场需求'], '中国', CURRENT_DATE, 'approved', NULL
)`,
			[]any{testBCIChainID},
		},
		{
			`INSERT INTO industry_chain_node_memberships (
    industry_chain_entity_id, chain_node_entity_id, position, contextual_stage,
    review_status, status, inclusion_reason, evidence_ids, source_name, source_url, verified_at
) VALUES
    ($1::text, $2::text, 1, 'downstream', 'approved', 'active',
     '系统节点', ARRAY['evidence:bci-system'], 'integration fixture', 'artifact://bci-system', now()),
    ($1::text, $3::text, 2, 'midstream', 'approved', 'active',
     '终端组成节点', ARRAY['evidence:bci-terminal'], 'integration fixture', 'artifact://bci-terminal', now()),
    ($1::text, $4::text, 3, 'upstream', 'approved', 'active',
     '电极组成节点', ARRAY['evidence:bci-electrode'], 'integration fixture', 'artifact://bci-electrode', now())`,
			[]any{testBCIChainID, testBCISystemNodeID, testBCITerminalNodeID, testBCIElectrodeNodeID},
		},
		{
			`INSERT INTO industry_chain_graph_edges (
    id, industry_chain_entity_id, from_chain_node_entity_id, to_chain_node_entity_id,
    relation_type, mechanism, condition_note, segment_kind, omitted_step_note,
    review_status, status, evidence_ids, source_name, source_url, verified_at
) VALUES
    ($1::text, $2::text, $3::text, $4::text, 'is_component_of',
     '采集终端是系统组成部分', NULL, 'direct_candidate', NULL,
     'approved', 'active', ARRAY['evidence:bci-terminal-system'],
     'integration fixture', 'artifact://bci-terminal-system', now()),
    ($5::text, $2::text, $6::text, $3::text, 'is_component_of',
     '采集电极是采集终端组成部分', NULL, 'direct_candidate', NULL,
     'approved', 'active', ARRAY['evidence:bci-electrode-terminal'],
     'integration fixture', 'artifact://bci-electrode-terminal', now())`,
			[]any{
				testBCITerminalEdgeID, testBCIChainID, testBCITerminalNodeID,
				testBCISystemNodeID, testBCIElectrodeEdgeID, testBCIElectrodeNodeID,
			},
		},
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement.query, statement.args...); err != nil {
			t.Fatalf("seed BCI reverse graph: %v\n%s", err, statement.query)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit BCI reverse graph: %v", err)
	}
}

func assertBCIPersistedLineage(t *testing.T, ctx context.Context, db *sql.DB, treeID string) {
	t.Helper()
	rows, err := db.QueryContext(ctx, `SELECT node.position,
       node.incoming_source_kind,
       node.inference_upstream_variable_signal_id::text,
       signal.source_kind,
       signal.upstream_variable_signal_id::text,
       signal.industry_chain_graph_edge_id::text
FROM research_reasoning_tree_nodes node
JOIN research_reasoning_tree_node_signals signal
  ON signal.reasoning_tree_node_id = node.id AND signal.signal_role = 'primary'
WHERE node.reasoning_tree_id = $1::text
ORDER BY node.position`, treeID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	expectedEdges := []string{"", testBCITerminalEdgeID, testBCIElectrodeEdgeID}
	rowCount := 0
	for rows.Next() {
		rowCount++
		var position int
		var incomingSourceKind, incomingUpstreamSignalID sql.NullString
		var signalSourceKind string
		var signalUpstreamSignalID, signalGraphEdgeID sql.NullString
		if err := rows.Scan(
			&position,
			&incomingSourceKind,
			&incomingUpstreamSignalID,
			&signalSourceKind,
			&signalUpstreamSignalID,
			&signalGraphEdgeID,
		); err != nil {
			t.Fatal(err)
		}
		if position == 1 {
			if incomingSourceKind.Valid || signalSourceKind != "formal_signal" ||
				signalUpstreamSignalID.Valid || signalGraphEdgeID.Valid {
				t.Fatalf("root BCI lineage is not formal-only")
			}
			continue
		}
		if !incomingSourceKind.Valid || incomingSourceKind.String != "analyst_inference" ||
			!incomingUpstreamSignalID.Valid || incomingUpstreamSignalID.String != testTypedSignalID ||
			signalSourceKind != "analyst_inference" ||
			!signalUpstreamSignalID.Valid || signalUpstreamSignalID.String != testTypedSignalID ||
			!signalGraphEdgeID.Valid || signalGraphEdgeID.String != expectedEdges[position-1] {
			t.Fatalf("BCI persisted lineage at position %d is incomplete", position)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if rowCount != 3 {
		t.Fatalf("BCI persisted lineage rows = %d, want 3", rowCount)
	}
}

func equalOptionalString(actual, expected *string) bool {
	if actual == nil || expected == nil {
		return actual == nil && expected == nil
	}
	return *actual == *expected
}

func optionalString(value string) *string { return &value }

func seedTypedResearchSemanticFact(
	t *testing.T,
	ctx context.Context,
	db *sql.DB,
	now time.Time,
) {
	t.Helper()
	availableAt := now.Add(-30 * time.Minute)
	acceptedAt := now.Add(-20 * time.Minute)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	statements := []struct {
		query string
		args  []any
	}{
		{
			`UPDATE events
	SET first_seen_at = $1, knowable_at = $1::timestamptz - interval '1 minute', event_time = $1,
	    fact_payload = '{"statement_source":"Integration Source"}'::jsonb
	WHERE id = $2`,
			[]any{availableAt, testTypedEventID},
		},
		{
			`INSERT INTO raw_documents (
    id, ingest_channel, source_type, source_name, source_url, title, content_text,
    raw_mime_type, language, published_at, collected_at, content_hash, ingest_status
) VALUES (
    'EER11000000-0000-4000-8000-000000000001', 'integration', 'filing',
    'Integration Source', 'https://integration.invalid/source', 'Supply disclosure',
    'Market supply decreased 10 percent.', 'text/plain', 'en',
    $1::timestamptz - interval '1 minute', $1, $2, 'collected'
)`,
			[]any{availableAt, strings.Repeat("a", 64)},
		},
		{
			`INSERT INTO event_sources (
    id, event_id, raw_document_id, source_level, evidence_statement, evidence_hash,
    evidence_relation, supports_fields, contract_version, created_at
) VALUES (
    $1, $2, 'EER11000000-0000-4000-8000-000000000001', 'primary',
    'Market supply decreased 10 percent.', $3, 'supports',
    ARRAY['factual_summary','fact_payload'], 3, $4
)`,
			[]any{testTypedEvidenceID, testTypedEventID, researchEvidenceHash("Market supply decreased 10 percent."), acceptedAt},
		},
		{
			`INSERT INTO event_semantic_context_leases (
    id, event_id, agent_execution_id, worker_id, status, lease_expires_at,
    context_snapshot, leased_at, consumed_at
) VALUES (
    'SCL11000000-0000-4000-8000-000000000003', $1,
    'typed-research-execution', 'integration-worker', 'consumed',
    $2, '{}'::jsonb, $3, $4
)`,
			[]any{testTypedEventID, now.Add(time.Hour), availableAt, acceptedAt},
		},
		{
			`INSERT INTO event_semantic_submissions (
    id, context_lease_id, event_id, agent_execution_id, agent_key, agent_version,
    generator_prompt_hash, generator_model, reviewer_prompt_hash, reviewer_model,
    ontology_version, acceptance_policy_key, acceptance_policy_version,
    canonical_payload_hash, status, candidate_counts, decision_summary,
    created_at, finalized_at
) VALUES (
    $1, 'SCL11000000-0000-4000-8000-000000000003', $2,
    'typed-research-execution', 'event-semantic', 'integration',
    $3, 'fake-generator', $4, 'fake-reviewer', 'integration-ontology',
    'event-semantics.phase-one', 1, $5, 'accepted',
    '{"entity_links":1,"variable_signals":1,"direct_impacts":0}'::jsonb,
    '{"decision":"accepted"}'::jsonb, $6, $6
)`,
			[]any{
				testTypedSubmissionID, testTypedEventID, strings.Repeat("8", 64),
				strings.Repeat("9", 64), strings.Repeat("b", 64), acceptedAt,
			},
		},
		{
			`INSERT INTO event_entity_links (
    id, event_id, entity_id, entity_role, assign_source, review_status,
    evidence_note, semantic_submission_id, candidate_key, resolved_mention,
    resolution_method, resolution_confidence, evidence_ids, provenance,
    created_at, updated_at
) VALUES (
    'ENL11000000-0000-4000-8000-000000000005', $1, $2,
    'event_subject', 'ai', 'accepted', '', $3, 'supply-node',
    '高速光模块', 'resolved', 0.99, ARRAY[$4::text], 'semantic', $5, $5
)`,
			[]any{
				testTypedEventID, testTypedNodeID, testTypedSubmissionID,
				testTypedEvidenceID, acceptedAt,
			},
		},
		{
			`INSERT INTO variable_signals (
    id, semantic_submission_id, candidate_key, source_event_id,
    subject_event_entity_link_id, variable_key, variable_version, direction,
    assertion_modality, evidence_ids, extraction_confidence, review_status,
    created_at, updated_at
) VALUES (
    $1, $2, 'market-supply', $3,
    'ENL11000000-0000-4000-8000-000000000005', 'market_supply', 1,
    'decrease', 'actual', ARRAY[$4::text], 0.98, 'accepted', $5, $5
)`,
			[]any{
				testTypedSignalID, testTypedSubmissionID, testTypedEventID,
				testTypedEvidenceID, acceptedAt,
			},
		},
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement.query, statement.args...); err != nil {
			t.Fatalf("seed typed Research semantic fact: %v\n%s", err, statement.query)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit typed Research semantic fact: %v", err)
	}
}

func seedTypedForwardSemanticFact(
	t *testing.T,
	ctx context.Context,
	db *sql.DB,
	now time.Time,
) {
	t.Helper()
	availableAt := now.Add(-25 * time.Minute)
	acceptedAt := now.Add(-15 * time.Minute)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	statements := []struct {
		query string
		args  []any
	}{
		{
			`UPDATE events
SET first_seen_at = $1, knowable_at = $1, event_time = $1,
    event_status = 'confirmed', fact_status = 'verified',
    fact_payload = '{"statement_source":"Integration Company"}'::jsonb
WHERE id = $2`,
			[]any{availableAt, testTypedForwardEventID},
		},
		{
			`INSERT INTO raw_documents (
    id, ingest_channel, source_type, source_name, source_url, title, content_text,
    raw_mime_type, language, published_at, collected_at, content_hash, ingest_status
) VALUES (
    'EER12000000-0000-4000-8000-000000000001', 'integration', 'filing',
    'Integration Company', 'https://integration.invalid/forecast', 'Demand forecast',
    'The company forecasts demand growth of 15 percent.', 'text/plain', 'en',
    $1::timestamptz - interval '1 minute', $1, $2, 'collected'
)`,
			[]any{availableAt, strings.Repeat("e", 64)},
		},
		{
			`INSERT INTO event_sources (
    id, event_id, raw_document_id, source_level, evidence_statement, evidence_hash,
    evidence_relation, supports_fields, contract_version, created_at
) VALUES (
    $1, $2, 'EER12000000-0000-4000-8000-000000000001', 'primary',
    'The company forecasts demand growth of 15 percent.', $3, 'supports',
    ARRAY['factual_summary','fact_payload'], 3, $4
)`,
			[]any{
				testTypedForwardEvidenceID, testTypedForwardEventID,
				researchEvidenceHash("The company forecasts demand growth of 15 percent."), acceptedAt,
			},
		},
		{
			`INSERT INTO event_semantic_context_leases (
    id, event_id, agent_execution_id, worker_id, status, lease_expires_at,
    context_snapshot, leased_at, consumed_at
) VALUES (
    'SCL12000000-0000-4000-8000-000000000003', $1,
    'typed-forward-execution', 'integration-worker', 'consumed',
    $2, '{}'::jsonb, $3, $4
)`,
			[]any{testTypedForwardEventID, now.Add(time.Hour), availableAt, acceptedAt},
		},
		{
			`INSERT INTO event_semantic_submissions (
    id, context_lease_id, event_id, agent_execution_id, agent_key, agent_version,
    generator_prompt_hash, generator_model, reviewer_prompt_hash, reviewer_model,
    ontology_version, acceptance_policy_key, acceptance_policy_version,
    canonical_payload_hash, status, candidate_counts, decision_summary,
    created_at, finalized_at
) VALUES (
    $1, 'SCL12000000-0000-4000-8000-000000000003', $2,
    'typed-forward-execution', 'event-semantic', 'integration',
    $3, 'fake-generator', $4, 'fake-reviewer', 'integration-ontology',
    'event-semantics.phase-one', 1, $5, 'accepted',
    '{"entity_links":1,"variable_signals":1,"direct_impacts":0}'::jsonb,
    '{"decision":"accepted"}'::jsonb, $6, $6
)`,
			[]any{
				testTypedForwardSubmissionID, testTypedForwardEventID,
				strings.Repeat("1", 64), strings.Repeat("2", 64),
				strings.Repeat("3", 64), acceptedAt,
			},
		},
		{
			`INSERT INTO event_entity_links (
    id, event_id, entity_id, entity_role, assign_source, review_status,
    evidence_note, semantic_submission_id, candidate_key, resolved_mention,
    resolution_method, resolution_confidence, evidence_ids, provenance,
    created_at, updated_at
) VALUES (
    'ENL12000000-0000-4000-8000-000000000005', $1, $2,
    'statement_source', 'ai', 'accepted', '', $3, 'forecast-source',
    'Integration Company', 'resolved', 0.99, ARRAY[$4::text], 'semantic', $5, $5
)`,
			[]any{
				testTypedForwardEventID, testTypedNodeID, testTypedForwardSubmissionID,
				testTypedForwardEvidenceID, acceptedAt,
			},
		},
		{
			`INSERT INTO variable_signals (
    id, semantic_submission_id, candidate_key, source_event_id,
    subject_event_entity_link_id, variable_key, variable_version, direction,
    assertion_modality, evidence_ids, statement_at, valid_from, valid_until,
    forecast_period_start, forecast_period_end, extraction_confidence,
    review_status, created_at, updated_at
) VALUES (
    $1, $2, 'demand-forecast', $3,
    'ENL12000000-0000-4000-8000-000000000005', 'market_demand', 1,
    'increase', 'source_forecast', ARRAY[$4::text], $5, $6, $7, $6, $7,
    0.97, 'accepted', $8, $8
)`,
			[]any{
				testTypedForwardSignalID, testTypedForwardSubmissionID,
				testTypedForwardEventID, testTypedForwardEvidenceID, availableAt,
				now.Add(24 * time.Hour), now.Add(30 * 24 * time.Hour), acceptedAt,
			},
		},
		{
			`INSERT INTO variable_signal_measurements (
    id, variable_signal_id, measurement_role, value_shape, raw_value, raw_unit,
    canonical_value, canonical_unit, raw_text, is_approximate, evidence_id, created_at
) VALUES (
    'VSM12000000-0000-4000-8000-000000000007', $1, 'relative_change', 'exact',
    15, '%', 15, 'percent', 'demand growth of 15 percent', false, $2, $3
)`,
			[]any{testTypedForwardSignalID, testTypedForwardEvidenceID, acceptedAt},
		},
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement.query, statement.args...); err != nil {
			t.Fatalf("seed typed forward semantic fact: %v\n%s", err, statement.query)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit typed forward semantic fact: %v", err)
	}
}

func typedResearchAggregate(now time.Time) researchbiz.Aggregate {
	signalID, submissionID, evidenceID := testTypedSignalID, testTypedSubmissionID, testTypedEvidenceID
	evidenceHash := researchEvidenceHash("Market supply decreased 10 percent.")
	return researchbiz.Aggregate{
		AnalysisBatchID:      "typed-research-publication",
		AnalysisAsOf:         now.Format(time.RFC3339),
		DiscoveryWindowStart: now.Add(-time.Hour).Format(time.RFC3339),
		DiscoveryWindowEnd:   now.Format(time.RFC3339),
		Theme: researchbiz.ThemeInput{
			ThemeKey: "typed-supply", Title: "Typed supply context",
			OneLineConclusion: "Supply decreases", ConclusionDirection: "negative",
			ImpactStrength: "medium", TransmissionStage: "validation",
			InvestmentGuidanceAction:  "observe",
			InvestmentGuidanceSummary: "Observe accepted supply signal",
			TimeHorizonCategory:       "short_term",
			Impacts: []researchbiz.ThemeImpactInput{{
				ChainNodeEntityID: testTypedNodeID, RelationRole: "driver",
				ImpactDirection: "negative", DisplayOrder: 1,
			}},
			Events: []researchbiz.ThemeEventInput{{
				EventID: testTypedEventID, EvidenceRole: "driver",
			}},
		},
		ReasoningTrees: []researchbiz.ReasoningTree{{
			ReasonTreeInput: researchbiz.ReasonTreeInput{
				IndustryChainEntityID: testTypedChainID, Title: "Typed chain",
				DisplayOrder: 1, OneLineConclusion: "Supply decreases",
				ImpactDirection: "negative", ImpactStrength: "medium",
				InvalidationConditions: []string{"Supply recovers"},
				Checkpoints: []researchbiz.ReasonTreeCheckpoint{{
					Type: "metric", Summary: "Track market supply",
				}},
				Events: []researchbiz.ReasonTreeEventInput{{
					EventID: testTypedEventID, EvidenceRole: "driver", DisplayOrder: 1,
				}},
			},
			Nodes: []researchbiz.Node{{
				Position: 1, ChainNodeEntityID: testTypedNodeID,
				ImpactDirection: "negative", ImpactStrength: "medium",
				Signals: []researchbiz.Signal{{
					VariableSignalKey: "market_supply", SignalRole: "primary",
					SignalDirection: "decrease", DisplaySummary: "Supply decreases",
					DisplayOrder: 1,
					Lineage: researchbiz.SignalLineage{
						SourceKind: "formal_signal", VariableSignalID: &signalID,
						SemanticSubmissionID: &submissionID, EvidenceID: &evidenceID,
						EvidenceHash: &evidenceHash,
					},
				}},
			}},
		}},
	}
}

func bciReverseResearchAggregate(now time.Time) researchbiz.Aggregate {
	aggregate := typedResearchAggregate(now)
	aggregate.AnalysisBatchID = "bci-reverse-research-publication"
	aggregate.Theme.ThemeKey = "bci-demand"
	aggregate.Theme.Title = "BCI component demand"
	aggregate.Theme.OneLineConclusion = "BCI system demand may propagate to terminal and electrode demand"
	aggregate.Theme.Impacts = []researchbiz.ThemeImpactInput{
		{
			ChainNodeEntityID: testBCISystemNodeID,
			RelationRole:      "driver",
			ImpactDirection:   "uncertain",
			DisplayOrder:      1,
		},
		{
			ChainNodeEntityID: testBCITerminalNodeID,
			RelationRole:      "exposure",
			ImpactDirection:   "uncertain",
			DisplayOrder:      2,
		},
		{
			ChainNodeEntityID: testBCIElectrodeNodeID,
			RelationRole:      "exposure",
			ImpactDirection:   "uncertain",
			DisplayOrder:      3,
		},
	}
	tree := &aggregate.ReasoningTrees[0]
	tree.IndustryChainEntityID = testBCIChainID
	tree.Title = "BCI system component chain"
	tree.OneLineConclusion = aggregate.Theme.OneLineConclusion
	tree.ImpactDirection = "uncertain"
	tree.ImpactStrength = "unknown"
	tree.Nodes[0].ChainNodeEntityID = testBCISystemNodeID
	tree.Nodes[0].ImpactDirection = "uncertain"
	tree.Nodes[0].ImpactStrength = "unknown"
	tree.Nodes[0].Signals[0].VariableSignalKey = "market_supply"
	tree.Nodes[0].Signals[0].SignalDirection = "decrease"
	tree.Nodes[0].Signals[0].DisplaySummary = "BCI system supply decreases"

	rootSignalID := testTypedSignalID
	for _, step := range []struct {
		position  int
		nodeID    string
		edgeID    string
		signalKey string
		summary   string
	}{
		{
			position:  2,
			nodeID:    testBCITerminalNodeID,
			edgeID:    testBCITerminalEdgeID,
			signalKey: "terminal_market_supply",
			summary:   "BCI terminal supply may decrease",
		},
		{
			position:  3,
			nodeID:    testBCIElectrodeNodeID,
			edgeID:    testBCIElectrodeEdgeID,
			signalKey: "electrode_market_supply",
			summary:   "BCI electrode supply may decrease",
		},
	} {
		title := "Demand propagates to the adjacent BCI component"
		mechanism := "The component is required by the previous BCI node"
		condition := "The previous-node demand is realized"
		edgeID := step.edgeID
		tree.Nodes = append(tree.Nodes, researchbiz.Node{
			Position:                         step.position,
			ChainNodeEntityID:                step.nodeID,
			ImpactDirection:                  "uncertain",
			ImpactStrength:                   "unknown",
			IncomingIndustryChainGraphEdgeID: &edgeID,
			IncomingTransmissionTitle:        &title,
			IncomingTransmissionMechanism:    &mechanism,
			IncomingConditionSummary:         &condition,
			IncomingLineage: &researchbiz.IncomingLineage{
				SourceKind:               "analyst_inference",
				UpstreamVariableSignalID: &rootSignalID,
			},
			Signals: []researchbiz.Signal{{
				VariableSignalKey: step.signalKey,
				SignalRole:        "primary",
				SignalDirection:   "decrease",
				DisplaySummary:    step.summary,
				DisplayOrder:      1,
				Lineage: researchbiz.SignalLineage{
					SourceKind:               "analyst_inference",
					UpstreamVariableSignalID: &rootSignalID,
					IndustryChainGraphEdgeID: &edgeID,
				},
			}},
		})
	}
	return aggregate
}
func openResearchV1TestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	databaseURL := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TIDEWISE_TEST_DATABASE_URL to run Research V1 PostgreSQL integration tests")
	}
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	host := parsed.Hostname()
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		t.Fatalf("Research V1 integration database must use a loopback host, got %q", host)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	admin, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("tw_research_v1_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		admin.Close()
		t.Fatal(err)
	}
	config.RuntimeParams["search_path"] = schema
	config.RuntimeParams["tidewise.phase_a_cleanup_write_authorized"] = "reviewed_backup_verified"
	config.RuntimeParams["tidewise.external_identifier_schema_write_authorized"] = "reviewed_backup_verified"
	config.RuntimeParams["tidewise.alliance_economy_schema_write_authorized"] = "reviewed_local_cleanup_verified"
	db := stdlib.OpenDB(*config)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		admin.Close()
		t.Fatal(err)
	}
	migrationDirectory, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpContext(ctx, db, migrationDirectory); err != nil {
		t.Fatalf("apply migrations in isolated Research V1 schema: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		_, _ = admin.ExecContext(context.Background(), `DROP SCHEMA IF EXISTS `+schema+` CASCADE`)
		admin.Close()
	})
	return db
}

func seedResearchV1MasterData(t *testing.T, db *sql.DB) {
	t.Helper()
	const (
		chainID      = "ENT10000000-0000-4000-8000-000000000001"
		nodeID       = "ENT10000000-0000-4000-8000-000000000002"
		eventID      = "EVT10000000-0000-4000-8000-000000000003"
		draftEventID = "EVT10000000-0000-4000-8000-000000000004"
	)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, statement := range []string{
		`INSERT INTO entity_nodes (
		    id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status
		) VALUES
		    ('` + chainID + `', 'industry-chain:optical-module', 'industry_chain', 'industry_chain',
		     '高速光模块产业链', '高速光模块产业链', '{}', 'active'),
		    ('` + nodeID + `', 'chain-node:optical-module', 'chain_node', 'chain_node',
		     '高速光模块', '高速光模块', '{}', 'active')`,
		`INSERT INTO chain_node_profiles (
		    entity_id, definition, boundary_note, review_status
		) VALUES ('` + nodeID + `', '高速光模块生产节点', '仅覆盖高速光模块', 'approved')`,
		`INSERT INTO industry_chain_definitions (
		    entity_id, scope, target_output, end_use, technology_route_qualifier,
		    observable_variables, geography, as_of_date, review_status, review_note
		) VALUES (
		    '` + chainID + `', '高速光模块供需链', '高速光模块', '数据中心互联', NULL,
		    ARRAY['采购数量'], '中国', CURRENT_DATE, 'approved', NULL
		)`,
		`INSERT INTO industry_chain_node_memberships (
		    industry_chain_entity_id, chain_node_entity_id, position, contextual_stage,
		    review_status, status, inclusion_reason, evidence_ids, source_name, source_url, verified_at
		) VALUES (
		    '` + chainID + `', '` + nodeID + `', 1, 'midstream',
		    'approved', 'active', '核心产出节点', ARRAY['evidence:integration'],
		    'integration fixture', 'artifact://research-v1-integration', now()
		)`,
		`INSERT INTO events (
		    id, title, summary, first_seen_at, event_status, fact_status, dedupe_key
		) VALUES
		(
		    '` + eventID + `', '端口计划上调', '端口计划上调 80%', now(),
		    'confirmed', 'verified', 'research-v1-integration-event'
		),
		(
		    '` + draftEventID + `', '待核验端口计划', '未经核验的端口计划', now(),
		    'candidate', 'unverified', 'research-v1-integration-draft-event'
		)`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("seed Research V1 master data: %v\n%s", err, statement)
		}
	}
}

func researchEvidenceHash(statement string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(statement)))
}
