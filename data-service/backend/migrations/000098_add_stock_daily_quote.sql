-- +goose Up
SET LOCAL lock_timeout = '5s';
CREATE TABLE stock_daily_quote (
 id VARCHAR(39) PRIMARY KEY CHECK (id ~ '^SDQ[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
 stock_id VARCHAR(39) NOT NULL REFERENCES stock(id) ON DELETE RESTRICT,
 trade_date DATE NOT NULL,
 open_price NUMERIC(20,6),
 high_price NUMERIC(20,6),
 low_price NUMERIC(20,6),
 close_price NUMERIC(20,6) NOT NULL CHECK (close_price > 0 AND close_price <> 'NaN'::numeric),
 volume_lots NUMERIC(24,6) NOT NULL CHECK (volume_lots >= 0 AND volume_lots <> 'NaN'::numeric),
 turnover_rate_pct NUMERIC(20,10) CHECK (turnover_rate_pct >= 0 AND turnover_rate_pct <> 'NaN'::numeric),
 change_pct NUMERIC(20,10) CHECK (change_pct <> 'NaN'::numeric),
 record_status VARCHAR(16) NOT NULL CHECK (record_status IN ('observed','placeholder')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(stock_id,trade_date),
 CHECK (
  (record_status='placeholder' AND open_price IS NULL AND high_price IS NULL AND low_price IS NULL AND volume_lots=0)
  OR
  (record_status='observed' AND open_price IS NOT NULL AND high_price IS NOT NULL AND low_price IS NOT NULL
   AND low_price>0 AND high_price >= low_price AND high_price <> 'NaN'::numeric
   AND open_price BETWEEN low_price AND high_price AND close_price BETWEEN low_price AND high_price)
 )
);
CREATE INDEX stock_daily_quote_date_idx ON stock_daily_quote(trade_date,stock_id);
COMMENT ON TABLE stock_daily_quote IS 'Unadjusted CNY daily quotes. Source-declared basis; not adjusted returns or intraday trades.';
COMMENT ON COLUMN stock_daily_quote.volume_lots IS 'Source lots (手), not shares';
COMMENT ON COLUMN stock_daily_quote.change_pct IS 'Source daily percent, signed; not recomputed from adjacent closes';
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'stock daily quotes migration is forward-only; retain quotes on application rollback'; END $$;
-- +goose StatementEnd
