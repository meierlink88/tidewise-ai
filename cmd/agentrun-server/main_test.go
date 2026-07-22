package main

import (
	"net/http"
	"testing"
	"time"

	agentrunconfig "github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/config"
)

func TestNewHTTPServerUsesConfiguredAddressAndFixedTimeouts(t *testing.T) {
	cfg := agentrunconfig.Config{Server: agentrunconfig.ServerConfig{
		Host: "0.0.0.0", Port: 9080,
	}}

	server := newHTTPServer(cfg, http.NewServeMux())
	if server.Addr != "0.0.0.0:9080" {
		t.Fatalf("server address = %q, want 0.0.0.0:9080", server.Addr)
	}
	if server.ReadHeaderTimeout != 5*time.Second || server.ReadTimeout != 15*time.Second || server.WriteTimeout != 30*time.Second || server.IdleTimeout != 60*time.Second {
		t.Fatalf("server timeouts = %#v", server)
	}
}
