-- +goose Up
SET LOCAL lock_timeout = '5s';

LOCK TABLE geopolitic_rivalries, macro_economics, geopolitic_rivalry_domain_links, macro_economic_domain_links IN ACCESS EXCLUSIVE MODE;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM geopolitic_rivalries s WHERE NOT EXISTS (
        SELECT 1 FROM geopolitic_rivalry_domain_links l WHERE l.geopolitic_rivalry_id=s.id AND l.geopolitic_domain_id=s.geopolitic_domain_id
    )) THEN
        RAISE EXCEPTION 'Run reviewed storyline-domain-backfill at migration 92 before removing legacy domain columns' USING ERRCODE = '55000';
    END IF;
    IF EXISTS (SELECT 1 FROM macro_economics s WHERE NOT EXISTS (
        SELECT 1 FROM macro_economic_domain_links l WHERE l.macro_economic_id=s.id AND l.macro_economic_domain_id=s.macro_economics_domain_id
    )) THEN
        RAISE EXCEPTION 'Run reviewed storyline-domain-backfill at migration 92 before removing legacy domain columns' USING ERRCODE = '55000';
    END IF;
END;
$$;
-- +goose StatementEnd

ALTER TABLE geopolitic_rivalries DROP COLUMN geopolitic_domain_id;
ALTER TABLE macro_economics DROP COLUMN macro_economics_domain_id;

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION 'storyline domain migration is forward-only; restore a reviewed backup with its matching application' USING ERRCODE = '55000';
END;
$$;
-- +goose StatementEnd
