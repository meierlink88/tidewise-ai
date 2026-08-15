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
	ObjectTypeCountry      = "country"
	ObjectTypeRegion       = "region"
	ObjectTypeOrganization = "organization"
	EntityIDPrefix         = coreid.Entity
	EntityRelationIDPrefix = coreid.EntityRelation
	CountryIDPrefix        = coreid.Country
	RegionIDPrefix         = coreid.Region
	OrganizationIDPrefix   = coreid.Organization
)

func IsEntityID(value string) bool         { return coreid.Is(value, EntityIDPrefix) }
func IsEntityRelationID(value string) bool { return coreid.Is(value, EntityRelationIDPrefix) }
func IsCountryID(value string) bool        { return coreid.Is(value, CountryIDPrefix) }
func IsRegionID(value string) bool         { return coreid.Is(value, RegionIDPrefix) }
func IsOrganizationID(value string) bool   { return coreid.Is(value, OrganizationIDPrefix) }

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
	MembershipReviewStatus    string
	MembershipStatus          string
	GraphEdgeReviewStatus     string
	GraphEdgeStatus           string
}

func ApprovedActiveResearchGraphFactPolicy() ResearchGraphFactPolicy {
	return ResearchGraphFactPolicy{
		EntityStatus:              "active",
		EntityRelationStatus:      "active",
		IndustryChainReviewStatus: "approved",
		MembershipReviewStatus:    "approved",
		MembershipStatus:          "active",
		GraphEdgeReviewStatus:     "approved",
		GraphEdgeStatus:           "active",
	}
}

type ResearchGraphQuery struct {
	AnalysisAsOf          time.Time
	SeedEntityIDs         []string
	RelationFilters       []ResearchGraphRelationFilter
	MaxDepth              int
	IndustryChainEntityID *string
	NodeBudget            int
	EdgeBudget            int
	FactPolicy            ResearchGraphFactPolicy
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
	IndustryChainEntityID string `json:"industry_chain_entity_id"`
	Scope                 string `json:"scope"`
	TargetOutput          string `json:"target_output"`
	EndUse                string `json:"end_use"`
	Geography             string `json:"geography"`
	PrimaryCountryID      string `json:"primary_country_id,omitempty"`
	AsOfDate              string `json:"as_of_date"`
	ReviewStatus          string `json:"review_status"`
}

type ResearchGraphMembership struct {
	IndustryChainEntityID string `json:"industry_chain_entity_id"`
	ChainNodeEntityID     string `json:"chain_node_entity_id"`
	Position              int    `json:"position"`
	ContextualStage       string `json:"contextual_stage"`
	ReviewStatus          string `json:"review_status"`
	Status                string `json:"status"`
}

type ResearchGraphIndustryEdge struct {
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

type EntityType string

const (
	EntityTypePolicyBody    EntityType = "policy_body"
	EntityTypeMarket        EntityType = "market"
	EntityTypeIndex         EntityType = "index"
	EntityTypeSector        EntityType = "sector"
	EntityTypeIndustry      EntityType = "industry"
	EntityTypeConcept       EntityType = "concept"
	EntityTypeIndustryChain EntityType = "industry_chain"
	EntityTypeChainNode     EntityType = "chain_node"
	EntityTypeTheme         EntityType = "theme"
	EntityTypeCompany       EntityType = "company"
	EntityTypeSecurity      EntityType = "security"
	EntityTypeInstrument    EntityType = "instrument"
	EntityTypeCommodity     EntityType = "commodity"
	EntityTypeProduct       EntityType = "product"
	EntityTypePerson        EntityType = "person"
)

type Entity struct {
	ID            string
	EntityType    EntityType
	LayerCode     string
	Name          string
	CanonicalName string
	Aliases       []string
	Status        Status
}

func (e Entity) Validate() error {
	if !IsEntityID(e.ID) {
		return fmt.Errorf("entity id must equal %s immediately followed by a canonical lowercase UUID", EntityIDPrefix)
	}
	if e.EntityType == "" {
		return fmt.Errorf("entity type is required")
	}
	if !validEntityType(e.EntityType) {
		return fmt.Errorf("unsupported entity type %q", e.EntityType)
	}
	if e.LayerCode == "" {
		return fmt.Errorf("layer code is required")
	}
	if e.Name == "" {
		return fmt.Errorf("name is required")
	}
	if e.CanonicalName == "" {
		return fmt.Errorf("canonical name is required")
	}
	if !validStatus(e.Status, StatusActive, StatusInactive, StatusMerged) {
		return fmt.Errorf("unsupported entity status %q", e.Status)
	}
	return nil
}

type EntityRelation struct {
	ID           string
	FromEntityID string
	ToEntityID   string
	RelationType string
	EvidenceNote string
	Status       Status
}

func (r EntityRelation) Validate() error {
	if !IsEntityRelationID(r.ID) {
		return fmt.Errorf("entity relation id must equal %s immediately followed by a canonical lowercase UUID", EntityRelationIDPrefix)
	}
	if !IsEntityID(r.FromEntityID) || !IsEntityID(r.ToEntityID) || r.FromEntityID == r.ToEntityID {
		return fmt.Errorf("entity relation endpoints must be distinct Entity IDs")
	}
	if strings.TrimSpace(r.RelationType) == "" {
		return fmt.Errorf("entity relation type is required")
	}
	if !validStatus(r.Status, StatusActive, StatusInactive) {
		return fmt.Errorf("unsupported entity relation status %q", r.Status)
	}
	return nil
}

type EntityExternalIdentifier struct {
	ID                 string
	EntityID           string
	SourceSystem       string
	SourceTaxonomyType string
	ExternalCode       string
	ExternalName       string
	Status             Status
}

func (i EntityExternalIdentifier) Validate() error {
	if strings.TrimSpace(i.ID) == "" || strings.TrimSpace(i.EntityID) == "" {
		return fmt.Errorf("external identifier id and entity id are required")
	}
	if strings.TrimSpace(i.SourceSystem) == "" || strings.TrimSpace(i.SourceTaxonomyType) == "" || strings.TrimSpace(i.ExternalCode) == "" || strings.TrimSpace(i.ExternalName) == "" {
		return fmt.Errorf("external identifier identity fields are required")
	}
	if !validStatus(i.Status, StatusActive, StatusInactive) {
		return fmt.Errorf("unsupported external identifier status %q", i.Status)
	}
	return nil
}

type ConceptType string

const (
	ConceptTypeTechnology       ConceptType = "technology"
	ConceptTypePolicy           ConceptType = "policy"
	ConceptTypeApplication      ConceptType = "application"
	ConceptTypeDemand           ConceptType = "demand"
	ConceptTypeBusinessModel    ConceptType = "business_model"
	ConceptTypeCompanyEcosystem ConceptType = "company_ecosystem"
	ConceptTypeProductEcosystem ConceptType = "product_ecosystem"
	ConceptTypeEventNarrative   ConceptType = "event_narrative"
	ConceptTypeMarketTheme      ConceptType = "market_theme"
)

type IndustryChainContextualStage string

const (
	IndustryChainContextualStageUpstream   IndustryChainContextualStage = "upstream"
	IndustryChainContextualStageMidstream  IndustryChainContextualStage = "midstream"
	IndustryChainContextualStageDownstream IndustryChainContextualStage = "downstream"
)

type IndustryChainSegmentKind string

type ChainNodeRelationType string

const (
	ChainNodeRelationSubcategoryOf ChainNodeRelationType = "is_subcategory_of"
	ChainNodeRelationComponentOf   ChainNodeRelationType = "is_component_of"
	ChainNodeRelationInputTo       ChainNodeRelationType = "input_to"
	ChainNodeRelationDependsOn     ChainNodeRelationType = "depends_on"
)

const (
	IndustryChainSegmentDirectCandidate     IndustryChainSegmentKind = "direct_candidate"
	IndustryChainSegmentCompressedCandidate IndustryChainSegmentKind = "compressed_candidate"
)

type IndustryProfile struct {
	EntityID               string
	ClassificationSystem   string
	ClassificationVersion  string
	IndustryCode           string
	ClassificationLevel    int
	ParentIndustryEntityID string
	HierarchyPathCodes     []string
	Definition             string
	BoundaryNote           string
	ReviewStatus           ReviewStatus
}

func (p IndustryProfile) Validate() error {
	if strings.TrimSpace(p.EntityID) == "" ||
		strings.TrimSpace(p.ClassificationSystem) == "" ||
		strings.TrimSpace(p.ClassificationVersion) == "" ||
		strings.TrimSpace(p.IndustryCode) == "" ||
		strings.TrimSpace(p.Definition) == "" ||
		strings.TrimSpace(p.BoundaryNote) == "" {
		return fmt.Errorf("industry identity, classification, definition, and boundary are required")
	}
	if p.ClassificationLevel < 1 || p.ClassificationLevel > 3 {
		return fmt.Errorf("industry classification level must be 1, 2, or 3")
	}
	if (p.ClassificationLevel == 1) != (strings.TrimSpace(p.ParentIndustryEntityID) == "") {
		return fmt.Errorf("industry parent must be absent only at level 1")
	}
	if len(p.HierarchyPathCodes) != p.ClassificationLevel {
		return fmt.Errorf("industry hierarchy path length must equal classification level")
	}
	for _, code := range p.HierarchyPathCodes {
		if strings.TrimSpace(code) == "" {
			return fmt.Errorf("industry hierarchy path contains a blank code")
		}
	}
	if p.HierarchyPathCodes[len(p.HierarchyPathCodes)-1] != p.IndustryCode {
		return fmt.Errorf("industry hierarchy path must end with industry code")
	}
	if !validMasterDataReviewStatus(p.ReviewStatus) {
		return fmt.Errorf("unsupported industry review status %q", p.ReviewStatus)
	}
	return nil
}

type ConceptProfile struct {
	EntityID     string
	ConceptType  ConceptType
	Definition   string
	BoundaryNote string
	ReviewStatus ReviewStatus
}

type IndustryChainDefinition struct {
	EntityID         string
	Scope            string
	TargetOutput     string
	EndUse           string
	Geography        string
	PrimaryCountryID string
	AsOfDate         time.Time
	ReviewStatus     ReviewStatus
	ReviewNote       string
}

func (d IndustryChainDefinition) Validate() error {
	if strings.TrimSpace(d.EntityID) == "" ||
		strings.TrimSpace(d.Scope) == "" ||
		strings.TrimSpace(d.TargetOutput) == "" ||
		strings.TrimSpace(d.EndUse) == "" ||
		strings.TrimSpace(d.Geography) == "" ||
		d.AsOfDate.IsZero() {
		return fmt.Errorf("industry chain identity, scope, output, end use, geography, and as-of date are required")
	}
	if !validMasterDataReviewStatus(d.ReviewStatus) {
		return fmt.Errorf("unsupported industry chain review status %q", d.ReviewStatus)
	}
	if d.PrimaryCountryID != "" && !IsCountryID(d.PrimaryCountryID) {
		return fmt.Errorf("industry chain primary country must be a stable Country ID")
	}
	if d.ReviewNote != "" && strings.TrimSpace(d.ReviewNote) == "" {
		return fmt.Errorf("industry chain review note must be nonblank when present")
	}
	return nil
}

type IndustryChainNodeMembership struct {
	IndustryChainEntityID string
	ChainNodeEntityID     string
	Position              int
	ContextualStage       IndustryChainContextualStage
	ReviewStatus          ReviewStatus
	Status                Status
}

func (m IndustryChainNodeMembership) Validate() error {
	if strings.TrimSpace(m.IndustryChainEntityID) == "" || strings.TrimSpace(m.ChainNodeEntityID) == "" {
		return fmt.Errorf("industry chain membership identities are required")
	}
	if m.Position <= 0 {
		return fmt.Errorf("industry chain membership position must be positive")
	}
	if !validStatus(m.ContextualStage, IndustryChainContextualStageUpstream, IndustryChainContextualStageMidstream, IndustryChainContextualStageDownstream) {
		return fmt.Errorf("unsupported industry chain contextual stage %q", m.ContextualStage)
	}
	if !validMasterDataReviewStatus(m.ReviewStatus) {
		return fmt.Errorf("unsupported industry chain membership review status %q", m.ReviewStatus)
	}
	if !validStatus(m.Status, StatusActive, StatusInactive) {
		return fmt.Errorf("unsupported industry chain membership status %q", m.Status)
	}
	return nil
}

type IndustryChainGraphEdge struct {
	ID                    string
	IndustryChainEntityID string
	FromChainNodeEntityID string
	ToChainNodeEntityID   string
	RelationType          ChainNodeRelationType
	Mechanism             string
	ConditionNote         string
	SegmentKind           IndustryChainSegmentKind
	OmittedStepNote       string
	ReviewStatus          ReviewStatus
	Status                Status
}

func (e IndustryChainGraphEdge) Validate() error {
	if strings.TrimSpace(e.ID) == "" || strings.TrimSpace(e.IndustryChainEntityID) == "" ||
		strings.TrimSpace(e.FromChainNodeEntityID) == "" || strings.TrimSpace(e.ToChainNodeEntityID) == "" ||
		strings.TrimSpace(e.Mechanism) == "" {
		return fmt.Errorf("industry chain graph identity and mechanism are required")
	}
	if e.FromChainNodeEntityID == e.ToChainNodeEntityID {
		return fmt.Errorf("industry chain graph self edge is forbidden")
	}
	if !validStatus(e.RelationType, ChainNodeRelationInputTo, ChainNodeRelationComponentOf, ChainNodeRelationDependsOn) {
		return fmt.Errorf("unsupported industry chain graph relation %q", e.RelationType)
	}
	if e.ConditionNote != "" && strings.TrimSpace(e.ConditionNote) == "" {
		return fmt.Errorf("industry chain graph condition note must be nonblank when present")
	}
	if !validStatus(e.SegmentKind, IndustryChainSegmentDirectCandidate, IndustryChainSegmentCompressedCandidate) {
		return fmt.Errorf("unsupported industry chain segment kind %q", e.SegmentKind)
	}
	if e.SegmentKind == IndustryChainSegmentDirectCandidate && e.OmittedStepNote != "" {
		return fmt.Errorf("direct industry chain segment cannot omit steps")
	}
	if e.SegmentKind == IndustryChainSegmentCompressedCandidate && strings.TrimSpace(e.OmittedStepNote) == "" {
		return fmt.Errorf("compressed industry chain segment requires omitted step note")
	}
	if !validMasterDataReviewStatus(e.ReviewStatus) {
		return fmt.Errorf("unsupported industry chain graph review status %q", e.ReviewStatus)
	}
	if !validStatus(e.Status, StatusActive, StatusInactive) {
		return fmt.Errorf("unsupported industry chain graph status %q", e.Status)
	}
	return nil
}

func (p ConceptProfile) Validate() error {
	if strings.TrimSpace(p.EntityID) == "" || strings.TrimSpace(p.Definition) == "" || strings.TrimSpace(p.BoundaryNote) == "" {
		return fmt.Errorf("concept identity, definition, and boundary are required")
	}
	if !validStatus(
		p.ConceptType,
		ConceptTypeTechnology,
		ConceptTypePolicy,
		ConceptTypeApplication,
		ConceptTypeDemand,
		ConceptTypeBusinessModel,
		ConceptTypeCompanyEcosystem,
		ConceptTypeProductEcosystem,
		ConceptTypeEventNarrative,
		ConceptTypeMarketTheme,
	) {
		return fmt.Errorf("unsupported concept type %q", p.ConceptType)
	}
	if !validMasterDataReviewStatus(p.ReviewStatus) {
		return fmt.Errorf("unsupported concept review status %q", p.ReviewStatus)
	}
	return nil
}

func validMasterDataReviewStatus(value ReviewStatus) bool {
	return validStatus(value, ReviewStatusCandidate, ReviewStatusApproved)
}

type PolicyBodyProfile struct {
	EntityID     string
	BodyType     string
	Jurisdiction string
	PolicyDomain string
}

type MarketProfile struct {
	EntityID     string
	MarketType   string
	CountryID    string
	CurrencyCode string
	Timezone     string
}

type IndexProfile struct {
	EntityID       string
	IndexCode      string
	IndexType      string
	MarketEntityID string
	Provider       string
	CurrencyCode   string
	ListDate       *time.Time
}

type SectorProfile struct {
	EntityID              string
	SectorSystem          string
	SectorCode            string
	SectorType            string
	ExchangeScope         string
	ConstituentCount      int
	ListDate              *time.Time
	ParentSectorEntityID  string
	ClassificationCode    SectorClassification
	PrimaryMarketEntityID string
	PrimaryCountryID      string
	MethodologyURL        string
	ReviewStatus          SectorReviewStatus
}

type SectorClassification string

const (
	SectorClassificationIndustry SectorClassification = "industry_sector"
	SectorClassificationTheme    SectorClassification = "theme_sector"
	SectorClassificationMarket   SectorClassification = "market_sector"
	SectorClassificationStyle    SectorClassification = "style_sector"
	SectorClassificationRegion   SectorClassification = "region_sector"
)

type SectorReviewStatus string

const (
	SectorReviewCandidate SectorReviewStatus = "candidate"
	SectorReviewApproved  SectorReviewStatus = "approved"
	SectorReviewRejected  SectorReviewStatus = "rejected"
)

func (p SectorProfile) Validate() error {
	if p.PrimaryCountryID != "" && !IsCountryID(p.PrimaryCountryID) {
		return fmt.Errorf("sector primary country must be a stable Country ID")
	}
	if p.EntityID == "" {
		return fmt.Errorf("entity id is required")
	}
	if !validStatus(p.ClassificationCode, SectorClassificationIndustry, SectorClassificationTheme, SectorClassificationMarket, SectorClassificationStyle, SectorClassificationRegion) {
		return fmt.Errorf("unsupported sector classification %q", p.ClassificationCode)
	}
	if !validStatus(p.ReviewStatus, SectorReviewCandidate, SectorReviewApproved, SectorReviewRejected) {
		return fmt.Errorf("unsupported sector review status %q", p.ReviewStatus)
	}
	return nil
}

type SectorSourceTaxonomyType string

const (
	SectorSourceTaxonomyConcept     SectorSourceTaxonomyType = "concept"
	SectorSourceTaxonomyIndustry    SectorSourceTaxonomyType = "industry"
	SectorSourceTaxonomyIndexSector SectorSourceTaxonomyType = "index_sector"
)

type SectorSourceMappingStatus string

const (
	SectorSourceMappingCandidate SectorSourceMappingStatus = "candidate"
	SectorSourceMappingApproved  SectorSourceMappingStatus = "approved"
	SectorSourceMappingRejected  SectorSourceMappingStatus = "rejected"
	SectorSourceMappingMerged    SectorSourceMappingStatus = "merged"
)

type SectorSourceMapping struct {
	ID                         string
	SectorEntityID             string
	SourceSystem               string
	SourceTaxonomyType         SectorSourceTaxonomyType
	SourceSectorCode           string
	SourceSectorName           string
	SourceSectorNameNormalized string
	SourceMarketScope          string
	SourceURL                  string
	RankSnapshot               int
	SnapshotDate               *time.Time
	MappingStatus              SectorSourceMappingStatus
	ReviewNote                 string
}

func (m SectorSourceMapping) Validate() error {
	if m.ID == "" || m.SectorEntityID == "" || m.SourceSystem == "" || m.SourceSectorName == "" || m.SourceSectorNameNormalized == "" {
		return fmt.Errorf("sector source mapping identity fields are required")
	}
	if !validStatus(m.SourceTaxonomyType, SectorSourceTaxonomyConcept, SectorSourceTaxonomyIndustry, SectorSourceTaxonomyIndexSector) {
		return fmt.Errorf("unsupported source taxonomy type %q", m.SourceTaxonomyType)
	}
	if !validStatus(m.MappingStatus, SectorSourceMappingCandidate, SectorSourceMappingApproved, SectorSourceMappingRejected, SectorSourceMappingMerged) {
		return fmt.Errorf("unsupported sector source mapping status %q", m.MappingStatus)
	}
	return nil
}

type ChainNodeProfile struct {
	EntityID     string
	Definition   string
	BoundaryNote string
	ReviewStatus ReviewStatus
}

func (p ChainNodeProfile) Validate() error {
	if strings.TrimSpace(p.EntityID) == "" || strings.TrimSpace(p.Definition) == "" {
		return fmt.Errorf("chain node identity and definition are required")
	}
	if p.BoundaryNote != "" && strings.TrimSpace(p.BoundaryNote) == "" {
		return fmt.Errorf("chain node boundary note must be nonblank when present")
	}
	if p.ReviewStatus != "" && !validMasterDataReviewStatus(p.ReviewStatus) {
		return fmt.Errorf("unsupported chain node review status %q", p.ReviewStatus)
	}
	return nil
}

type Theme struct {
	Entity
}

type ThemeProfile struct {
	EntityID     string
	Definition   string
	BoundaryNote string
}

func (p ThemeProfile) Validate() error {
	if strings.TrimSpace(p.EntityID) == "" || strings.TrimSpace(p.Definition) == "" || strings.TrimSpace(p.BoundaryNote) == "" {
		return fmt.Errorf("theme identity, definition, and boundary note are required")
	}
	return nil
}

type CompanyProfile struct {
	EntityID              string
	RegistrationCountryID string
	Area                  string
	IndustryName          string
	ControllerName        string
	ControllerType        string
}

type SecurityProfile struct {
	EntityID              string
	Ticker                string
	Symbol                string
	Exchange              string
	MarketBoard           string
	SecurityType          string
	IssuerCompanyEntityID string
	ListDate              *time.Time
	DelistDate            *time.Time
	ListStatus            string
	CurrencyCode          string
}

type InstrumentProfile struct {
	EntityID           string
	InstrumentType     string
	UnderlyingEntityID string
	Exchange           string
	CurrencyCode       string
}

type CommodityProfile struct {
	EntityID      string
	CommodityType string
}

type PersonProfile struct {
	EntityID             string
	RoleTitle            string
	OrganizationEntityID string
	CountryID            string
}

type ProductProfile struct {
	EntityID        string
	ProductCategory string
	Specification   string
	ReviewStatus    ReviewStatus
}

func (p ProductProfile) Validate() error {
	if strings.TrimSpace(p.EntityID) == "" {
		return fmt.Errorf("product entity id is required")
	}
	if !validStatus(p.ReviewStatus, ReviewStatusCandidate, ReviewStatusApproved) {
		return fmt.Errorf("unsupported product review status %q", p.ReviewStatus)
	}
	return nil
}

type ReviewStatus string

const (
	ReviewStatusCandidate ReviewStatus = "candidate"
	ReviewStatusReviewed  ReviewStatus = "reviewed"
	ReviewStatusPending   ReviewStatus = "pending"
	ReviewStatusApproved  ReviewStatus = "approved"
	ReviewStatusRejected  ReviewStatus = "rejected"
)

func validStatus[T comparable](value T, allowed ...T) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func validEntityType(value EntityType) bool {
	return validStatus(
		value,
		EntityTypePolicyBody,
		EntityTypeMarket,
		EntityTypeIndex,
		EntityTypeSector,
		EntityTypeIndustry,
		EntityTypeConcept,
		EntityTypeIndustryChain,
		EntityTypeChainNode,
		EntityTypeTheme,
		EntityTypeCompany,
		EntityTypeSecurity,
		EntityTypeInstrument,
		EntityTypeCommodity,
		EntityTypeProduct,
		EntityTypePerson,
	)
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
