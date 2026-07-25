package main

import (
	"context"
	"log/slog"
	"time"

	kratos "github.com/go-kratos/kratos/v3"
	"github.com/go-kratos/kratos/v3/transport"

	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/biz"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/conf"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/data"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/server"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/service"
)

const applicationStopTimeout = 10 * time.Second
const resourceCleanupTimeout = 5 * time.Second

func buildApp(config conf.RuntimeConfig, logger *slog.Logger) (*kratos.App, error) {
	repository, err := data.NewHTTPClient(data.HTTPConfig{
		BaseURL:      config.DataService.BaseURL,
		ServiceToken: config.DataService.IdentityToken,
		Timeout:      config.DataService.Timeout,
	})
	if err != nil {
		return nil, err
	}
	useCase := biz.NewResearchService(repository)
	applicationService := service.NewResearchService(useCase)
	httpServer := server.NewHTTPServer(config, applicationService, logger)

	return newApp(httpServer, logger, func(context.Context) error {
		return repository.Close()
	}), nil
}

func newApp(server transport.Server, logger *slog.Logger, cleanup func(context.Context) error) *kratos.App {
	options := []kratos.Option{
		kratos.Name(conf.ServiceName),
		kratos.Version(conf.ServiceVersion),
		kratos.Logger(logger),
		kratos.StopTimeout(applicationStopTimeout),
		kratos.Server(server),
	}
	if cleanup != nil {
		options = append(options, kratos.AfterStop(func(ctx context.Context) error {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), resourceCleanupTimeout)
			defer cancel()
			return cleanup(cleanupCtx)
		}))
	}
	return kratos.New(
		options...,
	)
}
