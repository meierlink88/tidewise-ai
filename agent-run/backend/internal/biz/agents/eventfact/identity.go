package eventfact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/google/uuid"
)

type workIdentity struct {
	Schema                string   `json:"schema"`
	CollectorExecutionIDs []string `json:"collector_execution_ids"`
	ExtractorAgentVersion string   `json:"extractor_agent_version"`
}

type unitIdentity struct {
	Schema        string `json:"schema"`
	WorkItemKey   string `json:"work_item_key"`
	ArtifactID    string `json:"artifact_id"`
	ContentSHA256 string `json:"content_sha256"`
}

func WorkItemIdentity(collectorExecutionIDs []string, agentVersion string) (string, []string, error) {
	if strings.TrimSpace(agentVersion) == "" {
		return "", nil, errors.New("Extractor Agent Version is required")
	}
	unique := make(map[string]struct{}, len(collectorExecutionIDs))
	for _, id := range collectorExecutionIDs {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return "", nil, errors.New("Collector Execution identity is invalid")
		}
		unique[parsed.String()] = struct{}{}
	}
	if len(unique) == 0 {
		return "", nil, errors.New("at least one Collector Execution identity is required")
	}
	ids := make([]string, 0, len(unique))
	for id := range unique {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	encoded, err := json.Marshal(workIdentity{
		Schema:                "event_fact_work.v1",
		CollectorExecutionIDs: ids,
		ExtractorAgentVersion: agentVersion,
	})
	if err != nil {
		return "", nil, err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), ids, nil
}

func ArtifactUnitIdentity(parentIdentity, artifactID, contentSHA256 string) (string, error) {
	if len(parentIdentity) != 64 || strings.Trim(parentIdentity, "0123456789abcdef") != "" {
		return "", errors.New("Event extraction Work Item identity is invalid")
	}
	if strings.TrimSpace(artifactID) == "" {
		return "", errors.New("Artifact identity is required")
	}
	if len(contentSHA256) != 64 || strings.Trim(contentSHA256, "0123456789abcdef") != "" {
		return "", errors.New("Artifact content identity is invalid")
	}
	encoded, err := json.Marshal(unitIdentity{
		Schema:        "event_artifact_extraction_unit.v1",
		WorkItemKey:   parentIdentity,
		ArtifactID:    strings.TrimSpace(artifactID),
		ContentSHA256: contentSHA256,
	})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}
