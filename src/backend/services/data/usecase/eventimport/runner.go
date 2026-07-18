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

	domainimport "github.com/meierlink88/tidewise-ai/backend/services/data/domain/eventimport"
)

var (
	ErrInputValidation = errors.New("event import input validation failed")
	ErrInputIO         = errors.New("event import input I/O failed")
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
		return nil, fmt.Errorf("%w: exactly one of file or dir is required", ErrInputValidation)
	}
	if file != "" && dir != "" {
		return nil, fmt.Errorf("%w: file and dir are mutually exclusive", ErrInputValidation)
	}
	path := file
	if dir != "" {
		path = dir
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("%w: stat input: %v", ErrInputIO, err)
	}
	paths := []string{path}
	if info.IsDir() {
		if file != "" {
			return nil, fmt.Errorf("%w: file input %q is a directory", ErrInputValidation, file)
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
			return nil, fmt.Errorf("%w: walk input directory: %v", ErrInputIO, err)
		}
		sort.Strings(paths)
	} else if dir != "" {
		return nil, fmt.Errorf("%w: dir input %q is not a directory", ErrInputValidation, dir)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("%w: input directory contains no .json files", ErrInputValidation)
	}

	packages := make([]domainimport.Package, 0, len(paths))
	for _, filename := range paths {
		content, readErr := os.ReadFile(filename)
		if readErr != nil {
			return nil, fmt.Errorf("%w: read %s: %v", ErrInputIO, filename, readErr)
		}
		pkg, decodeErr := domainimport.DecodeStrict(bytes.NewReader(content))
		if decodeErr != nil {
			return nil, fmt.Errorf("%w: decode %s: %v", ErrInputValidation, filename, decodeErr)
		}
		if _, validateErr := pkg.Validate(); validateErr != nil {
			return nil, fmt.Errorf("%w: validate %s: %v", ErrInputValidation, filename, validateErr)
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
