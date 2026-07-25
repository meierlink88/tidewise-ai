package research

import (
	"context"
	"database/sql"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
)

type Repository struct {
	store biz.Repository
}

func NewRepository(db *sql.DB) Repository {
	return Repository{store: postgres.NewResearchRepository(db)}
}

func (r Repository) ListResearchThemes(ctx context.Context, filter biz.ThemeListFilter) (biz.ThemeStorePage, error) {
	return r.store.ListResearchThemes(ctx, filter)
}

func (r Repository) GetResearchTheme(ctx context.Context, id string, filter biz.DetailFilter) (biz.ThemeDetailRecord, error) {
	return r.store.GetResearchTheme(ctx, id, filter)
}

func (r Repository) ListResearchThemeReasoningTrees(ctx context.Context, themeID string) (biz.ReasoningTreeListRecord, error) {
	return r.store.ListResearchThemeReasoningTrees(ctx, themeID)
}

func (r Repository) GetResearchThemeReasoningTree(ctx context.Context, themeID, anchorID string) (biz.ReasoningTreeDetailRecord, error) {
	return r.store.GetResearchThemeReasoningTree(ctx, themeID, anchorID)
}

var _ biz.Repository = Repository{}
