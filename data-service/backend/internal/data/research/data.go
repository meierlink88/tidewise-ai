package research

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	researchbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/research"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

var (
	researchKeyPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9._:-]{0,127}$`)
	researchHashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("Research database is required")
	}
	return &Store{db: db}, nil
}

var _ researchbiz.Repository = (*Store)(nil)
var ErrResearchNotFound = researchbiz.ErrResearchNotFound

type ResearchThemeImpact = researchbiz.ThemeImpactRecord
type ResearchEvent = researchbiz.EventRecord
type ResearchThemeSummary = researchbiz.ThemeSummaryRecord
type ResearchThemeDetail = researchbiz.ThemeDetailRecord
type ResearchThemeListFilter = researchbiz.ThemeListFilter
type ResearchThemePage = researchbiz.ThemeStorePage
type ResearchReadRepository = researchbiz.Repository

const researchThemeSummaryColumns = `
t.id, t.analysis_batch_id, t.title, t.one_line_conclusion,
t.conclusion_direction, t.impact_strength, t.attention_level, t.conclusion_status,
t.transmission_stage, t.investment_guidance_action, t.investment_guidance_summary,
t.time_horizon_category, t.time_horizon_summary, t.transmission_summary,
t.checkpoint_summary, t.risk_summary, t.analysis_as_of, t.window_start, t.window_end,
t.published_at,
COALESCE((
    SELECT jsonb_agg(jsonb_build_object(
        'node_key', impact.node_key,
        'display_name', impact.display_name,
        'relation_role', impact.relation_role,
        'impact_direction', impact.impact_direction,
        'impact_summary', impact.impact_summary,
        'display_order', impact.display_order
    ) ORDER BY impact.display_order)
    FROM research_theme_impacts impact
    WHERE impact.theme_id = t.id
), '[]'::jsonb),
(SELECT count(*) FROM research_theme_events event WHERE event.theme_id = t.id),
(SELECT count(*) FROM research_reasoning_trees tree WHERE tree.theme_id = t.id)`

const listResearchThemesQuery = `
WITH page AS (
    SELECT theme.*
    FROM research_themes theme
    WHERE theme.published_at >= $1
      AND theme.published_at < $2
      AND ($3::timestamptz IS NULL
        OR theme.published_at < $3
        OR (theme.published_at = $3 AND theme.id > $4))
    ORDER BY theme.published_at DESC, theme.id ASC
    LIMIT $5
)
SELECT ` + researchThemeSummaryColumns + `
FROM page t
ORDER BY t.published_at DESC, t.id ASC`

const countResearchThemesQuery = `
SELECT count(DISTINCT theme.id), count(DISTINCT event.event_id)
FROM research_themes theme
LEFT JOIN research_theme_events event ON event.theme_id = theme.id
WHERE theme.published_at >= $1 AND theme.published_at < $2`

const getResearchThemeQuery = `
SELECT ` + researchThemeSummaryColumns + `,
COALESCE((
    SELECT jsonb_agg(jsonb_build_object(
        'event_id', event.id,
		'evidence_ids', association.evidence_ids,
        'title', event.title,
        'summary', event.summary,
        'event_time', event.occurred_at,
        'evidence_role', association.evidence_role,
        'supported_claim', association.supported_claim
    ) ORDER BY event.occurred_at DESC NULLS LAST, event.id)
    FROM research_theme_events association
    JOIN events event ON event.id = association.event_id
    WHERE association.theme_id = t.id
), '[]'::jsonb),
t.theme_key,
(SELECT receipt.publication_mode FROM research_theme_import_receipts receipt WHERE receipt.id = t.import_receipt_id),
(SELECT receipt.publication_contract_version FROM research_theme_import_receipts receipt WHERE receipt.id = t.import_receipt_id)
FROM research_themes t
WHERE t.id = $1`

const getResearchThemeByIDQuery = `
SELECT ` + researchThemeSummaryColumns + `
FROM research_themes t
WHERE t.id = $1`

func (r Store) ListResearchThemes(ctx context.Context, filter ResearchThemeListFilter) (ResearchThemePage, error) {
	rows, err := r.db.QueryContext(ctx, listResearchThemesQuery,
		filter.WindowStart, filter.WindowEnd, nullableTime(filter.CursorPublishedAt),
		nullableString(filter.CursorID), filter.Limit+1)
	if err != nil {
		return ResearchThemePage{}, fmt.Errorf("list research themes: %w", err)
	}
	defer rows.Close()
	items := make([]ResearchThemeSummary, 0, filter.Limit+1)
	for rows.Next() {
		item, err := scanResearchThemeSummary(rows)
		if err != nil {
			return ResearchThemePage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ResearchThemePage{}, fmt.Errorf("iterate research themes: %w", err)
	}
	var themeCount, eventCount int
	if err := r.db.QueryRowContext(ctx, countResearchThemesQuery,
		filter.WindowStart, filter.WindowEnd).Scan(&themeCount, &eventCount); err != nil {
		return ResearchThemePage{}, fmt.Errorf("count research themes: %w", err)
	}
	hasMore := len(items) > filter.Limit
	if hasMore {
		items = items[:filter.Limit]
	}
	return ResearchThemePage{
		AsOf: filter.AsOf, WindowStart: filter.WindowStart, WindowEnd: filter.WindowEnd,
		ThemeCount: themeCount, EventCount: eventCount, Items: items, HasMore: hasMore,
	}, nil
}

func (r Store) GetResearchTheme(ctx context.Context, id string) (ResearchThemeDetail, error) {
	item, err := scanResearchThemeDetail(r.db.QueryRowContext(ctx, getResearchThemeQuery, id))
	if errors.Is(err, sql.ErrNoRows) {
		return ResearchThemeDetail{}, ErrResearchNotFound
	}
	if err != nil {
		return ResearchThemeDetail{}, fmt.Errorf("get research theme: %w", err)
	}
	return item, nil
}

type researchRow interface{ Scan(...any) error }

func scanResearchThemeSummary(row researchRow) (ResearchThemeSummary, error) {
	var item ResearchThemeSummary
	var impacts []byte
	if err := row.Scan(
		&item.ID, &item.AnalysisBatchID, &item.Title, &item.OneLineConclusion,
		&item.ConclusionDirection, &item.ImpactStrength, &item.AttentionLevel,
		&item.ConclusionStatus, &item.TransmissionStage, &item.InvestmentGuidanceAction,
		&item.InvestmentGuidanceSummary, &item.TimeHorizonCategory, &item.TimeHorizonSummary,
		&item.TransmissionSummary, &item.CheckpointSummary, &item.RiskSummary,
		&item.AnalysisAsOf, &item.WindowStart, &item.WindowEnd, &item.PublishedAt,
		&impacts, &item.EvidenceEventCount, &item.ReasoningTreeCount,
	); err != nil {
		return ResearchThemeSummary{}, err
	}
	if err := json.Unmarshal(impacts, &item.Impacts); err != nil {
		return ResearchThemeSummary{}, fmt.Errorf("decode research Theme Impacts: %w", err)
	}
	if err := validatePersistedResearchThemeSummary(item); err != nil {
		return ResearchThemeSummary{}, err
	}
	return item, nil
}

func scanResearchThemeDetail(row researchRow) (ResearchThemeDetail, error) {
	var item ResearchThemeDetail
	var impacts, events []byte
	if err := row.Scan(
		&item.ID, &item.AnalysisBatchID, &item.Title, &item.OneLineConclusion,
		&item.ConclusionDirection, &item.ImpactStrength, &item.AttentionLevel,
		&item.ConclusionStatus, &item.TransmissionStage, &item.InvestmentGuidanceAction,
		&item.InvestmentGuidanceSummary, &item.TimeHorizonCategory, &item.TimeHorizonSummary,
		&item.TransmissionSummary, &item.CheckpointSummary, &item.RiskSummary,
		&item.AnalysisAsOf, &item.WindowStart, &item.WindowEnd, &item.PublishedAt,
		&impacts, &item.EvidenceEventCount, &item.ReasoningTreeCount, &events,
		&item.ThemeKey, &item.PublicationMode, &item.PublicationContractVersion,
	); err != nil {
		return ResearchThemeDetail{}, err
	}
	if err := json.Unmarshal(impacts, &item.Impacts); err != nil {
		return ResearchThemeDetail{}, fmt.Errorf("decode research Theme Impacts: %w", err)
	}
	if len(item.Impacts) == 0 {
		return ResearchThemeDetail{}, fmt.Errorf("research Theme %s has no Impacts", item.ID)
	}
	if err := json.Unmarshal(events, &item.Events); err != nil {
		return ResearchThemeDetail{}, fmt.Errorf("decode research Theme Events: %w", err)
	}
	if item.Events == nil {
		item.Events = []ResearchEvent{}
	}
	if err := validatePersistedResearchThemeDetail(item); err != nil {
		return ResearchThemeDetail{}, err
	}
	return item, nil
}

func validatePersistedResearchThemeSummary(item ResearchThemeSummary) error {
	invalid := func(reason string) error {
		return fmt.Errorf("persisted Research Theme %q violates invariants: %s", item.ID, reason)
	}
	if !coreid.Is(item.ID, coreid.ResearchTheme) || strings.TrimSpace(item.AnalysisBatchID) == "" ||
		strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.OneLineConclusion) == "" ||
		strings.TrimSpace(item.InvestmentGuidanceSummary) == "" {
		return invalid("required identity or text is missing")
	}
	if !oneOf(item.ConclusionDirection, "positive", "negative", "mixed", "neutral", "uncertain") ||
		!oneOf(item.ImpactStrength, "strong", "medium", "weak", "unknown") ||
		!oneOf(item.TransmissionStage, "identification", "validation", "diffusion", "dampening") ||
		!oneOf(item.InvestmentGuidanceAction, "focus", "avoid", "observe", "differentiate") ||
		!oneOf(item.TimeHorizonCategory, "short_term", "medium_term", "long_term", "custom") {
		return invalid("a controlled value is unsupported")
	}
	if item.AttentionLevel != nil && !oneOf(*item.AttentionLevel, "high", "medium", "low") {
		return invalid("attention_level is unsupported")
	}
	if item.ConclusionStatus != nil && !oneOf(*item.ConclusionStatus, "supported", "partial", "conflicted") {
		return invalid("conclusion_status is unsupported")
	}
	if item.AnalysisAsOf.IsZero() || item.WindowStart.IsZero() || item.WindowEnd.IsZero() ||
		item.PublishedAt.IsZero() || !item.WindowStart.Before(item.WindowEnd) ||
		item.EvidenceEventCount < 0 || item.ReasoningTreeCount < 0 || len(item.Impacts) == 0 {
		return invalid("time, count, or Impact cardinality is invalid")
	}
	seen := make(map[string]struct{}, len(item.Impacts))
	for index, impact := range item.Impacts {
		if strings.TrimSpace(impact.NodeKey) == "" || strings.TrimSpace(impact.DisplayName) == "" ||
			impact.DisplayOrder != index+1 || !oneOf(impact.RelationRole, "driver", "beneficiary", "constraint", "exposure") ||
			!oneOf(impact.ImpactDirection, "positive", "negative", "mixed", "neutral", "uncertain") {
			return invalid("an Impact is malformed")
		}
		if _, duplicate := seen[impact.NodeKey]; duplicate {
			return invalid("Impact identity is duplicated")
		}
		seen[impact.NodeKey] = struct{}{}
	}
	return nil
}

func validatePersistedResearchThemeDetail(item ResearchThemeDetail) error {
	if err := validatePersistedResearchThemeSummary(item.ThemeSummaryRecord); err != nil {
		return err
	}
	if !researchKeyPattern.MatchString(item.ThemeKey) ||
		item.PublicationMode != researchbiz.SnapshotPublicationMode ||
		item.PublicationContractVersion != 3 {
		return fmt.Errorf("persisted Research Theme %q has invalid publication identity", item.ID)
	}
	seen := make(map[string]struct{}, len(item.Events))
	for _, event := range item.Events {
		if !coreid.Is(event.EventID, coreid.Event) || strings.TrimSpace(event.Title) == "" ||
			!oneOf(event.EvidenceRole, "driver", "supporting", "contradicting", "context") {
			return fmt.Errorf("persisted Research Theme %q has a malformed Event reference", item.ID)
		}
		if _, duplicate := seen[event.EventID]; duplicate {
			return fmt.Errorf("persisted Research Theme %q has a duplicated Event reference", item.ID)
		}
		seen[event.EventID] = struct{}{}
		for _, evidenceID := range event.EvidenceIDs {
			if !coreid.Is(evidenceID, coreid.EventEvidenceLink) {
				return fmt.Errorf("persisted Research Theme %q has a malformed Evidence reference", item.ID)
			}
		}
	}
	return nil
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

var (
	ErrResearchThemeNotFound          = researchbiz.ErrResearchThemeNotFound
	ErrResearchReasoningTreesNotFound = researchbiz.ErrResearchReasoningTreesNotFound
	ErrResearchReasoningTreeNotFound  = researchbiz.ErrResearchReasoningTreeNotFound
	ErrResearchReasoningTreeInvariant = researchbiz.ErrResearchReasoningTreeInvariant
)

type ResearchReasoningTreeSummary = researchbiz.ReasoningTreeSummaryRecord
type ResearchReasoningTreeList = researchbiz.ReasoningTreeListRecord
type ResearchReasoningTree = researchbiz.ReasoningTreeRecord
type ResearchReasoningTreeDetail = researchbiz.ReasoningTreeDetailRecord

const getResearchReasoningTreePublicationQuery = `
SELECT receipt.id::text, receipt.reasoning_tree_ids_by_tree_key, receipt.write_counts,
       COALESCE((
           SELECT jsonb_agg(jsonb_build_object(
               'ReasoningTreeID', tree.id, 'TreeKey', tree.tree_key,
               'DisplayName', tree.display_name, 'Title', tree.title,
               'DisplayOrder', tree.display_order,
               'EventCount', (SELECT count(*) FROM research_reasoning_tree_events e WHERE e.reasoning_tree_id = tree.id),
               'PublishedAt', receipt.published_at
           ) ORDER BY tree.display_order)
           FROM research_reasoning_trees tree
           WHERE tree.theme_id = receipt.theme_id AND tree.import_receipt_id = receipt.id
       ), '[]'::jsonb),
       (SELECT count(*) FROM research_reasoning_tree_nodes node
        JOIN research_reasoning_trees tree ON tree.id = node.reasoning_tree_id WHERE tree.import_receipt_id = receipt.id),
       (SELECT count(*) FROM research_reasoning_tree_events event
        JOIN research_reasoning_trees tree ON tree.id = event.reasoning_tree_id WHERE tree.import_receipt_id = receipt.id),
       (SELECT count(*) FROM research_reasoning_tree_node_signals signal
        JOIN research_reasoning_tree_nodes node ON node.id = signal.reasoning_tree_node_id
        JOIN research_reasoning_trees tree ON tree.id = node.reasoning_tree_id WHERE tree.import_receipt_id = receipt.id)
FROM research_reasoning_tree_import_receipts receipt
WHERE receipt.theme_id = $1`

const getResearchReasoningTreeDetailQuery = `
SELECT theme.theme_key, receipt.publication_mode, receipt.publication_contract_version,
       tree.id::text, tree.theme_id::text, tree.tree_key, tree.display_name,
       tree.title, tree.display_order, tree.one_line_conclusion,
       tree.fact_summary, tree.transmission_summary, tree.impact_direction,
       tree.impact_strength, tree.impact_summary, tree.conclusion_boundary_summary,
       tree.support_summary, tree.counter_summary, tree.invalidation_conditions,
       tree.checkpoints, receipt.published_at,
       COALESCE((
           SELECT jsonb_agg(jsonb_build_object(
               'event_id', event.id, 'evidence_ids', association.evidence_ids,
               'title', event.title, 'summary', event.summary, 'event_time', event.occurred_at,
               'evidence_role', association.evidence_role, 'display_order', association.display_order
           ) ORDER BY association.display_order)
           FROM research_reasoning_tree_events association
           JOIN events event ON event.id = association.event_id
           WHERE association.reasoning_tree_id = tree.id
       ), '[]'::jsonb),
       COALESCE((
           SELECT jsonb_agg(jsonb_build_object(
               'ID', node.id, 'NodeKey', node.node_key, 'DisplayName', node.display_name,
               'Position', node.position, 'StateSummary', node.state_summary,
               'ImpactDirection', node.impact_direction, 'ImpactStrength', node.impact_strength,
               'ImpactSummary', node.impact_summary, 'ReasoningBasisSummary', node.reasoning_basis_summary,
               'EvidenceGapSummary', node.evidence_gap_summary,
               'IncomingTransmissionTitle', node.incoming_transmission_title,
               'IncomingTransmissionMechanism', node.incoming_transmission_mechanism,
               'IncomingConditionSummary', node.incoming_condition_summary,
               'Signals', COALESCE((
                   SELECT jsonb_agg(jsonb_build_object(
                       'SignalKey', signal.signal_key, 'VariableName', signal.variable_name,
                       'Direction', signal.signal_direction, 'SignalRole', signal.signal_role,
                       'DisplaySummary', signal.display_summary, 'DisplayOrder', signal.display_order
                   ) ORDER BY signal.display_order)
                   FROM research_reasoning_tree_node_signals signal
                   WHERE signal.reasoning_tree_node_id = node.id
               ), '[]'::jsonb)
           ) ORDER BY node.position)
           FROM research_reasoning_tree_nodes node
           WHERE node.reasoning_tree_id = tree.id
       ), '[]'::jsonb),
       COALESCE((
           SELECT jsonb_agg(impact.node_key ORDER BY impact.display_order)
           FROM research_theme_impacts impact
           WHERE impact.theme_id = tree.theme_id
             AND EXISTS (SELECT 1 FROM research_reasoning_tree_nodes impact_node
                         WHERE impact_node.reasoning_tree_id = tree.id AND impact_node.node_key = impact.node_key)
       ), '[]'::jsonb),
       (SELECT count(*) FROM research_reasoning_tree_events event
        WHERE event.reasoning_tree_id = tree.id
          AND NOT EXISTS (SELECT 1 FROM research_theme_events theme_event
                          WHERE theme_event.theme_id = tree.theme_id AND theme_event.event_id = event.event_id))
FROM research_reasoning_trees tree
JOIN research_reasoning_tree_import_receipts receipt ON receipt.id = tree.import_receipt_id
JOIN research_themes theme ON theme.id = tree.theme_id
WHERE tree.theme_id = $1 AND tree.id = $2`

type researchReasoningTreePublication struct {
	ReceiptID string
	Mapping   map[string]string
	Counts    researchbiz.ReasonTreeCounts
	Trees     []ResearchReasoningTreeSummary
}

func (r Store) ListResearchThemeReasoningTrees(ctx context.Context, themeID string) (ResearchReasoningTreeList, error) {
	theme, err := r.readResearchReasoningTreeTheme(ctx, themeID)
	if err != nil {
		return ResearchReasoningTreeList{}, err
	}
	publication, err := r.readResearchReasoningTreePublication(ctx, themeID)
	if err != nil {
		return ResearchReasoningTreeList{}, err
	}
	return ResearchReasoningTreeList{Theme: theme, ReasoningTrees: publication.Trees}, nil
}

func (r Store) GetResearchThemeReasoningTree(ctx context.Context, themeID, reasoningTreeID string) (ResearchReasoningTreeDetail, error) {
	if _, err := r.readResearchReasoningTreeTheme(ctx, themeID); err != nil {
		return ResearchReasoningTreeDetail{}, err
	}
	publication, err := r.readResearchReasoningTreePublication(ctx, themeID)
	if err != nil {
		return ResearchReasoningTreeDetail{}, err
	}
	if !publicationContainsReasoningTree(publication.Mapping, reasoningTreeID) {
		return ResearchReasoningTreeDetail{}, ErrResearchReasoningTreeNotFound
	}
	var result ResearchReasoningTreeDetail
	var tree ResearchReasoningTree
	var invalidationJSON, checkpointsJSON, eventsJSON, nodesJSON, impactIDsJSON []byte
	var invalidThemeEventCount int
	err = r.db.QueryRowContext(ctx, getResearchReasoningTreeDetailQuery,
		themeID, reasoningTreeID).Scan(
		&result.ThemeKey, &result.PublicationMode, &result.PublicationContractVersion,
		&tree.ReasoningTreeID, &tree.ThemeID, &tree.TreeKey, &tree.DisplayName,
		&tree.Title, &tree.DisplayOrder,
		&tree.OneLineConclusion, &tree.FactSummary, &tree.TransmissionSummary,
		&tree.ImpactDirection, &tree.ImpactStrength, &tree.ImpactSummary,
		&tree.ConclusionBoundarySummary, &tree.SupportSummary, &tree.CounterSummary,
		&invalidationJSON, &checkpointsJSON, &tree.PublishedAt,
		&eventsJSON, &nodesJSON, &impactIDsJSON, &invalidThemeEventCount,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ResearchReasoningTreeDetail{}, ErrResearchReasoningTreeInvariant
	}
	if err != nil {
		return ResearchReasoningTreeDetail{}, fmt.Errorf("get research reasoning tree: %w", err)
	}
	var impactNodeIDs []string
	if err := json.Unmarshal(invalidationJSON, &tree.InvalidationConditions); err != nil {
		return ResearchReasoningTreeDetail{}, ErrResearchReasoningTreeInvariant
	}
	if err := json.Unmarshal(checkpointsJSON, &tree.Checkpoints); err != nil {
		return ResearchReasoningTreeDetail{}, ErrResearchReasoningTreeInvariant
	}
	if err := json.Unmarshal(eventsJSON, &tree.Events); err != nil {
		return ResearchReasoningTreeDetail{}, ErrResearchReasoningTreeInvariant
	}
	if err := json.Unmarshal(nodesJSON, &tree.Nodes); err != nil {
		return ResearchReasoningTreeDetail{}, ErrResearchReasoningTreeInvariant
	}
	if err := json.Unmarshal(impactIDsJSON, &impactNodeIDs); err != nil {
		return ResearchReasoningTreeDetail{}, ErrResearchReasoningTreeInvariant
	}
	tree.EventCount = len(tree.Events)
	if tree.ThemeID != themeID || invalidThemeEventCount != 0 ||
		!validReasoningTreeDetail(result, tree, impactNodeIDs) {
		return ResearchReasoningTreeDetail{}, ErrResearchReasoningTreeInvariant
	}
	result.ThemeID = themeID
	result.ImpactNodeIDs = impactNodeIDs
	result.ReasoningTree = tree
	return result, nil
}

func (r Store) readResearchReasoningTreeTheme(ctx context.Context, themeID string) (ResearchThemeSummary, error) {
	theme, err := scanResearchThemeSummary(r.db.QueryRowContext(ctx, getResearchThemeByIDQuery, themeID))
	if errors.Is(err, sql.ErrNoRows) {
		return ResearchThemeSummary{}, ErrResearchThemeNotFound
	}
	if err != nil {
		return ResearchThemeSummary{}, fmt.Errorf("get research reasoning tree Theme: %w", err)
	}
	return theme, nil
}

func (r Store) readResearchReasoningTreePublication(ctx context.Context, themeID string) (researchReasoningTreePublication, error) {
	var publication researchReasoningTreePublication
	var mappingJSON, countsJSON, treesJSON []byte
	var nodeCount, eventCount, signalCount int
	err := r.db.QueryRowContext(ctx, getResearchReasoningTreePublicationQuery, themeID).Scan(
		&publication.ReceiptID, &mappingJSON, &countsJSON, &treesJSON,
		&nodeCount, &eventCount, &signalCount,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return researchReasoningTreePublication{}, ErrResearchReasoningTreesNotFound
	}
	if err != nil {
		return researchReasoningTreePublication{}, fmt.Errorf("get research reasoning tree publication: %w", err)
	}
	if json.Unmarshal(mappingJSON, &publication.Mapping) != nil ||
		json.Unmarshal(countsJSON, &publication.Counts) != nil ||
		json.Unmarshal(treesJSON, &publication.Trees) != nil ||
		!validReasoningTreePublication(publication, nodeCount, eventCount, signalCount) {
		return researchReasoningTreePublication{}, ErrResearchReasoningTreeInvariant
	}
	return publication, nil
}

func validReasoningTreePublication(publication researchReasoningTreePublication, nodeCount, eventCount, signalCount int) bool {
	if !coreid.Is(publication.ReceiptID, coreid.ResearchReasoningTreeReceipt) || len(publication.Mapping) == 0 ||
		len(publication.Trees) == 0 || nodeCount < 1 || eventCount < 0 || signalCount < 1 {
		return false
	}
	if publication.Counts != (researchbiz.ReasonTreeCounts{
		ReasoningTrees: len(publication.Trees), Nodes: nodeCount,
		EventAssociations: eventCount, SignalAssociations: signalCount, Receipts: 1,
	}) {
		return false
	}
	actual := make(map[string]string, len(publication.Trees))
	seenTreeIDs := make(map[string]struct{}, len(publication.Trees))
	for index, tree := range publication.Trees {
		if !coreid.Is(tree.ReasoningTreeID, coreid.ResearchReasoningTree) ||
			!researchKeyPattern.MatchString(tree.TreeKey) || tree.DisplayOrder != index+1 ||
			tree.EventCount < 0 || tree.PublishedAt.IsZero() || strings.TrimSpace(tree.Title) == "" ||
			strings.TrimSpace(tree.DisplayName) == "" {
			return false
		}
		if _, duplicate := actual[tree.TreeKey]; duplicate {
			return false
		}
		if _, duplicate := seenTreeIDs[tree.ReasoningTreeID]; duplicate {
			return false
		}
		actual[tree.TreeKey] = tree.ReasoningTreeID
		seenTreeIDs[tree.ReasoningTreeID] = struct{}{}
	}
	for key, id := range publication.Mapping {
		if !researchKeyPattern.MatchString(key) || !coreid.Is(id, coreid.ResearchReasoningTree) {
			return false
		}
	}
	return reflect.DeepEqual(actual, publication.Mapping)
}

func publicationContainsReasoningTree(mapping map[string]string, reasoningTreeID string) bool {
	for _, value := range mapping {
		if value == reasoningTreeID {
			return true
		}
	}
	return false
}

func validReasoningTreeDetail(detail ResearchReasoningTreeDetail, tree ResearchReasoningTree, impactNodeIDs []string) bool {
	if detail.PublicationMode != researchbiz.SnapshotPublicationMode || detail.PublicationContractVersion != 3 ||
		!researchKeyPattern.MatchString(detail.ThemeKey) ||
		!coreid.Is(tree.ReasoningTreeID, coreid.ResearchReasoningTree) || !coreid.Is(tree.ThemeID, coreid.ResearchTheme) ||
		!researchKeyPattern.MatchString(tree.TreeKey) || strings.TrimSpace(tree.DisplayName) == "" ||
		strings.TrimSpace(tree.Title) == "" || strings.TrimSpace(tree.OneLineConclusion) == "" ||
		!oneOf(tree.ImpactDirection, "positive", "negative", "mixed", "neutral", "uncertain") ||
		!oneOf(tree.ImpactStrength, "strong", "medium", "weak", "unknown") ||
		tree.DisplayOrder < 1 || tree.EventCount != len(tree.Events) || tree.PublishedAt.IsZero() ||
		len(tree.Nodes) == 0 || len(impactNodeIDs) == 0 {
		return false
	}
	for _, checkpoint := range tree.Checkpoints {
		if !oneOf(checkpoint.Type, "event", "relationship", "metric") || strings.TrimSpace(checkpoint.Summary) == "" {
			return false
		}
	}
	seenEvents := make(map[string]struct{}, len(tree.Events))
	for index, event := range tree.Events {
		if !coreid.Is(event.EventID, coreid.Event) || event.DisplayOrder != index+1 ||
			strings.TrimSpace(event.Title) == "" ||
			!oneOf(event.EvidenceRole, "driver", "supporting", "contradicting", "context") {
			return false
		}
		if _, duplicate := seenEvents[event.EventID]; duplicate {
			return false
		}
		seenEvents[event.EventID] = struct{}{}
		evidenceIDs := make(map[string]struct{}, len(event.EvidenceIDs))
		for _, evidenceID := range event.EvidenceIDs {
			if !coreid.Is(evidenceID, coreid.EventEvidenceLink) {
				return false
			}
			if _, duplicate := evidenceIDs[evidenceID]; duplicate {
				return false
			}
			evidenceIDs[evidenceID] = struct{}{}
		}
	}
	impactSet := make(map[string]struct{}, len(impactNodeIDs))
	for _, id := range impactNodeIDs {
		if !researchKeyPattern.MatchString(id) {
			return false
		}
		if _, duplicate := impactSet[id]; duplicate {
			return false
		}
		impactSet[id] = struct{}{}
	}
	matchedImpacts := 0
	seenNodes := make(map[string]struct{}, len(tree.Nodes))
	seenNodeIDs := make(map[string]struct{}, len(tree.Nodes))
	for index, node := range tree.Nodes {
		if !coreid.Is(node.ID, coreid.ResearchReasoningTreeNode) || !researchKeyPattern.MatchString(node.NodeKey) ||
			strings.TrimSpace(node.DisplayName) == "" || node.Position != index+1 ||
			!oneOf(node.ImpactDirection, "positive", "negative", "mixed", "neutral", "uncertain") ||
			!oneOf(node.ImpactStrength, "strong", "medium", "weak", "unknown") ||
			len(node.Signals) < 1 || len(node.Signals) > 5 {
			return false
		}
		if _, duplicate := seenNodes[node.NodeKey]; duplicate {
			return false
		}
		if _, duplicate := seenNodeIDs[node.ID]; duplicate {
			return false
		}
		seenNodes[node.NodeKey] = struct{}{}
		seenNodeIDs[node.ID] = struct{}{}
		if _, impact := impactSet[node.NodeKey]; impact {
			matchedImpacts++
		}
		primaryCount := 0
		seenSignals := make(map[string]struct{}, len(node.Signals))
		for signalIndex, signal := range node.Signals {
			if !researchKeyPattern.MatchString(signal.SignalKey) || strings.TrimSpace(signal.DisplaySummary) == "" ||
				signal.DisplayOrder != signalIndex+1 || !oneOf(signal.SignalRole, "primary", "supporting", "contradicting") ||
				(signal.Direction != nil && !oneOf(*signal.Direction, "increase", "decrease", "mixed", "unchanged", "uncertain")) {
				return false
			}
			if _, duplicate := seenSignals[signal.SignalKey]; duplicate {
				return false
			}
			seenSignals[signal.SignalKey] = struct{}{}
			if signal.SignalRole == "primary" {
				primaryCount++
				if signal.DisplayOrder != 1 {
					return false
				}
			}
		}
		if primaryCount != 1 {
			return false
		}
		if index == 0 {
			if node.IncomingTransmissionTitle != nil || node.IncomingTransmissionMechanism != nil || node.IncomingConditionSummary != nil {
				return false
			}
		} else if node.IncomingTransmissionMechanism == nil || strings.TrimSpace(*node.IncomingTransmissionMechanism) == "" {
			return false
		}
	}
	return matchedImpacts == len(impactSet)
}
