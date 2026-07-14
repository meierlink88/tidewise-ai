package domain

import (
	"fmt"
	"strings"
	"time"
)

type ChainNodeRelationType string

const (
	ChainNodeRelationSubcategoryOf ChainNodeRelationType = "is_subcategory_of"
	ChainNodeRelationComponentOf   ChainNodeRelationType = "is_component_of"
	ChainNodeRelationInputTo       ChainNodeRelationType = "input_to"
	ChainNodeRelationDependsOn     ChainNodeRelationType = "depends_on"
)

type ChainNodeRelation struct {
	ID                    string                `json:"id"`
	FromChainNodeEntityID string                `json:"from_chain_node_entity_id"`
	ToChainNodeEntityID   string                `json:"to_chain_node_entity_id"`
	RelationType          ChainNodeRelationType `json:"relation_type"`
	Mechanism             string                `json:"mechanism"`
	ConditionNote         string                `json:"condition_note,omitempty"`
	EvidenceNote          string                `json:"evidence_note"`
	Provenance            string                `json:"provenance"`
	Status                Status                `json:"status"`
	VerifiedAt            time.Time             `json:"verified_at"`
}

type ChainNodePhysicalConstraintType string

const (
	ChainNodeConstraintPowerCapacity          ChainNodePhysicalConstraintType = "power_capacity"
	ChainNodeConstraintThermalDissipation     ChainNodePhysicalConstraintType = "thermal_dissipation"
	ChainNodeConstraintProductionCapacity     ChainNodePhysicalConstraintType = "production_capacity"
	ChainNodeConstraintProcessYield           ChainNodePhysicalConstraintType = "process_yield"
	ChainNodeConstraintMaterialPurity         ChainNodePhysicalConstraintType = "material_purity"
	ChainNodeConstraintEquipmentCapacity      ChainNodePhysicalConstraintType = "equipment_capacity"
	ChainNodeConstraintInfrastructureAccess   ChainNodePhysicalConstraintType = "infrastructure_access"
	ChainNodeConstraintPhysicalExpansionCycle ChainNodePhysicalConstraintType = "physical_expansion_cycle"
	ChainNodeConstraintResourceAvailability   ChainNodePhysicalConstraintType = "resource_availability"
	ChainNodeConstraintProcessCycleTime       ChainNodePhysicalConstraintType = "process_cycle_time"
)

type ChainNodePhysicalConstraint struct {
	ID                  string                          `json:"id"`
	ChainNodeEntityID   string                          `json:"chain_node_entity_id,omitempty"`
	ChainNodeRelationID string                          `json:"chain_node_relation_id,omitempty"`
	ConstraintType      ChainNodePhysicalConstraintType `json:"constraint_type"`
	Description         string                          `json:"description"`
	ConditionNote       string                          `json:"condition_note,omitempty"`
	EvidenceNote        string                          `json:"evidence_note"`
	Provenance          string                          `json:"provenance"`
	Status              Status                          `json:"status"`
	VerifiedAt          time.Time                       `json:"verified_at"`
}

func (c ChainNodePhysicalConstraint) Validate() error {
	if c.ID == "" || (c.ChainNodeEntityID == "") == (c.ChainNodeRelationID == "") {
		return fmt.Errorf("physical constraint requires id and exactly one new subject")
	}
	if !validStatus(c.ConstraintType, ChainNodeConstraintPowerCapacity, ChainNodeConstraintThermalDissipation, ChainNodeConstraintProductionCapacity, ChainNodeConstraintProcessYield, ChainNodeConstraintMaterialPurity, ChainNodeConstraintEquipmentCapacity, ChainNodeConstraintInfrastructureAccess, ChainNodeConstraintPhysicalExpansionCycle, ChainNodeConstraintResourceAvailability, ChainNodeConstraintProcessCycleTime) {
		return fmt.Errorf("unsupported physical constraint %q", c.ConstraintType)
	}
	if strings.TrimSpace(c.Description) == "" || strings.TrimSpace(c.EvidenceNote) == "" || strings.TrimSpace(c.Provenance) == "" || c.VerifiedAt.IsZero() {
		return fmt.Errorf("physical constraint description, evidence and provenance are required")
	}
	if c.ConditionNote != "" && strings.TrimSpace(c.ConditionNote) == "" {
		return fmt.Errorf("physical constraint condition note cannot be blank")
	}
	for _, forbidden := range []string{"价格", "涨跌", "情绪", "政策支持", "市场表现"} {
		if strings.Contains(c.Description, forbidden) {
			return fmt.Errorf("physical constraint contains non-physical claim %q", forbidden)
		}
	}
	if !validStatus(c.Status, StatusActive, StatusInactive) {
		return fmt.Errorf("unsupported physical constraint status %q", c.Status)
	}
	return nil
}

func (r ChainNodeRelation) Validate() error {
	if r.ID == "" || r.FromChainNodeEntityID == "" || r.ToChainNodeEntityID == "" || strings.TrimSpace(r.Mechanism) == "" || strings.TrimSpace(r.EvidenceNote) == "" || strings.TrimSpace(r.Provenance) == "" || r.VerifiedAt.IsZero() {
		return fmt.Errorf("chain node relation identity, mechanism, evidence and provenance are required")
	}
	if r.FromChainNodeEntityID == r.ToChainNodeEntityID {
		return fmt.Errorf("chain node relation self edge is forbidden")
	}
	if r.ConditionNote != "" && strings.TrimSpace(r.ConditionNote) == "" {
		return fmt.Errorf("chain node relation condition note cannot be blank")
	}
	if !validStatus(r.RelationType, ChainNodeRelationSubcategoryOf, ChainNodeRelationComponentOf, ChainNodeRelationInputTo, ChainNodeRelationDependsOn) {
		return fmt.Errorf("unsupported chain node relation %q", r.RelationType)
	}
	if !validStatus(r.Status, StatusActive, StatusInactive) {
		return fmt.Errorf("unsupported chain node relation status %q", r.Status)
	}
	return nil
}

func ValidateChainNodeRelationBatch(relations []ChainNodeRelation) error {
	seen := map[string]ChainNodeRelationType{}
	mechanisms := map[string]ChainNodeRelationType{}
	for _, relation := range relations {
		if err := relation.Validate(); err != nil {
			return err
		}
		key := relation.FromChainNodeEntityID + "|" + relation.ToChainNodeEntityID + "|" + string(relation.RelationType)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate chain node relation %q", key)
		}
		seen[key] = relation.RelationType
		if relation.RelationType == ChainNodeRelationInputTo || relation.RelationType == ChainNodeRelationDependsOn {
			mechanismKey := relation.FromChainNodeEntityID + "|" + relation.ToChainNodeEntityID + "|" + strings.ToLower(strings.TrimSpace(relation.Mechanism))
			if prior, exists := mechanisms[mechanismKey]; exists && prior != relation.RelationType {
				return fmt.Errorf("input_to and depends_on cannot share mechanism %q", mechanismKey)
			}
			mechanisms[mechanismKey] = relation.RelationType
		}
	}
	return nil
}
