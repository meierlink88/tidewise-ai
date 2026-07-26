package eventpublication

import (
	"context"
	"database/sql"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventpublication"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
)

type Repository struct {
	store biz.Store
}

func NewRepository(db *sql.DB) Repository {
	return Repository{store: postgres.NewEventPublicationStore(db)}
}

func (r Repository) InEventPublicationTransaction(ctx context.Context, fn func(biz.Transaction) error) error {
	return r.store.InEventPublicationTransaction(ctx, fn)
}

var _ biz.Store = Repository{}
