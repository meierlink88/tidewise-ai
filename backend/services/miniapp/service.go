// Package miniapp owns the Miniapp BFF HTTP process boundary.
package miniapp

import (
	"net/http"

	"github.com/meierlink88/tidewise-ai/backend/internal/config"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/servicehttp"
)

const ServiceName = "miniapp"

// NewHandler returns the Package 3 health-only facade. BFF routes move behind
// the DataServiceClient in Package 6.
func NewHandler(cfg config.Config) http.Handler {
	return servicehttp.NewHealthHandler(ServiceName, cfg.App.Env)
}

// NewServer preserves an injected legacy handler during the compatibility window.
func NewServer(cfg config.Config, compatibilityHandler http.Handler) *http.Server {
	return servicehttp.NewServer(cfg, ServiceName, compatibilityHandler)
}
