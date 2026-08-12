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

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	eventapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1/event"
	eventsemanticapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1/eventsemantic"
	evidenceapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1/evidence"
	rawdocumentapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1/rawdocument"
	researchapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1/research"
	runtimehealthapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1/runtimehealth"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/conf"
)

const ServiceName = conf.ServiceName

type healthResponse struct {
	Status      string            `json:"status"`
	Service     string            `json:"service"`
	Environment conf.Environment  `json:"environment"`
	Checks      map[string]string `json:"checks,omitempty"`
}

func NewHTTPServer(config conf.Config, runtimeHealthApplication runtimehealthapi.Service, researchApplication researchapi.Service, eventApplication eventapi.Service, eventSemanticApplication eventsemanticapi.Service, evidenceApplication evidenceapi.Service, rawDocumentApplication rawdocumentapi.Service, authenticator *Authenticator, logger *slog.Logger) (*kratoshttp.Server, error) {
	if runtimeHealthApplication == nil {
		return nil, errors.New("Runtime Health API service is required")
	}
	if evidenceApplication == nil {
		return nil, errors.New("Evidence API service is required")
	}
	if researchApplication == nil {
		return nil, errors.New("Research API service is required")
	}
	if eventApplication == nil {
		return nil, errors.New("Event API service is required")
	}
	if eventSemanticApplication == nil {
		return nil, errors.New("Event Semantic API service is required")
	}
	if rawDocumentApplication == nil {
		return nil, errors.New("RawDocument API service is required")
	}
	if authenticator == nil {
		return nil, errors.New("Data API authenticator is required")
	}
	server := kratoshttp.NewServer(
		kratoshttp.Address(config.Server.Address()),
		kratoshttp.Timeout(0),
		kratoshttp.StrictSlash(false),
		kratoshttp.Middleware(
			sanitizedLoggingMiddleware(config.App, logger),
			authenticationMiddleware(authenticator),
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
	runtimehealthapi.RegisterHTTPServer(server, runtimeHealthApplication)
	researchapi.RegisterHTTPServer(server, researchApplication)
	eventapi.RegisterHTTPServer(server, eventApplication)
	eventsemanticapi.RegisterHTTPServer(server, eventSemanticApplication)
	evidenceapi.RegisterHTTPServer(server, evidenceApplication)
	rawdocumentapi.RegisterHTTPServer(server, rawDocumentApplication)

	documented := wrapAPIDocs(config.App.Env, server.Server.Handler, apiDocsConfig{
		Title:    "Tidewise Data Service API",
		Document: v1.Document(),
	})
	server.Server.Handler = observabilityFilter(config.App, logger)(documented)
	return server, nil
}

func registerHealthRoutes(server *kratoshttp.Server, app conf.AppConfig) {
	serviceName := app.Name
	if serviceName == "" {
		serviceName = conf.ServiceName
	}
	router := server.Route("/")
	router.GET("/healthz", func(ctx kratoshttp.Context) error {
		return operationalResponse(ctx, operationHealth, healthResponse{
			Status: "ok", Service: serviceName, Environment: app.Env,
		})
	})
	router.GET("/readyz", func(ctx kratoshttp.Context) error {
		return operationalResponse(ctx, operationReady, healthResponse{
			Status: "ready", Service: serviceName, Environment: app.Env,
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

func observabilityFilter(app conf.AppConfig, logger *slog.Logger) kratoshttp.FilterFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			startedAt := time.Now()
			requestID := resolveRequestID(request.Header.Get("X-Request-ID"))
			request.Header.Set("X-Request-ID", requestID)
			response.Header().Set("X-Request-ID", requestID)
			recorder := &responseStatusWriter{ResponseWriter: response}
			defer func() {
				if recovered := recover(); recovered != nil {
					if !recorder.wroteHeader {
						writeError(recorder, requestID, http.StatusInternalServerError, "INTERNAL_ERROR", "internal data service error")
					} else {
						recorder.status = http.StatusInternalServerError
					}
				}
				if logger != nil {
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
			}()
			next.ServeHTTP(recorder, request)
		})
	}
}

func sanitizedLoggingMiddleware(app conf.AppConfig, logger *slog.Logger) middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, request any) (any, error) {
			reply, err := next(ctx, request)
			if err != nil && logger != nil {
				operation := "data.unknown"
				requestID := ""
				if serverTransport, ok := transport.FromServerContext(ctx); ok {
					operation = serverTransport.Operation()
					requestID = serverTransport.RequestHeader().Get("X-Request-ID")
				}
				logger.WarnContext(ctx, "business request failed",
					slog.String("service", app.Name),
					slog.String("environment", string(app.Env)),
					slog.String("operation", operation),
					slog.String("request_id", requestID),
				)
			}
			return reply, err
		}
	}
}

func resolveRequestID(value string) string {
	value = strings.TrimSpace(value)
	if value != "" && len(value) <= 128 {
		return value
	}
	return "data-" + time.Now().UTC().Format("20060102T150405.000000000")
}

func operationForRequest(request *http.Request) string {
	switch request.URL.Path {
	case "/healthz":
		return "data.health"
	case "/readyz":
		return "data.ready"
	case "/docs", "/openapi.yaml":
		return "data.docs"
	}
	if strings.HasPrefix(request.URL.Path, "/docs/") {
		return "data.docs"
	}
	return "data.v1"
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

func notFoundHandler(response http.ResponseWriter, request *http.Request) {
	writeError(response, request.Header.Get("X-Request-ID"), http.StatusNotFound, "NOT_FOUND", "resource not found")
}

func methodNotAllowedHandler(response http.ResponseWriter, request *http.Request) {
	writeError(response, request.Header.Get("X-Request-ID"), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
}

func responseEncoder(response http.ResponseWriter, request *http.Request, result any) error {
	response.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(response).Encode(map[string]any{
		"request_id": request.Header.Get("X-Request-ID"),
		"result":     result,
	})
}

func errorEncoder(response http.ResponseWriter, request *http.Request, err error) {
	var public *v1.PublicError
	if errors.As(err, &public) {
		writeError(response, request.Header.Get("X-Request-ID"), public.Status, public.Code, public.Message, public.Details)
		return
	}
	writeError(response, request.Header.Get("X-Request-ID"), http.StatusInternalServerError, "INTERNAL_ERROR", "internal data service error", map[string]any{})
}

func writeError(response http.ResponseWriter, requestID string, status int, code, message string, details ...any) {
	errorDetails := any(map[string]any{})
	if len(details) > 0 && details[0] != nil {
		errorDetails = details[0]
	}
	_ = writeJSON(response, status, map[string]any{
		"request_id": requestID,
		"error": map[string]any{
			"code": code, "message": message, "details": errorDetails,
		},
	})
}

func writeJSON(response http.ResponseWriter, status int, value any) error {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	return json.NewEncoder(response).Encode(value)
}
