package stock

import (
	"context"
	"encoding/json"
	"fmt"

	biz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/stock"
)

func (s *Store) ResolveClassificationStocks(ctx context.Context, symbols []string) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT code||'.'||exchange,id FROM stock WHERE code||'.'||exchange=ANY($1)`, symbols)
	if err != nil {
		return nil, fmt.Errorf("resolve classification stocks: %w", biz.ErrPersistence)
	}
	defer rows.Close()
	result := map[string]string{}
	for rows.Next() {
		var symbol, id string
		if err = rows.Scan(&symbol, &id); err != nil {
			return nil, biz.ErrPersistence
		}
		result[symbol] = id
	}
	if err = rows.Err(); err != nil {
		return nil, biz.ErrPersistence
	}
	return result, nil
}

func (s *Store) PublishClassifications(ctx context.Context, p biz.ClassificationPublication, apply bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return biz.ErrPersistence
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('stock-classification-publish',0))`); err != nil {
		return biz.ErrPersistence
	}
	// The publication's reference set and all six writable tables remain stable
	// for the whole transaction, including replay and removal of stale links.
	if _, err = tx.ExecContext(ctx, `LOCK TABLE stock IN SHARE MODE; LOCK TABLE s_industry,s_concept,s_industry_chain,stock_s_industry_links,stock_s_concept_links,stock_s_industry_chain_links IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return fmt.Errorf("lock classification tables: %w", biz.ErrPersistence)
	}
	var count int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM stock WHERE id=ANY($1)`, p.StockIDs).Scan(&count); err != nil {
		return biz.ErrPersistence
	}
	if count != len(p.StockIDs) {
		return biz.ErrInvalid
	}
	// Verify natural identities before any mutation; do not silently remap an
	// existing catalog ID and invalidate relationships owned by other callers.
	for _, spec := range []struct {
		table string
		value any
	}{
		{"s_concept", p.Concepts}, {"s_industry_chain", p.Chains},
	} {
		payload, e := json.Marshal(spec.value)
		if e != nil {
			return biz.ErrInvalid
		}
		if string(payload) == "null" {
			payload = []byte("[]")
		}
		var conflict bool
		err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM jsonb_to_recordset($1::jsonb) AS x(id text,name text) JOIN `+spec.table+` t ON t.id=x.id OR t.name=x.name WHERE t.id<>x.id OR t.name<>x.name)`, string(payload)).Scan(&conflict)
		if err != nil {
			return biz.ErrPersistence
		}
		if conflict {
			return biz.ErrConflict
		}
	}
	payload, err := json.Marshal(p.Industries)
	if err != nil {
		return biz.ErrInvalid
	}
	var conflict bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM jsonb_to_recordset($1::jsonb) AS x(id text,name text,code text,parent text,path text[]) JOIN s_industry t ON t.id=x.id OR (t.classification_system='wind' AND t.industry_code=x.code) WHERE t.id<>x.id OR t.name<>x.name OR t.classification_system<>'wind' OR t.industry_code<>x.code OR t.parent_industry_id IS DISTINCT FROM x.parent OR t.hierarchy_path_codes<>x.path)`, string(payload)).Scan(&conflict)
	if err != nil {
		return biz.ErrPersistence
	}
	if conflict {
		return biz.ErrConflict
	}
	if !apply {
		return nil
	}
	// Parents are first. Existing reviewed definitions/aliases remain intact when
	// the source only supplies a matching name; replay performs no UPDATE.
	for _, item := range p.Industries {
		_, err = tx.ExecContext(ctx, `INSERT INTO s_industry(id,name,aliases,classification_system,industry_code,parent_industry_id,hierarchy_path_codes,definition,review_status) VALUES($1,$2,'{}','wind',$3,$4,$5,NULL,'candidate') ON CONFLICT(id) DO NOTHING`, item.ID, item.Name, item.Code, item.Parent, item.Path)
		if err != nil {
			return fmt.Errorf("publish stock industry: %w", biz.ErrPersistence)
		}
	}
	for _, spec := range []struct {
		sql   string
		value any
	}{
		{`INSERT INTO s_concept(id,name,aliases,concept_type,definition,review_status) SELECT id,name,'{}','market_theme',NULL,'candidate' FROM jsonb_to_recordset($1::jsonb) AS x(id text,name text) ON CONFLICT(id) DO NOTHING`, p.Concepts},
		{`INSERT INTO s_industry_chain(id,name,aliases,scope,target_output,end_use,geography,as_of_date,review_status,observable_variables) SELECT id,name,'{}',NULL,NULL,NULL,NULL,NULL,'candidate',NULL FROM jsonb_to_recordset($1::jsonb) AS x(id text,name text) ON CONFLICT(id) DO NOTHING`, p.Chains},
	} {
		raw, e := json.Marshal(spec.value)
		if e != nil {
			return biz.ErrInvalid
		}
		if string(raw) == "null" {
			raw = []byte("[]")
		}
		if _, err = tx.ExecContext(ctx, spec.sql, string(raw)); err != nil {
			return fmt.Errorf("publish stock master: %w", biz.ErrPersistence)
		}
	}
	for _, spec := range []struct {
		table, column string
		links         []biz.ClassificationLink
	}{
		{"stock_s_industry_links", "industry_id", p.IndustryLinks},
		{"stock_s_concept_links", "concept_id", p.ConceptLinks},
		{"stock_s_industry_chain_links", "industry_chain_id", p.ChainLinks},
	} {
		raw, e := json.Marshal(spec.links)
		if e != nil {
			return biz.ErrInvalid
		}
		if string(raw) == "null" {
			raw = []byte("[]")
		}
		if _, err = tx.ExecContext(ctx, `CREATE TEMP TABLE classification_link_stage (id text,stock_id text,target_id text,PRIMARY KEY(stock_id,target_id),UNIQUE(id)) ON COMMIT DROP`); err != nil {
			return biz.ErrPersistence
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO classification_link_stage SELECT * FROM jsonb_to_recordset($1::jsonb) AS x(id text,stock_id text,target_id text)`, string(raw)); err != nil {
			return biz.ErrInvalid
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM `+spec.table+` t WHERE t.stock_id=ANY($1) AND NOT EXISTS(SELECT 1 FROM classification_link_stage x WHERE x.stock_id=t.stock_id AND x.target_id=t.`+spec.column+`)`, p.StockIDs); err != nil {
			return biz.ErrPersistence
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO `+spec.table+`(id,stock_id,`+spec.column+`) SELECT id,stock_id,target_id FROM classification_link_stage ON CONFLICT(stock_id,`+spec.column+`) DO NOTHING`); err != nil {
			return fmt.Errorf("publish stock links: %w", biz.ErrPersistence)
		}
		var difference bool
		err = tx.QueryRowContext(ctx, `SELECT EXISTS((SELECT stock_id,target_id FROM classification_link_stage EXCEPT SELECT stock_id,`+spec.column+` FROM `+spec.table+` WHERE stock_id=ANY($1)) UNION ALL (SELECT stock_id,`+spec.column+` FROM `+spec.table+` WHERE stock_id=ANY($1) EXCEPT SELECT stock_id,target_id FROM classification_link_stage))`, p.StockIDs).Scan(&difference)
		if err != nil {
			return biz.ErrPersistence
		}
		if difference {
			return biz.ErrConflict
		}
		if _, err = tx.ExecContext(ctx, `DROP TABLE classification_link_stage`); err != nil {
			return biz.ErrPersistence
		}
	}
	if err = tx.Commit(); err != nil {
		return biz.ErrPersistence
	}
	return nil
}
