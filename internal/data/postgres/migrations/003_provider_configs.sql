CREATE TABLE IF NOT EXISTS provider_configs (
    provider_key text PRIMARY KEY,
    base_url text NOT NULL,
    model text NOT NULL DEFAULT '',
    api_key text NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now()
);
