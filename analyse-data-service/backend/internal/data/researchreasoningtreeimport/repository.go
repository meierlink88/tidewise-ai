package researchreasoningtreeimport

import (
	"context"
	"database/sql"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchreasoningtreeimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
)

type Repository struct {
	store biz.Store
}

func NewRepository(db *sql.DB) Repository {
	return Repository{store: postgres.NewResearchReasoningTreeImportStore(db)}
}

func (r Repository) InResearchReasoningTreeImportTransaction(ctx context.Context, fn func(biz.Transaction) error) error {
	return r.store.InResearchReasoningTreeImportTransaction(ctx, fn)
}

var _ biz.Store = Repository{}
