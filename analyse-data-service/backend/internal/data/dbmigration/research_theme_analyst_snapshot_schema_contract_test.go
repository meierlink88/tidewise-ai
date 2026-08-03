package dbmigration

import (
	"strings"
	"testing"
)

func TestAnalystSnapshotMigrationKeepsFormalSignalIdentityStrict(t *testing.T) {
	raw := readMigration(t, "000041_add_research_theme_analyst_snapshot_v3.sql")
	up, _ := migrationSections(t, raw)
	normalized := strings.Join(strings.Fields(strings.ToLower(up)), " ")

	for _, fragment := range []string{
		"source_kind = 'analyst_snapshot' and variable_signal_key is null and signal_key is not null",
		"source_kind <> 'analyst_snapshot' and variable_signal_key is not null and signal_key is null",
		"source_kind = 'analyst_snapshot' or signal_direction is not null",
	} {
		if !strings.Contains(normalized, fragment) {
			t.Errorf("analyst snapshot migration must enforce %q", fragment)
		}
	}
}
