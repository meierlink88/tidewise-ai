package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchanalysiscontext"
)

type ResearchAnalysisContextStore struct {
	db *sql.DB
}

func NewResearchAnalysisContextStore(db *sql.DB) *ResearchAnalysisContextStore {
	return &ResearchAnalysisContextStore{db: db}
}

func (s *ResearchAnalysisContextStore) List(
	ctx context.Context,
	query researchanalysiscontext.StoreQuery,
) (researchanalysiscontext.StorePage, error) {
	if s == nil || s.db == nil {
		return researchanalysiscontext.StorePage{}, errors.New("research Analysis Context database is required")
	}
	var historicalGap bool
	if err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS (
		    SELECT 1
		    FROM event_semantic_submissions submission
		    JOIN events event ON event.id = submission.event_id
		    WHERE submission.status = 'superseded'
		      AND submission.created_at <= $3
		      AND submission.finalized_at > $3
		      AND COALESCE(event.knowable_at, event.first_seen_at) >= $1
		      AND COALESCE(event.knowable_at, event.first_seen_at) < $2
		      AND COALESCE(event.knowable_at, event.first_seen_at) <= $3
		)
	`, query.DiscoveryWindowStart, query.DiscoveryWindowEnd, query.AnalysisAsOf).Scan(&historicalGap); err != nil {
		return researchanalysiscontext.StorePage{}, err
	}
	if historicalGap {
		return researchanalysiscontext.StorePage{},
			researchanalysiscontext.ErrHistoricalSemanticsUnavailable
	}
	if err := s.preflightDictionaryBudget(ctx, query.AnalysisAsOf); err != nil {
		return researchanalysiscontext.StorePage{}, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id::text, COALESCE(knowable_at, first_seen_at)
		FROM events
		WHERE event_status = 'confirmed'
		  AND fact_status = 'verified'
		  AND COALESCE(knowable_at, first_seen_at) >= $1
		  AND COALESCE(knowable_at, first_seen_at) < $2
		  AND COALESCE(knowable_at, first_seen_at) <= $3
		  AND (
		      $4::timestamptz IS NULL
		      OR (COALESCE(knowable_at, first_seen_at), id)
		         > ($4::timestamptz, $5::uuid)
		  )
		ORDER BY COALESCE(knowable_at, first_seen_at), id
		LIMIT $6
	`,
		query.DiscoveryWindowStart, query.DiscoveryWindowEnd, query.AnalysisAsOf,
		query.AfterKnowledgeAvailableAt, nullUUID(query.AfterEventID), query.PageSize+1,
	)
	if err != nil {
		return researchanalysiscontext.StorePage{}, err
	}
	defer rows.Close()
	type selectedEvent struct {
		id        string
		available time.Time
	}
	selected := make([]selectedEvent, 0, query.PageSize+1)
	for rows.Next() {
		var item selectedEvent
		if err := rows.Scan(&item.id, &item.available); err != nil {
			return researchanalysiscontext.StorePage{}, err
		}
		selected = append(selected, item)
	}
	if err := rows.Err(); err != nil {
		return researchanalysiscontext.StorePage{}, err
	}
	hasMore := len(selected) > query.PageSize
	if hasMore {
		selected = selected[:query.PageSize]
	}
	page := researchanalysiscontext.StorePage{
		Bundles: make([]researchanalysiscontext.BundleRecord, 0, len(selected)),
		HasMore: hasMore,
	}
	for _, event := range selected {
		if err := s.preflightBundleBudget(ctx, event.id, query.AnalysisAsOf); err != nil {
			return researchanalysiscontext.StorePage{}, err
		}
		payload, err := s.eventBundle(ctx, event.id, query.AnalysisAsOf)
		if err != nil {
			return researchanalysiscontext.StorePage{}, err
		}
		page.Bundles = append(page.Bundles, researchanalysiscontext.BundleRecord{
			KnowledgeAvailableAt: event.available.UTC(),
			EventID:              event.id,
			Bundle:               payload,
		})
	}
	var dictionaryPayload []byte
	page.Dictionaries, dictionaryPayload, err = s.dictionaries(ctx, query.AnalysisAsOf)
	if err != nil {
		return researchanalysiscontext.StorePage{}, err
	}
	dictionaryHash := sha256.Sum256(dictionaryPayload)
	page.DictionaryFingerprint = fmt.Sprintf("%x", dictionaryHash)
	return page, nil
}

func (s *ResearchAnalysisContextStore) preflightDictionaryBudget(
	ctx context.Context,
	analysisAsOf time.Time,
) error {
	var rows, bytes int64
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(sum(item_count), 0), COALESCE(sum(item_bytes), 0)
		FROM (
		    SELECT count(*)::bigint item_count,
		           COALESCE(sum(pg_column_size(entity)), 0)::bigint item_bytes
		    FROM entity_nodes entity
		    WHERE entity.status = 'active'
		      AND entity.created_at <= $1 AND entity.updated_at <= $1
		    UNION ALL
		    SELECT count(*)::bigint, COALESCE(sum(pg_column_size(relation)), 0)::bigint
		    FROM entity_edges relation
		    WHERE relation.status = 'active'
		      AND relation.created_at <= $1 AND relation.updated_at <= $1
		    UNION ALL
		    SELECT count(*)::bigint, COALESCE(sum(pg_column_size(definition)), 0)::bigint
		    FROM industry_chain_definitions definition
		    WHERE definition.review_status = 'approved'
		      AND definition.created_at <= $1 AND definition.updated_at <= $1
		    UNION ALL
		    SELECT count(*)::bigint, COALESCE(sum(pg_column_size(membership)), 0)::bigint
		    FROM industry_chain_node_memberships membership
		    WHERE membership.review_status = 'approved' AND membership.status = 'active'
		      AND membership.created_at <= $1 AND membership.updated_at <= $1
		    UNION ALL
		    SELECT count(*)::bigint, COALESCE(sum(pg_column_size(edge)), 0)::bigint
		    FROM industry_chain_graph_edges edge
		    WHERE edge.review_status = 'approved' AND edge.status = 'active'
		      AND edge.created_at <= $1 AND edge.updated_at <= $1
		    UNION ALL
		    SELECT count(*)::bigint, COALESCE(sum(pg_column_size(definition)), 0)::bigint
		    FROM entity_type_definitions definition
		    WHERE definition.status = 'active' AND definition.created_at <= $1
		    UNION ALL
		    SELECT count(*)::bigint, COALESCE(sum(pg_column_size(definition)), 0)::bigint
		    FROM variable_definitions definition
		    WHERE definition.status = 'active' AND definition.created_at <= $1
		    UNION ALL
		    SELECT count(*)::bigint, COALESCE(sum(pg_column_size(applicable)), 0)::bigint
		    FROM variable_definition_entity_types applicable
		    WHERE applicable.created_at <= $1
		    UNION ALL
		    SELECT count(*)::bigint, COALESCE(sum(pg_column_size(rule)), 0)::bigint
		    FROM direct_transmission_rules rule
		    WHERE rule.status = 'approved'
		      AND rule.created_at <= $1
		      AND COALESCE(rule.reviewed_at, rule.created_at) <= $1
		    UNION ALL
		    SELECT count(*)::bigint, COALESCE(sum(pg_column_size(policy)), 0)::bigint
		    FROM event_semantic_acceptance_policies policy
		    WHERE policy.status = 'active' AND policy.created_at <= $1
		) budget
	`, analysisAsOf).Scan(&rows, &bytes)
	if err != nil {
		return err
	}
	if rows > researchanalysiscontext.MaxDictionaryRows ||
		bytes > researchanalysiscontext.MaxDictionaryBytes {
		return &researchanalysiscontext.ResourceLimitError{
			Reason: "Research Analysis Context dictionaries exceed the preflight budget",
		}
	}
	return nil
}

func (s *ResearchAnalysisContextStore) preflightBundleBudget(
	ctx context.Context,
	eventID string,
	analysisAsOf time.Time,
) error {
	var rows, bytes int64
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(sum(item_count), 0), COALESCE(sum(item_bytes), 0)
		FROM (
		    SELECT count(*)::bigint item_count,
		           COALESCE(sum(
		               pg_column_size(source)
		               + octet_length(COALESCE(document.source_name, ''))
		               + octet_length(COALESCE(document.source_type, ''))
		               + octet_length(COALESCE(document.source_url, ''))
		               + octet_length(COALESCE(document.title, ''))
		           ), 0)::bigint item_bytes
		    FROM event_sources source
		    JOIN raw_documents document ON document.id = source.raw_document_id
		    WHERE source.event_id = $1::uuid
		      AND source.created_at <= $2
		      AND GREATEST(
		          COALESCE(document.published_at, document.collected_at),
		          document.collected_at
		      ) <= $2
		    UNION ALL
		    SELECT count(*)::bigint, COALESCE(sum(pg_column_size(link)), 0)::bigint
		    FROM event_entity_links link
		    JOIN event_semantic_submissions submission
		      ON submission.id = link.semantic_submission_id
		    WHERE link.event_id = $1::uuid
		      AND link.review_status = 'accepted'
		      AND submission.status = 'accepted'
		      AND link.updated_at <= $2
		      AND COALESCE(submission.finalized_at, submission.created_at) <= $2
		    UNION ALL
		    SELECT count(*)::bigint, COALESCE(sum(pg_column_size(signal)), 0)::bigint
		    FROM variable_signals signal
		    JOIN event_semantic_submissions submission
		      ON submission.id = signal.semantic_submission_id
		    WHERE signal.source_event_id = $1::uuid
		      AND signal.review_status = 'accepted'
		      AND submission.status = 'accepted'
		      AND signal.updated_at <= $2
		      AND COALESCE(submission.finalized_at, submission.created_at) <= $2
		    UNION ALL
		    SELECT count(*)::bigint, COALESCE(sum(pg_column_size(measurement)), 0)::bigint
		    FROM variable_signal_measurements measurement
		    JOIN variable_signals signal ON signal.id = measurement.variable_signal_id
		    JOIN event_semantic_submissions submission
		      ON submission.id = signal.semantic_submission_id
		    WHERE signal.source_event_id = $1::uuid
		      AND signal.review_status = 'accepted'
		      AND submission.status = 'accepted'
		      AND measurement.created_at <= $2
		      AND COALESCE(submission.finalized_at, submission.created_at) <= $2
		    UNION ALL
		    SELECT count(*)::bigint, COALESCE(sum(pg_column_size(impact)), 0)::bigint
		    FROM direct_impact_assertions impact
		    JOIN variable_signals source_signal
		      ON source_signal.id = impact.source_variable_signal_id
		    JOIN event_semantic_submissions submission
		      ON submission.id = impact.semantic_submission_id
		    WHERE source_signal.source_event_id = $1::uuid
		      AND impact.review_status = 'accepted'
		      AND submission.status = 'accepted'
		      AND impact.updated_at <= $2
		      AND COALESCE(submission.finalized_at, submission.created_at) <= $2
		) budget
	`, eventID, analysisAsOf).Scan(&rows, &bytes)
	if err != nil {
		return err
	}
	if rows > researchanalysiscontext.MaxEventSemanticBundleRows ||
		bytes > researchanalysiscontext.MaxEventSemanticBundleBytes {
		return &researchanalysiscontext.ResourceLimitError{
			Reason: "an Event Semantic Bundle exceeds the preflight budget",
		}
	}
	return nil
}

func (s *ResearchAnalysisContextStore) eventBundle(
	ctx context.Context,
	eventID string,
	analysisAsOf time.Time,
) (researchanalysiscontext.EventSemanticBundle, error) {
	var payload []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT jsonb_build_object(
		    'event', jsonb_build_object(
		        'id', event.id,
		        'title', event.title,
		        'summary', event.summary,
		        'occurred_at', event.event_time,
		        'first_seen_at', event.first_seen_at,
		        'knowledge_available_at', COALESCE(event.knowable_at, event.first_seen_at),
		        'event_status', event.event_status,
		        'fact_status', event.fact_status
		    ),
		    'evidence', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'evidence_id', source.id,
		            'evidence_hash', source.evidence_hash,
		            'excerpt', source.evidence_excerpt,
		            'source_level', source.source_level,
		            'relation', source.evidence_relation,
		            'supports_fields', source.supports_fields,
		            'is_primary', COALESCE(source.is_primary, false),
		            'raw_document_id', document.id,
		            'source_name', document.source_name,
		            'source_type', document.source_type,
		            'source_url', document.source_url,
		            'title', document.title,
		            'published_at', document.published_at,
		            'first_seen_at', document.collected_at,
		            'knowledge_available_at', GREATEST(
		                COALESCE(document.published_at, document.collected_at),
		                document.collected_at
		            ),
		            'accepted_at', source.created_at,
		            'statement_source', COALESCE(
		                NULLIF(event.fact_payload ->> 'statement_source', ''),
		                ''
		            )
		        ) ORDER BY COALESCE(source.is_primary, false) DESC, source.id)
		        FROM event_sources source
		        JOIN raw_documents document ON document.id = source.raw_document_id
		        WHERE source.event_id = event.id
		          AND GREATEST(
		              COALESCE(document.published_at, document.collected_at),
		              document.collected_at
		          ) <= $2
		          AND source.created_at <= $2
		    ), '[]'::jsonb),
		    'entity_links', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'event_entity_link_id', link.id,
		            'semantic_submission_id', link.semantic_submission_id,
		            'entity_id', link.entity_id,
		            'entity_role', link.entity_role,
		            'resolved_mention', link.resolved_mention,
		            'resolution_method', link.resolution_method,
		            'resolution_confidence', link.resolution_confidence,
		            'evidence_ids', link.evidence_ids,
		            'review_status', link.review_status
		        ) ORDER BY link.entity_role, link.entity_id, link.id)
		        FROM event_entity_links link
		        JOIN event_semantic_submissions submission
		          ON submission.id = link.semantic_submission_id
		        WHERE link.event_id = event.id
		          AND link.review_status = 'accepted'
		          AND submission.status = 'accepted'
		          AND link.updated_at <= $2
		          AND COALESCE(submission.finalized_at, submission.created_at) <= $2
		          AND NOT EXISTS (
		              SELECT 1
		              FROM unnest(link.evidence_ids) evidence_id
		              LEFT JOIN event_sources linked_source ON linked_source.id = evidence_id
		              LEFT JOIN raw_documents linked_document
		                ON linked_document.id = linked_source.raw_document_id
		              WHERE linked_source.id IS NULL
		                 OR GREATEST(
		                     COALESCE(linked_document.published_at, linked_document.collected_at),
		                     linked_document.collected_at
		                 ) > $2
		          )
		    ), '[]'::jsonb),
		    'variable_signals', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'variable_signal_id', signal.id,
		            'semantic_submission_id', signal.semantic_submission_id,
		            'source_event_id', signal.source_event_id,
		            'subject_event_entity_link_id', signal.subject_event_entity_link_id,
		            'subject_entity_id', subject.entity_id,
		            'variable_key', signal.variable_key,
		            'variable_version', signal.variable_version,
		            'direction', signal.direction,
		            'assertion_modality', signal.assertion_modality,
		            'evidence_ids', signal.evidence_ids,
		            'statement_at', signal.statement_at,
		            'valid_from', signal.valid_from,
		            'valid_until', signal.valid_until,
		            'forecast_period_start', signal.forecast_period_start,
		            'forecast_period_end', signal.forecast_period_end,
		            'extraction_confidence', signal.extraction_confidence,
		            'review_status', signal.review_status,
		            'measurements', COALESCE((
		                SELECT jsonb_agg(jsonb_build_object(
		                    'measurement_id', measurement.id,
		                    'measurement_role', measurement.measurement_role,
		                    'value_shape', measurement.value_shape,
		                    'raw_value', measurement.raw_value::text,
		                    'raw_lower', measurement.raw_lower::text,
		                    'raw_upper', measurement.raw_upper::text,
		                    'raw_unit', measurement.raw_unit,
		                    'canonical_value', measurement.canonical_value::text,
		                    'canonical_lower', measurement.canonical_lower::text,
		                    'canonical_upper', measurement.canonical_upper::text,
		                    'canonical_unit', measurement.canonical_unit,
		                    'currency', measurement.currency,
		                    'scale', measurement.scale,
		                    'comparison_basis', measurement.comparison_basis,
		                    'comparison_period', measurement.comparison_period,
		                    'raw_text', measurement.raw_text,
		                    'is_approximate', measurement.is_approximate,
		                    'evidence_id', measurement.evidence_id
		                ) ORDER BY measurement.id)
		                FROM variable_signal_measurements measurement
		                WHERE measurement.variable_signal_id = signal.id
		            ), '[]'::jsonb),
		            'direct_impacts', COALESCE((
		                SELECT jsonb_agg(jsonb_build_object(
		                    'direct_impact_assertion_id', impact.id,
		                    'semantic_submission_id', impact.semantic_submission_id,
		                    'source_variable_signal_id', impact.source_variable_signal_id,
		                    'target_entity_id', impact.target_entity_id,
		                    'affected_variable_key', impact.affected_variable_key,
		                    'affected_variable_version', impact.affected_variable_version,
		                    'affected_direction', impact.affected_direction,
		                    'derivation_type', impact.derivation_type,
		                    'mechanism_summary', impact.mechanism_summary,
		                    'evidence_ids', impact.evidence_ids,
		                    'entity_relation_id', impact.entity_relation_id,
		                    'rule_key', impact.rule_key,
		                    'rule_version', impact.rule_version,
		                    'assertion_confidence', impact.assertion_confidence,
		                    'effective_from', impact.effective_from,
		                    'effective_to', impact.effective_to,
		                    'review_status', impact.review_status
		                ) ORDER BY impact.target_entity_id, impact.affected_variable_key, impact.id)
		                FROM direct_impact_assertions impact
		                JOIN event_semantic_submissions impact_submission
		                  ON impact_submission.id = impact.semantic_submission_id
		                WHERE impact.source_variable_signal_id = signal.id
		                  AND impact.review_status = 'accepted'
		                  AND impact_submission.status = 'accepted'
		                  AND impact.updated_at <= $2
		                  AND COALESCE(
		                      impact_submission.finalized_at,
		                      impact_submission.created_at
		                  ) <= $2
		                  AND NOT EXISTS (
		                      SELECT 1
		                      FROM unnest(impact.evidence_ids) evidence_id
		                      LEFT JOIN event_sources linked_source ON linked_source.id = evidence_id
		                      LEFT JOIN raw_documents linked_document
		                        ON linked_document.id = linked_source.raw_document_id
		                      WHERE linked_source.id IS NULL
		                         OR GREATEST(
		                             COALESCE(linked_document.published_at, linked_document.collected_at),
		                             linked_document.collected_at
		                         ) > $2
		                  )
		            ), '[]'::jsonb)
		        ) ORDER BY signal.created_at, signal.id)
		        FROM variable_signals signal
		        JOIN event_entity_links subject
		          ON subject.id = signal.subject_event_entity_link_id
		        JOIN event_semantic_submissions submission
		          ON submission.id = signal.semantic_submission_id
		        WHERE signal.source_event_id = event.id
		          AND signal.review_status = 'accepted'
		          AND submission.status = 'accepted'
		          AND signal.updated_at <= $2
		          AND COALESCE(submission.finalized_at, submission.created_at) <= $2
		          AND NOT EXISTS (
		              SELECT 1
		              FROM unnest(signal.evidence_ids) evidence_id
		              LEFT JOIN event_sources linked_source ON linked_source.id = evidence_id
		              LEFT JOIN raw_documents linked_document
		                ON linked_document.id = linked_source.raw_document_id
		              WHERE linked_source.id IS NULL
		                 OR GREATEST(
		                     COALESCE(linked_document.published_at, linked_document.collected_at),
		                     linked_document.collected_at
		                 ) > $2
		          )
		    ), '[]'::jsonb)
		)
		FROM events event
		WHERE event.id = $1::uuid
	`, eventID, analysisAsOf).Scan(&payload)
	if err != nil {
		return researchanalysiscontext.EventSemanticBundle{}, err
	}
	var bundle researchanalysiscontext.EventSemanticBundle
	if err := strictDecodeResearchContext(payload, &bundle); err != nil {
		return researchanalysiscontext.EventSemanticBundle{}, err
	}
	return bundle, nil
}

func (s *ResearchAnalysisContextStore) dictionaries(
	ctx context.Context,
	analysisAsOf time.Time,
) (researchanalysiscontext.Dictionaries, []byte, error) {
	var payload []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT jsonb_build_object(
		    'entities', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'entity_id', entity.id,
		            'entity_type', entity.entity_type,
		            'name', entity.name,
		            'canonical_name', entity.canonical_name,
		            'aliases', entity.aliases,
		            'status', entity.status
		        ) ORDER BY entity.entity_type, entity.canonical_name, entity.id)
		        FROM entity_nodes entity
	        WHERE entity.status = 'active'
	          AND entity.created_at <= $1
	          AND entity.updated_at <= $1
		    ), '[]'::jsonb),
		    'relation_definitions', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'relation_type', definition.relation_type,
		            'direction', 'directed'
		        ) ORDER BY definition.relation_type)
		        FROM (
		            SELECT relation_type
		            FROM entity_edges
	            WHERE status = 'active'
	              AND created_at <= $1
	              AND updated_at <= $1
	            UNION
	            SELECT relation_type
	            FROM industry_chain_graph_edges
	            WHERE status = 'active' AND review_status = 'approved'
	              AND created_at <= $1
	              AND updated_at <= $1
		        ) definition
		    ), '[]'::jsonb),
		    'entity_relations', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'entity_relation_id', relation.id,
		            'from_entity_id', relation.from_entity_id,
		            'to_entity_id', relation.to_entity_id,
		            'relation_type', relation.relation_type,
		            'status', relation.status
		        ) ORDER BY relation.relation_type, relation.from_entity_id, relation.to_entity_id, relation.id)
		        FROM entity_edges relation
	        WHERE relation.status = 'active'
	          AND relation.created_at <= $1
	          AND relation.updated_at <= $1
		    ), '[]'::jsonb),
		    'industry_chains', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'industry_chain_entity_id', definition.entity_id,
		            'scope', definition.scope,
		            'target_output', definition.target_output,
		            'end_use', definition.end_use,
		            'geography', definition.geography,
		            'as_of_date', definition.as_of_date,
		            'review_status', definition.review_status
		        ) ORDER BY definition.entity_id)
		        FROM industry_chain_definitions definition
	        WHERE definition.review_status = 'approved'
	          AND definition.created_at <= $1
	          AND definition.updated_at <= $1
		    ), '[]'::jsonb),
		    'industry_chain_memberships', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'industry_chain_entity_id', membership.industry_chain_entity_id,
		            'chain_node_entity_id', membership.chain_node_entity_id,
		            'position', membership.position,
		            'contextual_stage', membership.contextual_stage,
		            'review_status', membership.review_status,
		            'status', membership.status
		        ) ORDER BY membership.industry_chain_entity_id, membership.position, membership.chain_node_entity_id)
		        FROM industry_chain_node_memberships membership
	        WHERE membership.review_status = 'approved' AND membership.status = 'active'
	          AND membership.created_at <= $1
	          AND membership.updated_at <= $1
		    ), '[]'::jsonb),
		    'industry_chain_graph_edges', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'industry_chain_graph_edge_id', edge.id,
		            'industry_chain_entity_id', edge.industry_chain_entity_id,
		            'from_chain_node_entity_id', edge.from_chain_node_entity_id,
		            'to_chain_node_entity_id', edge.to_chain_node_entity_id,
		            'relation_type', edge.relation_type,
		            'mechanism', edge.mechanism,
		            'condition_note', edge.condition_note,
		            'segment_kind', edge.segment_kind,
		            'omitted_step_note', edge.omitted_step_note,
		            'review_status', edge.review_status,
		            'status', edge.status
		        ) ORDER BY edge.industry_chain_entity_id, edge.from_chain_node_entity_id, edge.to_chain_node_entity_id, edge.id)
		        FROM industry_chain_graph_edges edge
	        WHERE edge.review_status = 'approved' AND edge.status = 'active'
	          AND edge.created_at <= $1
	          AND edge.updated_at <= $1
		    ), '[]'::jsonb),
		    'entity_type_definitions', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'type_key', definition.type_key,
		            'version', definition.version,
		            'signal_subject_allowed', definition.signal_subject_allowed,
		            'direct_target_mode', definition.direct_target_mode,
		            'status', definition.status
		        ) ORDER BY definition.type_key, definition.version)
		        FROM entity_type_definitions definition
	        WHERE definition.status = 'active'
	          AND definition.created_at <= $1
		    ), '[]'::jsonb),
		    'variable_definitions', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'key', definition.variable_key,
		            'version', definition.version,
		            'name_zh', definition.name_zh,
		            'name_en', definition.name_en,
		            'domain', definition.domain,
		            'business_definition', definition.business_definition,
		            'value_type', definition.value_type,
		            'allowed_directions', definition.allowed_directions,
		            'canonical_unit', definition.canonical_unit,
		            'status', definition.status,
		            'applicable_entity_types', COALESCE((
		                SELECT jsonb_agg(applicable.entity_type ORDER BY applicable.entity_type)
		                FROM variable_definition_entity_types applicable
		                WHERE applicable.variable_key = definition.variable_key
	                  AND applicable.variable_version = definition.version
	                  AND applicable.created_at <= $1
	            ), '[]'::jsonb)
		        ) ORDER BY definition.variable_key, definition.version)
		        FROM variable_definitions definition
	        WHERE definition.status = 'active'
	          AND definition.created_at <= $1
		    ), '[]'::jsonb),
		    'direct_transmission_rules', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'rule_key', rule.rule_key,
		            'version', rule.version,
		            'source_entity_type', rule.source_entity_type,
		            'source_variable_key', rule.source_variable_key,
		            'source_variable_version', rule.source_variable_version,
		            'source_direction', rule.source_direction,
		            'relation_type', rule.relation_type,
		            'target_entity_type', rule.target_entity_type,
		            'affected_variable_key', rule.affected_variable_key,
		            'affected_variable_version', rule.affected_variable_version,
		            'affected_direction', rule.affected_direction,
		            'condition_summary', rule.condition_summary,
		            'mechanism_template', rule.mechanism_template,
		            'status', rule.status
		        ) ORDER BY rule.rule_key, rule.version)
		        FROM direct_transmission_rules rule
	        WHERE rule.status = 'approved'
	          AND rule.created_at <= $1
	          AND COALESCE(rule.reviewed_at, rule.created_at) <= $1
		    ), '[]'::jsonb),
		    'acceptance_policies', COALESCE((
		        SELECT jsonb_agg(jsonb_build_object(
		            'policy_key', policy.policy_key,
		            'version', policy.version,
		            'retry_budget', policy.retry_budget,
		            'status', policy.status,
		            'policy', policy.policy
		        ) ORDER BY policy.policy_key, policy.version)
		        FROM event_semantic_acceptance_policies policy
	        WHERE policy.status = 'active'
	          AND policy.created_at <= $1
		    ), '[]'::jsonb)
		)
	`, analysisAsOf).Scan(&payload)
	if err != nil {
		return researchanalysiscontext.Dictionaries{}, nil, err
	}
	var dictionaries researchanalysiscontext.Dictionaries
	if err := strictDecodeResearchContext(payload, &dictionaries); err != nil {
		return researchanalysiscontext.Dictionaries{}, nil, err
	}
	return dictionaries, payload, nil
}

func strictDecodeResearchContext(payload []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode typed Research Analysis Context: %w", err)
	}
	return nil
}

func nullUUID(value string) any {
	if value == "" {
		return nil
	}
	return value
}
