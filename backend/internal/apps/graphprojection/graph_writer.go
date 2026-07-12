package graphprojection

import (
	"context"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type GraphNode struct {
	EntityID           string
	EntityKey          string
	EntityType         string
	LayerCode          string
	Name               string
	CanonicalName      string
	Aliases            []string
	ClassificationCode domain.SectorClassification
	Status             string
	Namespace          string
	UpdatedAt          time.Time
}

type GraphRelationship struct {
	EdgeID               string
	FromEntityID         string
	ToEntityID           string
	RelationshipType     string
	OriginalRelationType string
	Source               string
	Confidence           float64
	Status               string
	Namespace            string
	UpdatedAt            time.Time
}

type GraphWriteResult struct {
	Projected int
}

type GraphWriter interface {
	UpsertEntities(context.Context, []GraphNode) (GraphWriteResult, error)
	UpsertRelationships(context.Context, []GraphRelationship) (GraphWriteResult, error)
	DeleteNamespace(context.Context, string) error
}
