package chainnode

import (
	"context"
	"testing"
)

type repositoryStub struct{ created ChainNode }

func (s *repositoryStub) Create(_ context.Context, input ChainNode) (ChainNode, error) {
	s.created = input
	return input, nil
}

func (*repositoryStub) Get(context.Context, ID) (ChainNode, error) { return ChainNode{}, nil }
func (*repositoryStub) List(context.Context, ListQuery) (ListResult, error) {
	return ListResult{}, nil
}
func (*repositoryStub) Update(context.Context, ID, Update) (ChainNode, error) {
	return ChainNode{}, nil
}

func TestCreateGeneratesChainNodeIdentity(t *testing.T) {
	store := &repositoryStub{}
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	created, err := useCase.Create(context.Background(), ChainNode{
		Name: "晶圆制造", Aliases: []string{}, Definition: "晶圆制造环节", ReviewStatus: ReviewStatusCandidate,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !IsID(string(created.ID)) || string(created.ID)[:3] != "CND" || store.created.ID != created.ID {
		t.Fatalf("generated ChainNode identity = %q, persisted = %q", created.ID, store.created.ID)
	}
	for _, value := range []string{
		"ENT11111111-1111-4111-8111-111111111111",
		"IND11111111-1111-4111-8111-111111111111",
		"CON11111111-1111-4111-8111-111111111111",
		"ICH11111111-1111-4111-8111-111111111111",
	} {
		if IsID(value) {
			t.Errorf("ChainNode accepted identity %q with another object's prefix", value)
		}
	}
}
