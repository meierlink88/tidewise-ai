-- +goose Up
CREATE TABLE research_themes (
    id UUID PRIMARY KEY,
    analysis_batch_id TEXT NOT NULL,
    name TEXT NOT NULL,
    one_line_conclusion TEXT NOT NULL,
    impact_level TEXT NOT NULL,
    transmission_path TEXT NOT NULL,
    trading_direction TEXT NOT NULL,
    transmission_stage TEXT NOT NULL,
    next_checkpoint TEXT NOT NULL,
    index_impact_summary TEXT NOT NULL,
    window_start TIMESTAMPTZ,
    window_end TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_research_themes_batch_nonblank CHECK (btrim(analysis_batch_id) <> ''),
    CONSTRAINT chk_research_themes_name_nonblank CHECK (btrim(name) <> ''),
    CONSTRAINT chk_research_themes_conclusion_nonblank CHECK (btrim(one_line_conclusion) <> ''),
    CONSTRAINT chk_research_themes_path_nonblank CHECK (btrim(transmission_path) <> ''),
    CONSTRAINT chk_research_themes_direction_nonblank CHECK (btrim(trading_direction) <> ''),
    CONSTRAINT chk_research_themes_checkpoint_nonblank CHECK (btrim(next_checkpoint) <> ''),
    CONSTRAINT chk_research_themes_index_summary_nonblank CHECK (btrim(index_impact_summary) <> ''),
    CONSTRAINT chk_research_themes_impact_level CHECK (impact_level IN ('high', 'focus', 'watch')),
    CONSTRAINT chk_research_themes_stage CHECK (transmission_stage IN ('upstream', 'midstream', 'downstream', 'infrastructure', 'service')),
    CONSTRAINT chk_research_themes_window CHECK ((window_start IS NULL AND window_end IS NULL) OR (window_start IS NOT NULL AND window_end IS NOT NULL AND window_end >= window_start))
);

CREATE TABLE research_theme_chain_nodes (
    theme_id UUID NOT NULL REFERENCES research_themes(id) ON DELETE CASCADE,
    chain_node_entity_id UUID NOT NULL REFERENCES chain_node_profiles(entity_id),
    relation_role TEXT NOT NULL,
    impact_summary TEXT NOT NULL,
    PRIMARY KEY (theme_id, chain_node_entity_id),
    CONSTRAINT chk_research_theme_chain_role CHECK (relation_role IN ('driver', 'beneficiary', 'constraint', 'exposure')),
    CONSTRAINT chk_research_theme_chain_summary_nonblank CHECK (btrim(impact_summary) <> '')
);

CREATE TABLE research_theme_indices (
    theme_id UUID NOT NULL REFERENCES research_themes(id) ON DELETE CASCADE,
    index_entity_id UUID NOT NULL REFERENCES index_profiles(entity_id),
    impact_direction TEXT NOT NULL,
    impact_summary TEXT NOT NULL,
    PRIMARY KEY (theme_id, index_entity_id),
    CONSTRAINT chk_research_theme_index_direction CHECK (impact_direction IN ('positive', 'negative', 'mixed', 'neutral')),
    CONSTRAINT chk_research_theme_index_summary_nonblank CHECK (btrim(impact_summary) <> '')
);

CREATE TABLE research_theme_events (
    theme_id UUID NOT NULL REFERENCES research_themes(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES events(id),
    evidence_role TEXT NOT NULL,
    supported_claim TEXT NOT NULL,
    PRIMARY KEY (theme_id, event_id),
    CONSTRAINT chk_research_theme_event_role CHECK (evidence_role IN ('driver', 'supporting', 'contradicting', 'context')),
    CONSTRAINT chk_research_theme_event_claim_nonblank CHECK (btrim(supported_claim) <> '')
);

CREATE TABLE research_anchors (
    id UUID PRIMARY KEY,
    analysis_batch_id TEXT NOT NULL,
    anchor_type TEXT NOT NULL,
    name TEXT NOT NULL,
    one_line_conclusion TEXT NOT NULL,
    importance TEXT NOT NULL,
    transmission_path TEXT NOT NULL,
    trading_direction TEXT NOT NULL,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_research_anchors_batch_nonblank CHECK (btrim(analysis_batch_id) <> ''),
    CONSTRAINT chk_research_anchors_type CHECK (anchor_type IN ('policy', 'supply', 'demand', 'technology', 'cost', 'geopolitics', 'market_structure')),
    CONSTRAINT chk_research_anchors_name_nonblank CHECK (btrim(name) <> ''),
    CONSTRAINT chk_research_anchors_conclusion_nonblank CHECK (btrim(one_line_conclusion) <> ''),
    CONSTRAINT chk_research_anchors_importance CHECK (importance IN ('primary', 'secondary', 'contextual')),
    CONSTRAINT chk_research_anchors_path_nonblank CHECK (btrim(transmission_path) <> ''),
    CONSTRAINT chk_research_anchors_direction_nonblank CHECK (btrim(trading_direction) <> '')
);

CREATE TABLE research_anchor_chain_nodes (
    anchor_id UUID NOT NULL REFERENCES research_anchors(id) ON DELETE CASCADE,
    chain_node_entity_id UUID NOT NULL REFERENCES chain_node_profiles(entity_id),
    relation_role TEXT NOT NULL,
    relation_summary TEXT NOT NULL,
    PRIMARY KEY (anchor_id, chain_node_entity_id),
    CONSTRAINT chk_research_anchor_chain_role CHECK (relation_role IN ('driver', 'beneficiary', 'constraint', 'exposure')),
    CONSTRAINT chk_research_anchor_chain_summary_nonblank CHECK (btrim(relation_summary) <> '')
);

CREATE TABLE research_anchor_indices (
    anchor_id UUID NOT NULL REFERENCES research_anchors(id) ON DELETE CASCADE,
    index_entity_id UUID NOT NULL REFERENCES index_profiles(entity_id),
    impact_direction TEXT NOT NULL,
    impact_summary TEXT NOT NULL,
    PRIMARY KEY (anchor_id, index_entity_id),
    CONSTRAINT chk_research_anchor_index_direction CHECK (impact_direction IN ('positive', 'negative', 'mixed', 'neutral')),
    CONSTRAINT chk_research_anchor_index_summary_nonblank CHECK (btrim(impact_summary) <> '')
);

CREATE TABLE research_anchor_events (
    anchor_id UUID NOT NULL REFERENCES research_anchors(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES events(id),
    evidence_role TEXT NOT NULL,
    supported_claim TEXT NOT NULL,
    PRIMARY KEY (anchor_id, event_id),
    CONSTRAINT chk_research_anchor_event_role CHECK (evidence_role IN ('driver', 'supporting', 'contradicting', 'context')),
    CONSTRAINT chk_research_anchor_event_claim_nonblank CHECK (btrim(supported_claim) <> '')
);

CREATE INDEX research_themes_batch_published_idx ON research_themes (analysis_batch_id, published_at DESC, id);
CREATE INDEX research_themes_published_idx ON research_themes (published_at DESC, id);
CREATE INDEX research_anchors_batch_published_idx ON research_anchors (analysis_batch_id, published_at DESC, id);
CREATE INDEX research_anchors_published_idx ON research_anchors (published_at DESC, id);
CREATE INDEX research_theme_chain_nodes_node_idx ON research_theme_chain_nodes (chain_node_entity_id);
CREATE INDEX research_theme_indices_index_idx ON research_theme_indices (index_entity_id);
CREATE INDEX research_theme_events_event_idx ON research_theme_events (event_id);
CREATE INDEX research_anchor_chain_nodes_node_idx ON research_anchor_chain_nodes (chain_node_entity_id);
CREATE INDEX research_anchor_indices_index_idx ON research_anchor_indices (index_entity_id);
CREATE INDEX research_anchor_events_event_idx ON research_anchor_events (event_id);

-- +goose Down
SELECT 'research theme and anchor rollback requires a reviewed forward migration or restored backup';
