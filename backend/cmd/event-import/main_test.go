package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRequiresInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if got := run(nil, &stdout, &stderr); got != exitUsage {
		t.Fatalf("exit code = %d, want %d", got, exitUsage)
	}
}

func TestRunSupportsFrozenFileAndDirFlagsWithStableResultObject(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "..", "testdata", "event-import", "reviewed-outbox-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "one.json")
	if err := os.WriteFile(file, fixture, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"--file", file, "--dry-run"}, {"--dir", dir, "--dry-run"}} {
		var stdout, stderr bytes.Buffer
		if got := run(args, &stdout, &stderr); got != exitOK {
			t.Fatalf("args %v exit code = %d, stderr=%s", args, got, stderr.String())
		}
		var body map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &body); err != nil {
			t.Fatalf("args %v output is not JSON: %v", args, err)
		}
		if body["ok"] != true || body["mode"] != "dry-run" {
			t.Fatalf("unexpected output: %s", stdout.String())
		}
		result, ok := body["result"].(map[string]any)
		if !ok || result["package_count"] != float64(1) {
			t.Fatalf("result is not stable object: %s", stdout.String())
		}
	}
	var stdout, stderr bytes.Buffer
	if got := run([]string{"--file", file, "--dir", dir}, &stdout, &stderr); got != exitUsage || !strings.Contains(stderr.String(), "exactly one") {
		t.Fatalf("mutually exclusive flags exit=%d stderr=%q", got, stderr.String())
	}
}

func TestRunRejectsMalformedInputWithoutDatabase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "outbox.json")
	if err := os.WriteFile(path, []byte(`{"unexpected":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if got := run([]string{"-input", path, "-dry-run"}, &stdout, &stderr); got != exitRejected {
		t.Fatalf("exit code = %d, want %d", got, exitRejected)
	}
	if bytes.Contains(stdout.Bytes(), []byte("password")) {
		t.Fatal("CLI output contains a credential-like field")
	}
}
