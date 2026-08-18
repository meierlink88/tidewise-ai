package research

import (
	"context"
	"errors"
	"fmt"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	researchapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/research"
	researchbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/research"
)

func researchResourceLimitDetails(
	component string,
	actualRows *int64,
	maxRows *int64,
	actualBytes *int64,
	maxBytes *int64,
	retryGuidance string,
) researchapi.ResearchResourceLimitDetails {
	return researchapi.ResearchResourceLimitDetails{
		Component:     component,
		ActualRows:    actualRows,
		MaxRows:       maxRows,
		ActualBytes:   actualBytes,
		MaxBytes:      maxBytes,
		RetryGuidance: retryGuidance,
	}
}

func researchGraphEntityDTOs(values []researchbiz.GraphEntity) []researchapi.ResearchGraphEntity {
	result := make([]researchapi.ResearchGraphEntity, 0, len(values))
	for _, item := range values {
		result = append(result, researchapi.ResearchGraphEntity{
			EntityID: item.EntityID, EntityType: item.EntityType, Name: item.Name,
			CanonicalName: item.CanonicalName, Aliases: item.Aliases, Status: item.Status,
		})
	}
	return result
}

func researchGraphRelationDTOs(values []researchbiz.GraphRelationDefinition) []researchapi.ResearchGraphRelationDefinition {
	result := make([]researchapi.ResearchGraphRelationDefinition, 0, len(values))
	for _, item := range values {
		result = append(result, researchapi.ResearchGraphRelationDefinition{RelationType: item.RelationType, Direction: item.Direction})
	}
	return result
}

func researchGraphEntityRelationDTOs(values []researchbiz.GraphEntityRelation) []researchapi.ResearchGraphEntityRelation {
	result := make([]researchapi.ResearchGraphEntityRelation, 0, len(values))
	for _, item := range values {
		result = append(result, researchapi.ResearchGraphEntityRelation{
			EntityRelationID: item.EntityRelationID, FromEntityID: item.FromEntityID,
			ToEntityID: item.ToEntityID, RelationType: item.RelationType, Status: item.Status,
		})
	}
	return result
}

func researchGraphIndustryChainDTOs(values []researchbiz.GraphIndustryChain) []researchapi.ResearchGraphIndustryChain {
	result := make([]researchapi.ResearchGraphIndustryChain, 0, len(values))
	for _, item := range values {
		result = append(result, researchapi.ResearchGraphIndustryChain{
			IndustryChainID: item.IndustryChainID, Scope: item.Scope,
			TargetOutput: item.TargetOutput, EndUse: item.EndUse, Geography: item.Geography,
			AsOfDate: item.AsOfDate, ReviewStatus: item.ReviewStatus,
		})
	}
	return result
}

func researchGraphMembershipDTOs(values []researchbiz.GraphIndustryChainMembership) []researchapi.ResearchGraphIndustryChainMembership {
	result := make([]researchapi.ResearchGraphIndustryChainMembership, 0, len(values))
	for _, item := range values {
		result = append(result, researchapi.ResearchGraphIndustryChainMembership{
			IndustryChainID: item.IndustryChainID, ChainNodeID: item.ChainNodeID,
			Position: item.Position, ContextualStage: item.ContextualStage,
			ReviewStatus: item.ReviewStatus, Status: item.Status,
		})
	}
	return result
}

func researchGraphIndustryEdgeDTOs(values []researchbiz.GraphIndustryChainEdge) []researchapi.ResearchGraphIndustryChainGraphEdge {
	result := make([]researchapi.ResearchGraphIndustryChainGraphEdge, 0, len(values))
	for _, item := range values {
		result = append(result, researchapi.ResearchGraphIndustryChainGraphEdge{
			IndustryChainGraphEdgeID: item.IndustryChainGraphEdgeID,
			IndustryChainID:          item.IndustryChainID, FromChainNodeID: item.FromChainNodeID,
			ToChainNodeID: item.ToChainNodeID, RelationType: item.RelationType,
			Mechanism: item.Mechanism, ConditionNote: item.ConditionNote, SegmentKind: item.SegmentKind,
			OmittedStepNote: item.OmittedStepNote, ReviewStatus: item.ReviewStatus, Status: item.Status,
		})
	}
	return result
}

type UseCase interface {
	PublishSnapshot(context.Context, string, researchbiz.SnapshotAggregate) (researchbiz.Result, error)
	ListThemes(context.Context, researchbiz.ResearchListRequest) (researchbiz.ResearchThemePage, error)
	GetTheme(context.Context, string, researchbiz.ResearchDetailRequest) (researchbiz.ResearchThemeDetail, error)
	ListReasoningTrees(context.Context, string) (researchbiz.ResearchReasoningTreeList, error)
	GetReasoningTree(context.Context, string, string) (researchbiz.ResearchReasoningTreeDetail, error)
	Search(context.Context, researchbiz.GraphSearchRequest) (researchbiz.GraphSearchResult, error)
}

type Service struct{ useCase UseCase }

func NewService(useCase UseCase) (*Service, error) {
	if useCase == nil {
		return nil, errors.New("Research use case is required")
	}
	return &Service{useCase: useCase}, nil
}

var _ researchapi.Service = (*Service)(nil)

func researchThemeImportInput(request *researchapi.ResearchThemeImportRequest) researchbiz.SnapshotAggregate {
	theme := request.Theme
	impacts := make([]researchbiz.SnapshotImpact, 0, len(theme.Impacts))
	for _, impact := range theme.Impacts {
		impacts = append(impacts, researchbiz.SnapshotImpact{
			NodeKey: impact.NodeKey, DisplayName: impact.DisplayName, RelationRole: impact.RelationRole,
			ImpactDirection: impact.ImpactDirection, ImpactSummary: impact.ImpactSummary, DisplayOrder: impact.DisplayOrder,
		})
	}
	themeEvents := make([]researchbiz.SnapshotEvent, 0, len(theme.Events))
	for _, event := range theme.Events {
		themeEvents = append(themeEvents, researchbiz.SnapshotEvent{
			EventID: event.EventID, EvidenceIDs: append([]string(nil), event.EvidenceIDs...),
			EvidenceRole: event.EvidenceRole, SupportedClaim: event.SupportedClaim,
		})
	}
	trees := make([]researchbiz.SnapshotReasoningTree, 0, len(request.ReasoningTrees))
	for _, tree := range request.ReasoningTrees {
		checkpoints := make([]researchbiz.ReasonTreeCheckpoint, 0, len(tree.Checkpoints))
		for _, checkpoint := range tree.Checkpoints {
			checkpoints = append(checkpoints, researchbiz.ReasonTreeCheckpoint{Type: checkpoint.Type, Summary: checkpoint.Summary})
		}
		events := make([]researchbiz.SnapshotTreeEvent, 0, len(tree.Events))
		for _, event := range tree.Events {
			events = append(events, researchbiz.SnapshotTreeEvent{
				EventID: event.EventID, EvidenceIDs: append([]string(nil), event.EvidenceIDs...),
				EvidenceRole: event.EvidenceRole, DisplayOrder: event.DisplayOrder,
			})
		}
		nodes := make([]researchbiz.SnapshotNode, 0, len(tree.Nodes))
		for _, node := range tree.Nodes {
			signals := make([]researchbiz.SnapshotSignal, 0, len(node.Signals))
			for _, signal := range node.Signals {
				signals = append(signals, researchbiz.SnapshotSignal{
					SignalKey: signal.SignalKey, DisplaySummary: signal.DisplaySummary, Role: signal.Role,
					DisplayOrder: signal.DisplayOrder, VariableName: signal.VariableName, Direction: signal.Direction,
				})
			}
			var incoming *researchbiz.SnapshotIncomingTransmission
			if node.IncomingTransmission != nil {
				incoming = &researchbiz.SnapshotIncomingTransmission{
					Title: node.IncomingTransmission.Title, Mechanism: node.IncomingTransmission.Mechanism,
					ConditionSummary: node.IncomingTransmission.ConditionSummary,
				}
			}
			nodes = append(nodes, researchbiz.SnapshotNode{
				NodeKey: node.NodeKey, DisplayName: node.DisplayName, Position: node.Position,
				StateSummary: node.StateSummary, ImpactDirection: node.ImpactDirection,
				ImpactStrength: node.ImpactStrength, ImpactSummary: node.ImpactSummary,
				ReasoningBasisSummary: node.ReasoningBasisSummary, EvidenceGapSummary: node.EvidenceGapSummary,
				IncomingTransmission: incoming, Signals: signals,
			})
		}
		trees = append(trees, researchbiz.SnapshotReasoningTree{
			TreeKey: tree.TreeKey, DisplayName: tree.DisplayName, Title: tree.Title,
			DisplayOrder: tree.DisplayOrder, OneLineConclusion: tree.OneLineConclusion,
			FactSummary: tree.FactSummary, TransmissionSummary: tree.TransmissionSummary,
			ImpactDirection: tree.ImpactDirection, ImpactStrength: tree.ImpactStrength,
			ImpactSummary: tree.ImpactSummary, ConclusionBoundarySummary: tree.ConclusionBoundarySummary,
			SupportSummary: tree.SupportSummary, CounterSummary: tree.CounterSummary,
			InvalidationConditions: append([]string(nil), tree.InvalidationConditions...),
			Checkpoints:            checkpoints, Events: events, Nodes: nodes,
		})
	}
	return researchbiz.SnapshotAggregate{
		PublicationMode: request.PublicationMode, AnalysisBatchID: request.AnalysisBatchID,
		AnalysisAsOf: request.AnalysisAsOf, DiscoveryWindowStart: request.DiscoveryWindowStart,
		DiscoveryWindowEnd: request.DiscoveryWindowEnd,
		Theme: researchbiz.SnapshotTheme{
			ThemeKey: theme.ThemeKey, Title: theme.Title, OneLineConclusion: theme.OneLineConclusion,
			ConclusionDirection: theme.ConclusionDirection, ImpactStrength: theme.ImpactStrength,
			AttentionLevel: theme.AttentionLevel, ConclusionStatus: theme.ConclusionStatus,
			TransmissionStage: theme.TransmissionStage, InvestmentGuidanceAction: theme.InvestmentGuidanceAction,
			InvestmentGuidanceSummary: theme.InvestmentGuidanceSummary,
			TimeHorizonCategory:       theme.TimeHorizonCategory, TimeHorizonSummary: theme.TimeHorizonSummary,
			TransmissionSummary: theme.TransmissionSummary, CheckpointSummary: theme.CheckpointSummary,
			RiskSummary: theme.RiskSummary, Impacts: impacts, Events: themeEvents,
		},
		ReasoningTrees: trees,
	}
}

func researchThemeImportDTO(result researchbiz.Result) researchapi.ResearchThemeImportResult {
	return researchapi.ResearchThemeImportResult{
		ReceiptID: result.ReceiptID, AnalysisBatchID: result.AnalysisBatchID, PayloadHash: result.PayloadHash,
		ThemeID:                   result.ThemeID,
		PublicationMode:           result.PublicationMode,
		ReasoningTreeIDsByTreeKey: result.ReasoningTreeIDsByTreeKey,
		Counts: researchapi.ResearchThemeImportCounts{
			Themes: result.Counts.Themes, Impacts: result.Counts.Impacts,
			ThemeEventAssociations: result.Counts.ThemeEventAssociations,
			ReasoningTrees:         result.Counts.ReasoningTrees, Nodes: result.Counts.Nodes,
			TreeEventAssociations: result.Counts.TreeEventAssociations,
			SignalAssociations:    result.Counts.SignalAssociations, Receipts: result.Counts.Receipts,
		},
		PublishedAt: result.PublishedAt, ImportedAt: result.ImportedAt, Replayed: result.Replayed,
	}
}

func researchThemeDTO(value researchbiz.ResearchTheme) researchapi.ResearchTheme {
	impacts := make([]researchapi.ResearchThemeImpact, 0, len(value.Impacts))
	for _, impact := range value.Impacts {
		impacts = append(impacts, researchapi.ResearchThemeImpact{
			NodeKey: impact.NodeKey, DisplayName: impact.DisplayName,
			RelationRole: impact.RelationRole, ImpactDirection: impact.ImpactDirection,
			ImpactSummary: impact.ImpactSummary, DisplayOrder: impact.DisplayOrder,
		})
	}
	return researchapi.ResearchTheme{
		ID: value.ID, AnalysisBatchID: value.AnalysisBatchID, Title: value.Title,
		OneLineConclusion: value.OneLineConclusion, ConclusionDirection: value.ConclusionDirection,
		ImpactStrength: value.ImpactStrength, AttentionLevel: value.AttentionLevel,
		ConclusionStatus: value.ConclusionStatus, TransmissionStage: value.TransmissionStage,
		InvestmentGuidanceAction:  value.InvestmentGuidanceAction,
		InvestmentGuidanceSummary: value.InvestmentGuidanceSummary,
		TimeHorizonCategory:       value.TimeHorizonCategory, TimeHorizonSummary: value.TimeHorizonSummary,
		TransmissionSummary: value.TransmissionSummary, CheckpointSummary: value.CheckpointSummary,
		RiskSummary: value.RiskSummary, AnalysisAsOf: value.AnalysisAsOf,
		WindowStart: value.WindowStart, WindowEnd: value.WindowEnd, PublishedAt: value.PublishedAt,
		Impacts: impacts, EvidenceEventCount: value.EvidenceEventCount, ReasoningTreeCount: value.ReasoningTreeCount,
	}
}

func researchEventsDTO(values []researchbiz.ResearchEvent) []researchapi.ResearchEvent {
	result := make([]researchapi.ResearchEvent, 0, len(values))
	for _, event := range values {
		result = append(result, researchapi.ResearchEvent{
			EvidenceIDs: append([]string(nil), event.EvidenceIDs...),
			EventID:     event.EventID, Title: event.Title, Summary: event.Summary, EventTime: event.EventTime,
			EvidenceRole: event.EvidenceRole, SupportedClaim: event.SupportedClaim, DisplayOrder: event.DisplayOrder,
		})
	}
	return result
}

func researchThemePageDTO(value researchbiz.ResearchThemePage) researchapi.ResearchThemePage {
	items := make([]researchapi.ResearchTheme, 0, len(value.Items))
	for _, item := range value.Items {
		items = append(items, researchThemeDTO(item))
	}
	return researchapi.ResearchThemePage{
		WindowStart: value.WindowStart, WindowEnd: value.WindowEnd, AsOf: value.AsOf,
		ThemeCount: value.ThemeCount, EventCount: value.EventCount, Items: items, NextCursor: value.NextCursor,
	}
}

func researchThemeDetailDTO(value researchbiz.ResearchThemeDetail) researchapi.ResearchThemeDetail {
	return researchapi.ResearchThemeDetail{
		ThemeKey: value.ThemeKey, PublicationMode: value.PublicationMode,
		PublicationContractVersion: value.PublicationContractVersion,
		Theme:                      researchThemeDTO(value.Theme), Events: researchEventsDTO(value.Events),
	}
}

func reasoningTreeListDTO(value researchbiz.ResearchReasoningTreeList) researchapi.ResearchReasoningTreeList {
	trees := make([]researchapi.ResearchReasoningTreeSummary, 0, len(value.ReasoningTrees))
	for _, tree := range value.ReasoningTrees {
		trees = append(trees, researchapi.ResearchReasoningTreeSummary{
			TreeKey: tree.TreeKey, DisplayName: tree.DisplayName,
			ReasoningTreeID: tree.ReasoningTreeID, Title: tree.Title, DisplayOrder: tree.DisplayOrder,
			EventCount: tree.EventCount, PublishedAt: tree.PublishedAt,
		})
	}
	return researchapi.ResearchReasoningTreeList{Theme: researchThemeDTO(value.Theme), ReasoningTrees: trees}
}

func reasoningTreeDetailDTO(value researchbiz.ResearchReasoningTreeDetail) researchapi.ResearchReasoningTreeDetail {
	tree := value.ReasoningTree
	checkpoints := make([]researchapi.ResearchReasoningTreeCheckpoint, 0, len(tree.Checkpoints))
	for _, checkpoint := range tree.Checkpoints {
		checkpoints = append(checkpoints, researchapi.ResearchReasoningTreeCheckpoint{Type: checkpoint.Type, Summary: checkpoint.Summary})
	}
	nodes := make([]researchapi.ResearchReasoningTreeNode, 0, len(tree.Nodes))
	for _, node := range tree.Nodes {
		signals := make([]researchapi.ResearchReasoningTreeSignal, 0, len(node.Signals))
		for _, signal := range node.Signals {
			signals = append(signals, researchSignalDTO(signal))
		}
		nodes = append(nodes, researchapi.ResearchReasoningTreeNode{
			NodeKey: node.NodeKey, DisplayName: node.DisplayName,
			ID: node.ID, Position: node.Position,
			StateSummary: node.StateSummary, ImpactDirection: node.ImpactDirection,
			ImpactStrength: node.ImpactStrength, ImpactSummary: node.ImpactSummary,
			ReasoningBasisSummary: node.ReasoningBasisSummary, EvidenceGapSummary: node.EvidenceGapSummary,
			IncomingTransmissionTitle:     node.IncomingTransmissionTitle,
			IncomingTransmissionMechanism: node.IncomingTransmissionMechanism,
			IncomingConditionSummary:      node.IncomingConditionSummary,
			Signals:                       signals, PrimarySignal: researchSignalDTO(node.PrimarySignal),
			SignalDisplaySummary: node.SignalDisplaySummary,
		})
	}
	return researchapi.ResearchReasoningTreeDetail{
		ThemeID: value.ThemeID, ThemeKey: value.ThemeKey, PublicationMode: value.PublicationMode,
		PublicationContractVersion: value.PublicationContractVersion,
		ImpactNodeIDs:              append([]string(nil), value.ImpactNodeIDs...),
		ReasoningTree: researchapi.ResearchReasoningTree{
			TreeKey: tree.TreeKey, DisplayName: tree.DisplayName,
			ReasoningTreeID: tree.ReasoningTreeID, ThemeID: tree.ThemeID,
			Title: tree.Title, DisplayOrder: tree.DisplayOrder, OneLineConclusion: tree.OneLineConclusion,
			FactSummary: tree.FactSummary, TransmissionSummary: tree.TransmissionSummary,
			ImpactDirection: tree.ImpactDirection, ImpactStrength: tree.ImpactStrength,
			ImpactSummary: tree.ImpactSummary, ConclusionBoundarySummary: tree.ConclusionBoundarySummary,
			SupportSummary: tree.SupportSummary, CounterSummary: tree.CounterSummary,
			InvalidationConditions: append([]string(nil), tree.InvalidationConditions...),
			Checkpoints:            checkpoints, PublishedAt: tree.PublishedAt, EventCount: tree.EventCount,
			Events: researchEventsDTO(tree.Events), Nodes: nodes,
		},
	}
}

func researchSignalDTO(signal researchbiz.ResearchSignal) researchapi.ResearchReasoningTreeSignal {
	return researchapi.ResearchReasoningTreeSignal{
		SignalKey: signal.SignalKey, VariableName: signal.VariableName, Direction: signal.Direction,
		SignalRole: signal.SignalRole, DisplaySummary: signal.DisplaySummary,
		DisplayOrder: signal.DisplayOrder,
	}
}
func (s *Service) SearchResearchGraph(
	ctx context.Context,
	request *researchapi.ResearchGraphSearchRequest,
) (*v1.Response[researchapi.ResearchGraphSearchResult], error) {
	if s == nil || s.useCase == nil {
		return nil, publicError(
			v1.StatusServiceUnavailable,
			"RESEARCH_GRAPH_NOT_READY",
			"Research Graph service is unavailable",
		)
	}
	filters := make([]researchbiz.RelationFilter, 0, len(request.RelationFilters))
	for _, filter := range request.RelationFilters {
		filters = append(filters, researchbiz.RelationFilter{
			RelationType: filter.RelationType,
			Direction:    researchbiz.Direction(filter.Direction),
		})
	}
	result, err := s.useCase.Search(ctx, researchbiz.GraphSearchRequest{
		AnalysisAsOf:    request.AnalysisAsOf,
		SeedEntityIDs:   request.SeedEntityIDs,
		RelationFilters: filters,
		MaxDepth:        request.MaxDepth,
		IndustryChainID: request.IndustryChainID,
		NodeBudget:      request.NodeBudget,
		EdgeBudget:      request.EdgeBudget,
	})
	if err != nil {
		var validation *researchbiz.GraphValidationError
		if errors.As(err, &validation) {
			return nil, publicError(
				v1.StatusBadRequest,
				"RESEARCH_GRAPH_INVALID",
				validation.Reason,
			)
		}
		var resourceLimit *researchbiz.ResearchResourceLimitError
		if errors.As(err, &resourceLimit) {
			return nil, publicErrorWithDetails(
				v1.StatusTooManyRequests,
				"RESEARCH_GRAPH_RESOURCE_LIMIT",
				resourceLimit.Reason,
				researchResourceLimitDetails(
					resourceLimit.Component,
					resourceLimit.ActualRows,
					resourceLimit.MaxRows,
					resourceLimit.ActualBytes,
					resourceLimit.MaxBytes,
					resourceLimit.RetryGuidance,
				),
			)
		}
		return nil, publicError(
			v1.StatusInternalServerError,
			"RESEARCH_GRAPH_FAILED",
			"Research Graph query failed",
		)
	}
	dto := researchapi.ResearchGraphSearchResult{
		ContractVersion: result.ContractVersion, AnalysisAsOf: result.AnalysisAsOf,
		QueryFingerprint: result.QueryFingerprint, GraphFingerprint: result.GraphFingerprint,
		ActualDepth: result.ActualDepth, Entities: researchGraphEntityDTOs(result.Entities),
		RelationDefinitions:      researchGraphRelationDTOs(result.RelationDefinitions),
		EntityRelations:          researchGraphEntityRelationDTOs(result.EntityRelations),
		IndustryChains:           researchGraphIndustryChainDTOs(result.IndustryChains),
		IndustryChainMemberships: researchGraphMembershipDTOs(result.IndustryChainMemberships),
		IndustryChainGraphEdges:  researchGraphIndustryEdgeDTOs(result.IndustryChainGraphEdges),
	}
	return &v1.Response[researchapi.ResearchGraphSearchResult]{
		Status: v1.StatusOK,
		Result: dto,
	}, nil
}
func publicError(status int, code, message string) error {
	return v1.NewPublicError(status, code, message, nil)
}

func publicErrorWithDetails(status int, code, message string, details any) error {
	return v1.NewPublicError(status, code, message, details)
}

func optionalUTC(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	_, offset := value.Zone()
	if offset != 0 {
		return nil, fmt.Errorf("timestamp is not UTC")
	}
	value = value.UTC()
	return &value, nil
}

func formatOptionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339Nano)
	return &formatted
}

func principalIdentity(ctx context.Context) string {
	if principal, ok := v1.PrincipalFromContext(ctx); ok {
		return principal.Identity
	}
	return ""
}
func (s *Service) PublishResearchTheme(ctx context.Context, request *researchapi.ResearchThemeImportRequest) (*v1.Response[researchapi.ResearchThemeImportResult], error) {
	if s == nil || s.useCase == nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research Theme import service is unavailable")
	}
	result, err := s.useCase.PublishSnapshot(ctx, principalIdentity(ctx), researchThemeImportInput(request))
	if err != nil {
		return nil, researchThemeImportError(err)
	}
	status := v1.StatusCreated
	if result.Replayed {
		status = v1.StatusOK
	}
	return &v1.Response[researchapi.ResearchThemeImportResult]{Status: status, Result: researchThemeImportDTO(result)}, nil
}

func researchThemeImportError(err error) error {
	var validation *researchbiz.ValidationError
	if errors.As(err, &validation) {
		return publicErrorWithDetails(v1.StatusUnprocessableEntity, "RESEARCH_THEME_IMPORT_REJECTED", "research Theme aggregate failed validation", map[string]any{
			"path": validation.Path, "reference": validation.Reference,
		})
	}
	var reference *researchbiz.ReferenceError
	if errors.As(err, &reference) {
		return publicErrorWithDetails(v1.StatusUnprocessableEntity, "RESEARCH_THEME_REFERENCE_INVALID", "research Theme aggregate references unavailable or inconsistent Data records", map[string]any{
			"path": reference.Path, "reference": reference.Reference,
		})
	}
	switch {
	case errors.Is(err, researchbiz.ErrPayloadConflict):
		return publicError(v1.StatusConflict, "RESEARCH_THEME_PAYLOAD_CONFLICT", "analysis_batch_id conflicts with the published payload")
	case errors.Is(err, researchbiz.ErrPublisherConflict):
		return publicError(v1.StatusConflict, "RESEARCH_THEME_PUBLISHER_CONFLICT", "analysis_batch_id belongs to another publisher subject")
	default:
		return publicError(v1.StatusInternalServerError, "RESEARCH_THEME_IMPORT_FAILED", "research Theme import failed")
	}
}
func (s *Service) ListResearchThemes(ctx context.Context, request *researchapi.ListResearchThemesRequest) (*v1.Response[researchapi.ResearchThemePage], error) {
	publishedFrom, err := optionalUTC(request.PublishedFrom)
	if err != nil {
		return nil, publicError(v1.StatusBadRequest, "INVALID_REQUEST", "published_from must be an RFC3339 UTC timestamp")
	}
	publishedTo, err := optionalUTC(request.PublishedTo)
	if err != nil {
		return nil, publicError(v1.StatusBadRequest, "INVALID_REQUEST", "published_to must be an RFC3339 UTC timestamp")
	}
	window := 0
	if publishedFrom == nil && publishedTo == nil {
		window, err = v1.ParseBoundedInt(request.WindowHours, researchbiz.DefaultResearchWindowHours, researchbiz.MinResearchWindowHours, researchbiz.MaxResearchWindowHours, "window_hours")
		if err != nil {
			return nil, err
		}
	} else if request.WindowHours != "" {
		return nil, publicError(v1.StatusBadRequest, "INVALID_REQUEST", "window_hours cannot be combined with published_from and published_to")
	}
	limit, err := v1.ParseBoundedInt(request.Limit, researchbiz.DefaultResearchLimit, 1, researchbiz.MaxResearchLimit, "limit")
	if err != nil {
		return nil, err
	}
	if s == nil || s.useCase == nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research service is unavailable")
	}
	result, err := s.useCase.ListThemes(ctx, researchbiz.ResearchListRequest{
		WindowHours: window, PublishedFrom: publishedFrom, PublishedTo: publishedTo,
		Limit: limit, Cursor: request.Cursor,
	})
	if err != nil {
		return nil, researchError(err)
	}
	return &v1.Response[researchapi.ResearchThemePage]{Status: v1.StatusOK, Result: researchThemePageDTO(result)}, nil
}

func (s *Service) GetResearchTheme(ctx context.Context, request *researchapi.GetResearchThemeRequest) (*v1.Response[researchapi.ResearchThemeDetail], error) {
	window, err := v1.ParseBoundedInt(request.WindowHours, researchbiz.DefaultResearchWindowHours, researchbiz.MinResearchWindowHours, researchbiz.MaxResearchWindowHours, "window_hours")
	if err != nil {
		return nil, err
	}
	if s == nil || s.useCase == nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research service is unavailable")
	}
	result, err := s.useCase.GetTheme(ctx, request.ThemeID, researchbiz.ResearchDetailRequest{WindowHours: window})
	if err != nil {
		return nil, researchError(err)
	}
	return &v1.Response[researchapi.ResearchThemeDetail]{Status: v1.StatusOK, Result: researchThemeDetailDTO(result)}, nil
}

func (s *Service) ListResearchReasoningTrees(ctx context.Context, request *researchapi.ReasoningTreeListRequest) (*v1.Response[researchapi.ResearchReasoningTreeList], error) {
	if request.HasQuery {
		return nil, publicError(v1.StatusBadRequest, "INVALID_REQUEST", "reasoning tree list does not accept query parameters")
	}
	if s == nil || s.useCase == nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research service is unavailable")
	}
	result, err := s.useCase.ListReasoningTrees(ctx, request.ThemeID)
	if err != nil {
		return nil, reasoningTreeError(err)
	}
	return &v1.Response[researchapi.ResearchReasoningTreeList]{Status: v1.StatusOK, Result: reasoningTreeListDTO(result)}, nil
}

func (s *Service) GetResearchReasoningTree(ctx context.Context, request *researchapi.ReasoningTreeDetailRequest) (*v1.Response[researchapi.ResearchReasoningTreeDetail], error) {
	if request.HasQuery {
		return nil, publicError(v1.StatusBadRequest, "INVALID_REQUEST", "reasoning tree detail does not accept query parameters")
	}
	if s == nil || s.useCase == nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research service is unavailable")
	}
	result, err := s.useCase.GetReasoningTree(ctx, request.ThemeID, request.ReasoningTreeID)
	if err != nil {
		return nil, reasoningTreeError(err)
	}
	return &v1.Response[researchapi.ResearchReasoningTreeDetail]{Status: v1.StatusOK, Result: reasoningTreeDetailDTO(result)}, nil
}

func researchError(err error) error {
	switch {
	case errors.Is(err, researchbiz.ErrInvalidRequest):
		return publicError(v1.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case errors.Is(err, researchbiz.ErrNotFound):
		return publicError(v1.StatusNotFound, "NOT_FOUND", "research aggregate was not found")
	default:
		return publicError(v1.StatusInternalServerError, "DATA_REPOSITORY_FAILURE", "research aggregate failed")
	}
}

func reasoningTreeError(err error) error {
	switch {
	case errors.Is(err, researchbiz.ErrInvalidRequest):
		return publicError(v1.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case errors.Is(err, researchbiz.ErrThemeNotFound):
		return publicError(v1.StatusNotFound, "RESEARCH_THEME_NOT_FOUND", "research Theme was not found")
	case errors.Is(err, researchbiz.ErrReasoningTreesNotFound):
		return publicError(v1.StatusNotFound, "RESEARCH_REASONING_TREES_NOT_FOUND", "research Theme has no published reasoning trees")
	case errors.Is(err, researchbiz.ErrReasoningTreeNotFound):
		return publicError(v1.StatusNotFound, "RESEARCH_REASONING_TREE_NOT_FOUND", "research reasoning tree was not found for the Theme")
	case errors.Is(err, researchbiz.ErrReasoningTreeInvariantViolation):
		return publicError(v1.StatusInternalServerError, "RESEARCH_REASONING_TREE_INVARIANT_VIOLATION", "published research reasoning tree data is incomplete")
	default:
		return publicError(v1.StatusInternalServerError, "DATA_REPOSITORY_FAILURE", "research reasoning tree failed")
	}
}
