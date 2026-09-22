package watchlist

import (
	"database/sql"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository {
	if db == nil {
		panic("missing user database")
	}
	return &Repository{db}
}
