package source

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

func TestCreateDynamicSourceAppearsInTheCompleteActiveSnapshot(t *testing.T) {
	store := newMemoryStore()
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}

	created, err := useCase.CreateDynamic(context.Background(), MutableSource{
		Code:               "people-rss",
		Name:               "人民网 RSS",
		Enabled:            true,
		Endpoint:           "https://example.test/people.xml",
		Config:             []byte(`{"max_bytes":5000000}`),
		Priority:           2,
		TimeoutSeconds:     30,
		MaxResults:         10,
		DefaultSourceLevel: SourceLevelOfficial,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.Code != "people-rss" || created.OwnershipType != OwnershipDynamic ||
		created.ChannelType != ChannelRSS || created.AdapterKey != AdapterGenericRSS {
		t.Fatalf("created Source = %#v", created)
	}

	snapshot, err := useCase.ActiveSnapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot) != 1 || snapshot[0].ID != created.ID || string(snapshot[0].Config) != `{"max_bytes":5000000}` {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}

func TestCurrentFixedManifestPreservesAgentOSDefaults(t *testing.T) {
	manifest := CurrentFixedManifest(FixedManifestOptions{
		Endpoints: map[string]string{"bocha": "https://override.example/search"},
		AppKeys:   map[string]string{"bocha": "plain-key"},
	})
	if len(manifest) != 7 {
		t.Fatalf("manifest length = %d, want 7", len(manifest))
	}
	activeWeb := 0
	for _, item := range manifest {
		if item.OwnershipType != OwnershipFixed {
			t.Errorf("%s ownership = %q, want fixed", item.Code, item.OwnershipType)
		}
		if item.Enabled && item.ChannelType == ChannelWebSearch {
			activeWeb++
		}
	}
	if activeWeb != 1 {
		t.Fatalf("active web Sources = %d, want 1", activeWeb)
	}
	if manifest[0].Endpoint != "https://override.example/search" || manifest[0].AppKey == nil || *manifest[0].AppKey != "plain-key" {
		t.Fatalf("bocha deployment override not applied: %+v", manifest[0])
	}
}

func TestFixedSourceAllowsIncompatibleAdapterUpdateButCannotBeDeleted(t *testing.T) {
	store := newMemoryStore()
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	fixed := Source{
		Code: "bocha", Name: "博查", OwnershipType: OwnershipFixed, ChannelType: ChannelWebSearch,
		AdapterKey: AdapterBocha, Enabled: true, Endpoint: "https://api.bocha.test/search", Config: []byte(`{}`),
		Priority: 1, TimeoutSeconds: 30, MaxResults: 10, DefaultSourceLevel: SourceLevelMedia,
	}
	published, err := useCase.PublishFixed(context.Background(), []Source{fixed})
	if err != nil || len(published) != 1 {
		t.Fatalf("PublishFixed() = %#v, %v", published, err)
	}

	updated, err := useCase.Update(context.Background(), published[0].ID, MutableSource{
		Name: "博查", AdapterKey: AdapterGenericRSS, Enabled: true,
		Endpoint: "https://api.bocha.test/search", Config: []byte(`{}`), Priority: 1,
		TimeoutSeconds: 30, MaxResults: 10, DefaultSourceLevel: SourceLevelMedia,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ChannelType != ChannelWebSearch || updated.AdapterKey != AdapterGenericRSS {
		t.Fatalf("updated fixed Source = %#v", updated)
	}
	if err := useCase.Delete(context.Background(), updated.ID); !errors.Is(err, ErrFixedDeleteForbidden) {
		t.Fatalf("Delete(fixed) error = %v", err)
	}
}

func TestImportPreservesTimestampsAndRejectsDriftOnReplay(t *testing.T) {
	store := newMemoryStore()
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	createdAt := time.Date(2026, 8, 1, 1, 2, 3, 123_456_400, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	input := Source{
		Code: "imported-rss", Name: "Imported", OwnershipType: OwnershipDynamic, ChannelType: ChannelRSS,
		AdapterKey: AdapterGenericRSS, Enabled: true, Endpoint: "https://example.test/imported.xml",
		Config: []byte(`{"nested":{"z":1,"a":2},"max_bytes":5000000}`), Priority: 3, TimeoutSeconds: 40, MaxResults: 20,
		DefaultSourceLevel: SourceLevelWire, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}

	first, err := useCase.Import(context.Background(), []Source{input})
	if err != nil || len(first) != 1 || !first[0].CreatedAt.Equal(createdAt) || !first[0].UpdatedAt.Equal(updatedAt) {
		t.Fatalf("Import() = %#v, %v", first, err)
	}
	store.rows[0].Config = []byte(`{"max_bytes":5000000,"nested":{"a":2,"z":1}}`)
	store.rows[0].CreatedAt = store.rows[0].CreatedAt.Round(time.Microsecond)
	store.rows[0].UpdatedAt = store.rows[0].UpdatedAt.Round(time.Microsecond)
	second, err := useCase.Import(context.Background(), []Source{input})
	if err != nil || len(second) != 1 || second[0].ID != first[0].ID {
		t.Fatalf("Import(replay) = %#v, %v", second, err)
	}

	drift := input
	drift.Name = "Drifted"
	if _, err := useCase.Import(context.Background(), []Source{drift}); !errors.Is(err, ErrConflict) {
		t.Fatalf("Import(drift) error = %v", err)
	}
}

func TestActiveSnapshotReturnsTwoHundredSourcesInStableOrderWithinBudget(t *testing.T) {
	store := newMemoryStore()
	for index := MaxSources - 1; index >= 0; index-- {
		store.rows = append(store.rows, validTestSource(t, index, true))
	}
	useCase, err := NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	snapshot, err := useCase.ActiveSnapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed >= 3*time.Second {
		t.Fatalf("200-Source snapshot took %s", elapsed)
	}
	if len(snapshot) != MaxSources || snapshot[0].Code != "source-000" || snapshot[len(snapshot)-1].Code != "source-199" {
		t.Fatalf("snapshot order/count = %d, %q..%q", len(snapshot), snapshot[0].Code, snapshot[len(snapshot)-1].Code)
	}
	if size := snapshotEnvelopeSize(snapshot); size > MaxSnapshotEnvelopeSize {
		t.Fatalf("snapshot envelope size = %d", size)
	}
}

func TestSourceValidationFreezesConfigRSSLevelAndPriorityBounds(t *testing.T) {
	validConfig := []byte(`{"max_bytes":5000000,"source_levels":{"example.com":"L1_OFFICIAL"}}`)
	if err := validateConfig(validConfig); err != nil {
		t.Fatalf("valid config: %v", err)
	}
	for name, config := range map[string][]byte{
		"invalid source level": []byte(`{"source_levels":{"example.com":"PRIMARY"}}`),
		"RSS bytes too small":  []byte(`{"max_bytes":65535}`),
		"RSS bytes too large":  []byte(`{"max_bytes":10485761}`),
		"not an object":        []byte(`[]`),
		"over 4096 bytes":      []byte(`{"x":"` + strings.Repeat("x", 4089) + `"}`),
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateConfig(config); err == nil {
				t.Fatalf("validateConfig(%d bytes) accepted invalid config", len(config))
			}
		})
	}
	if err := validateConfig([]byte(`{"x":"` + strings.Repeat("x", 4088) + `"}`)); err != nil {
		t.Fatalf("exact 4096-byte config: %v", err)
	}

	item := validTestSource(t, 0, true)
	item.Priority = 6
	if err := validateSource(item, true); err == nil {
		t.Fatal("priority 6 was accepted")
	}
}

func TestProjectedSetEnforcesTotalCountAndCompleteEnvelopeBudget(t *testing.T) {
	items := make([]Source, MaxSources)
	for index := range items {
		items[index] = validTestSource(t, index, false)
	}
	if err := validateProjectedSet(items); err != nil {
		t.Fatalf("200 disabled Sources: %v", err)
	}
	items = append(items, validTestSource(t, MaxSources, false))
	if err := validateProjectedSet(items); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("201 Sources error = %v, want capacity", err)
	}

	large := make([]Source, 0, MaxSources)
	for index := 0; index < MaxSources; index++ {
		item := validTestSource(t, index, true)
		item.Config = []byte(`{"x":"` + strings.Repeat("x", 4088) + `"}`)
		large = append(large, item)
	}
	if err := validateProjectedSet(large); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("oversized active snapshot error = %v, want capacity", err)
	}
}

func validTestSource(t *testing.T, index int, enabled bool) Source {
	t.Helper()
	code := fmt.Sprintf("source-%03d", index)
	id, err := coreid.Derive(coreid.Source, "source", code)
	if err != nil {
		t.Fatal(err)
	}
	stamp := time.Date(2026, 8, 19, 0, 0, 0, 0, time.UTC)
	return Source{
		ID: id, Code: code, Name: code, OwnershipType: OwnershipDynamic, ChannelType: ChannelRSS,
		AdapterKey: AdapterGenericRSS, Enabled: enabled, Endpoint: "https://example.com/feed.xml",
		Config: []byte(`{}`), Priority: 1, TimeoutSeconds: 30, MaxResults: 10,
		DefaultSourceLevel: SourceLevelMedia, CreatedAt: stamp, UpdatedAt: stamp,
	}
}

type memoryStore struct{ rows []Source }

func newMemoryStore() *memoryStore { return &memoryStore{} }

func (s *memoryStore) List(context.Context, bool) ([]Source, error) {
	return cloneSources(s.rows), nil
}

func (s *memoryStore) InTransaction(_ context.Context, run func(Transaction) error) error {
	copy := &memoryTransaction{rows: cloneSources(s.rows)}
	if err := run(copy); err != nil {
		return err
	}
	s.rows = copy.rows
	return nil
}

type memoryTransaction struct{ rows []Source }

func (s *memoryTransaction) Lock(context.Context) error { return nil }

func (s *memoryTransaction) List(context.Context) ([]Source, error) {
	return cloneSources(s.rows), nil
}

func (s *memoryTransaction) Insert(_ context.Context, value Source) (Source, error) {
	if value.CreatedAt.IsZero() {
		stamp := time.Date(2026, 8, 19, 8, 0, 0, 0, time.UTC)
		value.CreatedAt, value.UpdatedAt = stamp, stamp
	}
	s.rows = append(s.rows, cloneSource(value))
	return cloneSource(value), nil
}

func (s *memoryTransaction) Update(_ context.Context, value Source) (Source, error) {
	for index := range s.rows {
		if s.rows[index].ID == value.ID {
			value.CreatedAt = s.rows[index].CreatedAt
			value.UpdatedAt = s.rows[index].UpdatedAt.Add(time.Second)
			s.rows[index] = cloneSource(value)
			return cloneSource(value), nil
		}
	}
	return Source{}, ErrNotFound
}

func (s *memoryTransaction) Delete(_ context.Context, id string) error {
	for index := range s.rows {
		if s.rows[index].ID == id {
			s.rows = append(s.rows[:index], s.rows[index+1:]...)
			return nil
		}
	}
	return ErrNotFound
}
