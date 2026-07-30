package researchgraph

import (
	"context"
	"database/sql"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchgraph"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
)

type Repository struct {
	store *postgres.ResearchGraphStore
}

func NewRepository(db *sql.DB) Repository {
	return Repository{store: postgres.NewResearchGraphStore(db)}
}

func (r Repository) Search(
	ctx context.Context,
	query biz.Query,
) (biz.Subgraph, error) {
	return r.store.Search(ctx, query)
}

var _ biz.Store = Repository{}
