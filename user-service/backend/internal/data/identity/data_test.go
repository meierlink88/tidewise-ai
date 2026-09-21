package identity

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	biz "github.com/meierlink88/tidewise-ai/user-service/backend/internal/biz/identity"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestWechatExchangeBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		want       error
	}{
		{"valid", `{"openid":"open","session_key":"private","unionid":"union"}`, 200, nil},
		{"missing identity", `{"session_key":"private"}`, 200, biz.ErrProvider},
		{"invalid code", `{"errcode":40029,"errmsg":"secret internal message"}`, 200, biz.ErrCodeInvalid},
		{"risk", `{"errcode":40226}`, 200, biz.ErrRejected},
		{"rate", `{"errcode":45011}`, 200, biz.ErrRateLimited},
		{"upstream", `{"errcode":-1}`, 200, biz.ErrProvider},
		{"html", `secret html`, 200, biz.ErrProvider},
		{"redirect", ``, 302, biz.ErrProvider},
		{"oversize", strings.Repeat("x", 16385), 200, biz.ErrProvider},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := NewWechat("app", "secret", roundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Scheme != "https" || r.URL.Host != "api.weixin.qq.com" || r.URL.Path != "/sns/jscode2session" || r.URL.Query().Get("js_code") != "code" {
					t.Fatal("incorrect provider request")
				}
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
			}))
			_, err := client.Exchange(context.Background(), "code")
			if !errors.Is(err, tc.want) || calls != 1 {
				t.Fatalf("err=%v calls=%d", err, calls)
			}
		})
	}
}
func TestWechatTransportErrorSanitized(t *testing.T) {
	client := NewWechat("app", "secret", roundTrip(func(r *http.Request) (*http.Response, error) { return nil, errors.New("secret code URL") }))
	_, err := client.Exchange(context.Background(), "code")
	if err != biz.ErrProvider || strings.Contains(err.Error(), "secret") {
		t.Fatal(err)
	}
}

func TestPhoneProviderBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       error
	}{
		{"valid", `{"phone_info":{"purePhoneNumber":"13800000000","countryCode":"86","watermark":{"appid":"app"}}}`, nil},
		{"wrong app", `{"phone_info":{"purePhoneNumber":"13800000000","countryCode":"86","watermark":{"appid":"other"}}}`, biz.ErrProvider},
		{"invalid", `{"errcode":40029}`, biz.ErrCodeInvalid},
		{"malformed", `{"phone_info":{"purePhoneNumber":"bad","countryCode":"86","watermark":{"appid":"app"}}}`, biz.ErrProvider},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			w := NewWechat("app", "secret", roundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				body := tc.body
				if r.Method != "POST" {
					t.Fatal("unexpected method")
				}
				if r.URL.Path == "/cgi-bin/stable_token" {
					body = `{"access_token":"server-token","expires_in":7200}`
				} else if r.URL.Path != "/wxa/business/getuserphonenumber" || r.URL.Query().Get("access_token") != "server-token" {
					t.Fatal("wrong endpoint")
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
			}))
			phone, err := w.Phone(context.Background(), "phone-code")
			if !errors.Is(err, tc.want) || calls != 2 {
				t.Fatalf("err %v calls %d", err, calls)
			}
			if err == nil && phone != "+8613800000000" {
				t.Fatal("invalid normalization")
			}
		})
	}
}

func TestPhoneAndConsentPersistence(t *testing.T) {
	dsn := os.Getenv("USER_DATABASE_URL")
	if dsn == "" || os.Getenv("USER_DATABASE_NAME") != "tidewise_user_test" {
		t.Skip("requires isolated migrated tidewise_user_test")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var actualDatabase string
	if err = db.QueryRow("SELECT current_database()").Scan(&actualDatabase); err != nil || actualDatabase != "tidewise_user_test" {
		t.Fatal("requires disposable test database")
	}
	// Serialize destructive test fixtures across Go packages sharing the CI database.
	lock, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if _, err = lock.ExecContext(context.Background(), "SELECT pg_advisory_lock(20260921528)"); err != nil {
		t.Fatal(err)
	}
	defer lock.ExecContext(context.Background(), "SELECT pg_advisory_unlock(20260921528)")

	repo := NewRepository(db)
	now := time.Now().UTC()
	userID, identityID, sessionID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	defer func() {
		for _, q := range []string{`DELETE FROM user_sessions WHERE wechat_identity_id IN (SELECT id FROM wechat_identities WHERE user_id=$1)`, `DELETE FROM wechat_identities WHERE user_id=$1`, `DELETE FROM users WHERE id=$1`} {
			if _, e := db.Exec(q, userID); e != nil {
				t.Error("test cleanup failed", e)
			}
		}
	}()
	state := biz.State{UserID: userID, IdentityID: identityID, Status: "active", AppID: "app", OpenID: uuid.NewString()}
	session := biz.Session{ID: sessionID, IdentityID: identityID, UserID: userID, Hash: make([]byte, 32), CreatedAt: now, ExpiresAt: now.Add(time.Hour), Phone: "+8613800000000", PrivacyVersion: biz.PrivacyVersion}
	_, err = rand.Read(session.Hash)
	if err != nil {
		t.Fatal(err)
	}
	err = repo.Within(context.Background(), func(tx biz.Transaction) error {
		if e := tx.Create(context.Background(), state, now); e != nil {
			return e
		}
		return tx.Save(context.Background(), session, "", now, nil)
	})
	if err != nil {
		t.Fatal(err)
	}
	var phone, version string
	var accepted time.Time
	if err = db.QueryRow(`SELECT u.phone_number,s.privacy_version,s.privacy_accepted_at FROM users u JOIN wechat_identities i ON i.user_id=u.id JOIN user_sessions s ON s.wechat_identity_id=i.id WHERE s.id=$1`, sessionID).Scan(&phone, &version, &accepted); err != nil {
		t.Fatal(err)
	}
	if phone != session.Phone || version != biz.PrivacyVersion || accepted.IsZero() {
		t.Fatal("phone or consent not persisted")
	}
	// A duplicate session must roll back the preceding phone update.
	session.Phone = "+8613900000000"
	err = repo.Within(context.Background(), func(tx biz.Transaction) error { return tx.Save(context.Background(), session, "", now, nil) })
	if !errors.Is(err, biz.ErrConflict) {
		t.Fatal("expected duplicate session conflict")
	}
	if err = db.QueryRow(`SELECT phone_number FROM users WHERE id=$1`, userID).Scan(&phone); err != nil || phone != "+8613800000000" {
		t.Fatal("phone update escaped failed transaction")
	}
}
