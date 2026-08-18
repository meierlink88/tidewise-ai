package event

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

func TestCreateBuildsOwnedAggregateIdentitiesAndDefaults(t *testing.T) {
	store := new(fakeStore)
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	created, err := useCase.Create(context.Background(), validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	if !coreid.Is(created.Event.ID, coreid.Event) || created.Event.Status != LifecycleStatusActive {
		t.Fatalf("created Event = %#v", created.Event)
	}
	if len(created.Evidence) != 1 || !coreid.Is(created.Evidence[0].ID, coreid.EventEvidenceLink) ||
		created.Evidence[0].EventID != created.Event.ID {
		t.Fatalf("created Evidence Links = %#v", created.Evidence)
	}
	if len(created.Actors) != 1 || !coreid.Is(created.Actors[0].ID, coreid.EventActorLink) || created.Actors[0].Confidence != 0.70 {
		t.Fatalf("created Actor Links = %#v", created.Actors)
	}
	if len(created.Assets) != 1 || !coreid.Is(created.Assets[0].ID, coreid.EventAssetLink) {
		t.Fatalf("created Asset Links = %#v", created.Assets)
	}
}

func TestCreateRejectsInvalidAggregateBeforePersistence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*CreateInput)
	}{
		{name: "missing evidence", mutate: func(input *CreateInput) { input.Evidence = nil }},
		{name: "unknown evidence identity", mutate: func(input *CreateInput) { input.Evidence[0].EvidenceID = "bad" }},
		{name: "duplicate evidence", mutate: func(input *CreateInput) { input.Evidence = append(input.Evidence, input.Evidence[0]) }},
		{name: "invalid modality", mutate: func(input *CreateInput) { input.Modality = "UNKNOWN" }},
		{name: "invalid weight", mutate: func(input *CreateInput) { input.Evidence[0].ContributionWeight = 1.01 }},
		{name: "nan actor strength", mutate: func(input *CreateInput) { value := math.NaN(); input.Actors[0].RelationStrength = &value }},
		{name: "invalid actor confidence", mutate: func(input *CreateInput) { value := 1.0; input.Actors[0].Confidence = &value }},
		{name: "duplicate actor relation", mutate: func(input *CreateInput) { input.Actors = append(input.Actors, input.Actors[0]) }},
		{name: "invalid asset type", mutate: func(input *CreateInput) { input.Assets[0].AssetType = "UNKNOWN" }},
		{name: "duplicate asset", mutate: func(input *CreateInput) { input.Assets = append(input.Assets, input.Assets[0]) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := new(fakeStore)
			useCase, err := NewUseCase(store)
			if err != nil {
				t.Fatal(err)
			}
			input := validCreateInput()
			test.mutate(&input)
			if _, err := useCase.Create(context.Background(), input); err == nil {
				t.Fatal("Create() error = nil")
			}
			if store.createCalls != 0 {
				t.Fatalf("CreateEvent calls = %d, want 0", store.createCalls)
			}
		})
	}
}

func validCreateInput() CreateInput {
	return CreateInput{
		Title: "Example Event", Summary: "Example Event summary.",
		Semantic: Semantic{}, Modality: ModalityFact,
		Evidence: []EvidenceLinkInput{{EvidenceID: "EVD11111111-1111-4111-8111-111111111111", ContributionWeight: 0.8}},
		Actors: []ActorLinkInput{{
			ActorID: "actor:1", ActorType: ActorTypeCompany, RelationType: ActorRelationMentions,
		}},
		Assets: []AssetLinkInput{{
			AssetID: "asset:1", AssetType: AssetTypeSecurity, ImpactDirection: ImpactDirectionPositive,
		}},
	}
}

type fakeStore struct {
	aggregate   Aggregate
	createCalls int
	createErr   error
}

func (s *fakeStore) CreateEvent(_ context.Context, aggregate Aggregate) error {
	s.createCalls++
	if s.createErr != nil {
		return s.createErr
	}
	now := time.Date(2026, 8, 18, 1, 0, 0, 0, time.UTC)
	for index := range aggregate.Actors {
		aggregate.Actors[index].CreatedAt = now
		aggregate.Actors[index].UpdatedAt = now
	}
	s.aggregate = aggregate
	return nil
}

func (s *fakeStore) EventByID(context.Context, string) (Aggregate, error) {
	if s.aggregate.Event.ID == "" {
		return Aggregate{}, ErrEventNotFound
	}
	return s.aggregate, nil
}

func (s *fakeStore) ListEvents(context.Context, EventListFilter) (EventStorePage, error) {
	return EventStorePage{}, errors.New("not implemented")
}
