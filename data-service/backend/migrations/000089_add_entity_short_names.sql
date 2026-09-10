-- +goose Up
SET LOCAL lock_timeout = '5s';

ALTER TABLE industry_chain ADD COLUMN short_name TEXT
    CONSTRAINT chk_industry_chain_short_name CHECK (short_name IS NULL OR short_name ~ '[^[:space:]]');
ALTER TABLE chain_node ADD COLUMN short_name TEXT
    CONSTRAINT chk_chain_node_short_name CHECK (short_name IS NULL OR short_name ~ '[^[:space:]]');
ALTER TABLE macro_economics ADD COLUMN short_name TEXT
    CONSTRAINT chk_macro_economics_short_name CHECK (short_name IS NULL OR short_name ~ '[^[:space:]]');
ALTER TABLE geopolitic_rivalries ADD COLUMN short_name TEXT
    CONSTRAINT chk_geopolitic_rivalries_short_name CHECK (short_name IS NULL OR short_name ~ '[^[:space:]]');

COMMENT ON COLUMN industry_chain.short_name IS '可选简称，用于后续小程序锚点标签；NULL 表示未填写，不替代正式名称或检索别名。';
COMMENT ON COLUMN chain_node.short_name IS '可选简称，用于后续小程序锚点标签；NULL 表示未填写，不替代正式名称或检索别名。';
COMMENT ON COLUMN macro_economics.short_name IS '可选简称，用于后续小程序锚点标签；NULL 表示未填写，不替代正式名称。';
COMMENT ON COLUMN geopolitic_rivalries.short_name IS '可选简称，用于后续小程序锚点标签；NULL 表示未填写，不替代正式名称。';

-- +goose Down
-- Preserve reviewed short names when rolling back application code.
SELECT 1;
