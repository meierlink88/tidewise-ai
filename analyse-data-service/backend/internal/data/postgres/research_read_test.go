package postgres

import (
	"strings"
	"testing"
)

func TestResearchReadQueriesUseHalfOpenPublicationRangeAndUnboundedDetailID(t *testing.T) {
	if !strings.Contains(listResearchThemesQuery, "theme.published_at >= $1") ||
		!strings.Contains(listResearchThemesQuery, "theme.published_at < $2") ||
		strings.Contains(listResearchThemesQuery, "theme.published_at <= $2") {
		t.Fatalf("list query does not use [published_from, published_to): %s", listResearchThemesQuery)
	}
	if !strings.Contains(countResearchThemesQuery, "theme.published_at >= $1 AND theme.published_at < $2") {
		t.Fatalf("count query does not use the list's half-open range: %s", countResearchThemesQuery)
	}
	if !strings.Contains(getResearchThemeQuery, "WHERE t.id = $1") ||
		strings.Contains(getResearchThemeQuery, "WHERE t.id = $1 AND") {
		t.Fatalf("detail query still applies list-window membership: %s", getResearchThemeQuery)
	}
}
