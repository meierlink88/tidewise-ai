package entityseed

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"
)

type IndustryChainMembershipSeed struct {
	ID               string                   `json:"id"`
	IndustryChainKey string                   `json:"industry_chain_key"`
	ChainNodeKey     string                   `json:"chain_node_key"`
	StageCode        model.IndustryChainStage `json:"stage_code"`
	RoleCode         model.IndustryChainRole  `json:"role_code"`
	StageOrder       int                      `json:"stage_order"`
	IsCore           bool                     `json:"is_core"`
	SourceName       string                   `json:"source_name"`
	SourceURL        string                   `json:"source_url"`
	VerifiedAt       time.Time                `json:"verified_at"`
	Status           model.Status             `json:"status"`
}

type IndustryChainTopologySeed struct {
	ID               string                          `json:"id"`
	IndustryChainKey string                          `json:"industry_chain_key"`
	FromChainNodeKey string                          `json:"from_chain_node_key"`
	ToChainNodeKey   string                          `json:"to_chain_node_key"`
	RelationType     model.IndustryChainRelationType `json:"relation_type"`
	EvidenceNote     string                          `json:"evidence_note"`
	SourceName       string                          `json:"source_name"`
	SourceURL        string                          `json:"source_url"`
	VerifiedAt       time.Time                       `json:"verified_at"`
	Status           model.Status                    `json:"status"`
}

type IndustryChainPhysicalConstraintSeed struct {
	ID                string                       `json:"id"`
	IndustryChainKey  string                       `json:"industry_chain_key"`
	ChainNodeKey      string                       `json:"chain_node_key,omitempty"`
	TopologyEdgeID    string                       `json:"topology_edge_id,omitempty"`
	ConstraintType    model.PhysicalConstraintType `json:"constraint_type"`
	Mechanism         string                       `json:"mechanism"`
	PhysicalLimitNote string                       `json:"physical_limit_note,omitempty"`
	MitigationPath    string                       `json:"mitigation_path,omitempty"`
	SourceName        string                       `json:"source_name"`
	SourceURL         string                       `json:"source_url"`
	VerifiedAt        time.Time                    `json:"verified_at"`
	ReviewStatus      model.ReviewStatus           `json:"review_status"`
	Status            model.Status                 `json:"status"`
	GeneratedByAI     bool                         `json:"generated_by_ai,omitempty"`
	ApprovedByHuman   bool                         `json:"approved_by_human,omitempty"`
}

type IndustryChainBatch struct {
	Profiles            []model.IndustryChainProfile
	Memberships         []model.IndustryChainMembership
	TopologyEdges       []model.IndustryChainTopologyEdge
	PhysicalConstraints []model.IndustryChainPhysicalConstraint
	ApprovalGate        model.IndustryChainApprovalGate
}

type IndustryChainWriteReport struct {
	Created, Updated, Unchanged int
}

func ValidateIndustryChainBatch(batch IndustryChainBatch) error {
	for _, value := range batch.Profiles {
		if err := value.Validate(); err != nil {
			return err
		}
	}
	return model.ValidateIndustryChainBatch(batch.Memberships, batch.TopologyEdges, batch.PhysicalConstraints, batch.ApprovalGate)
}

func IsTopologyOnlyIndustryChainBatch(batch IndustryChainBatch) bool {
	return len(batch.Profiles) == 0 && len(batch.Memberships) == 0 && len(batch.TopologyEdges) > 0 && len(batch.PhysicalConstraints) == 0
}

func IsConstraintOnlyIndustryChainBatch(batch IndustryChainBatch) bool {
	return len(batch.Profiles) == 0 && len(batch.Memberships) == 0 && len(batch.TopologyEdges) == 0 && len(batch.PhysicalConstraints) > 0
}

func ValidateIndustryChainConstraintsAgainstPersistedSubjects(constraints []model.IndustryChainPhysicalConstraint, memberships map[string]model.IndustryChainMembership, topology map[string]model.IndustryChainTopologyEdge) error {
	for _, constraint := range constraints {
		if constraint.ChainNodeEntityID != "" {
			found := false
			for _, membership := range memberships {
				if membership.IndustryChainEntityID == constraint.IndustryChainEntityID && membership.ChainNodeEntityID == constraint.ChainNodeEntityID && membership.Status == model.StatusActive {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("node constraint must reference persisted same chain active membership")
			}
			continue
		}
		edge, ok := topology[constraint.TopologyEdgeID]
		if !ok || edge.IndustryChainEntityID != constraint.IndustryChainEntityID || edge.Status != model.StatusActive {
			return fmt.Errorf("edge constraint must reference persisted same chain active topology")
		}
	}
	return nil
}

func ValidateIndustryChainTopologyAgainstPersistedMemberships(edges []model.IndustryChainTopologyEdge, memberships map[string]model.IndustryChainMembership) error {
	statusByEndpoint := map[string]model.Status{}
	for _, membership := range memberships {
		statusByEndpoint[membership.IndustryChainEntityID+"|"+membership.ChainNodeEntityID] = membership.Status
	}
	for _, edge := range edges {
		for _, nodeID := range []string{edge.FromChainNodeEntityID, edge.ToChainNodeEntityID} {
			status, exists := statusByEndpoint[edge.IndustryChainEntityID+"|"+nodeID]
			if !exists {
				return fmt.Errorf("topology endpoint must reference persisted same chain membership")
			}
			if edge.Status == model.StatusActive && status != model.StatusActive {
				return fmt.Errorf("active topology endpoint must reference persisted active membership")
			}
		}
	}
	return nil
}

func validateIndustryChainManifest(manifest Manifest, entities map[string]Entity) error {
	memberships := make([]model.IndustryChainMembership, 0, len(manifest.IndustryChainMemberships))
	for _, item := range manifest.IndustryChainMemberships {
		chain, chainOK := entities[item.IndustryChainKey]
		node, nodeOK := entities[item.ChainNodeKey]
		if !chainOK || chain.EntityType != model.EntityTypeIndustryChain || !nodeOK || node.EntityType != model.EntityTypeChainNode {
			return fmt.Errorf("industry chain membership has invalid endpoint")
		}
		memberships = append(memberships, model.IndustryChainMembership{ID: item.ID, IndustryChainEntityID: item.IndustryChainKey, ChainNodeEntityID: item.ChainNodeKey, StageCode: item.StageCode, RoleCode: item.RoleCode, StageOrder: item.StageOrder, IsCore: item.IsCore, SourceName: item.SourceName, SourceURL: item.SourceURL, VerifiedAt: item.VerifiedAt, Status: item.Status})
	}
	topology := make([]model.IndustryChainTopologyEdge, 0, len(manifest.IndustryChainTopologyEdges))
	for _, item := range manifest.IndustryChainTopologyEdges {
		topology = append(topology, model.IndustryChainTopologyEdge{ID: item.ID, IndustryChainEntityID: item.IndustryChainKey, FromChainNodeEntityID: item.FromChainNodeKey, ToChainNodeEntityID: item.ToChainNodeKey, RelationType: item.RelationType, EvidenceNote: item.EvidenceNote, SourceName: item.SourceName, SourceURL: item.SourceURL, VerifiedAt: item.VerifiedAt, Status: item.Status})
	}
	constraints := make([]model.IndustryChainPhysicalConstraint, 0, len(manifest.IndustryChainPhysicalConstraints))
	gate := model.IndustryChainApprovalGate{HumanApprovedConstraintIDs: map[string]struct{}{}}
	for _, item := range manifest.IndustryChainPhysicalConstraints {
		constraints = append(constraints, model.IndustryChainPhysicalConstraint{ID: item.ID, IndustryChainEntityID: item.IndustryChainKey, ChainNodeEntityID: item.ChainNodeKey, TopologyEdgeID: item.TopologyEdgeID, ConstraintType: item.ConstraintType, Mechanism: item.Mechanism, PhysicalLimitNote: item.PhysicalLimitNote, MitigationPath: item.MitigationPath, SourceName: item.SourceName, SourceURL: item.SourceURL, VerifiedAt: item.VerifiedAt, ReviewStatus: item.ReviewStatus, Status: item.Status, GeneratedByAI: item.GeneratedByAI})
		if item.ApprovedByHuman {
			gate.HumanApprovedConstraintIDs[item.ID] = struct{}{}
		}
	}
	return model.ValidateIndustryChainBatch(memberships, topology, constraints, gate)
}

func validateIndustryChainProfileFields(fields map[string]json.RawMessage) error {
	var version int
	if raw, ok := fields["version"]; !ok || json.Unmarshal(raw, &version) != nil || version <= 0 {
		return fmt.Errorf("version must be a positive integer")
	}
	var scope model.IndustryChainScope
	if json.Unmarshal(fields["scope_type"], &scope) != nil || (scope != model.IndustryChainScopeGlobal && scope != model.IndustryChainScopeEconomy && scope != model.IndustryChainScopeRegional) {
		return fmt.Errorf("unsupported industry chain scope %q", scope)
	}
	var review model.ReviewStatus
	if json.Unmarshal(fields["review_status"], &review) != nil || (review != model.ReviewStatusCandidate && review != model.ReviewStatusReviewed && review != model.ReviewStatusApproved) {
		return fmt.Errorf("unsupported industry chain review status %q", review)
	}
	return nil
}
