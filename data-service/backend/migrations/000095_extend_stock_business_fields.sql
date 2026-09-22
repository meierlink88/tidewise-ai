-- +goose Up
SET LOCAL lock_timeout = '5s';

-- Nullable, without defaults: existing stocks have unknown rather than empty facts.
ALTER TABLE stock
    ADD COLUMN full_name TEXT,
    ADD COLUMN former_name TEXT,
    ADD COLUMN list_date DATE,
    ADD COLUMN established DATE,
    ADD COLUMN industry_l1 TEXT,
    ADD COLUMN industry_l2 TEXT,
    ADD COLUMN main_business JSONB,
    ADD COLUMN main_product_type TEXT[],
    ADD COLUMN index_core TEXT[],
    ADD COLUMN concepts TEXT[],
    ADD CONSTRAINT stock_main_business_shape CHECK (
        main_business IS NULL OR (
            jsonb_typeof(main_business) = 'array'
            AND NOT jsonb_path_exists(main_business,
                '$[*] ? (@.type() != "object" || !exists(@.name) || @.name.type() != "string" || @.name like_regex "^\\s*$" || !exists(@.pct) || @.pct.type() != "number")')
        )
    ),
    ADD CONSTRAINT stock_product_types_shape CHECK (
        (array_ndims(main_product_type) IS NULL OR array_ndims(main_product_type) = 1)
        AND array_position(main_product_type, NULL) IS NULL
    ),
    ADD CONSTRAINT stock_core_indexes_shape CHECK (
        (array_ndims(index_core) IS NULL OR array_ndims(index_core) = 1)
        AND array_position(index_core, NULL) IS NULL
    ),
    ADD CONSTRAINT stock_concepts_shape CHECK (
        (array_ndims(concepts) IS NULL OR array_ndims(concepts) = 1)
        AND array_position(concepts, NULL) IS NULL
    );

COMMENT ON COLUMN stock.former_name IS 'Source text of former company names; not a reconstructed name history';
COMMENT ON COLUMN stock.main_business IS 'Ordered revenue breakdown {name,pct}; pct is a percentage, may be negative, and need not sum to 100. NULL means unknown; [] means known empty';
COMMENT ON COLUMN stock.main_product_type IS 'Ordered product categories; NULL means unknown, empty array means known empty';
COMMENT ON COLUMN stock.index_core IS 'Core broad-based index names, independent of concepts; NULL means unknown, empty array means no memberships';
COMMENT ON COLUMN stock.concepts IS 'Ordered source theme labels without weights; not formal Concept references. NULL means unknown, empty array means known empty';

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'stock business fields migration is forward-only; retain fields on application rollback'; END $$;
-- +goose StatementEnd
