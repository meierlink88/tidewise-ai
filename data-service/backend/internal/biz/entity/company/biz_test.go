package company

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	testCompanyID       = ID("COM11111111-1111-4111-8111-111111111111")
	testIndustryID      = IndustryID("INDaaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	testOtherIndustryID = IndustryID("INDbbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb")
)

type repositoryStub struct {
	created       Company
	updatedID     ID
	updated       Update
	replacedID    ID
	replacedLinks []IndustryLink
}

func (s *repositoryStub) Create(_ context.Context, input Company) (Company, error) {
	s.created = input
	return input, nil
}

func (*repositoryStub) Get(context.Context, ID) (Company, error) { return Company{}, nil }

func (*repositoryStub) List(context.Context) ([]Company, error) { return nil, nil }

func (s *repositoryStub) Update(_ context.Context, id ID, input Update) (Company, error) {
	s.updatedID = id
	s.updated = input
	return Company{ID: id}, nil
}

func (s *repositoryStub) ReplaceIndustries(_ context.Context, id ID, links []IndustryLink) (Company, error) {
	s.replacedID = id
	s.replacedLinks = append([]IndustryLink(nil), links...)
	return Company{ID: id}, nil
}

func TestCreateGeneratesCompanyIdentityAndCopiesAcceptedFacts(t *testing.T) {
	repository := &repositoryStub{}
	useCase, err := NewUseCase(repository)
	if err != nil {
		t.Fatal(err)
	}
	aliases := []string{"台积电"}
	input := validCreateInput()
	input.Aliases = aliases
	created, err := useCase.Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	aliases[0] = "mutated"
	if !IsID(string(created.ID)) || repository.created.ID != created.ID {
		t.Fatalf("Create() identity = %q, persisted = %q", created.ID, repository.created.ID)
	}
	if repository.created.Aliases[0] != "台积电" {
		t.Fatalf("persisted aliases = %#v", repository.created.Aliases)
	}
}

func TestCreateRejectsInvalidCompanyBeforePersistence(t *testing.T) {
	founding := time.Date(2020, time.January, 2, 0, 0, 0, 0, time.UTC)
	beforeFounding := founding.AddDate(-1, 0, 0)
	for _, test := range []struct {
		name   string
		mutate func(*CreateInput)
	}{
		{name: "blank code", mutate: func(value *CreateInput) { value.Code = " " }},
		{name: "long code", mutate: func(value *CreateInput) { value.Code = strings.Repeat("x", 31) }},
		{name: "blank name", mutate: func(value *CreateInput) { value.Name = " " }},
		{name: "duplicate alias", mutate: func(value *CreateInput) { value.Aliases = []string{"TSMC", "TSMC"} }},
		{name: "blank optional fact", mutate: func(value *CreateInput) { value.Description = stringPointer(" ") }},
		{name: "unknown registration Country", mutate: func(value *CreateInput) { value.RegistrationCountryID = stringPointer("COU_CN") }},
		{name: "unsupported ownership", mutate: func(value *CreateInput) { value.OwnershipType = ownershipPointer("UNKNOWN") }},
		{name: "IPO before founding", mutate: func(value *CreateInput) { value.FoundingDate = &founding; value.IPODate = &beforeFounding }},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &repositoryStub{}
			useCase, err := NewUseCase(repository)
			if err != nil {
				t.Fatal(err)
			}
			input := validCreateInput()
			test.mutate(&input)
			if _, err := useCase.Create(context.Background(), input); err == nil {
				t.Fatal("Create() error = nil")
			}
			if repository.created.ID != "" {
				t.Fatalf("invalid Company reached persistence: %#v", repository.created)
			}
		})
	}
}

func TestUpdateKeepsCodeOutsideTheMutableContract(t *testing.T) {
	repository := &repositoryStub{}
	useCase, err := NewUseCase(repository)
	if err != nil {
		t.Fatal(err)
	}
	input := validUpdate()
	if _, err := useCase.Update(context.Background(), testCompanyID, input); err != nil {
		t.Fatal(err)
	}
	if repository.updatedID != testCompanyID || repository.updated.Name != input.Name {
		t.Fatalf("Update() persisted %q %#v", repository.updatedID, repository.updated)
	}
}

func TestReplaceIndustriesRejectsDuplicatesAndDerivesStableLinks(t *testing.T) {
	repository := &repositoryStub{}
	useCase, err := NewUseCase(repository)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := useCase.ReplaceIndustries(context.Background(), testCompanyID, []IndustryID{testIndustryID, testIndustryID}); err == nil {
		t.Fatal("ReplaceIndustries() duplicate error = nil")
	}
	input := []IndustryID{testIndustryID, testOtherIndustryID}
	if _, err := useCase.ReplaceIndustries(context.Background(), testCompanyID, input); err != nil {
		t.Fatal(err)
	}
	first := append([]IndustryLink(nil), repository.replacedLinks...)
	input[0] = IndustryID("INDcccccccc-cccc-4ccc-8ccc-cccccccccccc")
	if repository.replacedID != testCompanyID || len(first) != 2 || first[0].IndustryID != testIndustryID {
		t.Fatalf("ReplaceIndustries() persisted %q %#v", repository.replacedID, first)
	}
	if _, err := useCase.ReplaceIndustries(context.Background(), testCompanyID, []IndustryID{testIndustryID, testOtherIndustryID}); err != nil {
		t.Fatal(err)
	}
	if first[0].ID != repository.replacedLinks[0].ID || first[1].ID != repository.replacedLinks[1].ID {
		t.Fatalf("derived links are not stable: %#v then %#v", first, repository.replacedLinks)
	}
}

func TestValidatePersistedRejectsBrokenCompanyState(t *testing.T) {
	input := validCompany()
	input.ID = testCompanyID
	input.CreatedAt = time.Now().UTC()
	input.UpdatedAt = input.CreatedAt.Add(-time.Second)
	if err := ValidatePersisted(input); err == nil {
		t.Fatal("ValidatePersisted() error = nil")
	}
	input.UpdatedAt = input.CreatedAt
	input.Industries = nil
	if err := ValidatePersisted(input); err == nil {
		t.Fatal("ValidatePersisted() nil industries error = nil")
	}
}

func TestNewUseCaseRequiresRepository(t *testing.T) {
	_, err := NewUseCase(nil)
	if err == nil || !strings.Contains(err.Error(), "repository") {
		t.Fatalf("NewUseCase(nil) error = %v", err)
	}
}

func TestGetRejectsInvalidIdentity(t *testing.T) {
	useCase, err := NewUseCase(&repositoryStub{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = useCase.Get(context.Background(), "ENT11111111-1111-4111-8111-111111111111")
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("Get() error = %v, want ValidationError", err)
	}
}

func validCompany() Company {
	return Company{
		Code:       "TSM",
		Name:       "台积电",
		Aliases:    []string{},
		Status:     StatusActive,
		Industries: []Industry{},
	}
}

func validCreateInput() CreateInput {
	return CreateInput{Code: "TSM", Name: "台积电", Aliases: []string{}, Status: StatusActive}
}

func validUpdate() Update {
	return Update{Name: "台积电", Aliases: []string{}, Status: StatusActive}
}

func stringPointer(value string) *string { return &value }

func ownershipPointer(value OwnershipType) *OwnershipType { return &value }
