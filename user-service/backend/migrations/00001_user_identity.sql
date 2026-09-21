-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY,
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMPTZ
);

CREATE TABLE wechat_identities (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    appid TEXT COLLATE "C" NOT NULL
        CHECK (appid <> '' AND appid = btrim(appid)),
    openid TEXT COLLATE "C" NOT NULL
        CHECK (openid <> '' AND openid = btrim(openid)),
    unionid TEXT COLLATE "C"
        CHECK (unionid IS NULL OR (unionid <> '' AND unionid = btrim(unionid))),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT wechat_identities_app_openid_unique UNIQUE (appid, openid),
    CONSTRAINT wechat_identities_user_app_unique UNIQUE (user_id, appid)
);
-- V1 has no identity rebind/merge interface. unionid is metadata, not a login key.
-- Never automatically merge different users based on unionid.

CREATE TABLE user_sessions (
    id UUID PRIMARY KEY,
    wechat_identity_id UUID NOT NULL REFERENCES wechat_identities(id) ON DELETE RESTRICT,
    token_hash BYTEA NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    CONSTRAINT user_sessions_valid_expiry CHECK (expires_at > created_at),
    CONSTRAINT user_sessions_valid_revocation CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);
CREATE INDEX user_sessions_identity_active_idx
    ON user_sessions (wechat_identity_id) WHERE revoked_at IS NULL;
CREATE INDEX user_sessions_expiry_idx ON user_sessions (expires_at);

-- A session maps to exactly one identity and user through FK joins;
-- do not duplicate user_id/appid here and permit contradictory relationships.
-- updated_at and last_login_at are maintained explicitly by business transactions.


-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'User migrations are forward-only'; END $$;
-- +goose StatementEnd
