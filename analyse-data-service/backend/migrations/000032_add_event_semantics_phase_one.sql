-- +goose Up
CREATE TABLE entity_type_definitions (
    type_key VARCHAR(64) NOT NULL,
    version INTEGER NOT NULL,
    signal_subject_allowed BOOLEAN NOT NULL,
    direct_target_mode VARCHAR(16) NOT NULL,
    status VARCHAR(16) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (type_key, version),
    CONSTRAINT chk_entity_type_definition_version CHECK (version > 0),
    CONSTRAINT chk_entity_type_definition_target CHECK (direct_target_mode IN ('allow', 'conditional', 'deny', 'context')),
    CONSTRAINT chk_entity_type_definition_status CHECK (status IN ('active', 'deprecated'))
);

CREATE TABLE product_profiles (
    entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id) ON DELETE RESTRICT,
    product_category TEXT NOT NULL DEFAULT '',
    specification TEXT NOT NULL DEFAULT '',
    review_status VARCHAR(16) NOT NULL DEFAULT 'candidate',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_product_profile_review_status CHECK (review_status IN ('candidate', 'approved'))
);

CREATE TABLE variable_definitions (
    variable_key VARCHAR(128) NOT NULL,
    version INTEGER NOT NULL,
    name_zh TEXT NOT NULL,
    name_en TEXT NOT NULL,
    domain VARCHAR(64) NOT NULL,
    business_definition TEXT NOT NULL,
    value_type VARCHAR(64) NOT NULL,
    allowed_directions TEXT[] NOT NULL,
    canonical_unit VARCHAR(64),
    status VARCHAR(16) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (variable_key, version),
    CONSTRAINT chk_variable_definition_version CHECK (version > 0),
    CONSTRAINT chk_variable_definition_names CHECK (btrim(name_zh) <> '' AND btrim(name_en) <> ''),
    CONSTRAINT chk_variable_definition_directions CHECK (
        cardinality(allowed_directions) > 0
        AND allowed_directions <@ ARRAY['increase', 'decrease', 'unchanged', 'mixed', 'uncertain']::text[]
        AND array_position(allowed_directions, NULL::text) IS NULL
    ),
    CONSTRAINT chk_variable_definition_status CHECK (status IN ('active', 'deprecated'))
);

CREATE TABLE variable_definition_entity_types (
    variable_key VARCHAR(128) NOT NULL,
    variable_version INTEGER NOT NULL,
    entity_type VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (variable_key, variable_version, entity_type),
    FOREIGN KEY (variable_key, variable_version)
        REFERENCES variable_definitions(variable_key, version) ON DELETE RESTRICT
);

CREATE TABLE direct_transmission_rules (
    rule_key VARCHAR(128) NOT NULL,
    version INTEGER NOT NULL,
    source_entity_type VARCHAR(64) NOT NULL,
    source_variable_key VARCHAR(128) NOT NULL,
    source_variable_version INTEGER NOT NULL,
    source_direction VARCHAR(32) NOT NULL,
    relation_type VARCHAR(64) NOT NULL,
    target_entity_type VARCHAR(64) NOT NULL,
    affected_variable_key VARCHAR(128) NOT NULL,
    affected_variable_version INTEGER NOT NULL,
    affected_direction VARCHAR(32) NOT NULL,
    condition_summary TEXT NOT NULL DEFAULT '',
    mechanism_template TEXT NOT NULL,
    status VARCHAR(16) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at TIMESTAMPTZ,
    PRIMARY KEY (rule_key, version),
    FOREIGN KEY (source_variable_key, source_variable_version)
        REFERENCES variable_definitions(variable_key, version) ON DELETE RESTRICT,
    FOREIGN KEY (affected_variable_key, affected_variable_version)
        REFERENCES variable_definitions(variable_key, version) ON DELETE RESTRICT,
    CONSTRAINT chk_direct_transmission_rule_version CHECK (version > 0),
    CONSTRAINT chk_direct_transmission_rule_status CHECK (status IN ('draft', 'approved', 'deprecated')),
    CONSTRAINT chk_direct_transmission_rule_mechanism CHECK (btrim(mechanism_template) <> '')
);

CREATE TABLE event_semantic_acceptance_policies (
    policy_key VARCHAR(128) NOT NULL,
    version INTEGER NOT NULL,
    retry_budget INTEGER NOT NULL,
    status VARCHAR(16) NOT NULL,
    policy JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (policy_key, version),
    CONSTRAINT chk_event_semantic_policy_version CHECK (version > 0),
    CONSTRAINT chk_event_semantic_policy_retry CHECK (retry_budget BETWEEN 0 AND 3),
    CONSTRAINT chk_event_semantic_policy_status CHECK (status IN ('active', 'deprecated')),
    CONSTRAINT chk_event_semantic_policy_json CHECK (jsonb_typeof(policy) = 'object')
);

CREATE TABLE event_semantic_context_leases (
    id UUID PRIMARY KEY,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    supersedes_submission_id UUID,
    agent_execution_id TEXT NOT NULL UNIQUE,
    worker_id TEXT NOT NULL,
    status VARCHAR(16) NOT NULL,
    lease_expires_at TIMESTAMPTZ NOT NULL,
    context_snapshot JSONB NOT NULL,
    leased_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    consumed_at TIMESTAMPTZ,
    CONSTRAINT chk_event_semantic_context_lease_identity CHECK (
        btrim(agent_execution_id) <> '' AND btrim(worker_id) <> ''
    ),
    CONSTRAINT chk_event_semantic_context_lease_snapshot CHECK (jsonb_typeof(context_snapshot) = 'object'),
    CONSTRAINT chk_event_semantic_context_lease_status CHECK (status IN ('active', 'consumed', 'expired'))
);

CREATE UNIQUE INDEX ux_event_semantic_context_lease_active_event
    ON event_semantic_context_leases(event_id)
    WHERE status = 'active';
CREATE INDEX idx_event_semantic_context_lease_expiry
    ON event_semantic_context_leases(status, lease_expires_at);

CREATE TABLE event_semantic_submissions (
    id UUID PRIMARY KEY,
    context_lease_id UUID NOT NULL REFERENCES event_semantic_context_leases(id) ON DELETE RESTRICT,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    agent_execution_id TEXT NOT NULL,
    agent_key VARCHAR(128) NOT NULL,
    agent_version VARCHAR(128) NOT NULL,
    supersedes_submission_id UUID REFERENCES event_semantic_submissions(id) ON DELETE RESTRICT,
    generator_prompt_hash CHAR(64) NOT NULL,
    generator_model TEXT NOT NULL,
    reviewer_prompt_hash CHAR(64) NOT NULL,
    reviewer_model TEXT NOT NULL,
    adjudicator_prompt_hash CHAR(64),
    adjudicator_model TEXT,
    ontology_version VARCHAR(128) NOT NULL,
    acceptance_policy_key VARCHAR(128) NOT NULL,
    acceptance_policy_version INTEGER NOT NULL,
    canonical_payload_hash CHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL,
    candidate_counts JSONB NOT NULL DEFAULT '{}'::jsonb,
    decision_summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finalized_at TIMESTAMPTZ,
    UNIQUE (agent_execution_id),
    UNIQUE (context_lease_id),
    FOREIGN KEY (acceptance_policy_key, acceptance_policy_version)
        REFERENCES event_semantic_acceptance_policies(policy_key, version) ON DELETE RESTRICT,
    CONSTRAINT chk_event_semantic_submission_hashes CHECK (
        generator_prompt_hash ~ '^[0-9a-f]{64}$'
        AND reviewer_prompt_hash ~ '^[0-9a-f]{64}$'
        AND canonical_payload_hash ~ '^[0-9a-f]{64}$'
        AND (adjudicator_prompt_hash IS NULL OR adjudicator_prompt_hash ~ '^[0-9a-f]{64}$')
    ),
    CONSTRAINT chk_event_semantic_submission_status CHECK (
        status IN ('pending_review', 'needs_reanalysis', 'quarantined', 'accepted', 'rejected', 'superseded')
    )
);

CREATE INDEX idx_event_semantic_submissions_event_created
    ON event_semantic_submissions(event_id, created_at DESC);

ALTER TABLE event_semantic_context_leases
    ADD CONSTRAINT fk_event_semantic_context_lease_supersedes_submission
    FOREIGN KEY (supersedes_submission_id)
    REFERENCES event_semantic_submissions(id) ON DELETE RESTRICT;

CREATE TABLE event_semantic_candidate_snapshots (
    id UUID PRIMARY KEY,
    semantic_submission_id UUID NOT NULL REFERENCES event_semantic_submissions(id) ON DELETE RESTRICT,
    payload JSONB NOT NULL,
    canonical_payload_hash CHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (semantic_submission_id),
    CONSTRAINT chk_event_semantic_candidate_snapshot_payload CHECK (jsonb_typeof(payload) = 'object'),
    CONSTRAINT chk_event_semantic_candidate_snapshot_hash CHECK (canonical_payload_hash ~ '^[0-9a-f]{64}$')
);

CREATE TABLE event_semantic_review_snapshots (
    id UUID PRIMARY KEY,
    semantic_submission_id UUID NOT NULL REFERENCES event_semantic_submissions(id) ON DELETE RESTRICT,
    reviewer_execution_key TEXT NOT NULL,
    payload JSONB NOT NULL,
    canonical_payload_hash CHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (semantic_submission_id, reviewer_execution_key),
    CONSTRAINT chk_event_semantic_review_snapshot_payload CHECK (jsonb_typeof(payload) = 'object'),
    CONSTRAINT chk_event_semantic_review_snapshot_hash CHECK (canonical_payload_hash ~ '^[0-9a-f]{64}$')
);

ALTER TABLE event_entity_links
    DROP CONSTRAINT event_entity_links_event_id_entity_id_entity_role_key,
    DROP CONSTRAINT chk_event_entity_links_review_status,
    ADD COLUMN semantic_submission_id UUID REFERENCES event_semantic_submissions(id) ON DELETE RESTRICT,
    ADD COLUMN candidate_key VARCHAR(128),
    ADD COLUMN resolved_mention TEXT,
    ADD COLUMN resolution_method VARCHAR(64),
    ADD COLUMN resolution_confidence NUMERIC(6, 5),
    ADD COLUMN evidence_ids UUID[],
    ADD COLUMN provenance VARCHAR(16) NOT NULL DEFAULT 'legacy',
    ADD COLUMN reason_code VARCHAR(128);

UPDATE event_entity_links
SET review_status = CASE review_status
    WHEN 'approved' THEN 'accepted'
    WHEN 'pending' THEN 'pending_review'
    ELSE review_status
END;

ALTER TABLE event_entity_links
    ADD CONSTRAINT chk_event_entity_links_review_status CHECK (
        review_status IN ('pending_review', 'needs_reanalysis', 'quarantined', 'accepted', 'rejected', 'superseded')
    ),
    ADD CONSTRAINT chk_event_entity_links_provenance CHECK (provenance IN ('legacy', 'semantic')),
    ADD CONSTRAINT chk_event_entity_links_semantic_lineage CHECK (
        provenance = 'legacy'
        OR (
            semantic_submission_id IS NOT NULL
            AND candidate_key IS NOT NULL
            AND btrim(candidate_key) <> ''
            AND evidence_ids IS NOT NULL
            AND cardinality(evidence_ids) > 0
        )
    ),
    ADD CONSTRAINT chk_event_entity_links_confidence CHECK (
        resolution_confidence IS NULL OR resolution_confidence BETWEEN 0 AND 1
    );

CREATE UNIQUE INDEX ux_event_entity_link_run_candidate
    ON event_entity_links(semantic_submission_id, candidate_key)
    WHERE semantic_submission_id IS NOT NULL;
CREATE UNIQUE INDEX ux_event_entity_link_active_accepted
    ON event_entity_links(event_id, entity_id, entity_role)
    WHERE review_status = 'accepted';

CREATE TABLE variable_signals (
    id UUID PRIMARY KEY,
    semantic_submission_id UUID NOT NULL REFERENCES event_semantic_submissions(id) ON DELETE RESTRICT,
    candidate_key VARCHAR(128) NOT NULL,
    source_event_id UUID NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    subject_event_entity_link_id UUID NOT NULL REFERENCES event_entity_links(id) ON DELETE RESTRICT,
    variable_key VARCHAR(128) NOT NULL,
    variable_version INTEGER NOT NULL,
    direction VARCHAR(32) NOT NULL,
    assertion_modality VARCHAR(32) NOT NULL,
    evidence_ids UUID[] NOT NULL,
    statement_at TIMESTAMPTZ,
    valid_from TIMESTAMPTZ,
    valid_until TIMESTAMPTZ,
    forecast_period_start TIMESTAMPTZ,
    forecast_period_end TIMESTAMPTZ,
    extraction_confidence NUMERIC(6, 5),
    review_status VARCHAR(32) NOT NULL,
    reason_code VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (semantic_submission_id, candidate_key),
    FOREIGN KEY (variable_key, variable_version)
        REFERENCES variable_definitions(variable_key, version) ON DELETE RESTRICT,
    CONSTRAINT chk_variable_signal_direction CHECK (
        direction IN ('increase', 'decrease', 'unchanged', 'mixed', 'uncertain')
    ),
    CONSTRAINT chk_variable_signal_modality CHECK (
        assertion_modality IN ('actual', 'stated_intent', 'source_forecast')
    ),
    CONSTRAINT chk_variable_signal_status CHECK (
        review_status IN ('pending_review', 'needs_reanalysis', 'quarantined', 'accepted', 'rejected', 'superseded')
    ),
    CONSTRAINT chk_variable_signal_evidence CHECK (
        cardinality(evidence_ids) > 0 AND array_position(evidence_ids, NULL::uuid) IS NULL
    ),
    CONSTRAINT chk_variable_signal_time_range CHECK (valid_until IS NULL OR valid_from IS NULL OR valid_until >= valid_from),
    CONSTRAINT chk_variable_signal_forecast_range CHECK (
        forecast_period_end IS NULL OR forecast_period_start IS NULL OR forecast_period_end >= forecast_period_start
    ),
    CONSTRAINT chk_variable_signal_confidence CHECK (
        extraction_confidence IS NULL OR extraction_confidence BETWEEN 0 AND 1
    )
);

CREATE TABLE variable_signal_measurements (
    id UUID PRIMARY KEY,
    variable_signal_id UUID NOT NULL REFERENCES variable_signals(id) ON DELETE RESTRICT,
    measurement_role VARCHAR(32) NOT NULL,
    value_shape VARCHAR(16) NOT NULL,
    raw_value NUMERIC,
    raw_lower NUMERIC,
    raw_upper NUMERIC,
    raw_unit VARCHAR(64),
    canonical_value NUMERIC,
    canonical_lower NUMERIC,
    canonical_upper NUMERIC,
    canonical_unit VARCHAR(64),
    currency VARCHAR(16),
    scale VARCHAR(32),
    comparison_basis VARCHAR(64),
    comparison_period TEXT,
    raw_text TEXT NOT NULL,
    is_approximate BOOLEAN NOT NULL DEFAULT false,
    evidence_id UUID NOT NULL REFERENCES event_sources(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_variable_signal_measurement_role CHECK (
        measurement_role IN ('absolute_level', 'absolute_change', 'relative_change', 'percentage_point_change')
    ),
    CONSTRAINT chk_variable_signal_measurement_shape CHECK (
        value_shape IN ('exact', 'range', 'lower_bound', 'upper_bound')
    ),
    CONSTRAINT chk_variable_signal_measurement_text CHECK (btrim(raw_text) <> ''),
    CONSTRAINT chk_variable_signal_measurement_values CHECK (
        (
            value_shape = 'exact'
            AND raw_value IS NOT NULL
            AND canonical_value IS NOT NULL
            AND raw_lower IS NULL AND raw_upper IS NULL
            AND canonical_lower IS NULL AND canonical_upper IS NULL
        )
        OR (
            value_shape = 'range'
            AND raw_lower IS NOT NULL AND raw_upper IS NOT NULL
            AND canonical_lower IS NOT NULL AND canonical_upper IS NOT NULL
            AND raw_value IS NULL AND canonical_value IS NULL
        )
        OR (
            value_shape = 'lower_bound'
            AND raw_lower IS NOT NULL
            AND canonical_lower IS NOT NULL
            AND raw_value IS NULL AND raw_upper IS NULL
            AND canonical_value IS NULL AND canonical_upper IS NULL
        )
        OR (
            value_shape = 'upper_bound'
            AND raw_upper IS NOT NULL
            AND canonical_upper IS NOT NULL
            AND raw_value IS NULL AND raw_lower IS NULL
            AND canonical_value IS NULL AND canonical_lower IS NULL
        )
    ),
    CONSTRAINT chk_variable_signal_measurement_change_unit CHECK (
        (measurement_role <> 'relative_change' OR canonical_unit = 'percent')
        AND (measurement_role <> 'percentage_point_change' OR canonical_unit = 'percentage_point')
    ),
    CONSTRAINT chk_variable_signal_measurement_units CHECK (
        raw_unit IS NOT NULL
        AND canonical_unit IS NOT NULL
        AND btrim(raw_unit) <> ''
        AND btrim(canonical_unit) <> ''
        AND (
            lower(btrim(raw_unit)) = lower(btrim(canonical_unit))
            OR (lower(btrim(raw_unit)) IN ('%', 'percent') AND lower(btrim(canonical_unit)) = 'percent')
            OR (
                lower(btrim(raw_unit)) IN ('pp', 'percentage_point', '个百分点')
                AND lower(btrim(canonical_unit)) = 'percentage_point'
            )
        )
    ),
    CONSTRAINT chk_variable_signal_measurement_conversion CHECK (
        (raw_value IS NULL OR canonical_value IS NULL OR raw_value = canonical_value)
        AND (raw_lower IS NULL OR canonical_lower IS NULL OR raw_lower = canonical_lower)
        AND (raw_upper IS NULL OR canonical_upper IS NULL OR raw_upper = canonical_upper)
    ),
    CONSTRAINT chk_variable_signal_measurement_raw_range CHECK (raw_upper IS NULL OR raw_lower IS NULL OR raw_upper >= raw_lower),
    CONSTRAINT chk_variable_signal_measurement_canonical_range CHECK (
        canonical_upper IS NULL OR canonical_lower IS NULL OR canonical_upper >= canonical_lower
    )
);

CREATE TABLE direct_impact_assertions (
    id UUID PRIMARY KEY,
    semantic_submission_id UUID NOT NULL REFERENCES event_semantic_submissions(id) ON DELETE RESTRICT,
    candidate_key VARCHAR(128) NOT NULL,
    source_variable_signal_id UUID NOT NULL REFERENCES variable_signals(id) ON DELETE RESTRICT,
    target_entity_id UUID NOT NULL REFERENCES entity_nodes(id) ON DELETE RESTRICT,
    affected_variable_key VARCHAR(128) NOT NULL,
    affected_variable_version INTEGER NOT NULL,
    affected_direction VARCHAR(32) NOT NULL,
    derivation_type VARCHAR(32) NOT NULL,
    mechanism_summary TEXT NOT NULL DEFAULT '',
    evidence_ids UUID[] NOT NULL,
    entity_relation_id UUID REFERENCES entity_edges(id) ON DELETE RESTRICT,
    rule_key VARCHAR(128),
    rule_version INTEGER,
    assertion_confidence NUMERIC(6, 5),
    review_status VARCHAR(32) NOT NULL,
    reason_code VARCHAR(128),
    effective_from TIMESTAMPTZ,
    effective_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (semantic_submission_id, candidate_key),
    FOREIGN KEY (affected_variable_key, affected_variable_version)
        REFERENCES variable_definitions(variable_key, version) ON DELETE RESTRICT,
    FOREIGN KEY (rule_key, rule_version)
        REFERENCES direct_transmission_rules(rule_key, version) ON DELETE RESTRICT,
    CONSTRAINT chk_direct_impact_direction CHECK (
        affected_direction IN ('increase', 'decrease', 'unchanged', 'mixed', 'uncertain')
    ),
    CONSTRAINT chk_direct_impact_derivation CHECK (derivation_type IN ('event_explicit', 'rule_inferred')),
    CONSTRAINT chk_direct_impact_rule_lineage CHECK (
        (derivation_type = 'event_explicit' AND rule_key IS NULL AND rule_version IS NULL)
        OR
        (derivation_type = 'rule_inferred' AND rule_key IS NOT NULL AND rule_version IS NOT NULL AND entity_relation_id IS NOT NULL)
    ),
    CONSTRAINT chk_direct_impact_status CHECK (
        review_status IN ('pending_review', 'needs_reanalysis', 'quarantined', 'accepted', 'rejected', 'superseded')
    ),
    CONSTRAINT chk_direct_impact_evidence CHECK (
        cardinality(evidence_ids) > 0 AND array_position(evidence_ids, NULL::uuid) IS NULL
    ),
    CONSTRAINT chk_direct_impact_confidence CHECK (
        assertion_confidence IS NULL OR assertion_confidence BETWEEN 0 AND 1
    ),
    CONSTRAINT chk_direct_impact_time_range CHECK (
        effective_to IS NULL OR effective_from IS NULL OR effective_to >= effective_from
    )
);

CREATE UNIQUE INDEX ux_variable_signal_active_accepted
    ON variable_signals(source_event_id, subject_event_entity_link_id, variable_key, variable_version, direction, assertion_modality)
    WHERE review_status = 'accepted';
CREATE UNIQUE INDEX ux_direct_impact_active_accepted
    ON direct_impact_assertions(source_variable_signal_id, target_entity_id, affected_variable_key, affected_variable_version, affected_direction, derivation_type)
    WHERE review_status = 'accepted';

INSERT INTO entity_type_definitions(type_key, version, signal_subject_allowed, direct_target_mode, status) VALUES
    ('commodity', 1, true, 'allow', 'active'),
    ('product', 1, true, 'allow', 'active'),
    ('chain_node', 1, true, 'allow', 'active'),
    ('industry_chain', 1, true, 'deny', 'active'),
    ('industry', 1, true, 'conditional', 'active'),
    ('company', 1, true, 'allow', 'active'),
    ('security', 1, true, 'deny', 'active'),
    ('sector', 1, true, 'deny', 'active'),
    ('concept', 1, true, 'deny', 'active'),
    ('policy_body', 1, false, 'context', 'active'),
    ('person', 1, false, 'context', 'active'),
    ('alliance_org', 1, false, 'context', 'active');

INSERT INTO variable_definitions(
    variable_key, version, name_zh, name_en, domain, business_definition,
    value_type, allowed_directions, canonical_unit, status
) VALUES
    ('market_supply', 1, '市场供给', 'Market Supply', 'supply_demand', '指定市场与期间内可供给的数量或充足程度', 'quantity_or_index', ARRAY['increase','decrease','unchanged','mixed','uncertain'], NULL, 'active'),
    ('market_demand', 1, '市场需求', 'Market Demand', 'supply_demand', '指定市场与期间内需求数量或强弱', 'quantity_or_index', ARRAY['increase','decrease','unchanged','mixed','uncertain'], NULL, 'active'),
    ('market_price', 1, '市场价格', 'Market Price', 'pricing', '明确成交、现货、合同或公开报价', 'monetary_per_unit', ARRAY['increase','decrease','unchanged','mixed','uncertain'], NULL, 'active'),
    ('production_volume', 1, '产量', 'Production Volume', 'operations', '指定期间内实际或声明的生产数量', 'quantity', ARRAY['increase','decrease','unchanged','mixed','uncertain'], NULL, 'active'),
    ('sales_volume', 1, '销量', 'Sales Volume', 'operations', '指定期间内销售或交付数量', 'quantity', ARRAY['increase','decrease','unchanged','mixed','uncertain'], NULL, 'active'),
    ('order_quantity', 1, '订单数量', 'Order Quantity', 'company_operations', '订单中的产品数量', 'quantity', ARRAY['increase','decrease','unchanged','mixed','uncertain'], NULL, 'active'),
    ('order_value', 1, '订单金额', 'Order Value', 'company_operations', '订单或合同金额', 'monetary', ARRAY['increase','decrease','unchanged','mixed','uncertain'], NULL, 'active'),
    ('revenue', 1, '营业收入', 'Revenue', 'company_financials', '指定报告期营业收入', 'monetary', ARRAY['increase','decrease','unchanged','mixed','uncertain'], NULL, 'active'),
    ('net_profit', 1, '净利润', 'Net Profit', 'company_financials', '指定报告期口径明确的净利润', 'monetary', ARRAY['increase','decrease','unchanged','mixed','uncertain'], NULL, 'active'),
    ('gross_margin', 1, '毛利率', 'Gross Margin', 'company_financials', '指定报告期毛利率', 'ratio', ARRAY['increase','decrease','unchanged','mixed','uncertain'], 'percent', 'active'),
    ('policy_support_intensity', 1, '政策支持强度', 'Policy Support Intensity', 'policy', '明确政策措施对规范对象的支持方向', 'ordinal_directional', ARRAY['increase','decrease','unchanged','mixed','uncertain'], NULL, 'active'),
    ('regulatory_restriction_intensity', 1, '监管限制强度', 'Regulatory Restriction Intensity', 'policy', '明确监管措施对规范对象的限制方向', 'ordinal_directional', ARRAY['increase','decrease','unchanged','mixed','uncertain'], NULL, 'active');

INSERT INTO variable_definition_entity_types(variable_key, variable_version, entity_type) VALUES
    ('market_supply',1,'commodity'),('market_supply',1,'product'),('market_supply',1,'chain_node'),('market_supply',1,'industry'),('market_supply',1,'industry_chain'),
    ('market_demand',1,'commodity'),('market_demand',1,'product'),('market_demand',1,'chain_node'),('market_demand',1,'industry'),('market_demand',1,'industry_chain'),
    ('market_price',1,'commodity'),('market_price',1,'product'),
    ('production_volume',1,'commodity'),('production_volume',1,'product'),('production_volume',1,'chain_node'),('production_volume',1,'company'),('production_volume',1,'industry'),
    ('sales_volume',1,'product'),('sales_volume',1,'company'),('sales_volume',1,'industry'),
    ('order_quantity',1,'company'),('order_quantity',1,'product'),
    ('order_value',1,'company'),('order_value',1,'product'),
    ('revenue',1,'company'),('net_profit',1,'company'),('gross_margin',1,'company'),
    ('policy_support_intensity',1,'commodity'),('policy_support_intensity',1,'product'),('policy_support_intensity',1,'industry_chain'),('policy_support_intensity',1,'chain_node'),('policy_support_intensity',1,'industry'),('policy_support_intensity',1,'company'),('policy_support_intensity',1,'sector'),('policy_support_intensity',1,'concept'),
    ('regulatory_restriction_intensity',1,'commodity'),('regulatory_restriction_intensity',1,'product'),('regulatory_restriction_intensity',1,'industry_chain'),('regulatory_restriction_intensity',1,'chain_node'),('regulatory_restriction_intensity',1,'industry'),('regulatory_restriction_intensity',1,'company'),('regulatory_restriction_intensity',1,'security'),('regulatory_restriction_intensity',1,'sector'),('regulatory_restriction_intensity',1,'concept');

INSERT INTO direct_transmission_rules(
    rule_key, version, source_entity_type, source_variable_key, source_variable_version,
    source_direction, relation_type, target_entity_type, affected_variable_key,
    affected_variable_version, affected_direction, condition_summary, mechanism_template,
    status, reviewed_at
) VALUES
    ('production_decrease_reduces_product_supply', 1, 'company', 'production_volume', 1, 'decrease', 'produces', 'product', 'market_supply', 1, 'decrease', 'Company production and Product identity must be explicit', 'Producer output falls, reducing direct Product supply', 'approved', now()),
    ('production_increase_raises_product_supply', 1, 'company', 'production_volume', 1, 'increase', 'produces', 'product', 'market_supply', 1, 'increase', 'Company production and Product identity must be explicit', 'Producer output rises, increasing direct Product supply', 'approved', now()),
    ('chain_node_production_decrease_reduces_product_supply', 1, 'chain_node', 'production_volume', 1, 'decrease', 'produces', 'product', 'market_supply', 1, 'decrease', 'Chain Node production relation must be direct', 'Production at the Chain Node falls, reducing direct Product supply', 'approved', now());

INSERT INTO event_semantic_acceptance_policies(policy_key, version, retry_budget, status, policy)
VALUES (
    'event-semantics.phase-one', 1, 1, 'active',
    '{"requires_independent_reviewer":true,"confidence_is_routing_only":true,"quarantine_after_retry_budget":true}'::jsonb
);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000032 is forward-only; use a reviewed forward migration or restore the pre-migration backup';
END;
$$;
-- +goose StatementEnd
