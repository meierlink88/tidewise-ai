package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunRequiresInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if got := run(nil, &stdout, &stderr); got != exitUsage {
		t.Fatalf("exit code = %d, want %d", got, exitUsage)
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
