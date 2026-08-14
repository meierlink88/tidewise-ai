package main

import (
	"testing"

	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/server"
)

func TestBuildAuthenticatorUsesOneDataServiceTokenForAllBusinessScopes(t *testing.T) {
	cfg := conf.Config{}
	if _, err := buildAuthenticator(cfg); err == nil {
		t.Fatal("buildAuthenticator accepted a missing Data Service token")
	}

	cfg.Secrets.ServiceToken = "data-service-token"
	authenticator, err := buildAuthenticator(cfg)
	if err != nil {
		t.Fatal(err)
	}
	assertPrincipal(t, authenticator, "data-service-token", "tidewise-internal-service", []string{
		server.ScopeReviewedEventImport,
		server.ScopeEventTagRead,
		server.ScopeResearchImport,
		server.ScopeResearchRead,
		server.ScopeAdminRead,
	})
}

func assertPrincipal(t *testing.T, authenticator *server.Authenticator, token string, identity string, scopes []string) {
	t.Helper()
	principal, ok := authenticator.Authenticate("Bearer " + token)
	if !ok || principal.Identity != identity {
		t.Fatalf("principal for %q = %#v, authenticated=%v", identity, principal, ok)
	}
	for _, scope := range scopes {
		if !principal.HasScope(scope) {
			t.Fatalf("principal %q lacks scope %q", identity, scope)
		}
	}
}
