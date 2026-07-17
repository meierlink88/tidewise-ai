package collector

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCollectorPromptAcceptsAbsoluteRelativeAndSymlinkPaths(t *testing.T) {
	tempDir := t.TempDir()
	want := "intent line one\nintent line two\n"
	absolutePath := filepath.Join(tempDir, "intent.md")
	if err := os.WriteFile(absolutePath, []byte(want), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := LoadCollectorPrompt(absolutePath)
	if err != nil || got != want {
		t.Fatalf("absolute load = %q, %v", got, err)
	}

	linkPath := filepath.Join(tempDir, "intent-link.md")
	if err = os.Symlink(absolutePath, linkPath); err != nil {
		t.Fatal(err)
	}
	got, err = LoadCollectorPrompt(linkPath)
	if err != nil || got != want {
		t.Fatalf("symlink load = %q, %v", got, err)
	}

	previousWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if restoreErr := os.Chdir(previousWorkingDirectory); restoreErr != nil {
			t.Errorf("restore working directory: %v", restoreErr)
		}
	})
	got, err = LoadCollectorPrompt("intent.md")
	if err != nil || got != want {
		t.Fatalf("relative load = %q, %v", got, err)
	}
}

func TestLoadCollectorPromptSanitizesUnderlyingOpenError(t *testing.T) {
	const prompt = "complete secret prompt"
	const rawResponse = `{"raw":"secret response"}`
	const apiKey = "secret-api-key"
	path := filepath.Join(t.TempDir(), "intent.md")
	_, err := loadCollectorPrompt(path, func(string) (*os.File, error) {
		return nil, fmt.Errorf("open failed prompt=%s response=%s key=%s", prompt, rawResponse, apiKey)
	})
	if err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("open error = %v", err)
	}
	for _, secret := range []string{prompt, rawResponse, apiKey} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("error leaked %q: %v", secret, err)
		}
	}
}

func TestLoadCollectorPromptReloadsFileWithoutMutatingLoadedValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "intent.md")
	if err := os.WriteFile(path, []byte("intent A\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	loadedA, err := LoadCollectorPrompt(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, []byte("intent B\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	loadedB, err := LoadCollectorPrompt(path)
	if err != nil {
		t.Fatal(err)
	}
	if loadedA != "intent A\n" || loadedB != "intent B\n" {
		t.Fatalf("loadedA=%q loadedB=%q", loadedA, loadedB)
	}
}

func TestLoadCollectorPromptEnforcesSizeUTF8AndNonEmptyRules(t *testing.T) {
	tests := map[string][]byte{
		"blank":         []byte(" \n\t"),
		"too large":     []byte(strings.Repeat("a", 64*1024+1)),
		"invalid UTF-8": append([]byte("complete secret prompt"), 0xff),
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "intent.md")
			if err := os.WriteFile(path, content, 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := LoadCollectorPrompt(path)
			if err == nil {
				t.Fatal("expected invalid prompt error")
			}
			if strings.Contains(err.Error(), string(content)) || strings.Contains(err.Error(), "complete secret prompt") {
				t.Fatalf("error leaked prompt content: %v", err)
			}
		})
	}

	exactLimit := strings.Repeat("界", (64*1024)/3) + "a"
	if len([]byte(exactLimit)) != 64*1024 {
		t.Fatalf("test fixture size = %d", len([]byte(exactLimit)))
	}
	path := filepath.Join(t.TempDir(), "intent.md")
	if err := os.WriteFile(path, []byte(exactLimit), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := LoadCollectorPrompt(path)
	if err != nil || got != exactLimit {
		t.Fatalf("exact limit load bytes=%d err=%v", len([]byte(got)), err)
	}
}

func TestLoadCollectorPromptRejectsMissingDirectoryAndUnreadableFilesSafely(t *testing.T) {
	tempDir := t.TempDir()
	missingPath := filepath.Join(tempDir, "missing.md")
	_, err := LoadCollectorPrompt(missingPath)
	if err == nil || !strings.Contains(err.Error(), missingPath) {
		t.Fatalf("missing error = %v", err)
	}

	_, err = LoadCollectorPrompt(tempDir)
	if err == nil || !strings.Contains(err.Error(), tempDir) {
		t.Fatalf("directory error = %v", err)
	}

	unreadablePath := filepath.Join(tempDir, "unreadable.md")
	if err = os.WriteFile(unreadablePath, []byte("secret prompt content"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(unreadablePath, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(unreadablePath, 0o600) })
	_, err = LoadCollectorPrompt(unreadablePath)
	if err == nil {
		t.Skip("platform permits reading a mode-000 file; unreadable behavior cannot be asserted")
	}
	if strings.Contains(err.Error(), "secret prompt content") {
		t.Fatalf("unreadable error leaked prompt content: %v", err)
	}
}
