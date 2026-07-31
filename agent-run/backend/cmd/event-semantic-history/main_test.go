package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
)

type historicalDispositionStoreStub struct {
	lockHeld       bool
	planUnderLock  bool
	applyUnderLock bool
	applyCalled    bool
	plan           eventsemantic.HistoricalDispositionReport
}

func (s *historicalDispositionStoreStub) WithHistoricalEventSemanticMaintenance(
	_ context.Context,
	operation func() error,
) error {
	s.lockHeld = true
	defer func() { s.lockHeld = false }()
	return operation()
}

func (s *historicalDispositionStoreStub) PlanHistoricalEventDisposition(
	context.Context,
	eventsemantic.HistoricalManifest,
) (eventsemantic.HistoricalDispositionReport, error) {
	s.planUnderLock = s.lockHeld
	return s.plan, nil
}

func (s *historicalDispositionStoreStub) ApplyHistoricalEventDisposition(
	context.Context,
	eventsemantic.HistoricalManifest,
	time.Time,
) (eventsemantic.HistoricalDispositionReport, error) {
	s.applyCalled = true
	s.applyUnderLock = s.lockHeld
	return eventsemantic.HistoricalDispositionReport{Mode: "apply"}, nil
}

func TestHistoricalApplyRequiresExplicitEnvironmentAndPreChangeExport(t *testing.T) {
	for _, arguments := range [][]string{
		{"-apply", "-manifest", "audit.json"},
		{"-apply", "-manifest", "audit.json", "-allow-env", "uat"},
		{"-dry-run", "-manifest", "audit.json", "-export", "before.json"},
	} {
		if _, err := parseOptions(arguments); err == nil {
			t.Fatalf("arguments %#v were accepted", arguments)
		}
	}
	options, err := parseOptions([]string{
		"-apply", "-manifest", "audit.json",
		"-allow-env", "uat", "-export", "before.json",
	})
	if err != nil || !options.Apply {
		t.Fatalf("options = %#v err = %v", options, err)
	}
}

func TestPreChangeExportNeverOverwritesReviewedEvidence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "before.json")
	if err := os.WriteFile(path, []byte("reviewed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeExclusiveJSON(path, map[string]int{"changed": 1}); err == nil {
		t.Fatal("existing pre-change export was overwritten")
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != "reviewed" {
		t.Fatalf("payload = %q", payload)
	}
}

func TestHistoricalApplyLocksPlanExportAndApplyAsOneOperation(t *testing.T) {
	store := &historicalDispositionStoreStub{
		plan: eventsemantic.HistoricalDispositionReport{Mode: "dry_run"},
	}
	path := filepath.Join(t.TempDir(), "before.json")
	var output bytes.Buffer
	err := runHistoricalDisposition(
		context.Background(),
		store,
		eventsemantic.HistoricalManifest{},
		options{Apply: true, ExportPath: path},
		&output,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !store.planUnderLock || !store.applyUnderLock || !store.applyCalled {
		t.Fatalf("maintenance lock coverage = %#v", store)
	}
	var exported eventsemantic.HistoricalDispositionReport
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, &exported); err != nil {
		t.Fatal(err)
	}
	if exported.Mode != "dry_run" {
		t.Fatalf("exported mode = %q", exported.Mode)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("export mode = %o", info.Mode().Perm())
	}
}
