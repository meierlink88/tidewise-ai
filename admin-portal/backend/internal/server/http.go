package server

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/middleware"
	kratosrecovery "github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/transport"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"

	v1 "github.com/meierlink88/tidewise-ai/admin-portal/backend/api/admin/v1"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/service"
)

type HealthResponse struct {
	Status      string           `json:"status"`
	Service     string           `json:"service"`
	Environment conf.Environment `json:"environment"`
}

type ReadyResponse struct {
	Status      string            `json:"status"`
	Service     string            `json:"service"`
	Environment conf.Environment  `json:"environment"`
	Checks      map[string]string `json:"checks"`
}

func NewHTTPServer(
	config conf.RuntimeConfig,
	admin *service.AdminService,
	logger *slog.Logger,
) *kratoshttp.Server {
	httpServer := kratoshttp.NewServer(
		kratoshttp.Address(config.Server.Address()),
		kratoshttp.Timeout(0),
		kratoshttp.StrictSlash(false),
		kratoshttp.Middleware(
			sanitizedLoggingMiddleware(config.App, logger),
			adminAuthenticationMiddleware(config.AdminToken),
			kratosrecovery.Recovery(
				kratosrecovery.WithLogger(slog.New(slog.NewJSONHandler(io.Discard, nil))),
			),
		),
		kratoshttp.ResponseEncoder(responseEncoder),
		kratoshttp.ErrorEncoder(errorEncoder),
		kratoshttp.NotFoundHandler(http.HandlerFunc(notFoundHandler)),
		kratoshttp.MethodNotAllowedHandler(http.HandlerFunc(methodNotAllowedHandler)),
	)
	httpServer.Server.ReadTimeout = time.Duration(config.Server.ReadTimeoutSeconds) * time.Second
	httpServer.Server.WriteTimeout = time.Duration(config.Server.WriteTimeoutSeconds) * time.Second

	registerHealthRoutes(httpServer, config.App)
	if admin != nil {
		v1.RegisterAdminHTTPServer(httpServer, admin)
	}

	application := httpServer.Server.Handler
	documentedApplication := wrapAPIDocs(config.App.Env, application, apiDocsConfig{
		Title:    "Tidewise Admin Portal Service API",
		Document: v1.Document(),
	})
	httpServer.Server.Handler = observabilityAndSecurityFilter(config, logger)(documentedApplication)
	return httpServer
}

func registerHealthRoutes(httpServer *kratoshttp.Server, app conf.AppConfig) {
	router := httpServer.Route("/")
	router.GET("/healthz", func(ctx kratoshttp.Context) error {
		return operationalResponse(ctx, "admin.health", HealthResponse{
			Status: "ok", Service: app.Name, Environment: app.Env,
		})
	})
	router.GET("/readyz", func(ctx kratoshttp.Context) error {
		return operationalResponse(ctx, "admin.ready", ReadyResponse{
			Status: "ready", Service: app.Name, Environment: app.Env,
			Checks: map[string]string{"config": "ok"},
		})
	})
}

func operationalResponse(ctx kratoshttp.Context, operation string, response any) error {
	kratoshttp.SetOperation(ctx, operation)
	handler := ctx.Middleware(func(context.Context, any) (any, error) {
		return response, nil
	})
	result, err := handler(ctx, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, result)
}

func observabilityAndSecurityFilter(
	config conf.RuntimeConfig,
	logger *slog.Logger,
) kratoshttp.FilterFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			startedAt := time.Now()
			requestID := resolveRequestID(request.Header.Get(requestIDHeader), "admin")
			request.Header.Set(requestIDHeader, requestID)
			response.Header().Set(requestIDHeader, requestID)
			recorder := &responseStatusWriter{ResponseWriter: response}
			defer func() {
				if recovered := recover(); recovered != nil {
					if !recorder.wroteHeader {
						_ = writeJSON(recorder, http.StatusInternalServerError, errorResponse(
							requestID,
							"INTERNAL_ERROR",
							"internal server error",
							map[string]any{},
						))
					} else {
						recorder.status = http.StatusInternalServerError
					}
				}
				logAccess(config.App, logger, request, requestID, recorder, startedAt)
			}()

			if strings.HasPrefix(request.URL.Path, v1.APIPrefix) {
				if !applyCORS(recorder, request, requestID, config.AllowedOrigin) {
					return
				}
				if request.Method == http.MethodOptions {
					recorder.WriteHeader(http.StatusNoContent)
					return
				}
			}
			next.ServeHTTP(recorder, request)
		})
	}
}

func applyCORS(
	response http.ResponseWriter,
	request *http.Request,
	requestID string,
	allowedOrigin string,
) bool {
	origin := strings.TrimSpace(request.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	if allowedOrigin == "" || origin != allowedOrigin {
		_ = writeJSON(response, http.StatusForbidden, errorResponse(
			requestID, "FORBIDDEN", "origin is not allowed", map[string]any{},
		))
		return false
	}
	response.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
	response.Header().Set("Access-Control-Allow-Methods", "GET, PUT, PATCH, OPTIONS")
	response.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
	response.Header().Set("Access-Control-Max-Age", "600")
	response.Header().Set("Vary", "Origin")
	return true
}

func logAccess(
	app conf.AppConfig,
	logger *slog.Logger,
	request *http.Request,
	requestID string,
	recorder *responseStatusWriter,
	startedAt time.Time,
) {
	if logger == nil {
		return
	}
	status := recorder.status
	if status == 0 {
		status = http.StatusOK
	}
	logger.InfoContext(request.Context(), "http access",
		slog.String("service", app.Name),
		slog.String("environment", string(app.Env)),
		slog.String("operation", operationForRequest(request)),
		slog.String("request_id", requestID),
		slog.Int("status", status),
		slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
	)
}

func adminAuthenticationMiddleware(adminToken string) middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, request any) (any, error) {
			httpContext, ok := ctx.(kratoshttp.Context)
			if !ok || !strings.HasPrefix(httpContext.Request().URL.Path, v1.APIPrefix) {
				return next(ctx, request)
			}
			if adminToken == "" {
				return nil, v1.NewHTTPError(
					http.StatusServiceUnavailable,
					"ADMIN_NOT_CONFIGURED",
					"admin token is not configured",
				)
			}
			header := httpContext.Request().Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") ||
				subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(header, "Bearer ")), []byte(adminToken)) != 1 {
				return nil, v1.NewHTTPError(
					http.StatusUnauthorized,
					"UNAUTHENTICATED",
					"valid admin identity is required",
				)
			}
			return next(ctx, request)
		}
	}
}

func sanitizedLoggingMiddleware(app conf.AppConfig, logger *slog.Logger) middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, request any) (any, error) {
			reply, err := next(ctx, request)
			if logger == nil {
				return reply, err
			}
			operation := "admin.unknown"
			requestID := ""
			if serverTransport, ok := transport.FromServerContext(ctx); ok {
				operation = serverTransport.Operation()
				requestID = serverTransport.RequestHeader().Get(requestIDHeader)
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
			}
			return reply, err
		}
	}
}

func operationForRequest(request *http.Request) string {
	switch request.URL.Path {
	case "/healthz":
		return "admin.health"
	case "/readyz":
		return "admin.ready"
	case "/docs", "/openapi.yaml":
		return "admin.docs"
	case v1.APIPrefix + "/events":
		return v1.OperationListEvents
	case v1.APIPrefix + "/runtime-health":
		return v1.OperationGetRuntimeHealth
	}
	if strings.HasPrefix(request.URL.Path, "/docs/") {
		return "admin.docs"
	}
	if !strings.HasPrefix(request.URL.Path, v1.APIPrefix+"/") {
		return "admin.unknown"
	}
	return "admin.unknown"
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
	return writeJSON(response, http.StatusOK, successResponse(
		request.Header.Get(requestIDHeader),
		result,
	))
}

func errorEncoder(response http.ResponseWriter, request *http.Request, err error) {
	status, code, message := publicError(err)
	_ = writeJSON(response, status, errorResponse(
		request.Header.Get(requestIDHeader),
		code,
		message,
		map[string]any{},
	))
}

func publicError(err error) (int, string, string) {
	if public, ok := v1.PublicError(err); ok {
		return public.Status(), public.Code(), public.Message()
	}
	return http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error"
}

func notFoundHandler(response http.ResponseWriter, request *http.Request) {
	_ = writeJSON(response, http.StatusNotFound, errorResponse(
		request.Header.Get(requestIDHeader),
		"NOT_FOUND",
		"resource not found",
		map[string]any{},
	))
}

func methodNotAllowedHandler(response http.ResponseWriter, request *http.Request) {
	_ = writeJSON(response, http.StatusMethodNotAllowed, errorResponse(
		request.Header.Get(requestIDHeader),
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
