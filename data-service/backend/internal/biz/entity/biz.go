package entity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

var ErrResearchHistoricalReferencesUnavailable = errors.New("strict historical Entity references are unavailable because selected facts changed after analysis_as_of")

const (
	ObjectTypeCountry       = "country"
	ObjectTypeRegion        = "region"
	ObjectTypeOrganization  = "organization"
	ObjectTypeIndustry      = "industry"
	ObjectTypeConcept       = "concept"
	ObjectTypeChainNode     = "chain_node"
	ObjectTypeIndustryChain = "industry_chain"
	EntityRelationIDPrefix  = coreid.EntityRelation
	CountryIDPrefix         = coreid.Country
	RegionIDPrefix          = coreid.Region
	OrganizationIDPrefix    = coreid.Organization
)

func IsEntityRelationID(value string) bool { return coreid.Is(value, EntityRelationIDPrefix) }
func IsCountryID(value string) bool        { return coreid.Is(value, CountryIDPrefix) }
func IsRegionID(value string) bool         { return coreid.Is(value, RegionIDPrefix) }
func IsOrganizationID(value string) bool   { return coreid.Is(value, OrganizationIDPrefix) }
func IsIndustryID(value string) bool       { return coreid.Is(value, coreid.Industry) }
func IsConceptID(value string) bool        { return coreid.Is(value, coreid.Concept) }
func IsChainNodeID(value string) bool      { return coreid.Is(value, coreid.ChainNode) }
func IsIndustryChainID(value string) bool  { return coreid.Is(value, coreid.IndustryChain) }

func IsObjectID(value string) bool {
	return IsCountryID(value) || IsRegionID(value) || IsOrganizationID(value) ||
		IsIndustryID(value) || IsConceptID(value) || IsChainNodeID(value) || IsIndustryChainID(value)
}

func ObjectTypeMatchesID(objectType, value string) bool {
	switch objectType {
	case ObjectTypeCountry:
		return IsCountryID(value)
	case ObjectTypeRegion:
		return IsRegionID(value)
	case ObjectTypeOrganization:
		return IsOrganizationID(value)
	case ObjectTypeIndustry:
		return IsIndustryID(value)
	case ObjectTypeConcept:
		return IsConceptID(value)
	case ObjectTypeChainNode:
		return IsChainNodeID(value)
	case ObjectTypeIndustryChain:
		return IsIndustryChainID(value)
	default:
		return false
	}
}

// ResearchGraphRepository exposes persisted Entity graph facts to the Entity domain.
type ResearchGraphRepository interface {
	SearchResearchGraph(context.Context, ResearchGraphQuery) (ResearchGraphSubgraph, error)
	ResearchReferenceClosure(context.Context, ResearchReferenceQuery) (ResearchReferenceDictionaries, error)
}

// UseCase owns Entity-domain behavior used by other Data Service domains.
type UseCase struct{ graph ResearchGraphRepository }

func NewUseCase(graph ResearchGraphRepository) (*UseCase, error) {
	if graph == nil {
		return nil, fmt.Errorf("Entity graph repository is required")
	}
	return &UseCase{graph: graph}, nil
}

// SearchResearchGraph exposes bounded Entity graph facts to the Research domain.
func (s *UseCase) SearchResearchGraph(ctx context.Context, query ResearchGraphQuery) (ResearchGraphSubgraph, error) {
	if s == nil || s.graph == nil {
		return ResearchGraphSubgraph{}, fmt.Errorf("Entity graph repository is required")
	}
	return s.graph.SearchResearchGraph(ctx, query)
}

func (s *UseCase) ResearchReferenceClosure(ctx context.Context, query ResearchReferenceQuery) (ResearchReferenceDictionaries, error) {
	if s == nil || s.graph == nil {
		return ResearchReferenceDictionaries{}, fmt.Errorf("Entity repository is required")
	}
	return s.graph.ResearchReferenceClosure(ctx, query)
}

type ResearchReferenceQuery struct {
	AnalysisAsOf      time.Time
	EntityIDs         []string
	EntityRelationIDs []string
	RelationTypes     []string
}

type ResearchReferenceDictionaries struct {
	Entities                 []ResearchGraphEntity         `json:"entities"`
	RelationDefinitions      []ResearchGraphRelation       `json:"relation_definitions"`
	EntityRelations          []ResearchGraphEntityRelation `json:"entity_relations"`
	IndustryChains           []ResearchGraphIndustryChain  `json:"industry_chains"`
	IndustryChainMemberships []ResearchGraphMembership     `json:"industry_chain_memberships"`
	IndustryChainGraphEdges  []ResearchGraphIndustryEdge   `json:"industry_chain_graph_edges"`
}

type ResearchGraphDirection string

const (
	ResearchGraphDirectionOutgoing ResearchGraphDirection = "outgoing"
	ResearchGraphDirectionIncoming ResearchGraphDirection = "incoming"
	ResearchGraphDirectionBoth     ResearchGraphDirection = "both"
)

type ResearchGraphRelationFilter struct {
	RelationType string                 `json:"relation_type"`
	Direction    ResearchGraphDirection `json:"direction"`
}

type ResearchGraphFactPolicy struct {
	EntityStatus              string
	EntityRelationStatus      string
	IndustryChainReviewStatus string
}

func ApprovedActiveResearchGraphFactPolicy() ResearchGraphFactPolicy {
	return ResearchGraphFactPolicy{
		EntityStatus:              "active",
		EntityRelationStatus:      "active",
		IndustryChainReviewStatus: "approved",
	}
}

type ResearchGraphQuery struct {
	AnalysisAsOf    time.Time
	SeedEntityIDs   []string
	RelationFilters []ResearchGraphRelationFilter
	MaxDepth        int
	IndustryChainID *string
	NodeBudget      int
	EdgeBudget      int
	FactPolicy      ResearchGraphFactPolicy
}

type ResearchGraphSubgraph struct {
	ActualDepth              int                           `json:"actual_depth"`
	Entities                 []ResearchGraphEntity         `json:"entities"`
	RelationDefinitions      []ResearchGraphRelation       `json:"relation_definitions"`
	EntityRelations          []ResearchGraphEntityRelation `json:"entity_relations"`
	IndustryChains           []ResearchGraphIndustryChain  `json:"industry_chains"`
	IndustryChainMemberships []ResearchGraphMembership     `json:"industry_chain_memberships"`
	IndustryChainGraphEdges  []ResearchGraphIndustryEdge   `json:"industry_chain_graph_edges"`
}

type ResearchGraphEntity struct {
	EntityID      string   `json:"entity_id"`
	EntityType    string   `json:"entity_type"`
	Name          string   `json:"name"`
	CanonicalName string   `json:"canonical_name"`
	Aliases       []string `json:"aliases"`
	Status        string   `json:"status"`
}

type ResearchGraphRelation struct {
	RelationType string `json:"relation_type"`
	Direction    string `json:"direction"`
}

type ResearchGraphEntityRelation struct {
	EntityRelationID string `json:"entity_relation_id"`
	FromEntityID     string `json:"from_entity_id"`
	ToEntityID       string `json:"to_entity_id"`
	RelationType     string `json:"relation_type"`
	Status           string `json:"status"`
}

type ResearchGraphIndustryChain struct {
	IndustryChainID  string `json:"industry_chain_id"`
	Scope            string `json:"scope"`
	TargetOutput     string `json:"target_output"`
	EndUse           string `json:"end_use"`
	Geography        string `json:"geography"`
	PrimaryCountryID string `json:"primary_country_id,omitempty"`
	AsOfDate         string `json:"as_of_date"`
	ReviewStatus     string `json:"review_status"`
}

type ResearchGraphMembership struct {
	IndustryChainID string `json:"industry_chain_id"`
	ChainNodeID     string `json:"chain_node_id"`
	Position        int    `json:"position"`
	ContextualStage string `json:"contextual_stage"`
}

type ResearchGraphIndustryEdge struct {
	IndustryChainGraphEdgeID string `json:"industry_chain_graph_edge_id"`
	IndustryChainID          string `json:"industry_chain_id"`
	FromChainNodeID          string `json:"from_chain_node_id"`
	ToChainNodeID            string `json:"to_chain_node_id"`
	RelationType             string `json:"relation_type"`
}

type ResearchGraphValidationError struct{ Reason string }

func (e *ResearchGraphValidationError) Error() string { return e.Reason }

type ResearchGraphResourceLimitError struct {
	Reason        string
	Component     string
	ActualRows    *int64
	MaxRows       *int64
	ActualBytes   *int64
	MaxBytes      *int64
	RetryGuidance string
}

func (e *ResearchGraphResourceLimitError) Error() string { return e.Reason }

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusMerged   Status = "merged"
)

type IndustryChainContextualStage string

const (
	IndustryChainContextualStageUpstream   IndustryChainContextualStage = "upstream"
	IndustryChainContextualStageMidstream  IndustryChainContextualStage = "midstream"
	IndustryChainContextualStageDownstream IndustryChainContextualStage = "downstream"
)

type IndustryChainGraphRelationType string

const (
	IndustryChainGraphRelationComponentOf IndustryChainGraphRelationType = "is_component_of"
	IndustryChainGraphRelationInputTo     IndustryChainGraphRelationType = "input_to"
	IndustryChainGraphRelationDependsOn   IndustryChainGraphRelationType = "depends_on"
)

type IndustryChainNodeMembership struct {
	IndustryChainID string
	ChainNodeID     string
	Position        int
	ContextualStage IndustryChainContextualStage
}

func (m IndustryChainNodeMembership) Validate() error {
	if strings.TrimSpace(m.IndustryChainID) == "" || strings.TrimSpace(m.ChainNodeID) == "" {
		return fmt.Errorf("industry chain membership identities are required")
	}
	if m.Position <= 0 {
		return fmt.Errorf("industry chain membership position must be positive")
	}
	if !validStatus(m.ContextualStage, IndustryChainContextualStageUpstream, IndustryChainContextualStageMidstream, IndustryChainContextualStageDownstream) {
		return fmt.Errorf("unsupported industry chain contextual stage %q", m.ContextualStage)
	}
	return nil
}

type IndustryChainGraphEdge struct {
	ID              string
	IndustryChainID string
	FromChainNodeID string
	ToChainNodeID   string
	RelationType    IndustryChainGraphRelationType
}

func (e IndustryChainGraphEdge) Validate() error {
	if strings.TrimSpace(e.ID) == "" || strings.TrimSpace(e.IndustryChainID) == "" ||
		strings.TrimSpace(e.FromChainNodeID) == "" || strings.TrimSpace(e.ToChainNodeID) == "" {
		return fmt.Errorf("industry chain graph identities are required")
	}
	if e.FromChainNodeID == e.ToChainNodeID {
		return fmt.Errorf("industry chain graph self edge is forbidden")
	}
	if !validStatus(e.RelationType, IndustryChainGraphRelationInputTo, IndustryChainGraphRelationComponentOf, IndustryChainGraphRelationDependsOn) {
		return fmt.Errorf("unsupported industry chain graph relation %q", e.RelationType)
	}
	return nil
}

func validStatus[T comparable](value T, allowed ...T) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

const (
	EntityTypeEventRoleSubject         = "event_subject"
	EntityTypeEventRoleActor           = "actor"
	EntityTypeEventRoleAffectedEntity  = "affected_entity"
	EntityTypeEventRoleStatementSource = "statement_source"
	EntityTypeEventRoleObject          = "event_object"
	EntityTypeEventRoleContext         = "context"
)

func validateStringSet(name string, values []string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			return fmt.Errorf("%s contains a blank value", name)
		}
		if _, ok := seen[normalized]; ok {
			return fmt.Errorf("%s contains duplicate value %q", name, normalized)
		}
		seen[normalized] = struct{}{}
	}
	return nil
}
