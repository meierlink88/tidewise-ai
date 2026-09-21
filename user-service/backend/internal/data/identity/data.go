package identity

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	biz "github.com/meierlink88/tidewise-ai/user-service/backend/internal/biz/identity"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository {
	if db == nil {
		panic("missing user database")
	}
	return &Repository{db}
}
func databaseError(err error) error {
	if err == nil {
		return nil
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return biz.ErrConflict
	}
	return biz.ErrUnavailable
}
func validID(s string) bool { _, err := uuid.Parse(s); return err == nil }
func validNickname(s string) bool {
	return utf8.ValidString(s) && utf8.RuneCountInString(s) <= 32 && strings.TrimSpace(s) == s
}
func validText(s string) bool {
	return len(s) > 0 && len(s) <= 256 && strings.TrimSpace(s) == s && !strings.ContainsAny(s, "\x00\r\n")
}
func (r *Repository) Lookup(ctx context.Context, hash []byte) (biz.Session, error) {
	var s biz.Session
	var revoked sql.NullTime
	err := r.db.QueryRowContext(ctx, `SELECT s.id,s.wechat_identity_id,i.user_id,u.status,i.appid,s.created_at,s.expires_at,s.revoked_at,u.nickname FROM user_sessions s JOIN wechat_identities i ON i.id=s.wechat_identity_id JOIN users u ON u.id=i.user_id WHERE s.token_hash=$1`, hash).Scan(&s.ID, &s.IdentityID, &s.UserID, &s.Status, &s.AppID, &s.CreatedAt, &s.ExpiresAt, &revoked, &s.Nickname)
	if errors.Is(err, sql.ErrNoRows) {
		return s, nil
	}
	if err != nil {
		return s, databaseError(err)
	}
	if !validNickname(s.Nickname) || !validID(s.ID) || !validID(s.IdentityID) || !validID(s.UserID) || !validText(s.AppID) || (s.Status != "active" && s.Status != "disabled") || !s.ExpiresAt.After(s.CreatedAt) {
		return biz.Session{}, biz.ErrUnavailable
	}
	s.Revoked = revoked.Valid
	return s, nil
}
func (r *Repository) Revoke(ctx context.Context, hash []byte, appid string, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE user_sessions s SET revoked_at=GREATEST($3,s.created_at) FROM wechat_identities i WHERE s.wechat_identity_id=i.id AND i.appid=$2 AND s.token_hash=$1 AND s.revoked_at IS NULL`, hash, appid, now)
	return databaseError(err)
}

type Wechat struct {
	client        *http.Client
	appID, secret string
}

func NewWechat(appid, secret string, transport http.RoundTripper) *Wechat {
	if appid == "" || secret == "" {
		panic("missing WeChat credentials")
	}
	if transport == nil {
		direct := http.DefaultTransport.(*http.Transport).Clone()
		direct.DisableKeepAlives = true
		// HTTP/2 may replay a GET after GOAWAY; code exchange must not replay.
		direct.ForceAttemptHTTP2 = false
		direct.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
		transport = direct
	}
	return &Wechat{client: &http.Client{Timeout: 5 * time.Second, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, appID: appid, secret: secret}
}
func (w *Wechat) Exchange(ctx context.Context, code string) (biz.Wechat, error) {
	endpoint := "https://api.weixin.qq.com/sns/jscode2session?" + url.Values{"appid": {w.appID}, "secret": {w.secret}, "js_code": {code}, "grant_type": {"authorization_code"}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return biz.Wechat{}, biz.ErrProvider
	}
	// A fresh connection prevents Transport from replaying this one-use code GET.
	req.Close = true
	resp, err := w.client.Do(req)
	if err != nil {
		return biz.Wechat{}, biz.ErrProvider
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return biz.Wechat{}, biz.ErrProvider
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16385))
	if err != nil || len(body) > 16384 {
		return biz.Wechat{}, biz.ErrProvider
	}
	var result struct {
		OpenID     string `json:"openid"`
		UnionID    string `json:"unionid"`
		SessionKey string `json:"session_key"`
		Code       int    `json:"errcode"`
	}
	if json.Unmarshal(body, &result) != nil {
		return biz.Wechat{}, biz.ErrProvider
	}
	switch result.Code {
	case 0:
	case 40029:
		return biz.Wechat{}, biz.ErrCodeInvalid
	case 40226:
		return biz.Wechat{}, biz.ErrRejected
	case 45011:
		return biz.Wechat{}, biz.ErrRateLimited
	default:
		return biz.Wechat{}, biz.ErrProvider
	}
	if !validText(result.OpenID) || !validText(result.SessionKey) || (result.UnionID != "" && !validText(result.UnionID)) {
		return biz.Wechat{}, biz.ErrProvider
	}
	return biz.Wechat{OpenID: result.OpenID, UnionID: result.UnionID}, nil
}
