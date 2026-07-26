package entityseed

import (
	"fmt"
	"strings"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"
)

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

type FrozenChainNodeRelationBaseline struct {
	ExistingRelations    int
	SubcategoryRelations int
	ComponentRelations   int
	InputRelations       int
	DependsRelations     int
}

type FrozenChainNodeRelationAggregate struct {
	Total       int
	Subcategory int
	Component   int
	Input       int
	Depends     int
	Incomplete  int
	SelfLoops   int
	Duplicates  int
	Orphans     int
}

type ChainNodeRelationDataPreflightReport struct {
	DatabaseName         string `json:"database_name"`
	ServerVersion        string `json:"server_version"`
	GooseVersion         int    `json:"goose_version"`
	ActiveChainNodes     int    `json:"active_chain_nodes"`
	ChainNodeProfiles    int    `json:"chain_node_profiles"`
	ExternalIdentifiers  int    `json:"external_identifiers"`
	EntityEdges          int    `json:"entity_edges"`
	ExistingRelations    int    `json:"existing_relations"`
	SubcategoryRelations int    `json:"subcategory_relations"`
	ComponentRelations   int    `json:"component_relations"`
	InputRelations       int    `json:"input_relations"`
	DependsRelations     int    `json:"depends_relations"`
	ExistingConstraints  int    `json:"existing_constraints"`
	SchemaValid          bool   `json:"schema_valid"`
}

func ValidateChainNodeRelationDataPreflight(report ChainNodeRelationDataPreflightReport, expectedRelations int) error {
	if report.DatabaseName != "tidewise_local" ||
		!strings.HasPrefix(report.ServerVersion, "16.14") ||
		report.GooseVersion != 18 ||
		report.ActiveChainNodes != 842 ||
		report.ChainNodeProfiles != 842 ||
		report.ExternalIdentifiers != 1169 ||
		report.EntityEdges != 241 ||
		report.ExistingRelations != expectedRelations ||
		report.ExistingConstraints != 0 ||
		!report.SchemaValid {
		return fmt.Errorf("relation data preflight baseline mismatch: %+v", report)
	}
	return nil
}
