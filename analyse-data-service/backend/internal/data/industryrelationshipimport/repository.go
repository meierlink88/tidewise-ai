package industryrelationshipimport

import (
	"context"
	"database/sql"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/industryrelationshipimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
)

type Repository struct {
	store biz.Store
}

func NewRepository(db *sql.DB) Repository {
	return Repository{store: postgres.NewIndustryRelationshipImportStore(db)}
}

func (r Repository) InIndustryRelationshipImportTransaction(
	ctx context.Context,
	fn func(biz.Transaction) error,
) error {
	return r.store.InIndustryRelationshipImportTransaction(ctx, fn)
}

var _ biz.Store = Repository{}
