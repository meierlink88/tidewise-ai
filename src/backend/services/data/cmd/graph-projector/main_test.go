package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/services/data/repositories"
)

func TestRunCheckCommand(t *testing.T) {
	executor := &recordingExecutor{}
	var out bytes.Buffer

	code := run(context.Background(), []string{"check"}, &out, executor)

	if code != 0 {
		t.Fatalf("run() code = %d, want 0", code)
	}
	if !executor.checked {
		t.Fatal("expected check to run")
	}
	if !strings.Contains(out.String(), "neo4j connectivity ok") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestRunProjectEntitiesCommand(t *testing.T) {
	executor := &recordingExecutor{}
	var out bytes.Buffer

	code := run(context.Background(), []string{"project-entities"}, &out, executor)

	if code != 0 {
		t.Fatalf("run() code = %d, want 0", code)
	}
	if executor.mode != repositories.GraphProjectionModeProjectEntities {
		t.Fatalf("project mode = %q, want project_entities", executor.mode)
	}
	if !strings.Contains(out.String(), "projected=3") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestRunRebuildEntitiesCommand(t *testing.T) {
	executor := &recordingExecutor{}
	var out bytes.Buffer

	code := run(context.Background(), []string{"rebuild-entities"}, &out, executor)

	if code != 0 {
		t.Fatalf("run() code = %d, want 0", code)
	}
	if executor.mode != repositories.GraphProjectionModeRebuildEntities {
		t.Fatalf("project mode = %q, want rebuild_entities", executor.mode)
	}
}

func TestRunRejectsInvalidCommand(t *testing.T) {
	var out bytes.Buffer
	code := run(context.Background(), []string{"bad-command"}, &out, &recordingExecutor{})

	if code == 0 {
		t.Fatal("run() code = 0, want failure")
	}
	if !strings.Contains(out.String(), "usage") {
		t.Fatalf("output = %q, want usage", out.String())
	}
}

func TestRunReturnsFailureWhenExecutorFails(t *testing.T) {
	var out bytes.Buffer
	code := run(context.Background(), []string{"check"}, &out, &recordingExecutor{err: errors.New("load config failed")})

	if code == 0 {
		t.Fatal("run() code = 0, want failure")
	}
	if !strings.Contains(out.String(), "load config failed") {
		t.Fatalf("output = %q", out.String())
	}
}

type recordingExecutor struct {
	checked bool
	mode    repositories.GraphProjectionMode
	err     error
}

func (e *recordingExecutor) Check(context.Context) error {
	e.checked = true
	return e.err
}

func (e *recordingExecutor) Project(ctx context.Context, mode repositories.GraphProjectionMode) (repositories.GraphProjectionRun, error) {
	e.mode = mode
	if e.err != nil {
		return repositories.GraphProjectionRun{}, e.err
	}
	return repositories.GraphProjectionRun{
		ID:             "run-1",
		ProjectionType: repositories.GraphProjectionTypeEntityGraph,
		Mode:           mode,
		Status:         repositories.GraphProjectionRunStatusSucceeded,
		SourceRowCount: 3,
		ProjectedCount: 3,
	}, nil
}
