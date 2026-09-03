package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	hostRoot             = "/host"
	reasonRunnerUnit     = "actions.runner.meierlink88-tidewise-reason.tidewise-reason-uat-ecs.service"
	reasonRunnerUnitPath = "/etc/systemd/system/actions.runner.meierlink88-tidewise-reason.tidewise-reason-uat-ecs.service"
	neo4jUnit            = "neo4j.service"
	neo4jUnitPath        = "/etc/systemd/system/neo4j.service"
)

var retiredHostPaths = []string{
	"/opt/tidewise/agentos-uat",
	"/opt/tidewise/uat/agentrun-artifacts",
	"/opt/tidewise/uat/logs/agentrun",
	"/opt/tidewise/reason-uat",
	"/opt/tidewise/neo4j-uat",
}

var retiredSystemdUnits = []unitTarget{
	{name: reasonRunnerUnit, fragmentPath: reasonRunnerUnitPath},
	{name: neo4jUnit, fragmentPath: neo4jUnitPath},
}

var (
	errInvalidAction             = errors.New("invalid retirement action")
	errUnexpectedSystemdState    = errors.New("unexpected systemd state")
	errUnexpectedSystemdFragment = errors.New("unexpected systemd fragment")
	errSymbolicLinkTarget        = errors.New("retirement target traverses symbolic link")
)

type retirementAction string

const (
	actionPreflight retirementAction = "preflight"
	actionApply     retirementAction = "apply"
)

type systemdLoadState string

const (
	loadStateLoaded   systemdLoadState = "loaded"
	loadStateNotFound systemdLoadState = "not-found"
)

type systemdActiveState string

const (
	activeStateActive       systemdActiveState = "active"
	activeStateReloading    systemdActiveState = "reloading"
	activeStateInactive     systemdActiveState = "inactive"
	activeStateFailed       systemdActiveState = "failed"
	activeStateActivating   systemdActiveState = "activating"
	activeStateDeactivating systemdActiveState = "deactivating"
	activeStateMaintenance  systemdActiveState = "maintenance"
	activeStateRefreshing   systemdActiveState = "refreshing"
)

type unitTarget struct {
	name         string
	fragmentPath string
}

type unitState struct {
	loadState    systemdLoadState
	activeState  systemdActiveState
	fragmentPath string
}

type systemdController interface {
	inspect(context.Context, string) (unitState, error)
	disableAndStop(context.Context, string) error
	reload(context.Context) error
}

type hostSystemd struct {
	root string
}

func main() {
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "FAIL root-retirement: root identity is required inside the bounded container")
		os.Exit(1)
	}
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "FAIL root-retirement: expected exactly one action: preflight or apply")
		os.Exit(1)
	}
	action, err := parseAction(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAIL root-retirement: expected exactly one action: preflight or apply")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := execute(ctx, hostRoot, action, hostSystemd{root: hostRoot}, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL root-retirement: %v\n", err)
		os.Exit(1)
	}
}

func parseAction(value string) (retirementAction, error) {
	action := retirementAction(value)
	if action != actionPreflight && action != actionApply {
		return "", fmt.Errorf("%w: %q", errInvalidAction, value)
	}
	return action, nil
}

func execute(ctx context.Context, root string, action retirementAction, systemd systemdController, output io.Writer) error {
	if action != actionPreflight && action != actionApply {
		return fmt.Errorf("%w: %q", errInvalidAction, action)
	}
	states, err := validateTargets(ctx, root, systemd)
	if err != nil {
		return err
	}
	if action == actionPreflight {
		fmt.Fprintln(output, "PASS root-retirement-preflight")
		return nil
	}

	for _, target := range retiredSystemdUnits {
		state := states[target.name]
		if state.loadState == loadStateNotFound {
			fmt.Fprintf(output, "ABSENT unit %s\n", target.name)
		} else {
			if err := systemd.disableAndStop(ctx, target.name); err != nil {
				return fmt.Errorf("disable and stop unit %s: %w", target.name, err)
			}
			stopped, err := systemd.inspect(ctx, target.name)
			if err != nil {
				return fmt.Errorf("verify stopped unit %s: %w", target.name, err)
			}
			if stopped.loadState != loadStateLoaded || stopped.activeState != activeStateInactive || stopped.fragmentPath != target.fragmentPath {
				return fmt.Errorf("%w: unit %s after stop has load_state=%s active_state=%s fragment=%s", errUnexpectedSystemdState, target.name, stopped.loadState, stopped.activeState, stopped.fragmentPath)
			}
			fmt.Fprintf(output, "STOPPED unit %s\n", target.name)
		}
		if err := removeExactFile(root, target.fragmentPath); err != nil {
			return fmt.Errorf("remove unit file %s: %w", target.fragmentPath, err)
		}
	}
	if err := systemd.reload(ctx); err != nil {
		return fmt.Errorf("reload systemd: %w", err)
	}

	for _, target := range retiredHostPaths {
		path := hostPath(root, target)
		if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(output, "ABSENT path %s\n", target)
			continue
		} else if err != nil {
			return fmt.Errorf("inspect path %s: %w", target, err)
		}
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove path %s: %w", target, err)
		}
		fmt.Fprintf(output, "REMOVED path %s\n", target)
	}

	for _, target := range retiredSystemdUnits {
		state, err := systemd.inspect(ctx, target.name)
		if err != nil {
			return fmt.Errorf("verify removed unit %s: %w", target.name, err)
		}
		if state.loadState != loadStateNotFound || state.activeState != activeStateInactive || state.fragmentPath != "" {
			return fmt.Errorf("unit %s remains load_state=%s fragment=%s", target.name, state.loadState, state.fragmentPath)
		}
		if _, err := os.Lstat(hostPath(root, target.fragmentPath)); !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("unit file %s remains", target.fragmentPath)
		}
	}
	for _, target := range retiredHostPaths {
		if _, err := os.Lstat(hostPath(root, target)); !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("path %s remains", target)
		}
	}
	fmt.Fprintln(output, "PASS root-retirement-apply")
	return nil
}

func validateTargets(ctx context.Context, root string, systemd systemdController) (map[string]unitState, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) == string(filepath.Separator) {
		return nil, fmt.Errorf("host root must be a bounded absolute mount")
	}
	for _, target := range retiredHostPaths {
		if err := validateNoSymlinkPath(root, target); err != nil {
			return nil, err
		}
	}

	states := make(map[string]unitState, len(retiredSystemdUnits))
	for _, target := range retiredSystemdUnits {
		if err := validateNoSymlinkPath(root, target.fragmentPath); err != nil {
			return nil, err
		}
		state, err := systemd.inspect(ctx, target.name)
		if err != nil {
			return nil, fmt.Errorf("inspect unit %s: %w", target.name, err)
		}
		if err := validateUnitState(state); err != nil {
			return nil, fmt.Errorf("unit %s: %w", target.name, err)
		}
		if state.loadState == loadStateNotFound {
			if state.fragmentPath != "" {
				return nil, fmt.Errorf("%w: unit %s is not found but has fragment %s", errUnexpectedSystemdFragment, target.name, state.fragmentPath)
			}
			if state.activeState != activeStateInactive {
				return nil, fmt.Errorf("%w: unit %s is not found but active_state=%s", errUnexpectedSystemdState, target.name, state.activeState)
			}
		} else if state.fragmentPath != target.fragmentPath {
			return nil, fmt.Errorf("%w: unit %s has fragment %s", errUnexpectedSystemdFragment, target.name, state.fragmentPath)
		}
		states[target.name] = state
	}
	return states, nil
}

func validateUnitState(state unitState) error {
	if state.loadState != loadStateLoaded && state.loadState != loadStateNotFound {
		return fmt.Errorf("%w: load_state=%q", errUnexpectedSystemdState, state.loadState)
	}
	switch state.activeState {
	case activeStateActive, activeStateReloading, activeStateInactive, activeStateFailed,
		activeStateActivating, activeStateDeactivating, activeStateMaintenance, activeStateRefreshing:
		return nil
	default:
		return fmt.Errorf("%w: active_state=%q", errUnexpectedSystemdState, state.activeState)
	}
}

func validateNoSymlinkPath(root, target string) error {
	cleanTarget := filepath.Clean(target)
	if !filepath.IsAbs(cleanTarget) || cleanTarget == string(filepath.Separator) {
		return fmt.Errorf("target %q is not a bounded absolute path", target)
	}
	current := filepath.Clean(root)
	parts := strings.Split(strings.TrimPrefix(cleanTarget, string(filepath.Separator)), string(filepath.Separator))
	for _, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect target %s: %w", target, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: target %s via %s", errSymbolicLinkTarget, target, current)
		}
	}
	return nil
}

func removeExactFile(root, target string) error {
	path := hostPath(root, target)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("expected a file, found directory")
	}
	return os.Remove(path)
}

func hostPath(root, target string) string {
	return filepath.Join(filepath.Clean(root), strings.TrimPrefix(filepath.Clean(target), string(filepath.Separator)))
}

func (systemd hostSystemd) inspect(ctx context.Context, name string) (unitState, error) {
	output, err := systemd.run(ctx, "show", "--property=LoadState", "--property=ActiveState", "--property=FragmentPath", name)
	if err != nil {
		return unitState{}, err
	}
	var loadState string
	var activeState string
	state := unitState{}
	for _, line := range strings.Split(string(output), "\n") {
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		switch key {
		case "LoadState":
			loadState = value
		case "ActiveState":
			activeState = value
		case "FragmentPath":
			state.fragmentPath = value
		}
	}
	state.loadState = systemdLoadState(loadState)
	state.activeState = systemdActiveState(activeState)
	return state, nil
}

func (systemd hostSystemd) disableAndStop(ctx context.Context, name string) error {
	_, err := systemd.run(ctx, "disable", "--now", name)
	return err
}

func (systemd hostSystemd) reload(ctx context.Context) error {
	_, err := systemd.run(ctx, "daemon-reload")
	return err
}

func (systemd hostSystemd) run(ctx context.Context, arguments ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, "/usr/bin/systemctl", arguments...)
	command.Dir = "/"
	command.SysProcAttr = &syscall.SysProcAttr{Chroot: systemd.root}
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("systemctl %s failed: %w", arguments[0], err)
	}
	return output, nil
}
