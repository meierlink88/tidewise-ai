package server

import (
	"net/http"

	"github.com/swaggest/swgui"
	"github.com/swaggest/swgui/v5emb"

	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
)

const (
	openAPIPath = "/openapi.yaml"
	docsPath    = "/docs"
	docsBase    = "/docs/"
)

type apiDocsConfig struct {
	Title    string
	Document []byte
}

func wrapAPIDocs(environment conf.Environment, application http.Handler, docs apiDocsConfig) http.Handler {
	if application == nil {
		application = http.NotFoundHandler()
	}
	if environment == conf.EnvProd {
		return application
	}

	ui := v5emb.NewWithConfig(swgui.Config{
		SettingsUI: map[string]string{"persistAuthorization": "false"},
	})(docs.Title, openAPIPath, docsBase)

	mux := http.NewServeMux()
	mux.HandleFunc("GET "+openAPIPath, func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write(docs.Document)
	})
	mux.HandleFunc("GET "+docsPath, func(response http.ResponseWriter, request *http.Request) {
		http.Redirect(response, request, docsBase, http.StatusTemporaryRedirect)
	})
	mux.Handle(docsBase, ui)
	mux.Handle("/", application)
	return mux
}
