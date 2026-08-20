package company

import (
	"context"

	companybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/company"
)

func (s *Store) ReplaceIndustries(ctx context.Context, id companybiz.ID, links []companybiz.IndustryLink) (companybiz.Company, error) {
	industryIDs := make([]string, len(links))
	linkIDs := make([]string, len(links))
	for index, link := range links {
		industryIDs[index] = string(link.IndustryID)
		linkIDs[index] = link.ID
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return companybiz.Company{}, classifyWriteError(err)
	}
	defer func() { _ = tx.Rollback() }()

	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT true FROM company WHERE id = $1 FOR UPDATE`, id).Scan(&exists); err != nil {
		return companybiz.Company{}, classifyWriteError(err)
	}
	var industryCount int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM industry WHERE id = ANY($1::text[])`, industryIDs).Scan(&industryCount); err != nil {
		return companybiz.Company{}, classifyWriteError(err)
	}
	if industryCount != len(industryIDs) {
		return companybiz.Company{}, &companybiz.ReferenceError{Field: "industry_ids", Message: "contains an unknown Industry"}
	}
	if _, err := tx.ExecContext(ctx, `
DELETE FROM company_industry_links
WHERE company_id = $1 AND NOT (industry_id = ANY($2::text[]))`, id, industryIDs); err != nil {
		return companybiz.Company{}, classifyWriteError(err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO company_industry_links (id, company_id, industry_id)
SELECT link_id, $1, industry_id
FROM unnest($2::text[], $3::text[]) AS link(link_id, industry_id)
ON CONFLICT (company_id, industry_id) DO NOTHING`, id, linkIDs, industryIDs); err != nil {
		return companybiz.Company{}, classifyWriteError(err)
	}
	result, err := getCompany(ctx, tx, id)
	if err != nil {
		return companybiz.Company{}, err
	}
	if err := tx.Commit(); err != nil {
		return companybiz.Company{}, classifyWriteError(err)
	}
	return result, nil
}

var _ companybiz.IndustryTransaction = (*Store)(nil)
