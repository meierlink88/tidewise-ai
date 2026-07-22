-- +goose Up
CREATE TRIGGER trg_chain_node_profile_review_status_entity_type
BEFORE UPDATE OF review_status ON chain_node_profiles
FOR EACH ROW
WHEN (
    NEW.review_status IS NOT NULL
    AND NEW.review_status IS DISTINCT FROM OLD.review_status
)
EXECUTE FUNCTION assert_entity_profile_type('chain_node');

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION 'migration 000028 is forward-only; apply a reviewed forward-fix migration';
END;
$$;
-- +goose StatementEnd
