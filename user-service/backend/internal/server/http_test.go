package server_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	_ "github.com/jackc/pgx/v5/stdlib"
	api "github.com/meierlink88/tidewise-ai/user-service/backend/api/user/v1/identity"
	biz "github.com/meierlink88/tidewise-ai/user-service/backend/internal/biz/identity"
	data "github.com/meierlink88/tidewise-ai/user-service/backend/internal/data"
	adapter "github.com/meierlink88/tidewise-ai/user-service/backend/internal/data/identity"
	"github.com/meierlink88/tidewise-ai/user-service/backend/internal/server"
	service "github.com/meierlink88/tidewise-ai/user-service/backend/internal/service/identity"
)

const serviceToken = "test-service-token-01234567890123456789"

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type envelope struct {
	RequestID string           `json:"request_id"`
	Result    api.UserResponse `json:"result"`
	Error     struct {
		Code string `json:"code"`
	} `json:"error"`
}

func request(t *testing.T, handler http.Handler, path, body, authorization string) (int, envelope) {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", authorization)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	var result envelope
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid response: %s", w.Body.String())
	}
	if result.RequestID == "" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("missing safety headers/envelope")
	}
	return w.Code, result
}

type stub struct{}

func (stub) Login(context.Context, api.LoginRequest) (api.UserResponse, error) {
	return api.UserResponse{}, nil
}
func (stub) Verify(context.Context, api.SessionRequest) (api.UserResponse, error) {
	return api.UserResponse{}, nil
}
func (stub) Revoke(context.Context, api.SessionRequest) (api.RevokeResponse, error) {
	return api.RevokeResponse{Revoked: true}, nil
}
func TestHTTPBindingAndServiceAuthentication(t *testing.T) {
	h := server.New(":0", serviceToken, stub{}, func(context.Context) error { return nil }, slog.New(slog.NewTextHandler(io.Discard, nil))).Server.Handler
	for _, tc := range []struct {
		body, auth string
		status     int
	}{
		{`{"code":"c"}`, "", 401}, {`{"code":"c","openid":"injected"}`, "Bearer " + serviceToken, 400}, {`null`, "Bearer " + serviceToken, 400},
		{`{"code":"a","code":"b"}`, "Bearer " + serviceToken, 400}, {`{"code":"c"} {}`, "Bearer " + serviceToken, 400}, {strings.Repeat("x", 4097), "Bearer " + serviceToken, 400},
	} {
		code, _ := request(t, h, "/api/user/v1/wechat/logins", tc.body, tc.auth)
		if code != tc.status {
			t.Fatalf("status %d want %d", code, tc.status)
		}
	}
}
func TestOpenAPIValid(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("../../api/user/v1/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err = doc.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/user/v1/wechat/logins", "/api/user/v1/sessions/verify", "/api/user/v1/sessions/revoke"} {
		if doc.Paths.Find(path) == nil || doc.Paths.Find(path).Post == nil {
			t.Fatal(path)
		}
	}
}

// A dedicated disposable database, already migrated by the production dbmigrate
// binary, is required. Never point this test at a developer or UAT database.
func TestPostgresLoginLifecycleAndConcurrency(t *testing.T) {
	dsn := os.Getenv("USER_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set USER_TEST_DATABASE_URL for disposable PostgreSQL seam")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var name string
	if err = db.QueryRow("SELECT current_database()").Scan(&name); err != nil || name != "tidewise_user_test" {
		t.Fatal("requires isolated tidewise_user_test database")
	}
	if err = data.Ready(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("TRUNCATE user_sessions,wechat_identities,users"); err != nil {
		t.Fatal(err)
	}
	provider := adapter.NewWechat("app", "secret", transport(func(r *http.Request) (*http.Response, error) {
		body, _ := json.Marshal(map[string]string{"openid": r.URL.Query().Get("js_code"), "session_key": "private"})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body))), Header: make(http.Header)}, nil
	}))
	repository := adapter.NewRepository(db)
	usecase := biz.New(repository, provider, "app", time.Hour, time.Now)
	h := server.New(":0", serviceToken, service.New(usecase), func(ctx context.Context) error { return data.Ready(ctx, db) }, slog.New(slog.NewTextHandler(io.Discard, nil))).Server.Handler
	login := func(code, previous string) api.UserResponse {
		t.Helper()
		body, _ := json.Marshal(api.LoginRequest{Code: code, PreviousSessionToken: previous})
		status, r := request(t, h, "/api/user/v1/wechat/logins", string(body), "Bearer "+serviceToken)
		if status != 200 {
			t.Fatalf("login %d %s", status, r.Error.Code)
		}
		return r.Result
	}
	sessionBody := func(token string) string {
		b, _ := json.Marshal(api.SessionRequest{SessionToken: token})
		return string(b)
	}
	first := login("alice", "")
	if len(first.SessionToken) != 43 {
		t.Fatal("invalid token")
	}
	code, r := request(t, h, "/api/user/v1/sessions/verify", sessionBody(first.SessionToken), "Bearer "+serviceToken)
	if code != 200 || r.Result.UserID != first.UserID || r.Result.SessionToken != "" {
		t.Fatal("verify mismatch")
	}
	if first.Nickname != "" {
		t.Fatal("new user nickname must be unset")
	}
	nicknameBody, _ := json.Marshal(api.NicknameRequest{SessionToken: first.SessionToken, Nickname: "  观潮用户  "})
	if status, result := request(t, h, "/api/user/v1/profiles/nickname", string(nicknameBody), "Bearer "+serviceToken); status != 200 || result.Result.Nickname != "观潮用户" || result.Result.SessionToken != "" {
		t.Fatal("nickname HTTP update failed", status)
	}
	status, profile := request(t, h, "/api/user/v1/sessions/verify", sessionBody(first.SessionToken), "Bearer "+serviceToken)
	if status != 200 || profile.Result.Nickname != "观潮用户" {
		t.Fatal("verify lost nickname")
	}
	second := login("alice", first.SessionToken)
	if second.Nickname != "观潮用户" {
		t.Fatal("login overwrote nickname")
	}
	if _, err = db.Exec("UPDATE users SET nickname=$2 WHERE id=$1", first.UserID, strings.Repeat("潮", 33)); err == nil {
		t.Fatal("nickname length constraint missing")
	}

	if second.UserID != first.UserID {
		t.Fatal("duplicate user")
	}
	code, _ = request(t, h, "/api/user/v1/sessions/verify", sessionBody(first.SessionToken), "Bearer "+serviceToken)
	if code != 401 {
		t.Fatal("old session not revoked")
	}
	if status, _ := request(t, h, "/api/user/v1/profiles/nickname", string(nicknameBody), "Bearer "+serviceToken); status != 401 {
		t.Fatal("revoked session changed nickname")
	}
	// A failed new-session insert must roll back revocation of the previous one.
	previousHash := sha256.Sum256([]byte(second.SessionToken))
	txErr := repository.Within(context.Background(), func(tx biz.Transaction) error {
		state, e := tx.Find(context.Background(), "app", "alice")
		if e != nil {
			return e
		}
		now := time.Now()
		return tx.Save(context.Background(), biz.Session{ID: "invalid-uuid", IdentityID: state.IdentityID, UserID: state.UserID, Hash: make([]byte, 32), CreatedAt: now, ExpiresAt: now.Add(time.Hour)}, "", now, previousHash[:])
	})
	if txErr == nil {
		t.Fatal("expected insert failure")
	}
	if _, err = usecase.Verify(context.Background(), second.SessionToken); err != nil {
		t.Fatal("failed login revoked existing session", err)
	}
	// Rollback also removes an uncommitted user/identity pair.
	txErr = repository.Within(context.Background(), func(tx biz.Transaction) error {
		err := tx.Create(context.Background(), biz.State{UserID: "10000000-0000-4000-8000-000000000001", IdentityID: "10000000-0000-4000-8000-000000000002", Status: "active", AppID: "app", OpenID: "rolled-back"}, time.Now())
		if err != nil {
			return err
		}
		return biz.ErrRejected
	})
	if !errors.Is(txErr, biz.ErrRejected) {
		t.Fatal(txErr)
	}
	bob := login("bob", second.SessionToken)
	code, _ = request(t, h, "/api/user/v1/sessions/verify", sessionBody(second.SessionToken), "Bearer "+serviceToken)
	if code != 200 {
		t.Fatal("cross identity revoked")
	}
	for i := 0; i < 2; i++ {
		code, _ = request(t, h, "/api/user/v1/sessions/revoke", sessionBody(bob.SessionToken), "Bearer "+serviceToken)
		if code != 200 {
			t.Fatal("idempotent revoke")
		}
	}
	if _, err = db.Exec("UPDATE users SET status='disabled' WHERE id=$1", first.UserID); err != nil {
		t.Fatal(err)
	}
	code, _ = request(t, h, "/api/user/v1/sessions/verify", sessionBody(second.SessionToken), "Bearer "+serviceToken)
	if code != 403 {
		t.Fatal("disabled user verified")
	}
	code, _ = request(t, h, "/api/user/v1/wechat/logins", `{"code":"alice"}`, "Bearer "+serviceToken)
	if code != 403 {
		t.Fatal("disabled user logged in")
	}
	nicknameBody, _ = json.Marshal(api.NicknameRequest{SessionToken: second.SessionToken, Nickname: "不允许"})
	if status, _ := request(t, h, "/api/user/v1/profiles/nickname", string(nicknameBody), "Bearer "+serviceToken); status != 403 {
		t.Fatal("disabled user changed nickname")
	}
	other := biz.New(repository, provider, "other-app", time.Hour, time.Now)
	if _, err = other.Verify(context.Background(), second.SessionToken); err != biz.ErrUnauthenticated {
		t.Fatal("appid isolation", err)
	}
	var wg sync.WaitGroup
	results := make(chan biz.LoginResult, 12)
	failures := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := usecase.Login(context.Background(), "concurrent", "")
			if e != nil {
				failures <- e
			} else {
				results <- r
			}
		}()
	}
	wg.Wait()
	close(results)
	close(failures)
	for e := range failures {
		t.Error(e)
	}
	ids := map[string]bool{}
	for r := range results {
		ids[r.Session.UserID] = true
	}
	if len(ids) != 1 {
		t.Fatal("concurrent users", ids)
	}
	var users, identities, sessions int
	if err = db.QueryRow("SELECT (SELECT count(*) FROM users),(SELECT count(*) FROM wechat_identities),(SELECT count(*) FROM user_sessions)").Scan(&users, &identities, &sessions); err != nil {
		t.Fatal(err)
	}
	if users != 3 || identities != 3 || sessions != 15 {
		t.Fatalf("unexpected counts %d %d %d", users, identities, sessions)
	}
	if _, err = db.Exec("UPDATE user_sessions SET expires_at=created_at"); err == nil {
		t.Fatal("missing expiry constraint")
	}
	var hashes int
	if err = db.QueryRow("SELECT count(*) FROM user_sessions WHERE octet_length(token_hash)=32").Scan(&hashes); err != nil || hashes != sessions {
		t.Fatal("token storage constraint")
	}
}

func (stub) UpdateNickname(context.Context, api.NicknameRequest) (api.UserResponse, error) {
	return api.UserResponse{}, nil
}
