package biz

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestListSourcesFiltersAndPaginatesWhilePreservingDataOrder(t *testing.T) {
	updated := time.Date(2026, 8, 19, 2, 0, 0, 0, time.UTC)
	repo := &FakeDataServiceRepo{ListSourcesFunc: func(context.Context) ([]Source, error) {
		return []Source{
			{ID: "one", Code: "official-a", Name: "Official A", OwnershipType: "fixed", ChannelType: "api", Enabled: true, Priority: 1, DefaultSourceLevel: "L1_OFFICIAL", UpdatedAt: updated},
			{ID: "two", Code: "official-b", Name: "Official B", OwnershipType: "fixed", ChannelType: "api", Enabled: true, Priority: 1, DefaultSourceLevel: "L1_OFFICIAL", UpdatedAt: updated},
		}, nil
	}}
	enabled, priority := true, 1
	page, err := NewService(repo).ListSources(context.Background(), SourceListQuery{Text: "OFFICIAL", OwnershipType: "fixed", ChannelType: "api", Enabled: &enabled, Priority: &priority, DefaultSourceLevel: "L1_OFFICIAL", UpdatedFrom: &updated, UpdatedTo: &updated, Page: 2, PageSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || page.Page != 2 || len(page.Items) != 1 || page.Items[0].ID != "two" {
		t.Fatalf("page = %#v", page)
	}
}

func TestCollectionDocumentBuildsOnlyCanonicalMinIODocumentURLs(t *testing.T) {
	repo := &FakeDataServiceRepo{GetRawEvidenceDocumentFunc: func(_ context.Context, id string) (RawEvidenceDocument, error) {
		if id != "RAW00000000-0000-5000-8000-000000000001" {
			t.Fatalf("Raw Evidence ID = %q", id)
		}
		return RawEvidenceDocument{RawText: "/raw-evidence/documents/2026/08/17/11f0864fc4078b47a4cc758149a2b0b7923654d2c7c8a694ad5b2d5ced4fc998.md"}, nil
	}}
	service := NewService(repo, WithRawEvidencePublicBaseURL("https://tideai.tripwise.cn"))

	document, err := service.GetCollectionDocument(context.Background(), "RAW00000000-0000-5000-8000-000000000001")
	if err != nil {
		t.Fatal(err)
	}
	if !document.Available || document.URL != "https://tideai.tripwise.cn/raw-evidence/documents/2026/08/17/11f0864fc4078b47a4cc758149a2b0b7923654d2c7c8a694ad5b2d5ced4fc998.md" {
		t.Fatalf("document = %#v", document)
	}
}

func TestCollectionDocumentHidesLegacyTextAndPreservesDependencyErrors(t *testing.T) {
	for _, rawText := range []string{"历史原始文章正文", "/raw-evidence/documents/2026/13/17/11f0864fc4078b47a4cc758149a2b0b7923654d2c7c8a694ad5b2d5ced4fc998.md", "/raw-evidence/documents/2026/08/17/../../secret.md"} {
		t.Run(rawText, func(t *testing.T) {
			repo := &FakeDataServiceRepo{GetRawEvidenceDocumentFunc: func(context.Context, string) (RawEvidenceDocument, error) {
				return RawEvidenceDocument{RawText: rawText}, nil
			}}
			document, err := NewService(repo, WithRawEvidencePublicBaseURL("http://127.0.0.1:9000")).GetCollectionDocument(context.Background(), "RAW00000000-0000-5000-8000-000000000001")
			if err != nil || document.Available || document.URL != "" {
				t.Fatalf("document/error = %#v/%v", document, err)
			}
		})
	}

	want := errors.New("dependency failed")
	repo := &FakeDataServiceRepo{GetRawEvidenceDocumentFunc: func(context.Context, string) (RawEvidenceDocument, error) {
		return RawEvidenceDocument{}, want
	}}
	_, err := NewService(repo, WithRawEvidencePublicBaseURL("http://127.0.0.1:9000")).GetCollectionDocument(context.Background(), "RAW00000000-0000-5000-8000-000000000001")
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
}
