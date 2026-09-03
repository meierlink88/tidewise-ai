package main

import (
	"fmt"
	"os"
)

type fakeUnit struct {
	fragmentPath string
	enabledPath  string
}

var units = map[string]fakeUnit{
	"actions.runner.meierlink88-tidewise-reason.tidewise-reason-uat-ecs.service": {
		fragmentPath: "/etc/systemd/system/actions.runner.meierlink88-tidewise-reason.tidewise-reason-uat-ecs.service",
		enabledPath:  "/etc/systemd/system/.uat-fake-reason-enabled",
	},
	"neo4j.service": {
		fragmentPath: "/usr/lib/systemd/system/neo4j.service",
		enabledPath:  "/etc/systemd/system/.uat-fake-neo4j-enabled",
	},
}

func main() {
	if len(os.Args) == 3 && os.Args[1] == "disable" && os.Args[2] == "--now" {
		fail("missing unit name")
	}
	if len(os.Args) == 4 && os.Args[1] == "disable" && os.Args[2] == "--now" {
		unit := requireKnownUnit(os.Args[3])
		if err := os.Remove(unit.enabledPath); err != nil && !os.IsNotExist(err) {
			fail("cannot disable unit")
		}
		return
	}
	if len(os.Args) == 2 && os.Args[1] == "daemon-reload" {
		return
	}
	if len(os.Args) == 7 && os.Args[1] == "show" &&
		os.Args[2] == "--property=LoadState" &&
		os.Args[3] == "--property=ActiveState" &&
		os.Args[4] == "--property=UnitFileState" &&
		os.Args[5] == "--property=FragmentPath" {
		show(os.Args[6])
		return
	}
	fail("unsupported invocation")
}

func show(unit string) {
	target := requireKnownUnit(unit)
	if _, err := os.Lstat(target.fragmentPath); err == nil {
		unitFileState := "disabled"
		if _, enabledErr := os.Lstat(target.enabledPath); enabledErr == nil {
			unitFileState = "enabled"
		} else if !os.IsNotExist(enabledErr) {
			fail("cannot inspect unit enablement")
		}
		fmt.Printf("LoadState=loaded\nActiveState=inactive\nUnitFileState=%s\nFragmentPath=%s\n", unitFileState, target.fragmentPath)
		return
	} else if !os.IsNotExist(err) {
		fail("cannot inspect unit")
	}
	fmt.Print("LoadState=not-found\nActiveState=inactive\nUnitFileState=\nFragmentPath=\n")
}

func requireKnownUnit(unit string) fakeUnit {
	target, ok := units[unit]
	if !ok {
		fail("unknown unit")
	}
	return target
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
