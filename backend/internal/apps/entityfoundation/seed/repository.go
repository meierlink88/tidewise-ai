package seed

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

type WriteAction string

const (
	WriteCreated   WriteAction = "created"
	WriteUpdated   WriteAction = "updated"
	WriteUnchanged WriteAction = "unchanged"
)

type WriteResult struct {
	Key    string
	Action WriteAction
}

type Repository interface {
	UpsertEntity(context.Context, Entity) (WriteResult, error)
	UpsertProfile(context.Context, Profile) (WriteResult, error)
	UpsertRelationship(context.Context, Relationship) (WriteResult, error)
}

type MemoryRepository struct {
	mu            sync.Mutex
	entities      map[string]Entity
	profiles      map[string]Profile
	relationships map[string]Relationship
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		entities:      map[string]Entity{},
		profiles:      map[string]Profile{},
		relationships: map[string]Relationship{},
	}
}

func (r *MemoryRepository) UpsertEntity(_ context.Context, entity Entity) (WriteResult, error) {
	if err := validateEntity(entity); err != nil {
		return WriteResult{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.entities[entity.Key]
	if !ok {
		r.entities[entity.Key] = entity
		return WriteResult{Key: entity.Key, Action: WriteCreated}, nil
	}
	if reflect.DeepEqual(existing, entity) {
		return WriteResult{Key: entity.Key, Action: WriteUnchanged}, nil
	}
	r.entities[entity.Key] = entity
	return WriteResult{Key: entity.Key, Action: WriteUpdated}, nil
}

func (r *MemoryRepository) UpsertProfile(_ context.Context, profile Profile) (WriteResult, error) {
	if profile.EntityKey == "" {
		return WriteResult{}, fmt.Errorf("profile entity key is required")
	}
	if err := validateProfileData(profile.EntityType, profile.Data); err != nil {
		return WriteResult{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	entity, ok := r.entities[profile.EntityKey]
	if !ok {
		return WriteResult{}, fmt.Errorf("unknown profile entity key %q", profile.EntityKey)
	}
	if profile.EntityType == "" {
		profile.EntityType = entity.EntityType
	}
	if profile.EntityType != entity.EntityType {
		return WriteResult{}, fmt.Errorf("profile entity type %q does not match entity %q type %q", profile.EntityType, profile.EntityKey, entity.EntityType)
	}

	existing, ok := r.profiles[profile.EntityKey]
	if !ok {
		r.profiles[profile.EntityKey] = profile
		return WriteResult{Key: profile.EntityKey, Action: WriteCreated}, nil
	}
	if reflect.DeepEqual(existing, profile) {
		return WriteResult{Key: profile.EntityKey, Action: WriteUnchanged}, nil
	}
	r.profiles[profile.EntityKey] = profile
	return WriteResult{Key: profile.EntityKey, Action: WriteUpdated}, nil
}

func (r *MemoryRepository) UpsertRelationship(_ context.Context, relationship Relationship) (WriteResult, error) {
	if relationship.Key == "" {
		return WriteResult{}, fmt.Errorf("relationship key is required")
	}
	if relationship.From == "" {
		return WriteResult{}, fmt.Errorf("relationship %q source is required", relationship.Key)
	}
	if relationship.To == "" {
		return WriteResult{}, fmt.Errorf("relationship %q target is required", relationship.Key)
	}
	if relationship.RelationType == "" {
		return WriteResult{}, fmt.Errorf("relationship %q relation type is required", relationship.Key)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.entities[relationship.From]; !ok {
		return WriteResult{}, fmt.Errorf("unknown relationship source %q", relationship.From)
	}
	if _, ok := r.entities[relationship.To]; !ok {
		return WriteResult{}, fmt.Errorf("unknown relationship target %q", relationship.To)
	}

	existing, ok := r.relationships[relationship.Key]
	if !ok {
		r.relationships[relationship.Key] = relationship
		return WriteResult{Key: relationship.Key, Action: WriteCreated}, nil
	}
	if reflect.DeepEqual(existing, relationship) {
		return WriteResult{Key: relationship.Key, Action: WriteUnchanged}, nil
	}
	r.relationships[relationship.Key] = relationship
	return WriteResult{Key: relationship.Key, Action: WriteUpdated}, nil
}

func (r *MemoryRepository) EntityCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.entities)
}
