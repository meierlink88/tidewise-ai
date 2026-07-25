package service

import (
	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventpublication"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchanchorimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
)

func eventPublicationDTO(result eventpublication.Result) v1.EventPublicationResult {
	events := make([]v1.EventPublicationEventResult, 0, len(result.Events))
	for _, event := range result.Events {
		events = append(events, v1.EventPublicationEventResult{
			DedupeKey: event.DedupeKey, EventID: event.EventID, Disposition: event.Disposition,
		})
	}
	rawDocuments := make([]v1.EventPublicationRawDocumentResult, 0, len(result.RawDocuments))
	for _, document := range result.RawDocuments {
		rawDocuments = append(rawDocuments, v1.EventPublicationRawDocumentResult{
			ArtifactID: document.ArtifactID, RawDocumentID: document.RawDocumentID, Disposition: document.Disposition,
		})
	}
	return v1.EventPublicationResult{
		ReceiptID: result.ReceiptID, PackageID: result.PackageID, ImportedAt: result.ImportedAt,
		Events: events, RawDocuments: rawDocuments,
		Counts: v1.EventPublicationCounts{
			EventsCreated: result.Counts.EventsCreated, EventsReused: result.Counts.EventsReused,
			RawDocumentsCreated: result.Counts.RawDocumentsCreated, RawDocumentsReused: result.Counts.RawDocumentsReused,
			EventSourcesCreated: result.Counts.EventSourcesCreated, EventSourcesReused: result.Counts.EventSourcesReused,
			EventTagsCreated: result.Counts.EventTagsCreated, EventTagsReused: result.Counts.EventTagsReused,
		},
	}
}

func researchThemeImportDTO(result researchthemeimport.Result) v1.ResearchThemeImportResult {
	return v1.ResearchThemeImportResult{
		ReceiptID: result.ReceiptID, AnalysisBatchID: result.AnalysisBatchID, PayloadHash: result.PayloadHash,
		ThemeIDsByKey: result.ThemeIDsByKey,
		Counts: v1.ResearchThemeImportCounts{
			Themes: result.Counts.Themes, ChainNodeAssociations: result.Counts.ChainNodeAssociations,
			EventAssociations: result.Counts.EventAssociations, Receipts: result.Counts.Receipts,
		},
		PublishedAt: result.PublishedAt, ImportedAt: result.ImportedAt, Replayed: result.Replayed,
	}
}

func researchAnchorImportDTO(result researchanchorimport.Result) v1.ResearchAnchorImportResult {
	return v1.ResearchAnchorImportResult{
		ReceiptID: result.ReceiptID, ThemeID: result.ThemeID, PayloadHash: result.PayloadHash,
		AnchorIDsByCenterChainNodeID: result.AnchorIDsByCenterChainNodeID,
		Counts: v1.ResearchAnchorImportCounts{
			Anchors: result.Counts.Anchors, EventAssociations: result.Counts.EventAssociations,
			PathNodes: result.Counts.PathNodes, Receipts: result.Counts.Receipts,
		},
		PublishedAt: result.PublishedAt, ImportedAt: result.ImportedAt, Replayed: result.Replayed,
	}
}

func researchThemeDTO(value research.ResearchTheme) v1.ResearchTheme {
	chainNodes := make([]v1.ResearchThemeChainNode, 0, len(value.AffectedChainNodes))
	for _, node := range value.AffectedChainNodes {
		chainNodes = append(chainNodes, v1.ResearchThemeChainNode{
			ID: node.ID, Name: node.Name, RelationRole: node.RelationRole, ImpactSummary: node.ImpactSummary,
		})
	}
	indices := make([]v1.ResearchIndex, 0, len(value.RelatedIndices))
	for _, index := range value.RelatedIndices {
		indices = append(indices, v1.ResearchIndex{
			ID: index.ID, Name: index.Name, ImpactDirection: string(index.ImpactDirection), ImpactSummary: index.ImpactSummary,
		})
	}
	return v1.ResearchTheme{
		ID: value.ID, Name: value.Name, OneLineConclusion: value.OneLineConclusion,
		ImpactLevel: string(value.ImpactLevel), TransmissionPath: value.TransmissionPath,
		TradingDirection: value.TradingDirection, TransmissionStage: string(value.TransmissionStage),
		NextCheckpoint: value.NextCheckpoint, MarketConfirmationSummary: value.MarketConfirmationSummary,
		PublishedAt: value.PublishedAt, AffectedChainNodes: chainNodes, RelatedIndices: indices,
		SupportingEventCount: value.SupportingEventCount, ContradictingEventCount: value.ContradictingEventCount,
	}
}

func researchEventsDTO(values []research.ResearchEvent) []v1.ResearchEvent {
	result := make([]v1.ResearchEvent, 0, len(values))
	for _, event := range values {
		result = append(result, v1.ResearchEvent{
			EventID: event.EventID, Title: event.Title, Summary: event.Summary, EventTime: event.EventTime,
			EvidenceRole: string(event.EvidenceRole), SupportedClaim: event.SupportedClaim,
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
	return v1.ResearchThemeDetail{Theme: researchThemeDTO(value.Theme), Events: researchEventsDTO(value.Events)}
}

func reasoningTreeListDTO(value research.ResearchReasoningTreeList) v1.ResearchReasoningTreeList {
	trees := make([]v1.ResearchReasoningTreeSummary, 0, len(value.ReasoningTrees))
	for _, tree := range value.ReasoningTrees {
		trees = append(trees, v1.ResearchReasoningTreeSummary{
			AnchorID: tree.AnchorID,
			CenterChainNode: v1.ResearchReasoningTreeChainNode{
				ID: tree.CenterChainNode.ID, Name: tree.CenterChainNode.Name,
			},
		})
	}
	return v1.ResearchReasoningTreeList{Theme: researchThemeDTO(value.Theme), ReasoningTrees: trees}
}

func reasoningTreeDetailDTO(value research.ResearchReasoningTreeDetail) v1.ResearchReasoningTreeDetail {
	tree := value.ReasoningTree
	events := make([]v1.ResearchReasoningTreeEvent, 0, len(tree.Events))
	for _, event := range tree.Events {
		events = append(events, v1.ResearchReasoningTreeEvent{
			EventID: event.EventID, Title: event.Title, Summary: event.Summary, EventTime: event.EventTime,
			EvidenceRole: event.EvidenceRole, EvidenceSummary: event.EvidenceSummary,
		})
	}
	pathNodes := make([]v1.ResearchReasoningTreePathNode, 0, len(tree.PathNodes))
	for _, node := range tree.PathNodes {
		pathNodes = append(pathNodes, v1.ResearchReasoningTreePathNode{
			ChainNodeID: node.ChainNodeID, Name: node.Name, ChangeDirection: node.ChangeDirection,
			ChangeSummary: node.ChangeSummary, ImpactSummary: node.ImpactSummary,
			IncomingTransmissionMechanism: node.IncomingTransmissionMechanism,
		})
	}
	return v1.ResearchReasoningTreeDetail{
		ThemeID: value.ThemeID,
		ReasoningTree: v1.ResearchReasoningTree{
			AnchorID: tree.AnchorID,
			CenterChainNode: v1.ResearchReasoningTreeChainNode{
				ID: tree.CenterChainNode.ID, Name: tree.CenterChainNode.Name,
			},
			OneLineConclusion: tree.OneLineConclusion, FactSummary: tree.FactSummary,
			NetDirectionSummary: tree.NetDirectionSummary, SupportSummary: tree.SupportSummary,
			CounterSummary: tree.CounterSummary, TradingDirection: tree.TradingDirection,
			NextCheckpoint: tree.NextCheckpoint, EventCount: tree.EventCount, Events: events, PathNodes: pathNodes,
		},
	}
}
