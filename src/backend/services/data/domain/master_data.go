package domain

import (
	"fmt"
	"strings"
	"time"
)

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
