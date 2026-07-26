package researchanchorimport

import (
	"context"
	"database/sql"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchanchorimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
)

type Repository struct {
	store biz.Store
}

func NewRepository(db *sql.DB) Repository {
	return Repository{store: postgres.NewResearchAnchorImportStore(db)}
}

func (r Repository) InResearchAnchorImportTransaction(ctx context.Context, fn func(biz.Transaction) error) error {
	return r.store.InResearchAnchorImportTransaction(ctx, fn)
}

var _ biz.Store = Repository{}
