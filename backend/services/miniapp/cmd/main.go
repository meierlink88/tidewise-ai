package main

import (
	"log"
	"net/http"

	"github.com/meierlink88/tidewise-ai/backend/services/miniapp"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/dataclient"
)

func main() {
	runtime, err := miniapp.LoadRuntimeConfig()
	if err != nil {
		log.Fatalf("load Miniapp config: %v", err)
	}
	client, err := dataclient.NewHTTPClient(dataclient.HTTPConfig{
		BaseURL: runtime.DataService.BaseURL, ServiceToken: runtime.DataService.IdentityToken, Timeout: runtime.DataService.Timeout,
	})
	if err != nil {
		log.Fatalf("configure Data Service client: %v", err)
	}
	cfg := runtime.ServiceConfig()
	server := miniapp.NewServer(cfg, miniapp.NewHandler(cfg, client))
	log.Printf("starting %s on %s in %s", miniapp.ServiceName, server.Addr, cfg.App.Env)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
