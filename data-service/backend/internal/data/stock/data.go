package stock

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	stockbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/stock"
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, stockbiz.ErrPersistence
	}
	return &Store{db: db}, nil
}
func (s *Store) Upsert(ctx context.Context, items []stockbiz.Stock) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return persistence(ctx, err)
	}
	defer tx.Rollback()
	// Serialize catalog publishers, including insertion of previously unseen natural keys.
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(533,1)`); err != nil {
		return persistence(ctx, err)
	}
	for _, item := range items {
		var old stockbiz.Stock
		err = tx.QueryRowContext(ctx, `SELECT id,name,board,as_of FROM stock WHERE exchange=$1 AND code=$2 FOR UPDATE`, item.Exchange, item.Code).Scan(&old.ID, &old.Name, &old.Board, &old.AsOf)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return persistence(ctx, err)
		}
		if err == nil {
			if err := stockbiz.ValidateReplacement(old, item); err != nil {
				return err
			}
			if old.AsOf.Equal(item.AsOf) {
				continue
			}
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO stock(id,code,name,exchange,board,as_of) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(exchange,code) DO UPDATE SET name=EXCLUDED.name,board=EXCLUDED.board,as_of=EXCLUDED.as_of,updated_at=now()`, item.ID, item.Code, item.Name, item.Exchange, item.Board, item.AsOf)
		if err != nil {
			return persistence(ctx, err)
		}
	}
	if err = tx.Commit(); err != nil {
		return persistence(ctx, err)
	}
	return nil
}
func (s *Store) Search(ctx context.Context, q stockbiz.Query) (stockbiz.Page, error) {
	literal := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q.Text)
	rows, err := s.db.QueryContext(ctx, `SELECT id,code,name,exchange,board,as_of FROM stock WHERE ($2='' OR exchange=$2) AND (code LIKE '%'||$1||'%' ESCAPE '\' OR name ILIKE '%'||$1||'%' ESCAPE '\' OR code||'.'||exchange ILIKE '%'||$1||'%' ESCAPE '\') ORDER BY CASE WHEN code=$3 OR code||'.'||exchange=upper($3) THEN 0 ELSE 1 END,exchange,code,id LIMIT $4 OFFSET $5`, literal, q.Exchange, q.Text, q.Limit+1, q.Offset)
	if err != nil {
		return stockbiz.Page{}, persistence(ctx, err)
	}
	defer rows.Close()
	page := stockbiz.Page{Items: []stockbiz.Stock{}}
	for rows.Next() {
		var item stockbiz.Stock
		if err = rows.Scan(&item.ID, &item.Code, &item.Name, &item.Exchange, &item.Board, &item.AsOf); err != nil {
			return stockbiz.Page{}, persistence(ctx, err)
		}
		page.Items = append(page.Items, item)
	}
	if err = rows.Err(); err != nil {
		return stockbiz.Page{}, persistence(ctx, err)
	}
	if len(page.Items) > q.Limit {
		page.HasMore = true
		page.Items = page.Items[:q.Limit]
	}
	return page, nil
}
func persistence(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return stockbiz.ErrPersistence
}
