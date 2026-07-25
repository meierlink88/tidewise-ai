package entityseed

import (
	"database/sql"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/entityseed"
)

// Repository is the offline entity-seed PostgreSQL adapter. It is kept behind
// the Data layer even though the historical implementation remains extensive.
type Repository struct {
	biz.PostgresRepository
}

func NewRepository(db *sql.DB) Repository {
	return Repository{PostgresRepository: biz.NewPostgresRepository(db)}
}

var _ biz.Repository = Repository{}
