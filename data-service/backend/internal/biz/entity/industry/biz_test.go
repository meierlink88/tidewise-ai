package industry

import (
	"context"
	"testing"
)

const (
	testIndustryID       = "ENT11111111-1111-4111-8111-111111111111"
	testParentIndustryID = "ENT22222222-2222-4222-8222-222222222222"
)

type repositoryStub struct{ created Industry }

func (s *repositoryStub) Create(_ context.Context, input Industry) (Industry, error) {
	s.created = input
	return input, nil
}

func (*repositoryStub) Get(context.Context, string) (Industry, error) { return Industry{}, nil }
func (*repositoryStub) List(context.Context) ([]Industry, error)      { return nil, nil }
func (*repositoryStub) Update(context.Context, string, Update) (Industry, error) {
	return Industry{}, nil
}

func TestCreateGeneratesIdentityAndRejectsInvalidIndustryBeforePersistence(t *testing.T) {
	valid := Industry{
		Name: "半导体", Aliases: []string{"芯片产业"}, ClassificationSystem: "TIDEWISE",
		IndustryCode: "SEMICONDUCTOR", HierarchyPathCodes: []string{"SEMICONDUCTOR"},
		Definition: "半导体材料、设计、制造及相关活动", ReviewStatus: ReviewStatusCandidate,
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
	if created.ID == "" || store.created.ID != created.ID || !IsID(created.ID) {
		t.Fatalf("generated Industry identity = %q, persisted = %q", created.ID, store.created.ID)
	}

	invalid := valid
	invalid.ID = testIndustryID
	if _, err := useCase.Create(context.Background(), invalid); err == nil {
		t.Fatal("caller supplied Industry ID error = nil")
	}
	invalid = valid
	invalid.Aliases = []string{"芯片", "芯片"}
	if _, err := useCase.Create(context.Background(), invalid); err == nil {
		t.Fatal("duplicate alias error = nil")
	}
	invalid = valid
	invalid.HierarchyPathCodes = []string{"OTHER"}
	if _, err := useCase.Create(context.Background(), invalid); err == nil {
		t.Fatal("mismatched hierarchy path error = nil")
	}
}

func TestUpdateValidatesParentAndHierarchyBeforePersistence(t *testing.T) {
	store := &repositoryStub{}
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	parent := testParentIndustryID
	valid := Update{
		Name: "集成电路", Aliases: []string{}, ParentIndustryID: &parent,
		HierarchyPathCodes: []string{"SEMICONDUCTOR", "IC"}, Definition: "集成电路行业",
		ReviewStatus: ReviewStatusApproved,
	}
	if _, err := useCase.Update(context.Background(), testIndustryID, valid); err != nil {
		t.Fatal(err)
	}
	self := testIndustryID
	invalid := valid
	invalid.ParentIndustryID = &self
	if _, err := useCase.Update(context.Background(), testIndustryID, invalid); err == nil {
		t.Fatal("self parent error = nil")
	}
}
