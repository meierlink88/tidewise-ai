package main

import (
	"context"
	"errors"
	"log/slog"
	"time"

	kratos "github.com/go-kratos/kratos/v3"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/biz/agents/collector"
	collectorusecase "github.com/guanchaojia/tidewise-ai-agentrun/internal/biz/agents/collector/usecase"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/biz/platform/admin"
	bizschedule "github.com/guanchaojia/tidewise-ai-agentrun/internal/biz/platform/scheduling"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/conf"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/data/artifacts"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/data/connectors"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/data/modelprovider/deepseek"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/data/postgres"
	scheduler "github.com/guanchaojia/tidewise-ai-agentrun/internal/data/scheduler"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/server"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/service"
)

const serviceVersion = "1.0.0"

func buildApp(config conf.Config, logger *slog.Logger) (*kratos.App, error) {
	startupContext, cancelStartup := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelStartup()
	databaseURL, err := config.PostgresURL()
	if err != nil {
		return nil, errors.New("database configuration is invalid")
	}
	database, err := postgres.Open(startupContext, databaseURL)
	if err != nil {
		return nil, errors.New("database is unavailable")
	}
	cleanupDatabase := true
	defer func() {
		if cleanupDatabase {
			database.Close()
		}
	}()

	store := postgres.New(database)
	if !store.SchemaReady(startupContext) {
		return nil, errors.New("database schema is incompatible")
	}
	artifactStore := artifacts.Store{
		Root: config.Artifact.Root, Publications: store, Now: time.Now,
	}
	if err := artifactStore.Ready(startupContext); err != nil {
		return nil, err
	}
	collectorApplication, err := collectorusecase.New(
		store,
		deepseek.Factory{},
		connectors.Factory{},
		artifactStore,
		collectorusecase.WithEnvironment(string(config.App.Env)),
		collectorusecase.WithEventLogger(slogEventLogger{logger: logger}),
	)
	if err != nil {
		return nil, err
	}
	if err := collectorApplication.ReconcileStartup(startupContext, time.Now().UTC()); err != nil {
		return nil, errors.New("startup reconciliation failed")
	}

	scheduleRunner := collector.NewScheduleRunner(collectorApplication)
	scheduleRunners := map[string]bizschedule.AgentRunner{collector.AgentKey: scheduleRunner}
	scheduleRuntime, err := scheduler.NewRuntime(
		store,
		config.Location,
		scheduleRunners,
		scheduler.WithEventLogger(slogScheduleEventLogger{logger: logger}),
	)
	if err != nil {
		return nil, err
	}
	scheduleService, err := bizschedule.New(store, scheduleRunners, scheduleRuntime)
	if err != nil {
		return nil, err
	}
	if err := scheduleService.Start(startupContext); err != nil {
		_ = scheduleService.Shutdown()
		return nil, errors.New("Agent Scheduler failed to start")
	}

	adminService, err := admin.New(
		store,
		admin.Registry{
			ModelProviderKeys: []string{collector.ModelProviderDeepSeek},
			ConnectorKeys:     collector.ConnectorKeys(),
		},
		string(config.App.Env),
		admin.WithScheduleManager(scheduleService),
	)
	if err != nil {
		_ = scheduleService.Shutdown()
		return nil, err
	}
	apiService, err := service.NewAgentRunService(collectorApplication, adminService, scheduleService)
	if err != nil {
		_ = scheduleService.Shutdown()
		return nil, err
	}
	httpServer := server.NewHTTPServer(config, apiService, apiService, logger)

	cleanupDatabase = false
	return kratos.New(
		kratos.Name(conf.ServiceName),
		kratos.Version(serviceVersion),
		kratos.Logger(logger),
		kratos.Server(httpServer),
		kratos.StopTimeout(10*time.Second),
		kratos.BeforeStop(func(context.Context) error {
			return scheduleService.Shutdown()
		}),
		kratos.AfterStop(func(context.Context) error {
			stopContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			collectorApplication.BeginShutdown()
			collectorErr := collectorApplication.Wait(stopContext)
			databaseErr := closeWithin(stopContext, database.Close)
			return errors.Join(collectorErr, databaseErr)
		}),
	), nil
}

func closeWithin(ctx context.Context, closeResource func()) error {
	closed := make(chan struct{})
	go func() {
		closeResource()
		close(closed)
	}()
	select {
	case <-closed:
		return nil
	case <-ctx.Done():
		return errors.New("resource cleanup exceeded shutdown deadline")
	}
}

type slogScheduleEventLogger struct {
	logger *slog.Logger
}

func (l slogScheduleEventLogger) Error(eventCode string, scheduleID string) {
	l.logger.Error(
		"Agent Schedule lifecycle event",
		"service", conf.ServiceName,
		"event_code", eventCode,
		"schedule_id", scheduleID,
	)
}

type slogEventLogger struct {
	logger *slog.Logger
}

func (l slogEventLogger) Error(eventCode string, executionID string) {
	l.logger.Error(
		"Collector lifecycle event",
		"service", conf.ServiceName,
		"event_code", eventCode,
		"execution_id", executionID,
	)
}
