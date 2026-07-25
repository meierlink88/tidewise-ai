package service

import (
	"context"
	"crypto/subtle"
	"fmt"
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

type principalContextKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	return principal, ok
}
