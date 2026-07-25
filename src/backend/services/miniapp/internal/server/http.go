package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	kratosrecovery "github.com/go-kratos/kratos/v3/middleware/recovery"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"

	"github.com/meierlink88/tidewise-ai/backend/internal/platform/apidocs"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/apihttp"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/runtimeconfig"
	v1 "github.com/meierlink88/tidewise-ai/backend/services/miniapp/api/miniapp/v1"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/biz"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/conf"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/service"
)

type HealthResponse struct {
	Status      string                    `json:"status"`
	Service     string                    `json:"service"`
	Environment runtimeconfig.Environment `json:"environment"`
}

type ReadyResponse struct {
	Status      string                    `json:"status"`
	Service     string                    `json:"service"`
	Environment runtimeconfig.Environment `json:"environment"`
	Checks      map[string]string         `json:"checks"`
}

func NewHTTPServer(config conf.RuntimeConfig, research *service.ResearchService) *kratoshttp.Server {
	server := kratoshttp.NewServer(
		kratoshttp.Address(config.Server.Address()),
		kratoshttp.Timeout(time.Duration(config.Server.WriteTimeoutSeconds)*time.Second),
		kratoshttp.Filter(requestIDFilter()),
		kratoshttp.Middleware(kratosrecovery.Recovery()),
		kratoshttp.ResponseEncoder(responseEncoder),
		kratoshttp.ErrorEncoder(errorEncoder),
		kratoshttp.NotFoundHandler(http.HandlerFunc(notFoundHandler)),
		kratoshttp.MethodNotAllowedHandler(http.HandlerFunc(methodNotAllowedHandler)),
	)
	server.Server.ReadTimeout = time.Duration(config.Server.ReadTimeoutSeconds) * time.Second
	server.Server.WriteTimeout = time.Duration(config.Server.WriteTimeoutSeconds) * time.Second

	registerHealthRoutes(server, config.App)
	if research != nil {
		v1.RegisterResearchHTTPServer(server, research)
	}

	application := server.Server.Handler
	server.Server.Handler = apidocs.Wrap(config.App.Env, application, apidocs.Config{
		Title:    "Tidewise Miniapp Service API",
		Document: v1.Document(),
	})
	return server
}

func registerHealthRoutes(server *kratoshttp.Server, app runtimeconfig.AppConfig) {
	router := server.Route("/")
	router.GET("/healthz", func(ctx kratoshttp.Context) error {
		return ctx.JSON(http.StatusOK, HealthResponse{
			Status: "ok", Service: app.Name, Environment: app.Env,
		})
	})
	router.GET("/readyz", func(ctx kratoshttp.Context) error {
		return ctx.JSON(http.StatusOK, ReadyResponse{
			Status: "ready", Service: app.Name, Environment: app.Env,
			Checks: map[string]string{"config": "ok"},
		})
	})
}

func requestIDFilter() kratoshttp.FilterFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			requestID := apihttp.ResolveRequestID(request.Header.Get(apihttp.RequestIDHeader), "miniapp")
			request.Header.Set(apihttp.RequestIDHeader, requestID)
			response.Header().Set(apihttp.RequestIDHeader, requestID)
			next.ServeHTTP(response, request)
		})
	}
}

func responseEncoder(response http.ResponseWriter, request *http.Request, result any) error {
	return writeJSON(response, http.StatusOK, apihttp.Success(
		request.Header.Get(apihttp.RequestIDHeader),
		result,
	))
}

func errorEncoder(response http.ResponseWriter, request *http.Request, err error) {
	status, code, message := publicError(err)
	writeJSON(response, status, apihttp.Error(
		request.Header.Get(apihttp.RequestIDHeader),
		code,
		message,
		map[string]any{},
	))
}

func publicError(err error) (int, string, string) {
	switch {
	case errors.Is(err, v1.ErrInvalidRequest), errors.Is(err, biz.ErrInvalidResearchRequest):
		return http.StatusBadRequest, "INVALID_REQUEST", "invalid research request"
	case errors.Is(err, biz.ErrResearchNotFound):
		return http.StatusNotFound, "RESEARCH_RESULT_NOT_FOUND", biz.ErrResearchNotFound.Error()
	case errors.Is(err, biz.ErrResearchThemeNotFound):
		return http.StatusNotFound, "RESEARCH_THEME_NOT_FOUND", "research Theme was not found"
	case errors.Is(err, biz.ErrResearchReasoningTreesNotFound):
		return http.StatusNotFound, "RESEARCH_REASONING_TREES_NOT_FOUND", "research Theme has no published reasoning trees"
	case errors.Is(err, biz.ErrResearchReasoningTreeNotFound):
		return http.StatusNotFound, "RESEARCH_REASONING_TREE_NOT_FOUND", "research reasoning tree was not found for the Theme"
	case errors.Is(err, biz.ErrResearchDataService):
		return http.StatusInternalServerError, "RESEARCH_DATA_UNAVAILABLE", biz.ErrResearchDataService.Error()
	case errors.Is(err, biz.ErrResearchDataUnavailable):
		return http.StatusBadGateway, "RESEARCH_DATA_UNAVAILABLE", "research data is temporarily unavailable"
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error"
	}
}

func notFoundHandler(response http.ResponseWriter, request *http.Request) {
	writeJSON(response, http.StatusNotFound, apihttp.Error(
		request.Header.Get(apihttp.RequestIDHeader),
		"NOT_FOUND",
		"resource not found",
		map[string]any{},
	))
}

func methodNotAllowedHandler(response http.ResponseWriter, request *http.Request) {
	writeJSON(response, http.StatusMethodNotAllowed, apihttp.Error(
		request.Header.Get(apihttp.RequestIDHeader),
		"METHOD_NOT_ALLOWED",
		"method not allowed",
		map[string]any{},
	))
}

func writeJSON(response http.ResponseWriter, status int, value any) error {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	return json.NewEncoder(response).Encode(value)
}
