package geopoliticrivalry

import (
	"context"
	"database/sql"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

// replaceDomainLinks reconciles an unordered set. Callers lock the parent row
// before replacing links; unchanged pairs keep their deterministic identities.
func replaceDomainLinks(ctx context.Context, tx *sql.Tx, id string, domains []string) (bool, error) {
	removed, err := tx.ExecContext(ctx, `DELETE FROM geopolitic_rivalry_domain_links WHERE geopolitic_rivalry_id=$1 AND NOT (geopolitic_domain_id=ANY($2::text[]))`, id, domains)
	if err != nil {
		return false, err
	}
	count, err := removed.RowsAffected()
	if err != nil {
		return false, err
	}
	changed := count > 0
	for _, domainID := range domains {
		linkID, err := coreid.Derive(coreid.GeopoliticRivalryDomainLink, "geopolitic_rivalry_domain_links", id, domainID)
		if err != nil {
			return false, err
		}
		inserted, err := tx.ExecContext(ctx, `INSERT INTO geopolitic_rivalry_domain_links (id, geopolitic_rivalry_id, geopolitic_domain_id) VALUES ($1,$2,$3) ON CONFLICT (geopolitic_rivalry_id, geopolitic_domain_id) DO NOTHING`, linkID, id, domainID)
		if err != nil {
			return false, err
		}
		count, err := inserted.RowsAffected()
		if err != nil {
			return false, err
		}
		changed = changed || count > 0
	}
	return changed, nil
}

// BackfillDomainLinks copies the legacy scalar relationship without rewriting
// any storyline facts. The maintenance caller owns the transaction and table locks.
func BackfillDomainLinks(ctx context.Context, tx *sql.Tx) (int, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id, geopolitic_domain_id FROM geopolitic_rivalries ORDER BY id`)
	if err != nil {
		return 0, classifyWriteError(err)
	}
	type pair struct{ story, domain string }
	pairs := make([]pair, 0)
	for rows.Next() {
		var item pair
		if err := rows.Scan(&item.story, &item.domain); err != nil {
			_ = rows.Close()
			return 0, classifyReadError(err)
		}
		if !coreid.Is(item.story, coreid.GeopoliticRivalry) || !coreid.Is(item.domain, coreid.GeopoliticDomain) {
			_ = rows.Close()
			return 0, ErrPersistence
		}
		pairs = append(pairs, item)
	}
	err = rows.Err()
	closeErr := rows.Close()
	if err != nil {
		return 0, classifyReadError(err)
	}
	if closeErr != nil {
		return 0, classifyReadError(closeErr)
	}
	for _, item := range pairs {
		linkID, err := coreid.Derive(coreid.GeopoliticRivalryDomainLink, "geopolitic_rivalry_domain_links", item.story, item.domain)
		if err != nil {
			return 0, ErrPersistence
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO geopolitic_rivalry_domain_links (id, geopolitic_rivalry_id, geopolitic_domain_id) VALUES ($1,$2,$3) ON CONFLICT (geopolitic_rivalry_id, geopolitic_domain_id) DO NOTHING`, linkID, item.story, item.domain); err != nil {
			return 0, classifyWriteError(err)
		}
	}
	var matched int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM geopolitic_rivalries s JOIN geopolitic_rivalry_domain_links l ON l.geopolitic_rivalry_id=s.id AND l.geopolitic_domain_id=s.geopolitic_domain_id`).Scan(&matched); err != nil {
		return 0, classifyReadError(err)
	}
	if matched != len(pairs) {
		return 0, ErrPersistence
	}
	return matched, nil
}
