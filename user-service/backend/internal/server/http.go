package server

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/uuid"
	api "github.com/meierlink88/tidewise-ai/user-service/backend/api/user/v1/identity"
)

func write(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func failure(w http.ResponseWriter, status int, code string) {
	write(w, status, map[string]any{"request_id": w.Header().Get("X-Request-ID"), "error": map[string]string{"code": code, "message": code}})
}

// One bounded, process-local budget; no unbounded per-IP/user map. Scale limits
// at the trusted gateway when deploying multiple replicas.
type loginBudget struct {
	mu     sync.Mutex
	window time.Time
	count  int
}

func (b *loginBudget) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	if now.Sub(b.window) >= time.Minute {
		b.window = now
		b.count = 0
	}
	if b.count >= 60 {
		return false
	}
	b.count++
	return true
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *responseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	return w.ResponseWriter.Write(b)
}
func operation(path string) string {
	switch path {
	case "/healthz":
		return "health"
	case "/readyz":
		return "ready"
	case "/api/user/v1/wechat/logins":
		return "login"
	case "/api/user/v1/sessions/verify":
		return "verify"
	case "/api/user/v1/profiles/avatar":
		return "avatar"
	case "/api/user/v1/profiles/nickname":
		return "nickname"
	case "/api/user/v1/watchlist/list", "/api/user/v1/watchlist/check", "/api/user/v1/watchlist/add", "/api/user/v1/watchlist/remove":
		return "watchlist"
	case "/api/user/v1/sessions/revoke":
		return "revoke"
	default:
		return "unknown"
	}
}
func New(address, token string, service api.Service, ready func(context.Context) error, logger *slog.Logger) *kratoshttp.Server {
	if service == nil || ready == nil || logger == nil || len(token) < 32 {
		panic("invalid User HTTP dependencies")
	}
	server := kratoshttp.NewServer(kratoshttp.Address(address), kratoshttp.Timeout(0), kratoshttp.StrictSlash(false),
		kratoshttp.ResponseEncoder(func(w http.ResponseWriter, r *http.Request, v any) error {
			write(w, 200, map[string]any{"request_id": w.Header().Get("X-Request-ID"), "result": v})
			return nil
		}),
		kratoshttp.ErrorEncoder(func(w http.ResponseWriter, r *http.Request, err error) {
			var apiErr *api.Error
			if errors.As(err, &apiErr) {
				failure(w, apiErr.Status, apiErr.Code)
			} else {
				failure(w, 500, "INTERNAL_ERROR")
			}
		}),
		kratoshttp.NotFoundHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { failure(w, 404, "NOT_FOUND") })),
		kratoshttp.MethodNotAllowedHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { failure(w, 405, "METHOD_NOT_ALLOWED") })))
	api.RegisterHTTPServer(server, service)
	server.Route("/").GET("/healthz", func(ctx kratoshttp.Context) error { return ctx.JSON(200, map[string]string{"status": "ok"}) })
	server.Route("/").GET("/readyz", func(ctx kratoshttp.Context) error {
		if ready(ctx) != nil {
			return &api.Error{Status: 503, Code: "USER_SERVICE_UNAVAILABLE"}
		}
		return ctx.JSON(200, map[string]string{"status": "ready"})
	})
	inner := server.Server.Handler
	expected := sha256.Sum256([]byte("Bearer " + token))
	budget := &loginBudget{}
	server.Server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &responseWriter{ResponseWriter: w}
		w = recorder
		w.Header().Set("X-Request-ID", uuid.NewString())
		w.Header().Set("Cache-Control", "no-store")
		started := time.Now()
		defer func() {
			if recover() != nil && recorder.status == 0 {
				failure(w, 500, "INTERNAL_ERROR")
			}
			logger.Info("user request completed", "request_id", w.Header().Get("X-Request-ID"), "service", "user", "operation", operation(r.URL.Path), "status", recorder.status, "elapsed_ms", time.Since(started).Milliseconds())
		}()
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		r = r.WithContext(ctx)
		if r.URL.Path != "/healthz" && r.URL.Path != "/readyz" {
			actual := sha256.Sum256([]byte(r.Header.Get("Authorization")))
			if subtle.ConstantTimeCompare(actual[:], expected[:]) != 1 {
				failure(w, 401, "SERVICE_UNAUTHENTICATED")
				return
			}
			if (r.URL.Path == "/api/user/v1/wechat/logins" || r.URL.Path == "/api/user/v1/profiles/nickname") && !budget.allow() {
				w.Header().Set("Retry-After", "60")
				failure(w, 429, "LOGIN_RATE_LIMITED")
				return
			}
		}
		inner.ServeHTTP(w, r)
	})
	server.Server.ReadHeaderTimeout = 5 * time.Second
	server.Server.ReadTimeout = 10 * time.Second
	server.Server.WriteTimeout = 15 * time.Second
	server.Server.IdleTimeout = 60 * time.Second
	server.Server.MaxHeaderBytes = 16384
	return server
}
