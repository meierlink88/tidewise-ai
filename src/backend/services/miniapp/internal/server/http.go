package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/middleware"
	kratosrecovery "github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/transport"
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

func NewHTTPServer(config conf.RuntimeConfig, research *service.ResearchService, logger *slog.Logger) *kratoshttp.Server {
	server := kratoshttp.NewServer(
		kratoshttp.Address(config.Server.Address()),
		kratoshttp.Timeout(0),
		kratoshttp.StrictSlash(false),
		kratoshttp.Middleware(
			sanitizedLoggingMiddleware(config.App, logger),
			kratosrecovery.Recovery(
				kratosrecovery.WithLogger(slog.New(slog.NewJSONHandler(io.Discard, nil))),
			),
		),
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
	documentedApplication := apidocs.Wrap(config.App.Env, application, apidocs.Config{
		Title:    "Tidewise Miniapp Service API",
		Document: v1.Document(),
	})
	server.Server.Handler = observabilityFilter(config.App, logger)(documentedApplication)
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

func observabilityFilter(app runtimeconfig.AppConfig, logger *slog.Logger) kratoshttp.FilterFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			startedAt := time.Now()
			requestID := apihttp.ResolveRequestID(request.Header.Get(apihttp.RequestIDHeader), "miniapp")
			request.Header.Set(apihttp.RequestIDHeader, requestID)
			response.Header().Set(apihttp.RequestIDHeader, requestID)
			recorder := &responseStatusWriter{ResponseWriter: response}
			defer func() {
				if recovered := recover(); recovered != nil {
					if !recorder.wroteHeader {
						_ = writeJSON(recorder, http.StatusInternalServerError, apihttp.Error(
							requestID,
							"INTERNAL_ERROR",
							"internal server error",
							map[string]any{},
						))
					} else {
						recorder.status = http.StatusInternalServerError
					}
				}
				status := recorder.status
				if status == 0 {
					status = http.StatusOK
				}
				if logger != nil {
					logger.InfoContext(request.Context(), "http access",
						slog.String("service", app.Name),
						slog.String("environment", string(app.Env)),
						slog.String("operation", operationForRequest(request)),
						slog.String("request_id", requestID),
						slog.Int("status", status),
						slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
					)
				}
			}()
			next.ServeHTTP(recorder, request)
		})
	}
}

func sanitizedLoggingMiddleware(app runtimeconfig.AppConfig, logger *slog.Logger) middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, request any) (any, error) {
			reply, err := next(ctx, request)
			if logger == nil {
				return reply, err
			}
			operation := "miniapp.unknown"
			requestID := ""
			if serverTransport, ok := transport.FromServerContext(ctx); ok {
				operation = serverTransport.Operation()
				requestID = serverTransport.RequestHeader().Get(apihttp.RequestIDHeader)
			}
			if err != nil {
				status, code, _ := publicError(err)
				logger.WarnContext(ctx, "business request failed",
					slog.String("service", app.Name),
					slog.String("environment", string(app.Env)),
					slog.String("operation", operation),
					slog.String("request_id", requestID),
					slog.Int("status", status),
					slog.String("error_code", code),
				)
			} else {
				logger.DebugContext(ctx, "business request completed",
					slog.String("service", app.Name),
					slog.String("environment", string(app.Env)),
					slog.String("operation", operation),
					slog.String("request_id", requestID),
					slog.Int("status", http.StatusOK),
				)
			}
			return reply, err
		}
	}
}

func operationForRequest(request *http.Request) string {
	switch request.URL.Path {
	case "/healthz":
		return "miniapp.health"
	case "/readyz":
		return "miniapp.ready"
	case "/docs", "/openapi.yaml":
		return "miniapp.docs"
	case v1.APIPrefix + "/research/themes":
		return "miniapp.research.listThemes"
	}
	if strings.HasPrefix(request.URL.Path, "/docs/") {
		return "miniapp.docs"
	}
	const prefix = v1.APIPrefix + "/research/themes/"
	if !strings.HasPrefix(request.URL.Path, prefix) {
		return "miniapp.unknown"
	}
	segments := strings.Split(strings.TrimPrefix(request.URL.Path, prefix), "/")
	switch {
	case len(segments) == 1 && segments[0] != "":
		return "miniapp.research.getTheme"
	case len(segments) == 2 && segments[0] != "" && segments[1] == "reasoning-trees":
		return "miniapp.research.listReasoningTrees"
	case len(segments) == 3 && segments[0] != "" && segments[1] == "reasoning-trees" && segments[2] != "":
		return "miniapp.research.getReasoningTree"
	default:
		return "miniapp.unknown"
	}
}

type responseStatusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *responseStatusWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseStatusWriter) Write(payload []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(payload)
}

func (w *responseStatusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
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
