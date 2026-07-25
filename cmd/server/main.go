package main

import (
	"log/slog"
	"os"

	kratoslog "github.com/go-kratos/kratos/v3/log"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/conf"
)

func main() {
	logger := slog.New(kratoslog.NewHandler(
		kratoslog.WithFormat(kratoslog.FormatJSON),
		kratoslog.WithWriter(os.Stdout),
	))
	config, err := conf.Load()
	if err != nil {
		fail(logger, "configuration_load_failed", "could not load AgentRun configuration")
	}
	app, err := buildApp(config, logger)
	if err != nil {
		fail(logger, "service_initialization_failed", "could not initialize AgentRun")
	}
	if err := app.Run(); err != nil {
		fail(logger, "service_runtime_failed", "AgentRun service failed")
	}
}

func fail(logger *slog.Logger, eventCode string, message string) {
	logger.Error(
		message,
		"service", conf.ServiceName,
		"event_code", eventCode,
	)
	os.Exit(1)
}
