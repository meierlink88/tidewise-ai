package main

import (
	"testing"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/service"
)

func TestBuildAuthenticatorRequiresAllScopedServiceCredentials(t *testing.T) {
	cfg := conf.Config{Secrets: conf.SecretConfig{
		DataServiceAgentToken:   "agent-token",
		DataServiceMiniappToken: "miniapp-token",
	}}
	if _, err := buildAuthenticator(cfg); err == nil {
		t.Fatal("buildAuthenticator accepted a missing Admin credential")
	}

	cfg.Secrets.DataServiceAdminToken = "admin-token"
	if _, err := buildAuthenticator(cfg); err == nil {
		t.Fatal("buildAuthenticator accepted a missing Research publisher credential")
	}
	cfg.Secrets.DataServiceResearchPublisherToken = "research-publisher-token"
	authenticator, err := buildAuthenticator(cfg)
	if err != nil {
		t.Fatal(err)
	}
	assertPrincipal(t, authenticator, "agent-token", "agent-run", []string{
		service.ScopeReviewedEventImport,
	})
	assertPrincipal(t, authenticator, "miniapp-token", "miniapp-bff", []string{service.ScopeResearchRead})
	assertPrincipal(t, authenticator, "admin-token", "admin-portal-bff", []string{service.ScopeAdminRead})
	assertPrincipal(t, authenticator, "research-publisher-token", "research-theme-publisher", []string{service.ScopeResearchImport})
}

func assertPrincipal(t *testing.T, authenticator *service.Authenticator, token string, identity string, scopes []string) {
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
