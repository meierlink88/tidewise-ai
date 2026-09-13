package data

import (
	"context"
	"database/sql"
	"errors"

	geo "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/geopoliticrivalry"
	macro "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/macroeconomic"
)

// StorylineDomainBackfillResult counts legacy pairs verified against their new tables.
type StorylineDomainBackfillResult struct {
	Geopolitical  int
	Macroeconomic int
}

// BackfillStorylineDomains owns the atomic cross-domain maintenance transaction.
// apply=false verifies exactly the same writes and then rolls them back.
func BackfillStorylineDomains(ctx context.Context, db *sql.DB, apply bool) (StorylineDomainBackfillResult, error) {
	if db == nil {
		return StorylineDomainBackfillResult{}, storylineBackfillFailure(ctx, "membership backfill database is required")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return StorylineDomainBackfillResult{}, storylineBackfillFailure(ctx, "begin membership backfill failed")
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `SET LOCAL lock_timeout = '5s'`); err != nil {
		return StorylineDomainBackfillResult{}, storylineBackfillFailure(ctx, "configure backfill lock timeout failed")
	}
	if _, err := tx.ExecContext(ctx, `LOCK TABLE goose_db_version, geopolitic_rivalries, macro_economics, geopolitic_rivalry_domain_links, macro_economic_domain_links IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return StorylineDomainBackfillResult{}, storylineBackfillFailure(ctx, "lock membership backfill failed")
	}
	var version int
	if err := tx.QueryRowContext(ctx, `SELECT max(version_id) FROM goose_db_version WHERE is_applied`).Scan(&version); err != nil || version != 92 {
		return StorylineDomainBackfillResult{}, storylineBackfillFailure(ctx, "membership backfill requires schema version 92")
	}
	geopolitical, err := geo.BackfillDomainLinks(ctx, tx)
	if err != nil {
		return StorylineDomainBackfillResult{}, err
	}
	macroeconomic, err := macro.BackfillDomainLinks(ctx, tx)
	if err != nil {
		return StorylineDomainBackfillResult{}, err
	}
	if apply {
		if err := tx.Commit(); err != nil {
			return StorylineDomainBackfillResult{}, storylineBackfillFailure(ctx, "commit membership backfill failed")
		}
	} else if err := tx.Rollback(); err != nil {
		return StorylineDomainBackfillResult{}, storylineBackfillFailure(ctx, "rollback membership verification failed")
	}
	return StorylineDomainBackfillResult{Geopolitical: geopolitical, Macroeconomic: macroeconomic}, nil
}

func storylineBackfillFailure(ctx context.Context, message string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return errors.New(message)
}
