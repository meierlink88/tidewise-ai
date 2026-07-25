package main

import (
	"log/slog"
	"os"

	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/conf"
)

func main() {
	os.Exit(run())
}

func run() int {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	config, err := conf.LoadRuntimeConfig()
	if err != nil {
		logger.Error("load Miniapp config", slog.String("error", err.Error()))
		return 1
	}
	app, cleanup, err := buildApp(config, logger)
	if err != nil {
		logger.Error("compose Miniapp service", slog.String("error", err.Error()))
		return 1
	}
	logger.Info("starting service",
		slog.String("service", conf.ServiceName),
		slog.String("address", config.Server.Address()),
		slog.String("environment", string(config.App.Env)),
	)
	if err := runApplication(app, cleanup); err != nil {
		logger.Error("Miniapp service failed",
			slog.String("service", conf.ServiceName),
			slog.String("error", err.Error()),
		)
		return 1
	}
	return 0
}
