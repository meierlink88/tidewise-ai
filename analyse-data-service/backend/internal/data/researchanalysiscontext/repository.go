package researchanalysiscontext

import (
	"context"
	"database/sql"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchanalysiscontext"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
)

type Repository struct {
	store *postgres.ResearchAnalysisContextStore
}

func NewRepository(db *sql.DB) Repository {
	return Repository{store: postgres.NewResearchAnalysisContextStore(db)}
}

func (r Repository) List(
	ctx context.Context,
	query biz.StoreQuery,
) (biz.StorePage, error) {
	return r.store.List(ctx, query)
}

var _ biz.Store = Repository{}
