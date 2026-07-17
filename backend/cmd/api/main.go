package main

import (
	"log"
	"net/http"

	miniappservice "github.com/meierlink88/tidewise-ai/backend/services/miniapp"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/dataclient"
)

func main() {
	runtime, err := miniappservice.LoadRuntimeConfig()
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
	server := miniappservice.NewServer(cfg, miniappservice.NewHandler(cfg, client))
	log.Printf("starting %s on %s in %s", miniappservice.ServiceName, server.Addr, cfg.App.Env)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
