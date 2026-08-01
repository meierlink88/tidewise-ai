package semanticprojection

import (
	"context"
	"reflect"
	"testing"
)

type sourceStub struct{ snapshot Snapshot }

func (s sourceStub) Current(context.Context) (Snapshot, error) { return s.snapshot, nil }

type embedderStub struct{ documents [][]string }

func (s *embedderStub) Embed(_ context.Context, documents []string) ([][]float32, error) {
	s.documents = append(s.documents, append([]string(nil), documents...))
	result := make([][]float32, len(documents))
	for index := range result {
		result[index] = make([]float32, VectorSize)
	}
	return result, nil
}

type storeStub struct{ calls map[string][]Point }

func (s *storeStub) Replace(_ context.Context, collection string, vectorSize int, points []Point) error {
	if vectorSize != VectorSize {
		panic("unexpected vector size")
	}
	if s.calls == nil {
		s.calls = make(map[string][]Point)
	}
	s.calls[collection] = points
	return nil
}

func TestRebuildCreatesDeterministicSeparateEntityAndVariableCollections(t *testing.T) {
	source := sourceStub{snapshot: Snapshot{
		Entities: []EntitySource{{
			ID: "33333333-3333-4333-8333-333333333333", EntityType: "company",
			LayerCode: "micro",
			Name:      "NVIDIA", CanonicalName: "英伟达", Aliases: []string{"NVIDIA", "英伟达"}, Status: "active",
		}},
		Variables: []VariableSource{{
			Key: "capacity_commitment", Version: 1, NameZH: "产能承诺", NameEN: "Capacity commitment",
			BusinessDefinition: "主体对产能占用或供应作出的客观承诺", Domain: "operations",
			ApplicableEntityTypes: []string{"company"}, ValueType: "narrative",
			AllowedDirections: []string{"increase"}, Status: "active",
		}},
	}}
	embedder := &embedderStub{}
	store := &storeStub{}
	service, err := New(source, embedder, store)
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.Rebuild(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	firstEntity := store.calls[EntityCollection][0]
	firstVariable := store.calls[VariableDefinitionCollection][0]
	second, err := service.Rebuild(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if first != second || !reflect.DeepEqual(firstEntity, store.calls[EntityCollection][0]) ||
		!reflect.DeepEqual(firstVariable, store.calls[VariableDefinitionCollection][0]) {
		t.Fatalf("projection is not idempotent: first=%#v second=%#v", first, second)
	}
	if firstEntity.ID != "33333333-3333-4333-8333-333333333333" ||
		firstVariable.ID == "" || firstVariable.ID == firstEntity.ID {
		t.Fatalf("stable identities entity=%q variable=%q", firstEntity.ID, firstVariable.ID)
	}
	if got := firstEntity.Payload["normalized_names"]; !reflect.DeepEqual(got, []string{"33333333333343338333333333333333", "nvidia", "英伟达"}) {
		t.Fatalf("normalized_names = %#v", got)
	}
	if firstEntity.Payload["description"] != "formal entity type: company; layer: micro" {
		t.Fatalf("description = %#v", firstEntity.Payload["description"])
	}
	if first.EntityCount != 1 || first.VariableCount != 1 || len(embedder.documents) != 4 {
		t.Fatalf("result=%#v embed calls=%d", first, len(embedder.documents))
	}
}
