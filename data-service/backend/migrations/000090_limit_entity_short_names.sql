-- +goose Up
SET LOCAL lock_timeout = '5s';

ALTER TABLE industry_chain ADD CONSTRAINT chk_industry_chain_short_name_length
    CHECK (short_name IS NULL OR char_length(short_name) BETWEEN 1 AND 5);
ALTER TABLE chain_node ADD CONSTRAINT chk_chain_node_short_name_length
    CHECK (short_name IS NULL OR char_length(short_name) BETWEEN 1 AND 5);
ALTER TABLE macro_economics ADD CONSTRAINT chk_macro_economics_short_name_length
    CHECK (short_name IS NULL OR char_length(short_name) BETWEEN 1 AND 5);
ALTER TABLE geopolitic_rivalries ADD CONSTRAINT chk_geopolitic_rivalries_short_name_length
    CHECK (short_name IS NULL OR char_length(short_name) BETWEEN 1 AND 5);

-- +goose Down
-- Preserve the display-name contract when reverting application code.
SELECT 1;
