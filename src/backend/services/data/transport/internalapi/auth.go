package internalapi

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"
)

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

type Credential struct {
	Secret    string
	Principal Principal
}

type Authenticator struct {
	credentials []Credential
}

func NewAuthenticator(credentials []Credential) (*Authenticator, error) {
	result := &Authenticator{credentials: make([]Credential, 0, len(credentials))}
	seenSecret := map[string]struct{}{}
	for _, credential := range credentials {
		credential.Secret = strings.TrimSpace(credential.Secret)
		credential.Principal.Identity = strings.TrimSpace(credential.Principal.Identity)
		if credential.Secret == "" || credential.Principal.Identity == "" || len(credential.Principal.Scopes) == 0 {
			return nil, fmt.Errorf("service credential, identity and scopes are required")
		}
		if utf8.RuneCountInString(credential.Principal.Identity) > 200 {
			return nil, fmt.Errorf("service identity must contain at most 200 characters")
		}
		if _, duplicate := seenSecret[credential.Secret]; duplicate {
			return nil, fmt.Errorf("service credentials must be unique")
		}
		seenSecret[credential.Secret] = struct{}{}
		result.credentials = append(result.credentials, credential)
	}
	return result, nil
}

func (a *Authenticator) Authenticate(header string) (Principal, bool) {
	const prefix = "Bearer "
	if a == nil || !strings.HasPrefix(header, prefix) {
		return Principal{}, false
	}
	presented := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	for _, credential := range a.credentials {
		if len(presented) == len(credential.Secret) && subtle.ConstantTimeCompare([]byte(presented), []byte(credential.Secret)) == 1 {
			return credential.Principal, true
		}
	}
	return Principal{}, false
}

func (d Dependencies) authorize(scope string, next operation) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestID := strings.TrimSpace(request.Header.Get("X-Request-ID"))
		if requestID == "" || len(requestID) > 128 {
			requestID = d.NewRequestID()
		}
		response.Header().Set("X-Request-ID", requestID)
		principal, ok := d.Authenticator.Authenticate(request.Header.Get("Authorization"))
		if !ok {
			writeError(response, requestID, http.StatusUnauthorized, "UNAUTHENTICATED", "valid service identity is required")
			return
		}
		if !principal.HasScope(scope) {
			writeError(response, requestID, http.StatusForbidden, "FORBIDDEN", "service identity lacks the required scope")
			return
		}
		defer func() {
			if recover() != nil {
				writeError(response, requestID, http.StatusInternalServerError, "INTERNAL_ERROR", "internal data service error")
			}
		}()
		next(response, request, principal, requestID)
	})
}
