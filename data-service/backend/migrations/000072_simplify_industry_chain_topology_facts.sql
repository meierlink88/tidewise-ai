-- +goose Up
DROP TRIGGER trg_protect_active_industry_chain_membership ON industry_chain_node_memberships;
DROP FUNCTION protect_active_industry_chain_membership();

DROP TRIGGER trg_reject_industry_chain_graph_cycle ON industry_chain_graph_edges;
DROP FUNCTION reject_industry_chain_graph_cycle();

DROP INDEX idx_industry_chain_node_memberships_chain_status_position;
DROP INDEX idx_industry_chain_node_memberships_node_status;
DROP INDEX idx_industry_chain_graph_chain_status;
DROP INDEX idx_industry_chain_graph_to_node_status;

ALTER TABLE industry_chain_node_memberships
    DROP CONSTRAINT chk_industry_chain_membership_evidence_ids,
    DROP CONSTRAINT chk_industry_chain_membership_inclusion_nonblank,
    DROP CONSTRAINT chk_industry_chain_membership_source_locator,
    DROP CONSTRAINT chk_industry_chain_membership_source_name_nonblank,
    DROP CONSTRAINT chk_industry_chain_node_membership_review_status,
    DROP CONSTRAINT chk_industry_chain_node_membership_status,
    DROP COLUMN review_status,
    DROP COLUMN status,
    DROP COLUMN inclusion_reason,
    DROP COLUMN evidence_ids,
    DROP COLUMN source_name,
    DROP COLUMN source_url,
    DROP COLUMN verified_at,
    ADD CONSTRAINT chk_industry_chain_node_membership_timestamp_order
        CHECK (updated_at >= created_at);

ALTER TABLE industry_chain_graph_edges
    DROP CONSTRAINT chk_industry_chain_graph_condition_nonblank,
    DROP CONSTRAINT chk_industry_chain_graph_evidence_ids,
    DROP CONSTRAINT chk_industry_chain_graph_mechanism_nonblank,
    DROP CONSTRAINT chk_industry_chain_graph_omitted_step,
    DROP CONSTRAINT chk_industry_chain_graph_review_status,
    DROP CONSTRAINT chk_industry_chain_graph_segment_kind,
    DROP CONSTRAINT chk_industry_chain_graph_source_locator,
    DROP CONSTRAINT chk_industry_chain_graph_source_name_nonblank,
    DROP CONSTRAINT chk_industry_chain_graph_status,
    DROP COLUMN mechanism,
    DROP COLUMN condition_note,
    DROP COLUMN segment_kind,
    DROP COLUMN omitted_step_note,
    DROP COLUMN review_status,
    DROP COLUMN status,
    DROP COLUMN evidence_ids,
    DROP COLUMN source_name,
    DROP COLUMN source_url,
    DROP COLUMN verified_at,
    ADD CONSTRAINT chk_industry_chain_graph_timestamp_order
        CHECK (updated_at >= created_at);

CREATE INDEX idx_industry_chain_node_memberships_chain_position
    ON industry_chain_node_memberships (industry_chain_id, position, chain_node_id);
CREATE INDEX idx_industry_chain_node_memberships_node
    ON industry_chain_node_memberships (chain_node_id);
CREATE INDEX idx_industry_chain_graph_chain_from_node
    ON industry_chain_graph_edges (industry_chain_id, from_chain_node_id);
CREATE INDEX idx_industry_chain_graph_to_node
    ON industry_chain_graph_edges (to_chain_node_id);

-- +goose StatementBegin
CREATE FUNCTION reject_industry_chain_graph_cycle()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM 1
    FROM industry_chain_node_memberships membership
    WHERE membership.industry_chain_id = NEW.industry_chain_id
      AND membership.chain_node_id IN (NEW.from_chain_node_id, NEW.to_chain_node_id)
    ORDER BY membership.chain_node_id
    FOR SHARE;

    PERFORM pg_advisory_xact_lock(hashtext('industry_chain_topology:' || NEW.industry_chain_id));

    IF EXISTS (
        WITH RECURSIVE reachable(node_id) AS (
            SELECT NEW.to_chain_node_id
            UNION
            SELECT edge.to_chain_node_id
            FROM industry_chain_graph_edges edge
            JOIN reachable current_path ON edge.from_chain_node_id = current_path.node_id
            WHERE edge.industry_chain_id = NEW.industry_chain_id
              AND edge.id <> NEW.id
        )
        SELECT 1 FROM reachable WHERE node_id = NEW.from_chain_node_id
    ) THEN
        RAISE EXCEPTION 'industry chain topology must remain acyclic';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_reject_industry_chain_graph_cycle
BEFORE INSERT OR UPDATE OF industry_chain_id, from_chain_node_id, to_chain_node_id
ON industry_chain_graph_edges
FOR EACH ROW EXECUTE FUNCTION reject_industry_chain_graph_cycle();

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000072 is forward-only; restore the pre-migration PostgreSQL snapshot';
END;
$$;
-- +goose StatementEnd
