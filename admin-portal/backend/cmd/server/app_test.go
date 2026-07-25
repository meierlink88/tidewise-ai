package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/transport"

	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/conf"
)

func TestBuildAdminAppComposesKratosApplicationWithoutStartingNetwork(t *testing.T) {
	app, cleanup, err := buildApp(conf.RuntimeConfig{
		App: conf.AppConfig{Name: conf.ServiceName, Env: conf.EnvLocal},
		Server: conf.ServerConfig{
			Host: "127.0.0.1", Port: 18083, ReadTimeoutSeconds: 5, WriteTimeoutSeconds: 10,
		},
		AdminToken:    "browser-admin-token",
		AllowedOrigin: "http://127.0.0.1:5174",
		DataService: conf.DataServiceRuntimeConfig{
			BaseURL: "http://127.0.0.1:18081", IdentityToken: "test-token", Timeout: time.Second,
		},
		AgentRun: conf.AgentRunRuntimeConfig{
			BaseURL: "http://127.0.0.1:18080", ServiceToken: "agent-token", Timeout: time.Second,
		},
	}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if app == nil {
		t.Fatal("buildApp() = nil")
	}
	if cleanup == nil {
		t.Fatal("buildApp() cleanup = nil")
	}
	if err := runCleanup(cleanup, time.Second); err != nil {
		t.Fatalf("cleanup built app: %v", err)
	}
}

func TestApplicationRunUsesBoundedStopContextAndAlwaysRunsCleanup(t *testing.T) {
	server := &lifecycleServer{
		started: make(chan struct{}),
		stopped: make(chan time.Duration, 1),
		done:    make(chan struct{}),
	}
	var cleaned atomic.Bool
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	app := newApp(server, logger)
	cleanup := func(ctx context.Context) error {
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Error("cleanup context has no deadline")
		} else if remaining := time.Until(deadline); remaining <= 0 || remaining > resourceCleanupTimeout {
			t.Errorf("cleanup deadline remaining = %s, want within %s", remaining, resourceCleanupTimeout)
		}
		cleaned.Store(true)
		return nil
	}
	if app.Name() != conf.ServiceName || app.Version() != conf.ServiceVersion {
		t.Fatalf("app identity = %q/%q", app.Name(), app.Version())
	}
	result := make(chan error, 1)
	go func() {
		result <- runApplication(app, cleanup)
	}()

	select {
	case <-server.started:
	case <-time.After(time.Second):
		t.Fatal("server did not start")
	}
	if err := app.Stop(); err != nil {
		t.Fatalf("stop app: %v", err)
	}
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("run app: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("app did not stop")
	}
	select {
	case remaining := <-server.stopped:
		if remaining < 9*time.Second || remaining > applicationStopTimeout {
			t.Fatalf("server stop deadline remaining = %s, want approximately %s", remaining, applicationStopTimeout)
		}
	default:
		t.Fatal("server Stop was not called")
	}
	if !cleaned.Load() {
		t.Fatal("client cleanup was not called")
	}
}

func TestApplicationRunsCleanupWhenServerStartFails(t *testing.T) {
	startErr := errors.New("start failed")
	var cleaned atomic.Bool
	app := newApp(failingLifecycleServer{startErr: startErr}, testCommandLogger())

	err := runApplication(app, func(context.Context) error {
		cleaned.Store(true)
		return nil
	})

	if !errors.Is(err, startErr) {
		t.Fatalf("run error = %v, want %v", err, startErr)
	}
	if !cleaned.Load() {
		t.Fatal("cleanup did not run after server start failure")
	}
}

func TestRunCleanupReturnsAtDeadlineWhenCloserBlocks(t *testing.T) {
	timeout := 20 * time.Millisecond
	release := make(chan struct{})
	startedAt := time.Now()
	err := runCleanup(func(context.Context) error {
		<-release
		return nil
	}, timeout)
	close(release)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cleanup error = %v, want deadline exceeded", err)
	}
	if elapsed := time.Since(startedAt); elapsed < timeout || elapsed > 10*timeout {
		t.Fatalf("bounded cleanup elapsed = %s, want approximately %s", elapsed, timeout)
	}
}

func testCommandLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

type lifecycleServer struct {
	started chan struct{}
	stopped chan time.Duration
	done    chan struct{}
}

var _ transport.Server = (*lifecycleServer)(nil)

func (s *lifecycleServer) Start(ctx context.Context) error {
	close(s.started)
	<-s.done
	return nil
}

func (s *lifecycleServer) Stop(ctx context.Context) error {
	close(s.done)
	deadline, ok := ctx.Deadline()
	if !ok {
		s.stopped <- 0
		return nil
	}
	s.stopped <- time.Until(deadline)
	return nil
}

type failingLifecycleServer struct {
	startErr error
}

var _ transport.Server = failingLifecycleServer{}

func (s failingLifecycleServer) Start(context.Context) error { return s.startErr }
func (failingLifecycleServer) Stop(context.Context) error    { return nil }
