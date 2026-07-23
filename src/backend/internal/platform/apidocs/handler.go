// Package apidocs delivers embedded OpenAPI contracts and Swagger UI.
package apidocs

import (
	"net/http"

	"github.com/meierlink88/tidewise-ai/backend/internal/platform/runtimeconfig"
	"github.com/swaggest/swgui"
	"github.com/swaggest/swgui/v5emb"
)

const (
	OpenAPIPath = "/openapi.yaml"
	DocsPath    = "/docs"
	DocsBase    = "/docs/"
)

type Config struct {
	Title    string
	Document []byte
}

// Wrap adds documentation routes outside production and preserves all
// application-owned routes.
func Wrap(environment runtimeconfig.Environment, application http.Handler, config Config) http.Handler {
	if application == nil {
		application = http.NotFoundHandler()
	}
	if environment == runtimeconfig.EnvProd {
		return application
	}

	ui := v5emb.NewWithConfig(swgui.Config{
		SettingsUI: map[string]string{
			"persistAuthorization": "false",
		},
	})(config.Title, OpenAPIPath, DocsBase)

	mux := http.NewServeMux()
	mux.HandleFunc("GET "+OpenAPIPath, func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write(config.Document)
	})
	mux.HandleFunc("GET "+DocsPath, func(response http.ResponseWriter, request *http.Request) {
		http.Redirect(response, request, DocsBase, http.StatusTemporaryRedirect)
	})
	mux.Handle(DocsBase, ui)
	mux.Handle("/", application)
	return mux
}
