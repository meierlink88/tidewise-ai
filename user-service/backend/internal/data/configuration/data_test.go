package configuration

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	biz "github.com/meierlink88/tidewise-ai/user-service/backend/internal/biz/configuration"
)

func TestPrivateConfigurationLifecycle(t *testing.T) {
	dsn := os.Getenv("USER_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("disposable PostgreSQL required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var database string
	if db.QueryRow("SELECT current_database()").Scan(&database) != nil || database != "tidewise_user_test" {
		t.Fatal("requires disposable tidewise_user_test")
	}
	if _, err = db.Exec("DELETE FROM user_configurations WHERE code='wechat_miniapp'"); err != nil {
		t.Fatal("cannot reset configuration")
	}
	repo := New(db)
	ctx := context.Background()
	if _, err = repo.Load(ctx); err != biz.ErrUnavailable {
		t.Fatal("missing configuration accepted")
	}
	first, err := biz.NewEntry(biz.Credentials{AppID: "app-a", AppSecret: "secret-a"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.Save(ctx, first); err != nil {
		t.Fatal(err)
	}
	result, err := repo.Load(ctx)
	if err != nil || result != first.Credentials {
		t.Fatal("configuration readback mismatch")
	}
	second, _ := biz.NewEntry(biz.Credentials{AppID: "app-b", AppSecret: "secret-b"}, time.Now())
	if err = repo.Save(ctx, second); err != nil {
		t.Fatal(err)
	}
	result, err = repo.Load(ctx)
	if err != nil || result != second.Credentials {
		t.Fatal("configuration replacement mismatch")
	}
	var id string
	if db.QueryRow("SELECT id FROM user_configurations WHERE code='wechat_miniapp'").Scan(&id) != nil || id != first.ID {
		t.Fatal("stable dictionary ID changed")
	}
	if _, err = db.Exec(`UPDATE user_configurations SET value='{"app_id":"bad","app_secret":""}' WHERE code='wechat_miniapp'`); err != nil {
		t.Fatal("failed to arrange corrupt config")
	}
	if _, err = repo.Load(ctx); err != biz.ErrUnavailable {
		t.Fatal("corrupt configuration accepted")
	}
	if _, err = db.Exec("DELETE FROM user_configurations WHERE code='wechat_miniapp'"); err != nil {
		t.Fatal("cleanup failed")
	}
}
