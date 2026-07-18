// Package data owns the Data Service HTTP process boundary.
package data

import (
	"net/http"

	"github.com/meierlink88/tidewise-ai/backend/internal/platform/servicehttp"
	"github.com/meierlink88/tidewise-ai/backend/services/data/config"
)

const ServiceName = config.ServiceName

// NewHandler returns the Data Service health endpoints.
func NewHandler(cfg config.Config) http.Handler {
	return servicehttp.NewHealthHandler(ServiceName, cfg.App.Env)
}

// NewServer composes the Data API with service-owned health endpoints.
func NewServer(cfg config.Config, apiHandler http.Handler) *http.Server {
	if apiHandler == nil {
		return servicehttp.NewServer(cfg.Server, ServiceName, cfg.App.Env, nil)
	}
	health := NewHandler(cfg)
	mux := http.NewServeMux()
	mux.Handle("/healthz", health)
	mux.Handle("/readyz", health)
	mux.Handle("/", apiHandler)
	return servicehttp.NewServer(cfg.Server, ServiceName, cfg.App.Env, mux)
}
