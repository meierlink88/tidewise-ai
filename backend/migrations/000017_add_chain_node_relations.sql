-- +goose Up
CREATE TABLE chain_node_relations (
    id UUID PRIMARY KEY,
    from_chain_node_entity_id UUID NOT NULL REFERENCES chain_node_profiles(entity_id) ON DELETE RESTRICT,
    to_chain_node_entity_id UUID NOT NULL REFERENCES chain_node_profiles(entity_id) ON DELETE RESTRICT,
    relation_type TEXT NOT NULL CHECK (relation_type IN ('is_subcategory_of', 'is_component_of', 'input_to', 'depends_on')),
    mechanism TEXT NOT NULL CHECK (btrim(mechanism) <> ''),
    condition_note TEXT NULL CHECK (condition_note IS NULL OR btrim(condition_note) <> ''),
    evidence_note TEXT NOT NULL CHECK (btrim(evidence_note) <> ''),
    provenance TEXT NOT NULL CHECK (btrim(provenance) <> ''),
    verified_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (from_chain_node_entity_id <> to_chain_node_entity_id),
    UNIQUE (from_chain_node_entity_id, to_chain_node_entity_id, relation_type)
);
CREATE INDEX chain_node_relations_to_type_idx ON chain_node_relations (to_chain_node_entity_id, relation_type);
CREATE UNIQUE INDEX chain_node_relations_input_dependency_mechanism_uidx
    ON chain_node_relations (from_chain_node_entity_id, to_chain_node_entity_id, lower(btrim(mechanism)))
    WHERE relation_type IN ('input_to', 'depends_on');

CREATE TABLE chain_node_physical_constraints (
    id UUID PRIMARY KEY,
    chain_node_entity_id UUID NULL REFERENCES chain_node_profiles(entity_id) ON DELETE RESTRICT,
    chain_node_relation_id UUID NULL REFERENCES chain_node_relations(id) ON DELETE RESTRICT,
    constraint_type TEXT NOT NULL CHECK (constraint_type IN ('power_capacity','thermal_dissipation','production_capacity','process_yield','material_purity','equipment_capacity','infrastructure_access','physical_expansion_cycle','resource_availability','process_cycle_time')),
    description TEXT NOT NULL CHECK (btrim(description) <> ''),
    condition_note TEXT NULL CHECK (condition_note IS NULL OR btrim(condition_note) <> ''),
    evidence_note TEXT NOT NULL CHECK (btrim(evidence_note) <> ''),
    provenance TEXT NOT NULL CHECK (btrim(provenance) <> ''),
    verified_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((chain_node_entity_id IS NOT NULL)::int + (chain_node_relation_id IS NOT NULL)::int = 1)
);

-- +goose Down
DROP TABLE chain_node_physical_constraints;
DROP TABLE chain_node_relations;
