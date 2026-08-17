package concept

import (
	"context"
	"testing"
)

const testConceptID = "ENT33333333-3333-4333-8333-333333333333"

type repositoryStub struct {
	created    Concept
	listQuery  ListQuery
	listResult ListResult
}

func (s *repositoryStub) Create(_ context.Context, input Concept) (Concept, error) {
	s.created = input
	return input, nil
}

func (*repositoryStub) Get(context.Context, ID) (Concept, error) { return Concept{}, nil }
func (s *repositoryStub) List(_ context.Context, query ListQuery) (ListResult, error) {
	s.listQuery = query
	return s.listResult, nil
}

func TestListUsesOpaqueStableKeysetCursor(t *testing.T) {
	item := Concept{ID: ID(testConceptID), Name: "人工智能"}
	store := &repositoryStub{listResult: ListResult{Items: []Concept{item}, HasMore: true}}
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	first, err := useCase.List(context.Background(), ListRequest{PageSize: 1})
	if err != nil || first.NextCursor == nil {
		t.Fatalf("first page = %#v, error = %v", first, err)
	}
	store.listResult = ListResult{}
	if _, err := useCase.List(context.Background(), ListRequest{PageSize: 1, Cursor: *first.NextCursor}); err != nil {
		t.Fatal(err)
	}
	if store.listQuery.After == nil || store.listQuery.After.ID != item.ID || store.listQuery.After.Name != item.Name {
		t.Fatalf("decoded Concept keyset = %#v", store.listQuery.After)
	}
	if _, err := useCase.List(context.Background(), ListRequest{PageSize: 1, Cursor: "not-a-cursor"}); err == nil {
		t.Fatal("invalid cursor error = nil")
	}
}
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
