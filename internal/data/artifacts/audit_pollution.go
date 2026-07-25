package artifacts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var scientificIdentifierPattern = regexp.MustCompile(`\b[0-9]+\.[0-9]+e[+-][0-9]+\b`)
var splitChinesePattern = regexp.MustCompile(`[\p{Han}]\s+[\p{Han}]`)
var splitDecimalPattern = regexp.MustCompile(`[0-9]\.\s+[0-9]`)

type PollutionFinding struct {
	Path    string   `json:"path"`
	SHA256  string   `json:"sha256"`
	Reasons []string `json:"reasons"`
}

type ArtifactIdentity struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Kind   string `json:"kind"`
}

type PollutionAuditReport struct {
	Documents int                `json:"documents"`
	IndexRows int                `json:"index_rows"`
	Ledgers   int                `json:"candidate_ledgers"`
	Files     []ArtifactIdentity `json:"files"`
	Findings  []PollutionFinding `json:"findings"`
}

func AuditPollution(root string) (PollutionAuditReport, error) {
	if root == "" {
		root = "data"
	}
	var report PollutionAuditReport
	documentPaths, err := matchingFiles(filepath.Join(root, "documents"), ".md")
	if err != nil {
		return PollutionAuditReport{}, err
	}
	report.Documents = len(documentPaths)
	ledgerPaths, err := namedFiles(filepath.Join(root, "runs"), "candidates.jsonl")
	if err != nil {
		return PollutionAuditReport{}, err
	}
	report.Ledgers = len(ledgerPaths)
	summaryPaths, err := namedFiles(filepath.Join(root, "runs"), "summary.md")
	if err != nil {
		return PollutionAuditReport{}, err
	}
	manifestPaths, err := namedFiles(filepath.Join(root, "runs"), "manifest.json")
	if err != nil {
		return PollutionAuditReport{}, err
	}
	type auditFile struct {
		path string
		kind string
	}
	files := make([]auditFile, 0, len(documentPaths)+len(ledgerPaths)+len(summaryPaths)+len(manifestPaths)+1)
	for _, path := range documentPaths {
		files = append(files, auditFile{path: path, kind: "accepted_document"})
	}
	for _, path := range ledgerPaths {
		files = append(files, auditFile{path: path, kind: "candidate_ledger"})
	}
	for _, path := range summaryPaths {
		files = append(files, auditFile{path: path, kind: "run_summary"})
	}
	for _, path := range manifestPaths {
		files = append(files, auditFile{path: path, kind: "run_manifest"})
	}
	indexPath := filepath.Join(root, "indexes", "dedup-index.tsv")
	if _, err := os.Stat(indexPath); err == nil {
		files = append(files, auditFile{path: indexPath, kind: "dedup_index"})
	} else if !os.IsNotExist(err) {
		return PollutionAuditReport{}, err
	}
	sort.Slice(files, func(left, right int) bool {
		return files[left].path < files[right].path
	})
	for _, file := range files {
		path := file.path
		payload, err := os.ReadFile(path)
		if err != nil {
			return PollutionAuditReport{}, fmt.Errorf("read Artifact audit input %s: %w", path, err)
		}
		text := string(payload)
		var reasons []string
		if scientificIdentifierPattern.MatchString(text) {
			reasons = append(reasons, "scientific_notation_identifier")
		}
		if strings.HasSuffix(path, ".md") &&
			(splitChinesePattern.MatchString(text) || splitDecimalPattern.MatchString(text)) {
			reasons = append(reasons, "suspected_html_tag_spacing_damage")
		}
		digest, err := fileSHA256(path)
		if err != nil {
			return PollutionAuditReport{}, err
		}
		report.Files = append(report.Files, ArtifactIdentity{Path: path, SHA256: digest, Kind: file.kind})
		if len(reasons) > 0 {
			report.Findings = append(report.Findings, PollutionFinding{
				Path: path, SHA256: digest, Reasons: reasons,
			})
		}
	}
	records, err := loadIndex(indexPath)
	if errors.Is(err, os.ErrNotExist) {
		report.IndexRows = 0
	} else if err != nil {
		return PollutionAuditReport{}, err
	} else {
		report.IndexRows = len(records)
	}
	sort.Slice(report.Findings, func(left, right int) bool {
		return report.Findings[left].Path < report.Findings[right].Path
	})
	return report, nil
}

func matchingFiles(root, extension string) ([]string, error) {
	return walkMatching(root, func(path string) bool {
		return strings.EqualFold(filepath.Ext(path), extension)
	})
}

func namedFiles(root, name string) ([]string, error) {
	return walkMatching(root, func(path string) bool {
		return filepath.Base(path) == name
	})
}

func walkMatching(root string, match func(string) bool) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && match(path) {
			paths = append(paths, path)
		}
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan Artifact audit inputs: %w", err)
	}
	sort.Strings(paths)
	return paths, nil
}
