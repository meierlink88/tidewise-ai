package server

import (
	"net/http"
	"strings"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/service"
)

func authenticationFilter(authenticator *service.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			scope, protected := requiredScope(request.Method, request.URL.Path)
			if !protected {
				next.ServeHTTP(response, request)
				return
			}

			requestID := request.Header.Get("X-Request-ID")
			principal, ok := authenticator.Authenticate(request.Header.Get("Authorization"))
			if !ok {
				writeError(response, requestID, http.StatusUnauthorized, "UNAUTHENTICATED", "valid service identity is required")
				return
			}
			if !principal.HasScope(scope) {
				writeError(response, requestID, http.StatusForbidden, "FORBIDDEN", "service identity lacks the required scope")
				return
			}

			ctx := service.WithPrincipal(request.Context(), principal)
			next.ServeHTTP(response, request.WithContext(ctx))
		})
	}
}

func requiredScope(method, path string) (string, bool) {
	switch {
	case method == http.MethodPost && path == service.Namespace+"/reviewed-event-imports":
		return service.ScopeReviewedEventImport, true
	case method == http.MethodPost && (path == service.Namespace+"/research-theme-imports" ||
		path == service.Namespace+"/research-anchor-imports"):
		return service.ScopeResearchImport, true
	case method == http.MethodGet && (path == service.Namespace+"/research/themes" ||
		strings.HasPrefix(path, service.Namespace+"/research/themes/")):
		return service.ScopeResearchRead, true
	case method == http.MethodGet && (path == service.Namespace+"/raw-documents" ||
		path == service.Namespace+"/events"):
		return service.ScopeAdminRead, true
	default:
		return "", false
	}
}
