// Package data owns the Data Service HTTP process boundary.
package data

import (
	"net/http"

	dataapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/config"
)

const ServiceName = config.ServiceName

// NewHandler composes the Data API, health endpoints and environment-gated docs.
func NewHandler(cfg config.Config, apiHandlers ...http.Handler) http.Handler {
	health := newHealthHandler(ServiceName, cfg.App.Env)
	application := health
	if len(apiHandlers) > 0 && apiHandlers[0] != nil {
		mux := http.NewServeMux()
		mux.Handle("/healthz", health)
		mux.Handle("/readyz", health)
		mux.Handle("/", apiHandlers[0])
		application = mux
	}
	return wrapAPIDocs(cfg.App.Env, application, apiDocsConfig{
		Title:    "Tidewise Data Service API",
		Document: dataapi.Document(),
	})
}

// NewServer composes the Data API with service-owned health endpoints.
func NewServer(cfg config.Config, apiHandler http.Handler) *http.Server {
	return newHTTPServer(cfg.Server, NewHandler(cfg, apiHandler))
}
