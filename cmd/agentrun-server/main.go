package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/admin"
	agentrunconfig "github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/config"
	agentrunhttp "github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/httpapi"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/persistence/postgres"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/scheduling"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector/application"
	collectorhttp "github.com/guanchaojia/tidewise-ai-agentrun/internal/collector/httpapi"
)

func main() {
	cfg, err := agentrunconfig.Load()
	if err != nil {
		serverFail("could not load AgentRun configuration")
	}
	databaseURL, err := cfg.PostgresURL()
	if err != nil {
		serverFail("could not build AgentRun database configuration")
	}
	database, err := postgres.Open(context.Background(), databaseURL)
	if err != nil {
		serverFail("could not open AgentRun database")
	}
	defer database.Close()
	store := postgres.New(database)

	collectorApplication, err := application.New(store, cfg.Artifact.Root, application.WithEnvironment(string(cfg.App.Env)))
	if err != nil {
		serverFail("could not initialize AgentRun")
	}
	if store.SchemaReady(context.Background()) {
		if err := collectorApplication.ReconcileStartup(context.Background(), time.Now().UTC()); err != nil {
			serverFail("could not reconcile AgentRun startup state")
		}
	}
	scheduleRunner := collectorScheduleRunner{collector: collectorApplication}
	scheduleService, err := scheduling.New(
		store,
		cfg.Location,
		map[string]scheduling.AgentRunner{collectorAgentKey: scheduleRunner},
	)
	if err != nil {
		serverFail("could not initialize Agent Scheduler")
	}
	defer func() { _ = scheduleService.Shutdown() }()
	if store.SchemaReady(context.Background()) {
		if err := scheduleService.Start(context.Background()); err != nil {
			serverFail("could not start Agent Scheduler")
		}
	}
	adminService, err := admin.New(
		store,
		admin.Registry{
			ModelProviderKeys: []string{collector.ModelProviderDeepSeek},
			ConnectorKeys:     collector.ConnectorKeys(),
		},
		string(cfg.App.Env),
		admin.WithScheduleManager(scheduleService),
	)
	if err != nil {
		serverFail("could not initialize AgentRun Admin API")
	}
	server := newHTTPServer(cfg, newRootHandler(
		collectorhttp.NewHandler(collectorApplication, cfg.Secrets.ServiceToken),
		agentrunhttp.NewAdminHandler(adminService, cfg.Secrets.AdminToken),
	))
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()
	fmt.Printf("AgentRun listening on %s in %s\n", server.Addr, cfg.App.Env)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		serverFail("AgentRun HTTP server failed")
	}
}

func newRootHandler(collectorHandler, adminHandler http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/admin/", adminHandler)
	mux.Handle("/", collectorHandler)
	return mux
}

func newHTTPServer(cfg agentrunconfig.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              cfg.Server.Address(),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func serverFail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
