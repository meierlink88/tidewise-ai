package postgres

import (
	"sync"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"
)

type InMemoryRepository struct {
	mu                  sync.Mutex
	documents           map[string]model.RawDocument
	entityTypes         map[string]model.EntityType
	observations        map[string]model.BenchmarkObservation
	physicalConstraints map[string]model.IndustryChainPhysicalConstraint
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		documents:           map[string]model.RawDocument{},
		entityTypes:         map[string]model.EntityType{},
		observations:        map[string]model.BenchmarkObservation{},
		physicalConstraints: map[string]model.IndustryChainPhysicalConstraint{},
	}
}

func (r *InMemoryRepository) SeedEntityType(id string, entityType model.EntityType) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entityTypes[id] = entityType
}
