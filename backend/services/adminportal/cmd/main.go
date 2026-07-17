package main

import (
	"log"
	"net/http"

	"github.com/meierlink88/tidewise-ai/backend/services/adminportal"
	"github.com/meierlink88/tidewise-ai/backend/services/adminportal/dataclient"
)

func main() {
	runtime, err := adminportal.LoadRuntimeConfig()
	if err != nil {
		log.Fatalf("load Admin config: %v", err)
	}
	client, err := dataclient.NewHTTPClient(dataclient.HTTPConfig{
		BaseURL: runtime.DataService.BaseURL, ServiceToken: runtime.DataService.IdentityToken, Timeout: runtime.DataService.Timeout,
	})
	if err != nil {
		log.Fatalf("configure Data Service client: %v", err)
	}
	cfg := runtime.ServiceConfig()
	server := adminportal.NewServer(cfg, adminportal.NewHandler(cfg, client, runtime.AdminToken))
	log.Printf("starting %s on %s in %s", adminportal.ServiceName, server.Addr, cfg.App.Env)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
