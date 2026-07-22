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

	agentrunconfig "github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/config"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/persistence/postgres"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector/application"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector/httpapi"
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
	if store.SchemaReady(context.Background()) {
		if err := store.FailStaleExecutions(context.Background(), time.Now().UTC()); err != nil {
			serverFail("could not reconcile stale Agent Executions")
		}
	}

	collectorApplication, err := application.New(store, cfg.Artifact.Root, application.WithEnvironment(string(cfg.App.Env)))
	if err != nil {
		serverFail("could not initialize AgentRun")
	}
	server := newHTTPServer(cfg, httpapi.NewHandler(collectorApplication, cfg.Secrets.ServiceToken))
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
