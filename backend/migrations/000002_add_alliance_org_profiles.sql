-- +goose Up
CREATE TABLE alliance_org_profiles (
    entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id),
    org_code VARCHAR(64) NOT NULL,
    org_type VARCHAR(64) NOT NULL,
    primary_domain VARCHAR(64) NOT NULL DEFAULT '',
    scope_region VARCHAR(64) NOT NULL DEFAULT '',
    official_url TEXT NOT NULL DEFAULT '',
    UNIQUE (org_code)
);

CREATE INDEX idx_alliance_org_profiles_org_type ON alliance_org_profiles (org_type);
CREATE INDEX idx_alliance_org_profiles_primary_domain ON alliance_org_profiles (primary_domain);

-- +goose Down
SELECT 'alliance org profile rollback requires a reviewed forward migration or restored backup';
