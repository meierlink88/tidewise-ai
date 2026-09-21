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
	err := t.tx.QueryRowContext(ctx, `SELECT u.id,i.id,u.status,i.appid,i.openid,COALESCE(i.unionid,''),u.nickname FROM wechat_identities i JOIN users u ON u.id=i.user_id WHERE i.appid=$1 AND i.openid=$2 FOR UPDATE OF u,i`, appid, openid).Scan(&s.UserID, &s.IdentityID, &s.Status, &s.AppID, &s.OpenID, &s.UnionID, &s.Nickname)
	if errors.Is(err, sql.ErrNoRows) {
		return s, nil
	}
	if err != nil {
		return s, databaseError(err)
	}
	if !validNickname(s.Nickname) || !validID(s.UserID) || !validID(s.IdentityID) || (s.Status != "active" && s.Status != "disabled") || !validText(s.AppID) || !validText(s.OpenID) || (s.UnionID != "" && !validText(s.UnionID)) {
		return biz.State{}, biz.ErrUnavailable
	}
	return s, nil
}
func (t *transaction) Create(ctx context.Context, s biz.State, now time.Time) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO users(id,status,created_at,updated_at,last_login_at,nickname) VALUES($1,$2,$3,$3,$3,$4)`, s.UserID, s.Status, now, s.Nickname)
	if err != nil {
		return databaseError(err)
	}
	_, err = t.tx.ExecContext(ctx, `INSERT INTO wechat_identities(id,user_id,appid,openid,unionid,created_at,last_login_at) VALUES($1,$2,$3,$4,NULLIF($5,''),$6,$6)`, s.IdentityID, s.UserID, s.AppID, s.OpenID, s.UnionID, now)
	return databaseError(err)
}
func (t *transaction) Save(ctx context.Context, s biz.Session, unionid string, now time.Time, previous []byte) error {
	_, err := t.tx.ExecContext(ctx, `UPDATE users SET updated_at=$2,last_login_at=$2,phone_number=COALESCE(NULLIF($3,''),phone_number),phone_verified_at=CASE WHEN $3<>'' THEN $2 ELSE phone_verified_at END WHERE id=$1`, s.UserID, now, s.Phone)
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
	_, err = t.tx.ExecContext(ctx, `INSERT INTO user_sessions(id,wechat_identity_id,token_hash,created_at,expires_at,privacy_version,privacy_accepted_at) VALUES($1,$2,$3,$4,$5,NULLIF($6,''),CASE WHEN $6<>'' THEN $4::timestamptz ELSE NULL END)`, s.ID, s.IdentityID, s.Hash, s.CreatedAt, s.ExpiresAt, s.PrivacyVersion)
	return databaseError(err)
}

func (t *transaction) LookupSession(ctx context.Context, hash []byte) (biz.Session, error) {
	return scanSession(t.tx.QueryRowContext(ctx, `SELECT s.id,s.wechat_identity_id,i.user_id,u.status,i.appid,s.created_at,s.expires_at,s.revoked_at,u.nickname FROM user_sessions s JOIN wechat_identities i ON i.id=s.wechat_identity_id JOIN users u ON u.id=i.user_id WHERE s.token_hash=$1 FOR UPDATE OF u,s`, hash))
}
func (t *transaction) SetNickname(ctx context.Context, userID, nickname string, now time.Time) error {
	_, err := t.tx.ExecContext(ctx, `UPDATE users SET nickname=$2,updated_at=$3 WHERE id=$1`, userID, nickname, now)
	return databaseError(err)
}
