package usecase

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/dataclient"
)

var (
	ErrResearchThemeNotFound          = errors.New("research Theme not found")
	ErrResearchReasoningTreesNotFound = errors.New("research reasoning trees not found")
	ErrResearchReasoningTreeNotFound  = errors.New("research reasoning tree not found")
	ErrResearchDataUnavailable        = errors.New("research data unavailable")
)

var reasoningResourceUUIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type ResearchReasoningTreeListResponse struct {
	Theme          ResearchThemeItem                 `json:"theme"`
	ReasoningTrees []ResearchReasoningTreeSummaryDTO `json:"reasoning_trees"`
}

type ResearchReasoningTreeSummaryDTO struct {
	AnchorID        string                            `json:"anchor_id"`
	CenterChainNode ResearchReasoningTreeChainNodeDTO `json:"center_chain_node"`
}

type ResearchReasoningTreeChainNodeDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ResearchReasoningTreeDetailResponse struct {
	ThemeID       string                   `json:"theme_id"`
	ReasoningTree ResearchReasoningTreeDTO `json:"reasoning_tree"`
}

type ResearchReasoningTreeDTO struct {
	AnchorID            string                             `json:"anchor_id"`
	CenterChainNode     ResearchReasoningTreeChainNodeDTO  `json:"center_chain_node"`
	OneLineConclusion   string                             `json:"one_line_conclusion"`
	FactSummary         string                             `json:"fact_summary"`
	NetDirectionSummary string                             `json:"net_direction_summary"`
	TradingDirection    string                             `json:"trading_direction"`
	NextCheckpoint      string                             `json:"next_checkpoint"`
	EventCount          int                                `json:"event_count"`
	Events              []ResearchReasoningTreeEventDTO    `json:"events"`
	PathNodes           []ResearchReasoningTreePathNodeDTO `json:"path_nodes"`
}

type ResearchReasoningTreeEventDTO struct {
	EventID         string  `json:"event_id"`
	Title           string  `json:"title"`
	Summary         string  `json:"summary"`
	EventTime       *string `json:"event_time"`
	EvidenceRole    string  `json:"evidence_role"`
	EvidenceSummary string  `json:"evidence_summary"`
}

type ResearchReasoningTreePathNodeDTO struct {
	ChainNodeID                   string  `json:"chain_node_id"`
	Name                          string  `json:"name"`
	ChangeDirection               string  `json:"change_direction"`
	ChangeSummary                 string  `json:"change_summary"`
	ImpactSummary                 string  `json:"impact_summary"`
	IncomingTransmissionMechanism *string `json:"incoming_transmission_mechanism"`
}

func (s *ResearchService) ListReasoningTrees(ctx context.Context, themeID string) (ResearchReasoningTreeListResponse, error) {
	if err := validateReasoningResourceID(themeID, "theme id"); err != nil {
		return ResearchReasoningTreeListResponse{}, err
	}
	if s == nil || s.client == nil {
		return ResearchReasoningTreeListResponse{}, ErrResearchDataUnavailable
	}
	result, err := s.client.ListResearchThemeReasoningTrees(ctx, themeID)
	if err != nil {
		return ResearchReasoningTreeListResponse{}, mapReasoningTreeDataError(err)
	}
	if !validReasoningTreeList(result, themeID) {
		return ResearchReasoningTreeListResponse{}, ErrResearchDataUnavailable
	}
	trees := make([]ResearchReasoningTreeSummaryDTO, 0, len(result.ReasoningTrees))
	for _, tree := range result.ReasoningTrees {
		trees = append(trees, ResearchReasoningTreeSummaryDTO{
			AnchorID: tree.AnchorID,
			CenterChainNode: ResearchReasoningTreeChainNodeDTO{
				ID: tree.CenterChainNode.ID, Name: tree.CenterChainNode.Name,
			},
		})
	}
	return ResearchReasoningTreeListResponse{Theme: themeItemDTO(result.Theme), ReasoningTrees: trees}, nil
}

func (s *ResearchService) GetReasoningTree(ctx context.Context, themeID, anchorID string) (ResearchReasoningTreeDetailResponse, error) {
	if err := validateReasoningResourceID(themeID, "theme id"); err != nil {
		return ResearchReasoningTreeDetailResponse{}, err
	}
	if err := validateReasoningResourceID(anchorID, "anchor id"); err != nil {
		return ResearchReasoningTreeDetailResponse{}, err
	}
	if s == nil || s.client == nil {
		return ResearchReasoningTreeDetailResponse{}, ErrResearchDataUnavailable
	}
	result, err := s.client.GetResearchThemeReasoningTree(ctx, themeID, anchorID)
	if err != nil {
		return ResearchReasoningTreeDetailResponse{}, mapReasoningTreeDataError(err)
	}
	if !validReasoningTreeDetail(result, themeID, anchorID) {
		return ResearchReasoningTreeDetailResponse{}, ErrResearchDataUnavailable
	}
	return ResearchReasoningTreeDetailResponse{
		ThemeID:       result.ThemeID,
		ReasoningTree: reasoningTreeDTO(result.ReasoningTree),
	}, nil
}

func validateReasoningResourceID(value, label string) error {
	if !reasoningResourceUUIDPattern.MatchString(value) {
		return fmt.Errorf("%w: %s must be a lowercase UUID", ErrInvalidResearchRequest, label)
	}
	return nil
}

func validReasoningTreeList(value dataclient.ResearchReasoningTreeList, themeID string) bool {
	if value.Theme.ID != themeID || !validReasoningTheme(value.Theme) || len(value.ReasoningTrees) == 0 {
		return false
	}
	seenAnchors := make(map[string]struct{}, len(value.ReasoningTrees))
	for _, tree := range value.ReasoningTrees {
		if !validReasoningUUID(tree.AnchorID) || !validReasoningChainNode(tree.CenterChainNode) {
			return false
		}
		if _, duplicate := seenAnchors[tree.AnchorID]; duplicate {
			return false
		}
		seenAnchors[tree.AnchorID] = struct{}{}
	}
	return true
}

func validReasoningTreeDetail(value dataclient.ResearchReasoningTreeDetail, themeID, anchorID string) bool {
	tree := value.ReasoningTree
	if value.ThemeID != themeID || tree.AnchorID != anchorID || !validReasoningChainNode(tree.CenterChainNode) {
		return false
	}
	if !allNonBlank(tree.OneLineConclusion, tree.FactSummary, tree.NetDirectionSummary, tree.TradingDirection, tree.NextCheckpoint) {
		return false
	}
	if tree.Events == nil || tree.EventCount != len(tree.Events) || len(tree.Events) == 0 || tree.PathNodes == nil || len(tree.PathNodes) < 2 {
		return false
	}
	seenEvents := make(map[string]struct{}, len(tree.Events))
	driverCount := 0
	for _, event := range tree.Events {
		if !validReasoningUUID(event.EventID) || !allNonBlank(event.Title, event.Summary, event.EvidenceSummary) || !validEvidenceRole(event.EvidenceRole) {
			return false
		}
		if _, duplicate := seenEvents[event.EventID]; duplicate {
			return false
		}
		seenEvents[event.EventID] = struct{}{}
		if event.EvidenceRole == dataclient.EvidenceRoleDriver {
			driverCount++
		}
	}
	if driverCount == 0 {
		return false
	}
	seenNodes := make(map[string]struct{}, len(tree.PathNodes))
	centerCount := 0
	for index, node := range tree.PathNodes {
		if !validReasoningUUID(node.ChainNodeID) || !allNonBlank(node.Name, node.ChangeSummary, node.ImpactSummary) || !validChangeDirection(node.ChangeDirection) {
			return false
		}
		if _, duplicate := seenNodes[node.ChainNodeID]; duplicate {
			return false
		}
		seenNodes[node.ChainNodeID] = struct{}{}
		if node.ChainNodeID == tree.CenterChainNode.ID {
			centerCount++
		}
		if index == 0 && node.IncomingTransmissionMechanism != nil {
			return false
		}
		if index > 0 && (node.IncomingTransmissionMechanism == nil || strings.TrimSpace(*node.IncomingTransmissionMechanism) == "") {
			return false
		}
	}
	return centerCount == 1
}

func validReasoningTheme(value dataclient.ResearchTheme) bool {
	if !validReasoningUUID(value.ID) || value.PublishedAt.IsZero() || value.AffectedChainNodes == nil || value.RelatedIndices == nil {
		return false
	}
	if !allNonBlank(
		value.Name,
		value.OneLineConclusion,
		value.TransmissionPath,
		value.TradingDirection,
		value.NextCheckpoint,
		value.MarketConfirmationSummary,
	) || !validImpactLevel(value.ImpactLevel) || !validTransmissionStage(value.TransmissionStage) {
		return false
	}
	for _, node := range value.AffectedChainNodes {
		if !validReasoningUUID(node.ID) || !allNonBlank(node.Name, node.ImpactSummary) || !validThemeRelationRole(node.RelationRole) {
			return false
		}
	}
	for _, index := range value.RelatedIndices {
		if !validReasoningUUID(index.ID) || !allNonBlank(index.Name, index.ImpactSummary) || !validImpactDirection(index.ImpactDirection) {
			return false
		}
	}
	return true
}

func validReasoningChainNode(value dataclient.ResearchReasoningTreeChainNode) bool {
	return validReasoningUUID(value.ID) && strings.TrimSpace(value.Name) != ""
}

func validReasoningUUID(value string) bool { return reasoningResourceUUIDPattern.MatchString(value) }

func allNonBlank(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return false
		}
	}
	return true
}

func validImpactLevel(value dataclient.ImpactLevel) bool {
	switch value {
	case dataclient.ImpactLevelHigh, dataclient.ImpactLevelFocus, dataclient.ImpactLevelWatch:
		return true
	default:
		return false
	}
}

func validTransmissionStage(value dataclient.TransmissionStage) bool {
	switch value {
	case dataclient.TransmissionStageIdentification,
		dataclient.TransmissionStageValidation,
		dataclient.TransmissionStageDiffusion,
		dataclient.TransmissionStageDampening:
		return true
	default:
		return false
	}
}

func validThemeRelationRole(value string) bool {
	switch value {
	case "driver", "beneficiary", "constraint", "exposure":
		return true
	default:
		return false
	}
}

func validImpactDirection(value dataclient.ImpactDirection) bool {
	switch value {
	case dataclient.ImpactDirectionPositive,
		dataclient.ImpactDirectionNegative,
		dataclient.ImpactDirectionMixed,
		dataclient.ImpactDirectionNeutral:
		return true
	default:
		return false
	}
}

func validEvidenceRole(value dataclient.EvidenceRole) bool {
	switch value {
	case dataclient.EvidenceRoleDriver,
		dataclient.EvidenceRoleSupporting,
		dataclient.EvidenceRoleContradicting,
		dataclient.EvidenceRoleContext:
		return true
	default:
		return false
	}
}

func validChangeDirection(value dataclient.ChangeDirection) bool {
	switch value {
	case dataclient.ChangeDirectionIncrease,
		dataclient.ChangeDirectionDecrease,
		dataclient.ChangeDirectionMixed,
		dataclient.ChangeDirectionUnchanged,
		dataclient.ChangeDirectionUncertain:
		return true
	default:
		return false
	}
}

func reasoningTreeDTO(value dataclient.ResearchReasoningTree) ResearchReasoningTreeDTO {
	return ResearchReasoningTreeDTO{
		AnchorID: value.AnchorID,
		CenterChainNode: ResearchReasoningTreeChainNodeDTO{
			ID: value.CenterChainNode.ID, Name: value.CenterChainNode.Name,
		},
		OneLineConclusion:   value.OneLineConclusion,
		FactSummary:         value.FactSummary,
		NetDirectionSummary: value.NetDirectionSummary,
		TradingDirection:    value.TradingDirection,
		NextCheckpoint:      value.NextCheckpoint,
		EventCount:          value.EventCount,
		Events:              reasoningTreeEventDTOs(value.Events),
		PathNodes:           reasoningTreePathNodeDTOs(value.PathNodes),
	}
}

func reasoningTreeEventDTOs(values []dataclient.ResearchReasoningTreeEvent) []ResearchReasoningTreeEventDTO {
	result := make([]ResearchReasoningTreeEventDTO, 0, len(values))
	for _, value := range values {
		var eventTime *string
		if value.EventTime != nil {
			formatted := formatTime(*value.EventTime)
			eventTime = &formatted
		}
		result = append(result, ResearchReasoningTreeEventDTO{
			EventID: value.EventID, Title: value.Title, Summary: value.Summary, EventTime: eventTime,
			EvidenceRole: string(value.EvidenceRole), EvidenceSummary: value.EvidenceSummary,
		})
	}
	return result
}

func reasoningTreePathNodeDTOs(values []dataclient.ResearchReasoningTreePathNode) []ResearchReasoningTreePathNodeDTO {
	result := make([]ResearchReasoningTreePathNodeDTO, 0, len(values))
	for _, value := range values {
		result = append(result, ResearchReasoningTreePathNodeDTO{
			ChainNodeID: value.ChainNodeID, Name: value.Name, ChangeDirection: string(value.ChangeDirection),
			ChangeSummary: value.ChangeSummary, ImpactSummary: value.ImpactSummary,
			IncomingTransmissionMechanism: value.IncomingTransmissionMechanism,
		})
	}
	return result
}

func mapReasoningTreeDataError(err error) error {
	var clientErr *dataclient.Error
	if errors.As(err, &clientErr) && clientErr.StatusCode == http.StatusNotFound {
		switch clientErr.Code {
		case "RESEARCH_THEME_NOT_FOUND":
			return ErrResearchThemeNotFound
		case "RESEARCH_REASONING_TREES_NOT_FOUND":
			return ErrResearchReasoningTreesNotFound
		case "RESEARCH_REASONING_TREE_NOT_FOUND":
			return ErrResearchReasoningTreeNotFound
		}
	}
	return ErrResearchDataUnavailable
}
