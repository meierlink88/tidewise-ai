package event

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	eventbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/event"
)

func Publication(suffix string) eventbiz.PublicationBatch {
	publishedAt := time.Date(2026, 7, 23, 1, 0, 0, 0, time.UTC)
	collectedAt := time.Date(2026, 7, 23, 1, 5, 0, 0, time.UTC)
	occurredAt := time.Date(2026, 7, 23, 0, 30, 0, 0, time.UTC)
	artifactID := "artifact-" + suffix
	return eventbiz.PublicationBatch{
		PackageID: "package-" + suffix,
		Provenance: eventbiz.Provenance{
			ExtractorExecutionID:  "extractor-" + suffix,
			ExtractorAgentVersion: "event-extractor-v2",
			CollectorExecutions: []eventbiz.CollectorExecution{{
				ArtifactID: artifactID, CollectorExecutionID: "collector-" + suffix,
			}},
		},
		RawDocuments: []eventbiz.EventEvidenceRecord{{
			ArtifactID: artifactID, ContentSHA256: fmt.Sprintf("%064x", len(suffix)+10),
			SourceRef: "source:" + suffix, SourceName: "Source " + suffix, SourceType: "news",
			SourceURL: "https://example.test/" + url.PathEscape(suffix), Title: "Source " + suffix,
			PublishedAt: &publishedAt, CollectedAt: collectedAt, Language: "en", MIMEType: "text/markdown",
		}},
		Events: []eventbiz.PublicationEvent{{
			DedupeKey: "event:" + suffix + ":1", Title: "Event " + suffix,
			FactualSummary: "A verifiable state change occurred for " + suffix + ".",
			OccurredAt:     &occurredAt,
			FactPayload:    map[string]any{"fixture": suffix},
			Evidence: []eventbiz.EventEvidenceLinkInput{{
				ArtifactID: artifactID, EvidenceRelation: "supports",
				EvidenceStatement: "Evidence for " + suffix,
				SupportsFields:    []string{"title", "factual_summary"},
				SourceLevel:       "primary",
			}},
			Tags: []eventbiz.EventTagInput{{
				TagID:   "22a5afc5-20ed-55ce-bf77-54c26bbcc6ea",
				TagKind: "news_category", TagCode: "technology_industry",
				Confidence: json.Number("0.94"), AssignmentReason: "Technology event",
				AssignSource: "ai",
			}},
			Review: eventbiz.Review{
				ReviewID: "review-" + suffix, EvidenceGrade: "A", Reasons: []string{"Reviewed"},
			},
		}},
	}
}

func ClonePublicationEvent(input eventbiz.PublicationEvent, suffix string) eventbiz.PublicationEvent {
	cloned := input
	cloned.DedupeKey = "event:" + suffix + ":1"
	cloned.Title = "Event " + suffix
	cloned.FactualSummary = "A verifiable state change occurred for " + suffix + "."
	cloned.FactPayload = map[string]any{"fixture": suffix}
	cloned.Evidence = append([]eventbiz.EventEvidenceLinkInput(nil), input.Evidence...)
	cloned.Tags = append([]eventbiz.EventTagInput(nil), input.Tags...)
	cloned.Review = eventbiz.Review{
		ReviewID: "review-" + suffix, EvidenceGrade: "A", Reasons: []string{"Reviewed"},
	}
	return cloned
}
