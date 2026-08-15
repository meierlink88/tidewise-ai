package event

import (
	"context"
	"encoding/json"
	"reflect"
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

func TestPublicationValidateAcceptsFrozenContract(t *testing.T) {
	if err := validPublication().Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestPublicationValidateRejectsStorageBoundaryOverflow(t *testing.T) {
	tests := []struct {
		name string
		edit func(*PublicationBatch)
		path string
	}{
		{
			name: "package id",
			edit: func(publication *PublicationBatch) {
				publication.PackageID = strings.Repeat("p", 257)
			},
			path: "package_id",
		},
		{
			name: "source type",
			edit: func(publication *PublicationBatch) {
				publication.RawDocuments[0].SourceType = strings.Repeat("s", 65)
			},
			path: "raw_documents[0].source_type",
		},
		{
			name: "language",
			edit: func(publication *PublicationBatch) {
				publication.RawDocuments[0].Language = strings.Repeat("l", 17)
			},
			path: "raw_documents[0].language",
		},
		{
			name: "mime type",
			edit: func(publication *PublicationBatch) {
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

func TestPublicationValidatePreservesFactEvidenceAndTagBoundaries(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*PublicationBatch)
	}{
		{name: "forbidden fact prediction", edit: func(publication *PublicationBatch) {
			publication.Events[0].FactPayload = map[string]any{"price_prediction": "上涨"}
		}},
		{name: "supports without fields", edit: func(publication *PublicationBatch) {
			publication.Events[0].Evidence[0].SupportsFields = nil
		}},
		{name: "unsupported evidence relation", edit: func(publication *PublicationBatch) {
			publication.Events[0].Evidence[0].EvidenceRelation = "irrelevant"
		}},
		{name: "assignment without reason", edit: func(publication *PublicationBatch) {
			publication.Events[0].Tags[0].AssignmentReason = ""
		}},
		{name: "confidence over one", edit: func(publication *PublicationBatch) {
			publication.Events[0].Tags[0].Confidence = json.Number("1.0001")
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			publication := validPublication()
			test.edit(&publication)
			if err := publication.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want rejection")
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

func TestActiveTagsSortsStableCurrentCollection(t *testing.T) {
	tags := []EventTag{
		{ID: "ETDb1a5438f-6e81-55e7-8ecb-33230b9ae965", Kind: "news_category", Code: "macroeconomy", Name: "宏观经济", Active: true},
		{ID: "ETD22a5afc5-20ed-55ce-bf77-54c26bbcc6ea", Kind: "news_category", Code: "technology_industry", Name: "科技产业", Active: true},
	}
	first, err := NewUseCase(fakeStore{tags: tags})
	if err != nil {
		t.Fatal(err)
	}
	firstCatalog, err := first.ActiveTags(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewUseCase(fakeStore{tags: []EventTag{tags[1], tags[0]}})
	if err != nil {
		t.Fatal(err)
	}
	secondCatalog, err := second.ActiveTags(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(firstCatalog.Tags, secondCatalog.Tags) {
		t.Fatalf("unstable collections: first=%#v second=%#v", firstCatalog, secondCatalog)
	}
	if firstCatalog.Tags[0].Code != "macroeconomy" || firstCatalog.Tags[1].Code != "technology_industry" {
		t.Fatalf("catalog order = %#v", firstCatalog.Tags)
	}
}

func TestActiveTagsRejectsInvalidCatalogRows(t *testing.T) {
	useCase, err := NewUseCase(fakeStore{tags: []EventTag{{
		ID: "22a5afc5-20ed-55ce-bf77-54c26bbcc6ea", Kind: "news_category",
		Code: "technology_industry", Name: "科技产业", Active: false,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := useCase.ActiveTags(context.Background()); err == nil {
		t.Fatal("inactive persisted Tag was accepted")
	}
}

func TestListEventsMapsReadProjection(t *testing.T) {
	now := time.Date(2026, 8, 11, 1, 0, 0, 0, time.UTC)
	useCase, err := NewUseCase(fakeStore{events: EventStorePage{
		Items: []EventListItem{{
			ID: "event-1", Title: "Event", FirstSeenAt: now, EventStatus: EventStatusConfirmed,
			FactStatus: FactStatusVerified, DedupeKey: "event:key",
		}}, Total: 1, Page: 2, PageSize: 10,
	}})
	if err != nil {
		t.Fatal(err)
	}
	page, err := useCase.ListEvents(context.Background(), EventListRequest{Page: 2, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || page.Items[0].ID != "event-1" || page.Page != 2 || page.PageSize != 10 {
		t.Fatalf("page = %#v", page)
	}
}

type fakeStore struct {
	tags           []EventTag
	events         EventStorePage
	researchEvents ResearchEventPage
}

func (fakeStore) InTransaction(context.Context, func(Transaction) error) error { return nil }
func (store fakeStore) ListActiveTags(context.Context) ([]EventTag, error) {
	return append([]EventTag(nil), store.tags...), nil
}
func (store fakeStore) ListEvents(context.Context, EventListFilter) (EventStorePage, error) {
	return store.events, nil
}

func (store fakeStore) ListResearchEvents(context.Context, ResearchEventQuery) (ResearchEventPage, error) {
	return store.researchEvents, nil
}

func TestResearchEventProviderReturnsFormalFacts(t *testing.T) {
	want := ResearchEventPage{Events: []ResearchEventRecord{{Event: ResearchEventFact{ID: "10000000-0000-4000-8000-000000000001"}}}}
	useCase, err := NewUseCase(fakeStore{researchEvents: want})
	if err != nil {
		t.Fatal(err)
	}
	got, err := useCase.ListResearchEvents(context.Background(), ResearchEventQuery{PageSize: 1})
	if err != nil || len(got.Events) != 1 || got.Events[0].Event.ID != want.Events[0].Event.ID {
		t.Fatalf("ListResearchEvents() = %#v, %v", got, err)
	}
}

func asValidationError(err error, target **ValidationError) bool {
	validation, ok := err.(*ValidationError)
	if ok {
		*target = validation
	}
	return ok
}

func validPublication() PublicationBatch {
	publishedAt := time.Date(2026, 7, 23, 1, 0, 0, 0, time.UTC)
	collectedAt := time.Date(2026, 7, 23, 1, 5, 0, 0, time.UTC)
	occurredAt := time.Date(2026, 7, 23, 0, 30, 0, 0, time.UTC)
	return PublicationBatch{
		PackageID: "package-1",
		Provenance: Provenance{
			ExtractorExecutionID:  "extractor-1",
			ExtractorAgentVersion: "extractor-v2",
			CollectorExecutions: []CollectorExecution{{
				ArtifactID: "artifact-1", CollectorExecutionID: "collector-1",
			}},
		},
		RawDocuments: []EventEvidenceRecord{{
			ArtifactID: "artifact-1", ContentSHA256: strings.Repeat("a", 64),
			SourceRef: "source:1", SourceName: "Source", SourceType: "news",
			SourceURL: "https://example.test/1", Title: "Source title",
			PublishedAt: &publishedAt, CollectedAt: collectedAt,
			Language: "en", MIMEType: "text/markdown",
		}},
		Events: []PublicationEvent{{
			DedupeKey: "event-1", Title: "Event title", FactualSummary: "Event summary",
			OccurredAt: &occurredAt, FactPayload: map[string]any{"metric": "example"},
			Evidence: []EventEvidenceLinkInput{{
				ArtifactID: "artifact-1", EvidenceRelation: "supports",
				EvidenceStatement: "Evidence statement", SupportsFields: []string{"title"},
				SourceLevel: "primary",
			}},
			Tags: []EventTagInput{{
				TagID:   "ETD22a5afc5-20ed-55ce-bf77-54c26bbcc6ea",
				TagKind: "news_category", TagCode: "technology_industry",
				Confidence: json.Number("0.9"), AssignmentReason: "Technology event",
				AssignSource: "ai",
			}},
			Review: Review{ReviewID: "review-1", EvidenceGrade: "A", Reasons: []string{"Reviewed"}},
		}},
	}
}
