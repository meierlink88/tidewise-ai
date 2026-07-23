package repositories

import (
	"sync"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

type InMemoryRepository struct {
	mu                  sync.Mutex
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

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
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
