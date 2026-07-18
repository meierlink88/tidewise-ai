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

type packageInfo struct {
	ImportPath string
	Imports    []string
}

func TestInternalTreeContainsOnlyBusinessFreePlatformCode(t *testing.T) {
	for _, pkg := range listInternalPackages(t) {
		owner := localPackageName(pkg.ImportPath)
		if owner == "internal/architecture" || strings.HasPrefix(owner, "internal/platform/") || owner == "internal/platform" {
			continue
		}
		t.Errorf("business package %q must be owned by one of the deployable services", owner)
	}
}

func TestSharedPlatformDoesNotDependOnServices(t *testing.T) {
	for _, pkg := range listInternalPackages(t) {
		owner := localPackageName(pkg.ImportPath)
		if owner == "internal/platform" || strings.HasPrefix(owner, "internal/platform/") {
			assertNoImport(t, pkg, "/services/")
		}
	}
}

func listInternalPackages(t *testing.T) []packageInfo {
	t.Helper()

	command := exec.Command("go", "list", "-json", "./internal/...")
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

func hasPackageSuffix(packages []packageInfo, suffix string) bool {
	for _, pkg := range packages {
		if strings.HasSuffix(pkg.ImportPath, suffix) {
			return true
		}
	}
	return false
}

func assertNoImport(t *testing.T, pkg packageInfo, forbidden string) {
	t.Helper()

	for _, imported := range pkg.Imports {
		if containsPath(imported, forbidden) {
			t.Fatalf("%s must not import %s", pkg.ImportPath, imported)
		}
	}
}

func containsPath(importPath string, fragment string) bool {
	return strings.Contains(importPath, fragment)
}

func localPackageName(importPath string) string {
	const marker = "/backend/"
	if index := strings.Index(importPath, marker); index >= 0 {
		return importPath[index+len(marker):]
	}
	return importPath
}
