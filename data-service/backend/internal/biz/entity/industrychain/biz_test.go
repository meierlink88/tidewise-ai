package industrychain

import (
	"context"
	"testing"
	"time"
)

type repositoryStub struct{ created IndustryChain }

func (s *repositoryStub) Create(_ context.Context, input IndustryChain) (IndustryChain, error) {
	s.created = input
	return input, nil
}

func (*repositoryStub) Get(context.Context, ID) (IndustryChain, error) { return IndustryChain{}, nil }
func (*repositoryStub) List(context.Context, ListQuery) (ListResult, error) {
	return ListResult{}, nil
}
func (*repositoryStub) Update(context.Context, ID, Update) (IndustryChain, error) {
	return IndustryChain{}, nil
}

func TestCreateGeneratesIndustryChainIdentity(t *testing.T) {
	store := &repositoryStub{}
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	created, err := useCase.Create(context.Background(), IndustryChain{
		Name: "半导体产业链", Aliases: []string{}, Scope: "全球", TargetOutput: "芯片", EndUse: "电子产品",
		Geography: "global", AsOfDate: time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC),
		ReviewStatus: ReviewStatusCandidate, ObservableVariables: []string{"产量"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !IsID(string(created.ID)) || string(created.ID)[:3] != "ICH" || store.created.ID != created.ID {
		t.Fatalf("generated IndustryChain identity = %q, persisted = %q", created.ID, store.created.ID)
	}
	for _, value := range []string{
		"ENT11111111-1111-4111-8111-111111111111",
		"IND11111111-1111-4111-8111-111111111111",
		"CON11111111-1111-4111-8111-111111111111",
		"CND11111111-1111-4111-8111-111111111111",
	} {
		if IsID(value) {
			t.Errorf("IndustryChain accepted identity %q with another object's prefix", value)
		}
	}
}
