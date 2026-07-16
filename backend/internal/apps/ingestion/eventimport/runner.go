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

type InputReport struct {
	Files         int      `json:"files"`
	Packages      int      `json:"packages"`
	PackageIDs    []string `json:"package_ids"`
	PayloadHashes []string `json:"payload_hashes"`
	DryRun        bool     `json:"dry_run"`
}

func LoadPackages(path string) ([]domainimport.Package, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat input: %w", err)
	}
	paths := []string{path}
	if info.IsDir() {
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

func DryRun(ctx context.Context, packages []domainimport.Package) (InputReport, error) {
	if err := ctx.Err(); err != nil {
		return InputReport{}, err
	}
	ids := make([]string, 0, len(packages))
	hashes := make([]string, 0, len(packages))
	for _, pkg := range packages {
		if _, err := pkg.Validate(); err != nil {
			return InputReport{}, err
		}
		ids = append(ids, pkg.PackageID)
		hash, err := pkg.CanonicalHash()
		if err != nil {
			return InputReport{}, err
		}
		hashes = append(hashes, hash)
	}
	return InputReport{Files: len(packages), Packages: len(packages), PackageIDs: ids, PayloadHashes: hashes, DryRun: true}, nil
}

func EncodeReport(v any) ([]byte, error) {
	return json.Marshal(v)
}
