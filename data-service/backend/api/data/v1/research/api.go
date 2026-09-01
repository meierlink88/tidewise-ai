package research

import (
	"context"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const OperationSearchResearchGraph = "data.v1.searchResearchGraph"

type Service interface {
	SearchResearchGraph(context.Context, *ResearchGraphSearchRequest) (*v1.Response[ResearchGraphSearchResult], error)
}

func BusinessOperations() []string {
	return []string{OperationSearchResearchGraph}
}

type ResearchResourceLimitDetails struct {
	Component     string `json:"component"`
	ActualRows    *int64 `json:"actual_rows,omitempty"`
	MaxRows       *int64 `json:"max_rows,omitempty"`
	ActualBytes   *int64 `json:"actual_bytes,omitempty"`
	MaxBytes      *int64 `json:"max_bytes,omitempty"`
	RetryGuidance string `json:"retry_guidance"`
}

type ResearchGraphEntity struct {
	EntityID      string   `json:"entity_id"`
	EntityType    string   `json:"entity_type"`
	Name          string   `json:"name"`
	CanonicalName string   `json:"canonical_name"`
	Aliases       []string `json:"aliases"`
	Status        string   `json:"status"`
}

type ResearchGraphEntityRelation struct {
	EntityRelationID string `json:"entity_relation_id"`
	FromEntityID     string `json:"from_entity_id"`
	ToEntityID       string `json:"to_entity_id"`
	RelationType     string `json:"relation_type"`
	Status           string `json:"status"`
}

type ResearchGraphRelationDefinition struct {
	RelationType string `json:"relation_type"`
	Direction    string `json:"direction"`
}

type ResearchGraphIndustryChain struct {
	IndustryChainID string `json:"industry_chain_id"`
	Scope           string `json:"scope"`
	TargetOutput    string `json:"target_output"`
	EndUse          string `json:"end_use"`
	Geography       string `json:"geography"`
	AsOfDate        string `json:"as_of_date"`
	ReviewStatus    string `json:"review_status"`
}

type ResearchGraphIndustryChainMembership struct {
	IndustryChainID string `json:"industry_chain_id"`
	ChainNodeID     string `json:"chain_node_id"`
	Position        int    `json:"position"`
	ContextualStage string `json:"contextual_stage"`
}

type ResearchGraphIndustryChainGraphEdge struct {
	IndustryChainGraphEdgeID string `json:"industry_chain_graph_edge_id"`
	IndustryChainID          string `json:"industry_chain_id"`
	FromChainNodeID          string `json:"from_chain_node_id"`
	ToChainNodeID            string `json:"to_chain_node_id"`
	RelationType             string `json:"relation_type"`
}

type ResearchGraphSearchRequest struct {
	AnalysisAsOf    string                        `json:"analysis_as_of"`
	SeedEntityIDs   []string                      `json:"seed_entity_ids"`
	RelationFilters []ResearchGraphRelationFilter `json:"relation_filters"`
	MaxDepth        int                           `json:"max_depth"`
	IndustryChainID *string                       `json:"industry_chain_id,omitempty"`
	NodeBudget      int                           `json:"node_budget"`
	EdgeBudget      int                           `json:"edge_budget"`
}

type ResearchGraphRelationFilter struct {
	RelationType string `json:"relation_type"`
	Direction    string `json:"direction"`
}

type ResearchGraphSearchResult struct {
	ContractVersion          string                                 `json:"contract_version"`
	AnalysisAsOf             string                                 `json:"analysis_as_of"`
	QueryFingerprint         string                                 `json:"query_fingerprint"`
	GraphFingerprint         string                                 `json:"graph_fingerprint"`
	ActualDepth              int                                    `json:"actual_depth"`
	Entities                 []ResearchGraphEntity                  `json:"entities"`
	RelationDefinitions      []ResearchGraphRelationDefinition      `json:"relation_definitions"`
	EntityRelations          []ResearchGraphEntityRelation          `json:"entity_relations"`
	IndustryChains           []ResearchGraphIndustryChain           `json:"industry_chains"`
	IndustryChainMemberships []ResearchGraphIndustryChainMembership `json:"industry_chain_memberships"`
	IndustryChainGraphEdges  []ResearchGraphIndustryChainGraphEdge  `json:"industry_chain_graph_edges"`
}
