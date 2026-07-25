package adminquery

import (
	"context"
	"database/sql"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/adminquery"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
)

type Repository struct {
	store biz.Repository
}

func NewRepository(db *sql.DB) Repository {
	return Repository{store: postgres.NewAdminQueryRepository(db)}
}

func (r Repository) ListRawDocuments(ctx context.Context, filter biz.RawDocumentListFilter) (biz.RawDocumentStorePage, error) {
	return r.store.ListRawDocuments(ctx, filter)
}

func (r Repository) ListEvents(ctx context.Context, filter biz.EventListFilter) (biz.EventStorePage, error) {
	return r.store.ListEvents(ctx, filter)
}

var _ biz.Repository = Repository{}
