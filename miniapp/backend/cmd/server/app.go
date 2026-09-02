package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	kratos "github.com/go-kratos/kratos/v3"
	"github.com/go-kratos/kratos/v3/transport"

	reportbiz "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz/report"
	"github.com/meierlink88/tidewise-ai/miniapp/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/miniapp/backend/internal/data"
	reportdata "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/data/report"
	"github.com/meierlink88/tidewise-ai/miniapp/backend/internal/server"
	reportservice "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/service/report"
)

const applicationStopTimeout = 10 * time.Second
const resourceCleanupTimeout = 5 * time.Second

func buildApp(config conf.RuntimeConfig, logger *slog.Logger) (*kratos.App, func(context.Context) error, error) {
	dataClient, err := data.NewHTTPClient(data.HTTPConfig{
		BaseURL: config.DataService.BaseURL, ServiceToken: config.DataService.IdentityToken,
		Timeout: config.DataService.Timeout, MaxReadAttempts: 2,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create Data API client: %w", err)
	}
	repository, err := reportdata.NewRepository(dataClient)
	if err != nil {
		_ = dataClient.Close()
		return nil, nil, fmt.Errorf("create Report repository: %w", err)
	}
	application, err := reportservice.NewService(reportbiz.NewUseCase(repository))
	if err != nil {
		_ = dataClient.Close()
		return nil, nil, fmt.Errorf("create Report service: %w", err)
	}
	httpServer := server.NewHTTPServer(config, logger, application)
	cleanup := func(context.Context) error { return dataClient.Close() }
	return newApp(httpServer, logger), cleanup, nil
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
