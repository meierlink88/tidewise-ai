package country

import (
	"context"
	"testing"
)

const (
	testCountryID  = "COU11111111-1111-4111-8111-111111111111"
	testRegionAPAC = "REGaaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	testRegionEM   = "REGbbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
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
func (s *countryStoreStub) ReplaceRegions(_ context.Context, id string, links []RegionLink) (Country, error) {
	s.replacedID = id
	s.replacedLinks = make([]string, len(links))
	for index, link := range links {
		s.replacedLinks[index] = link.RegionID
	}
	return Country{ID: id}, nil
}

func TestUseCaseEnforcesCountryIdentityAndValueRulesBeforePersistence(t *testing.T) {
	for _, test := range []struct {
		name  string
		input Country
	}{
		{name: "legacy ID", input: Country{ID: "COU_CHN", Code: "CN", Name: "中国", NameEn: "China"}},
		{name: "lowercase code", input: Country{ID: testCountryID, Code: "cn", Name: "中国", NameEn: "China"}},
		{name: "alpha-3 code", input: Country{ID: testCountryID, Code: "CHN", Name: "中国", NameEn: "China"}},
		{name: "blank name", input: Country{ID: testCountryID, Code: "CN", Name: " ", NameEn: "China"}},
		{name: "blank optional fact", input: Country{ID: testCountryID, Code: "CN", Name: "中国", NameEn: "China", KeyResources: stringPointer(" ")}},
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
	if _, err := useCase.ReplaceRegions(context.Background(), testCountryID, []string{testRegionAPAC, testRegionAPAC}); err == nil {
		t.Fatal("duplicate Region set error = nil")
	}
	input := []string{testRegionAPAC, testRegionEM}
	if _, err := useCase.ReplaceRegions(context.Background(), testCountryID, input); err != nil {
		t.Fatal(err)
	}
	input[0] = "REGcccccccc-cccc-4ccc-8ccc-cccccccccccc"
	if store.replacedID != testCountryID || store.replacedLinks[0] != testRegionAPAC {
		t.Fatalf("persisted Region replacement = %s %#v", store.replacedID, store.replacedLinks)
	}
}

func stringPointer(value string) *string { return &value }
