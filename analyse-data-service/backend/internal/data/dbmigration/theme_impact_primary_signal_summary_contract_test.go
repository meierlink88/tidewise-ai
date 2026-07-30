package dbmigration

import (
	"regexp"
	"strings"
	"testing"
)

func TestThemeImpactPrimarySignalSummaryMigrationIsAdditiveAndLegacySafe(t *testing.T) {
	raw := readMigration(t, "000034_add_theme_impact_primary_signal_summary.sql")
	up, down := migrationSections(t, raw)
	normalized := strings.ToLower(up)

	for _, fragment := range []string{
		"alter table research_theme_impacts",
		"add column primary_signal_display_summary text",
		"primary_signal_display_summary is null",
		"char_length(primary_signal_display_summary) between 1 and 200",
		"primary_signal_display_summary = btrim(primary_signal_display_summary)",
	} {
		if !strings.Contains(normalized, fragment) {
			t.Errorf("theme impact primary signal migration Up must contain %q", fragment)
		}
	}
	if strings.Contains(normalized, "primary_signal_display_summary text not null") {
		t.Fatal("legacy Theme Impact snapshots must remain readable without a semantic backfill")
	}
	dml := regexp.MustCompile(`(?mi)^\s*(insert\s+into|update\s+|delete\s+from|truncate\s+)`)
	if match := dml.FindString(up); match != "" {
		t.Fatalf("theme impact primary signal migration must contain no business DML, found %q", strings.TrimSpace(match))
	}
	down = strings.ToLower(down)
	if !strings.Contains(down, "migration 000034 is forward-only") ||
		!strings.Contains(down, "raise exception") {
		t.Fatal("theme impact primary signal migration Down must fail closed")
	}
}
