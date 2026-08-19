-- +goose Up
CREATE TABLE sources (
    id VARCHAR(39) PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    ownership_type VARCHAR(16) NOT NULL,
    channel_type VARCHAR(16) NOT NULL,
    adapter_key VARCHAR(64) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    endpoint VARCHAR(2048) NOT NULL,
    app_key VARCHAR(512),
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    priority SMALLINT NOT NULL DEFAULT 1,
    timeout_seconds SMALLINT NOT NULL DEFAULT 30,
    max_results SMALLINT NOT NULL DEFAULT 10,
    default_source_level VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_sources_id CHECK (
        id ~ '^SRC[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    ),
    CONSTRAINT chk_sources_code CHECK (code ~ '^[a-z0-9][a-z0-9_-]{0,63}$'),
    CONSTRAINT chk_sources_name CHECK (btrim(name) <> ''),
    CONSTRAINT chk_sources_ownership CHECK (ownership_type IN ('fixed', 'dynamic')),
    CONSTRAINT chk_sources_channel_type CHECK (channel_type IN ('web_search', 'api', 'rss')),
    CONSTRAINT chk_sources_adapter_key CHECK (adapter_key IN (
        'bocha', 'tavily', 'parallel', 'cls', 'eastmoney_fast', 'eastmoney_stock', 'stcn', 'generic_rss'
    )),
    CONSTRAINT chk_sources_dynamic_protocol CHECK (
        ownership_type = 'fixed' OR (channel_type = 'rss' AND adapter_key = 'generic_rss')
    ),
    CONSTRAINT chk_sources_endpoint CHECK (endpoint ~ '^https?://' AND btrim(endpoint) <> ''),
    CONSTRAINT chk_sources_app_key CHECK (app_key IS NULL OR btrim(app_key) <> ''),
    CONSTRAINT chk_sources_config_object CHECK (jsonb_typeof(config) = 'object'),
    CONSTRAINT chk_sources_priority CHECK (priority BETWEEN 1 AND 5),
    CONSTRAINT chk_sources_timeout CHECK (timeout_seconds BETWEEN 1 AND 300),
    CONSTRAINT chk_sources_max_results CHECK (max_results BETWEEN 1 AND 100),
    CONSTRAINT chk_sources_source_level CHECK (
        default_source_level IN ('L1_OFFICIAL', 'L2_WIRE', 'L3_MEDIA', 'L4_SOCIAL')
    ),
    CONSTRAINT chk_sources_timestamps CHECK (updated_at >= created_at)
);

CREATE UNIQUE INDEX uq_sources_one_enabled_web_search
    ON sources ((channel_type))
    WHERE enabled AND channel_type = 'web_search';

CREATE INDEX idx_sources_management_order ON sources (priority, code, id);
CREATE INDEX idx_sources_snapshot_order ON sources (channel_type, priority, code, id) WHERE enabled;

-- +goose StatementBegin
CREATE FUNCTION guard_source_identity()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' AND OLD.ownership_type = 'fixed' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'fixed Source cannot be deleted';
    END IF;
    IF TG_OP = 'UPDATE' AND (
        NEW.id <> OLD.id OR
        NEW.code <> OLD.code OR
        NEW.ownership_type <> OLD.ownership_type OR
        NEW.channel_type <> OLD.channel_type
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'Source identity fields are immutable';
    END IF;
    RETURN CASE WHEN TG_OP = 'DELETE' THEN OLD ELSE NEW END;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_sources_identity_guard
BEFORE UPDATE OR DELETE ON sources
FOR EACH ROW EXECUTE FUNCTION guard_source_identity();

COMMENT ON TABLE sources IS 'Data-owned Source configuration and runtime snapshot input.';
COMMENT ON COLUMN sources.app_key IS 'Plaintext provider credential returned only through authenticated Data APIs.';

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 000061 is forward-only; use a reviewed forward migration or restore the pre-migration backup';
END;
$$;
-- +goose StatementEnd
