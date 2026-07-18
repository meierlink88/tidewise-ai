package repositories

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type EventWriteRepository interface {
	UpsertEvent(context.Context, domain.Event) (EventWriteResult, error)
	AddEventSource(context.Context, domain.EventSource) (EventSourceWriteResult, error)
	AssignEventTag(context.Context, domain.EventTagMap) (EventTagMapWriteResult, error)
}

type EventWriteResult struct {
	Event       domain.Event
	Created     bool
	DuplicateOf string
}

type EventSourceWriteResult struct {
	Source      domain.EventSource
	Created     bool
	DuplicateOf string
}

type EventTagMapWriteResult struct {
	TagMap      domain.EventTagMap
	Created     bool
	DuplicateOf string
}

func (r *InMemoryRepository) UpsertEvent(_ context.Context, event domain.Event) (EventWriteResult, error) {
	if err := event.Validate(); err != nil {
		return EventWriteResult{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.events {
		if existing.DedupeKey == event.DedupeKey {
			return EventWriteResult{Event: cloneEvent(existing), DuplicateOf: existing.ID}, nil
		}
	}
	r.events[event.ID] = cloneEvent(event)
	return EventWriteResult{Event: cloneEvent(event), Created: true}, nil
}

func (r *InMemoryRepository) AddEventSource(_ context.Context, source domain.EventSource) (EventSourceWriteResult, error) {
	if err := source.Validate(); err != nil {
		return EventSourceWriteResult{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.events[source.EventID]; !exists {
		return EventSourceWriteResult{}, fmt.Errorf("event %q not found", source.EventID)
	}
	for _, existing := range r.eventSources {
		if existing.EventID == source.EventID && existing.RawDocumentID == source.RawDocumentID && existing.EvidenceHash == source.EvidenceHash {
			return EventSourceWriteResult{Source: cloneEventSource(existing), DuplicateOf: existing.ID}, nil
		}
	}
	r.eventSources[source.ID] = cloneEventSource(source)
	return EventSourceWriteResult{Source: cloneEventSource(source), Created: true}, nil
}

func (r *InMemoryRepository) SeedEventTagDef(definition domain.EventTagDef) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.eventTagDefs[definition.ID] = definition
}

func (r *InMemoryRepository) AssignEventTag(_ context.Context, tagMap domain.EventTagMap) (EventTagMapWriteResult, error) {
	if err := tagMap.Validate(); err != nil {
		return EventTagMapWriteResult{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.events[tagMap.EventID]; !exists {
		return EventTagMapWriteResult{}, fmt.Errorf("event %q not found", tagMap.EventID)
	}
	if _, exists := r.eventTagDefs[tagMap.TagID]; !exists {
		return EventTagMapWriteResult{}, fmt.Errorf("tag definition %q not found in Tidewise DB", tagMap.TagID)
	}
	for _, existing := range r.eventTagMaps {
		if existing.EventID == tagMap.EventID && existing.TagID == tagMap.TagID {
			return EventTagMapWriteResult{TagMap: cloneEventTagMap(existing), DuplicateOf: existing.ID}, nil
		}
	}
	r.eventTagMaps[tagMap.ID] = cloneEventTagMap(tagMap)
	return EventTagMapWriteResult{TagMap: cloneEventTagMap(tagMap), Created: true}, nil
}

func cloneFactPayload(payload domain.FactPayload) domain.FactPayload {
	if payload == nil {
		return nil
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		panic(fmt.Sprintf("clone validated fact payload: %v", err))
	}
	var cloned domain.FactPayload
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		panic(fmt.Sprintf("decode validated fact payload: %v", err))
	}
	return cloned
}

func cloneEventSource(source domain.EventSource) domain.EventSource {
	source.SupportsFields = append([]string(nil), source.SupportsFields...)
	return source
}

func cloneEventTagMap(tagMap domain.EventTagMap) domain.EventTagMap {
	if tagMap.Confidence != nil {
		confidence := *tagMap.Confidence
		tagMap.Confidence = &confidence
	}
	return tagMap
}
