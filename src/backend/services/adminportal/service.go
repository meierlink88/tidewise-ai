// Package adminportal owns the Admin Portal BFF HTTP process boundary.
package adminportal

import (
	"net/http"

	"github.com/meierlink88/tidewise-ai/backend/internal/apps/adminapi"
	"github.com/meierlink88/tidewise-ai/backend/internal/config"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/servicehttp"
	"github.com/meierlink88/tidewise-ai/backend/services/adminportal/dataclient"
)

const ServiceName = "adminportal"

// NewHandler composes the Admin BFF exclusively through its DataServiceClient.
func NewHandler(cfg config.Config, client dataclient.DataServiceClient, adminToken string) http.Handler {
	cfg.App.Name = ServiceName
	return adminapi.NewRouter(cfg, client, adminToken)
}

// NewServer preserves an injected legacy handler during the compatibility window.
func NewServer(cfg config.Config, compatibilityHandler http.Handler) *http.Server {
	return servicehttp.NewServer(cfg, ServiceName, compatibilityHandler)
}
