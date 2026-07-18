package repositories

import (
	"sync"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type InMemoryRepository struct {
	mu                  sync.Mutex
	sources             []domain.SourceCatalog
	documents           map[string]domain.RawDocument
	events              map[string]domain.Event
	eventSources        map[string]domain.EventSource
	eventTagDefs        map[string]domain.EventTagDef
	eventTagMaps        map[string]domain.EventTagMap
	graphEntities       map[string]GraphEntityNode
	graphEdges          map[string]GraphEntityEdge
	graphRuns           map[string]GraphProjectionRun
	graphRunItems       map[string][]GraphProjectionRunItem
	observations        map[string]domain.BenchmarkObservation
	physicalConstraints map[string]domain.IndustryChainPhysicalConstraint
}

func NewInMemoryRepository(sources []domain.SourceCatalog) *InMemoryRepository {
	copiedSources := make([]domain.SourceCatalog, len(sources))
	for index, source := range sources {
		copiedSources[index] = cloneSource(normalizeInMemorySource(source))
	}

	return &InMemoryRepository{
		sources:             copiedSources,
		documents:           map[string]domain.RawDocument{},
		events:              map[string]domain.Event{},
		eventSources:        map[string]domain.EventSource{},
		eventTagDefs:        map[string]domain.EventTagDef{},
		eventTagMaps:        map[string]domain.EventTagMap{},
		graphEntities:       map[string]GraphEntityNode{},
		graphEdges:          map[string]GraphEntityEdge{},
		graphRuns:           map[string]GraphProjectionRun{},
		graphRunItems:       map[string][]GraphProjectionRunItem{},
		observations:        map[string]domain.BenchmarkObservation{},
		physicalConstraints: map[string]domain.IndustryChainPhysicalConstraint{},
	}
}
