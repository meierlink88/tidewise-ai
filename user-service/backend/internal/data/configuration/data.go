package configuration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"

	biz "github.com/meierlink88/tidewise-ai/user-service/backend/internal/biz/configuration"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository {
	if db == nil {
		panic("missing configuration database")
	}
	return &Repository{db}
}
func (r *Repository) Load(ctx context.Context) (biz.Credentials, error) {
	var raw []byte
	if err := r.db.QueryRowContext(ctx, `SELECT value FROM user_configurations WHERE code=$1`, biz.WechatCode).Scan(&raw); err != nil {
		return biz.Credentials{}, biz.ErrUnavailable
	}
	var c biz.Credentials
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&c) != nil || decoder.Decode(new(any)) != io.EOF || c.Validate() != nil {
		return biz.Credentials{}, biz.ErrUnavailable
	}
	return c, nil
}

// The unique code is updated in one statement, preserving its stable row ID.
func (r *Repository) Save(ctx context.Context, e biz.Entry) error {
	if e.Credentials.Validate() != nil {
		return biz.ErrUnavailable
	}
	raw, err := json.Marshal(e.Credentials)
	if err != nil {
		return biz.ErrUnavailable
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO user_configurations(id,code,value,updated_at) VALUES($1,$2,$3,$4) ON CONFLICT(code) DO UPDATE SET value=EXCLUDED.value,updated_at=EXCLUDED.updated_at`, e.ID, biz.WechatCode, raw, e.UpdatedAt)
	if err != nil {
		return biz.ErrUnavailable
	}
	return nil
}
