package evidencepublication

import (
	"context"
	"database/sql"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/evidencepublication"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
)

type Repository struct{ store biz.Store }

func NewRepository(db *sql.DB) Repository {
	return Repository{store: postgres.NewEvidencePublicationStore(db)}
}

func (r Repository) InTransaction(ctx context.Context, fn func(biz.Transaction) error) error {
	return r.store.InTransaction(ctx, fn)
}

var _ biz.Store = Repository{}
