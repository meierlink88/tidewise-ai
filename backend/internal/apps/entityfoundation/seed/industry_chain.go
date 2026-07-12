package seed

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type IndustryChainMembershipSeed struct {
	ID               string                    `json:"id"`
	IndustryChainKey string                    `json:"industry_chain_key"`
	ChainNodeKey     string                    `json:"chain_node_key"`
	StageCode        domain.IndustryChainStage `json:"stage_code"`
	RoleCode         domain.IndustryChainRole  `json:"role_code"`
	StageOrder       int                       `json:"stage_order"`
	IsCore           bool                      `json:"is_core"`
	SourceName       string                    `json:"source_name"`
	SourceURL        string                    `json:"source_url"`
	VerifiedAt       time.Time                 `json:"verified_at"`
	Status           domain.Status             `json:"status"`
}

type IndustryChainTopologySeed struct {
	ID               string                           `json:"id"`
	IndustryChainKey string                           `json:"industry_chain_key"`
	FromChainNodeKey string                           `json:"from_chain_node_key"`
	ToChainNodeKey   string                           `json:"to_chain_node_key"`
	RelationType     domain.IndustryChainRelationType `json:"relation_type"`
	EvidenceNote     string                           `json:"evidence_note"`
	SourceName       string                           `json:"source_name"`
	SourceURL        string                           `json:"source_url"`
	VerifiedAt       time.Time                        `json:"verified_at"`
	Status           domain.Status                    `json:"status"`
}

type IndustryChainPhysicalConstraintSeed struct {
	ID                string                        `json:"id"`
	IndustryChainKey  string                        `json:"industry_chain_key"`
	ChainNodeKey      string                        `json:"chain_node_key,omitempty"`
	TopologyEdgeID    string                        `json:"topology_edge_id,omitempty"`
	ConstraintType    domain.PhysicalConstraintType `json:"constraint_type"`
	Mechanism         string                        `json:"mechanism"`
	PhysicalLimitNote string                        `json:"physical_limit_note,omitempty"`
	MitigationPath    string                        `json:"mitigation_path,omitempty"`
	SourceName        string                        `json:"source_name"`
	SourceURL         string                        `json:"source_url"`
	VerifiedAt        time.Time                     `json:"verified_at"`
	ReviewStatus      domain.ReviewStatus           `json:"review_status"`
	Status            domain.Status                 `json:"status"`
	GeneratedByAI     bool                          `json:"generated_by_ai,omitempty"`
}

func validateIndustryChainManifest(manifest Manifest, entities map[string]Entity) error {
	memberships := make(map[string]map[string]struct{})
	seenMemberships := map[string]struct{}{}
	for _, item := range manifest.IndustryChainMemberships {
		chain, chainOK := entities[item.IndustryChainKey]
		node, nodeOK := entities[item.ChainNodeKey]
		if !chainOK || chain.EntityType != domain.EntityTypeIndustryChain || !nodeOK || node.EntityType != domain.EntityTypeChainNode {
			return fmt.Errorf("industry chain membership has invalid endpoint")
		}
		value := domain.IndustryChainMembership{ID: item.ID, IndustryChainEntityID: item.IndustryChainKey, ChainNodeEntityID: item.ChainNodeKey, StageCode: item.StageCode, RoleCode: item.RoleCode, StageOrder: item.StageOrder, IsCore: item.IsCore, SourceName: item.SourceName, SourceURL: item.SourceURL, VerifiedAt: item.VerifiedAt, Status: item.Status}
		if err := value.Validate(); err != nil {
			return err
		}
		key := item.IndustryChainKey + "|" + item.ChainNodeKey
		if _, ok := seenMemberships[key]; ok {
			return fmt.Errorf("duplicate industry chain membership %q", key)
		}
		seenMemberships[key] = struct{}{}
		if memberships[item.IndustryChainKey] == nil {
			memberships[item.IndustryChainKey] = map[string]struct{}{}
		}
		memberships[item.IndustryChainKey][item.ChainNodeKey] = struct{}{}
	}
	topology := make([]domain.IndustryChainTopologyEdge, 0, len(manifest.IndustryChainTopologyEdges))
	topologyIDs := map[string]IndustryChainTopologySeed{}
	for _, item := range manifest.IndustryChainTopologyEdges {
		members := memberships[item.IndustryChainKey]
		if _, ok := members[item.FromChainNodeKey]; !ok {
			return fmt.Errorf("topology from endpoint is not an active membership")
		}
		if _, ok := members[item.ToChainNodeKey]; !ok {
			return fmt.Errorf("topology to endpoint is not an active membership")
		}
		edge := domain.IndustryChainTopologyEdge{ID: item.ID, IndustryChainEntityID: item.IndustryChainKey, FromChainNodeEntityID: item.FromChainNodeKey, ToChainNodeEntityID: item.ToChainNodeKey, RelationType: item.RelationType, EvidenceNote: item.EvidenceNote, SourceName: item.SourceName, SourceURL: item.SourceURL, VerifiedAt: item.VerifiedAt, Status: item.Status}
		topology = append(topology, edge)
		topologyIDs[item.ID] = item
	}
	if err := domain.ValidateIndustryChainTopology(topology); err != nil {
		return err
	}
	for _, item := range manifest.IndustryChainPhysicalConstraints {
		if item.ChainNodeKey != "" {
			if _, ok := memberships[item.IndustryChainKey][item.ChainNodeKey]; !ok {
				return fmt.Errorf("physical constraint node is not a membership")
			}
		}
		if item.TopologyEdgeID != "" {
			edge, ok := topologyIDs[item.TopologyEdgeID]
			if !ok || edge.IndustryChainKey != item.IndustryChainKey {
				return fmt.Errorf("physical constraint topology edge is outside chain")
			}
		}
		value := domain.IndustryChainPhysicalConstraint{ID: item.ID, IndustryChainEntityID: item.IndustryChainKey, ChainNodeEntityID: item.ChainNodeKey, TopologyEdgeID: item.TopologyEdgeID, ConstraintType: item.ConstraintType, Mechanism: item.Mechanism, PhysicalLimitNote: item.PhysicalLimitNote, MitigationPath: item.MitigationPath, SourceName: item.SourceName, SourceURL: item.SourceURL, VerifiedAt: item.VerifiedAt, ReviewStatus: item.ReviewStatus, Status: item.Status, GeneratedByAI: item.GeneratedByAI}
		if err := value.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func validateIndustryChainProfileFields(fields map[string]json.RawMessage) error {
	var version int
	if raw, ok := fields["version"]; !ok || json.Unmarshal(raw, &version) != nil || version <= 0 {
		return fmt.Errorf("version must be a positive integer")
	}
	var scope domain.IndustryChainScope
	if json.Unmarshal(fields["scope_type"], &scope) != nil || (scope != domain.IndustryChainScopeGlobal && scope != domain.IndustryChainScopeEconomy && scope != domain.IndustryChainScopeRegional) {
		return fmt.Errorf("unsupported industry chain scope %q", scope)
	}
	var review domain.ReviewStatus
	if json.Unmarshal(fields["review_status"], &review) != nil || (review != domain.ReviewStatusCandidate && review != domain.ReviewStatusReviewed && review != domain.ReviewStatusApproved) {
		return fmt.Errorf("unsupported industry chain review status %q", review)
	}
	return nil
}
