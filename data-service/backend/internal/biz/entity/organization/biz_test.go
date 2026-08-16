package organization

import (
	"context"
	"testing"
	"time"
)

type repositoryStub struct {
	created            Organization
	listed             Filter
	updatedID          string
	updated            UpdateCommand
	replacedID         string
	replacedLinks      []DomainTagLink
	createdMember      Member
	updatedMemberOrgID string
	updatedMemberID    string
	updatedMember      Member
	deletedMemberOrgID string
	deletedMemberID    string
}

func (s *repositoryStub) Create(_ context.Context, input Organization) (Organization, error) {
	s.created = input
	return input, nil
}
func (*repositoryStub) Get(context.Context, string) (Organization, error) { return Organization{}, nil }
func (s *repositoryStub) List(_ context.Context, input Filter) ([]Organization, error) {
	s.listed = input
	return nil, nil
}
func (s *repositoryStub) Update(_ context.Context, id string, input UpdateCommand) (Organization, error) {
	s.updatedID, s.updated = id, input
	return Organization{ID: id}, nil
}
func (s *repositoryStub) ReplaceDomainTags(_ context.Context, id string, links []DomainTagLink) (Organization, error) {
	s.replacedID, s.replacedLinks = id, links
	return Organization{ID: id}, nil
}
func (*repositoryStub) Catalog(context.Context) (Catalog, error) { return Catalog{}, nil }
func (*repositoryStub) ListMembers(context.Context, string, *time.Time) ([]Member, error) {
	return nil, nil
}
func (s *repositoryStub) CreateMember(_ context.Context, input Member) (Member, error) {
	s.createdMember = input
	return input, nil
}
func (s *repositoryStub) UpdateMember(_ context.Context, organizationID, id string, input Member) (Member, error) {
	s.updatedMemberOrgID, s.updatedMemberID, s.updatedMember = organizationID, id, input
	return input, nil
}
func (s *repositoryStub) DeleteMember(_ context.Context, organizationID, id string) error {
	s.deletedMemberOrgID, s.deletedMemberID = organizationID, id
	return nil
}

func TestNewUseCaseRejectsMissingRepository(t *testing.T) {
	if _, err := NewUseCase(nil); err == nil {
		t.Fatal("NewUseCase(nil) error = nil")
	}
}

func TestCurrentCatalogAssignsStableSystemOwnedIdentities(t *testing.T) {
	catalog, err := AssignCatalogIdentities(Catalog{
		Categories: []Category{{Code: "DIALOGUE_MECHANISM"}, {Code: "INTERGOVERNMENTAL"}},
		Functions:  []Function{{Code: "SECURITY"}},
		DomainTags: []DomainTag{{Code: "REGIONAL_SECURITY_DIALOGUE", FunctionCode: "SECURITY"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if catalog.Categories[1].ID != "OCA7cf04802-4d04-5a8c-9a10-7d805cf29a4d" {
		t.Fatalf("INTERGOVERNMENTAL Category ID = %q", catalog.Categories[1].ID)
	}
	if catalog.Functions[0].ID != "OFN1cd93122-d11c-5059-87aa-ddb8b2f2d25b" {
		t.Fatalf("SECURITY Function ID = %q", catalog.Functions[0].ID)
	}
	for _, tag := range catalog.DomainTags {
		if tag.Code == "REGIONAL_SECURITY_DIALOGUE" {
			if tag.ID != "ODT37166e5a-05da-5972-b5a8-ff2c85ddc76a" {
				t.Fatalf("REGIONAL_SECURITY_DIALOGUE Domain Tag ID = %q", tag.ID)
			}
			return
		}
	}
	t.Fatal("REGIONAL_SECURITY_DIALOGUE Domain Tag is missing")
}

func TestUseCaseRejectsInvalidOrganizationBeforePersistence(t *testing.T) {
	stringValue := func(value string) *string { return &value }
	valid := validOrganization()
	for _, test := range []struct {
		name   string
		mutate func(*Organization)
	}{
		{name: "legacy ID", mutate: func(input *Organization) { input.ID = "ORG_OTHER" }},
		{name: "lowercase code", mutate: func(input *Organization) { input.ID, input.Code = "ORG_un", "un" }},
		{name: "blank Chinese name", mutate: func(input *Organization) { input.Name = " " }},
		{name: "invalid category", mutate: func(input *Organization) { input.Category.Code = "trade" }},
		{name: "invalid Region ID", mutate: func(input *Organization) { input.RegionID = stringValue("region") }},
		{name: "invalid dominant Country", mutate: func(input *Organization) { input.DominantPartyID = stringValue("CHN") }},
		{name: "invalid binding power", mutate: func(input *Organization) { input.BindingPowerLevel = stringValue("VERY_HIGH") }},
		{name: "invalid influence", mutate: func(input *Organization) { input.InfluenceRating = stringValue("C") }},
		{name: "invalid LEI", mutate: func(input *Organization) { input.LegalEntityCode = stringValue("SHORT") }},
		{name: "blank optional fact", mutate: func(input *Organization) { input.Description = stringValue(" ") }},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := valid
			test.mutate(&input)
			repository := &repositoryStub{}
			useCase, err := NewUseCase(repository)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := useCase.Create(context.Background(), input); err == nil {
				t.Fatal("Create() error = nil")
			}
			if repository.created.ID != "" {
				t.Fatalf("invalid Organization reached persistence: %#v", repository.created)
			}
		})
	}
}

func TestUseCaseCopiesAcceptedOrganizationAndDomainTags(t *testing.T) {
	regionID := "REG13802abf-d1ef-5dec-95ec-a47d35813827"
	input := validOrganization()
	input.RegionID = &regionID
	input.DomainTags = []DomainTag{{Code: "REGIONAL_SECURITY_DIALOGUE", FunctionCode: "SECURITY", NameZh: "区域安全对话"}}
	repository := &repositoryStub{}
	useCase, err := NewUseCase(repository)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := useCase.Create(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	regionID = "REG_CHANGED"
	input.DomainTags[0].Code = "CHANGED"
	if *repository.created.RegionID != "REG13802abf-d1ef-5dec-95ec-a47d35813827" || repository.created.DomainTags[0].Code != "REGIONAL_SECURITY_DIALOGUE" {
		t.Fatalf("persisted Organization aliases caller input: %#v", repository.created)
	}

	codes := []string{"REGIONAL_SECURITY_DIALOGUE"}
	if _, err := useCase.ReplaceDomainTags(context.Background(), "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", codes); err != nil {
		t.Fatal(err)
	}
	codes[0] = "CHANGED"
	if repository.replacedLinks[0].DomainTagCode != "REGIONAL_SECURITY_DIALOGUE" {
		t.Fatalf("persisted Domain Tags alias caller input: %#v", repository.replacedLinks)
	}
	if repository.replacedLinks[0].ID != "ODL72d3c5ae-74ec-5d5e-9d3e-0d5ebbd189e8" {
		t.Fatalf("Domain Tag Link ID = %q", repository.replacedLinks[0].ID)
	}
	if _, err := useCase.ReplaceDomainTags(context.Background(), "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", []string{"REGIONAL_SECURITY_DIALOGUE", "REGIONAL_SECURITY_DIALOGUE"}); err == nil {
		t.Fatal("duplicate Domain Tags error = nil")
	}
}

func TestUseCaseValidatesFiltersAndMembershipBeforePersistence(t *testing.T) {
	repository := &repositoryStub{}
	useCase, err := NewUseCase(repository)
	if err != nil {
		t.Fatal(err)
	}
	for _, filter := range []Filter{
		{CategoryCode: "trade"},
		{FunctionCode: "security"},
		{RegionID: "REG_"},
		{CountryID: "CHN"},
	} {
		if _, err := useCase.List(context.Background(), filter); err == nil {
			t.Fatalf("List(%#v) error = nil", filter)
		}
	}
	effective := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	expiry := time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC)
	for _, member := range []Member{
		{OrganizationID: "UN", CountryID: "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b", MembershipType: "FULL_MEMBER"},
		{OrganizationID: "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", CountryID: "CHN", MembershipType: "FULL_MEMBER"},
		{OrganizationID: "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", CountryID: "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b", MembershipType: "MEMBER"},
		{OrganizationID: "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", CountryID: "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b", MembershipType: "FULL_MEMBER", EffectiveDate: &effective, ExpiryDate: &expiry},
	} {
		if _, err := useCase.CreateMember(context.Background(), member); err == nil {
			t.Fatalf("CreateMember(%#v) error = nil", member)
		}
	}
	if repository.createdMember.OrganizationID != "" {
		t.Fatalf("invalid member reached persistence: %#v", repository.createdMember)
	}
	valid := Member{CountryID: "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b", MembershipType: "OBSERVER"}
	if _, err := useCase.UpdateMember(context.Background(), "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", "OMB77777777-7777-4777-8777-777777777777", valid); err != nil {
		t.Fatal(err)
	}
	if repository.updatedMemberOrgID != "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549" || repository.updatedMemberID != "OMB77777777-7777-4777-8777-777777777777" || repository.updatedMember.OrganizationID != "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549" {
		t.Fatalf("UpdateMember persistence input = %#v", repository.updatedMember)
	}
	if err := useCase.DeleteMember(context.Background(), "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", "bad"); err == nil {
		t.Fatal("DeleteMember() accepted non-positive member ID")
	}
}

func validOrganization() Organization {
	return Organization{
		Code: "UN", Name: "联合国", NameEn: "United Nations",
		Category: Category{Code: "INTERGOVERNMENTAL"}, Function: Function{Code: "SECURITY"},
	}
}
