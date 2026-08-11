package service

import (
	"context"
	"errors"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/evidencepublication"
)

func (s *DataService) PublishRawEvidence(ctx context.Context, request *v1.RawEvidencePublicationRequest) (*v1.Response[v1.RawEvidencePublicationResult], error) {
	if s == nil || s.dependencies.EvidencePublications == nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "Evidence Publication service is unavailable")
	}
	result, err := s.dependencies.EvidencePublications.PublishRawEvidence(ctx, principalIdentity(ctx), rawEvidenceInput(request.RawEvidence))
	if err != nil {
		return nil, rawEvidencePublicationError(err)
	}
	return &v1.Response[v1.RawEvidencePublicationResult]{Status: v1.StatusCreated, Result: rawEvidenceResultDTO(result)}, nil
}

func (s *DataService) PublishEvidence(ctx context.Context, request *v1.EvidencePublicationRequest) (*v1.Response[v1.EvidencePublicationResult], error) {
	if s == nil || s.dependencies.EvidencePublications == nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "Evidence Publication service is unavailable")
	}
	items := make([]evidencepublication.Evidence, len(request.Evidences))
	for index, item := range request.Evidences {
		items[index] = evidenceInput(item)
	}
	result, err := s.dependencies.EvidencePublications.PublishEvidence(ctx, principalIdentity(ctx), request.RawEvidenceID, items)
	if err != nil {
		return nil, evidencePublicationError(err)
	}
	return &v1.Response[v1.EvidencePublicationResult]{Status: v1.StatusCreated, Result: evidenceResultDTO(result)}, nil
}

func evidencePublicationError(err error) error {
	var validation *evidencepublication.ValidationError
	if errors.As(err, &validation) {
		status := evidenceValidationStatus(validation.Issues)
		code := "EVIDENCE_PUBLICATION_INVALID"
		if status == v1.StatusBadRequest {
			code = "INVALID_REQUEST"
		}
		return publicErrorWithDetails(status, code, "Evidence Publication failed validation", map[string]any{"issues": validation.Issues})
	}
	var reference *evidencepublication.ReferenceError
	if errors.As(err, &reference) {
		return publicErrorWithDetails(v1.StatusUnprocessableEntity, "EVIDENCE_PUBLICATION_REFERENCE_INVALID", "Evidence Publication references unavailable data", map[string]any{"issues": reference.Issues})
	}
	var conflict *evidencepublication.ConflictError
	if errors.As(err, &conflict) {
		return publicErrorWithDetails(v1.StatusConflict, "EVIDENCE_PUBLICATION_CONFLICT", "Evidence Publication conflicts with stored data", map[string]any{"issues": conflict.Issues})
	}
	return publicError(v1.StatusInternalServerError, "EVIDENCE_PUBLICATION_FAILED", "Evidence Publication failed")
}

func rawEvidencePublicationError(err error) error {
	var validation *evidencepublication.ValidationError
	if errors.As(err, &validation) {
		return publicErrorWithDetails(v1.StatusBadRequest, "INVALID_REQUEST", "Raw Evidence Publication request is invalid", map[string]any{"issues": validation.Issues})
	}
	return evidencePublicationError(err)
}

func evidenceValidationStatus(issues []evidencepublication.Issue) int {
	for _, issue := range issues {
		if issue.Path == "evidences" || issue.Code == evidencepublication.IssueDuplicate {
			continue
		}
		return v1.StatusBadRequest
	}
	return v1.StatusUnprocessableEntity
}

func rawEvidenceInput(input v1.RawEvidence) evidencepublication.RawEvidence {
	return evidencepublication.RawEvidence{
		RawEvidenceID: input.RawEvidenceID, SourceID: input.SourceID, SourceName: input.SourceName,
		SourceLevel: evidencepublication.SourceLevel(input.SourceLevel), SourceURL: input.SourceURL, IsOriginal: input.IsOriginal,
		QuotedSourceID: input.QuotedSourceID, QuotedSourceName: input.QuotedSourceName,
		Title: input.Title, RawText: input.RawText, PublishedAt: input.PublishedAt,
		CollectedAt: input.CollectedAt, Keywords: append([]string(nil), input.Keywords...),
	}
}

func evidenceInput(input v1.AtomicEvidence) evidencepublication.Evidence {
	return evidencepublication.Evidence{
		EvidenceID: input.EvidenceID, SplitOrder: input.SplitOrder, LayerType: evidencepublication.LayerType(input.LayerType),
		SourceWho: input.SourceWho, SourceWhat: input.SourceWhat, SourceWhen: input.SourceWhen,
		SourceWhenRaw: input.SourceWhenRaw, SourceWhere: input.SourceWhere, SourceWhy: input.SourceWhy,
		SourceHow: input.SourceHow, SourceWhoCore: input.SourceWhoCore, SourceWhatCore: input.SourceWhatCore,
		SourceWhenCore: input.SourceWhenCore, SourceWhenRawCore: input.SourceWhenRawCore,
		SourceWhereCore: input.SourceWhereCore, SourceWhyCore: input.SourceWhyCore, SourceHowCore: input.SourceHowCore,
		ExpressionFingerprint: input.ExpressionFingerprint, ExpressionKey: input.ExpressionKey,
		FingerprintVersion: input.FingerprintVersion,
	}
}

func rawEvidenceResultDTO(result evidencepublication.RawEvidenceResult) v1.RawEvidencePublicationResult {
	return v1.RawEvidencePublicationResult{
		ReceiptID: result.ReceiptID, ImportedAt: result.ImportedAt,
		RawEvidence: v1.RawEvidencePublicationItemResult{
			RawEvidenceID: result.RawEvidence.RawEvidenceID, ContentHash: result.RawEvidence.ContentHash,
			Keywords: append([]string(nil), result.RawEvidence.Keywords...), Disposition: string(result.RawEvidence.Disposition),
		},
	}
}

func evidenceResultDTO(result evidencepublication.EvidenceResult) v1.EvidencePublicationResult {
	items := make([]v1.EvidencePublicationItemResult, len(result.Evidences))
	for index, item := range result.Evidences {
		items[index] = v1.EvidencePublicationItemResult{
			EvidenceID: item.EvidenceID, SplitOrder: item.SplitOrder,
			IsSplit: item.IsSplit, Disposition: string(item.Disposition),
		}
	}
	return v1.EvidencePublicationResult{
		ReceiptID: result.ReceiptID, RawEvidenceID: result.RawEvidenceID, ImportedAt: result.ImportedAt,
		Evidences: items, Counts: v1.EvidencePublicationCounts{
			EvidencesCreated: result.Counts.Created, EvidencesReused: result.Counts.Reused,
		},
	}
}
