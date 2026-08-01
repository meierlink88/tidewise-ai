package dbmigration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEventSemanticV2MigrationMakesMeasurementNarrativeOnlyForNewWrites(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", "000036_event_semantic_v2.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(payload))
	for _, fragment := range []string{
		"drop constraint chk_variable_signal_measurement_values",
		"drop constraint chk_variable_signal_measurement_change_unit",
		"drop constraint chk_variable_signal_measurement_units",
		"drop constraint chk_variable_signal_measurement_conversion",
		"drop constraint chk_variable_signal_measurement_raw_range",
		"drop constraint chk_variable_signal_measurement_canonical_range",
		"alter column measurement_role drop not null",
		"alter column value_shape drop not null",
		"alter column raw_unit drop not null",
		"alter column canonical_unit drop not null",
		"add column evidence_ids uuid[]",
		"set evidence_ids = array[evidence_id]",
		"create function event_semantic_measurement_evidence_ids_compat()",
		"create trigger trg_event_semantic_measurement_evidence_ids_compat",
		"new.evidence_ids := array[new.evidence_id]",
		"alter column evidence_ids set not null",
		"add column allowed_event_roles",
		"add column allowed_units",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("Event Semantic V2 migration missing %q", fragment)
		}
	}
	for _, forbidden := range []string{
		"drop table direct_impact_assertions",
		"delete from direct_impact_assertions",
		"drop table variable_signal_measurements",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("Event Semantic V2 migration contains destructive fragment %q", forbidden)
		}
	}
}
