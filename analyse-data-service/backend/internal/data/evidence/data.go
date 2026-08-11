package evidence

import (
	"database/sql"
	"errors"

	evidencebiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/evidence"
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (Store, error) {
	if db == nil {
		return Store{}, errors.New("Evidence database is required")
	}
	return Store{db: db}, nil
}

var _ evidencebiz.Store = Store{}
