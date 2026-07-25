package architecture

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplicationBackendOwnsRuntimeMechanisms(t *testing.T) {
	backendRoot := filepath.Clean(filepath.Join("..", ".."))
	for _, path := range []string{
		"config",
		"transport",
		"usecase",
		"http_runtime.go",
	} {
		if _, err := os.Stat(filepath.Join(backendRoot, path)); err != nil {
			t.Errorf("Admin Portal Backend-owned path %q is missing: %v", path, err)
		}
	}
}
