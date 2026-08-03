package service

import (
	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchpublication"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchreasoningtreeimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
)

func researchThemeImportInput(request *v1.ResearchThemeImportRequest) researchpublication.Aggregate {
	theme := request.Theme
	impacts := make([]researchthemeimport.Impact, 0, len(theme.Impacts))
	for _, impact := range theme.Impacts {
		impacts = append(impacts, researchthemeimport.Impact{
			ChainNodeEntityID: impact.ChainNodeEntityID, RelationRole: impact.RelationRole,
			ImpactDirection: impact.ImpactDirection, ImpactSummary: impact.ImpactSummary,
			DisplayOrder: impact.DisplayOrder,
		})
	}
	themeEvents := make([]researchthemeimport.Event, 0, len(theme.Events))
	for _, event := range theme.Events {
		themeEvents = append(themeEvents, researchthemeimport.Event{
			EventID: event.EventID, EvidenceRole: event.EvidenceRole, SupportedClaim: event.SupportedClaim,
		})
	}
	themeInput := researchthemeimport.Theme{
		ThemeKey: theme.ThemeKey, Title: theme.Title, OneLineConclusion: theme.OneLineConclusion,
		ConclusionDirection: theme.ConclusionDirection, ImpactStrength: theme.ImpactStrength,
		AttentionLevel: theme.AttentionLevel, ConclusionStatus: theme.ConclusionStatus,
		TransmissionStage: theme.TransmissionStage, InvestmentGuidanceAction: theme.InvestmentGuidanceAction,
		InvestmentGuidanceSummary: theme.InvestmentGuidanceSummary,
		TimeHorizonCategory:       theme.TimeHorizonCategory, TimeHorizonSummary: theme.TimeHorizonSummary,
		TransmissionSummary: theme.TransmissionSummary, CheckpointSummary: theme.CheckpointSummary,
		RiskSummary: theme.RiskSummary, Impacts: impacts, Events: themeEvents,
	}
	trees := make([]researchpublication.ReasoningTree, 0, len(request.ReasoningTrees))
	for _, tree := range request.ReasoningTrees {
		checkpoints := make([]researchreasoningtreeimport.Checkpoint, 0, len(tree.Checkpoints))
		for _, checkpoint := range tree.Checkpoints {
			checkpoints = append(checkpoints, researchreasoningtreeimport.Checkpoint{
				Type: checkpoint.Type, Summary: checkpoint.Summary,
			})
		}
		events := make([]researchreasoningtreeimport.Event, 0, len(tree.Events))
		for _, event := range tree.Events {
			events = append(events, researchreasoningtreeimport.Event{
				EventID: event.EventID, EvidenceRole: event.EvidenceRole, DisplayOrder: event.DisplayOrder,
			})
		}
		nodes := make([]researchpublication.Node, 0, len(tree.Nodes))
		for _, node := range tree.Nodes {
			signals := make([]researchpublication.Signal, 0, len(node.Signals))
			for _, signal := range node.Signals {
				signals = append(signals, researchpublication.Signal{
					VariableSignalKey: signal.VariableSignalKey, SignalRole: signal.SignalRole,
					SignalDirection: signal.SignalDirection, DisplaySummary: signal.DisplaySummary,
					DisplayOrder: signal.DisplayOrder,
					Lineage: researchpublication.SignalLineage{
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
			var incoming *researchpublication.IncomingLineage
			if node.IncomingLineage != nil {
				incoming = &researchpublication.IncomingLineage{
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
			nodes = append(nodes, researchpublication.Node{
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
		trees = append(trees, researchpublication.ReasoningTree{
			ReasoningTree: researchreasoningtreeimport.ReasoningTree{
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
	return researchpublication.Aggregate{
		AnalysisBatchID: request.AnalysisBatchID, AnalysisAsOf: request.AnalysisAsOf,
		DiscoveryWindowStart: request.DiscoveryWindowStart,
		DiscoveryWindowEnd:   request.DiscoveryWindowEnd,
		Theme:                themeInput, ReasoningTrees: trees,
	}
}

func researchThemeSnapshotImportInput(request *v1.ResearchThemeSnapshotImportRequest) researchpublication.SnapshotAggregate {
	theme := request.Theme
	impacts := make([]researchpublication.SnapshotImpact, 0, len(theme.Impacts))
	for _, impact := range theme.Impacts {
		impacts = append(impacts, researchpublication.SnapshotImpact{
			NodeKey: impact.NodeKey, DisplayName: impact.DisplayName, RelationRole: impact.RelationRole,
			ImpactDirection: impact.ImpactDirection, ImpactSummary: impact.ImpactSummary, DisplayOrder: impact.DisplayOrder,
		})
	}
	themeEvents := make([]researchpublication.SnapshotEvent, 0, len(theme.Events))
	for _, event := range theme.Events {
		themeEvents = append(themeEvents, researchpublication.SnapshotEvent{
			EventID: event.EventID, EvidenceIDs: append([]string(nil), event.EvidenceIDs...),
			EvidenceRole: event.EvidenceRole, SupportedClaim: event.SupportedClaim,
		})
	}
	trees := make([]researchpublication.SnapshotReasoningTree, 0, len(request.ReasoningTrees))
	for _, tree := range request.ReasoningTrees {
		checkpoints := make([]researchreasoningtreeimport.Checkpoint, 0, len(tree.Checkpoints))
		for _, checkpoint := range tree.Checkpoints {
			checkpoints = append(checkpoints, researchreasoningtreeimport.Checkpoint{Type: checkpoint.Type, Summary: checkpoint.Summary})
		}
		events := make([]researchpublication.SnapshotTreeEvent, 0, len(tree.Events))
		for _, event := range tree.Events {
			events = append(events, researchpublication.SnapshotTreeEvent{
				EventID: event.EventID, EvidenceIDs: append([]string(nil), event.EvidenceIDs...),
				EvidenceRole: event.EvidenceRole, DisplayOrder: event.DisplayOrder,
			})
		}
		nodes := make([]researchpublication.SnapshotNode, 0, len(tree.Nodes))
		for _, node := range tree.Nodes {
			signals := make([]researchpublication.SnapshotSignal, 0, len(node.Signals))
			for _, signal := range node.Signals {
				signals = append(signals, researchpublication.SnapshotSignal{
					SignalKey: signal.SignalKey, DisplaySummary: signal.DisplaySummary, Role: signal.Role,
					DisplayOrder: signal.DisplayOrder, VariableName: signal.VariableName, Direction: signal.Direction,
				})
			}
			var incoming *researchpublication.SnapshotIncomingTransmission
			if node.IncomingTransmission != nil {
				incoming = &researchpublication.SnapshotIncomingTransmission{
					Title: node.IncomingTransmission.Title, Mechanism: node.IncomingTransmission.Mechanism,
					ConditionSummary: node.IncomingTransmission.ConditionSummary,
				}
			}
			nodes = append(nodes, researchpublication.SnapshotNode{
				NodeKey: node.NodeKey, DisplayName: node.DisplayName, Position: node.Position,
				StateSummary: node.StateSummary, ImpactDirection: node.ImpactDirection,
				ImpactStrength: node.ImpactStrength, ImpactSummary: node.ImpactSummary,
				ReasoningBasisSummary: node.ReasoningBasisSummary, EvidenceGapSummary: node.EvidenceGapSummary,
				IncomingTransmission: incoming, Signals: signals,
			})
		}
		trees = append(trees, researchpublication.SnapshotReasoningTree{
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
	return researchpublication.SnapshotAggregate{
		PublicationMode: request.PublicationMode, AnalysisBatchID: request.AnalysisBatchID,
		AnalysisAsOf: request.AnalysisAsOf, DiscoveryWindowStart: request.DiscoveryWindowStart,
		DiscoveryWindowEnd: request.DiscoveryWindowEnd,
		Theme: researchpublication.SnapshotTheme{
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

func researchThemeImportDTO(result researchpublication.Result) v1.ResearchThemeImportResult {
	return v1.ResearchThemeImportResult{
		ReceiptID: result.ReceiptID, AnalysisBatchID: result.AnalysisBatchID, PayloadHash: result.PayloadHash,
		ThemeID:                                 result.ThemeID,
		PublicationMode:                         result.PublicationMode,
		ReasoningTreeIDsByIndustryChainEntityID: result.ReasoningTreeIDsByIndustryChainEntityID,
		ReasoningTreeIDsByTreeKey:               result.ReasoningTreeIDsByTreeKey,
		Counts: v1.ResearchThemeImportCounts{
			Themes: result.Counts.Themes, Impacts: result.Counts.Impacts,
			ThemeEventAssociations: result.Counts.ThemeEventAssociations,
			ReasoningTrees:         result.Counts.ReasoningTrees, Nodes: result.Counts.Nodes,
			TreeEventAssociations: result.Counts.TreeEventAssociations,
			SignalAssociations:    result.Counts.SignalAssociations, Receipts: result.Counts.Receipts,
		},
		PublishedAt: result.PublishedAt, ImportedAt: result.ImportedAt, Replayed: result.Replayed,
	}
}

func researchThemeDTO(value research.ResearchTheme) v1.ResearchTheme {
	impacts := make([]v1.ResearchThemeImpact, 0, len(value.Impacts))
	for _, impact := range value.Impacts {
		impacts = append(impacts, v1.ResearchThemeImpact{
			NodeKey: impact.NodeKey, DisplayName: impact.DisplayName,
			ChainNodeEntityID: impact.ChainNodeEntityID, Name: impact.Name,
			RelationRole: impact.RelationRole, ImpactDirection: impact.ImpactDirection,
			ImpactSummary: impact.ImpactSummary, DisplayOrder: impact.DisplayOrder,
		})
	}
	return v1.ResearchTheme{
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

func researchEventsDTO(values []research.ResearchEvent) []v1.ResearchEvent {
	result := make([]v1.ResearchEvent, 0, len(values))
	for _, event := range values {
		result = append(result, v1.ResearchEvent{
			EvidenceIDs: append([]string(nil), event.EvidenceIDs...),
			EventID:     event.EventID, Title: event.Title, Summary: event.Summary, EventTime: event.EventTime,
			EvidenceRole: event.EvidenceRole, SupportedClaim: event.SupportedClaim, DisplayOrder: event.DisplayOrder,
		})
	}
	return result
}

func researchThemePageDTO(value research.ResearchThemePage) v1.ResearchThemePage {
	items := make([]v1.ResearchTheme, 0, len(value.Items))
	for _, item := range value.Items {
		items = append(items, researchThemeDTO(item))
	}
	return v1.ResearchThemePage{
		WindowStart: value.WindowStart, WindowEnd: value.WindowEnd, AsOf: value.AsOf,
		ThemeCount: value.ThemeCount, EventCount: value.EventCount, Items: items, NextCursor: value.NextCursor,
	}
}

func researchThemeDetailDTO(value research.ResearchThemeDetail) v1.ResearchThemeDetail {
	return v1.ResearchThemeDetail{
		ThemeKey: value.ThemeKey, PublicationMode: value.PublicationMode,
		PublicationContractVersion: value.PublicationContractVersion,
		Theme:                      researchThemeDTO(value.Theme), Events: researchEventsDTO(value.Events),
	}
}

func reasoningTreeListDTO(value research.ResearchReasoningTreeList) v1.ResearchReasoningTreeList {
	trees := make([]v1.ResearchReasoningTreeSummary, 0, len(value.ReasoningTrees))
	for _, tree := range value.ReasoningTrees {
		trees = append(trees, v1.ResearchReasoningTreeSummary{
			TreeKey: tree.TreeKey, DisplayName: tree.DisplayName,
			ReasoningTreeID: tree.ReasoningTreeID, IndustryChainEntityID: tree.IndustryChainEntityID,
			IndustryChainName: tree.IndustryChainName, Title: tree.Title, DisplayOrder: tree.DisplayOrder,
			EventCount: tree.EventCount, PublishedAt: tree.PublishedAt,
		})
	}
	return v1.ResearchReasoningTreeList{Theme: researchThemeDTO(value.Theme), ReasoningTrees: trees}
}

func reasoningTreeDetailDTO(value research.ResearchReasoningTreeDetail) v1.ResearchReasoningTreeDetail {
	tree := value.ReasoningTree
	checkpoints := make([]v1.ResearchReasoningTreeCheckpoint, 0, len(tree.Checkpoints))
	for _, checkpoint := range tree.Checkpoints {
		checkpoints = append(checkpoints, v1.ResearchReasoningTreeCheckpoint{Type: checkpoint.Type, Summary: checkpoint.Summary})
	}
	nodes := make([]v1.ResearchReasoningTreeNode, 0, len(tree.Nodes))
	for _, node := range tree.Nodes {
		signals := make([]v1.ResearchReasoningTreeSignal, 0, len(node.Signals))
		for _, signal := range node.Signals {
			signals = append(signals, researchSignalDTO(signal))
		}
		var graphEdge *v1.ResearchReasoningTreeGraphEdge
		if node.IncomingGraphEdge != nil {
			graphEdge = &v1.ResearchReasoningTreeGraphEdge{
				ID: node.IncomingGraphEdge.ID, RelationType: node.IncomingGraphEdge.RelationType,
				ReviewStatus: node.IncomingGraphEdge.ReviewStatus, Status: node.IncomingGraphEdge.Status,
			}
		}
		nodes = append(nodes, v1.ResearchReasoningTreeNode{
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
	return v1.ResearchReasoningTreeDetail{
		ThemeID: value.ThemeID, ThemeKey: value.ThemeKey, PublicationMode: value.PublicationMode,
		PublicationContractVersion: value.PublicationContractVersion,
		ImpactNodeIDs:              append([]string(nil), value.ImpactNodeIDs...),
		ReasoningTree: v1.ResearchReasoningTree{
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

func researchSignalDTO(signal research.ResearchSignal) v1.ResearchReasoningTreeSignal {
	return v1.ResearchReasoningTreeSignal{
		SignalKey: signal.SignalKey, VariableName: signal.VariableName, Direction: signal.Direction,
		VariableSignalKey: signal.VariableSignalKey, SignalRole: signal.SignalRole,
		SignalDirection: signal.SignalDirection, DisplaySummary: signal.DisplaySummary,
		DisplayOrder: signal.DisplayOrder,
	}
}
