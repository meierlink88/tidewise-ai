package watchlist

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	identity "github.com/meierlink88/tidewise-ai/user-service/backend/internal/biz/identity"
	biz "github.com/meierlink88/tidewise-ai/user-service/backend/internal/biz/watchlist"
	"github.com/pressly/goose/v3"
)

func isolated(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("USER_DATABASE_URL")
	if dsn == "" {
		t.Skip("USER_DATABASE_URL not set")
	}
	u, err := url.Parse(dsn)
	if err != nil || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") || !strings.HasPrefix(u.Path, "/tidewise_user_") {
		t.Fatal("requires isolated local User database")
	}
	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "watchlist_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	db, err := sql.Open("pgx", u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close(); admin.Exec("DROP SCHEMA " + schema + " CASCADE"); admin.Close() })
	p, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("../../../migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.Up(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err = p.Up(context.Background()); err != nil {
		t.Fatal(err)
	}
	return db
}
func seed(t *testing.T, db *sql.DB, token string, now time.Time) string {
	t.Helper()
	uid, iid, sid := uuid.NewString(), uuid.NewString(), uuid.NewString()
	hash, err := identity.TokenHash(token)
	if err != nil {
		t.Fatal(err)
	}
	for _, cmd := range []struct {
		q string
		a []any
	}{
		{"INSERT INTO users(id)VALUES($1)", []any{uid}},
		{"INSERT INTO wechat_identities(id,user_id,appid,openid)VALUES($1,$2,'app',$3)", []any{iid, uid, uuid.NewString()}},
		{"INSERT INTO user_sessions(id,wechat_identity_id,token_hash,created_at,expires_at)VALUES($1,$2,$3,$4,$5)", []any{sid, iid, hash, now, now.Add(time.Hour)}},
	} {
		if _, err = db.Exec(cmd.q, cmd.a...); err != nil {
			t.Fatal(err)
		}
	}
	return uid
}
func TestPrivateWatchlistIsolationIdempotencyPaginationAndAuthorization(t *testing.T) {
	db := isolated(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	a, b := strings.Repeat("A", 43), strings.Repeat("B", 42)+"A"
	user := seed(t, db, a, now)
	seed(t, db, b, now)
	u := biz.New(New(db), "app", func() time.Time { return now })
	first := "STKf4a8eb61-c352-5980-91b1-9da6eba8f8af"
	second := "STK7e93fe75-d0eb-518a-943e-98593da4f04c"
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- u.Change(ctx, a, first, true) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	page, err := u.List(ctx, a, "", 20)
	if err != nil || page.Total != 1 {
		t.Fatalf("duplicate writes: %#v %v", page, err)
	}
	original := page.Items[0].AddedAt
	if err = u.Change(ctx, b, first, false); err != nil {
		t.Fatal(err)
	}
	other, err := u.List(ctx, b, "", 20)
	if err != nil || other.Total != 0 {
		t.Fatal("cross-user leakage", err)
	}
	now = now.Add(time.Second)
	if err = u.Change(ctx, a, first, true); err != nil {
		t.Fatal(err)
	}
	if err = u.Change(ctx, a, second, true); err != nil {
		t.Fatal(err)
	}
	page, err = u.List(ctx, a, "", 1)
	if err != nil || page.Total != 2 || page.Items[0].StockID != second || page.NextCursor == "" {
		t.Fatal(page, err)
	}
	tail, err := u.List(ctx, a, page.NextCursor, 1)
	if err != nil || len(tail.Items) != 1 || !tail.Items[0].AddedAt.Equal(original) || tail.NextCursor != "" {
		t.Fatal(tail, err)
	}
	for i := 0; i < 2; i++ {
		if err = u.Change(ctx, a, first, false); err != nil {
			t.Fatal(err)
		}
	}
	now = now.Add(time.Second)
	if err = u.Change(ctx, a, first, true); err != nil {
		t.Fatal(err)
	}
	page, err = u.List(ctx, a, "", 20)
	if err != nil || page.Items[0].StockID != first {
		t.Fatal("re-add must be newest", err)
	}
	if _, err = u.Check(ctx, b, []string{first, second}); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("UPDATE users SET status='disabled' WHERE id=$1", user); err != nil {
		t.Fatal(err)
	}
	if err = u.Change(ctx, a, first, false); !errors.Is(err, identity.ErrDisabled) {
		t.Fatal(err)
	}
	if _, err = db.Exec("UPDATE users SET status='active' WHERE id=$1", user); err != nil {
		t.Fatal(err)
	}
	wrong := biz.New(New(db), "other-app", func() time.Time { return now })
	if err = wrong.Change(ctx, a, first, false); !errors.Is(err, identity.ErrUnauthenticated) {
		t.Fatal(err)
	}
	expired := biz.New(New(db), "app", func() time.Time { return now.Add(2 * time.Hour) })
	if err = expired.Change(ctx, a, first, false); !errors.Is(err, identity.ErrUnauthenticated) {
		t.Fatal(err)
	}
	hash, _ := identity.TokenHash(a)
	if _, err = db.Exec("UPDATE user_sessions SET revoked_at=$2 WHERE token_hash=$1", hash, now); err != nil {
		t.Fatal(err)
	}
	if err = u.Change(ctx, a, first, false); !errors.Is(err, identity.ErrUnauthenticated) {
		t.Fatal(err)
	}
	var count int
	if err = db.QueryRow("SELECT count(*) FROM user_watchlist WHERE user_id=$1", user).Scan(&count); err != nil || count != 2 {
		t.Fatal("unauthorized mutation persisted")
	}
}
