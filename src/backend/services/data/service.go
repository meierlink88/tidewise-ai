// Package data owns the Data Service HTTP process boundary.
package data

import (
	"net/http"

	"github.com/meierlink88/tidewise-ai/backend/internal/config"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/servicehttp"
)

const ServiceName = "data"

// NewHandler returns the Package 3 health-only facade. Data API routes are
// introduced separately in Package 4.
func NewHandler(cfg config.Config) http.Handler {
	return servicehttp.NewHealthHandler(ServiceName, cfg.App.Env)
}

// NewServer owns Data Service process settings and composes its API with the
// service-owned liveness/readiness facade.
func NewServer(cfg config.Config, apiHandler http.Handler) *http.Server {
	if apiHandler == nil {
		return servicehttp.NewServer(cfg, ServiceName, nil)
	}
	health := NewHandler(cfg)
	mux := http.NewServeMux()
	mux.Handle("/healthz", health)
	mux.Handle("/readyz", health)
	mux.Handle("/", apiHandler)
	return servicehttp.NewServer(cfg, ServiceName, mux)
}
