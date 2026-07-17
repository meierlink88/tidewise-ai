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

// NewServer owns Data Service process settings without connecting to storage.
func NewServer(cfg config.Config, compatibilityHandler http.Handler) *http.Server {
	return servicehttp.NewServer(cfg, ServiceName, compatibilityHandler)
}
