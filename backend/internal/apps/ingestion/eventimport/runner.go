package eventimport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	domainimport "github.com/meierlink88/tidewise-ai/backend/internal/domain/eventimport"
)

func LoadPackages(path string) ([]domainimport.Package, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat input: %w", err)
	}
	if info.IsDir() {
		return LoadPackagesFromInput("", path)
	}
	return LoadPackagesFromInput(path, "")
}

func LoadPackagesFromInput(file, dir string) ([]domainimport.Package, error) {
	if file == "" && dir == "" {
		return nil, errors.New("exactly one of file or dir is required")
	}
	if file != "" && dir != "" {
		return nil, errors.New("file and dir are mutually exclusive")
	}
	path := file
	if dir != "" {
		path = dir
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat input: %w", err)
	}
	paths := []string{path}
	if info.IsDir() {
		if file != "" {
			return nil, fmt.Errorf("file input %q is a directory", file)
		}
		paths = paths[:0]
		err = filepath.WalkDir(path, func(candidate string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !entry.IsDir() && strings.EqualFold(filepath.Ext(candidate), ".json") {
				paths = append(paths, candidate)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walk input directory: %w", err)
		}
		sort.Strings(paths)
	} else if dir != "" {
		return nil, fmt.Errorf("dir input %q is not a directory", dir)
	}
	if len(paths) == 0 {
		return nil, errors.New("input directory contains no .json files")
	}

	packages := make([]domainimport.Package, 0, len(paths))
	for _, filename := range paths {
		content, readErr := os.ReadFile(filename)
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", filename, readErr)
		}
		pkg, decodeErr := domainimport.DecodeStrict(bytes.NewReader(content))
		if decodeErr != nil {
			return nil, fmt.Errorf("decode %s: %w", filename, decodeErr)
		}
		if _, validateErr := pkg.Validate(); validateErr != nil {
			return nil, fmt.Errorf("validate %s: %w", filename, validateErr)
		}
		packages = append(packages, pkg)
	}
	return packages, nil
}

func DryRun(ctx context.Context, packages []domainimport.Package) ([]Plan, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	plans := make([]Plan, 0, len(packages))
	service := NewService(nil)
	for _, pkg := range packages {
		plan, err := service.Plan(pkg)
		if err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

func EncodeReport(v any) ([]byte, error) {
	return json.Marshal(v)
}
