package eventpublication

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestDecodeStrictRejectsUnknownFields(t *testing.T) {
	_, err := DecodeStrict(strings.NewReader(`{"package_id":"pkg","unexpected":true}`))
	if err == nil || !strings.Contains(err.Error(), `unknown field "unexpected"`) {
		t.Fatalf("DecodeStrict error = %v, want unknown field", err)
	}
}

func TestPublicationValidateAcceptsFrozenV2Contract(t *testing.T) {
	if err := validPublication().Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestPublicationValidateRejectsStorageBoundaryOverflow(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Publication)
		path string
	}{
		{
			name: "package id",
			edit: func(publication *Publication) {
				publication.PackageID = strings.Repeat("p", 257)
			},
			path: "package_id",
		},
		{
			name: "source type",
			edit: func(publication *Publication) {
				publication.RawDocuments[0].SourceType = strings.Repeat("s", 65)
			},
			path: "raw_documents[0].source_type",
		},
		{
			name: "language",
			edit: func(publication *Publication) {
				publication.RawDocuments[0].Language = strings.Repeat("l", 17)
			},
			path: "raw_documents[0].language",
		},
		{
			name: "mime type",
			edit: func(publication *Publication) {
				publication.RawDocuments[0].MIMEType = strings.Repeat("m", 129)
			},
			path: "raw_documents[0].mime_type",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			publication := validPublication()
			test.edit(&publication)
			err := publication.Validate()
			var validation *ValidationError
			if !asValidationError(err, &validation) {
				t.Fatalf("Validate() error = %v, want ValidationError", err)
			}
			found := false
			for _, issue := range validation.Issues {
				if issue.Path == test.path && issue.Code == "MAX_LENGTH" {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("issues = %#v, want MAX_LENGTH at %s", validation.Issues, test.path)
			}
		})
	}
}

func TestNewValidationErrorSortsIssuesDeterministically(t *testing.T) {
	err := NewValidationError([]ValidationIssue{
		{Path: "events[1].title", Code: "REQUIRED", Message: "second"},
		{Path: "events[0].title", Code: "REQUIRED", Message: "first"},
		{Path: "events[0].title", Code: "MAX_LENGTH", Message: "bounded"},
	})
	got := err.Issues
	if got[0].Path != "events[0].title" || got[0].Code != "MAX_LENGTH" ||
		got[1].Path != "events[0].title" || got[1].Code != "REQUIRED" ||
		got[2].Path != "events[1].title" {
		t.Fatalf("issues not sorted deterministically: %#v", got)
	}
}

func TestSemanticJSONEqualPreservesNumberPrecision(t *testing.T) {
	if SemanticJSONEqual(
		map[string]any{"count": json.Number("9007199254740992")},
		map[string]any{"count": json.Number("9007199254740993")},
	) {
		t.Fatal("SemanticJSONEqual treated distinct integers above float64 precision as equal")
	}
	if !SemanticJSONEqual(
		map[string]any{"ratio": json.Number("1")},
		map[string]any{"ratio": json.Number("1.0")},
	) {
		t.Fatal("SemanticJSONEqual treated equivalent JSON numbers as different")
	}
}

func asValidationError(err error, target **ValidationError) bool {
	validation, ok := err.(*ValidationError)
	if ok {
		*target = validation
	}
	return ok
}

func validPublication() Publication {
	publishedAt := time.Date(2026, 7, 23, 1, 0, 0, 0, time.UTC)
	collectedAt := time.Date(2026, 7, 23, 1, 5, 0, 0, time.UTC)
	occurredAt := time.Date(2026, 7, 23, 0, 30, 0, 0, time.UTC)
	return Publication{
		PackageID: "package-1",
		Provenance: Provenance{
			ExtractorExecutionID:  "extractor-1",
			ExtractorAgentVersion: "extractor-v2",
			CollectorExecutions: []CollectorExecution{{
				ArtifactID: "artifact-1", CollectorExecutionID: "collector-1",
			}},
		},
		RawDocuments: []RawDocument{{
			ArtifactID: "artifact-1", ContentSHA256: strings.Repeat("a", 64),
			SourceRef: "source:1", SourceName: "Source", SourceType: "news",
			SourceURL: "https://example.test/1", Title: "Source title",
			PublishedAt: &publishedAt, CollectedAt: collectedAt,
			Language: "en", MIMEType: "text/markdown",
		}},
		Events: []Event{{
			DedupeKey: "event-1", Title: "Event title", FactualSummary: "Event summary",
			OccurredAt: &occurredAt, FactPayload: map[string]any{"metric": "example"},
			Evidence: []Evidence{{
				ArtifactID: "artifact-1", EvidenceRelation: "supports",
				EvidenceExcerpt: "Evidence excerpt", SupportsFields: []string{"title"},
				SourceLevel: "primary", IsPrimary: true,
			}},
			Tags: []Tag{{
				TagID:   "22a5afc5-20ed-55ce-bf77-54c26bbcc6ea",
				TagKind: "news_category", TagCode: "technology_industry",
				Confidence: json.Number("0.9"), AssignmentReason: "Technology event",
				AssignSource: "ai",
			}},
			Review: Review{ReviewID: "review-1", EvidenceGrade: "A", Reasons: []string{"Reviewed"}},
		}},
	}
}
