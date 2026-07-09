package main

import (
	"net/http"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/config"
)

func TestNewAdminServerUsesConfigAddressAndTimeouts(t *testing.T) {
	cfg := config.Config{
		Server: config.ServerConfig{
			Host:                "127.0.0.1",
			Port:                18080,
			ReadTimeoutSeconds:  5,
			WriteTimeoutSeconds: 10,
		},
	}

	server := newAdminServer(cfg, http.NewServeMux())
	if server.Addr != "127.0.0.1:18080" {
		t.Fatalf("Addr = %q, want 127.0.0.1:18080", server.Addr)
	}
	if server.ReadTimeout.Seconds() != 5 {
		t.Fatalf("ReadTimeout = %s, want 5s", server.ReadTimeout)
	}
	if server.WriteTimeout.Seconds() != 10 {
		t.Fatalf("WriteTimeout = %s, want 10s", server.WriteTimeout)
	}
}
