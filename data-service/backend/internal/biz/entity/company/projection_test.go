package company

import (
	"context"
	"errors"
	"testing"
	"time"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

type projectionRepositoryStub struct {
	*repositoryStub
	queries []ProjectionListQuery
	pages   []ProjectionListResult
	err     error
}

func (s *projectionRepositoryStub) ListProjection(_ context.Context, query ProjectionListQuery) (ProjectionListResult, error) {
	s.queries = append(s.queries, query)
	if s.err != nil {
		return ProjectionListResult{}, s.err
	}
	result := s.pages[0]
	s.pages = s.pages[1:]
	return result, nil
}

func TestListProjectionCarriesSnapshotInOpaqueCursor(t *testing.T) {
	first := validCompany()
	first.ID = "COM11111111-1111-4111-8111-111111111111"
	first.Code = "000001.SZ"
	first.CreatedAt = time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	first.UpdatedAt = first.CreatedAt
	first.IndustryLinks = []IndustryLink{}
	second := validCompany()
	second.ID = "COM22222222-2222-4222-8222-222222222222"
	second.Code = "000002.SZ"
	second.CreatedAt = first.CreatedAt
	second.UpdatedAt = first.CreatedAt
	second.IndustryLinks = []IndustryLink{}
	snapshotID := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	repository := &projectionRepositoryStub{
		repositoryStub: &repositoryStub{},
		pages: []ProjectionListResult{
			{SnapshotID: snapshotID, Items: []Company{first}, HasMore: true},
			{SnapshotID: snapshotID, Items: []Company{second}},
		},
	}
	useCase, err := NewProjectionUseCase(repository)
	if err != nil {
		t.Fatal(err)
	}
	page, err := useCase.ListProjection(context.Background(), ProjectionListRequest{PageSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	if page.SnapshotID != snapshotID || page.NextCursor == nil || len(page.Items) != 1 {
		t.Fatalf("first page = %#v", page)
	}
	secondPage, err := useCase.ListProjection(context.Background(), ProjectionListRequest{PageSize: 1, Cursor: *page.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(secondPage.Items) != 1 || secondPage.Items[0].ID != second.ID || secondPage.NextCursor != nil {
		t.Fatalf("second page = %#v", secondPage)
	}
	if len(repository.queries) != 2 || repository.queries[1].SnapshotID != snapshotID || repository.queries[1].After == nil || repository.queries[1].After.Code != first.Code || repository.queries[1].After.ID != first.ID {
		t.Fatalf("queries = %#v", repository.queries)
	}
}

func TestListProjectionRejectsMalformedCursorAndPropagatesSnapshotDrift(t *testing.T) {
	repository := &projectionRepositoryStub{repositoryStub: &repositoryStub{}, err: ErrProjectionSnapshotChanged}
	useCase, err := NewProjectionUseCase(repository)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := useCase.ListProjection(context.Background(), ProjectionListRequest{PageSize: 1, Cursor: "not-a-cursor"}); err == nil {
		t.Fatal("malformed cursor error = nil")
	}
	_, err = useCase.ListProjection(context.Background(), ProjectionListRequest{PageSize: 1})
	if !errors.Is(err, ErrProjectionSnapshotChanged) {
		t.Fatalf("snapshot drift error = %v", err)
	}
}

func TestListProjectionRejectsDuplicateRepositoryItems(t *testing.T) {
	item := validCompany()
	item.ID = "COM11111111-1111-4111-8111-111111111111"
	item.Code = "duplicate"
	item.CreatedAt = time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	item.UpdatedAt = item.CreatedAt
	item.IndustryLinks = []IndustryLink{}
	repository := &projectionRepositoryStub{
		repositoryStub: &repositoryStub{},
		pages: []ProjectionListResult{{
			SnapshotID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Items:      []Company{item, item},
		}},
	}
	useCase, err := NewProjectionUseCase(repository)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := useCase.ListProjection(context.Background(), ProjectionListRequest{PageSize: 2}); !errors.Is(err, ErrPersistence) {
		t.Fatalf("duplicate repository items error = %v", err)
	}
}

func TestValidateProjectionCompanyRejectsNonDerivedIndustryLinkIdentity(t *testing.T) {
	item := validCompany()
	item.ID = testCompanyID
	item.CreatedAt = time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	item.UpdatedAt = item.CreatedAt
	item.Industries = []Industry{{
		ID: testIndustryID, Name: "半导体", ClassificationSystem: "TIDEWISE", IndustryCode: "SEMICONDUCTOR",
	}}
	item.IndustryLinks = []IndustryLink{{
		ID: "CIL44444444-4444-4444-8444-444444444444", CompanyID: item.ID,
		IndustryID: testIndustryID, CreatedAt: item.CreatedAt,
	}}

	err := ValidateProjectionCompany(item)
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Field != "industry_links[0].id" {
		t.Fatalf("non-derived CompanyIndustryLink error = %v", err)
	}
	expectedID, err := coreid.Derive(
		coreid.CompanyIndustryLink,
		"company-industry-link",
		string(item.ID),
		string(testIndustryID),
	)
	if err != nil {
		t.Fatal(err)
	}
	item.IndustryLinks[0].ID = expectedID
	if err := ValidateProjectionCompany(item); err != nil {
		t.Fatalf("derived CompanyIndustryLink error = %v", err)
	}
}
