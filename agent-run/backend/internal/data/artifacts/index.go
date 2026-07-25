package artifacts

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector/materialization"
	"gopkg.in/yaml.v3"
)

type documentMetadata struct {
	DocumentID    string `yaml:"document_id"`
	SourceURL     string `yaml:"source_url"`
	PublishedAt   string `yaml:"published_at"`
	ContentSHA256 string `yaml:"content_sha256"`
	QualityStatus string `yaml:"quality_status"`
}

type IndexReport struct {
	Documents int `json:"documents"`
	Records   int `json:"records"`
}

func VerifyIndex(root string) (IndexReport, error) {
	indexPath, documentsRoot := indexLocations(root)
	records, err := loadIndex(indexPath)
	if err != nil {
		return IndexReport{}, err
	}
	documents, err := collectDocumentRecords(documentsRoot)
	if err != nil {
		return IndexReport{}, err
	}
	indexed := make(map[string]indexRecord, len(records))
	for _, record := range records {
		indexed[filepath.Clean(record.Path)] = record
	}
	for _, document := range documents {
		record, exists := indexed[filepath.Clean(document.Path)]
		if !exists {
			return IndexReport{}, fmt.Errorf("stale dedup index: missing document path %s", document.Path)
		}
		switch {
		case record.DocumentID != document.DocumentID:
			return IndexReport{}, fmt.Errorf("stale dedup index: document ID mismatch for %s", document.Path)
		case record.URLHash != document.URLHash:
			return IndexReport{}, fmt.Errorf("stale dedup index: URL hash mismatch for %s", document.Path)
		case record.ContentHash != document.ContentHash:
			return IndexReport{}, fmt.Errorf("stale dedup index: content hash mismatch for %s", document.Path)
		case record.SimHash != document.SimHash:
			return IndexReport{}, fmt.Errorf("stale dedup index: SimHash mismatch for %s", document.Path)
		}
		delete(indexed, filepath.Clean(document.Path))
	}
	if len(indexed) > 0 {
		paths := make([]string, 0, len(indexed))
		for path := range indexed {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		return IndexReport{}, fmt.Errorf("stale dedup index: extra document path %s", paths[0])
	}
	return IndexReport{Documents: len(documents), Records: len(records)}, nil
}

func RebuildIndex(root string) (IndexReport, error) {
	indexPath, documentsRoot := indexLocations(root)
	records, err := rebuildIndex(indexPath, documentsRoot)
	if err != nil {
		return IndexReport{}, err
	}
	return IndexReport{Documents: len(records), Records: len(records)}, nil
}

func indexLocations(root string) (string, string) {
	if root == "" {
		root = "data"
	}
	return filepath.Join(root, "indexes", "dedup-index.tsv"), filepath.Join(root, "documents")
}

func rebuildIndex(indexPath, documentsRoot string) ([]indexRecord, error) {
	records, err := collectDocumentRecords(documentsRoot)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		return nil, fmt.Errorf("create dedup index directory: %w", err)
	}
	if err := writeIndex(indexPath, records); err != nil {
		return nil, fmt.Errorf("rebuild dedup index: %w", err)
	}
	return records, nil
}

func collectDocumentRecords(documentsRoot string) ([]indexRecord, error) {
	paths := make([]string, 0)
	err := filepath.WalkDir(documentsRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(path), ".md") {
			paths = append(paths, path)
		}
		return nil
	})
	if os.IsNotExist(err) {
		paths = nil
	} else if err != nil {
		return nil, fmt.Errorf("scan accepted Markdown: %w", err)
	}
	sort.Strings(paths)
	records := make([]indexRecord, 0, len(paths))
	documentIDs := make(map[string]string, len(paths))
	for _, path := range paths {
		record, err := fingerprintDocument(path)
		if err != nil {
			return nil, err
		}
		if previousPath, exists := documentIDs[record.DocumentID]; exists {
			return nil, fmt.Errorf("invalid accepted Markdown %s: duplicate document ID also used by %s", path, previousPath)
		}
		documentIDs[record.DocumentID] = path
		records = append(records, record)
	}
	return records, nil
}

func fingerprintDocument(path string) (indexRecord, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return indexRecord{}, fmt.Errorf("read accepted Markdown %s: %w", path, err)
	}
	text := string(payload)
	if !strings.HasPrefix(text, "---\n") {
		return indexRecord{}, fmt.Errorf("invalid accepted Markdown %s: missing front matter", path)
	}
	end := strings.Index(text[4:], "\n---\n")
	if end < 0 {
		return indexRecord{}, fmt.Errorf("invalid accepted Markdown %s: unterminated front matter", path)
	}
	end += 4
	var metadata documentMetadata
	if err := yaml.Unmarshal([]byte(text[4:end]), &metadata); err != nil {
		return indexRecord{}, fmt.Errorf("invalid accepted Markdown %s: decode front matter", path)
	}
	if metadata.QualityStatus != "accepted" {
		return indexRecord{}, fmt.Errorf("invalid accepted Markdown %s: quality status", path)
	}
	canonical := materialization.CanonicalURL(metadata.SourceURL)
	body := materialization.NormalizeBody(text[end+5:])
	if canonical == "" || body == "" {
		return indexRecord{}, fmt.Errorf("invalid accepted Markdown %s: source URL or body", path)
	}
	contentHash := materialization.Hash(body)
	documentID := "sha256:" + materialization.Hash(canonical+"\n"+contentHash)
	if metadata.ContentSHA256 != contentHash || metadata.DocumentID != documentID {
		return indexRecord{}, fmt.Errorf("invalid accepted Markdown %s: identity mismatch", path)
	}
	return indexRecord{
		DocumentID:  documentID,
		PublishedAt: metadata.PublishedAt,
		URLHash:     materialization.Hash(canonical),
		ContentHash: contentHash,
		SimHash:     materialization.SimHash(body),
		Path:        path,
	}, nil
}
