// Package adminportal owns the Admin Portal BFF HTTP process boundary.
package adminportal

import (
	"net/http"

	"github.com/meierlink88/tidewise-ai/backend/internal/platform/servicehttp"
	adminconfig "github.com/meierlink88/tidewise-ai/backend/services/adminportal/config"
	"github.com/meierlink88/tidewise-ai/backend/services/adminportal/dataclient"
	"github.com/meierlink88/tidewise-ai/backend/services/adminportal/transport"
	"github.com/meierlink88/tidewise-ai/backend/services/adminportal/usecase"
)

const ServiceName = adminconfig.ServiceName

// NewHandler composes the Admin BFF exclusively through its DataServiceClient.
func NewHandler(cfg adminconfig.RuntimeConfig, client dataclient.DataServiceClient, adminToken string) http.Handler {
	cfg.App.Name = ServiceName
	return transport.NewRouter(cfg.App, usecase.NewService(client), adminToken)
}

func NewServer(cfg adminconfig.RuntimeConfig, handler http.Handler) *http.Server {
	return servicehttp.NewServer(cfg.Server, ServiceName, cfg.App.Env, handler)
}
