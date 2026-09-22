-- +goose Up
SET LOCAL lock_timeout = '5s';
CREATE TABLE stock (
 id VARCHAR(39) PRIMARY KEY CHECK (id ~ '^STK[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
 code VARCHAR(6) NOT NULL CHECK (code ~ '^[0-9]{6}$'),
 name VARCHAR(64) NOT NULL CHECK (btrim(name) <> ''),
 exchange VARCHAR(2) NOT NULL CHECK (exchange IN ('SH','SZ','BJ')),
 board VARCHAR(16) NOT NULL,
 as_of DATE NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(exchange,code),
 CHECK ((exchange='SH' AND board IN ('主板','科创板')) OR (exchange='SZ' AND board IN ('主板','创业板')) OR (exchange='BJ' AND board='北交所'))
);
CREATE INDEX stock_code_idx ON stock(code);
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'stock master migration is forward-only; retain catalog on application rollback'; END $$;
-- +goose StatementEnd
