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
