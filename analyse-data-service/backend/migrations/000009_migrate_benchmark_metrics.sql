-- +goose Up
-- +goose StatementBegin
DO $$
DECLARE
    old_metric_id UUID;
    old_reference_count INTEGER;
BEGIN
    SELECT id
    INTO old_metric_id
    FROM entity_nodes
    WHERE entity_key = 'metric:fear_index'
    FOR UPDATE;

    IF old_metric_id IS NOT NULL THEN
        IF old_metric_id <> '1a04c8dc-cba8-589c-92bb-6b30d9edb38d'::UUID THEN
            RAISE EXCEPTION 'metric:fear_index has unexpected id %', old_metric_id;
        END IF;
        IF EXISTS (SELECT 1 FROM entity_nodes WHERE entity_key = 'metric:implied_volatility') THEN
            RAISE EXCEPTION 'metric:implied_volatility already exists';
        END IF;
        IF NOT EXISTS (SELECT 1 FROM entity_nodes WHERE entity_key = 'index:vix' AND status = 'active') THEN
            RAISE EXCEPTION 'active index:vix must be preserved';
        END IF;

        SELECT
            (SELECT COUNT(*) FROM entity_edges WHERE from_entity_id = old_metric_id OR to_entity_id = old_metric_id) +
            (SELECT COUNT(*) FROM event_entity_links WHERE entity_id = old_metric_id)
        INTO old_reference_count;
        IF old_reference_count <> 0 THEN
            RAISE EXCEPTION 'metric:fear_index still has % edge or event references', old_reference_count;
        END IF;

        INSERT INTO entity_nodes (
            id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status
        ) VALUES (
            'dd8f7aa1-bb48-5687-a330-312436aacba0'::UUID,
            'metric:implied_volatility',
            'metric',
            'metric',
            '隐含波动率',
            '隐含波动率',
            '{}'::TEXT[],
            'active'
        );

        INSERT INTO metric_profiles (entity_id, metric_type, unit, frequency)
        VALUES (
            'dd8f7aa1-bb48-5687-a330-312436aacba0'::UUID,
            'market_volatility',
            'percent',
            'trading_day'
        );

        DELETE FROM metric_profiles WHERE entity_id = old_metric_id;
        DELETE FROM entity_nodes WHERE id = old_metric_id;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
SELECT 'benchmark metric migration rollback requires a reviewed forward migration or restored backup';
