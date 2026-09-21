package main

import (
	"context"
	"log/slog"
	"time"

	kratos "github.com/go-kratos/kratos/v3"
	biz "github.com/meierlink88/tidewise-ai/user-service/backend/internal/biz/identity"
	"github.com/meierlink88/tidewise-ai/user-service/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/user-service/backend/internal/data"
	configuration "github.com/meierlink88/tidewise-ai/user-service/backend/internal/data/configuration"
	adapter "github.com/meierlink88/tidewise-ai/user-service/backend/internal/data/identity"
	"github.com/meierlink88/tidewise-ai/user-service/backend/internal/server"
	service "github.com/meierlink88/tidewise-ai/user-service/backend/internal/service/identity"
)

func buildApp(ctx context.Context, c conf.Config, logger *slog.Logger) (*kratos.App, func(), error) {
	db, err := data.Open(ctx, c.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	if err = data.Ready(ctx, db); err != nil {
		db.Close()
		return nil, nil, err
	}
	credentials, err := configuration.New(db).Load(ctx)
	if err != nil {
		db.Close()
		return nil, nil, err
	}
	usecase := biz.New(adapter.NewRepository(db), adapter.NewWechat(credentials.AppID, credentials.AppSecret, nil), credentials.AppID, c.SessionTTL(), time.Now)
	httpServer := server.New(c.Address, c.ServiceToken, service.New(usecase), func(ctx context.Context) error { return data.Ready(ctx, db) }, logger)
	return kratos.New(kratos.Name("tidewise-user-service"), kratos.Version("v1"), kratos.Logger(logger), kratos.StopTimeout(10*time.Second), kratos.Server(httpServer)), func() { db.Close() }, nil
}
