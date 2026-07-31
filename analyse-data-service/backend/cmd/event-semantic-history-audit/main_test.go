package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestAuditOutputNeverOverwritesAnExistingManifest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.json")
	if err := os.WriteFile(path, []byte("reviewed"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, _, err := outputWriter(path); err == nil {
		t.Fatal("existing audit manifest was overwritten")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "reviewed" {
		t.Fatalf("content = %q", content)
	}
}

func TestAuditOutputCreatesPrivateManifestFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.json")
	writer, closeWriter, err := outputWriter(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(writer, "{}"); err != nil {
		t.Fatal(err)
	}
	closeWriter()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
}
