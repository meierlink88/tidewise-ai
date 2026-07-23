// Package data owns the Data Service HTTP process boundary.
package data

import (
	"net/http"

	"github.com/meierlink88/tidewise-ai/backend/internal/platform/apidocs"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/servicehttp"
	dataapi "github.com/meierlink88/tidewise-ai/backend/services/data/api"
	"github.com/meierlink88/tidewise-ai/backend/services/data/config"
)

const ServiceName = config.ServiceName

// NewHandler composes the Data API, health endpoints and environment-gated docs.
func NewHandler(cfg config.Config, apiHandlers ...http.Handler) http.Handler {
	health := servicehttp.NewHealthHandler(ServiceName, cfg.App.Env)
	application := health
	if len(apiHandlers) > 0 && apiHandlers[0] != nil {
		mux := http.NewServeMux()
		mux.Handle("/healthz", health)
		mux.Handle("/readyz", health)
		mux.Handle("/", apiHandlers[0])
		application = mux
	}
	return apidocs.Wrap(cfg.App.Env, application, apidocs.Config{
		Title:    "Tidewise Data Service API",
		Document: dataapi.Document(),
	})
}

// NewServer composes the Data API with service-owned health endpoints.
func NewServer(cfg config.Config, apiHandler http.Handler) *http.Server {
	return servicehttp.NewServer(cfg.Server, ServiceName, cfg.App.Env, NewHandler(cfg, apiHandler))
}
