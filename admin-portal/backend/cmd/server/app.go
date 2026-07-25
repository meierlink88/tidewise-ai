package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	kratos "github.com/go-kratos/kratos/v3"
	"github.com/go-kratos/kratos/v3/transport"

	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/biz"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/data"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/server"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/service"
)

const applicationStopTimeout = 10 * time.Second
const resourceCleanupTimeout = 5 * time.Second

func buildApp(config conf.RuntimeConfig, logger *slog.Logger) (*kratos.App, func(context.Context) error, error) {
	dataServiceRepo, err := data.NewDataHTTPClient(data.DataHTTPConfig{
		BaseURL: config.DataService.BaseURL, ServiceToken: config.DataService.IdentityToken,
		Timeout: config.DataService.Timeout,
	})
	if err != nil {
		return nil, nil, err
	}
	agentRunRepo, err := data.NewAgentRunHTTPClient(data.AgentRunHTTPConfig{
		BaseURL: config.AgentRun.BaseURL, ServiceToken: config.AgentRun.ServiceToken,
		Timeout: config.AgentRun.Timeout,
	})
	if err != nil {
		_ = dataServiceRepo.Close()
		return nil, nil, err
	}

	useCase := biz.NewService(dataServiceRepo, agentRunRepo)
	applicationService := service.NewAdminService(useCase)
	httpServer := server.NewHTTPServer(config, applicationService, logger)

	cleanup := func(context.Context) error {
		return errors.Join(dataServiceRepo.Close(), agentRunRepo.Close())
	}
	return newApp(httpServer, logger), cleanup, nil
}

func newApp(httpServer transport.Server, logger *slog.Logger) *kratos.App {
	return kratos.New(
		kratos.Name(conf.ServiceName),
		kratos.Version(conf.ServiceVersion),
		kratos.Logger(logger),
		kratos.StopTimeout(applicationStopTimeout),
		kratos.Server(httpServer),
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
			return fmt.Errorf("cleanup Admin Portal resources: %w", err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("cleanup Admin Portal resources: %w", ctx.Err())
	}
}
