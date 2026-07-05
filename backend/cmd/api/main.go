package main

import (
	"log"
	"net/http"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/config"
	httpserver "github.com/meierlink88/tidewise-ai/backend/internal/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	server := &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      httpserver.NewRouter(cfg),
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSeconds) * time.Second,
	}

	log.Printf("starting %s on %s in %s", cfg.App.Name, cfg.Server.Address(), cfg.App.Env)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
