// Package miniapp owns the Miniapp BFF HTTP process boundary.
package miniapp

import (
	"net/http"

	"github.com/meierlink88/tidewise-ai/backend/internal/platform/apidocs"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/servicehttp"
	miniappapi "github.com/meierlink88/tidewise-ai/backend/services/miniapp/api"
	miniappconfig "github.com/meierlink88/tidewise-ai/backend/services/miniapp/config"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/dataclient"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/transport"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/usecase"
)

const ServiceName = miniappconfig.ServiceName

// NewHandler composes the Miniapp API exclusively through DataServiceClient.
// A missing client produces health endpoints for process-level diagnostics.
func NewHandler(cfg miniappconfig.RuntimeConfig, clients ...dataclient.DataServiceClient) http.Handler {
	cfg.App.Name = ServiceName
	var application http.Handler
	if len(clients) > 0 && clients[0] != nil {
		application = transport.NewRouter(cfg.App, usecase.NewResearchService(clients[0]))
	} else {
		application = servicehttp.NewHealthHandler(ServiceName, cfg.App.Env)
	}
	return apidocs.Wrap(cfg.App.Env, application, apidocs.Config{
		Title:    "Tidewise Miniapp Service API",
		Document: miniappapi.Document(),
	})
}

func NewServer(cfg miniappconfig.RuntimeConfig, handler http.Handler) *http.Server {
	return servicehttp.NewServer(cfg.Server, ServiceName, cfg.App.Env, handler)
}
