package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	kratos "github.com/go-kratos/kratos/v3"
	"github.com/go-kratos/kratos/v3/transport"

	"github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz"
	"github.com/meierlink88/tidewise-ai/miniapp/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/miniapp/backend/internal/data"
	"github.com/meierlink88/tidewise-ai/miniapp/backend/internal/server"
	"github.com/meierlink88/tidewise-ai/miniapp/backend/internal/service"
)

const applicationStopTimeout = 10 * time.Second
const resourceCleanupTimeout = 5 * time.Second

func buildApp(config conf.RuntimeConfig, logger *slog.Logger) (*kratos.App, func(context.Context) error, error) {
	repository, err := data.NewHTTPClient(data.HTTPConfig{
		BaseURL:      config.DataService.BaseURL,
		ServiceToken: config.DataService.IdentityToken,
		Timeout:      config.DataService.Timeout,
	})
	if err != nil {
		return nil, nil, err
	}
	useCase := biz.NewResearchService(repository)
	applicationService := service.NewResearchService(useCase)
	httpServer := server.NewHTTPServer(config, applicationService, logger)

	return newApp(httpServer, logger), func(context.Context) error {
		return repository.Close()
	}, nil
}

func newApp(server transport.Server, logger *slog.Logger) *kratos.App {
	return kratos.New(
		kratos.Name(conf.ServiceName),
		kratos.Version(conf.ServiceVersion),
		kratos.Logger(logger),
		kratos.StopTimeout(applicationStopTimeout),
		kratos.Server(server),
	)
}

func runApplication(app interface{ Run() error }, cleanup func(context.Context) error) error {
	runErr := app.Run()
	cleanupErr := runCleanup(cleanup, resourceCleanupTimeout)
	return errors.Join(runErr, cleanupErr)
}

func runCleanup(cleanup func(context.Context) error, timeout time.Duration) error {
	if cleanup == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result := make(chan error, 1)
	go func() {
		result <- cleanup(ctx)
	}()

	select {
	case err := <-result:
		if err != nil {
			return fmt.Errorf("cleanup Miniapp resources: %w", err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("cleanup Miniapp resources: %w", ctx.Err())
	}
}
