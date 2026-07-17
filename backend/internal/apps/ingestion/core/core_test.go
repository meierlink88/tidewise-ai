package core

import (
	"context"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestRegistryReturnsRegisteredConnectorAndParser(t *testing.T) {
	registry := NewRegistry()
	connector := fakeConnector{}
	parser := fakeParser{}

	registry.RegisterConnector("rss_feed", connector)
	registry.RegisterParser("rss_item", parser)

	if _, err := registry.Connector("rss_feed"); err != nil {
		t.Fatalf("Connector() error = %v", err)
	}
	if _, err := registry.Parser("rss_item"); err != nil {
		t.Fatalf("Parser() error = %v", err)
	}
	if _, err := registry.Connector("missing"); err == nil {
		t.Fatal("Connector() error = nil, want missing connector error")
	}
	if _, err := registry.Parser("missing"); err == nil {
		t.Fatal("Parser() error = nil, want missing parser error")
	}
}

func TestEnvCredentialResolverReadsEnvironmentReferences(t *testing.T) {
	t.Setenv("TIDEWISE_TEST_TOKEN", "secret-value")

	resolver := EnvCredentialResolver{}
	value, err := resolver.Resolve("env:TIDEWISE_TEST_TOKEN")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if value != "secret-value" {
		t.Fatalf("resolved value = %q", value)
	}

	if _, err := resolver.Resolve("env:MISSING_TIDEWISE_TEST_TOKEN"); err == nil {
		t.Fatal("Resolve() error = nil, want missing env error")
	}
}

type fakeConnector struct{}

func (fakeConnector) Fetch(context.Context, domain.SourceCatalog, Credential) (RawResponse, error) {
	return RawResponse{}, nil
}

type fakeParser struct{}

func (fakeParser) Parse(context.Context, domain.SourceCatalog, RawResponse) ([]RawDocumentCandidate, error) {
	return nil, nil
}
