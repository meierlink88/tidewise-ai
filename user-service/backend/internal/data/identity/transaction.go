package identity

import (
	"context"
	"database/sql"
	"errors"
	"time"

	biz "github.com/meierlink88/tidewise-ai/user-service/backend/internal/biz/identity"
)

type transaction struct{ tx *sql.Tx }

func (r *Repository) Within(ctx context.Context, fn func(biz.Transaction) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return databaseError(err)
	}
	defer tx.Rollback()
	if err = fn(&transaction{tx}); err != nil {
		return err
	}
	return databaseError(tx.Commit())
}
func (t *transaction) Find(ctx context.Context, appid, openid string) (biz.State, error) {
	var s biz.State
	err := t.tx.QueryRowContext(ctx, `SELECT u.id,i.id,u.status,i.appid,i.openid,COALESCE(i.unionid,'') FROM wechat_identities i JOIN users u ON u.id=i.user_id WHERE i.appid=$1 AND i.openid=$2 FOR UPDATE OF u,i`, appid, openid).Scan(&s.UserID, &s.IdentityID, &s.Status, &s.AppID, &s.OpenID, &s.UnionID)
	if errors.Is(err, sql.ErrNoRows) {
		return s, nil
	}
	if err != nil {
		return s, databaseError(err)
	}
	if !validID(s.UserID) || !validID(s.IdentityID) || (s.Status != "active" && s.Status != "disabled") || !validText(s.AppID) || !validText(s.OpenID) || (s.UnionID != "" && !validText(s.UnionID)) {
		return biz.State{}, biz.ErrUnavailable
	}
	return s, nil
}
func (t *transaction) Create(ctx context.Context, s biz.State, now time.Time) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO users(id,status,created_at,updated_at,last_login_at) VALUES($1,$2,$3,$3,$3)`, s.UserID, s.Status, now)
	if err != nil {
		return databaseError(err)
	}
	_, err = t.tx.ExecContext(ctx, `INSERT INTO wechat_identities(id,user_id,appid,openid,unionid,created_at,last_login_at) VALUES($1,$2,$3,$4,NULLIF($5,''),$6,$6)`, s.IdentityID, s.UserID, s.AppID, s.OpenID, s.UnionID, now)
	return databaseError(err)
}
func (t *transaction) Save(ctx context.Context, s biz.Session, unionid string, now time.Time, previous []byte) error {
	_, err := t.tx.ExecContext(ctx, `UPDATE users SET updated_at=$2,last_login_at=$2 WHERE id=$1`, s.UserID, now)
	if err != nil {
		return databaseError(err)
	}
	_, err = t.tx.ExecContext(ctx, `UPDATE wechat_identities SET last_login_at=$2,unionid=COALESCE(unionid,NULLIF($3,'')) WHERE id=$1`, s.IdentityID, now, unionid)
	if err != nil {
		return databaseError(err)
	}
	if len(previous) > 0 {
		_, err = t.tx.ExecContext(ctx, `UPDATE user_sessions SET revoked_at=GREATEST($3,created_at) WHERE token_hash=$1 AND wechat_identity_id=$2 AND revoked_at IS NULL`, previous, s.IdentityID, now)
		if err != nil {
			return databaseError(err)
		}
	}
	_, err = t.tx.ExecContext(ctx, `INSERT INTO user_sessions(id,wechat_identity_id,token_hash,created_at,expires_at) VALUES($1,$2,$3,$4,$5)`, s.ID, s.IdentityID, s.Hash, s.CreatedAt, s.ExpiresAt)
	return databaseError(err)
}
