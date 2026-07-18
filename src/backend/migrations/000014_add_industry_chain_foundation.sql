-- +goose Up
ALTER TABLE chain_node_profiles
    ADD COLUMN IF NOT EXISTS node_category TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS definition TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS unit_of_analysis TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS granularity_note TEXT NOT NULL DEFAULT '';

CREATE TABLE industry_chain_profiles (
    entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id),
    chain_code VARCHAR(96) NOT NULL UNIQUE,
    definition TEXT NOT NULL,
    boundary_note TEXT NOT NULL DEFAULT '',
    scope_type VARCHAR(32) NOT NULL,
    primary_economy_entity_id UUID REFERENCES entity_nodes(id),
    version INTEGER NOT NULL,
    review_status VARCHAR(32) NOT NULL,
    source_name TEXT NOT NULL,
    source_url TEXT NOT NULL,
    verified_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT chk_industry_chain_scope CHECK (scope_type IN ('global', 'economy', 'regional')),
    CONSTRAINT chk_industry_chain_scope_economy CHECK ((scope_type = 'global' AND primary_economy_entity_id IS NULL) OR (scope_type <> 'global' AND primary_economy_entity_id IS NOT NULL)),
    CONSTRAINT chk_industry_chain_version CHECK (version > 0),
    CONSTRAINT chk_industry_chain_review CHECK (review_status IN ('candidate', 'reviewed', 'approved'))
);

CREATE TABLE industry_chain_memberships (
    id UUID PRIMARY KEY,
    industry_chain_entity_id UUID NOT NULL REFERENCES industry_chain_profiles(entity_id),
    chain_node_entity_id UUID NOT NULL REFERENCES chain_node_profiles(entity_id),
    stage_code VARCHAR(32) NOT NULL,
    role_code VARCHAR(32) NOT NULL,
    stage_order INTEGER NOT NULL,
    is_core BOOLEAN NOT NULL DEFAULT FALSE,
    source_name TEXT NOT NULL,
    source_url TEXT NOT NULL,
    verified_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_industry_chain_membership_stage CHECK (stage_code IN ('upstream', 'midstream', 'downstream', 'infrastructure', 'service')),
    CONSTRAINT chk_industry_chain_membership_role CHECK (role_code IN ('resource', 'material', 'equipment', 'component', 'process', 'product', 'service', 'infrastructure')),
    CONSTRAINT chk_industry_chain_membership_order CHECK (stage_order >= 0),
    CONSTRAINT chk_industry_chain_membership_status CHECK (status IN ('active', 'inactive')),
    UNIQUE (industry_chain_entity_id, chain_node_entity_id)
);

CREATE INDEX industry_chain_memberships_chain_status_order_idx
    ON industry_chain_memberships (industry_chain_entity_id, status, stage_order);

CREATE TABLE industry_chain_topology_edges (
    id UUID PRIMARY KEY,
    industry_chain_entity_id UUID NOT NULL REFERENCES industry_chain_profiles(entity_id),
    from_chain_node_entity_id UUID NOT NULL REFERENCES chain_node_profiles(entity_id),
    to_chain_node_entity_id UUID NOT NULL REFERENCES chain_node_profiles(entity_id),
    relation_type VARCHAR(32) NOT NULL,
    evidence_note TEXT NOT NULL DEFAULT '',
    source_name TEXT NOT NULL,
    source_url TEXT NOT NULL,
    verified_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_industry_chain_topology_type CHECK (relation_type IN ('supplies_to', 'depends_on', 'substitutes_for')),
    CONSTRAINT chk_industry_chain_topology_self CHECK (from_chain_node_entity_id <> to_chain_node_entity_id),
    CONSTRAINT chk_industry_chain_topology_status CHECK (status IN ('active', 'inactive')),
    UNIQUE (industry_chain_entity_id, from_chain_node_entity_id, relation_type, to_chain_node_entity_id)
);

CREATE INDEX industry_chain_topology_edges_chain_status_idx
    ON industry_chain_topology_edges (industry_chain_entity_id, status);

CREATE TABLE industry_chain_physical_constraints (
    id UUID PRIMARY KEY,
    industry_chain_entity_id UUID NOT NULL REFERENCES industry_chain_profiles(entity_id),
    chain_node_entity_id UUID REFERENCES chain_node_profiles(entity_id),
    topology_edge_id UUID REFERENCES industry_chain_topology_edges(id),
    constraint_type VARCHAR(48) NOT NULL,
    mechanism TEXT NOT NULL,
    physical_limit_note TEXT NOT NULL DEFAULT '',
    mitigation_path TEXT NOT NULL DEFAULT '',
    source_name TEXT NOT NULL,
    source_url TEXT NOT NULL,
    verified_at TIMESTAMPTZ NOT NULL,
	review_status VARCHAR(32) NOT NULL DEFAULT 'candidate',
	generated_by_ai BOOLEAN NOT NULL DEFAULT FALSE,
	status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_industry_chain_constraint_subject CHECK ((chain_node_entity_id IS NOT NULL)::INTEGER + (topology_edge_id IS NOT NULL)::INTEGER = 1),
    CONSTRAINT chk_industry_chain_constraint_type CHECK (constraint_type IN ('power_capacity', 'thermal_dissipation', 'bandwidth', 'latency', 'production_capacity', 'process_yield', 'material_purity', 'reliability', 'process_cycle_time', 'packaging_density', 'equipment_capacity', 'infrastructure_access', 'physical_expansion_cycle')),
    CONSTRAINT chk_industry_chain_constraint_review CHECK (review_status IN ('candidate', 'reviewed', 'approved')),
    CONSTRAINT chk_industry_chain_constraint_status CHECK (status IN ('active', 'inactive'))
);

CREATE INDEX industry_chain_physical_constraints_chain_idx
    ON industry_chain_physical_constraints (industry_chain_entity_id, status, review_status);
CREATE INDEX industry_chain_physical_constraints_node_idx
    ON industry_chain_physical_constraints (chain_node_entity_id) WHERE chain_node_entity_id IS NOT NULL;
CREATE INDEX industry_chain_physical_constraints_edge_idx
    ON industry_chain_physical_constraints (topology_edge_id) WHERE topology_edge_id IS NOT NULL;

-- +goose Down
SELECT 'industry chain foundation rollback requires a reviewed forward migration or restored backup';
