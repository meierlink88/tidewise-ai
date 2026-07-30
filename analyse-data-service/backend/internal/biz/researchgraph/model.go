package researchgraph

type Entity struct {
	EntityID      string   `json:"entity_id"`
	EntityType    string   `json:"entity_type"`
	Name          string   `json:"name"`
	CanonicalName string   `json:"canonical_name"`
	Aliases       []string `json:"aliases"`
	Status        string   `json:"status"`
}

type RelationDefinition struct {
	RelationType string `json:"relation_type"`
	Direction    string `json:"direction"`
}

type EntityRelation struct {
	EntityRelationID string `json:"entity_relation_id"`
	FromEntityID     string `json:"from_entity_id"`
	ToEntityID       string `json:"to_entity_id"`
	RelationType     string `json:"relation_type"`
	Status           string `json:"status"`
}

type IndustryChain struct {
	IndustryChainEntityID string `json:"industry_chain_entity_id"`
	Scope                 string `json:"scope"`
	TargetOutput          string `json:"target_output"`
	EndUse                string `json:"end_use"`
	Geography             string `json:"geography"`
	AsOfDate              string `json:"as_of_date"`
	ReviewStatus          string `json:"review_status"`
}

type IndustryChainMembership struct {
	IndustryChainEntityID string `json:"industry_chain_entity_id"`
	ChainNodeEntityID     string `json:"chain_node_entity_id"`
	Position              int    `json:"position"`
	ContextualStage       string `json:"contextual_stage"`
	ReviewStatus          string `json:"review_status"`
	Status                string `json:"status"`
}

type IndustryChainGraphEdge struct {
	IndustryChainGraphEdgeID string  `json:"industry_chain_graph_edge_id"`
	IndustryChainEntityID    string  `json:"industry_chain_entity_id"`
	FromChainNodeEntityID    string  `json:"from_chain_node_entity_id"`
	ToChainNodeEntityID      string  `json:"to_chain_node_entity_id"`
	RelationType             string  `json:"relation_type"`
	Mechanism                string  `json:"mechanism"`
	ConditionNote            *string `json:"condition_note"`
	SegmentKind              string  `json:"segment_kind"`
	OmittedStepNote          *string `json:"omitted_step_note"`
	ReviewStatus             string  `json:"review_status"`
	Status                   string  `json:"status"`
}

type Subgraph struct {
	ActualDepth              int                       `json:"actual_depth"`
	Entities                 []Entity                  `json:"entities"`
	RelationDefinitions      []RelationDefinition      `json:"relation_definitions"`
	EntityRelations          []EntityRelation          `json:"entity_relations"`
	IndustryChains           []IndustryChain           `json:"industry_chains"`
	IndustryChainMemberships []IndustryChainMembership `json:"industry_chain_memberships"`
	IndustryChainGraphEdges  []IndustryChainGraphEdge  `json:"industry_chain_graph_edges"`
}
