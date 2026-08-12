package v1

import (
	"context"
	"fmt"
	"net/http"
)

const (
	StatusOK                    = http.StatusOK
	StatusCreated               = http.StatusCreated
	StatusBadRequest            = http.StatusBadRequest
	StatusUnauthorized          = http.StatusUnauthorized
	StatusForbidden             = http.StatusForbidden
	StatusNotFound              = http.StatusNotFound
	StatusConflict              = http.StatusConflict
	StatusRequestEntityTooLarge = http.StatusRequestEntityTooLarge
	StatusUnprocessableEntity   = http.StatusUnprocessableEntity
	StatusTooManyRequests       = http.StatusTooManyRequests
	StatusInternalServerError   = http.StatusInternalServerError
	StatusServiceUnavailable    = http.StatusServiceUnavailable
)

type Response[T any] struct {
	Status int
	Result T
}

type RuntimeHealthRequest struct{}

type RuntimeHealth struct {
	CheckedAt string                 `json:"checked_at"`
	Services  []RuntimeHealthService `json:"services"`
}

type RuntimeHealthService struct {
	Key         string `json:"key"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
	CheckedAt   string `json:"checked_at"`
	LatencyMS   *int64 `json:"latency_ms,omitempty"`
	ReasonCode  string `json:"reason_code,omitempty"`
}

type PublicError struct {
	Status  int
	Code    string
	Message string
	Details any
}

func NewPublicError(status int, code, message string, details any) *PublicError {
	if details == nil {
		details = map[string]any{}
	}
	return &PublicError{Status: status, Code: code, Message: message, Details: details}
}

func (e *PublicError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

type Principal struct {
	Identity string
	Scopes   []string
}

func (p Principal) HasScope(scope string) bool {
	for _, candidate := range p.Scopes {
		if candidate == scope {
			return true
		}
	}
	return false
}

type principalContextKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	return principal, ok
}
