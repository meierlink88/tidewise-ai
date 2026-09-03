package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type recordingSystemd struct {
	root             string
	followFilesystem bool
	units            map[string]unitState
	mutations        []string
}

func (systemd *recordingSystemd) inspect(_ context.Context, name string) (unitState, error) {
	state, exists := systemd.units[name]
	if !exists {
		return unitState{loadState: loadStateNotFound, activeState: activeStateInactive}, nil
	}
	if systemd.followFilesystem {
		if _, err := os.Lstat(hostPath(systemd.root, state.fragmentPath)); errors.Is(err, os.ErrNotExist) {
			return unitState{loadState: loadStateNotFound, activeState: activeStateInactive}, nil
		} else if err != nil {
			return unitState{}, err
		}
	}
	return state, nil
}

func (systemd *recordingSystemd) disableAndStop(_ context.Context, name string) error {
	systemd.mutations = append(systemd.mutations, "disable:"+name)
	state := systemd.units[name]
	state.activeState = activeStateInactive
	systemd.units[name] = state
	return nil
}

func (systemd *recordingSystemd) reload(_ context.Context) error {
	systemd.mutations = append(systemd.mutations, "reload")
	return nil
}

func TestPreflightRejectsUnexpectedSystemdFragmentWithoutMutation(t *testing.T) {
	root := retirementFixtureRoot(t)
	systemd := &recordingSystemd{root: root, units: map[string]unitState{
		reasonRunnerUnit: {loadState: loadStateLoaded, activeState: activeStateInactive, fragmentPath: "/usr/lib/systemd/system/unexpected.service"},
	}}

	err := execute(context.Background(), root, actionPreflight, systemd, &bytes.Buffer{})
	if !errors.Is(err, errUnexpectedSystemdFragment) {
		t.Fatalf("preflight error = %v, want unexpected fragment", err)
	}
	if len(systemd.mutations) != 0 {
		t.Fatalf("preflight mutated systemd: %v", systemd.mutations)
	}
	for _, target := range retiredHostPaths {
		if _, statErr := os.Stat(hostPath(root, target)); statErr != nil {
			t.Fatalf("preflight changed %s: %v", target, statErr)
		}
	}
}

func TestApplyStopsExactUnitsAndDeletesOnlyApprovedPaths(t *testing.T) {
	root := retirementFixtureRoot(t)
	protected := filepath.Join(root, "opt", "tidewise", "uat", "state", "current.sha")
	writeRetirementFixture(t, protected, "retained\n")
	systemd := &recordingSystemd{root: root, followFilesystem: true, units: map[string]unitState{
		reasonRunnerUnit: {loadState: loadStateLoaded, activeState: activeStateActive, fragmentPath: reasonRunnerUnitPath},
		neo4jUnit:        {loadState: loadStateLoaded, activeState: activeStateFailed, fragmentPath: neo4jUnitPath},
	}}
	var output bytes.Buffer

	if err := execute(context.Background(), root, actionApply, systemd, &output); err != nil {
		t.Fatalf("apply retirement: %v\n%s", err, output.String())
	}
	for _, target := range retiredHostPaths {
		if _, err := os.Lstat(hostPath(root, target)); !os.IsNotExist(err) {
			t.Fatalf("retired path %s remains: %v", target, err)
		}
	}
	if content, err := os.ReadFile(protected); err != nil || string(content) != "retained\n" {
		t.Fatalf("retained state changed: content=%q err=%v", content, err)
	}
	wantMutations := []string{"disable:" + reasonRunnerUnit, "disable:" + neo4jUnit, "reload"}
	if strings.Join(systemd.mutations, "|") != strings.Join(wantMutations, "|") {
		t.Fatalf("systemd mutations = %v, want %v", systemd.mutations, wantMutations)
	}
	if !strings.Contains(output.String(), "PASS root-retirement-apply") {
		t.Fatalf("missing apply receipt: %s", output.String())
	}
}

func TestPreflightRejectsSymlinkedApprovedPath(t *testing.T) {
	root := retirementFixtureRoot(t)
	outside := t.TempDir()
	target := hostPath(root, "/opt/tidewise/reason-uat")
	if err := os.RemoveAll(target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, target); err != nil {
		t.Fatal(err)
	}
	systemd := &recordingSystemd{root: root, units: map[string]unitState{}}

	err := execute(context.Background(), root, actionPreflight, systemd, &bytes.Buffer{})
	if !errors.Is(err, errSymbolicLinkTarget) {
		t.Fatalf("preflight error = %v, want symbolic-link rejection", err)
	}
	if len(systemd.mutations) != 0 {
		t.Fatalf("preflight mutated systemd: %v", systemd.mutations)
	}
}

func TestParseActionRejectsUnknownValue(t *testing.T) {
	_, err := parseAction("destroy")
	if !errors.Is(err, errInvalidAction) {
		t.Fatalf("parseAction error = %v, want invalid action", err)
	}
}

func TestPreflightRejectsUnknownSystemdStateWithoutMutation(t *testing.T) {
	root := retirementFixtureRoot(t)
	systemd := &recordingSystemd{root: root, units: map[string]unitState{
		reasonRunnerUnit: {loadState: loadStateLoaded, activeState: systemdActiveState("surprising"), fragmentPath: reasonRunnerUnitPath},
	}}

	err := execute(context.Background(), root, actionPreflight, systemd, &bytes.Buffer{})
	if !errors.Is(err, errUnexpectedSystemdState) {
		t.Fatalf("preflight error = %v, want unexpected state", err)
	}
	if len(systemd.mutations) != 0 {
		t.Fatalf("preflight mutated systemd: %v", systemd.mutations)
	}
}

func retirementFixtureRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, target := range retiredHostPaths {
		writeRetirementFixture(t, filepath.Join(hostPath(root, target), "fixture"), "retired\n")
	}
	for _, target := range []string{reasonRunnerUnitPath, neo4jUnitPath} {
		writeRetirementFixture(t, hostPath(root, target), "unit\n")
	}
	return root
}

func writeRetirementFixture(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
