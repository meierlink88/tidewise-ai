package main

import (
	"context"
	"errors"
	"log/slog"
	"time"

	kratos "github.com/go-kratos/kratos/v3"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector"
	collectorusecase "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector/usecase"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact"
	eventusecase "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact/usecase"
	eventworkflow "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact/workflow"
	semanticusecase "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic/usecase"
	semanticworkflow "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic/workflow"
	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform/admin"
	bizschedule "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform/scheduling"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/data/artifacts"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/data/connectors"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/data/dataclient"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/data/modelprovider/deepseek"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/data/postgres"
	scheduler "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/data/scheduler"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/server"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/service"
)

const serviceVersion = "1.0.0"

func buildApp(config conf.Config, logger *slog.Logger) (*kratos.App, error) {
	agentLogger := slogAgentLifecycleLogger{
		logger: logger, environment: string(config.App.Env),
	}
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
	dataClient, err := dataclient.New(dataclient.Config{
		BaseURL: config.Data.BaseURL, ServiceToken: config.Secrets.DataServiceToken,
		Timeout:          time.Duration(config.Data.TimeoutSeconds) * time.Second,
		MaxResponseBytes: config.Data.MaxResponseBytes,
	})
	if err != nil {
		return nil, errors.New("Data Service client configuration is invalid")
	}
	eventApplication, err := eventusecase.New(
		store,
		dataClient,
		func(ctx context.Context) (eventusecase.Runtime, error) {
			modelConfigurations, err := store.LoadModelProviderConfigs(ctx)
			if err != nil {
				return eventusecase.Runtime{}, errors.New("Event Fact model configuration is unavailable")
			}
			modelConfiguration, exists := modelConfigurations[collector.ModelProviderDeepSeek]
			if !exists || collector.ValidateModelProviderConfigForEnvironment(
				modelConfiguration, string(config.App.Env),
			) != nil {
				return eventusecase.Runtime{}, errors.New("Event Fact model configuration is incomplete")
			}
			snapshot := eventfact.ExtractionSnapshot{
				PromptSHA256: eventworkflow.PromptSHA256(),
				SchemaSHA256: eventworkflow.SchemaSHA256(),
				ProviderKey:  modelConfiguration.ProviderKey,
				Model:        modelConfiguration.Model,
			}
			artifactReader := artifacts.EventReader{Root: config.Artifact.Root, Executions: store}
			return eventusecase.Runtime{
				Snapshot:      snapshot,
				ReadArtifacts: artifactReader.Read,
				ExtractFacts: func(
					runContext context.Context,
					attempt *eventfact.ExecutionAttempt,
				) (*eventfact.Result, error) {
					modelFactory := deepseek.Factory{
						Timeout: time.Duration(config.EventFact.ModelTimeoutSeconds) * time.Second,
					}
					extractionModel, err := modelFactory.New(runContext, modelConfiguration)
					if err != nil {
						return nil, eventworkflow.ErrExtractionModel
					}
					eventRunnable, err := eventworkflow.NewFactExtraction(
						runContext,
						artifactReader,
						extractionModel,
					)
					if err != nil {
						return nil, errors.New("Event Fact-only workflow could not compile")
					}
					return eventRunnable.Invoke(runContext, attempt)
				},
				Run: func(runContext context.Context, input *eventworkflow.Input) (*eventfact.Result, error) {
					modelFactory := deepseek.Factory{
						Timeout: time.Duration(config.EventFact.ModelTimeoutSeconds) * time.Second,
					}
					extractionModel, err := modelFactory.New(runContext, modelConfiguration)
					if err != nil {
						return nil, eventworkflow.ErrExtractionModel
					}
					reviewModel, err := modelFactory.New(runContext, modelConfiguration)
					if err != nil {
						return nil, eventworkflow.ErrReviewModel
					}
					eventRunnable, err := eventworkflow.New(
						runContext,
						artifactReader,
						store,
						extractionModel,
						reviewModel,
					)
					if err != nil {
						return nil, errors.New("Event Fact workflow could not compile")
					}
					return eventRunnable.Invoke(runContext, input)
				},
			}, nil
		},
		time.Duration(config.EventFact.ReconcileIntervalSeconds)*time.Second,
		eventusecase.WithEventLogger(agentLogger),
	)
	if err != nil {
		return nil, err
	}
	semanticApplication, err := semanticusecase.New(
		store,
		dataClient,
		func(ctx context.Context) (semanticusecase.Runtime, error) {
			modelConfigurations, err := store.LoadModelProviderConfigs(ctx)
			if err != nil {
				return semanticusecase.Runtime{}, errors.New("Event Semantic model configuration is unavailable")
			}
			modelConfiguration, exists := modelConfigurations[collector.ModelProviderDeepSeek]
			if !exists || collector.ValidateModelProviderConfigForEnvironment(
				modelConfiguration, string(config.App.Env),
			) != nil {
				return semanticusecase.Runtime{}, errors.New("Event Semantic model configuration is incomplete")
			}
			modelFactory := deepseek.Factory{
				Timeout: time.Duration(config.EventFact.ModelTimeoutSeconds) * time.Second,
			}
			generator, err := modelFactory.New(ctx, modelConfiguration)
			if err != nil {
				return semanticusecase.Runtime{}, errors.New("Event Semantic Generator is unavailable")
			}
			reviewer, err := modelFactory.New(ctx, modelConfiguration)
			if err != nil {
				return semanticusecase.Runtime{}, errors.New("Event Semantic Reviewer is unavailable")
			}
			runnable, err := semanticworkflow.New(ctx, dataClient, generator, reviewer)
			if err != nil {
				return semanticusecase.Runtime{}, errors.New("Event Semantic workflow could not compile")
			}
			return semanticusecase.Runtime{
				GeneratorModel: modelConfiguration.Model,
				ReviewerModel:  modelConfiguration.Model,
				Run:            runnable,
			}, nil
		},
		time.Duration(config.EventFact.ReconcileIntervalSeconds)*time.Second,
		semanticusecase.WithEventLogger(agentLogger),
	)
	if err != nil {
		return nil, err
	}

	artifactStore := artifacts.Store{
		Root: config.Artifact.Root, Publications: store, Now: time.Now,
		AfterPublication: eventApplication.Notify,
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
		collectorusecase.WithEventLogger(agentLogger),
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
	apiService, err := service.NewAgentRunService(
		collectorApplication,
		adminService,
		semanticApplication,
		scheduleService,
	)
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
		kratos.BeforeStart(func(context.Context) error {
			agentLogger.Info(agentrun.AgentLifecycleEvent{
				Code: "agent_runtime_started", AgentKey: collector.AgentKey,
				AgentVersion: "collector.v1", RuntimeMode: "request",
				Status: "running",
			})
			eventApplication.Start(context.Background())
			eventApplication.Notify()
			semanticApplication.Start(context.Background())
			semanticApplication.Notify()
			return nil
		}),
		kratos.BeforeStop(func(context.Context) error {
			return errors.Join(
				scheduleService.Shutdown(),
				shutdownWithinEach(
					10*time.Second,
					eventApplication.Shutdown,
					semanticApplication.Shutdown,
				),
			)
		}),
		kratos.AfterStop(func(context.Context) error {
			stopContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			collectorApplication.BeginShutdown()
			collectorErr := collectorApplication.Wait(stopContext)
			if collectorErr == nil {
				agentLogger.Info(agentrun.AgentLifecycleEvent{
					Code: "agent_runtime_stopped", AgentKey: collector.AgentKey,
					AgentVersion: "collector.v1", RuntimeMode: "request",
					Status: "stopped",
				})
			} else {
				agentLogger.Error(agentrun.AgentLifecycleEvent{
					Code: "agent_runtime_failed", AgentKey: collector.AgentKey,
					AgentVersion: "collector.v1", RuntimeMode: "request",
					Status: "failed", Stage: "shutdown",
					ErrorCode: "shutdown_deadline",
				})
			}
			databaseErr := closeWithin(stopContext, database.Close)
			return errors.Join(collectorErr, databaseErr)
		}),
	), nil
}

func shutdownWithinEach(
	timeout time.Duration,
	shutdowns ...func(context.Context) error,
) error {
	results := make(chan error, len(shutdowns))
	for _, shutdown := range shutdowns {
		go func(stop func(context.Context) error) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			results <- stop(ctx)
		}(shutdown)
	}
	var joined error
	for range shutdowns {
		joined = errors.Join(joined, <-results)
	}
	return joined
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

type slogAgentLifecycleLogger struct {
	logger      *slog.Logger
	environment string
}

func (l slogAgentLifecycleLogger) Info(event agentrun.AgentLifecycleEvent) {
	l.log(slog.LevelInfo, event)
}

func (l slogAgentLifecycleLogger) Warn(event agentrun.AgentLifecycleEvent) {
	l.log(slog.LevelWarn, event)
}

func (l slogAgentLifecycleLogger) Error(event agentrun.AgentLifecycleEvent) {
	l.log(slog.LevelError, event)
}

func (l slogAgentLifecycleLogger) log(
	level slog.Level,
	event agentrun.AgentLifecycleEvent,
) {
	attributes := []slog.Attr{
		slog.String("service", conf.ServiceName),
		slog.String("environment", l.environment),
		slog.String("event_code", event.Code),
		slog.String("agent_key", event.AgentKey),
		slog.String("agent_version", event.AgentVersion),
		slog.String("runtime_mode", event.RuntimeMode),
	}
	appendString := func(key, value string) {
		if value != "" {
			attributes = append(attributes, slog.String(key, value))
		}
	}
	appendString("execution_id", event.ExecutionID)
	appendString("work_item_id", event.WorkItemID)
	appendString("trigger_source", event.TriggerSource)
	appendString("status", event.Status)
	appendString("outcome", event.Outcome)
	appendString("stage", event.Stage)
	appendString("error_code", event.ErrorCode)
	if event.Attempt > 0 {
		attributes = append(attributes, slog.Int("attempt", event.Attempt))
	}
	if event.MaxAttempts > 0 {
		attributes = append(attributes, slog.Int("max_attempts", event.MaxAttempts))
	}
	if event.Duration > 0 {
		attributes = append(
			attributes,
			slog.Int64("duration_ms", event.Duration.Milliseconds()),
		)
	}
	if counts := safeAgentLifecycleCounts(event.Counts); len(counts) > 0 {
		attributes = append(attributes, slog.Any("counts", counts))
	}
	l.logger.LogAttrs(
		context.Background(), level, "Agent lifecycle event", attributes...,
	)
}

func safeAgentLifecycleCounts(counts map[string]int) map[string]int {
	allowed := map[string]struct{}{
		"raw_results": {}, "merged_results": {}, "accepted_artifacts": {},
		"artifacts": {}, "candidate_events": {}, "published_events": {},
		"events": {}, "submissions": {},
		"accepted_candidates": {}, "rejected_candidates": {},
	}
	result := make(map[string]int, len(counts))
	for key, value := range counts {
		if _, exists := allowed[key]; exists && value >= 0 {
			result[key] = value
		}
	}
	return result
}
