package testsupport

import (
	"context"
	"fmt"
	"net/url"

	"github.com/google/uuid"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/data/postgres"
	"github.com/jackc/pgx/v5"
)

func IsolatedPostgresDatabase(ctx context.Context, databaseURL, _ string) (string, func(), error) {
	base, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		return "", nil, err
	}
	databaseName := "tidewise_ai_server_test_" + uuid.NewString()
	identifier := pgx.Identifier{databaseName}.Sanitize()
	if _, err := base.Exec(ctx, "CREATE DATABASE "+identifier); err != nil {
		base.Close()
		return "", nil, fmt.Errorf("create isolated test database: %w", err)
	}
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		_, _ = base.Exec(context.Background(), "DROP DATABASE "+identifier+" WITH (FORCE)")
		base.Close()
		return "", nil, fmt.Errorf("parse isolated test database URL: %w", err)
	}
	parsed.Path = "/" + databaseName
	cleanup := func() {
		_, _ = base.Exec(context.Background(), "DROP DATABASE "+identifier+" WITH (FORCE)")
		base.Close()
	}
	return parsed.String(), cleanup, nil
}
