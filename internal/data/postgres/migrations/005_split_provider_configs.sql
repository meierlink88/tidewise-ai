CREATE TABLE model_provider_configs (
    provider_key text PRIMARY KEY,
    base_url text NOT NULL,
    model text NOT NULL,
    api_key text NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE connector_configs (
    connector_key text PRIMARY KEY,
    base_url text NOT NULL,
    api_key text NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now()
);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM provider_configs
        WHERE provider_key NOT IN (
            'deepseek',
            'parallel_search',
            'tavily',
            'bocha',
            'cls_telegraph',
            'eastmoney_fastnews',
            'eastmoney_stock_news',
            'stcn_quicknews'
        )
    ) THEN
        RAISE EXCEPTION 'provider_configs contains an unknown provider_key';
    END IF;
END
$$;

INSERT INTO model_provider_configs (provider_key, base_url, model, api_key, updated_at)
SELECT provider_key, base_url, model, api_key, updated_at
FROM provider_configs
WHERE provider_key = 'deepseek';

INSERT INTO connector_configs (connector_key, base_url, api_key, updated_at)
SELECT provider_key, base_url, api_key, updated_at
FROM provider_configs
WHERE provider_key IN (
    'parallel_search',
    'tavily',
    'bocha',
    'cls_telegraph',
    'eastmoney_fastnews',
    'eastmoney_stock_news',
    'stcn_quicknews'
);

DO $$
DECLARE
    old_count bigint;
    migrated_count bigint;
BEGIN
    SELECT count(*) INTO old_count FROM provider_configs;
    SELECT
        (SELECT count(*) FROM model_provider_configs) +
        (SELECT count(*) FROM connector_configs)
    INTO migrated_count;
    IF old_count <> migrated_count THEN
        RAISE EXCEPTION 'provider_configs migration did not preserve every row';
    END IF;
END
$$;

DROP TABLE provider_configs;
