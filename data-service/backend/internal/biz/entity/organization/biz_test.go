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
	updated            Update
	replacedID         string
	replacedCodes      []string
	createdMember      Member
	updatedMemberOrgID string
	updatedMemberID    int64
	updatedMember      Member
	deletedMemberOrgID string
	deletedMemberID    int64
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
func (s *repositoryStub) Update(_ context.Context, id string, input Update) (Organization, error) {
	s.updatedID, s.updated = id, input
	return Organization{ID: id}, nil
}
func (s *repositoryStub) ReplaceDomainTags(_ context.Context, id string, codes []string) (Organization, error) {
	s.replacedID, s.replacedCodes = id, codes
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
func (s *repositoryStub) UpdateMember(_ context.Context, organizationID string, id int64, input Member) (Member, error) {
	s.updatedMemberOrgID, s.updatedMemberID, s.updatedMember = organizationID, id, input
	return input, nil
}
func (s *repositoryStub) DeleteMember(_ context.Context, organizationID string, id int64) error {
	s.deletedMemberOrgID, s.deletedMemberID = organizationID, id
	return nil
}

func TestNewUseCaseRejectsMissingRepository(t *testing.T) {
	if _, err := NewUseCase(nil); err == nil {
		t.Fatal("NewUseCase(nil) error = nil")
	}
}

func TestUseCaseRejectsInvalidOrganizationBeforePersistence(t *testing.T) {
	stringValue := func(value string) *string { return &value }
	valid := validOrganization()
	for _, test := range []struct {
		name   string
		mutate func(*Organization)
	}{
		{name: "ID differs from code", mutate: func(input *Organization) { input.ID = "ORG_OTHER" }},
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
	regionID := "REG_GLOBAL"
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
	if *repository.created.RegionID != "REG_GLOBAL" || repository.created.DomainTags[0].Code != "REGIONAL_SECURITY_DIALOGUE" {
		t.Fatalf("persisted Organization aliases caller input: %#v", repository.created)
	}

	codes := []string{"REGIONAL_SECURITY_DIALOGUE"}
	if _, err := useCase.ReplaceDomainTags(context.Background(), "ORG_UN", codes); err != nil {
		t.Fatal(err)
	}
	codes[0] = "CHANGED"
	if repository.replacedCodes[0] != "REGIONAL_SECURITY_DIALOGUE" {
		t.Fatalf("persisted Domain Tags alias caller input: %#v", repository.replacedCodes)
	}
	if _, err := useCase.ReplaceDomainTags(context.Background(), "ORG_UN", []string{"REGIONAL_SECURITY_DIALOGUE", "REGIONAL_SECURITY_DIALOGUE"}); err == nil {
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
		{OrganizationID: "UN", CountryID: "COU_CHN", MembershipType: "FULL_MEMBER"},
		{OrganizationID: "ORG_UN", CountryID: "CHN", MembershipType: "FULL_MEMBER"},
		{OrganizationID: "ORG_UN", CountryID: "COU_CHN", MembershipType: "MEMBER"},
		{OrganizationID: "ORG_UN", CountryID: "COU_CHN", MembershipType: "FULL_MEMBER", EffectiveDate: &effective, ExpiryDate: &expiry},
	} {
		if _, err := useCase.CreateMember(context.Background(), member); err == nil {
			t.Fatalf("CreateMember(%#v) error = nil", member)
		}
	}
	if repository.createdMember.OrganizationID != "" {
		t.Fatalf("invalid member reached persistence: %#v", repository.createdMember)
	}
	valid := Member{CountryID: "COU_CHN", MembershipType: "OBSERVER"}
	if _, err := useCase.UpdateMember(context.Background(), "ORG_UN", 7, valid); err != nil {
		t.Fatal(err)
	}
	if repository.updatedMemberOrgID != "ORG_UN" || repository.updatedMemberID != 7 || repository.updatedMember.OrganizationID != "ORG_UN" {
		t.Fatalf("UpdateMember persistence input = %#v", repository.updatedMember)
	}
	if err := useCase.DeleteMember(context.Background(), "ORG_UN", 0); err == nil {
		t.Fatal("DeleteMember() accepted non-positive member ID")
	}
}

func validOrganization() Organization {
	return Organization{
		ID: "ORG_UN", Code: "UN", Name: "联合国", NameEn: "United Nations",
		Category: CatalogTerm{Code: "INTERGOVERNMENTAL"}, Function: CatalogTerm{Code: "SECURITY"},
	}
}
