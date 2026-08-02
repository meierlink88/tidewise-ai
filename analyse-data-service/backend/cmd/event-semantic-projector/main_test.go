package main

import "testing"

func TestProjectorRequiresExplicitApplyAuthorization(t *testing.T) {
	for _, args := range [][]string{{}, {"-apply"}, {"-allow-env", "local"}} {
		if _, err := parseOptions(args); err == nil {
			t.Fatalf("parseOptions(%v) unexpectedly succeeded", args)
		}
	}
	parsed, err := parseOptions([]string{"-apply", "-allow-env", "local"})
	if err != nil || !parsed.Apply || parsed.AllowEnv != "local" {
		t.Fatalf("parsed=%#v err=%v", parsed, err)
	}
}
