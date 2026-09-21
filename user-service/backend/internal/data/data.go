package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, errors.New("user database unavailable")
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	if db.PingContext(ctx) != nil {
		db.Close()
		return nil, errors.New("user database unavailable")
	}
	return db, nil
}

// Readiness never applies DDL, and therefore works with a DML-only runtime role.
func Ready(ctx context.Context, db *sql.DB) error {
	var version int
	if err := db.QueryRowContext(ctx, `SELECT version_id FROM goose_db_version WHERE is_applied ORDER BY id DESC LIMIT 1`).Scan(&version); err != nil || version != 2 {
		return errors.New("user schema unavailable")
	}
	rows, err := db.QueryContext(ctx, `SELECT s.token_hash,i.openid,u.status,u.nickname FROM user_sessions s JOIN wechat_identities i ON i.id=s.wechat_identity_id JOIN users u ON u.id=i.user_id LIMIT 0`)
	if err != nil {
		return errors.New("user schema unavailable")
	}
	return rows.Close()
}
