package postgres

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
)

func TestResearchReadAdapterUsesHalfOpenRangeAndUnboundedDetailID(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repository := newRepository(db)
	from := time.Date(2026, 8, 3, 16, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 4, 16, 0, 0, 0, time.UTC)
	asOf := time.Date(2026, 8, 4, 3, 0, 0, 0, time.UTC)
	themeID := "11111111-1111-4111-8111-111111111111"

	mock.ExpectQuery(listResearchThemesQuery).
		WithArgs(from, to, nil, nil, 6).
		WillReturnRows(researchThemeSummaryRows().AddRow(researchThemeSummaryValues(themeID, from)...))
	mock.ExpectQuery(countResearchThemesQuery).
		WithArgs(from, to).
		WillReturnRows(sqlmock.NewRows([]string{"theme_count", "event_count"}).AddRow(1, 2))

	page, err := repository.ListResearchThemes(context.Background(), research.ThemeListFilter{
		WindowStart: from, WindowEnd: to, AsOf: asOf, Limit: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != themeID || page.ThemeCount != 1 ||
		page.EventCount != 2 || !page.WindowStart.Equal(from) || !page.WindowEnd.Equal(to) {
		t.Fatalf("page = %#v", page)
	}

	detailValues := append(researchThemeSummaryValues(themeID, from),
		[]byte(`[]`), "theme-key", "formal", 2)
	mock.ExpectQuery(getResearchThemeQuery).
		WithArgs(themeID).
		WillReturnRows(researchThemeDetailRows().AddRow(detailValues...))

	detail, err := repository.GetResearchTheme(context.Background(), themeID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.ID != themeID || detail.ThemeKey != "theme-key" {
		t.Fatalf("detail = %#v", detail)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func researchThemeSummaryRows() *sqlmock.Rows {
	return sqlmock.NewRows(researchThemeSummaryColumnNames())
}

func researchThemeSummaryColumnNames() []string {
	return []string{
		"id", "analysis_batch_id", "title", "one_line_conclusion",
		"conclusion_direction", "impact_strength", "attention_level", "conclusion_status",
		"transmission_stage", "investment_guidance_action", "investment_guidance_summary",
		"time_horizon_category", "time_horizon_summary", "transmission_summary",
		"checkpoint_summary", "risk_summary", "analysis_as_of", "window_start", "window_end",
		"published_at", "impacts", "evidence_event_count", "reasoning_tree_count",
	}
}

func researchThemeDetailRows() *sqlmock.Rows {
	return sqlmock.NewRows(append(researchThemeSummaryColumnNames(),
		"events", "theme_key", "publication_mode", "publication_contract_version"))
}

func researchThemeSummaryValues(themeID string, publishedAt time.Time) []driver.Value {
	return []driver.Value{
		themeID, "batch", "Theme", "Conclusion", "positive", "medium", nil, nil,
		"validation", "focus", "Guidance", "short_term", nil, nil, nil, nil,
		publishedAt, publishedAt.Add(-time.Hour), publishedAt, publishedAt,
		[]byte(`[{"node_key":"node","display_name":"Node","chain_node_entity_id":"","name":"Node","relation_role":"beneficiary","impact_direction":"positive","impact_summary":null,"display_order":1}]`),
		2, 0,
	}
}
