-- +goose Up
CREATE TABLE user_watchlist (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    stock_id TEXT COLLATE "C" NOT NULL CHECK (stock_id ~ '^STK[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
    added_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, stock_id)
);
CREATE INDEX user_watchlist_page_idx ON user_watchlist(user_id, added_at DESC, stock_id ASC);
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'User migrations are forward-only'; END $$;
-- +goose StatementEnd
