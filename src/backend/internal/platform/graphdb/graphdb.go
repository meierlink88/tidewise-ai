package graphdb

import (
	"context"
	"fmt"
	"os"
	"time"

	appconfig "github.com/meierlink88/tidewise-ai/backend/internal/config"
	neo4j "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Driver interface {
	VerifyConnectivity(context.Context) error
	Close(context.Context) error
	ExecuteWrite(context.Context, string, string, map[string]any) error
}

type DriverConfig struct {
	URI                   string
	Database              string
	Username              string
	Password              string
	ConnectTimeout        time.Duration
	MaxConnectionPoolSize int
}

type CredentialLookup func(string) string

type DriverOpener func(DriverConfig) (Driver, error)

func Open(ctx context.Context, cfg appconfig.Neo4jConfig, lookup CredentialLookup, opener DriverOpener) (Driver, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if lookup == nil {
		lookup = os.Getenv
	}
	if opener == nil {
		opener = openDriver
	}

	username := lookup(cfg.UsernameEnv)
	password := lookup(cfg.PasswordEnv)
	if username == "" || password == "" {
		return nil, fmt.Errorf("resolve neo4j credentials from %s and %s", cfg.UsernameEnv, cfg.PasswordEnv)
	}

	driverCfg := DriverConfig{
		URI:                   cfg.URI,
		Database:              cfg.Database,
		Username:              username,
		Password:              password,
		ConnectTimeout:        time.Duration(cfg.ConnectTimeoutSeconds) * time.Second,
		MaxConnectionPoolSize: cfg.MaxConnectionPoolSize,
	}

	driver, err := opener(driverCfg)
	if err != nil {
		return nil, fmt.Errorf("open neo4j driver: %w", err)
	}
	if err := driver.VerifyConnectivity(ctx); err != nil {
		_ = driver.Close(ctx)
		return nil, fmt.Errorf("verify neo4j connectivity: %w", err)
	}

	return driver, nil
}

func openDriver(cfg DriverConfig) (Driver, error) {
	driver, err := neo4j.NewDriverWithContext(
		cfg.URI,
		neo4j.BasicAuth(cfg.Username, cfg.Password, ""),
		func(driverCfg *neo4j.Config) {
			driverCfg.SocketConnectTimeout = cfg.ConnectTimeout
			driverCfg.MaxConnectionPoolSize = cfg.MaxConnectionPoolSize
		},
	)
	if err != nil {
		return nil, err
	}
	return neo4jDriver{driver: driver}, nil
}

type neo4jDriver struct {
	driver neo4j.DriverWithContext
}

func (d neo4jDriver) VerifyConnectivity(ctx context.Context) error {
	return d.driver.VerifyConnectivity(ctx)
}

func (d neo4jDriver) Close(ctx context.Context) error {
	return d.driver.Close(ctx)
}

func (d neo4jDriver) ExecuteWrite(ctx context.Context, database string, query string, params map[string]any) error {
	_, err := neo4j.ExecuteQuery(
		ctx,
		d.driver,
		query,
		params,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(database),
		neo4j.ExecuteQueryWithWritersRouting(),
	)
	return err
}
