package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/meierlink88/tidewise-ai/user-service/backend/internal/conf"
)

func main() {
	path := flag.String("config", "", "optional non-secret YAML")
	flag.Parse()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	config, err := conf.Load(*path, true)
	if err != nil {
		logger.Error("invalid User Service configuration")
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	app, close, err := buildApp(ctx, config, logger)
	cancel()
	if err != nil {
		logger.Error("User Service initialization failed")
		os.Exit(1)
	}
	err = app.Run()
	close()
	if err != nil {
		logger.Error("User Service stopped with error")
		os.Exit(1)
	}
}
