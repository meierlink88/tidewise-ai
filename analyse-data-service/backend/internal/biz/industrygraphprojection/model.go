package industrygraphprojection

const (
	ContractVersion = "industry-graph-projection-v1"
	Namespace       = "tidewise-industry-v1"
)

type EntityType string

const (
	EntityTypeIndustry      EntityType = "industry"
	EntityTypeConcept       EntityType = "concept"
	EntityTypeIndustryChain EntityType = "industry_chain"
	EntityTypeChainNode     EntityType = "chain_node"
)

type RelationshipType string

const (
	RelationshipTypeMappedToIndustry RelationshipType = "MAPPED_TO_INDUSTRY"
	RelationshipTypeMappedToConcept  RelationshipType = "MAPPED_TO_CONCEPT"
	RelationshipTypeHasNode          RelationshipType = "HAS_NODE"
	RelationshipTypeInputTo          RelationshipType = "INPUT_TO"
	RelationshipTypeIsComponentOf    RelationshipType = "IS_COMPONENT_OF"
	RelationshipTypeDependsOn        RelationshipType = "DEPENDS_ON"
	RelationshipTypeIsSubcategoryOf  RelationshipType = "IS_SUBCATEGORY_OF"
)

type Node struct {
	EntityID      string     `json:"entity_id"`
	EntityKey     string     `json:"entity_key"`
	EntityType    EntityType `json:"entity_type"`
	CanonicalName string     `json:"canonical_name"`
	Aliases       []string   `json:"aliases"`
}

type Relationship struct {
	FromEntityID    string           `json:"from_entity_id"`
	ToEntityID      string           `json:"to_entity_id"`
	Type            RelationshipType `json:"type"`
	ChainID         string           `json:"chain_id,omitempty"`
	RelationKey     string           `json:"relation_key"`
	ContextualStage string           `json:"contextual_stage,omitempty"`
	Position        *int             `json:"position,omitempty"`
	Mechanism       string           `json:"mechanism"`
}

type Projection struct {
	PackageSHA256 string         `json:"package_sha256"`
	Nodes         []Node         `json:"nodes"`
	Relationships []Relationship `json:"relationships"`
}

type ProjectionState struct {
	Projection              Projection
	ContractVersion         string
	PackageSHA256           string
	IntegrityViolationCount int
}

type ProjectRequest struct {
	Baseline Projection
	Apply    bool
}

type Result struct {
	Namespace                      string            `json:"namespace"`
	ContractVersion                string            `json:"contract_version"`
	PackageSHA256                  string            `json:"package_sha256"`
	NodeCount                      int               `json:"node_count"`
	RelationshipCount              int               `json:"relationship_count"`
	Source                         ProjectionSummary `json:"source"`
	CurrentNeo4j                   ProjectionSummary `json:"current_neo4j"`
	FinalNeo4j                     ProjectionSummary `json:"final_neo4j"`
	CurrentIntegrityViolationCount int               `json:"current_integrity_violation_count"`
	FinalIntegrityViolationCount   int               `json:"final_integrity_violation_count"`
	DryRun                         bool              `json:"dry_run"`
	Applied                        bool              `json:"applied"`
	Unchanged                      bool              `json:"unchanged"`
}
