// Package adminportal owns the Admin Portal BFF HTTP process boundary.
package adminportal

import (
	"net/http"

	"github.com/meierlink88/tidewise-ai/admin-portal/backend/agentrunclient"
	adminapi "github.com/meierlink88/tidewise-ai/admin-portal/backend/api"
	adminconfig "github.com/meierlink88/tidewise-ai/admin-portal/backend/config"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/dataclient"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/transport"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/usecase"
)

const ServiceName = adminconfig.ServiceName

// NewHandler composes the Admin BFF through its owned downstream service clients.
func NewHandler(
	cfg adminconfig.RuntimeConfig,
	dataClient dataclient.DataServiceClient,
	agentRunClient agentrunclient.Client,
	adminToken string,
) http.Handler {
	cfg.App.Name = ServiceName
	application := transport.NewRouter(cfg.App, usecase.NewService(dataClient, agentRunClient), adminToken, cfg.AllowedOrigin)
	return wrapAPIDocs(cfg.App.Env, application, apiDocsConfig{
		Title:    "Tidewise Admin Portal Service API",
		Document: adminapi.Document(),
	})
}

func NewServer(cfg adminconfig.RuntimeConfig, handler http.Handler) *http.Server {
	return newHTTPServer(cfg.Server, handler)
}
