package country

import (
	"context"
	"testing"
)

type countryStoreStub struct {
	created       Country
	replacedID    string
	replacedLinks []string
}

func (s *countryStoreStub) Create(_ context.Context, input Country) (Country, error) {
	s.created = input
	return input, nil
}

func (*countryStoreStub) Get(context.Context, string) (Country, error)    { return Country{}, nil }
func (*countryStoreStub) List(context.Context, string) ([]Country, error) { return nil, nil }
func (*countryStoreStub) Update(context.Context, string, Update) (Country, error) {
	return Country{}, nil
}
func (s *countryStoreStub) ReplaceRegions(_ context.Context, id string, regionIDs []string) (Country, error) {
	s.replacedID = id
	s.replacedLinks = append([]string(nil), regionIDs...)
	return Country{ID: id}, nil
}

func TestUseCaseEnforcesCountryIdentityAndValueRulesBeforePersistence(t *testing.T) {
	for _, test := range []struct {
		name  string
		input Country
	}{
		{name: "ID and code differ", input: Country{ID: "COU_USA", Code: "CHN", Name: "中国", NameEn: "China"}},
		{name: "lowercase code", input: Country{ID: "COU_chn", Code: "chn", Name: "中国", NameEn: "China"}},
		{name: "blank name", input: Country{ID: "COU_CHN", Code: "CHN", Name: " ", NameEn: "China"}},
		{name: "blank optional fact", input: Country{ID: "COU_CHN", Code: "CHN", Name: "中国", NameEn: "China", KeyResources: stringPointer(" ")}},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &countryStoreStub{}
			useCase, err := NewUseCase(store)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := useCase.Create(context.Background(), test.input); err == nil {
				t.Fatal("Create() error = nil")
			}
			if store.created.ID != "" {
				t.Fatalf("invalid Country reached persistence: %#v", store.created)
			}
		})
	}
}

func TestReplaceRegionsRejectsDuplicatesAndCopiesTheAcceptedSet(t *testing.T) {
	store := &countryStoreStub{}
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := useCase.ReplaceRegions(context.Background(), "COU_CHN", []string{"REG_APAC", "REG_APAC"}); err == nil {
		t.Fatal("duplicate Region set error = nil")
	}
	input := []string{"REG_APAC", "REG_EM"}
	if _, err := useCase.ReplaceRegions(context.Background(), "COU_CHN", input); err != nil {
		t.Fatal(err)
	}
	input[0] = "REG_CHANGED"
	if store.replacedID != "COU_CHN" || store.replacedLinks[0] != "REG_APAC" {
		t.Fatalf("persisted Region replacement = %s %#v", store.replacedID, store.replacedLinks)
	}
}

func stringPointer(value string) *string { return &value }
