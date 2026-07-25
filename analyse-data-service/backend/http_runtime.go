package data

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/config"
	"github.com/swaggest/swgui"
	"github.com/swaggest/swgui/v5emb"
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

type healthResponse struct {
	Status      string             `json:"status"`
	Service     string             `json:"service"`
	Environment config.Environment `json:"environment"`
	Checks      map[string]string  `json:"checks,omitempty"`
}

func wrapAPIDocs(environment config.Environment, application http.Handler, docs apiDocsConfig) http.Handler {
	if application == nil {
		application = http.NotFoundHandler()
	}
	if environment == config.EnvProd {
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

func newHealthHandler(service string, environment config.Environment) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			response.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeOperationalJSON(response, healthResponse{
			Status: "ok", Service: service, Environment: environment,
		})
	})
	mux.HandleFunc("/readyz", func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			response.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeOperationalJSON(response, healthResponse{
			Status: "ready", Service: service, Environment: environment,
			Checks: map[string]string{"config": "ok"},
		})
	})
	return mux
}

func newHTTPServer(cfg config.ServerConfig, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         cfg.Address(),
		Handler:      handler,
		ReadTimeout:  time.Duration(cfg.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeoutSeconds) * time.Second,
	}
}

func writeOperationalJSON(response http.ResponseWriter, payload healthResponse) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(response).Encode(payload)
}
