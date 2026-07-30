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
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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
	if len(secondThemePage.Items[0].Impacts) != 1 ||
		secondThemePage.Items[0].Impacts[0].PrimarySignalDisplaySummary == nil ||
		*secondThemePage.Items[0].Impacts[0].PrimarySignalDisplaySummary !=
			aggregate.Theme.Impacts[0].PrimarySignalDisplaySummary {
		t.Fatalf(
			"published Theme impact = %#v, want primary signal display summary %q",
			secondThemePage.Items[0].Impacts,
			aggregate.Theme.Impacts[0].PrimarySignalDisplaySummary,
		)
	}

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
)

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
    id, event_id, raw_document_id, source_level, evidence_excerpt, evidence_hash,
    evidence_relation, supports_fields, is_primary, created_at
) VALUES (
    $1, $2, '11000000-0000-4000-8000-000000000001', 'primary',
    'Market supply decreased 10 percent.', $3, 'supports',
    ARRAY['summary','fact_payload'], true, $4
)`,
			[]any{testTypedEvidenceID, testTypedEventID, strings.Repeat("c", 64), acceptedAt},
		},
		{
			`UPDATE events SET primary_source_id = $1 WHERE id = $2`,
			[]any{testTypedEvidenceID, testTypedEventID},
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
    id, event_id, raw_document_id, source_level, evidence_excerpt, evidence_hash,
    evidence_relation, supports_fields, is_primary, created_at
) VALUES (
    $1, $2, '12000000-0000-4000-8000-000000000001', 'primary',
    'The company forecasts demand growth of 15 percent.', $3, 'supports',
    ARRAY['summary','fact_payload'], true, $4
)`,
			[]any{
				testTypedForwardEvidenceID, testTypedForwardEventID,
				strings.Repeat("f", 64), acceptedAt,
			},
		},
		{
			`UPDATE events SET primary_source_id = $1 WHERE id = $2`,
			[]any{testTypedForwardEvidenceID, testTypedForwardEventID},
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
				ImpactDirection: "negative", PrimarySignalDisplaySummary: "Market supply: decreases",
				DisplayOrder: 1,
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
