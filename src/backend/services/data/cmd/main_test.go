package main

import (
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/services/data/config"
	"github.com/meierlink88/tidewise-ai/backend/services/data/transport/internalapi"
)

func TestBuildAuthenticatorRequiresAllScopedServiceCredentials(t *testing.T) {
	cfg := config.Config{Secrets: config.SecretConfig{
		DataServiceAgentToken:   "agent-token",
		DataServiceMiniappToken: "miniapp-token",
	}}
	if _, err := buildAuthenticator(cfg); err == nil {
		t.Fatal("buildAuthenticator accepted a missing Admin credential")
	}

	cfg.Secrets.DataServiceAdminToken = "admin-token"
	authenticator, err := buildAuthenticator(cfg)
	if err != nil {
		t.Fatal(err)
	}
	assertPrincipal(t, authenticator, "agent-token", "agent-run", []string{
		internalapi.ScopeRawImport,
		internalapi.ScopeReviewedEventImport,
		internalapi.ScopeSourceMetadataRead,
	})
	assertPrincipal(t, authenticator, "miniapp-token", "miniapp-bff", []string{internalapi.ScopeResearchRead})
	assertPrincipal(t, authenticator, "admin-token", "admin-portal-bff", []string{internalapi.ScopeAdminRead})
}

func assertPrincipal(t *testing.T, authenticator *internalapi.Authenticator, token string, identity string, scopes []string) {
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
