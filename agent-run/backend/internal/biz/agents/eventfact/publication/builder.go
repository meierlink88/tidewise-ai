package publication

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact"
)

const maxEventsPerBatch = 10

type request struct {
	PackageID    string        `json:"package_id"`
	Provenance   provenance    `json:"provenance"`
	RawDocuments []rawDocument `json:"raw_documents"`
	Events       []event       `json:"events"`
}

type provenance struct {
	ExtractorExecutionID  string               `json:"extractor_execution_id"`
	ExtractorAgentVersion string               `json:"extractor_agent_version"`
	CollectorExecutions   []collectorExecution `json:"collector_executions"`
}

type collectorExecution struct {
	ArtifactID           string `json:"artifact_id"`
	CollectorExecutionID string `json:"collector_execution_id"`
}

type rawDocument struct {
	ArtifactID    string     `json:"artifact_id"`
	ContentSHA256 string     `json:"content_sha256"`
	SourceRef     string     `json:"source_ref"`
	SourceName    string     `json:"source_name"`
	SourceType    string     `json:"source_type"`
	SourceURL     string     `json:"source_url,omitempty"`
	Title         string     `json:"title"`
	PublishedAt   *time.Time `json:"published_at"`
	CollectedAt   time.Time  `json:"collected_at"`
	Language      string     `json:"language,omitempty"`
	MIMEType      string     `json:"mime_type,omitempty"`
}

type event struct {
	DedupeKey      string         `json:"dedupe_key"`
	Title          string         `json:"title"`
	FactualSummary string         `json:"factual_summary"`
	OccurredAt     *time.Time     `json:"occurred_at"`
	FactPayload    map[string]any `json:"fact_payload"`
	Evidence       []evidence     `json:"evidence"`
	Tags           []tag          `json:"tags"`
	Review         review         `json:"review"`
}

type evidence struct {
	ArtifactID        string   `json:"artifact_id"`
	EvidenceRelation  string   `json:"evidence_relation"`
	EvidenceStatement string   `json:"evidence_statement"`
	SupportsFields    []string `json:"supports_fields"`
	SourceLevel       string   `json:"source_level"`
}

type tag struct {
	TagID            string  `json:"tag_id"`
	TagKind          string  `json:"tag_kind"`
	TagCode          string  `json:"tag_code"`
	Confidence       float64 `json:"confidence"`
	AssignmentReason string  `json:"assignment_reason"`
	AssignSource     string  `json:"assign_source"`
}

type review struct {
	ReviewID      string   `json:"review_id"`
	EvidenceGrade string   `json:"evidence_grade"`
	Reasons       []string `json:"reasons"`
}

func Build(
	workItemKey, agentVersion string,
	result eventfact.Result,
) ([]eventfact.JournalEntry, error) {
	approved := make([]eventfact.Candidate, 0, len(result.Candidates))
	for _, candidate := range result.Candidates {
		if candidate.ReviewState == eventfact.ReviewAutoApproved {
			approved = append(approved, candidate)
		}
	}
	if len(approved) == 0 {
		return nil, nil
	}
	sort.Slice(approved, func(i, j int) bool {
		return approved[i].DedupeKey < approved[j].DedupeKey
	})
	for index := 1; index < len(approved); index++ {
		if approved[index-1].DedupeKey == approved[index].DedupeKey {
			return nil, errors.New("Event publication contains duplicate deterministic identities")
		}
	}
	artifacts := make(map[string]eventfact.Artifact, len(result.PublicationArtifacts))
	for _, artifact := range result.PublicationArtifacts {
		artifacts[artifact.ArtifactID] = artifact
	}
	var journals []eventfact.JournalEntry
	for start := 0; start < len(approved); start += maxEventsPerBatch {
		end := start + maxEventsPerBatch
		if end > len(approved) {
			end = len(approved)
		}
		ordinal := len(journals) + 1
		packageID := fmt.Sprintf("agentrun:event-fact:%s:%d", workItemKey, ordinal)
		payload := request{
			PackageID: packageID,
			Provenance: provenance{
				ExtractorExecutionID: result.ExecutionID, ExtractorAgentVersion: agentVersion,
			},
		}
		used := make(map[string]struct{})
		for _, candidate := range approved[start:end] {
			artifact, exists := artifacts[candidate.ArtifactID]
			if !exists {
				return nil, errors.New("Event publication Artifact is missing")
			}
			if _, exists := used[artifact.ArtifactID]; !exists {
				used[artifact.ArtifactID] = struct{}{}
				payload.RawDocuments = append(payload.RawDocuments, rawDocument{
					ArtifactID: artifact.ArtifactID, ContentSHA256: artifact.ContentSHA256,
					SourceRef: artifact.DocumentID, SourceName: artifact.SourceName,
					SourceType: artifact.SourceType, SourceURL: artifact.SourceURL, Title: artifact.Title,
					PublishedAt: artifact.PublishedAt, CollectedAt: artifact.CollectedAt,
					Language: artifact.Language, MIMEType: "text/markdown",
				})
				payload.Provenance.CollectorExecutions = append(
					payload.Provenance.CollectorExecutions,
					collectorExecution{
						ArtifactID:           artifact.ArtifactID,
						CollectorExecutionID: artifact.CollectorExecutionID,
					},
				)
			}
			item := event{
				DedupeKey: candidate.DedupeKey, Title: candidate.Title,
				FactualSummary: candidate.FactualSummary, OccurredAt: candidate.OccurredAt,
				FactPayload: candidate.FactPayload,
				Evidence: []evidence{{
					ArtifactID: candidate.ArtifactID, EvidenceRelation: "supports",
					EvidenceStatement: candidate.EvidenceStatement, SupportsFields: candidate.SupportsFields,
					SourceLevel: candidate.SourceLevel,
				}},
				Review: review{
					ReviewID:      "event-fact-review:" + candidate.IdentityHash,
					EvidenceGrade: evidenceGrade(artifact),
					Reasons:       append([]string(nil), candidate.Review.Reasons...),
				},
			}
			for _, assignment := range candidate.Tags {
				item.Tags = append(item.Tags, tag{
					TagID: assignment.ID, TagKind: assignment.Kind, TagCode: assignment.Code,
					Confidence: assignment.Confidence, AssignmentReason: assignment.AssignmentReason,
					AssignSource: "ai",
				})
			}
			payload.Events = append(payload.Events, item)
		}
		sort.Slice(payload.RawDocuments, func(i, j int) bool {
			return payload.RawDocuments[i].ArtifactID < payload.RawDocuments[j].ArtifactID
		})
		sort.Slice(payload.Provenance.CollectorExecutions, func(i, j int) bool {
			return payload.Provenance.CollectorExecutions[i].ArtifactID <
				payload.Provenance.CollectorExecutions[j].ArtifactID
		})
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("encode Event publication payload: %w", err)
		}
		sum := sha256.Sum256(encoded)
		journals = append(journals, eventfact.JournalEntry{
			WorkItemKey: workItemKey, BatchOrdinal: ordinal, PackageID: packageID,
			Payload: encoded, PayloadHash: hex.EncodeToString(sum[:]), Status: "prepared",
		})
	}
	return journals, nil
}

func BuildArtifactUnit(
	workItemKey, unitKey string,
	artifactOrdinal int,
	agentVersion string,
	result eventfact.Result,
) ([]eventfact.JournalEntry, error) {
	if len(workItemKey) != 64 || len(unitKey) != 64 || artifactOrdinal < 1 {
		return nil, errors.New("Event Artifact Unit publication identity is invalid")
	}
	journals, err := Build(unitKey, agentVersion, result)
	if err != nil {
		return nil, err
	}
	for index := range journals {
		journals[index].WorkItemKey = workItemKey
		journals[index].UnitKey = unitKey
	}
	return journals, nil
}

func CanonicalEvents(payload []byte) ([]eventfact.CanonicalEvent, error) {
	var decoded request
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, errors.New("stored Event publication payload is invalid")
	}
	result := make([]eventfact.CanonicalEvent, 0, len(decoded.Events))
	for _, item := range decoded.Events {
		core, err := json.Marshal(struct {
			Title          string         `json:"title"`
			FactualSummary string         `json:"factual_summary"`
			OccurredAt     *time.Time     `json:"occurred_at"`
			FactPayload    map[string]any `json:"fact_payload"`
		}{item.Title, item.FactualSummary, item.OccurredAt, item.FactPayload})
		if err != nil {
			return nil, err
		}
		identity := stringsTrimPrefix(item.DedupeKey, "event-fact:")
		if len(identity) != 64 {
			sum := sha256.Sum256(core)
			identity = hex.EncodeToString(sum[:])
		}
		result = append(result, eventfact.CanonicalEvent{
			DedupeKey: item.DedupeKey, IdentityHash: identity, CoreFacts: core,
		})
	}
	return result, nil
}

func evidenceGrade(artifact eventfact.Artifact) string {
	if artifact.ContentLevel == "full_text" &&
		(artifact.SourceType == "official" || artifact.SourceType == "government" ||
			artifact.SourceType == "filing") {
		return "A"
	}
	if artifact.ContentLevel == "full_text" {
		return "B"
	}
	return "C"
}

func stringsTrimPrefix(value, prefix string) string {
	if len(value) >= len(prefix) && value[:len(prefix)] == prefix {
		return value[len(prefix):]
	}
	return value
}
