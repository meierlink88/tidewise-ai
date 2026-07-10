package graphdb

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/config"
)

func TestOpenReturnsNilWhenDisabled(t *testing.T) {
	driver, err := Open(context.Background(), config.Neo4jConfig{Enabled: false}, nil, nil)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if driver != nil {
		t.Fatal("Open() driver != nil, want nil for disabled neo4j")
	}
}

func TestOpenRejectsMissingCredentials(t *testing.T) {
	cfg := enabledConfig()

	_, err := Open(context.Background(), cfg, func(string) string { return "" }, nil)
	if err == nil {
		t.Fatal("Open() error = nil, want missing credential error")
	}
	if !strings.Contains(err.Error(), "resolve neo4j credentials") {
		t.Fatalf("Open() error = %q, want credential context", err.Error())
	}
}

func TestOpenVerifiesConnectivity(t *testing.T) {
	cfg := enabledConfig()
	fake := &fakeDriver{}

	driver, err := Open(context.Background(), cfg, testCredentialLookup, func(DriverConfig) (Driver, error) {
		return fake, nil
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if driver != fake {
		t.Fatal("Open() returned unexpected driver")
	}
	if !fake.verified {
		t.Fatal("expected connectivity verification")
	}
}

func TestOpenClosesDriverWhenConnectivityFails(t *testing.T) {
	cfg := enabledConfig()
	fake := &fakeDriver{verifyErr: errors.New("neo4j offline")}

	_, err := Open(context.Background(), cfg, testCredentialLookup, func(DriverConfig) (Driver, error) {
		return fake, nil
	})
	if err == nil {
		t.Fatal("Open() error = nil, want connectivity error")
	}
	if !fake.closed {
		t.Fatal("expected driver to be closed after connectivity failure")
	}
}

func TestOpenPassesResolvedDriverConfig(t *testing.T) {
	cfg := enabledConfig()
	var got DriverConfig

	_, err := Open(context.Background(), cfg, testCredentialLookup, func(driverCfg DriverConfig) (Driver, error) {
		got = driverCfg
		return &fakeDriver{}, nil
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	if got.URI != cfg.URI {
		t.Fatalf("DriverConfig.URI = %q, want %q", got.URI, cfg.URI)
	}
	if got.Database != cfg.Database {
		t.Fatalf("DriverConfig.Database = %q, want %q", got.Database, cfg.Database)
	}
	if got.Username != "local-neo4j" || got.Password != "local-password" {
		t.Fatalf("DriverConfig credentials not resolved correctly")
	}
	if got.ConnectTimeout != 5*time.Second {
		t.Fatalf("DriverConfig.ConnectTimeout = %v", got.ConnectTimeout)
	}
	if got.MaxConnectionPoolSize != 10 {
		t.Fatalf("DriverConfig.MaxConnectionPoolSize = %d", got.MaxConnectionPoolSize)
	}
}

func TestDriverCanExecuteWriteQueries(t *testing.T) {
	fake := &fakeDriver{}
	params := map[string]any{"id": "entity-1"}

	if err := fake.ExecuteWrite(context.Background(), "neo4j", "MERGE (n)", params); err != nil {
		t.Fatalf("ExecuteWrite() error = %v", err)
	}
	if fake.database != "neo4j" || fake.query != "MERGE (n)" {
		t.Fatalf("ExecuteWrite() database/query = %q/%q", fake.database, fake.query)
	}
	if fake.params["id"] != "entity-1" {
		t.Fatalf("ExecuteWrite() params = %+v", fake.params)
	}
}

func enabledConfig() config.Neo4jConfig {
	return config.Neo4jConfig{
		Enabled:               true,
		URI:                   "bolt://localhost:7687",
		Database:              "neo4j",
		UsernameEnv:           "NEO4J_USERNAME",
		PasswordEnv:           "NEO4J_PASSWORD",
		ConnectTimeoutSeconds: 5,
		MaxConnectionPoolSize: 10,
	}
}

func testCredentialLookup(name string) string {
	switch name {
	case "NEO4J_USERNAME":
		return "local-neo4j"
	case "NEO4J_PASSWORD":
		return "local-password"
	default:
		return ""
	}
}

type fakeDriver struct {
	verified  bool
	closed    bool
	verifyErr error
	database  string
	query     string
	params    map[string]any
}

func (f *fakeDriver) VerifyConnectivity(context.Context) error {
	f.verified = true
	return f.verifyErr
}

func (f *fakeDriver) Close(context.Context) error {
	f.closed = true
	return nil
}

func (f *fakeDriver) ExecuteWrite(_ context.Context, database string, query string, params map[string]any) error {
	f.database = database
	f.query = query
	f.params = params
	return nil
}
