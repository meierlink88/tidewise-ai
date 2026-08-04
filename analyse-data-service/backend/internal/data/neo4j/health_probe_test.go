package neo4j

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/runtimehealth"
	driver "github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

type connectivityStub struct {
	connectivityErr error
	databaseErr     error
	database        *string
}

type blockingConnectivityStub struct{}

func (blockingConnectivityStub) VerifyConnectivity(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func (blockingConnectivityStub) VerifyDatabase(context.Context, string) error { return nil }

func (c connectivityStub) VerifyConnectivity(context.Context) error { return c.connectivityErr }
func (c connectivityStub) VerifyDatabase(_ context.Context, database string) error {
	if c.database != nil {
		*c.database = database
	}
	return c.databaseErr
}

func TestHealthProbeMapsDependencyErrorsToSafeReasonCodes(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status runtimehealth.Status
		reason runtimehealth.ReasonCode
	}{
		{name: "ready", status: runtimehealth.StatusReady},
		{name: "authentication", err: &driver.Neo4jError{Code: "Neo.ClientError.Security.Unauthorized", Msg: "secret detail"}, status: runtimehealth.StatusDown, reason: runtimehealth.ReasonAuthenticationFailed},
		{name: "unreachable", err: errors.New("dial tcp 10.0.0.1:7687: connection refused"), status: runtimehealth.StatusDown, reason: runtimehealth.ReasonUnreachable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			probe := &HealthProbe{driver: connectivityStub{connectivityErr: test.err}, database: "neo4j", timeout: time.Second, now: time.Now}
			result := probe.Check(context.Background())
			if result.Status != test.status || result.ReasonCode != test.reason {
				t.Fatalf("result = %#v", result)
			}
		})
	}
}

func TestHealthProbeHonorsCallerDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	probe := &HealthProbe{driver: connectivityStub{connectivityErr: context.Canceled}, database: "neo4j", timeout: time.Second, now: time.Now}

	result := probe.Check(ctx)

	if result.Status != runtimehealth.StatusDown || result.ReasonCode != runtimehealth.ReasonTimeout {
		t.Fatalf("result = %#v", result)
	}
}

func TestHealthProbeBoundsConnectivityCheckWithConfiguredTimeout(t *testing.T) {
	probe := &HealthProbe{
		driver: blockingConnectivityStub{}, database: "neo4j", timeout: time.Millisecond, now: time.Now,
	}

	result := probe.Check(context.Background())

	if result.Status != runtimehealth.StatusDown || result.ReasonCode != runtimehealth.ReasonTimeout {
		t.Fatalf("result = %#v", result)
	}
}

func TestHealthProbeVerifiesConfiguredDatabaseWithoutReadingGraphData(t *testing.T) {
	var database string
	probe := &HealthProbe{
		driver: connectivityStub{database: &database}, database: "industry-projection",
		timeout: time.Second, now: time.Now,
	}

	result := probe.Check(context.Background())

	if result.Status != runtimehealth.StatusReady || database != "industry-projection" {
		t.Fatalf("database=%q result=%#v", database, result)
	}
}
