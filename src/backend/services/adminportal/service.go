// Package adminportal owns the Admin Portal BFF HTTP process boundary.
package adminportal

import (
	"net/http"

	"github.com/meierlink88/tidewise-ai/backend/internal/platform/apidocs"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/servicehttp"
	adminapi "github.com/meierlink88/tidewise-ai/backend/services/adminportal/api"
	adminconfig "github.com/meierlink88/tidewise-ai/backend/services/adminportal/config"
	"github.com/meierlink88/tidewise-ai/backend/services/adminportal/dataclient"
	"github.com/meierlink88/tidewise-ai/backend/services/adminportal/transport"
	"github.com/meierlink88/tidewise-ai/backend/services/adminportal/usecase"
)

const ServiceName = adminconfig.ServiceName

// NewHandler composes the Admin BFF exclusively through its DataServiceClient.
func NewHandler(cfg adminconfig.RuntimeConfig, client dataclient.DataServiceClient, adminToken string) http.Handler {
	cfg.App.Name = ServiceName
	application := transport.NewRouter(cfg.App, usecase.NewService(client), adminToken, cfg.AllowedOrigin)
	return apidocs.Wrap(cfg.App.Env, application, apidocs.Config{
		Title:    "Tidewise Admin Portal Service API",
		Document: adminapi.Document(),
	})
}

func NewServer(cfg adminconfig.RuntimeConfig, handler http.Handler) *http.Server {
	return servicehttp.NewServer(cfg.Server, ServiceName, cfg.App.Env, handler)
}
