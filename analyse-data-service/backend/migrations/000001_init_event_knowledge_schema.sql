-- +goose Up
CREATE TABLE entity_nodes (
    id UUID PRIMARY KEY,
    entity_type VARCHAR(64) NOT NULL,
    layer_code VARCHAR(64) NOT NULL,
    name TEXT NOT NULL,
    canonical_name TEXT NOT NULL,
    aliases TEXT[] NOT NULL DEFAULT '{}',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_entity_nodes_status CHECK (status IN ('active', 'inactive', 'merged'))
);

CREATE TABLE entity_edges (
    id UUID PRIMARY KEY,
    from_entity_id UUID NOT NULL REFERENCES entity_nodes(id),
    to_entity_id UUID NOT NULL REFERENCES entity_nodes(id),
    relation_type VARCHAR(64) NOT NULL,
    evidence_note TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_entity_edges_status CHECK (status IN ('active', 'inactive'))
);

CREATE TABLE economy_profiles (
    entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id),
    country_code VARCHAR(16) NOT NULL,
    currency_code VARCHAR(16) NOT NULL,
    region TEXT NOT NULL DEFAULT ''
);

CREATE TABLE policy_body_profiles (
    entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id),
    body_type VARCHAR(64) NOT NULL,
    jurisdiction TEXT NOT NULL,
    policy_domain TEXT NOT NULL DEFAULT ''
);

CREATE TABLE market_profiles (
    entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id),
    market_type VARCHAR(64) NOT NULL,
    economy_entity_id UUID REFERENCES entity_nodes(id),
    currency_code VARCHAR(16) NOT NULL,
    timezone TEXT NOT NULL
);

CREATE TABLE index_profiles (
    entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id),
    index_code VARCHAR(64) NOT NULL,
    index_type VARCHAR(64) NOT NULL,
    market_entity_id UUID REFERENCES entity_nodes(id),
    provider VARCHAR(64) NOT NULL DEFAULT '',
    currency_code VARCHAR(16) NOT NULL,
    list_date DATE
);

CREATE TABLE sector_profiles (
    entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id),
    sector_system VARCHAR(64) NOT NULL,
    sector_code VARCHAR(64) NOT NULL,
    sector_type VARCHAR(64) NOT NULL,
    exchange_scope VARCHAR(64) NOT NULL DEFAULT '',
    constituent_count INTEGER NOT NULL DEFAULT 0,
    list_date DATE,
    parent_sector_entity_id UUID REFERENCES entity_nodes(id)
);

CREATE TABLE chain_node_profiles (
    entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id),
    chain_position VARCHAR(64) NOT NULL
);

CREATE TABLE company_profiles (
    entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id),
    registration_economy_entity_id UUID REFERENCES entity_nodes(id),
    area TEXT NOT NULL DEFAULT '',
    industry_name TEXT NOT NULL DEFAULT '',
    controller_name TEXT NOT NULL DEFAULT '',
    controller_type VARCHAR(64) NOT NULL DEFAULT ''
);

CREATE TABLE security_profiles (
    entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id),
    ticker VARCHAR(64) NOT NULL,
    symbol VARCHAR(64) NOT NULL,
    exchange VARCHAR(64) NOT NULL,
    market_board VARCHAR(64) NOT NULL DEFAULT '',
    security_type VARCHAR(64) NOT NULL,
    issuer_company_entity_id UUID REFERENCES entity_nodes(id),
    list_date DATE,
    delist_date DATE,
    list_status VARCHAR(32) NOT NULL DEFAULT 'listed',
    currency_code VARCHAR(16) NOT NULL
);

CREATE TABLE instrument_profiles (
    entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id),
    instrument_type VARCHAR(64) NOT NULL,
    underlying_entity_id UUID REFERENCES entity_nodes(id),
    exchange VARCHAR(64) NOT NULL DEFAULT '',
    currency_code VARCHAR(16) NOT NULL
);

CREATE TABLE metric_profiles (
    entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id),
    metric_type VARCHAR(64) NOT NULL,
    unit VARCHAR(64) NOT NULL DEFAULT '',
    frequency VARCHAR(64) NOT NULL DEFAULT ''
);

CREATE TABLE commodity_profiles (
    entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id),
    commodity_type VARCHAR(64) NOT NULL
);

CREATE TABLE person_profiles (
    entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id),
    role_title TEXT NOT NULL DEFAULT '',
    organization_entity_id UUID REFERENCES entity_nodes(id),
    economy_entity_id UUID REFERENCES entity_nodes(id)
);

CREATE TABLE source_catalogs (
    id UUID PRIMARY KEY,
    ingest_channel VARCHAR(64) NOT NULL,
    provider_key VARCHAR(64) NOT NULL,
    connector_key VARCHAR(64) NOT NULL,
    parser_key VARCHAR(64) NOT NULL,
    source_type VARCHAR(64) NOT NULL,
    source_name TEXT NOT NULL,
    source_url TEXT NOT NULL DEFAULT '',
    source_level VARCHAR(32) NOT NULL DEFAULT 'secondary',
    topic_hint TEXT NOT NULL DEFAULT '',
    route_template TEXT NOT NULL DEFAULT '',
    code_style VARCHAR(64) NOT NULL DEFAULT '',
    auth_required BOOLEAN NOT NULL DEFAULT false,
    auth_type VARCHAR(64) NOT NULL DEFAULT 'none',
    credential_ref TEXT NOT NULL DEFAULT '',
    rate_limit_policy JSONB NOT NULL DEFAULT '{}'::jsonb,
    usage_policy TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_source_catalogs_status CHECK (status IN ('active', 'inactive', 'disabled'))
);

CREATE TABLE raw_documents (
    id UUID PRIMARY KEY,
    source_id UUID NOT NULL REFERENCES source_catalogs(id),
    ingest_channel VARCHAR(64) NOT NULL,
    source_type VARCHAR(64) NOT NULL,
    source_name TEXT NOT NULL,
    source_url TEXT NOT NULL DEFAULT '',
    source_external_id TEXT,
    title TEXT NOT NULL,
    content_text TEXT NOT NULL DEFAULT '',
    raw_object_uri TEXT NOT NULL DEFAULT '',
    raw_mime_type VARCHAR(128) NOT NULL DEFAULT '',
    language VARCHAR(16) NOT NULL DEFAULT '',
    published_at TIMESTAMPTZ,
    collected_at TIMESTAMPTZ NOT NULL,
    content_hash VARCHAR(128) NOT NULL,
    ingest_status VARCHAR(32) NOT NULL DEFAULT 'collected',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_raw_documents_ingest_status CHECK (ingest_status IN ('collected', 'duplicate', 'failed', 'pending_extract'))
);

CREATE TABLE events (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    event_time TIMESTAMPTZ,
    first_seen_at TIMESTAMPTZ NOT NULL,
    knowable_at TIMESTAMPTZ,
    event_status VARCHAR(32) NOT NULL DEFAULT 'candidate',
    fact_status VARCHAR(32) NOT NULL DEFAULT 'unverified',
    dedupe_key TEXT NOT NULL,
    primary_source_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_events_event_status CHECK (event_status IN ('candidate', 'confirmed', 'rejected')),
    CONSTRAINT chk_events_fact_status CHECK (fact_status IN ('unverified', 'verified', 'disputed'))
);

CREATE TABLE event_sources (
    id UUID PRIMARY KEY,
    event_id UUID NOT NULL REFERENCES events(id),
    raw_document_id UUID NOT NULL REFERENCES raw_documents(id),
    source_level VARCHAR(32) NOT NULL,
    evidence_excerpt TEXT NOT NULL DEFAULT '',
    evidence_hash VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE events
    ADD CONSTRAINT fk_events_primary_source
    FOREIGN KEY (primary_source_id) REFERENCES event_sources(id);

CREATE TABLE event_tag_defs (
    id UUID PRIMARY KEY,
    tag_kind VARCHAR(64) NOT NULL,
    code VARCHAR(128) NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tag_kind, code)
);

CREATE TABLE event_tag_maps (
    id UUID PRIMARY KEY,
    event_id UUID NOT NULL REFERENCES events(id),
    tag_id UUID NOT NULL REFERENCES event_tag_defs(id),
    assign_source VARCHAR(64) NOT NULL,
    review_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (event_id, tag_id),
    CONSTRAINT chk_event_tag_maps_review_status CHECK (review_status IN ('pending', 'approved', 'rejected'))
);

CREATE TABLE event_entity_links (
    id UUID PRIMARY KEY,
    event_id UUID NOT NULL REFERENCES events(id),
    entity_id UUID NOT NULL REFERENCES entity_nodes(id),
    entity_role VARCHAR(64) NOT NULL,
    assign_source VARCHAR(64) NOT NULL,
    review_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    evidence_note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (event_id, entity_id, entity_role),
    CONSTRAINT chk_event_entity_links_review_status CHECK (review_status IN ('pending', 'approved', 'rejected'))
);

CREATE INDEX idx_entity_edges_from_entity_id ON entity_edges (from_entity_id);
CREATE INDEX idx_entity_edges_to_entity_id ON entity_edges (to_entity_id);
CREATE INDEX idx_entity_edges_relation_type ON entity_edges (relation_type);

CREATE INDEX idx_source_catalogs_status ON source_catalogs (status);
CREATE INDEX idx_source_catalogs_provider_channel ON source_catalogs (provider_key, ingest_channel);
CREATE INDEX idx_source_catalogs_connector_key ON source_catalogs (connector_key);

CREATE UNIQUE INDEX ux_raw_documents_source_external_id
    ON raw_documents (source_id, source_external_id)
    WHERE source_external_id IS NOT NULL AND source_external_id <> '';
CREATE UNIQUE INDEX ux_raw_documents_source_content_hash ON raw_documents (source_id, content_hash);
CREATE INDEX idx_raw_documents_source_id ON raw_documents (source_id);
CREATE INDEX idx_raw_documents_published_at ON raw_documents (published_at);
CREATE INDEX idx_raw_documents_collected_at ON raw_documents (collected_at);
CREATE INDEX idx_raw_documents_ingest_status ON raw_documents (ingest_status);

CREATE UNIQUE INDEX ux_events_dedupe_key ON events (dedupe_key);
CREATE INDEX idx_events_dedupe_key ON events (dedupe_key);
CREATE INDEX idx_events_event_time ON events (event_time);
CREATE INDEX idx_events_first_seen_at ON events (first_seen_at);
CREATE INDEX idx_events_knowable_at ON events (knowable_at);
CREATE INDEX idx_events_status ON events (event_status, fact_status);

CREATE INDEX idx_event_sources_event_id ON event_sources (event_id);
CREATE INDEX idx_event_sources_raw_document_id ON event_sources (raw_document_id);
CREATE INDEX idx_event_sources_evidence_hash ON event_sources (evidence_hash);

CREATE INDEX idx_event_tag_maps_event_id ON event_tag_maps (event_id);
CREATE INDEX idx_event_entity_links_event_id ON event_entity_links (event_id);
CREATE INDEX idx_event_entity_links_entity_id ON event_entity_links (entity_id);

-- +goose Down
SELECT 'initial event knowledge schema rollback requires a reviewed forward migration or restored backup';
