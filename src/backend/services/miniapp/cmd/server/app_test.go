package main

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/transport"

	"github.com/meierlink88/tidewise-ai/backend/internal/platform/runtimeconfig"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/conf"
)

func TestBuildAppComposesKratosApplicationWithoutStartingNetwork(t *testing.T) {
	app, err := buildApp(conf.RuntimeConfig{
		App: runtimeconfig.AppConfig{Name: conf.ServiceName, Env: runtimeconfig.EnvLocal},
		Server: runtimeconfig.ServerConfig{
			Host: "127.0.0.1", Port: 18082, ReadTimeoutSeconds: 5, WriteTimeoutSeconds: 10,
		},
		DataService: conf.DataServiceRuntimeConfig{
			BaseURL: "http://127.0.0.1:18081", IdentityToken: "test-token", Timeout: time.Second,
		},
	}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if app == nil {
		t.Fatal("buildApp() = nil")
	}
}

func TestApplicationStopUsesBoundedContextAndRunsCleanup(t *testing.T) {
	server := &lifecycleServer{
		started: make(chan struct{}),
		stopped: make(chan time.Duration, 1),
		done:    make(chan struct{}),
	}
	var cleaned atomic.Bool
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	app := newApp(server, logger, func(ctx context.Context) error {
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Error("cleanup context has no deadline")
		} else if remaining := time.Until(deadline); remaining <= 0 || remaining > resourceCleanupTimeout {
			t.Errorf("cleanup deadline remaining = %s, want within %s", remaining, resourceCleanupTimeout)
		}
		cleaned.Store(true)
		return nil
	})
	if app.Name() != conf.ServiceName || app.Version() != conf.ServiceVersion {
		t.Fatalf("app identity = %q/%q", app.Name(), app.Version())
	}
	result := make(chan error, 1)
	go func() {
		result <- app.Run()
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
		t.Fatal("after-stop client cleanup was not called")
	}
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
