// Package servicehttp provides shared, business-free HTTP process primitives.
package servicehttp

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/config"
)

type healthResponse struct {
	Status      string             `json:"status"`
	Service     string             `json:"service"`
	Environment config.Environment `json:"environment"`
	Checks      map[string]string  `json:"checks,omitempty"`
}

// NewHealthHandler returns the service skeleton's liveness and readiness facade.
func NewHealthHandler(service string, environment config.Environment) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			response.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeJSON(response, healthResponse{
			Status:      "ok",
			Service:     service,
			Environment: environment,
		})
	})
	mux.HandleFunc("/readyz", func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			response.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeJSON(response, healthResponse{
			Status:      "ready",
			Service:     service,
			Environment: environment,
			Checks:      map[string]string{"config": "ok"},
		})
	})
	return mux
}

// NewServer applies common process settings while preserving an injected
// compatibility handler byte-for-byte during the service-boundary transition.
func NewServer(cfg config.Config, service string, compatibilityHandler http.Handler) *http.Server {
	handler := compatibilityHandler
	if handler == nil {
		handler = NewHealthHandler(service, cfg.App.Env)
	}
	return &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      handler,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSeconds) * time.Second,
	}
}

func writeJSON(response http.ResponseWriter, payload healthResponse) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(response).Encode(payload)
}
