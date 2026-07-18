package dbmigration

import (
	"strings"
	"testing"
)

func TestBenchmarkSchemaMigrationDefinesProfilesAndObservations(t *testing.T) {
	sql := readMigration(t, "000008_add_benchmark_foundation.sql")
	normalized := strings.ToLower(sql)

	required := []string{
		"create table if not exists benchmark_profiles",
		"entity_id uuid primary key",
		"references entity_nodes(id)",
		"benchmark_type text not null",
		"official_series_code text",
		"provider text not null",
		"tenor text",
		"underlying_symbol text",
		"currency_code text not null",
		"unit text not null",
		"frequency text not null",
		"source_url text not null",
		"create table if not exists benchmark_observations",
		"benchmark_entity_id uuid not null",
		"observed_at timestamptz not null",
		"value numeric not null",
		"source_name text not null",
		"quality_status text not null",
		"unique (benchmark_entity_id, observed_at, source_name)",
		"quality_status in ('raw', 'validated', 'suspect', 'rejected')",
		"create index if not exists idx_benchmark_observations_benchmark_time",
	}
	for _, fragment := range required {
		if !strings.Contains(normalized, fragment) {
			t.Fatalf("migration missing required fragment %q", fragment)
		}
	}
}

func TestBenchmarkSchemaMigrationIsNonDestructive(t *testing.T) {
	sql := readMigration(t, "000008_add_benchmark_foundation.sql")
	normalized := strings.ToLower(sql)

	forbidden := []string{
		"drop table",
		"truncate",
		"delete from entity_nodes",
		"delete from entity_edges",
		"alter table entity_nodes",
		"alter table entity_edges",
	}
	for _, fragment := range forbidden {
		if strings.Contains(normalized, fragment) {
			t.Fatalf("migration contains destructive fragment %q", fragment)
		}
	}
}

func TestBenchmarkMetricMigrationUsesReviewedDeterministicReplacement(t *testing.T) {
	sql := readMigration(t, "000009_migrate_benchmark_metrics.sql")
	normalized := strings.ToLower(sql)

	required := []string{
		"-- +goose statementbegin",
		"-- +goose statementend",
		"metric:fear_index",
		"metric:implied_volatility",
		"1a04c8dc-cba8-589c-92bb-6b30d9edb38d",
		"dd8f7aa1-bb48-5687-a330-312436aacba0",
		"for update",
		"insert into entity_nodes",
		"insert into metric_profiles",
		"market_volatility",
		"delete from metric_profiles",
		"delete from entity_nodes",
		"index:vix",
	}
	for _, fragment := range required {
		if !strings.Contains(normalized, fragment) {
			t.Fatalf("metric migration missing required fragment %q", fragment)
		}
	}
	if strings.Contains(normalized, "metric:gold_price") {
		t.Fatal("metric migration must leave gold_price creation to reviewed entity seed")
	}
}
