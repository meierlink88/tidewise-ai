package main

import (
	"fmt"
	"os"
)

var unitPaths = map[string]string{
	"actions.runner.meierlink88-tidewise-reason.tidewise-reason-uat-ecs.service": "/etc/systemd/system/actions.runner.meierlink88-tidewise-reason.tidewise-reason-uat-ecs.service",
	"neo4j.service": "/etc/systemd/system/neo4j.service",
}

func main() {
	if len(os.Args) == 3 && os.Args[1] == "disable" && os.Args[2] == "--now" {
		fail("missing unit name")
	}
	if len(os.Args) == 4 && os.Args[1] == "disable" && os.Args[2] == "--now" {
		requireKnownUnit(os.Args[3])
		return
	}
	if len(os.Args) == 2 && os.Args[1] == "daemon-reload" {
		return
	}
	if len(os.Args) == 6 && os.Args[1] == "show" &&
		os.Args[2] == "--property=LoadState" &&
		os.Args[3] == "--property=ActiveState" &&
		os.Args[4] == "--property=FragmentPath" {
		show(os.Args[5])
		return
	}
	fail("unsupported invocation")
}

func show(unit string) {
	path := requireKnownUnit(unit)
	if _, err := os.Lstat(path); err == nil {
		fmt.Printf("LoadState=loaded\nActiveState=inactive\nFragmentPath=%s\n", path)
		return
	} else if !os.IsNotExist(err) {
		fail("cannot inspect unit")
	}
	fmt.Print("LoadState=not-found\nActiveState=inactive\nFragmentPath=\n")
}

func requireKnownUnit(unit string) string {
	path, ok := unitPaths[unit]
	if !ok {
		fail("unknown unit")
	}
	return path
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
