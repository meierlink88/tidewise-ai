package domain

import (
	"fmt"
	"strings"
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
	EntityTypeIndustryChain EntityType = "industry_chain"
	EntityTypeChainNode     EntityType = "chain_node"
	EntityTypeTheme         EntityType = "theme"
	EntityTypeCompany       EntityType = "company"
	EntityTypeSecurity      EntityType = "security"
	EntityTypeInstrument    EntityType = "instrument"
	EntityTypeMetric        EntityType = "metric"
	EntityTypeCommodity     EntityType = "commodity"
	EntityTypePerson        EntityType = "person"
)

type EntityNode struct {
	ID            string
	EntityType    EntityType
	LayerCode     string
	Name          string
	CanonicalName string
	Aliases       []string
	Status        Status
}

func (e EntityNode) Validate() error {
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

type EntityEdge struct {
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
