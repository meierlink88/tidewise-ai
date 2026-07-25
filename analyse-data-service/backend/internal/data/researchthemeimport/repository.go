package researchthemeimport

import (
	"context"
	"database/sql"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
)

type Repository struct {
	store biz.Store
}

func NewRepository(db *sql.DB) Repository {
	return Repository{store: postgres.NewResearchThemeImportStore(db)}
}

func (r Repository) InResearchThemeImportTransaction(ctx context.Context, fn func(biz.Transaction) error) error {
	return r.store.InResearchThemeImportTransaction(ctx, fn)
}

var _ biz.Store = Repository{}
