package entityseed

import "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"

type ChainNodeRelationManifest struct {
	Relations           []model.ChainNodeRelation           `json:"relations"`
	PhysicalConstraints []model.ChainNodePhysicalConstraint `json:"physical_constraints,omitempty"`
}

type ChainNodeRelationReport struct {
	Created        int                                 `json:"created"`
	Updated        int                                 `json:"updated"`
	Unchanged      int                                 `json:"unchanged"`
	ByRelationType map[model.ChainNodeRelationType]int `json:"by_relation_type"`
}
