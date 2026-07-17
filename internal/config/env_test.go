package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadEnvFile(t *testing.T) {
	const plainKey = "TIDEWISE_TEST_ENV_PLAIN"
	const quotedKey = "TIDEWISE_TEST_ENV_QUOTED"
	const exportKey = "TIDEWISE_TEST_ENV_EXPORT"
	unsetForTest(t, plainKey, quotedKey, exportKey)

	path := filepath.Join(t.TempDir(), ".env")
	content := strings.Join([]string{
		"# collector credentials",
		plainKey + "=plain-value",
		quotedKey + "=\"value with spaces\"",
		"export " + exportKey + "='exported-value'",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := LoadEnvFile(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv(plainKey); got != "plain-value" {
		t.Fatalf("plain value = %q", got)
	}
	if got := os.Getenv(quotedKey); got != "value with spaces" {
		t.Fatalf("quoted value = %q", got)
	}
	if got := os.Getenv(exportKey); got != "exported-value" {
		t.Fatalf("exported value = %q", got)
	}
}

func TestLoadEnvFileDoesNotOverrideProcessEnvironment(t *testing.T) {
	const key = "TIDEWISE_TEST_ENV_PRECEDENCE"
	t.Setenv(key, "injected-value")

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(key+"=file-value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := LoadEnvFile(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv(key); got != "injected-value" {
		t.Fatalf("value = %q, want injected-value", got)
	}
}

func TestLoadEnvFileAllowsMissingFile(t *testing.T) {
	if err := LoadEnvFile(filepath.Join(t.TempDir(), "missing.env")); err != nil {
		t.Fatal(err)
	}
}

func TestLoadEnvFileRejectsInvalidAssignmentWithoutLeakingValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("INVALID-NAME=secret-value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := LoadEnvFile(path)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if strings.Contains(err.Error(), "secret-value") {
		t.Fatal("error leaked environment value")
	}
}

func unsetForTest(t *testing.T, names ...string) {
	t.Helper()
	for _, name := range names {
		value, existed := os.LookupEnv(name)
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if existed {
				_ = os.Setenv(name, value)
			} else {
				_ = os.Unsetenv(name)
			}
		})
	}
}
