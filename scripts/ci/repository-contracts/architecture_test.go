package architecture

import (
	"path/filepath"
	"strings"
)

type packageInfo struct {
	ImportPath string
	Imports    []string
}

func hasPackageSuffix(packages []packageInfo, suffix string) bool {
	for _, pkg := range packages {
		if strings.HasSuffix(pkg.ImportPath, suffix) {
			return true
		}
	}
	return false
}

func localPackageName(importPath string) string {
	return strings.TrimPrefix(importPath, "github.com/meierlink88/tidewise-ai/")
}

func repositoryRoot() string {
	return filepath.Clean(filepath.Join("..", "..", ".."))
}
