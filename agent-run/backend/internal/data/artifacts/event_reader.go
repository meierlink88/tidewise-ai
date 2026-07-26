package artifacts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector/materialization"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact"
	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
	"gopkg.in/yaml.v3"
)

type ExecutionReader interface {
	GetExecution(context.Context, string) (agentrun.Execution, error)
}

type EventReader struct {
	Root       string
	Executions ExecutionReader
}

type eventManifest = manifestPayload

type eventDocumentMetadata struct {
	SchemaVersion      string   `yaml:"schema_version"`
	DocumentID         string   `yaml:"document_id"`
	Title              string   `yaml:"title"`
	SourceName         string   `yaml:"source_name"`
	SourceType         string   `yaml:"source_type"`
	SourceURL          string   `yaml:"source_url"`
	SourceExternalID   string   `yaml:"source_external_id"`
	Connectors         []string `yaml:"connectors"`
	PrimaryConnector   string   `yaml:"primary_connector"`
	ContentOrigin      string   `yaml:"content_origin"`
	ContentLevel       string   `yaml:"content_level"`
	PublishedAt        *string  `yaml:"published_at"`
	CollectedAt        string   `yaml:"collected_at"`
	TimeBasis          string   `yaml:"time_basis"`
	Language           string   `yaml:"language"`
	ContentSHA256      string   `yaml:"content_sha256"`
	QualityStatus      string   `yaml:"quality_status"`
	SupersedesDocument *string  `yaml:"supersedes_document_id"`
}

func (r EventReader) Read(ctx context.Context, collectorExecutionIDs []string) ([]eventfact.Artifact, error) {
	if r.Executions == nil {
		return nil, errors.New("ArtifactReader Execution repository is required")
	}
	root, err := filepath.Abs(r.Root)
	if err != nil || strings.TrimSpace(r.Root) == "" {
		return nil, errors.New("ArtifactReader root is invalid")
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, errors.New("ArtifactReader root is unavailable")
	}
	var result []eventfact.Artifact
	seenArtifacts := make(map[string]string)
	for _, executionID := range collectorExecutionIDs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		execution, err := r.Executions.GetExecution(ctx, executionID)
		if err != nil {
			return nil, fmt.Errorf("read Collector Execution: %w", err)
		}
		if execution.AgentKey != "collector" ||
			(execution.Status != agentrun.StatusSucceeded && execution.Status != agentrun.StatusPartiallySucceeded) {
			return nil, errors.New("Collector Execution is not eligible for Event extraction")
		}
		expectedManifest := filepath.Join(root, "runs", executionID, "manifest.json")
		manifestPath, err := secureArtifactPath(root, execution.Artifacts["manifest"])
		if err != nil || manifestPath != expectedManifest {
			return nil, errors.New("Collector Manifest identity is invalid")
		}
		manifest, err := readEventManifest(manifestPath)
		if err != nil {
			return nil, err
		}
		if manifest.Schema != "collector_artifact_manifest.v1" ||
			manifest.ExecutionID != executionID ||
			manifest.AgentKey != "collector" ||
			manifest.AgentVersion != execution.AgentVersion ||
			manifest.ResultsPending != 0 ||
			manifest.ExecutionStatus != string(execution.Status) {
			return nil, errors.New("Collector Manifest contract is invalid")
		}
		for _, accepted := range manifest.Accepted {
			path, err := secureArtifactPath(root, accepted["path"])
			if err != nil || !validArtifactSHA256(accepted["sha256"]) {
				return nil, errors.New("accepted Artifact identity is invalid")
			}
			payload, err := os.ReadFile(path)
			if err != nil {
				return nil, errors.New("accepted Artifact is unavailable")
			}
			sum := sha256.Sum256(payload)
			if hex.EncodeToString(sum[:]) != accepted["sha256"] {
				return nil, errors.New("accepted Artifact hash mismatch")
			}
			artifact, err := parseEventArtifact(payload, executionID)
			if err != nil {
				return nil, err
			}
			if existing, exists := seenArtifacts[artifact.ArtifactID]; exists && existing != artifact.ContentSHA256 {
				return nil, errors.New("accepted Artifact identity has conflicting content")
			}
			if _, exists := seenArtifacts[artifact.ArtifactID]; !exists {
				result = append(result, artifact)
				seenArtifacts[artifact.ArtifactID] = artifact.ContentSHA256
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CollectorExecutionID != result[j].CollectorExecutionID {
			return result[i].CollectorExecutionID < result[j].CollectorExecutionID
		}
		return result[i].ArtifactID < result[j].ArtifactID
	})
	return result, nil
}

func readEventManifest(path string) (eventManifest, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return eventManifest{}, errors.New("Collector Manifest is unavailable")
	}
	var manifest eventManifest
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return eventManifest{}, errors.New("Collector Manifest contract is invalid")
	}
	if decoder.Decode(&struct{}{}) == nil {
		return eventManifest{}, errors.New("Collector Manifest has trailing content")
	}
	return manifest, nil
}

func parseEventArtifact(payload []byte, collectorExecutionID string) (eventfact.Artifact, error) {
	if !bytes.HasPrefix(payload, []byte("---\n")) {
		return eventfact.Artifact{}, errors.New("accepted Artifact front matter is missing")
	}
	parts := bytes.SplitN(payload[4:], []byte("\n---\n"), 2)
	if len(parts) != 2 {
		return eventfact.Artifact{}, errors.New("accepted Artifact front matter is invalid")
	}
	var metadata eventDocumentMetadata
	decoder := yaml.NewDecoder(bytes.NewReader(parts[0]))
	decoder.KnownFields(true)
	if err := decoder.Decode(&metadata); err != nil {
		return eventfact.Artifact{}, errors.New("accepted Artifact front matter is invalid")
	}
	body := strings.TrimSpace(string(parts[1]))
	if metadata.SchemaVersion != "connector_result_md.v1" ||
		metadata.QualityStatus != "accepted" ||
		metadata.DocumentID == "" || metadata.Title == "" ||
		metadata.SourceURL == "" ||
		!validArtifactSHA256(metadata.ContentSHA256) {
		return eventfact.Artifact{}, errors.New("accepted Artifact contract is invalid")
	}
	contentHash := materialization.Hash(materialization.NormalizeBody(body))
	if contentHash != metadata.ContentSHA256 {
		return eventfact.Artifact{}, errors.New("accepted Artifact content hash mismatch")
	}
	if "sha256:"+materialization.Hash(metadata.SourceURL+"\n"+contentHash) != metadata.DocumentID {
		return eventfact.Artifact{}, errors.New("accepted Artifact document identity mismatch")
	}
	collectedAt, err := time.Parse(time.RFC3339, metadata.CollectedAt)
	if err != nil {
		return eventfact.Artifact{}, errors.New("accepted Artifact collected_at is invalid")
	}
	var publishedAt *time.Time
	if metadata.PublishedAt != nil && *metadata.PublishedAt != "" {
		parsed, err := time.Parse(time.RFC3339, *metadata.PublishedAt)
		if err != nil {
			return eventfact.Artifact{}, errors.New("accepted Artifact published_at is invalid")
		}
		parsed = parsed.UTC()
		publishedAt = &parsed
	}
	sourceName := metadata.SourceName
	if sourceName == "" {
		sourceName = metadata.PrimaryConnector
	}
	sourceType := metadata.SourceType
	if sourceType == "" {
		sourceType = "unknown"
	}
	return eventfact.Artifact{
		ArtifactID:           metadata.DocumentID,
		CollectorExecutionID: collectorExecutionID,
		DocumentID:           metadata.DocumentID,
		Title:                metadata.Title,
		SourceName:           sourceName,
		SourceType:           sourceType,
		SourceURL:            metadata.SourceURL,
		ContentLevel:         metadata.ContentLevel,
		PublishedAt:          publishedAt,
		CollectedAt:          collectedAt.UTC(),
		Language:             metadata.Language,
		ContentSHA256:        metadata.ContentSHA256,
		Body:                 body,
	}, nil
}

func secureArtifactPath(root, candidate string) (string, error) {
	if strings.TrimSpace(candidate) == "" {
		return "", errors.New("Artifact path is required")
	}
	path := candidate
	if !filepath.IsAbs(path) {
		fromWorkingDirectory, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		if pathWithinRoot(root, fromWorkingDirectory) {
			path = fromWorkingDirectory
		} else {
			path = filepath.Join(root, path)
		}
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	absolute, err = filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", errors.New("Artifact path is unavailable")
	}
	if !pathWithinRoot(root, absolute) {
		return "", errors.New("Artifact path escapes root")
	}
	return absolute, nil
}

func pathWithinRoot(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." &&
		!strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func validArtifactSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && value == strings.ToLower(value)
}
