package main

import (
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/platform/runtimeconfig"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/conf"
)

func TestBuildAppComposesKratosApplicationWithoutStartingNetwork(t *testing.T) {
	app, err := buildApp(conf.RuntimeConfig{
		App: runtimeconfig.AppConfig{Name: conf.ServiceName, Env: runtimeconfig.EnvLocal},
		Server: runtimeconfig.ServerConfig{
			Host: "127.0.0.1", Port: 18082, ReadTimeoutSeconds: 5, WriteTimeoutSeconds: 10,
		},
		DataService: conf.DataServiceRuntimeConfig{
			BaseURL: "http://127.0.0.1:18081", IdentityToken: "test-token", Timeout: time.Second,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if app == nil {
		t.Fatal("buildApp() = nil")
	}
}
