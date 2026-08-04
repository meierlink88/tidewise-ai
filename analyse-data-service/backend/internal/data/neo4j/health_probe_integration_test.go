package neo4j

import (
	"context"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/runtimehealth"
)

const neo4jHealthIntegrationOptIn = "TIDEWISE_NEO4J_HEALTH_INTEGRATION_TEST"

func TestHealthProbeAgainstRealNeo4j(t *testing.T) {
	config := neo4jHealthIntegrationConfig(t)
	probe, err := NewHealthProbe(config)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := probe.Close(context.Background()); err != nil {
			t.Errorf("close ready probe: %v", err)
		}
	}()

	t.Run("ready", func(t *testing.T) {
		result := probe.Check(context.Background())
		if result.Status != runtimehealth.StatusReady || result.ReasonCode != runtimehealth.ReasonNone {
			t.Fatalf("result = %#v", result)
		}
	})

	t.Run("authentication failure", func(t *testing.T) {
		invalidConfig := config
		invalidConfig.Password += "-invalid"
		invalidProbe, err := NewHealthProbe(invalidConfig)
		if err != nil {
			t.Fatal(err)
		}
		defer invalidProbe.Close(context.Background())
		result := invalidProbe.Check(context.Background())
		if result.Status != runtimehealth.StatusDown || result.ReasonCode != runtimehealth.ReasonAuthenticationFailed {
			t.Fatalf("result = %#v", result)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		timeoutConfig := config
		timeoutConfig.Timeout = time.Nanosecond
		timeoutProbe, err := NewHealthProbe(timeoutConfig)
		if err != nil {
			t.Fatal(err)
		}
		defer timeoutProbe.Close(context.Background())
		result := timeoutProbe.Check(context.Background())
		if result.Status != runtimehealth.StatusDown || result.ReasonCode != runtimehealth.ReasonTimeout {
			t.Fatalf("result = %#v", result)
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		result := probe.Check(ctx)
		if result.Status != runtimehealth.StatusDown || result.ReasonCode != runtimehealth.ReasonTimeout {
			t.Fatalf("result = %#v", result)
		}
	})
}

func neo4jHealthIntegrationConfig(t *testing.T) HealthConfig {
	t.Helper()
	if os.Getenv(neo4jHealthIntegrationOptIn) != "1" {
		t.Skip("set TIDEWISE_NEO4J_HEALTH_INTEGRATION_TEST=1 to run against the disposable local Neo4j container")
	}
	config := HealthConfig{
		URI: os.Getenv("TIDEWISE_TEST_NEO4J_URI"), Database: os.Getenv("TIDEWISE_TEST_NEO4J_DATABASE"),
		Username: os.Getenv("TIDEWISE_TEST_NEO4J_USERNAME"), Password: os.Getenv("TIDEWISE_TEST_NEO4J_PASSWORD"),
		Timeout: 5 * time.Second,
	}
	if config.URI == "" || config.Database == "" || config.Username == "" || config.Password == "" {
		t.Skip("set all TIDEWISE_TEST_NEO4J_URI/DATABASE/USERNAME/PASSWORD values")
	}
	parsed, err := url.Parse(config.URI)
	if err != nil || !isLoopbackHost(parsed.Hostname()) {
		t.Fatalf("health integration test requires a loopback Neo4j container URI, got %q", config.URI)
	}
	return config
}

func isLoopbackHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
