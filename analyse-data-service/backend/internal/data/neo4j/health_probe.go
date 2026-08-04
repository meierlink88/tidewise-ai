package neo4j

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/runtimehealth"
	driver "github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

type HealthConfig struct {
	URI      string
	Database string
	Username string
	Password string
	Timeout  time.Duration
}

type connectivityDriver interface {
	VerifyConnectivity(context.Context) error
	VerifyDatabase(context.Context, string) error
}

type liveConnectivityDriver struct{ driver.Driver }

func (d liveConnectivityDriver) VerifyDatabase(ctx context.Context, database string) error {
	_, err := driver.ExecuteQuery(
		ctx,
		d.Driver,
		"RETURN 1 AS ok",
		nil,
		driver.EagerResultTransformer,
		driver.ExecuteQueryWithDatabase(database),
		driver.ExecuteQueryWithReadersRouting(),
	)
	return err
}

type HealthProbe struct {
	driver   connectivityDriver
	database string
	timeout  time.Duration
	now      func() time.Time
}

func NewHealthProbe(config HealthConfig) (*HealthProbe, error) {
	if strings.TrimSpace(config.URI) == "" || strings.TrimSpace(config.Database) == "" ||
		strings.TrimSpace(config.Username) == "" || config.Password == "" || config.Timeout <= 0 {
		return nil, errors.New("Neo4j health URI, database, username, password and timeout are required")
	}
	value, err := driver.NewDriver(config.URI, driver.BasicAuth(config.Username, config.Password, ""))
	if err != nil {
		return nil, errors.New("configure Neo4j health driver failed")
	}
	return &HealthProbe{
		driver: liveConnectivityDriver{Driver: value}, database: config.Database,
		timeout: config.Timeout, now: time.Now,
	}, nil
}

func (p *HealthProbe) Check(ctx context.Context) runtimehealth.Check {
	startedAt := p.now()
	if ctx.Err() != nil {
		return runtimehealth.Check{Status: runtimehealth.StatusDown, ReasonCode: runtimehealth.ReasonTimeout}
	}
	checkContext, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	err := p.driver.VerifyConnectivity(checkContext)
	if err == nil {
		err = p.driver.VerifyDatabase(checkContext, p.database)
	}
	latency := p.now().Sub(startedAt)
	if err == nil {
		return runtimehealth.Check{Status: runtimehealth.StatusReady, Latency: latency}
	}
	if checkContext.Err() != nil || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return runtimehealth.Check{Status: runtimehealth.StatusDown, ReasonCode: runtimehealth.ReasonTimeout, Latency: latency}
	}
	var neo4jError *driver.Neo4jError
	var authenticationError *driver.InvalidAuthenticationError
	if errors.As(err, &authenticationError) || errors.As(err, &neo4jError) && neo4jError.Code == "Neo.ClientError.Security.Unauthorized" {
		return runtimehealth.Check{Status: runtimehealth.StatusDown, ReasonCode: runtimehealth.ReasonAuthenticationFailed, Latency: latency}
	}
	return runtimehealth.Check{Status: runtimehealth.StatusDown, ReasonCode: runtimehealth.ReasonUnreachable, Latency: latency}
}

func (p *HealthProbe) Close(ctx context.Context) error {
	if p == nil || p.driver == nil {
		return nil
	}
	closer, ok := p.driver.(interface{ Close(context.Context) error })
	if !ok {
		return nil
	}
	if err := closer.Close(ctx); err != nil {
		return errors.New("close Neo4j health driver failed")
	}
	return nil
}

var _ runtimehealth.Probe = (*HealthProbe)(nil)
