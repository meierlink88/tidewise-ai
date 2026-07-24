package main

import (
	"log"
	"net/http"

	"github.com/meierlink88/tidewise-ai/admin-portal/backend"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/agentrunclient"
	adminconfig "github.com/meierlink88/tidewise-ai/admin-portal/backend/config"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/dataclient"
)

func main() {
	runtime, err := adminconfig.LoadRuntimeConfig()
	if err != nil {
		log.Fatalf("load Admin config: %v", err)
	}
	dataClient, err := dataclient.NewHTTPClient(dataclient.HTTPConfig{
		BaseURL: runtime.DataService.BaseURL, ServiceToken: runtime.DataService.IdentityToken, Timeout: runtime.DataService.Timeout,
	})
	if err != nil {
		log.Fatalf("configure Data Service client: %v", err)
	}
	agentRunClient, err := agentrunclient.NewHTTPClient(agentrunclient.HTTPConfig{
		BaseURL: runtime.AgentRun.BaseURL, ServiceToken: runtime.AgentRun.ServiceToken, Timeout: runtime.AgentRun.Timeout,
	})
	if err != nil {
		log.Fatalf("configure AgentRun client: %v", err)
	}
	server := adminportal.NewServer(runtime, adminportal.NewHandler(runtime, dataClient, agentRunClient, runtime.AdminToken))
	log.Printf("starting %s on %s in %s", adminportal.ServiceName, server.Addr, runtime.App.Env)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
