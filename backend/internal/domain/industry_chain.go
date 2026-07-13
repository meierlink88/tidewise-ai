package domain

import (
	"fmt"
	"strings"
	"time"
)

type IndustryChainScope string

const (
	IndustryChainScopeGlobal   IndustryChainScope = "global"
	IndustryChainScopeEconomy  IndustryChainScope = "economy"
	IndustryChainScopeRegional IndustryChainScope = "regional"
)

type IndustryChainStage string

const (
	IndustryChainStageUpstream       IndustryChainStage = "upstream"
	IndustryChainStageMidstream      IndustryChainStage = "midstream"
	IndustryChainStageDownstream     IndustryChainStage = "downstream"
	IndustryChainStageInfrastructure IndustryChainStage = "infrastructure"
	IndustryChainStageService        IndustryChainStage = "service"
)

type IndustryChainRole string

const (
	IndustryChainRoleResource       IndustryChainRole = "resource"
	IndustryChainRoleMaterial       IndustryChainRole = "material"
	IndustryChainRoleEquipment      IndustryChainRole = "equipment"
	IndustryChainRoleComponent      IndustryChainRole = "component"
	IndustryChainRoleProcess        IndustryChainRole = "process"
	IndustryChainRoleProduct        IndustryChainRole = "product"
	IndustryChainRoleService        IndustryChainRole = "service"
	IndustryChainRoleInfrastructure IndustryChainRole = "infrastructure"
)

type IndustryChainRelationType string

const (
	IndustryChainRelationSuppliesTo     IndustryChainRelationType = "supplies_to"
	IndustryChainRelationDependsOn      IndustryChainRelationType = "depends_on"
	IndustryChainRelationSubstitutesFor IndustryChainRelationType = "substitutes_for"
)

type PhysicalConstraintType string

const (
	PhysicalConstraintPowerCapacity          PhysicalConstraintType = "power_capacity"
	PhysicalConstraintThermalDissipation     PhysicalConstraintType = "thermal_dissipation"
	PhysicalConstraintBandwidth              PhysicalConstraintType = "bandwidth"
	PhysicalConstraintLatency                PhysicalConstraintType = "latency"
	PhysicalConstraintProductionCapacity     PhysicalConstraintType = "production_capacity"
	PhysicalConstraintProcessYield           PhysicalConstraintType = "process_yield"
	PhysicalConstraintMaterialPurity         PhysicalConstraintType = "material_purity"
	PhysicalConstraintReliability            PhysicalConstraintType = "reliability"
	PhysicalConstraintProcessCycleTime       PhysicalConstraintType = "process_cycle_time"
	PhysicalConstraintPackagingDensity       PhysicalConstraintType = "packaging_density"
	PhysicalConstraintEquipmentCapacity      PhysicalConstraintType = "equipment_capacity"
	PhysicalConstraintInfrastructureAccess   PhysicalConstraintType = "infrastructure_access"
	PhysicalConstraintPhysicalExpansionCycle PhysicalConstraintType = "physical_expansion_cycle"
)

type IndustryChainProfile struct {
	EntityID               string
	ChainCode              string
	Definition             string
	BoundaryNote           string
	ScopeType              IndustryChainScope
	PrimaryEconomyEntityID string
	Version                int
	ReviewStatus           ReviewStatus
	SourceName             string
	SourceURL              string
	VerifiedAt             time.Time
}

func (p IndustryChainProfile) Validate() error {
	if p.EntityID == "" || p.ChainCode == "" || p.Definition == "" {
		return fmt.Errorf("industry chain identity fields are required")
	}
	if !validStatus(p.ScopeType, IndustryChainScopeGlobal, IndustryChainScopeEconomy, IndustryChainScopeRegional) {
		return fmt.Errorf("unsupported industry chain scope %q", p.ScopeType)
	}
	if (p.ScopeType == IndustryChainScopeGlobal) != (p.PrimaryEconomyEntityID == "") {
		return fmt.Errorf("industry chain scope economy mismatch")
	}
	if p.Version <= 0 {
		return fmt.Errorf("industry chain version must be positive")
	}
	if !validStatus(p.ReviewStatus, ReviewStatusCandidate, ReviewStatusReviewed, ReviewStatusApproved) {
		return fmt.Errorf("unsupported review status %q", p.ReviewStatus)
	}
	return validateIndustryChainProvenance(p.SourceName, p.SourceURL, p.VerifiedAt)
}

type IndustryChainMembership struct {
	ID                    string
	IndustryChainEntityID string
	ChainNodeEntityID     string
	StageCode             IndustryChainStage
	RoleCode              IndustryChainRole
	StageOrder            int
	IsCore                bool
	SourceName            string
	SourceURL             string
	VerifiedAt            time.Time
	Status                Status
}

func (m IndustryChainMembership) Validate() error {
	if m.ID == "" || m.IndustryChainEntityID == "" || m.ChainNodeEntityID == "" {
		return fmt.Errorf("membership identity fields are required")
	}
	if !validStatus(m.StageCode, IndustryChainStageUpstream, IndustryChainStageMidstream, IndustryChainStageDownstream, IndustryChainStageInfrastructure, IndustryChainStageService) {
		return fmt.Errorf("unsupported stage %q", m.StageCode)
	}
	if !validIndustryChainRole(m.RoleCode) {
		return fmt.Errorf("unsupported role %q", m.RoleCode)
	}
	if m.StageOrder < 0 {
		return fmt.Errorf("stage order must not be negative")
	}
	if !validStatus(m.Status, StatusActive, StatusInactive) {
		return fmt.Errorf("unsupported membership status %q", m.Status)
	}
	return validateIndustryChainProvenance(m.SourceName, m.SourceURL, m.VerifiedAt)
}

type IndustryChainTopologyEdge struct {
	ID                    string
	IndustryChainEntityID string
	FromChainNodeEntityID string
	ToChainNodeEntityID   string
	RelationType          IndustryChainRelationType
	EvidenceNote          string
	SourceName            string
	SourceURL             string
	VerifiedAt            time.Time
	Status                Status
}

func (e IndustryChainTopologyEdge) Validate() error {
	if e.ID == "" || e.IndustryChainEntityID == "" || e.FromChainNodeEntityID == "" || e.ToChainNodeEntityID == "" {
		return fmt.Errorf("topology identity fields are required")
	}
	if e.FromChainNodeEntityID == e.ToChainNodeEntityID {
		return fmt.Errorf("topology self edge is forbidden")
	}
	if !validStatus(e.RelationType, IndustryChainRelationSuppliesTo, IndustryChainRelationDependsOn, IndustryChainRelationSubstitutesFor) {
		return fmt.Errorf("unsupported topology relation %q", e.RelationType)
	}
	if !validStatus(e.Status, StatusActive, StatusInactive) {
		return fmt.Errorf("unsupported topology status %q", e.Status)
	}
	return validateIndustryChainProvenance(e.SourceName, e.SourceURL, e.VerifiedAt)
}

type IndustryChainPhysicalConstraint struct {
	ID                    string
	IndustryChainEntityID string
	ChainNodeEntityID     string
	TopologyEdgeID        string
	ConstraintType        PhysicalConstraintType
	Mechanism             string
	PhysicalLimitNote     string
	MitigationPath        string
	SourceName            string
	SourceURL             string
	VerifiedAt            time.Time
	ReviewStatus          ReviewStatus
	Status                Status
	GeneratedByAI         bool
	ReasoningFields       map[string]string
}

func (c IndustryChainPhysicalConstraint) Validate() error {
	if c.ID == "" || c.IndustryChainEntityID == "" || c.Mechanism == "" {
		return fmt.Errorf("physical constraint identity and mechanism are required")
	}
	if (c.ChainNodeEntityID == "") == (c.TopologyEdgeID == "") {
		return fmt.Errorf("physical constraint requires exactly one subject")
	}
	if !validPhysicalConstraintType(c.ConstraintType) {
		return fmt.Errorf("unsupported physical constraint %q", c.ConstraintType)
	}
	if !validStatus(c.ReviewStatus, ReviewStatusCandidate, ReviewStatusReviewed, ReviewStatusApproved) {
		return fmt.Errorf("unsupported review status %q", c.ReviewStatus)
	}
	if len(c.ReasoningFields) > 0 {
		return fmt.Errorf("reasoning fields are forbidden")
	}
	if !validStatus(c.Status, StatusActive, StatusInactive) {
		return fmt.Errorf("unsupported constraint status %q", c.Status)
	}
	return validateIndustryChainProvenance(c.SourceName, c.SourceURL, c.VerifiedAt)
}

type IndustryChainApprovalGate struct {
	HumanApprovedConstraintIDs map[string]struct{}
}

func ValidateIndustryChainBatch(memberships []IndustryChainMembership, edges []IndustryChainTopologyEdge, constraints []IndustryChainPhysicalConstraint, gate IndustryChainApprovalGate) error {
	membershipStatus := make(map[string]Status, len(memberships))
	for _, membership := range memberships {
		if err := membership.Validate(); err != nil {
			return err
		}
		key := membership.IndustryChainEntityID + "|" + membership.ChainNodeEntityID
		if _, exists := membershipStatus[key]; exists {
			return fmt.Errorf("duplicate industry chain membership %q", key)
		}
		membershipStatus[key] = membership.Status
	}
	if err := ValidateIndustryChainTopology(edges); err != nil {
		return err
	}
	topologyByID := make(map[string]IndustryChainTopologyEdge, len(edges))
	for _, edge := range edges {
		if _, exists := topologyByID[edge.ID]; exists {
			return fmt.Errorf("duplicate topology ID %q", edge.ID)
		}
		for _, nodeID := range []string{edge.FromChainNodeEntityID, edge.ToChainNodeEntityID} {
			status, exists := membershipStatus[edge.IndustryChainEntityID+"|"+nodeID]
			if !exists {
				return fmt.Errorf("topology endpoint must reference same chain membership")
			}
			if edge.Status == StatusActive && status != StatusActive {
				return fmt.Errorf("active topology endpoint must reference active membership")
			}
		}
		topologyByID[edge.ID] = edge
	}
	if err := ValidateIndustryChainPhysicalConstraints(constraints, gate); err != nil {
		return err
	}
	for _, constraint := range constraints {
		if constraint.ChainNodeEntityID != "" {
			status, exists := membershipStatus[constraint.IndustryChainEntityID+"|"+constraint.ChainNodeEntityID]
			if !exists || status != StatusActive {
				return fmt.Errorf("node constraint must reference same chain active membership")
			}
			continue
		}
		edge, exists := topologyByID[constraint.TopologyEdgeID]
		if !exists || edge.IndustryChainEntityID != constraint.IndustryChainEntityID {
			return fmt.Errorf("edge constraint must reference same chain topology")
		}
		if edge.Status != StatusActive {
			return fmt.Errorf("edge constraint must reference active topology")
		}
	}
	return nil
}

func ValidateIndustryChainPhysicalConstraints(constraints []IndustryChainPhysicalConstraint, gate IndustryChainApprovalGate) error {
	for _, constraint := range constraints {
		if err := constraint.Validate(); err != nil {
			return err
		}
		if constraint.GeneratedByAI && constraint.ReviewStatus == ReviewStatusApproved {
			if _, approved := gate.HumanApprovedConstraintIDs[constraint.ID]; !approved {
				return fmt.Errorf("AI-generated approved constraint requires explicit human approval")
			}
		}
	}
	return nil
}

func ValidateIndustryChainTopology(edges []IndustryChainTopologyEdge) error {
	seen := map[string]IndustryChainRelationType{}
	for _, edge := range edges {
		if err := edge.Validate(); err != nil {
			return err
		}
		key := strings.Join([]string{edge.IndustryChainEntityID, edge.FromChainNodeEntityID, edge.ToChainNodeEntityID}, "|")
		reverse := strings.Join([]string{edge.IndustryChainEntityID, edge.ToChainNodeEntityID, edge.FromChainNodeEntityID}, "|")
		if prior, ok := seen[key]; ok && (prior == IndustryChainRelationSubstitutesFor || edge.RelationType == IndustryChainRelationSubstitutesFor) {
			return fmt.Errorf("conflicting topology relations")
		}
		if prior, ok := seen[reverse]; ok && ((prior == IndustryChainRelationSuppliesTo && edge.RelationType == IndustryChainRelationDependsOn) || (prior == IndustryChainRelationDependsOn && edge.RelationType == IndustryChainRelationSuppliesTo)) {
			return fmt.Errorf("canonical supplies_to fact duplicated by reverse depends_on")
		}
		seen[key] = edge.RelationType
	}
	return nil
}

func validIndustryChainRole(value IndustryChainRole) bool {
	return validStatus(value, IndustryChainRoleResource, IndustryChainRoleMaterial, IndustryChainRoleEquipment, IndustryChainRoleComponent, IndustryChainRoleProcess, IndustryChainRoleProduct, IndustryChainRoleService, IndustryChainRoleInfrastructure)
}

func validPhysicalConstraintType(value PhysicalConstraintType) bool {
	return validStatus(value, PhysicalConstraintPowerCapacity, PhysicalConstraintThermalDissipation, PhysicalConstraintBandwidth, PhysicalConstraintLatency, PhysicalConstraintProductionCapacity, PhysicalConstraintProcessYield, PhysicalConstraintMaterialPurity, PhysicalConstraintReliability, PhysicalConstraintProcessCycleTime, PhysicalConstraintPackagingDensity, PhysicalConstraintEquipmentCapacity, PhysicalConstraintInfrastructureAccess, PhysicalConstraintPhysicalExpansionCycle)
}

func validateIndustryChainProvenance(sourceName, sourceURL string, verifiedAt time.Time) error {
	if sourceName == "" || sourceURL == "" || verifiedAt.IsZero() {
		return fmt.Errorf("source name, source url and verified at are required")
	}
	return nil
}
