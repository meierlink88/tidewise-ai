package server

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/uuid"
	"github.com/swaggest/swgui/v5emb"

	v1 "github.com/meierlink88/tidewise-ai/agent-run/backend/api/agentrun/v1"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/conf"
)

const RequestIDHeader = "X-Request-ID"
const accessLoggedHeader = "X-AgentRun-Access-Logged"

const (
	operationHealth  = "/agentrun.v1.Operations/Health"
	operationReady   = "/agentrun.v1.Operations/Ready"
	operationOpenAPI = "/agentrun.v1.Operations/OpenAPI"
	operationDocs    = "/agentrun.v1.Operations/Docs"
)

type Readiness interface {
	Ready(context.Context) error
}

type successEnvelope struct {
	RequestID string `json:"request_id"`
	Result    any    `json:"result"`
}

type errorEnvelope struct {
	Error     errorDetail `json:"error"`
	RequestID string      `json:"request_id"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details"`
}

type directResponse struct {
	Status      int
	ContentType string
	Body        any
	Location    string
	Headers     http.Header
}

func NewHTTPServer(
	config conf.Config,
	apiService v1.AgentRunHTTPServer,
	readiness Readiness,
	logger *slog.Logger,
) *kratoshttp.Server {
	if logger == nil {
		logger = slog.Default()
	}
	server := kratoshttp.NewServer(
		kratoshttp.Address(config.Server.Address()),
		kratoshttp.Timeout(30*time.Second),
		kratoshttp.StrictSlash(false),
		kratoshttp.Filter(requestBoundaryFilter(logger, string(config.App.Env))),
		kratoshttp.Middleware(
			accessLogMiddleware(logger, string(config.App.Env)),
			authenticationMiddleware(config.Secrets.ServiceToken),
		),
		kratoshttp.ResponseEncoder(responseEncoder),
		kratoshttp.ErrorEncoder(errorEncoder),
		kratoshttp.NotFoundHandler(http.HandlerFunc(notFoundHandler)),
		kratoshttp.MethodNotAllowedHandler(http.HandlerFunc(methodNotAllowedHandler)),
	)
	server.Server.ReadHeaderTimeout = 5 * time.Second
	server.Server.ReadTimeout = 15 * time.Second
	server.Server.WriteTimeout = 30 * time.Second
	server.Server.IdleTimeout = 60 * time.Second

	registerOperationalRoutes(server, config, readiness)
	v1.RegisterAgentRunHTTPServer(server, apiService)
	registerDocumentation(server)
	return server
}

func registerOperationalRoutes(server *kratoshttp.Server, config conf.Config, readiness Readiness) {
	router := server.Route("/")
	router.GET("/healthz", func(ctx kratoshttp.Context) error {
		return callDirect(ctx, operationHealth, func(context.Context) directResponse {
			return directResponse{Status: http.StatusOK, Body: map[string]any{
				"status": "ok", "service": config.App.Name, "environment": config.App.Env,
			}}
		})
	})
	router.GET("/readyz", func(ctx kratoshttp.Context) error {
		return callDirect(ctx, operationReady, func(callContext context.Context) directResponse {
			if strings.TrimSpace(config.Secrets.ServiceToken) == "" ||
				readiness == nil || readiness.Ready(callContext) != nil {
				return directResponse{
					Status: http.StatusServiceUnavailable,
					Body:   map[string]any{"status": "not_ready"},
				}
			}
			return directResponse{Status: http.StatusOK, Body: map[string]any{"status": "ready"}}
		})
	})
}

func registerDocumentation(server *kratoshttp.Server) {
	router := server.Route("/")
	router.GET("/openapi.yaml", func(ctx kratoshttp.Context) error {
		return callDirect(ctx, operationOpenAPI, func(context.Context) directResponse {
			return directResponse{
				Status: http.StatusOK, ContentType: "application/yaml; charset=utf-8", Body: v1.Document(),
			}
		})
	})
	router.GET("/docs", func(ctx kratoshttp.Context) error {
		return callDirect(ctx, operationDocs, func(context.Context) directResponse {
			return directResponse{Status: http.StatusMovedPermanently, Location: "/docs/"}
		})
	})
	ui := v5emb.New("Tidewise AI AgentRun API", "/openapi.yaml", "/docs/")
	router.GET("/docs/{asset:.*}", func(ctx kratoshttp.Context) error {
		return callDirect(ctx, operationDocs, func(context.Context) directResponse {
			captured := newResponseBuffer()
			ui.ServeHTTP(captured, ctx.Request())
			return directResponse{
				Status: captured.status, ContentType: captured.header.Get("Content-Type"),
				Body: append([]byte(nil), captured.body.Bytes()...), Headers: captured.header.Clone(),
			}
		})
	})
}

func authenticationMiddleware(serviceToken string) middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, request any) (any, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, v1.NewPublicError(http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required", nil)
			}
			if isPublicOperation(tr.Operation()) {
				return next(ctx, request)
			}
			if !strings.HasPrefix(tr.Operation(), "/agentrun.v1.AgentRun/") ||
				serviceToken == "" ||
				tr.RequestHeader().Get("Authorization") != "Bearer "+serviceToken {
				return nil, v1.NewPublicError(http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required", nil)
			}
			return next(ctx, request)
		}
	}
}

func accessLogMiddleware(logger *slog.Logger, environment string) middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, request any) (any, error) {
			started := time.Now()
			operation := "unknown"
			requestID := ""
			if tr, ok := transport.FromServerContext(ctx); ok {
				operation = tr.Operation()
				requestID = tr.RequestHeader().Get(RequestIDHeader)
			}
			response, err := next(ctx, request)
			status := http.StatusOK
			var public *v1.PublicError
			if err != nil {
				status = http.StatusInternalServerError
				if errorsAsPublic(err, &public) {
					status = public.Status
				}
			} else {
				switch result := response.(type) {
				case directResponse:
					status = result.Status
				default:
					if operation == v1.OperationCreateCollectorRun {
						status = http.StatusAccepted
					}
				}
			}
			logger.InfoContext(ctx, "AgentRun request",
				"service", conf.ServiceName,
				"environment", environment,
				"operation", operation,
				"request_id", requestID,
				"status", status,
				"duration_ms", time.Since(started).Milliseconds(),
			)
			if tr, ok := transport.FromServerContext(ctx); ok {
				tr.RequestHeader().Set(accessLoggedHeader, "1")
			}
			return response, err
		}
	}
}

func callDirect(
	ctx kratoshttp.Context,
	operation string,
	build func(context.Context) directResponse,
) error {
	kratoshttp.SetOperation(ctx, operation)
	handler := ctx.Middleware(func(callContext context.Context, _ any) (any, error) {
		return build(callContext), nil
	})
	result, err := handler(ctx, nil)
	if err != nil {
		return err
	}
	response := result.(directResponse)
	for key, values := range response.Headers {
		for _, value := range values {
			ctx.Response().Header().Add(key, value)
		}
	}
	if response.Location != "" {
		ctx.Response().Header().Set("Location", response.Location)
		ctx.Response().WriteHeader(response.Status)
		return nil
	}
	if response.ContentType != "" {
		return ctx.Blob(response.Status, response.ContentType, response.Body.([]byte))
	}
	if content, ok := response.Body.([]byte); ok {
		return ctx.Blob(response.Status, "application/octet-stream", content)
	}
	return ctx.JSON(response.Status, response.Body)
}

func isPublicOperation(operation string) bool {
	switch operation {
	case operationHealth, operationReady, operationOpenAPI, operationDocs:
		return true
	default:
		return false
	}
}

func responseEncoder(writer http.ResponseWriter, request *http.Request, result any) error {
	writer.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(writer).Encode(successEnvelope{
		RequestID: request.Header.Get(RequestIDHeader),
		Result:    result,
	})
}

func errorEncoder(writer http.ResponseWriter, request *http.Request, err error) {
	status := http.StatusInternalServerError
	code := "INTERNAL_ERROR"
	message := "internal server error"
	details := any(map[string]any{})
	var public *v1.PublicError
	if errorsAsPublic(err, &public) {
		status = public.Status
		code = public.Code
		message = public.Message
		details = public.Details
	}
	_ = writeJSON(writer, status, errorEnvelope{
		Error:     errorDetail{Code: code, Message: message, Details: details},
		RequestID: request.Header.Get(RequestIDHeader),
	})
}

func notFoundHandler(writer http.ResponseWriter, request *http.Request) {
	_ = writeJSON(writer, http.StatusNotFound, errorEnvelope{
		Error:     errorDetail{Code: "NOT_FOUND", Message: "resource not found", Details: map[string]any{}},
		RequestID: request.Header.Get(RequestIDHeader),
	})
}

func methodNotAllowedHandler(writer http.ResponseWriter, request *http.Request) {
	_ = writeJSON(writer, http.StatusMethodNotAllowed, errorEnvelope{
		Error:     errorDetail{Code: "METHOD_NOT_ALLOWED", Message: "method not allowed", Details: map[string]any{}},
		RequestID: request.Header.Get(RequestIDHeader),
	})
}

func requestBoundaryFilter(logger *slog.Logger, environment string) kratoshttp.FilterFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			started := time.Now()
			requestID := resolveRequestID(request.Header.Get(RequestIDHeader))
			request.Header.Set(RequestIDHeader, requestID)
			request.Header.Del(accessLoggedHeader)
			buffer := newResponseBuffer()
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("AgentRun request panicked", "request_id", requestID)
					writer.Header().Set(RequestIDHeader, requestID)
					_ = writeJSON(writer, http.StatusInternalServerError, errorEnvelope{
						Error:     errorDetail{Code: "INTERNAL_ERROR", Message: "internal server error", Details: map[string]any{}},
						RequestID: requestID,
					})
					logBoundaryAccess(logger, environment, "HTTP_PANIC", requestID, http.StatusInternalServerError, started)
					return
				}
				buffer.flush(writer, requestID)
				if request.Header.Get(accessLoggedHeader) != "1" {
					switch buffer.status {
					case http.StatusNotFound:
						logBoundaryAccess(logger, environment, "HTTP_NOT_FOUND", requestID, buffer.status, started)
					case http.StatusMethodNotAllowed:
						logBoundaryAccess(logger, environment, "HTTP_METHOD_NOT_ALLOWED", requestID, buffer.status, started)
					}
				}
			}()
			next.ServeHTTP(buffer, request)
		})
	}
}

func logBoundaryAccess(
	logger *slog.Logger,
	environment string,
	operation string,
	requestID string,
	status int,
	started time.Time,
) {
	logger.Info(
		"AgentRun request",
		"service", conf.ServiceName,
		"environment", environment,
		"operation", operation,
		"request_id", requestID,
		"status", status,
		"duration_ms", time.Since(started).Milliseconds(),
	)
}

func resolveRequestID(value string) string {
	value = strings.TrimSpace(value)
	if value != "" && len(value) <= 128 && isSafeRequestID(value) {
		return value
	}
	return "agentrun-" + uuid.NewString()
}

func isSafeRequestID(value string) bool {
	for _, character := range value {
		if character >= 'a' && character <= 'z' ||
			character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' ||
			strings.ContainsRune("-._:", character) {
			continue
		}
		return false
	}
	return true
}

func writeJSON(writer http.ResponseWriter, status int, value any) error {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	return json.NewEncoder(writer).Encode(value)
}

type responseBuffer struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newResponseBuffer() *responseBuffer {
	return &responseBuffer{header: make(http.Header), status: http.StatusOK}
}

func (b *responseBuffer) Header() http.Header {
	return b.header
}

func (b *responseBuffer) WriteHeader(status int) {
	b.status = status
}

func (b *responseBuffer) Write(content []byte) (int, error) {
	return b.body.Write(content)
}

func (b *responseBuffer) flush(writer http.ResponseWriter, requestID string) {
	for key, values := range b.header {
		for _, value := range values {
			writer.Header().Add(key, value)
		}
	}
	writer.Header().Set(RequestIDHeader, requestID)
	writer.WriteHeader(b.status)
	_, _ = writer.Write(b.body.Bytes())
}

func errorsAsPublic(err error, target **v1.PublicError) bool {
	for err != nil {
		if public, ok := err.(*v1.PublicError); ok {
			*target = public
			return true
		}
		type unwrapper interface{ Unwrap() error }
		wrapped, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = wrapped.Unwrap()
	}
	return false
}
