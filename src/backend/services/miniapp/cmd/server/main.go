package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/conf"
)

func main() {
	config, err := conf.LoadRuntimeConfig()
	if err != nil {
		log.Fatalf("load Miniapp config: %v", err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	app, err := buildApp(config, logger)
	if err != nil {
		log.Fatalf("compose Miniapp service: %v", err)
	}
	log.Printf("starting %s on %s in %s", conf.ServiceName, config.Server.Address(), config.App.Env)
	if err := app.Run(); err != nil {
		log.Fatalf("Miniapp service failed: %v", err)
	}
}
