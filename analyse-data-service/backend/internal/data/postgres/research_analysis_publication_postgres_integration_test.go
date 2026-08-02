package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchanalysiscontext"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchpublication"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchreasoningtreeimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
)

func TestResearchAnalysisContextAndAtomicPublicationPostgres(t *testing.T) {
	db := openResearchV1TestDatabase(t)
	seedResearchV1MasterData(t, db)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	now := time.Now().UTC().Add(time.Second).Truncate(time.Second)
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

	contextStore := NewResearchAnalysisContextStore(db)
	contextService := researchanalysiscontext.NewService(contextStore)
	request := researchanalysiscontext.Request{
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
		firstPage.TemporalSemantics != researchanalysiscontext.TemporalSemantics ||
		firstPage.TBoxContractVersion != researchanalysiscontext.TBoxContractVersion ||
		len(firstPage.Dictionaries.VariableDefinitions) == 0 ||
		len(secondPage.Dictionaries.VariableDefinitions) == 0 {
		t.Fatalf("typed Analysis Context bundles = %#v", bundles)
	}
	for _, dictionaries := range []researchanalysiscontext.Dictionaries{
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
		`UPDATE entity_nodes SET updated_at = $2 WHERE id = $1::uuid`,
		testTypedNodeID,
		now.Add(time.Hour),
	); err != nil {
		t.Fatalf("move referenced Entity update after analysis_as_of: %v", err)
	}
	request.Cursor = ""
	if _, err := contextService.List(ctx, request); !errors.Is(
		err, researchanalysiscontext.ErrHistoricalSemanticsUnavailable,
	) {
		t.Fatalf("dangling historical Entity reference error = %v", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`UPDATE entity_nodes SET updated_at = $2 WHERE id = $1::uuid`,
		testTypedNodeID,
		now.Add(-time.Hour),
	); err != nil {
		t.Fatalf("restore referenced Entity update time: %v", err)
	}
	seedPostAsOfSupersession(t, ctx, db, now)
	request.Cursor = ""
	if _, err := contextService.List(ctx, request); !errors.Is(
		err, researchanalysiscontext.ErrHistoricalSemanticsUnavailable,
	) {
		t.Fatalf("historical Analysis Context error = %v", err)
	}

	publicationService := researchpublication.NewService(NewResearchPublicationStore(db))
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
	readService := research.NewService(NewResearchRepository(db), func() time.Time {
		return time.Now().UTC().Add(time.Second)
	})
	firstThemePage, err := readService.ListThemes(
		ctx,
		research.ResearchListRequest{WindowHours: 24, Limit: 1},
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
		research.ResearchListRequest{
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

	seedBCIReverseGraph(t, ctx, db)
	if _, err := db.ExecContext(
		ctx,
		`UPDATE event_entity_links SET entity_id = $1::uuid WHERE id = '11000000-0000-4000-8000-000000000005'`,
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

	store := NewResearchPublicationStore(db)
	rollbackBatch := "typed-research-rollback"
	rollbackErr := errors.New("synthetic late failure")
	err = store.InResearchPublicationTransaction(ctx, func(tx researchpublication.Transaction) error {
		if err := tx.InsertThemeReceipt(ctx, researchpublication.Receipt{
			ID:              "11000000-0000-4000-8000-000000000010",
			AnalysisBatchID: rollbackBatch, PublisherSubject: "integration-analyst",
			PayloadHash:     strings.Repeat("d", 64),
			ThemeID:         "11000000-0000-4000-8000-000000000011",
			ThemeKey:        "rollback",
			ContractVersion: 2,
			ReasoningTreeIDsByIndustryChainEntityID: map[string]string{
				testTypedChainID: "11000000-0000-4000-8000-000000000012",
			},
			Counts: researchpublication.Counts{
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
    '13000000-0000-4000-8000-000000000003', $1,
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
    '13000000-0000-4000-8000-000000000004',
    '13000000-0000-4000-8000-000000000003', $1,
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
	testTypedChainID             = "10000000-0000-4000-8000-000000000001"
	testTypedNodeID              = "10000000-0000-4000-8000-000000000002"
	testTypedEventID             = "10000000-0000-4000-8000-000000000003"
	testTypedEvidenceID          = "11000000-0000-4000-8000-000000000002"
	testTypedSubmissionID        = "11000000-0000-4000-8000-000000000004"
	testTypedSignalID            = "11000000-0000-4000-8000-000000000006"
	testTypedForwardEventID      = "10000000-0000-4000-8000-000000000004"
	testTypedForwardEvidenceID   = "12000000-0000-4000-8000-000000000002"
	testTypedForwardSubmissionID = "12000000-0000-4000-8000-000000000004"
	testTypedForwardSignalID     = "12000000-0000-4000-8000-000000000006"
	testBCIChainID               = "822a8ddc-5ebc-5f03-8ef8-ba9bfba192d9"
	testBCISystemNodeID          = "c38d2f7b-9900-5e81-af06-76393bcc2617"
	testBCITerminalNodeID        = "96336148-76c0-504e-b82e-ac395f8fe268"
	testBCIElectrodeNodeID       = "d3882237-d639-5660-b7d8-aa3563706113"
	testBCITerminalEdgeID        = "300188b0-d01c-5987-ad8a-646067edc7cd"
	testBCIElectrodeEdgeID       = "dc00a16e-0d8e-5db9-9a5d-fbc1fd9a84cf"
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
    ($1::uuid, 'industry-chain:bci-system', 'industry_chain', 'industry_chain',
     '脑机接口系统产业链', '脑机接口系统产业链', '{}', 'active'),
    ($2::uuid, 'chain-node:bci-system', 'chain_node', 'chain_node',
     '非侵入式脑机接口系统', '非侵入式脑机接口系统', '{}', 'active'),
    ($3::uuid, 'chain-node:bci-terminal', 'chain_node', 'chain_node',
     '脑机接口采集终端', '脑机接口采集终端', '{}', 'active'),
    ($4::uuid, 'chain-node:bci-electrode', 'chain_node', 'chain_node',
     '脑机接口采集电极', '脑机接口采集电极', '{}', 'active')`,
			[]any{testBCIChainID, testBCISystemNodeID, testBCITerminalNodeID, testBCIElectrodeNodeID},
		},
		{
			`INSERT INTO chain_node_profiles (entity_id, definition, boundary_note, review_status)
VALUES
    ($1::uuid, '非侵入式脑机接口系统节点', '系统边界', 'approved'),
    ($2::uuid, '脑机接口采集终端节点', '终端边界', 'approved'),
    ($3::uuid, '脑机接口采集电极节点', '电极边界', 'approved')`,
			[]any{testBCISystemNodeID, testBCITerminalNodeID, testBCIElectrodeNodeID},
		},
		{
			`INSERT INTO industry_chain_definitions (
    entity_id, scope, target_output, end_use, technology_route_qualifier,
    observable_variables, geography, as_of_date, review_status, review_note
) VALUES (
    $1::uuid, '非侵入式脑机接口系统与采集硬件', '脑机接口系统', '康复与人机交互', NULL,
    ARRAY['市场需求'], '中国', CURRENT_DATE, 'approved', NULL
)`,
			[]any{testBCIChainID},
		},
		{
			`INSERT INTO industry_chain_node_memberships (
    industry_chain_entity_id, chain_node_entity_id, position, contextual_stage,
    review_status, status, inclusion_reason, evidence_ids, source_name, source_url, verified_at
) VALUES
    ($1::uuid, $2::uuid, 1, 'downstream', 'approved', 'active',
     '系统节点', ARRAY['evidence:bci-system'], 'integration fixture', 'artifact://bci-system', now()),
    ($1::uuid, $3::uuid, 2, 'midstream', 'approved', 'active',
     '终端组成节点', ARRAY['evidence:bci-terminal'], 'integration fixture', 'artifact://bci-terminal', now()),
    ($1::uuid, $4::uuid, 3, 'upstream', 'approved', 'active',
     '电极组成节点', ARRAY['evidence:bci-electrode'], 'integration fixture', 'artifact://bci-electrode', now())`,
			[]any{testBCIChainID, testBCISystemNodeID, testBCITerminalNodeID, testBCIElectrodeNodeID},
		},
		{
			`INSERT INTO industry_chain_graph_edges (
    id, industry_chain_entity_id, from_chain_node_entity_id, to_chain_node_entity_id,
    relation_type, mechanism, condition_note, segment_kind, omitted_step_note,
    review_status, status, evidence_ids, source_name, source_url, verified_at
) VALUES
    ($1::uuid, $2::uuid, $3::uuid, $4::uuid, 'is_component_of',
     '采集终端是系统组成部分', NULL, 'direct_candidate', NULL,
     'approved', 'active', ARRAY['evidence:bci-terminal-system'],
     'integration fixture', 'artifact://bci-terminal-system', now()),
    ($5::uuid, $2::uuid, $6::uuid, $3::uuid, 'is_component_of',
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
WHERE node.reasoning_tree_id = $1::uuid
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
SET first_seen_at = $1, knowable_at = $1, event_time = $1,
    fact_payload = '{"statement_source":"Integration Source"}'::jsonb
WHERE id = $2`,
			[]any{availableAt, testTypedEventID},
		},
		{
			`INSERT INTO raw_documents (
    id, ingest_channel, source_type, source_name, source_url, title, content_text,
    raw_mime_type, language, published_at, collected_at, content_hash, ingest_status
) VALUES (
    '11000000-0000-4000-8000-000000000001', 'integration', 'filing',
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
    $1, $2, '11000000-0000-4000-8000-000000000001', 'primary',
    'Market supply decreased 10 percent.', $3, 'supports',
    ARRAY['factual_summary','fact_payload'], 3, $4
)`,
			[]any{testTypedEvidenceID, testTypedEventID, strings.Repeat("c", 64), acceptedAt},
		},
		{
			`INSERT INTO event_semantic_context_leases (
    id, event_id, agent_execution_id, worker_id, status, lease_expires_at,
    context_snapshot, leased_at, consumed_at
) VALUES (
    '11000000-0000-4000-8000-000000000003', $1,
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
    $1, '11000000-0000-4000-8000-000000000003', $2,
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
    '11000000-0000-4000-8000-000000000005', $1, $2,
    'event_subject', 'ai', 'accepted', '', $3, 'supply-node',
    '高速光模块', 'resolved', 0.99, ARRAY[$4::uuid], 'semantic', $5, $5
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
    '11000000-0000-4000-8000-000000000005', 'market_supply', 1,
    'decrease', 'actual', ARRAY[$4::uuid], 0.98, 'accepted', $5, $5
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
    '12000000-0000-4000-8000-000000000001', 'integration', 'filing',
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
    $1, $2, '12000000-0000-4000-8000-000000000001', 'primary',
    'The company forecasts demand growth of 15 percent.', $3, 'supports',
    ARRAY['factual_summary','fact_payload'], 3, $4
)`,
			[]any{
				testTypedForwardEvidenceID, testTypedForwardEventID,
				strings.Repeat("f", 64), acceptedAt,
			},
		},
		{
			`INSERT INTO event_semantic_context_leases (
    id, event_id, agent_execution_id, worker_id, status, lease_expires_at,
    context_snapshot, leased_at, consumed_at
) VALUES (
    '12000000-0000-4000-8000-000000000003', $1,
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
    $1, '12000000-0000-4000-8000-000000000003', $2,
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
    '12000000-0000-4000-8000-000000000005', $1, $2,
    'statement_source', 'ai', 'accepted', '', $3, 'forecast-source',
    'Integration Company', 'resolved', 0.99, ARRAY[$4::uuid], 'semantic', $5, $5
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
    '12000000-0000-4000-8000-000000000005', 'market_demand', 1,
    'increase', 'source_forecast', ARRAY[$4::uuid], $5, $6, $7, $6, $7,
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
    '12000000-0000-4000-8000-000000000007', $1, 'relative_change', 'exact',
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

func typedResearchAggregate(now time.Time) researchpublication.Aggregate {
	signalID, submissionID, evidenceID := testTypedSignalID, testTypedSubmissionID, testTypedEvidenceID
	evidenceHash := strings.Repeat("c", 64)
	return researchpublication.Aggregate{
		AnalysisBatchID:      "typed-research-publication",
		AnalysisAsOf:         now.Format(time.RFC3339),
		DiscoveryWindowStart: now.Add(-time.Hour).Format(time.RFC3339),
		DiscoveryWindowEnd:   now.Format(time.RFC3339),
		Theme: researchthemeimport.Theme{
			ThemeKey: "typed-supply", Title: "Typed supply context",
			OneLineConclusion: "Supply decreases", ConclusionDirection: "negative",
			ImpactStrength: "medium", TransmissionStage: "validation",
			InvestmentGuidanceAction:  "observe",
			InvestmentGuidanceSummary: "Observe accepted supply signal",
			TimeHorizonCategory:       "short_term",
			Impacts: []researchthemeimport.Impact{{
				ChainNodeEntityID: testTypedNodeID, RelationRole: "driver",
				ImpactDirection: "negative", DisplayOrder: 1,
			}},
			Events: []researchthemeimport.Event{{
				EventID: testTypedEventID, EvidenceRole: "driver",
			}},
		},
		ReasoningTrees: []researchpublication.ReasoningTree{{
			ReasoningTree: researchreasoningtreeimport.ReasoningTree{
				IndustryChainEntityID: testTypedChainID, Title: "Typed chain",
				DisplayOrder: 1, OneLineConclusion: "Supply decreases",
				ImpactDirection: "negative", ImpactStrength: "medium",
				InvalidationConditions: []string{"Supply recovers"},
				Checkpoints: []researchreasoningtreeimport.Checkpoint{{
					Type: "metric", Summary: "Track market supply",
				}},
				Events: []researchreasoningtreeimport.Event{{
					EventID: testTypedEventID, EvidenceRole: "driver", DisplayOrder: 1,
				}},
			},
			Nodes: []researchpublication.Node{{
				Position: 1, ChainNodeEntityID: testTypedNodeID,
				ImpactDirection: "negative", ImpactStrength: "medium",
				Signals: []researchpublication.Signal{{
					VariableSignalKey: "market_supply", SignalRole: "primary",
					SignalDirection: "decrease", DisplaySummary: "Supply decreases",
					DisplayOrder: 1,
					Lineage: researchpublication.SignalLineage{
						SourceKind: "formal_signal", VariableSignalID: &signalID,
						SemanticSubmissionID: &submissionID, EvidenceID: &evidenceID,
						EvidenceHash: &evidenceHash,
					},
				}},
			}},
		}},
	}
}

func bciReverseResearchAggregate(now time.Time) researchpublication.Aggregate {
	aggregate := typedResearchAggregate(now)
	aggregate.AnalysisBatchID = "bci-reverse-research-publication"
	aggregate.Theme.ThemeKey = "bci-demand"
	aggregate.Theme.Title = "BCI component demand"
	aggregate.Theme.OneLineConclusion = "BCI system demand may propagate to terminal and electrode demand"
	aggregate.Theme.Impacts = []researchthemeimport.Impact{
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
		tree.Nodes = append(tree.Nodes, researchpublication.Node{
			Position:                         step.position,
			ChainNodeEntityID:                step.nodeID,
			ImpactDirection:                  "uncertain",
			ImpactStrength:                   "unknown",
			IncomingIndustryChainGraphEdgeID: &edgeID,
			IncomingTransmissionTitle:        &title,
			IncomingTransmissionMechanism:    &mechanism,
			IncomingConditionSummary:         &condition,
			IncomingLineage: &researchpublication.IncomingLineage{
				SourceKind:               "analyst_inference",
				UpstreamVariableSignalID: &rootSignalID,
			},
			Signals: []researchpublication.Signal{{
				VariableSignalKey: step.signalKey,
				SignalRole:        "primary",
				SignalDirection:   "decrease",
				DisplaySummary:    step.summary,
				DisplayOrder:      1,
				Lineage: researchpublication.SignalLineage{
					SourceKind:               "analyst_inference",
					UpstreamVariableSignalID: &rootSignalID,
					IndustryChainGraphEdgeID: &edgeID,
				},
			}},
		})
	}
	return aggregate
}
