package v1

type ResearchGraphSearchRequest struct {
	AnalysisAsOf          string                        `json:"analysis_as_of"`
	SeedEntityIDs         []string                      `json:"seed_entity_ids"`
	RelationFilters       []ResearchGraphRelationFilter `json:"relation_filters"`
	MaxDepth              int                           `json:"max_depth"`
	IndustryChainEntityID *string                       `json:"industry_chain_entity_id,omitempty"`
	NodeBudget            int                           `json:"node_budget"`
	EdgeBudget            int                           `json:"edge_budget"`
}

type ResearchGraphRelationFilter struct {
	RelationType string `json:"relation_type"`
	Direction    string `json:"direction"`
}

type ResearchGraphSearchResult struct {
	ContractVersion          string                                    `json:"contract_version"`
	AnalysisAsOf             string                                    `json:"analysis_as_of"`
	QueryFingerprint         string                                    `json:"query_fingerprint"`
	GraphFingerprint         string                                    `json:"graph_fingerprint"`
	ActualDepth              int                                       `json:"actual_depth"`
	Entities                 []EventSemanticEntity                     `json:"entities"`
	RelationDefinitions      []ResearchAnalysisRelationDefinition      `json:"relation_definitions"`
	EntityRelations          []EventSemanticEntityRelation             `json:"entity_relations"`
	IndustryChains           []ResearchAnalysisIndustryChain           `json:"industry_chains"`
	IndustryChainMemberships []ResearchAnalysisIndustryChainMembership `json:"industry_chain_memberships"`
	IndustryChainGraphEdges  []ResearchAnalysisIndustryChainGraphEdge  `json:"industry_chain_graph_edges"`
}
