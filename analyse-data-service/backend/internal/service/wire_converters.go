package service

import (
	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventpublication"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchanchorimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
)

func eventPublicationInput(request *v1.EventPublicationRequest) eventpublication.Publication {
	collectors := make([]eventpublication.CollectorExecution, 0, len(request.Provenance.CollectorExecutions))
	for _, collector := range request.Provenance.CollectorExecutions {
		collectors = append(collectors, eventpublication.CollectorExecution{
			ArtifactID: collector.ArtifactID, CollectorExecutionID: collector.CollectorExecutionID,
		})
	}
	rawDocuments := make([]eventpublication.RawDocument, 0, len(request.RawDocuments))
	for _, document := range request.RawDocuments {
		rawDocuments = append(rawDocuments, eventpublication.RawDocument{
			ArtifactID: document.ArtifactID, ContentSHA256: document.ContentSHA256,
			SourceRef: document.SourceRef, SourceName: document.SourceName, SourceType: document.SourceType,
			SourceURL: document.SourceURL, Title: document.Title, PublishedAt: document.PublishedAt,
			CollectedAt: document.CollectedAt, Language: document.Language, MIMEType: document.MIMEType,
		})
	}
	events := make([]eventpublication.Event, 0, len(request.Events))
	for _, event := range request.Events {
		evidence := make([]eventpublication.Evidence, 0, len(event.Evidence))
		for _, item := range event.Evidence {
			evidence = append(evidence, eventpublication.Evidence{
				ArtifactID: item.ArtifactID, EvidenceRelation: item.EvidenceRelation,
				EvidenceExcerpt: item.EvidenceExcerpt, SupportsFields: item.SupportsFields,
				SourceLevel: item.SourceLevel, IsPrimary: item.IsPrimary,
			})
		}
		tags := make([]eventpublication.Tag, 0, len(event.Tags))
		for _, tag := range event.Tags {
			tags = append(tags, eventpublication.Tag{
				TagID: tag.TagID, TagKind: tag.TagKind, TagCode: tag.TagCode, Confidence: tag.Confidence,
				AssignmentReason: tag.AssignmentReason, AssignSource: tag.AssignSource,
			})
		}
		events = append(events, eventpublication.Event{
			DedupeKey: event.DedupeKey, Title: event.Title, FactualSummary: event.FactualSummary,
			OccurredAt: event.OccurredAt, FactPayload: event.FactPayload, Evidence: evidence, Tags: tags,
			Review: eventpublication.Review{
				ReviewID: event.Review.ReviewID, EvidenceGrade: event.Review.EvidenceGrade, Reasons: event.Review.Reasons,
			},
		})
	}
	return eventpublication.Publication{
		PackageID: request.PackageID,
		Provenance: eventpublication.Provenance{
			ExtractorExecutionID:  request.Provenance.ExtractorExecutionID,
			ExtractorAgentVersion: request.Provenance.ExtractorAgentVersion,
			CollectorExecutions:   collectors,
		},
		RawDocuments: rawDocuments,
		Events:       events,
	}
}

func researchThemeImportInput(request *v1.ResearchThemeImportRequest) researchthemeimport.Batch {
	themes := make([]researchthemeimport.Theme, 0, len(request.Themes))
	for _, theme := range request.Themes {
		nodes := make([]researchthemeimport.ChainNode, 0, len(theme.ChainNodes))
		for _, node := range theme.ChainNodes {
			nodes = append(nodes, researchthemeimport.ChainNode{
				ChainNodeID: node.ChainNodeID, RelationRole: node.RelationRole, ImpactSummary: node.ImpactSummary,
			})
		}
		events := make([]researchthemeimport.Event, 0, len(theme.Events))
		for _, event := range theme.Events {
			events = append(events, researchthemeimport.Event{
				EventID: event.EventID, EvidenceRole: event.EvidenceRole, SupportedClaim: event.SupportedClaim,
			})
		}
		themes = append(themes, researchthemeimport.Theme{
			ThemeKey: theme.ThemeKey, Name: theme.Name, OneLineConclusion: theme.OneLineConclusion,
			ImpactLevel: theme.ImpactLevel, TransmissionPath: theme.TransmissionPath,
			TradingDirection: theme.TradingDirection, TransmissionStage: theme.TransmissionStage,
			NextCheckpoint: theme.NextCheckpoint, MarketConfirmationSummary: theme.MarketConfirmationSummary,
			ChainNodes: nodes, Events: events,
		})
	}
	return researchthemeimport.Batch{
		AnalysisBatchID: request.AnalysisBatchID, WindowStart: request.WindowStart,
		WindowEnd: request.WindowEnd, Themes: themes,
	}
}

func researchAnchorImportInput(request *v1.ResearchAnchorImportRequest) researchanchorimport.Publication {
	anchors := make([]researchanchorimport.Anchor, 0, len(request.Anchors))
	for _, anchor := range request.Anchors {
		events := make([]researchanchorimport.Event, 0, len(anchor.Events))
		for _, event := range anchor.Events {
			events = append(events, researchanchorimport.Event{
				EventID: event.EventID, EvidenceRole: event.EvidenceRole, EvidenceSummary: event.EvidenceSummary,
			})
		}
		pathNodes := make([]researchanchorimport.PathNode, 0, len(anchor.PathNodes))
		for _, node := range anchor.PathNodes {
			pathNodes = append(pathNodes, researchanchorimport.PathNode{
				ChainNodeID: node.ChainNodeID, ChangeDirection: node.ChangeDirection,
				ChangeSummary: node.ChangeSummary, ImpactSummary: node.ImpactSummary,
				IncomingTransmissionMechanism: node.IncomingTransmissionMechanism,
			})
		}
		anchors = append(anchors, researchanchorimport.Anchor{
			CenterChainNodeID: anchor.CenterChainNodeID, OneLineConclusion: anchor.OneLineConclusion,
			FactSummary: anchor.FactSummary, NetDirectionSummary: anchor.NetDirectionSummary,
			SupportSummary: anchor.SupportSummary, CounterSummary: anchor.CounterSummary,
			TradingDirection: anchor.TradingDirection, NextCheckpoint: anchor.NextCheckpoint,
			Events: events, PathNodes: pathNodes,
		})
	}
	return researchanchorimport.Publication{ThemeID: request.ThemeID, Anchors: anchors}
}

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
