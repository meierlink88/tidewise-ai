package watchlist

import (
	"context"
	"database/sql"
	"errors"
	identity "github.com/meierlink88/tidewise-ai/user-service/backend/internal/biz/identity"
	biz "github.com/meierlink88/tidewise-ai/user-service/backend/internal/biz/watchlist"
	"time"
)

type transaction struct{ tx *sql.Tx }

func (r *Repository) Within(ctx context.Context, f func(biz.Transaction) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return identity.ErrUnavailable
	}
	defer tx.Rollback()
	if err = f(&transaction{tx}); err != nil {
		return err
	}
	if tx.Commit() != nil {
		return identity.ErrUnavailable
	}
	return nil
}
func (t *transaction) Session(ctx context.Context, hash []byte) (identity.Session, error) {
	var s identity.Session
	err := t.tx.QueryRowContext(ctx, `SELECT i.user_id,u.status,i.appid,s.expires_at,s.revoked_at IS NOT NULL
 FROM user_sessions s JOIN wechat_identities i ON i.id=s.wechat_identity_id JOIN users u ON u.id=i.user_id
 WHERE s.token_hash=$1 FOR UPDATE OF u,s`, hash).Scan(&s.UserID, &s.Status, &s.AppID, &s.ExpiresAt, &s.Revoked)
	if errors.Is(err, sql.ErrNoRows) {
		return s, identity.ErrUnauthenticated
	}
	if err != nil {
		return s, identity.ErrUnavailable
	}
	return s, nil
}
func (t *transaction) List(ctx context.Context, user string, after biz.Cursor, limit int) (biz.Page, error) {
	page := biz.Page{Items: []biz.Entry{}}
	if t.tx.QueryRowContext(ctx, `SELECT count(*) FROM user_watchlist WHERE user_id=$1`, user).Scan(&page.Total) != nil {
		return page, identity.ErrUnavailable
	}
	rows, err := t.tx.QueryContext(ctx, `SELECT stock_id,added_at FROM user_watchlist WHERE user_id=$1
 AND ($2='' OR added_at<$3 OR (added_at=$3 AND stock_id>$2))
 ORDER BY added_at DESC,stock_id ASC LIMIT $4`, user, after.StockID, after.AddedAt, limit)
	if err != nil {
		return page, identity.ErrUnavailable
	}
	defer rows.Close()
	for rows.Next() {
		var e biz.Entry
		if rows.Scan(&e.StockID, &e.AddedAt) != nil || !biz.ValidID(e.StockID) {
			return page, identity.ErrUnavailable
		}
		page.Items = append(page.Items, e)
	}
	if rows.Err() != nil {
		return page, identity.ErrUnavailable
	}
	return page, nil
}
func (t *transaction) Check(ctx context.Context, user string, ids []string) ([]string, error) {
	result := []string{}
	rows, err := t.tx.QueryContext(ctx, `SELECT stock_id FROM user_watchlist WHERE user_id=$1 AND stock_id=ANY($2) ORDER BY stock_id`, user, ids)
	if err != nil {
		return nil, identity.ErrUnavailable
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if rows.Scan(&id) != nil {
			return nil, identity.ErrUnavailable
		}
		result = append(result, id)
	}
	if rows.Err() != nil {
		return nil, identity.ErrUnavailable
	}
	return result, nil
}
func (t *transaction) Change(ctx context.Context, user, id string, add bool, now time.Time) error {
	var err error
	if add {
		_, err = t.tx.ExecContext(ctx, `INSERT INTO user_watchlist(user_id,stock_id,added_at) VALUES($1,$2,$3) ON CONFLICT(user_id,stock_id) DO NOTHING`, user, id, now)
	} else {
		_, err = t.tx.ExecContext(ctx, `DELETE FROM user_watchlist WHERE user_id=$1 AND stock_id=$2`, user, id)
	}
	if err != nil {
		return identity.ErrUnavailable
	}
	return nil
}
