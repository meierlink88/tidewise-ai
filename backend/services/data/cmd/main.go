package main

import (
	"log"
	"net/http"

	"github.com/meierlink88/tidewise-ai/backend/internal/config"
	"github.com/meierlink88/tidewise-ai/backend/services/data"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	server := data.NewServer(cfg, nil)
	log.Printf("starting %s on %s in %s", data.ServiceName, server.Addr, cfg.App.Env)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
