package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
)

func TestResearchThemeHistoryPostgresHalfOpenBoundsAndStablePaging(t *testing.T) {
	db := openResearchV1TestDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	from := time.Date(2026, 8, 3, 16, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 4, 16, 0, 0, 0, time.UTC)
	themes := historyIntegrationThemes(from, to)
	seedHistoryIntegrationThemes(t, ctx, db, themes, to.Add(time.Hour))

	service := research.NewService(NewResearchRepository(db), func() time.Time { return to.Add(time.Hour) })
	var cursor string
	var gotIDs []string
	for pageNumber := 1; pageNumber <= 3; pageNumber++ {
		page, err := service.ListThemes(ctx, research.ResearchListRequest{
			PublishedFrom: &from, PublishedTo: &to, Limit: 5, Cursor: cursor,
		})
		if err != nil {
			t.Fatalf("page %d: %v", pageNumber, err)
		}
		if page.ThemeCount != 15 || page.EventCount != 0 || len(page.Items) != 5 {
			t.Fatalf("page %d counts/items = %d/%d/%d", pageNumber, page.ThemeCount, page.EventCount, len(page.Items))
		}
		for _, item := range page.Items {
			gotIDs = append(gotIDs, item.ID)
		}
		if pageNumber < 3 {
			if page.NextCursor == nil {
				t.Fatalf("page %d has no next cursor", pageNumber)
			}
			cursor = *page.NextCursor
		} else if page.NextCursor != nil {
			t.Fatalf("last page cursor = %q, want nil", *page.NextCursor)
		}
	}

	wantIncluded := append([]historyIntegrationTheme(nil), themes[1:16]...)
	sort.Slice(wantIncluded, func(i, j int) bool {
		if wantIncluded[i].publishedAt.Equal(wantIncluded[j].publishedAt) {
			return wantIncluded[i].id < wantIncluded[j].id
		}
		return wantIncluded[i].publishedAt.After(wantIncluded[j].publishedAt)
	})
	wantIDs := make([]string, 0, len(wantIncluded))
	for _, theme := range wantIncluded {
		wantIDs = append(wantIDs, theme.id)
	}
	if fmt.Sprint(gotIDs) != fmt.Sprint(wantIDs) {
		t.Fatalf("paged IDs = %v, want %v", gotIDs, wantIDs)
	}

	detail, err := service.GetTheme(ctx, themes[0].id, research.ResearchDetailRequest{})
	if err != nil {
		t.Fatalf("historical detail outside list range: %v", err)
	}
	if detail.Theme.ID != themes[0].id {
		t.Fatalf("detail ID = %q, want %q", detail.Theme.ID, themes[0].id)
	}
}

type historyIntegrationTheme struct {
	id          string
	key         string
	publishedAt time.Time
}

func historyIntegrationThemes(from, to time.Time) []historyIntegrationTheme {
	values := []historyIntegrationTheme{{
		id: "20000000-0000-4000-8000-000000000001", key: "history.before", publishedAt: from.Add(-time.Nanosecond),
	}}
	for index := 0; index < 15; index++ {
		publishedAt := from.Add(time.Duration(index) * time.Hour)
		if index == 2 {
			publishedAt = from.Add(time.Hour)
		}
		values = append(values, historyIntegrationTheme{
			id:  fmt.Sprintf("20000000-0000-4000-8000-%012d", index+2),
			key: fmt.Sprintf("history.in-range-%02d", index), publishedAt: publishedAt,
		})
	}
	values = append(values,
		historyIntegrationTheme{id: "20000000-0000-4000-8000-000000000017", key: "history.at-upper", publishedAt: to},
		historyIntegrationTheme{id: "20000000-0000-4000-8000-000000000018", key: "history.after-upper", publishedAt: to.Add(time.Nanosecond)},
	)
	return values
}

func seedHistoryIntegrationThemes(t *testing.T, ctx context.Context, db *sql.DB, themes []historyIntegrationTheme, importedAt time.Time) {
	t.Helper()
	themeIDs := make(map[string]string, len(themes))
	for _, theme := range themes {
		themeIDs[theme.key] = theme.id
	}
	encodedThemeIDs, err := json.Marshal(themeIDs)
	if err != nil {
		t.Fatal(err)
	}
	const receiptID = "20000000-0000-4000-8000-000000000000"
	const batchID = "theme-history-pagination-integration"
	if _, err := db.ExecContext(ctx, `
INSERT INTO research_theme_import_receipts (
    id, analysis_batch_id, publisher_subject, payload_hash, theme_ids_by_key,
    write_counts, published_at, imported_at
) VALUES ($1, $2, 'integration-test', repeat('0', 64), $3,
    jsonb_build_object('themes', $4::int, 'impacts', $4::int,
        'event_associations', 0, 'receipts', 1), $5::timestamptz, $5::timestamptz)`,
		receiptID, batchID, encodedThemeIDs, len(themes), importedAt); err != nil {
		t.Fatal(err)
	}
	for index, theme := range themes {
		if _, err := db.ExecContext(ctx, `
INSERT INTO research_themes (
    id, theme_key, analysis_batch_id, import_receipt_id, title, one_line_conclusion,
    conclusion_direction, impact_strength, transmission_stage, investment_guidance_action,
    investment_guidance_summary, time_horizon_category, analysis_as_of, window_start,
    window_end, published_at
) VALUES ($1, $2, $3, $4, $2, 'Conclusion', 'positive', 'medium', 'validation',
    'focus', 'Guidance', 'short_term', $5::timestamptz,
    $5::timestamptz - interval '1 hour', $5::timestamptz, $5::timestamptz)`,
			theme.id, theme.key, batchID, receiptID, theme.publishedAt); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, `
INSERT INTO research_theme_impacts (
    theme_id, node_key, display_name, relation_role, impact_direction, display_order
) VALUES ($1, $2, $3, 'beneficiary', 'positive', 1)`,
			theme.id, fmt.Sprintf("history.node-%02d", index), fmt.Sprintf("Node %02d", index)); err != nil {
			t.Fatal(err)
		}
	}
}
