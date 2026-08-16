package organization

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	organizationapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/organization"
	organizationbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/organization"
)

type useCaseStub struct {
	err             error
	organization    organizationbiz.Organization
	organizations   []organizationbiz.Organization
	catalog         organizationbiz.Catalog
	members         []organizationbiz.Member
	listFilter      organizationbiz.Filter
	listMembersAsOf *time.Time
	createdMember   organizationbiz.Member
}

func (s *useCaseStub) Create(context.Context, organizationbiz.Organization) (organizationbiz.Organization, error) {
	return s.organization, s.err
}
func (s *useCaseStub) List(_ context.Context, filter organizationbiz.Filter) ([]organizationbiz.Organization, error) {
	s.listFilter = filter
	return s.organizations, s.err
}
func (s *useCaseStub) Get(context.Context, string) (organizationbiz.Organization, error) {
	return s.organization, s.err
}
func (s *useCaseStub) Update(context.Context, string, organizationbiz.Update) (organizationbiz.Organization, error) {
	return s.organization, s.err
}
func (s *useCaseStub) ReplaceDomainTags(context.Context, string, []string) (organizationbiz.Organization, error) {
	return s.organization, s.err
}
func (s *useCaseStub) Catalog(context.Context) (organizationbiz.Catalog, error) {
	return s.catalog, s.err
}
func (s *useCaseStub) ListMembers(_ context.Context, _ string, asOf *time.Time) ([]organizationbiz.Member, error) {
	s.listMembersAsOf = asOf
	return s.members, s.err
}
func (s *useCaseStub) CreateMember(_ context.Context, input organizationbiz.Member) (organizationbiz.Member, error) {
	s.createdMember = input
	return input, s.err
}
func (s *useCaseStub) UpdateMember(context.Context, string, string, organizationbiz.Member) (organizationbiz.Member, error) {
	return organizationbiz.Member{}, s.err
}
func (s *useCaseStub) DeleteMember(context.Context, string, string) error { return s.err }

func TestNewServiceRejectsMissingUseCase(t *testing.T) {
	if _, err := NewService(nil); err == nil {
		t.Fatal("NewService(nil) error = nil")
	}
}

func TestServiceParsesFiltersAndMapsOrganizationDTO(t *testing.T) {
	established := time.Date(1945, 10, 24, 0, 0, 0, 0, time.UTC)
	created := time.Date(2026, 8, 15, 1, 2, 3, 4, time.FixedZone("test", 8*60*60))
	stub := &useCaseStub{organizations: []organizationbiz.Organization{{
		ID: "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", Code: "UN", Name: "联合国", NameEn: "United Nations",
		Category:        organizationbiz.Category{ID: "OCA7cf04802-4d04-5a8c-9a10-7d805cf29a4d", Code: "INTERGOVERNMENTAL", NameZh: "政府间国际组织"},
		Function:        organizationbiz.Function{Code: "SECURITY", NameZh: "安全与防务"},
		EstablishedDate: &established,
		DomainTags:      []organizationbiz.DomainTag{{ID: "ODT37166e5a-05da-5972-b5a8-ff2c85ddc76a", Code: "REGIONAL_SECURITY_DIALOGUE", FunctionCode: "SECURITY", NameZh: "区域安全对话"}},
		CreatedAt:       created, UpdatedAt: created,
	}}}
	service, err := NewService(stub)
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.List(context.Background(), &organizationapi.ListRequest{
		CategoryCode: "INTERGOVERNMENTAL", FunctionCode: "SECURITY", RegionID: "REG13802abf-d1ef-5dec-95ec-a47d35813827",
		CountryID: "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b", AsOf: "2026-08-15",
	})
	if err != nil {
		t.Fatal(err)
	}
	if stub.listFilter.AsOfDate == nil || stub.listFilter.AsOfDate.Format("2006-01-02") != "2026-08-15" {
		t.Fatalf("List filter = %#v", stub.listFilter)
	}
	if len(response.Result.Items) != 1 || response.Result.Items[0].EstablishedDate == nil ||
		*response.Result.Items[0].EstablishedDate != "1945-10-24" || response.Result.Items[0].CreatedAt != "2026-08-14T17:02:03.000000004Z" ||
		response.Result.Items[0].Category.ID != "OCA7cf04802-4d04-5a8c-9a10-7d805cf29a4d" ||
		len(response.Result.Items[0].DomainTags) != 1 || response.Result.Items[0].DomainTags[0].ID != "ODT37166e5a-05da-5972-b5a8-ff2c85ddc76a" {
		t.Fatalf("Organization DTO = %#v", response.Result.Items)
	}

	_, err = service.List(context.Background(), &organizationapi.ListRequest{AsOf: "2026/08/15"})
	assertPublicError(t, err, v1.StatusUnprocessableEntity, "ORGANIZATION_INVALID")
}

func TestServiceMapsCatalogIdentities(t *testing.T) {
	stub := &useCaseStub{catalog: organizationbiz.Catalog{
		Categories: []organizationbiz.Category{{ID: "OCA7cf04802-4d04-5a8c-9a10-7d805cf29a4d", Code: "INTERGOVERNMENTAL", NameZh: "政府间国际组织"}},
		Functions:  []organizationbiz.Function{{Code: "SECURITY", NameZh: "安全与防务"}},
		DomainTags: []organizationbiz.DomainTag{{ID: "ODT37166e5a-05da-5972-b5a8-ff2c85ddc76a", Code: "REGIONAL_SECURITY_DIALOGUE", FunctionCode: "SECURITY", NameZh: "区域安全对话"}},
	}}
	service, err := NewService(stub)
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.GetCatalog(context.Background(), &organizationapi.CatalogRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if response.Result.Categories[0].ID != stub.catalog.Categories[0].ID || response.Result.DomainTags[0].ID != stub.catalog.DomainTags[0].ID {
		t.Fatalf("catalog identities = %#v", response.Result)
	}
}

func TestServiceMapsMemberDatesAndNullableValues(t *testing.T) {
	effective := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	expiry := time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC)
	created := time.Date(2026, 8, 15, 1, 2, 3, 0, time.UTC)
	stub := &useCaseStub{members: []organizationbiz.Member{{
		ID: "OMB77777777-7777-4777-8777-777777777777", OrganizationID: "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", CountryID: "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b", MembershipType: "FULL_MEMBER",
		EffectiveDate: &effective, ExpiryDate: &expiry, CreatedAt: created, UpdatedAt: created,
	}}}
	service, err := NewService(stub)
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.ListMembers(context.Background(), &organizationapi.ListMembersRequest{OrganizationID: "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", AsOf: "2020-06-01"})
	if err != nil {
		t.Fatal(err)
	}
	if stub.listMembersAsOf == nil || len(response.Result.Items) != 1 || *response.Result.Items[0].EffectiveDate != "2020-01-01" || *response.Result.Items[0].ExpiryDate != "2020-12-31" {
		t.Fatalf("member response = %#v; asOf=%v", response.Result, stub.listMembersAsOf)
	}
	date := organizationapi.Date{Time: effective}
	if _, err := service.CreateMember(context.Background(), &organizationapi.CreateMemberRequest{
		OrganizationID: "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", CountryID: "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b", MembershipType: "OBSERVER", EffectiveDate: &date,
	}); err != nil {
		t.Fatal(err)
	}
	if stub.createdMember.EffectiveDate == nil || stub.createdMember.ExpiryDate != nil {
		t.Fatalf("member input = %#v", stub.createdMember)
	}
}

func TestServiceMapsStableOrganizationErrors(t *testing.T) {
	for _, test := range []struct {
		name       string
		domainErr  error
		wantStatus int
		wantCode   string
		wantError  error
	}{
		{name: "validation", domainErr: &organizationbiz.ValidationError{Field: "code", Message: "invalid"}, wantStatus: v1.StatusUnprocessableEntity, wantCode: "ORGANIZATION_INVALID"},
		{name: "reference", domainErr: &organizationbiz.ReferenceError{Field: "country_id", Message: "missing"}, wantStatus: v1.StatusUnprocessableEntity, wantCode: "ORGANIZATION_REFERENCE_INVALID"},
		{name: "not found", domainErr: organizationbiz.ErrNotFound, wantStatus: v1.StatusNotFound, wantCode: "ORGANIZATION_NOT_FOUND"},
		{name: "conflict", domainErr: organizationbiz.ErrConflict, wantStatus: v1.StatusConflict, wantCode: "ORGANIZATION_CONFLICT"},
		{name: "deadline", domainErr: context.DeadlineExceeded, wantStatus: v1.StatusServiceUnavailable, wantCode: "ORGANIZATION_TIMEOUT"},
		{name: "persistence", domainErr: organizationbiz.ErrPersistence, wantStatus: v1.StatusInternalServerError, wantCode: "ORGANIZATION_FAILED"},
		{name: "cancellation", domainErr: context.Canceled, wantError: context.Canceled},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, err := NewService(&useCaseStub{err: test.domainErr})
			if err != nil {
				t.Fatal(err)
			}
			_, err = service.Get(context.Background(), &organizationapi.GetRequest{OrganizationID: "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549"})
			if test.wantError != nil {
				if !errors.Is(err, test.wantError) {
					t.Fatalf("Get() error = %v, want %v", err, test.wantError)
				}
				return
			}
			assertPublicError(t, err, test.wantStatus, test.wantCode)
		})
	}
}

func assertPublicError(t *testing.T, err error, wantStatus int, wantCode string) {
	t.Helper()
	var public *v1.PublicError
	if !errors.As(err, &public) || public.Status != wantStatus || public.Code != wantCode {
		t.Fatalf("public error = %#v, want status=%d code=%s", err, wantStatus, wantCode)
	}
}
