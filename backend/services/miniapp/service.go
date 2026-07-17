// Package miniapp owns the Miniapp BFF HTTP process boundary.
package miniapp

import (
	"net/http"

	"github.com/meierlink88/tidewise-ai/backend/internal/apps/miniappapi"
	"github.com/meierlink88/tidewise-ai/backend/internal/config"
	httpserver "github.com/meierlink88/tidewise-ai/backend/internal/http"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/servicehttp"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/dataclient"
)

const ServiceName = "miniapp"

// NewHandler preserves the health-only facade when no client is supplied and
// otherwise composes the Miniapp API exclusively through DataServiceClient.
func NewHandler(cfg config.Config, clients ...dataclient.DataServiceClient) http.Handler {
	cfg.App.Name = ServiceName
	if len(clients) > 0 && clients[0] != nil {
		return httpserver.NewRouter(cfg, miniappapi.NewResearchService(clients[0]))
	}
	return servicehttp.NewHealthHandler(ServiceName, cfg.App.Env)
}

// NewServer preserves an injected legacy handler during the compatibility window.
func NewServer(cfg config.Config, compatibilityHandler http.Handler) *http.Server {
	return servicehttp.NewServer(cfg, ServiceName, compatibilityHandler)
}
