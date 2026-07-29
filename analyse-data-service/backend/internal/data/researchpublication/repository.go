package researchpublication

import (
	"context"
	"database/sql"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchpublication"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
)

type Repository struct{ store researchpublication.Store }

func NewRepository(db *sql.DB) Repository {
	return Repository{store: postgres.NewResearchPublicationStore(db)}
}

func (r Repository) InResearchPublicationTransaction(
	ctx context.Context,
	fn func(researchpublication.Transaction) error,
) error {
	return r.store.InResearchPublicationTransaction(ctx, fn)
}

var _ researchpublication.Store = Repository{}
