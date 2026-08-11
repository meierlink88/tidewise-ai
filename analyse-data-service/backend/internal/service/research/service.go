package research

import (
	"context"
	"errors"
	"fmt"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	researchapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1/research"
	researchbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
)

func (s *Service) ListResearchAnalysisContext(
	ctx context.Context,
	request *researchapi.ResearchAnalysisContextRequest,
) (*v1.Response[researchapi.ResearchAnalysisContext], error) {
	if s == nil || s.useCase == nil {
		return nil, publicError(
			v1.StatusServiceUnavailable,
			"RESEARCH_ANALYSIS_CONTEXT_NOT_READY",
			"Research Analysis Context service is unavailable",
		)
	}
	result, err := s.useCase.List(
		ctx,
		researchbiz.AnalysisContextRequest{
			DiscoveryWindowStart:   request.DiscoveryWindowStart,
			DiscoveryWindowEnd:     request.DiscoveryWindowEnd,
			AnalysisAsOf:           request.AnalysisAsOf,
			PredictionHorizonStart: request.PredictionHorizonStart,
			PredictionHorizonEnd:   request.PredictionHorizonEnd,
			PageSize:               request.PageSize,
			Cursor:                 request.Cursor,
		},
	)
	if err != nil {
		if errors.Is(err, researchbiz.ErrReferenceClosureInconsistent) {
			return nil, publicErrorWithDetails(
				v1.StatusConflict,
				"RESEARCH_ANALYSIS_CONTEXT_INCONSISTENT",
				"Research Analysis Context references changed during the live query",
				researchapi.ResearchAnalysisContextInconsistentDetails{
					RetryGuidance: "restart_from_first_page",
				},
			)
		}
		if errors.Is(err, researchbiz.ErrHistoricalSemanticsUnavailable) {
			return nil, publicError(
				v1.StatusUnprocessableEntity,
				"RESEARCH_ANALYSIS_CONTEXT_HISTORY_UNAVAILABLE",
				"strict historical Event semantics are unavailable; choose a current analysis_as_of or use a future snapshot capability",
			)
		}
		var validation *researchbiz.AnalysisContextValidationError
		if errors.As(err, &validation) {
			return nil, publicError(
				v1.StatusBadRequest,
				"RESEARCH_ANALYSIS_CONTEXT_INVALID",
				validation.Reason,
			)
		}
		var resourceLimit *researchbiz.ResearchResourceLimitError
		if errors.As(err, &resourceLimit) {
			return nil, publicErrorWithDetails(
				v1.StatusTooManyRequests,
				"RESEARCH_ANALYSIS_CONTEXT_RESOURCE_LIMIT",
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
			"RESEARCH_ANALYSIS_CONTEXT_FAILED",
			"Research Analysis Context query failed",
		)
	}
	dto := researchAnalysisContextDTO(result)
	return &v1.Response[researchapi.ResearchAnalysisContext]{Status: v1.StatusOK, Result: dto}, nil
}

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

func researchAnalysisContextDTO(
	result researchbiz.AnalysisContextResult,
) researchapi.ResearchAnalysisContext {
	return researchapi.ResearchAnalysisContext{
		ContractVersion:             result.ContractVersion,
		TBoxContractVersion:         result.TBoxContractVersion,
		TemporalSemantics:           result.TemporalSemantics,
		TemporalLimitation:          result.TemporalLimitation,
		EventPageFingerprint:        result.EventPageFingerprint,
		ReferenceClosureFingerprint: result.ReferenceClosureFingerprint,
		DiscoveryWindowStart:        result.DiscoveryWindowStart,
		DiscoveryWindowEnd:          result.DiscoveryWindowEnd,
		AnalysisAsOf:                result.AnalysisAsOf,
		PredictionHorizonStart:      result.PredictionHorizonStart,
		PredictionHorizonEnd:        result.PredictionHorizonEnd,
		EventSemanticBundles:        researchAnalysisBundlesDTO(result.EventSemanticBundles),
		Dictionaries:                researchAnalysisDictionariesDTO(result.Dictionaries),
		NextCursor:                  result.NextCursor,
		HasMore:                     result.HasMore,
	}
}

func researchAnalysisBundlesDTO(values []researchbiz.EventSemanticBundle) []researchapi.ResearchAnalysisEventSemanticBundle {
	result := make([]researchapi.ResearchAnalysisEventSemanticBundle, 0, len(values))
	for _, value := range values {
		result = append(result, researchapi.ResearchAnalysisEventSemanticBundle{
			Event: researchAnalysisEventDTO(value.Event), Evidence: researchAnalysisEvidenceDTOs(value.Evidence),
			EntityLinks:     researchAnalysisEntityLinkDTOs(value.EntityLinks),
			VariableSignals: researchAnalysisVariableSignalDTOs(value.VariableSignals),
		})
	}
	return result
}

func researchAnalysisEventDTO(value researchbiz.Event) researchapi.ResearchAnalysisEvent {
	return researchapi.ResearchAnalysisEvent{
		ID: value.ID, Title: value.Title, Summary: value.Summary, OccurredAt: value.OccurredAt,
		FirstSeenAt: value.FirstSeenAt, KnowledgeAvailableAt: value.KnowledgeAvailableAt,
		EventStatus: value.EventStatus, FactStatus: value.FactStatus,
	}
}

func researchAnalysisEvidenceDTOs(values []researchbiz.Evidence) []researchapi.ResearchAnalysisEvidence {
	result := make([]researchapi.ResearchAnalysisEvidence, 0, len(values))
	for _, value := range values {
		sourceURL := ""
		if value.SourceURL != nil {
			sourceURL = *value.SourceURL
		}
		var publishedAt *string
		if value.PublishedAt != nil {
			formatted := value.PublishedAt.Format(time.RFC3339Nano)
			publishedAt = &formatted
		}
		result = append(result, researchapi.ResearchAnalysisEvidence{
			EvidenceID: value.EvidenceID, EvidenceHash: value.EvidenceHash, Statement: value.Statement,
			SourceLevel: value.SourceLevel, Relation: value.Relation, SupportsFields: value.SupportsFields,
			RawDocumentID: value.RawDocumentID, SourceName: value.SourceName, SourceType: value.SourceType,
			SourceURL: sourceURL, Title: value.Title, PublishedAt: publishedAt,
			FirstSeenAt:          value.FirstSeenAt.Format(time.RFC3339Nano),
			KnowledgeAvailableAt: value.KnowledgeAvailableAt.Format(time.RFC3339Nano),
			AcceptedAt:           value.AcceptedAt.Format(time.RFC3339Nano), StatementSource: value.StatementSource,
		})
	}
	return result
}

func researchAnalysisEntityLinkDTOs(values []researchbiz.EntityLink) []researchapi.ResearchAnalysisEntityLink {
	result := make([]researchapi.ResearchAnalysisEntityLink, 0, len(values))
	for _, value := range values {
		result = append(result, researchapi.ResearchAnalysisEntityLink{
			EventEntityLinkID: value.EventEntityLinkID, SemanticSubmissionID: value.SemanticSubmissionID,
			EntityID: value.EntityID, EntityRole: value.EntityRole, ResolvedMention: value.ResolvedMention,
			ResolutionMethod: value.ResolutionMethod, ResolutionConfidence: value.ResolutionConfidence,
			EvidenceIDs: value.EvidenceIDs, ReviewStatus: value.ReviewStatus,
		})
	}
	return result
}

func researchAnalysisVariableSignalDTOs(values []researchbiz.VariableSignal) []researchapi.ResearchAnalysisVariableSignal {
	result := make([]researchapi.ResearchAnalysisVariableSignal, 0, len(values))
	for _, value := range values {
		result = append(result, researchapi.ResearchAnalysisVariableSignal{
			VariableSignalID: value.VariableSignalID, SemanticSubmissionID: value.SemanticSubmissionID,
			SourceEventID: value.SourceEventID, SubjectEventEntityLinkID: value.SubjectEventEntityLinkID,
			SubjectEntityID: value.SubjectEntityID, VariableKey: value.VariableKey,
			VariableVersion: value.VariableVersion, Direction: value.Direction,
			AssertionModality: value.AssertionModality, EvidenceIDs: value.EvidenceIDs,
			StatementAt: value.StatementAt, ValidFrom: value.ValidFrom, ValidUntil: value.ValidUntil,
			ForecastPeriodStart: value.ForecastPeriodStart, ForecastPeriodEnd: value.ForecastPeriodEnd,
			ExtractionConfidence: value.ExtractionConfidence, ReviewStatus: value.ReviewStatus,
			Measurements:  researchAnalysisMeasurementDTOs(value.Measurements),
			DirectImpacts: researchAnalysisDirectImpactDTOs(value.DirectImpacts),
		})
	}
	return result
}

func researchAnalysisMeasurementDTOs(values []researchbiz.Measurement) []researchapi.ResearchAnalysisMeasurement {
	result := make([]researchapi.ResearchAnalysisMeasurement, 0, len(values))
	for _, value := range values {
		result = append(result, researchapi.ResearchAnalysisMeasurement{
			MeasurementID: value.MeasurementID, MeasurementRole: value.MeasurementRole, ValueShape: value.ValueShape,
			RawValue: value.RawValue, RawLower: value.RawLower, RawUpper: value.RawUpper, RawUnit: value.RawUnit,
			CanonicalValue: value.CanonicalValue, CanonicalLower: value.CanonicalLower,
			CanonicalUpper: value.CanonicalUpper, CanonicalUnit: value.CanonicalUnit, Currency: value.Currency,
			Scale: value.Scale, ComparisonBasis: value.ComparisonBasis, ComparisonPeriod: value.ComparisonPeriod,
			RawText: value.RawText, IsApproximate: value.IsApproximate, EvidenceID: value.EvidenceID,
		})
	}
	return result
}

func researchAnalysisDirectImpactDTOs(values []researchbiz.DirectImpact) []researchapi.ResearchAnalysisDirectImpact {
	result := make([]researchapi.ResearchAnalysisDirectImpact, 0, len(values))
	for _, value := range values {
		result = append(result, researchapi.ResearchAnalysisDirectImpact{
			DirectImpactAssertionID: value.DirectImpactAssertionID, SemanticSubmissionID: value.SemanticSubmissionID,
			SourceVariableSignalID: value.SourceVariableSignalID, TargetEntityID: value.TargetEntityID,
			AffectedVariableKey: value.AffectedVariableKey, AffectedVariableVersion: value.AffectedVariableVersion,
			AffectedDirection: value.AffectedDirection, DerivationType: value.DerivationType,
			MechanismSummary: value.MechanismSummary, EvidenceIDs: value.EvidenceIDs,
			EntityRelationID: value.EntityRelationID, RuleKey: value.RuleKey, RuleVersion: value.RuleVersion,
			AssertionConfidence: value.AssertionConfidence, EffectiveFrom: value.EffectiveFrom,
			EffectiveTo: value.EffectiveTo, ReviewStatus: value.ReviewStatus,
		})
	}
	return result
}

func researchAnalysisDictionariesDTO(value researchbiz.Dictionaries) researchapi.ResearchAnalysisDictionaries {
	result := researchapi.ResearchAnalysisDictionaries{
		Entities:                 researchAnalysisEntityDTOs(value.Entities),
		RelationDefinitions:      researchAnalysisRelationDTOs(value.RelationDefinitions),
		EntityRelations:          researchAnalysisEntityRelationDTOs(value.EntityRelations),
		IndustryChains:           researchAnalysisIndustryChainDTOs(value.IndustryChains),
		IndustryChainMemberships: researchAnalysisMembershipDTOs(value.IndustryChainMemberships),
		IndustryChainGraphEdges:  researchAnalysisIndustryEdgeDTOs(value.IndustryChainGraphEdges),
		EntityTypeDefinitions:    make([]researchapi.ResearchAnalysisEntityTypeDefinition, 0, len(value.EntityTypeDefinitions)),
		VariableDefinitions:      make([]researchapi.ResearchAnalysisVariableDefinition, 0, len(value.VariableDefinitions)),
		DirectTransmissionRules:  make([]researchapi.ResearchAnalysisTransmissionRule, 0, len(value.DirectTransmissionRules)),
		AcceptancePolicies:       make([]researchapi.ResearchAnalysisAcceptancePolicy, 0, len(value.AcceptancePolicies)),
	}
	for _, item := range value.EntityTypeDefinitions {
		result.EntityTypeDefinitions = append(result.EntityTypeDefinitions, researchapi.ResearchAnalysisEntityTypeDefinition{
			TypeKey: item.TypeKey, Version: item.Version, NameZH: item.NameZH, NameEN: item.NameEN,
			BusinessDefinition: item.BusinessDefinition, InclusionCriteria: item.InclusionCriteria,
			ExclusionCriteria: item.ExclusionCriteria, EventLinkAllowed: item.EventLinkAllowed,
			SignalSubjectAllowed: item.SignalSubjectAllowed, DirectTargetMode: item.DirectTargetMode,
			Status: item.Status,
		})
	}
	for _, item := range value.VariableDefinitions {
		result.VariableDefinitions = append(result.VariableDefinitions, researchapi.ResearchAnalysisVariableDefinition{
			Key: item.Key, Version: item.Version, NameZH: item.NameZH, NameEN: item.NameEN,
			Domain: item.Domain, BusinessDefinition: item.BusinessDefinition, ValueType: item.ValueType,
			AllowedDirections: item.AllowedDirections, CanonicalUnit: item.CanonicalUnit,
			Status: item.Status, ApplicableEntityTypes: item.ApplicableEntityTypes,
		})
	}
	for _, item := range value.DirectTransmissionRules {
		result.DirectTransmissionRules = append(result.DirectTransmissionRules, researchapi.ResearchAnalysisTransmissionRule{
			RuleKey: item.RuleKey, Version: item.Version, Status: item.Status,
			SourceEntityType: item.SourceEntityType, SourceVariableKey: item.SourceVariableKey,
			SourceVariableVersion: item.SourceVariableVersion, SourceDirection: item.SourceDirection,
			RelationType: item.RelationType, TargetEntityType: item.TargetEntityType,
			AffectedVariableKey: item.AffectedVariableKey, AffectedVariableVersion: item.AffectedVariableVersion,
			AffectedDirection: item.AffectedDirection, ConditionSummary: item.ConditionSummary,
			MechanismTemplate: item.MechanismTemplate,
		})
	}
	for _, item := range value.AcceptancePolicies {
		result.AcceptancePolicies = append(result.AcceptancePolicies, researchapi.ResearchAnalysisAcceptancePolicy{
			PolicyKey: item.PolicyKey, Version: item.Version, RetryBudget: item.RetryBudget,
			Status: item.Status, Policy: append([]byte(nil), item.Policy...),
		})
	}
	return result
}

func researchAnalysisEntityDTOs(values []researchbiz.Entity) []researchapi.ResearchAnalysisEntity {
	result := make([]researchapi.ResearchAnalysisEntity, 0, len(values))
	for _, item := range values {
		result = append(result, researchapi.ResearchAnalysisEntity{
			EntityID: item.EntityID, EntityType: item.EntityType, Name: item.Name,
			CanonicalName: item.CanonicalName, Aliases: item.Aliases, Status: item.Status,
		})
	}
	return result
}

func researchAnalysisRelationDTOs(values []researchbiz.RelationDefinition) []researchapi.ResearchAnalysisRelationDefinition {
	result := make([]researchapi.ResearchAnalysisRelationDefinition, 0, len(values))
	for _, item := range values {
		result = append(result, researchapi.ResearchAnalysisRelationDefinition{RelationType: item.RelationType, Direction: item.Direction})
	}
	return result
}

func researchAnalysisEntityRelationDTOs(values []researchbiz.EntityRelation) []researchapi.ResearchAnalysisEntityRelation {
	result := make([]researchapi.ResearchAnalysisEntityRelation, 0, len(values))
	for _, item := range values {
		result = append(result, researchapi.ResearchAnalysisEntityRelation{
			EntityRelationID: item.EntityRelationID, FromEntityID: item.FromEntityID,
			ToEntityID: item.ToEntityID, RelationType: item.RelationType, Status: item.Status,
		})
	}
	return result
}

func researchAnalysisIndustryChainDTOs(values []researchbiz.IndustryChain) []researchapi.ResearchAnalysisIndustryChain {
	result := make([]researchapi.ResearchAnalysisIndustryChain, 0, len(values))
	for _, item := range values {
		result = append(result, researchapi.ResearchAnalysisIndustryChain{
			IndustryChainEntityID: item.IndustryChainEntityID, Scope: item.Scope,
			TargetOutput: item.TargetOutput, EndUse: item.EndUse, Geography: item.Geography,
			AsOfDate: item.AsOfDate, ReviewStatus: item.ReviewStatus,
		})
	}
	return result
}

func researchAnalysisMembershipDTOs(values []researchbiz.IndustryChainMembership) []researchapi.ResearchAnalysisIndustryChainMembership {
	result := make([]researchapi.ResearchAnalysisIndustryChainMembership, 0, len(values))
	for _, item := range values {
		result = append(result, researchapi.ResearchAnalysisIndustryChainMembership{
			IndustryChainEntityID: item.IndustryChainEntityID, ChainNodeEntityID: item.ChainNodeEntityID,
			Position: item.Position, ContextualStage: item.ContextualStage,
			ReviewStatus: item.ReviewStatus, Status: item.Status,
		})
	}
	return result
}

func researchAnalysisIndustryEdgeDTOs(values []researchbiz.IndustryChainGraphEdge) []researchapi.ResearchAnalysisIndustryChainGraphEdge {
	result := make([]researchapi.ResearchAnalysisIndustryChainGraphEdge, 0, len(values))
	for _, item := range values {
		result = append(result, researchapi.ResearchAnalysisIndustryChainGraphEdge{
			IndustryChainGraphEdgeID: item.IndustryChainGraphEdgeID,
			IndustryChainEntityID:    item.IndustryChainEntityID, FromChainNodeEntityID: item.FromChainNodeEntityID,
			ToChainNodeEntityID: item.ToChainNodeEntityID, RelationType: item.RelationType,
			Mechanism: item.Mechanism, ConditionNote: item.ConditionNote, SegmentKind: item.SegmentKind,
			OmittedStepNote: item.OmittedStepNote, ReviewStatus: item.ReviewStatus, Status: item.Status,
		})
	}
	return result
}

type UseCase interface {
	Publish(context.Context, string, researchbiz.Aggregate) (researchbiz.Result, error)
	PublishSnapshot(context.Context, string, researchbiz.SnapshotAggregate) (researchbiz.Result, error)
	ListThemes(context.Context, researchbiz.ResearchListRequest) (researchbiz.ResearchThemePage, error)
	GetTheme(context.Context, string, researchbiz.ResearchDetailRequest) (researchbiz.ResearchThemeDetail, error)
	ListReasoningTrees(context.Context, string) (researchbiz.ResearchReasoningTreeList, error)
	GetReasoningTree(context.Context, string, string) (researchbiz.ResearchReasoningTreeDetail, error)
	List(context.Context, researchbiz.AnalysisContextRequest) (researchbiz.AnalysisContextResult, error)
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

func researchThemeImportInput(request *researchapi.ResearchThemeImportRequest) researchbiz.Aggregate {
	theme := request.Theme
	impacts := make([]researchbiz.ThemeImpactInput, 0, len(theme.Impacts))
	for _, impact := range theme.Impacts {
		impacts = append(impacts, researchbiz.ThemeImpactInput{
			ChainNodeEntityID: impact.ChainNodeEntityID, RelationRole: impact.RelationRole,
			ImpactDirection: impact.ImpactDirection, ImpactSummary: impact.ImpactSummary,
			DisplayOrder: impact.DisplayOrder,
		})
	}
	themeEvents := make([]researchbiz.ThemeEventInput, 0, len(theme.Events))
	for _, event := range theme.Events {
		themeEvents = append(themeEvents, researchbiz.ThemeEventInput{
			EventID: event.EventID, EvidenceRole: event.EvidenceRole, SupportedClaim: event.SupportedClaim,
		})
	}
	themeInput := researchbiz.ThemeInput{
		ThemeKey: theme.ThemeKey, Title: theme.Title, OneLineConclusion: theme.OneLineConclusion,
		ConclusionDirection: theme.ConclusionDirection, ImpactStrength: theme.ImpactStrength,
		AttentionLevel: theme.AttentionLevel, ConclusionStatus: theme.ConclusionStatus,
		TransmissionStage: theme.TransmissionStage, InvestmentGuidanceAction: theme.InvestmentGuidanceAction,
		InvestmentGuidanceSummary: theme.InvestmentGuidanceSummary,
		TimeHorizonCategory:       theme.TimeHorizonCategory, TimeHorizonSummary: theme.TimeHorizonSummary,
		TransmissionSummary: theme.TransmissionSummary, CheckpointSummary: theme.CheckpointSummary,
		RiskSummary: theme.RiskSummary, Impacts: impacts, Events: themeEvents,
	}
	trees := make([]researchbiz.ReasoningTree, 0, len(request.ReasoningTrees))
	for _, tree := range request.ReasoningTrees {
		checkpoints := make([]researchbiz.ReasonTreeCheckpoint, 0, len(tree.Checkpoints))
		for _, checkpoint := range tree.Checkpoints {
			checkpoints = append(checkpoints, researchbiz.ReasonTreeCheckpoint{
				Type: checkpoint.Type, Summary: checkpoint.Summary,
			})
		}
		events := make([]researchbiz.ReasonTreeEventInput, 0, len(tree.Events))
		for _, event := range tree.Events {
			events = append(events, researchbiz.ReasonTreeEventInput{
				EventID: event.EventID, EvidenceRole: event.EvidenceRole, DisplayOrder: event.DisplayOrder,
			})
		}
		nodes := make([]researchbiz.Node, 0, len(tree.Nodes))
		for _, node := range tree.Nodes {
			signals := make([]researchbiz.Signal, 0, len(node.Signals))
			for _, signal := range node.Signals {
				signals = append(signals, researchbiz.Signal{
					VariableSignalKey: signal.VariableSignalKey, SignalRole: signal.SignalRole,
					SignalDirection: signal.SignalDirection, DisplaySummary: signal.DisplaySummary,
					DisplayOrder: signal.DisplayOrder,
					Lineage: researchbiz.SignalLineage{
						SourceKind:           signal.Lineage.SourceKind,
						VariableSignalID:     signal.Lineage.VariableSignalID,
						SemanticSubmissionID: signal.Lineage.SemanticSubmissionID,
						EvidenceID:           signal.Lineage.EvidenceID, EvidenceHash: signal.Lineage.EvidenceHash,
						UpstreamVariableSignalID:        signal.Lineage.UpstreamVariableSignalID,
						UpstreamDirectImpactAssertionID: signal.Lineage.UpstreamDirectImpactAssertionID,
						EntityRelationID:                signal.Lineage.EntityRelationID,
						IndustryChainGraphEdgeID:        signal.Lineage.IndustryChainGraphEdgeID,
					},
				})
			}
			var incoming *researchbiz.IncomingLineage
			if node.IncomingLineage != nil {
				incoming = &researchbiz.IncomingLineage{
					SourceKind:                      node.IncomingLineage.SourceKind,
					DirectImpactAssertionID:         node.IncomingLineage.DirectImpactAssertionID,
					SemanticSubmissionID:            node.IncomingLineage.SemanticSubmissionID,
					EvidenceID:                      node.IncomingLineage.EvidenceID,
					EvidenceHash:                    node.IncomingLineage.EvidenceHash,
					AffectedVariableKey:             node.IncomingLineage.AffectedVariableKey,
					AffectedDirection:               node.IncomingLineage.AffectedDirection,
					UpstreamVariableSignalID:        node.IncomingLineage.UpstreamVariableSignalID,
					UpstreamDirectImpactAssertionID: node.IncomingLineage.UpstreamDirectImpactAssertionID,
					EntityRelationID:                node.IncomingLineage.EntityRelationID,
				}
			}
			nodes = append(nodes, researchbiz.Node{
				Position: node.Position, ChainNodeEntityID: node.ChainNodeEntityID,
				StateSummary: node.StateSummary, ImpactDirection: node.ImpactDirection,
				ImpactStrength: node.ImpactStrength, ImpactSummary: node.ImpactSummary,
				ReasoningBasisSummary: node.ReasoningBasisSummary, EvidenceGapSummary: node.EvidenceGapSummary,
				IncomingIndustryChainGraphEdgeID: node.IncomingIndustryChainGraphEdgeID,
				IncomingTransmissionTitle:        node.IncomingTransmissionTitle,
				IncomingTransmissionMechanism:    node.IncomingTransmissionMechanism,
				IncomingConditionSummary:         node.IncomingConditionSummary,
				IncomingLineage:                  incoming, Signals: signals,
			})
		}
		trees = append(trees, researchbiz.ReasoningTree{
			ReasonTreeInput: researchbiz.ReasonTreeInput{
				IndustryChainEntityID: tree.IndustryChainEntityID, Title: tree.Title, DisplayOrder: tree.DisplayOrder,
				OneLineConclusion: tree.OneLineConclusion, FactSummary: tree.FactSummary,
				TransmissionSummary: tree.TransmissionSummary, ImpactDirection: tree.ImpactDirection,
				ImpactStrength: tree.ImpactStrength, ImpactSummary: tree.ImpactSummary,
				ConclusionBoundarySummary: tree.ConclusionBoundarySummary, SupportSummary: tree.SupportSummary,
				CounterSummary: tree.CounterSummary, InvalidationConditions: tree.InvalidationConditions,
				Checkpoints: checkpoints, Events: events,
			},
			Nodes: nodes,
		})
	}
	return researchbiz.Aggregate{
		AnalysisBatchID: request.AnalysisBatchID, AnalysisAsOf: request.AnalysisAsOf,
		DiscoveryWindowStart: request.DiscoveryWindowStart,
		DiscoveryWindowEnd:   request.DiscoveryWindowEnd,
		Theme:                themeInput, ReasoningTrees: trees,
	}
}

func researchThemeSnapshotImportInput(request *researchapi.ResearchThemeSnapshotImportRequest) researchbiz.SnapshotAggregate {
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
		ThemeID:                                 result.ThemeID,
		PublicationMode:                         result.PublicationMode,
		ReasoningTreeIDsByIndustryChainEntityID: result.ReasoningTreeIDsByIndustryChainEntityID,
		ReasoningTreeIDsByTreeKey:               result.ReasoningTreeIDsByTreeKey,
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
			ChainNodeEntityID: impact.ChainNodeEntityID, Name: impact.Name,
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
			ReasoningTreeID: tree.ReasoningTreeID, IndustryChainEntityID: tree.IndustryChainEntityID,
			IndustryChainName: tree.IndustryChainName, Title: tree.Title, DisplayOrder: tree.DisplayOrder,
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
		var graphEdge *researchapi.ResearchReasoningTreeGraphEdge
		if node.IncomingGraphEdge != nil {
			graphEdge = &researchapi.ResearchReasoningTreeGraphEdge{
				ID: node.IncomingGraphEdge.ID, RelationType: node.IncomingGraphEdge.RelationType,
				ReviewStatus: node.IncomingGraphEdge.ReviewStatus, Status: node.IncomingGraphEdge.Status,
			}
		}
		nodes = append(nodes, researchapi.ResearchReasoningTreeNode{
			NodeKey: node.NodeKey, DisplayName: node.DisplayName,
			ID: node.ID, Position: node.Position, ChainNodeEntityID: node.ChainNodeEntityID, Name: node.Name,
			StateSummary: node.StateSummary, ImpactDirection: node.ImpactDirection,
			ImpactStrength: node.ImpactStrength, ImpactSummary: node.ImpactSummary,
			ReasoningBasisSummary: node.ReasoningBasisSummary, EvidenceGapSummary: node.EvidenceGapSummary,
			IncomingIndustryChainGraphEdgeID: node.IncomingIndustryChainGraphEdgeID,
			IncomingTransmissionTitle:        node.IncomingTransmissionTitle,
			IncomingTransmissionMechanism:    node.IncomingTransmissionMechanism,
			IncomingConditionSummary:         node.IncomingConditionSummary, IncomingGraphEdge: graphEdge,
			Signals: signals, PrimarySignal: researchSignalDTO(node.PrimarySignal),
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
			IndustryChainEntityID: tree.IndustryChainEntityID, IndustryChainName: tree.IndustryChainName,
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
		VariableSignalKey: signal.VariableSignalKey, SignalRole: signal.SignalRole,
		SignalDirection: signal.SignalDirection, DisplaySummary: signal.DisplaySummary,
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
		AnalysisAsOf:          request.AnalysisAsOf,
		SeedEntityIDs:         request.SeedEntityIDs,
		RelationFilters:       filters,
		MaxDepth:              request.MaxDepth,
		IndustryChainEntityID: request.IndustryChainEntityID,
		NodeBudget:            request.NodeBudget,
		EdgeBudget:            request.EdgeBudget,
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
		ActualDepth: result.ActualDepth, Entities: researchAnalysisEntityDTOs(result.Entities),
		RelationDefinitions:      researchAnalysisRelationDTOs(result.RelationDefinitions),
		EntityRelations:          researchAnalysisEntityRelationDTOs(result.EntityRelations),
		IndustryChains:           researchAnalysisIndustryChainDTOs(result.IndustryChains),
		IndustryChainMemberships: researchAnalysisMembershipDTOs(result.IndustryChainMemberships),
		IndustryChainGraphEdges:  researchAnalysisIndustryEdgeDTOs(result.IndustryChainGraphEdges),
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
	var result researchbiz.Result
	var err error
	if request.PublicationMode == researchbiz.SnapshotPublicationMode && request.Snapshot != nil {
		result, err = s.useCase.PublishSnapshot(ctx, principalIdentity(ctx), researchThemeSnapshotImportInput(request.Snapshot))
	} else {
		result, err = s.useCase.Publish(ctx, principalIdentity(ctx), researchThemeImportInput(request))
	}
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
	var themeValidation *researchbiz.ThemeValidationError
	if errors.As(err, &themeValidation) {
		return publicErrorWithDetails(v1.StatusUnprocessableEntity, "RESEARCH_THEME_IMPORT_REJECTED", "research Theme aggregate failed validation", map[string]any{
			"path": themeValidation.Path, "reference": themeValidation.Reference,
		})
	}
	var treeValidation *researchbiz.ReasonTreeValidationError
	if errors.As(err, &treeValidation) {
		return publicErrorWithDetails(v1.StatusUnprocessableEntity, "RESEARCH_THEME_IMPORT_REJECTED", "research Theme aggregate failed validation", map[string]any{
			"path": treeValidation.Path, "reference": treeValidation.Reference,
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
