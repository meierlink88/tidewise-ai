package entity

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusMerged   Status = "merged"
)

type EntityType string

const (
	EntityTypeAllianceOrg   EntityType = "alliance_org"
	EntityTypeEconomy       EntityType = "economy"
	EntityTypePolicyBody    EntityType = "policy_body"
	EntityTypeMarket        EntityType = "market"
	EntityTypeIndex         EntityType = "index"
	EntityTypeBenchmark     EntityType = "benchmark"
	EntityTypeSector        EntityType = "sector"
	EntityTypeIndustry      EntityType = "industry"
	EntityTypeConcept       EntityType = "concept"
	EntityTypeIndustryChain EntityType = "industry_chain"
	EntityTypeChainNode     EntityType = "chain_node"
	EntityTypeTheme         EntityType = "theme"
	EntityTypeCompany       EntityType = "company"
	EntityTypeSecurity      EntityType = "security"
	EntityTypeInstrument    EntityType = "instrument"
	EntityTypeMetric        EntityType = "metric"
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
	if e.ID == "" {
		return fmt.Errorf("entity id is required")
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

type AllianceOrgProfile struct {
	EntityID              string
	Abbreviation          string
	LeadershipSummary     string
	InfluenceScopeSummary string
}

func (p AllianceOrgProfile) Validate() error {
	if strings.TrimSpace(p.EntityID) == "" {
		return fmt.Errorf("entity id is required")
	}
	abbreviation := strings.TrimSpace(p.Abbreviation)
	if utf8.RuneCountInString(abbreviation) > 32 {
		return fmt.Errorf("abbreviation exceeds 32 characters")
	}
	if abbreviation == "—" {
		return fmt.Errorf("abbreviation placeholder is not allowed")
	}
	leadership := strings.TrimSpace(p.LeadershipSummary)
	if leadership == "" {
		return fmt.Errorf("leadership summary is required")
	}
	if utf8.RuneCountInString(leadership) > 500 {
		return fmt.Errorf("leadership summary exceeds 500 characters")
	}
	influence := strings.TrimSpace(p.InfluenceScopeSummary)
	if influence == "" {
		return fmt.Errorf("influence scope summary is required")
	}
	if utf8.RuneCountInString(influence) > 1000 {
		return fmt.Errorf("influence scope summary exceeds 1000 characters")
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
	EntityID     string
	Scope        string
	TargetOutput string
	EndUse       string
	Geography    string
	AsOfDate     time.Time
	ReviewStatus ReviewStatus
	ReviewNote   string
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

type EconomyProfile struct {
	EntityID     string
	CountryCode  string
	CurrencyCode string
	Region       string
}

type PolicyBodyProfile struct {
	EntityID     string
	BodyType     string
	Jurisdiction string
	PolicyDomain string
}

type MarketProfile struct {
	EntityID        string
	MarketType      string
	EconomyEntityID string
	CurrencyCode    string
	Timezone        string
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

type BenchmarkType string

const (
	BenchmarkTypeGovernmentBondYield BenchmarkType = "government_bond_yield"
	BenchmarkTypeFuturesPrice        BenchmarkType = "futures_price"
	BenchmarkTypeSpotPrice           BenchmarkType = "spot_price"
	BenchmarkTypeReferenceRate       BenchmarkType = "reference_rate"
)

type BenchmarkProfile struct {
	EntityID           string
	BenchmarkType      BenchmarkType
	OfficialSeriesCode string
	Provider           string
	Tenor              string
	UnderlyingSymbol   string
	CurrencyCode       string
	Unit               string
	Frequency          string
	SourceURL          string
}

func (p BenchmarkProfile) Validate() error {
	if p.EntityID == "" {
		return fmt.Errorf("entity id is required")
	}
	if !validStatus(
		p.BenchmarkType,
		BenchmarkTypeGovernmentBondYield,
		BenchmarkTypeFuturesPrice,
		BenchmarkTypeSpotPrice,
		BenchmarkTypeReferenceRate,
	) {
		return fmt.Errorf("unsupported benchmark type %q", p.BenchmarkType)
	}
	if p.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if p.CurrencyCode == "" {
		return fmt.Errorf("currency code is required")
	}
	if p.Unit == "" {
		return fmt.Errorf("unit is required")
	}
	if p.Frequency == "" {
		return fmt.Errorf("frequency is required")
	}
	if p.SourceURL == "" {
		return fmt.Errorf("source url is required")
	}
	return nil
}

type BenchmarkObservationQualityStatus string

const (
	BenchmarkObservationQualityRaw       BenchmarkObservationQualityStatus = "raw"
	BenchmarkObservationQualityValidated BenchmarkObservationQualityStatus = "validated"
	BenchmarkObservationQualitySuspect   BenchmarkObservationQualityStatus = "suspect"
	BenchmarkObservationQualityRejected  BenchmarkObservationQualityStatus = "rejected"
)

type BenchmarkObservation struct {
	ID                 string
	BenchmarkEntityID  string
	ObservedAt         time.Time
	Value              string
	Unit               string
	SourceName         string
	SourceURL          string
	ExternalSeriesCode string
	QualityStatus      BenchmarkObservationQualityStatus
}

func (o BenchmarkObservation) Validate() error {
	if o.ID == "" {
		return fmt.Errorf("observation id is required")
	}
	if o.BenchmarkEntityID == "" {
		return fmt.Errorf("benchmark entity id is required")
	}
	if o.ObservedAt.IsZero() {
		return fmt.Errorf("observed at is required")
	}
	if o.Value == "" {
		return fmt.Errorf("value is required")
	}
	if o.Unit == "" {
		return fmt.Errorf("unit is required")
	}
	if o.SourceName == "" {
		return fmt.Errorf("source name is required")
	}
	if !validStatus(
		o.QualityStatus,
		BenchmarkObservationQualityRaw,
		BenchmarkObservationQualityValidated,
		BenchmarkObservationQualitySuspect,
		BenchmarkObservationQualityRejected,
	) {
		return fmt.Errorf("unsupported benchmark observation quality status %q", o.QualityStatus)
	}
	return nil
}

type BenchmarkObservationFilter struct {
	BenchmarkEntityID string
	Limit             int
}

type BenchmarkObservationWriteResult struct {
	Observation BenchmarkObservation
	Created     bool
}

type SectorProfile struct {
	EntityID               string
	SectorSystem           string
	SectorCode             string
	SectorType             string
	ExchangeScope          string
	ConstituentCount       int
	ListDate               *time.Time
	ParentSectorEntityID   string
	ClassificationCode     SectorClassification
	PrimaryMarketEntityID  string
	PrimaryEconomyEntityID string
	MethodologyURL         string
	ReviewStatus           SectorReviewStatus
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
	EntityID                    string
	RegistrationEconomyEntityID string
	Area                        string
	IndustryName                string
	ControllerName              string
	ControllerType              string
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

type MetricProfile struct {
	EntityID   string
	MetricType string
	Unit       string
	Frequency  string
}

type CommodityProfile struct {
	EntityID      string
	CommodityType string
}

type PersonProfile struct {
	EntityID             string
	RoleTitle            string
	OrganizationEntityID string
	EconomyEntityID      string
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
		EntityTypeAllianceOrg,
		EntityTypeEconomy,
		EntityTypePolicyBody,
		EntityTypeMarket,
		EntityTypeIndex,
		EntityTypeBenchmark,
		EntityTypeSector,
		EntityTypeIndustry,
		EntityTypeConcept,
		EntityTypeIndustryChain,
		EntityTypeChainNode,
		EntityTypeTheme,
		EntityTypeCompany,
		EntityTypeSecurity,
		EntityTypeInstrument,
		EntityTypeMetric,
		EntityTypeCommodity,
		EntityTypeProduct,
		EntityTypePerson,
	)
}

// EntityTypeDefinition defines one versioned entity vocabulary owned by the Entity domain.
type EntityTypeDefinition struct {
	TypeKey              string
	Version              int
	NameZH               string
	NameEN               string
	BusinessDefinition   string
	InclusionCriteria    []string
	ExclusionCriteria    []string
	EventLinkAllowed     bool
	SignalSubjectAllowed bool
	DirectTargetMode     string
	AllowedEventRoles    []string
	Status               EntityTypeDefinitionStatus
}

type EntityTypeDefinitionStatus string

const (
	EntityTypeDefinitionActive         EntityTypeDefinitionStatus = "active"
	EntityTypeDefinitionDeprecated     EntityTypeDefinitionStatus = "deprecated"
	EntityTypeDirectTargetAllow                                   = "allow"
	EntityTypeDirectTargetConditional                             = "conditional"
	EntityTypeDirectTargetDeny                                    = "deny"
	EntityTypeDirectTargetContext                                 = "context"
	EntityTypeEventRoleSubject                                    = "event_subject"
	EntityTypeEventRoleActor                                      = "actor"
	EntityTypeEventRoleAffectedEntity                             = "affected_entity"
	EntityTypeEventRoleStatementSource                            = "statement_source"
	EntityTypeEventRoleObject                                     = "event_object"
	EntityTypeEventRoleContext                                    = "context"
)

func (d EntityTypeDefinition) Validate() error {
	if strings.TrimSpace(d.TypeKey) == "" || d.Version <= 0 || strings.TrimSpace(d.NameZH) == "" || strings.TrimSpace(d.NameEN) == "" || strings.TrimSpace(d.BusinessDefinition) == "" {
		return fmt.Errorf("entity type definition identity, version, names, and business definition are required")
	}
	if !validStatus(d.Status, EntityTypeDefinitionActive, EntityTypeDefinitionDeprecated) {
		return fmt.Errorf("unsupported entity type definition status %q", d.Status)
	}
	if !validStatus(d.DirectTargetMode, EntityTypeDirectTargetAllow, EntityTypeDirectTargetConditional, EntityTypeDirectTargetDeny, EntityTypeDirectTargetContext) {
		return fmt.Errorf("unsupported entity type direct target mode %q", d.DirectTargetMode)
	}
	if len(d.InclusionCriteria) == 0 || len(d.ExclusionCriteria) == 0 {
		return fmt.Errorf("entity type definition inclusion and exclusion criteria are required")
	}
	if err := validateStringSet("inclusion criteria", d.InclusionCriteria); err != nil {
		return err
	}
	if err := validateStringSet("exclusion criteria", d.ExclusionCriteria); err != nil {
		return err
	}
	if len(d.AllowedEventRoles) == 0 {
		return fmt.Errorf("entity type definition allowed event roles are required")
	}
	if err := validateStringSet("allowed event roles", d.AllowedEventRoles); err != nil {
		return err
	}
	for _, role := range d.AllowedEventRoles {
		if !validStatus(role, EntityTypeEventRoleSubject, EntityTypeEventRoleActor, EntityTypeEventRoleAffectedEntity, EntityTypeEventRoleStatementSource, EntityTypeEventRoleObject, EntityTypeEventRoleContext) {
			return fmt.Errorf("unsupported entity type event role %q", role)
		}
	}
	return nil
}

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
