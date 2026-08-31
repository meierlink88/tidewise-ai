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
	if created.Event.Semantic.Jurisdictions == nil {
		t.Fatal("empty jurisdictions must remain a JSON array, not null")
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

func TestPublishCreatesOnceAndReplaysTheSamePublicationWithoutAnotherWrite(t *testing.T) {
	store := new(fakeStore)
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	input := validCreateInput()
	first, err := useCase.Publish(context.Background(), "reasoning-server", "submission-1", input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := useCase.Publish(context.Background(), "reasoning-server", "submission-1", input)
	if err != nil {
		t.Fatal(err)
	}
	if first.Replayed || !second.Replayed || first.Event.ID != second.Event.ID {
		t.Fatalf("publication results = first %#v, second %#v", first, second)
	}
	if store.publishWrites != 1 {
		t.Fatalf("publication writes = %d, want 1", store.publishWrites)
	}
}

func TestPublishRejectsPublicationKeyReuseWithDifferentPayload(t *testing.T) {
	store := new(fakeStore)
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	input := validCreateInput()
	if _, err := useCase.Publish(context.Background(), "reasoning-server", "submission-1", input); err != nil {
		t.Fatal(err)
	}
	input.Summary = "A different occurrence payload."
	if _, err := useCase.Publish(context.Background(), "reasoning-server", "submission-1", input); !errors.Is(err, ErrPublicationPayloadConflict) {
		t.Fatalf("Publish() error = %v, want ErrPublicationPayloadConflict", err)
	}
	if store.publishWrites != 1 {
		t.Fatalf("publication writes = %d, want 1", store.publishWrites)
	}
}

func TestPublishRejectsAnUnknownEvidenceBeforeAnyWrite(t *testing.T) {
	store := &fakeStore{missingEvidence: true}
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}

	_, err = useCase.Publish(
		context.Background(),
		"reasoning-server",
		"submission-missing-evidence",
		validCreateInput(),
	)
	var reference *ReferenceError
	if !errors.As(err, &reference) {
		t.Fatalf("Publish() error = %T %v, want ReferenceError", err, err)
	}
	if store.publishWrites != 0 {
		t.Fatalf("publication writes = %d, want 0", store.publishWrites)
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
		{name: "invalid modality", mutate: func(input *CreateInput) { input.Semantic.Modality = "UNKNOWN" }},
		{name: "missing jurisdictions", mutate: func(input *CreateInput) { input.Semantic.Jurisdictions = nil }},
		{name: "missing metrics", mutate: func(input *CreateInput) { input.Semantic.Metrics = nil }},
		{name: "blank metric name", mutate: func(input *CreateInput) {
			value := "10"
			input.Semantic.Metrics = []Metric{{Name: " ", Value: &value}}
		}},
		{name: "metric without value or change", mutate: func(input *CreateInput) {
			input.Semantic.Metrics = []Metric{{Name: "capacity"}}
		}},
		{name: "missing time anchor", mutate: func(input *CreateInput) { input.Semantic.Time = EventTime{Precision: TimePrecisionUnknown} }},
		{name: "business and observed time together", mutate: func(input *CreateInput) {
			observedAt := time.Date(2026, 8, 29, 13, 46, 38, 0, time.UTC)
			input.Semantic.Time.ObservedAt = &observedAt
		}},
		{name: "duplicate semantic actor", mutate: func(input *CreateInput) {
			input.Semantic.Actors = append(input.Semantic.Actors, input.Semantic.Actors[0])
		}},
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

func TestCreateAcceptsObservedOnlyEventTime(t *testing.T) {
	store := new(fakeStore)
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	input := validCreateInput()
	observedAt := time.Date(2026, 8, 29, 13, 46, 38, 0, time.UTC)
	input.Semantic.Time = EventTime{ObservedAt: &observedAt, Precision: TimePrecisionInstant}

	if _, err := useCase.Create(context.Background(), input); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if store.createCalls != 1 || store.aggregate.Event.Semantic.Time.ObservedAt == nil ||
		!store.aggregate.Event.Semantic.Time.ObservedAt.Equal(observedAt) {
		t.Fatalf("stored observed time = %#v, create calls = %d", store.aggregate.Event.Semantic.Time, store.createCalls)
	}
}

func validCreateInput() CreateInput {
	return CreateInput{
		Title: "Example Event", Summary: "Example Event summary.",
		Semantic: Semantic{Actors: []string{"Example actor"}, Action: "announces", Objects: []string{"Example object"},
			Stage: EventStageAnnounced, Modality: ModalityFact, Jurisdictions: []string{},
			Time:    EventTime{AnnouncedAt: timePointer(time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)), Precision: TimePrecisionDay},
			Metrics: []Metric{}},
		Evidence: []EvidenceLinkInput{{EvidenceID: "EVD11111111-1111-4111-8111-111111111111", ContributionWeight: 0.8}},
		Actors: []ActorLinkInput{{
			ActorID: "actor:1", ActorType: ActorTypeCompany, RelationType: ActorRelationMentions,
		}},
		Assets: []AssetLinkInput{{
			AssetID: "asset:1", AssetType: AssetTypeSecurity, ImpactDirection: ImpactDirectionPositive,
		}},
	}
}

func timePointer(value time.Time) *time.Time { return &value }

type fakeStore struct {
	aggregate       Aggregate
	createCalls     int
	createErr       error
	receipts        map[string]PublicationReceipt
	publishWrites   int
	missingEvidence bool
}

func (s *fakeStore) InEventPublicationTransaction(
	ctx context.Context,
	fn func(PublicationTransaction) error,
) error {
	if s.receipts == nil {
		s.receipts = make(map[string]PublicationReceipt)
	}
	return fn((*fakePublicationTransaction)(s))
}

type fakePublicationTransaction fakeStore

func (*fakePublicationTransaction) Lock(context.Context, string) error { return nil }

func (t *fakePublicationTransaction) Receipt(
	_ context.Context,
	publisher string,
	key string,
) (*PublicationReceipt, error) {
	receipt, ok := t.receipts[publisher+"\x00"+key]
	if !ok {
		return nil, nil
	}
	return &receipt, nil
}

func (t *fakePublicationTransaction) ExistingEvidenceIDs(
	_ context.Context,
	ids []string,
) ([]string, error) {
	if t.missingEvidence {
		return nil, nil
	}
	return append([]string(nil), ids...), nil
}

func (t *fakePublicationTransaction) InsertAggregate(
	_ context.Context,
	aggregate Aggregate,
) error {
	t.publishWrites++
	t.aggregate = aggregate
	return nil
}

func (t *fakePublicationTransaction) InsertReceipt(
	_ context.Context,
	receipt PublicationReceipt,
) error {
	t.receipts[receipt.PublisherSubject+"\x00"+receipt.PublicationKey] = receipt
	return nil
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
