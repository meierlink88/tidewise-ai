package researchgraph

import (
	"context"
	"time"
)

type Store interface {
	Search(context.Context, Query) (Subgraph, error)
}

type Query struct {
	AnalysisAsOf          time.Time
	SeedEntityIDs         []string
	RelationFilters       []RelationFilter
	MaxDepth              int
	IndustryChainEntityID *string
	NodeBudget            int
	EdgeBudget            int
	FactPolicy            FactEligibilityPolicy
}

type FactEligibilityPolicy struct {
	EntityStatus              string
	EntityRelationStatus      string
	IndustryChainReviewStatus string
	MembershipReviewStatus    string
	MembershipStatus          string
	GraphEdgeReviewStatus     string
	GraphEdgeStatus           string
}
