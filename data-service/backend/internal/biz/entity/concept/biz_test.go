package concept

import (
	"context"
	"testing"
)

const testConceptID = "ENT33333333-3333-4333-8333-333333333333"

type repositoryStub struct{ created Concept }

func (s *repositoryStub) Create(_ context.Context, input Concept) (Concept, error) {
	s.created = input
	return input, nil
}

func (*repositoryStub) Get(context.Context, ID) (Concept, error) { return Concept{}, nil }
func (*repositoryStub) List(context.Context) ([]Concept, error)  { return nil, nil }
func (*repositoryStub) Update(context.Context, ID, Update) (Concept, error) {
	return Concept{}, nil
}

func TestCreateGeneratesIdentityAndRejectsInvalidConceptBeforePersistence(t *testing.T) {
	valid := Concept{
		Name: "人工智能", Aliases: []string{"AI"}, ConceptType: TypeTechnology,
		Definition: "跨行业的人工智能技术主题", ReviewStatus: ReviewStatusCandidate,
	}
	store := &repositoryStub{}
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	created, err := useCase.Create(context.Background(), valid)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || store.created.ID != created.ID || !IsID(string(created.ID)) {
		t.Fatalf("generated Concept identity = %q, persisted = %q", created.ID, store.created.ID)
	}

	for name, mutate := range map[string]func(*Concept){
		"caller ID":        func(input *Concept) { input.ID = ID(testConceptID) },
		"unsupported type": func(input *Concept) { input.ConceptType = "sector" },
		"duplicate alias":  func(input *Concept) { input.Aliases = []string{"AI", "AI"} },
		"missing aliases":  func(input *Concept) { input.Aliases = nil },
	} {
		t.Run(name, func(t *testing.T) {
			input := valid
			mutate(&input)
			if _, err := useCase.Create(context.Background(), input); err == nil {
				t.Fatal("Create() error = nil")
			}
		})
	}
}

func TestUpdateAcceptsCompleteConceptReplacement(t *testing.T) {
	store := &repositoryStub{}
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	_, err = useCase.Update(context.Background(), testConceptID, Update{
		Name: "生成式人工智能", Aliases: []string{"GenAI"}, ConceptType: TypeTechnology,
		Definition: "生成内容的人工智能技术主题", ReviewStatus: ReviewStatusApproved,
	})
	if err != nil {
		t.Fatal(err)
	}
}
