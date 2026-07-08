package architecture

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"
)

func TestIngestionCommandsUseSubsystemBoundary(t *testing.T) {
	packages := listCommandPackages(t)
	for _, pkg := range packages {
		switch {
		case strings.HasSuffix(pkg.ImportPath, "/cmd/source-ingest"):
			assertNoImport(t, pkg, "/internal/jobs")
			assertNoImport(t, pkg, "/internal/integrations")
		case strings.HasSuffix(pkg.ImportPath, "/cmd/source-seed"):
			assertNoImport(t, pkg, "/internal/sourcecatalog")
		case strings.HasSuffix(pkg.ImportPath, "/cmd/ingest-smoke"):
			assertNoImport(t, pkg, "/internal/jobs")
		case strings.HasSuffix(pkg.ImportPath, "/cmd/entity-seed"):
			assertNoImport(t, pkg, "/internal/entityseed")
			assertNoImport(t, pkg, "/internal/apps/ingestion")
		case strings.HasSuffix(pkg.ImportPath, "/cmd/dbmigrate"):
			assertNoImport(t, pkg, "/internal/migrations")
		}
	}
}

func listCommandPackages(t *testing.T) []packageInfo {
	t.Helper()

	command := exec.Command("go", "list", "-json", "./cmd/...")
	command.Dir = "../.."
	output, err := command.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Fatalf("go list failed: %v\n%s", err, string(exitErr.Stderr))
		}
		t.Fatalf("go list failed: %v", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(output))
	var packages []packageInfo
	for {
		var pkg packageInfo
		if err := decoder.Decode(&pkg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatalf("decode go list output: %v", err)
		}
		packages = append(packages, pkg)
	}
	return packages
}
