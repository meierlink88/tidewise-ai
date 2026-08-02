package service

import (
	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventpublication"
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
				EvidenceStatement: item.EvidenceStatement, SupportsFields: item.SupportsFields,
				SourceLevel: item.SourceLevel,
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
