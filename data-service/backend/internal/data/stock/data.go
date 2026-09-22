package stock

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	pinyin "github.com/mozillazg/go-pinyin"
	"strings"
	"unicode"

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
				if _, err = tx.ExecContext(ctx, `UPDATE stock SET name_initials=$2 WHERE id=$1 AND name_initials IS DISTINCT FROM $2`, item.ID, Initials(item.Name)); err != nil {
					return persistence(ctx, err)
				}
				continue
			}
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO stock(id,code,name,exchange,board,as_of,name_initials) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(exchange,code) DO UPDATE SET name_initials=EXCLUDED.name_initials,name=EXCLUDED.name,board=EXCLUDED.board,as_of=EXCLUDED.as_of,updated_at=now()`, item.ID, item.Code, item.Name, item.Exchange, item.Board, item.AsOf, Initials(item.Name))
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
	var rows *sql.Rows
	var err error
	if len(q.IDs) > 0 {
		rows, err = s.db.QueryContext(ctx, `SELECT id,code,name,exchange,board,as_of,full_name,industry_l1,industry_l2,array_to_json(concepts) FROM stock WHERE id=ANY($1) ORDER BY exchange,code,id`, q.IDs)
	} else {
		rows, err = s.db.QueryContext(ctx, `SELECT id,code,name,exchange,board,as_of,full_name,industry_l1,industry_l2,array_to_json(concepts) FROM stock
 WHERE ($2='' OR exchange=$2) AND (code LIKE '%'||$1||'%' ESCAPE '\' OR name ILIKE '%'||$1||'%' ESCAPE '\'
 OR name_initials LIKE '%'||upper($1)||'%' ESCAPE '\' OR code||'.'||exchange ILIKE '%'||$1||'%' ESCAPE '\')
 ORDER BY CASE WHEN code=$3 OR code||'.'||exchange=upper($3) THEN 0 ELSE 1 END,exchange,code,id LIMIT $4 OFFSET $5`, literal, q.Exchange, q.Text, q.Limit+1, q.Offset)
	}
	if err != nil {
		return stockbiz.Page{}, persistence(ctx, err)
	}
	defer rows.Close()
	page := stockbiz.Page{Items: []stockbiz.Stock{}}
	for rows.Next() {
		var item stockbiz.Stock
		var concepts []byte
		if err = rows.Scan(&item.ID, &item.Code, &item.Name, &item.Exchange, &item.Board, &item.AsOf, &item.FullName, &item.IndustryL1, &item.IndustryL2, &concepts); err != nil {
			return stockbiz.Page{}, persistence(ctx, err)
		}
		if concepts != nil && json.Unmarshal(concepts, &item.Concepts) != nil {
			return stockbiz.Page{}, stockbiz.ErrPersistence
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

// Initials is a derived search representation, not an additional source fact.
// Phrase overrides retain common security-name pronunciations; ASCII is kept.
func Initials(name string) string {
	name = strings.NewReplacer("银行", "银H", "重庆", "C庆", "重工", "Z工", "长江", "C江", "长城", "C城", "长安", "C安", "长沙", "C沙", "长春", "C春", "长虹", "C虹", "长电", "C电", "长信", "C信", "长源", "C源", "长高", "C高", "长亮", "C亮", "长盛", "C盛", "长龄", "C龄", "长青", "C青", "长华", "C华", "长缆", "C缆", "厦门", "X门", "六安", "L安").Replace(name)
	args := pinyin.NewArgs()
	args.Style = pinyin.FirstLetter
	args.Fallback = func(v rune, _ pinyin.Args) []string {
		if unicode.IsLetter(v) || unicode.IsDigit(v) {
			return []string{string(v)}
		}
		return nil
	}
	return strings.ToUpper(strings.Join(pinyin.LazyPinyin(name, args), ""))
}
func (s *Store) Reindex(ctx context.Context) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, stockbiz.ErrPersistence
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(533,1)`); err != nil {
		return 0, persistence(ctx, err)
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,name FROM stock ORDER BY id FOR UPDATE`)
	if err != nil {
		return 0, persistence(ctx, err)
	}
	type entry struct{ id, name string }
	items := []entry{}
	for rows.Next() {
		var x entry
		if rows.Scan(&x.id, &x.name) != nil {
			rows.Close()
			return 0, stockbiz.ErrPersistence
		}
		items = append(items, x)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, persistence(ctx, err)
	}
	for _, x := range items {
		if _, err = tx.ExecContext(ctx, `UPDATE stock SET name_initials=$2 WHERE id=$1 AND name_initials IS DISTINCT FROM $2`, x.id, Initials(x.name)); err != nil {
			return 0, persistence(ctx, err)
		}
	}
	if err = tx.Commit(); err != nil {
		return 0, persistence(ctx, err)
	}
	return len(items), nil
}

func (s *Store) PublishProfiles(ctx context.Context, batch stockbiz.ProfileBatch) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return persistence(ctx, err)
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(533,1)`); err != nil {
		return persistence(ctx, err)
	}
	for _, p := range batch.Items {
		var old stockbiz.Stock
		if err = tx.QueryRowContext(ctx, `SELECT id,name,board,as_of FROM stock WHERE exchange=$1 AND code=$2 FOR UPDATE`, p.Exchange, p.Code).Scan(&old.ID, &old.Name, &old.Board, &old.AsOf); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return stockbiz.ErrConflict
			}
			return persistence(ctx, err)
		}
		next := stockbiz.Stock{ID: old.ID, Name: p.Name, Board: p.Board, AsOf: batch.AsOf}
		business, _ := json.Marshal(p.MainBusiness)
		// Same snapshot replay is a no-op. Changing already-published same-date
		// profile facts is rejected; initial nullable profile enrichment is allowed.
		var present, different bool
		err = tx.QueryRowContext(ctx, `SELECT
   (full_name IS NOT NULL OR former_name IS NOT NULL OR list_date IS NOT NULL OR established IS NOT NULL OR industry_l1 IS NOT NULL OR industry_l2 IS NOT NULL OR main_business IS NOT NULL OR main_product_type IS NOT NULL OR index_core IS NOT NULL OR concepts IS NOT NULL),
   ROW(full_name,former_name,list_date,established,industry_l1,industry_l2,main_business,main_product_type,index_core,concepts)
   IS DISTINCT FROM ROW($2::text,$3::text,$4::date,$5::date,$6::text,$7::text,NULLIF($8::jsonb,'null'::jsonb),$9::text[],$10::text[],$11::text[])
   FROM stock WHERE id=$1`, old.ID, p.FullName, p.FormerName, p.ListDate, p.Established, p.IndustryL1, p.IndustryL2, business, p.MainProductType, p.IndexCore, p.Concepts).Scan(&present, &different)
		if err != nil {
			return persistence(ctx, err)
		}
		write, err := stockbiz.ProfileReplacement(old, next, present, different)
		if err != nil {
			return err
		}
		if !write {
			continue
		}
		_, err = tx.ExecContext(ctx, `UPDATE stock SET name=$2,board=$3,as_of=$4,full_name=$5,former_name=$6,list_date=$7,established=$8,industry_l1=$9,industry_l2=$10,main_business=NULLIF($11::jsonb,'null'::jsonb),main_product_type=$12,index_core=$13,concepts=$14,name_initials=$15,updated_at=now() WHERE id=$1`,
			old.ID, p.Name, p.Board, batch.AsOf, p.FullName, p.FormerName, p.ListDate, p.Established, p.IndustryL1, p.IndustryL2, business, p.MainProductType, p.IndexCore, p.Concepts, Initials(p.Name))
		if err != nil {
			return persistence(ctx, err)
		}
	}
	if err = tx.Commit(); err != nil {
		return persistence(ctx, err)
	}
	return nil
}
