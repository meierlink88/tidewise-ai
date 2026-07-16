package eventimport

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	domainimport "github.com/meierlink88/tidewise-ai/backend/internal/domain/eventimport"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

const FixedSourceID = "cd209afe-2ea9-54b8-bdd7-db64eebf0d71"

var ErrIdempotencyConflict = errors.New("event import idempotency key conflicts with payload hash")

type Store = repositories.EventImportStore
type Transaction = repositories.EventImportTransaction
type Receipt = repositories.EventImportReceipt

type Result struct {
	ReceiptID      string   `json:"receipt_id"`
	EventID        string   `json:"event_id"`
	RawDocumentIDs []string `json:"raw_document_ids"`
	EventSourceIDs []string `json:"event_source_ids"`
	EventTagMapIDs []string `json:"event_tag_map_ids"`
	PayloadHash    string   `json:"payload_hash"`
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{store: store, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) Import(ctx context.Context, pkg domainimport.Package) (Result, error) {
	if s == nil || s.store == nil {
		return Result{}, errors.New("event import store is required")
	}
	if _, err := pkg.Validate(); err != nil {
		return Result{}, err
	}
	payloadHash, err := packageHash(pkg)
	if err != nil {
		return Result{}, err
	}
	var result Result
	err = s.store.InTransaction(ctx, func(tx Transaction) error {
		existing, err := tx.LockReceipt(ctx, pkg.IdempotencyKey)
		if err != nil {
			return fmt.Errorf("lock import receipt: %w", err)
		}
		if existing != nil {
			if existing.PayloadHash != payloadHash {
				return ErrIdempotencyConflict
			}
			result = resultFromReceipt(*existing)
			return nil
		}

		source, err := tx.Source(ctx, FixedSourceID)
		if err != nil {
			return fmt.Errorf("resolve fixed event source: %w", err)
		}
		if source.Status != domain.SourceCatalogStatusActive {
			return fmt.Errorf("fixed event source %q is not active", FixedSourceID)
		}

		rawIDs := make([]string, 0, len(pkg.RawDocuments))
		rawByToken := make(map[string]string, len(pkg.RawDocuments))
		for _, input := range pkg.RawDocuments {
			doc := domain.RawDocument{
				ID:               repositories.RawDocumentUUID(FixedSourceID, input.DocumentID, input.DocumentID, input.ContentHash),
				SourceID:         FixedSourceID,
				IngestChannel:    source.IngestChannel,
				SourceType:       source.SourceType,
				SourceName:       input.SourceName,
				SourceURL:        input.SourceURL,
				SourceExternalID: input.DocumentID,
				Title:            input.Title,
				ContentText:      input.ContentText,
				ContentLevel:     input.ContentLevel,
				PublishedAt:      input.PublishedAt,
				CollectedAt:      input.CollectedAt,
				ContentHash:      input.ContentHash,
				IngestStatus:     domain.IngestStatusCollected,
			}
			id, err := tx.UpsertRawDocument(ctx, doc)
			if err != nil {
				return fmt.Errorf("upsert raw document %q: %w", input.DocumentID, err)
			}
			rawIDs = append(rawIDs, id)
			rawByToken[input.DocumentID] = id
		}

		mapping, _ := pkg.Validate()
		eventID := repositories.NormalizeUUID("event", pkg.Event.DedupeKey)
		event := domain.Event{
			ID:          eventID,
			Title:       pkg.Event.Title,
			Summary:     pkg.Event.FactualSummary,
			EventTime:   pkg.Event.OccurredAt,
			FirstSeenAt: mapping.FirstSeenAt,
			EventStatus: domain.EventStatus(pkg.Event.EventStatus),
			FactStatus:  domain.FactStatus(pkg.Event.FactStatus),
			DedupeKey:   pkg.Event.DedupeKey,
			FactPayload: domain.FactPayload(pkg.Event.FactPayload),
		}
		if !mapping.KnowableAt.IsZero() {
			event.KnowableAt = &mapping.KnowableAt
		}
		storedEventID, err := tx.UpsertEvent(ctx, event)
		if err != nil {
			return fmt.Errorf("upsert event: %w", err)
		}

		sourceIDs := make([]string, 0, len(pkg.EventSources))
		for _, input := range pkg.EventSources {
			rawID := rawByToken[input.DocumentID]
			evidenceRelation := input.EvidenceRelation
			if evidenceRelation == "" {
				evidenceRelation = "supports"
			}
			evidenceID := repositories.NormalizeUUID("event_source", storedEventID, rawID, input.EvidenceHash)
			storedSourceID, err := tx.AddEventSource(ctx, domain.EventSource{
				ID: evidenceID, EventID: storedEventID, RawDocumentID: rawID,
				SourceLevel: input.SourceLevel, EvidenceExcerpt: input.EvidenceExcerpt,
				EvidenceHash: input.EvidenceHash, EvidenceRelation: domain.EvidenceRelation(evidenceRelation), SupportsFields: input.SupportsFields,
			})
			if err != nil {
				return fmt.Errorf("upsert event source: %w", err)
			}
			sourceIDs = append(sourceIDs, storedSourceID)
		}

		tagIDs := make([]string, 0, len(pkg.EventTags))
		for _, input := range pkg.EventTags {
			definition, err := tx.Tag(ctx, input.TagID, input.TagKind, input.TagCode)
			if err != nil {
				return fmt.Errorf("resolve event tag %q: %w", input.TagCode, err)
			}
			if definition.ID != input.TagID || definition.TagKind != input.TagKind || definition.Code != input.TagCode {
				return fmt.Errorf("event tag identity mismatch for %q", input.TagCode)
			}
			confidence, err := strconv.ParseFloat(string(input.Confidence), 64)
			if err != nil {
				return fmt.Errorf("parse event tag confidence %q: %w", input.TagCode, err)
			}
			tagMapID := repositories.NormalizeUUID("event_tag_map", storedEventID, input.TagID)
			storedTagID, err := tx.AssignEventTag(ctx, domain.EventTagMap{
				ID: tagMapID, EventID: storedEventID, TagID: input.TagID,
				AssignSource: input.AssignSource, ReviewStatus: domain.ReviewStatus(input.ReviewStatus), Confidence: &confidence, AssignmentReason: input.AssignmentReason,
			})
			if err != nil {
				return fmt.Errorf("upsert event tag %q: %w", input.TagCode, err)
			}
			tagIDs = append(tagIDs, storedTagID)
		}

		receipt := Receipt{ID: repositories.NormalizeUUID("event_import_receipt", pkg.IdempotencyKey), IdempotencyKey: pkg.IdempotencyKey, PackageID: pkg.PackageID, ReviewID: pkg.Review.ReviewID, ReviewDecision: pkg.Review.Decision, PayloadHash: payloadHash, EventID: storedEventID, RawDocumentIDs: rawIDs, EventSourceIDs: sourceIDs, EventTagMapIDs: tagIDs, ReviewMetadata: map[string]any{"evidence_grade": pkg.Review.EvidenceGrade, "reasons": pkg.Review.Reasons, "component_versions": pkg.Review.ComponentVersions}, ImportedAt: s.now()}
		if err := tx.InsertReceipt(ctx, receipt); err != nil {
			return fmt.Errorf("insert event import receipt: %w", err)
		}
		result = resultFromReceipt(receipt)
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func packageHash(pkg domainimport.Package) (string, error) {
	hash, err := pkg.CanonicalHash()
	if err != nil {
		return "", fmt.Errorf("canonicalize reviewed outbox for hash: %w", err)
	}
	return hash, nil
}

func resultFromReceipt(receipt Receipt) Result {
	return Result{ReceiptID: receipt.ID, EventID: receipt.EventID, RawDocumentIDs: append([]string(nil), receipt.RawDocumentIDs...), EventSourceIDs: append([]string(nil), receipt.EventSourceIDs...), EventTagMapIDs: append([]string(nil), receipt.EventTagMapIDs...), PayloadHash: receipt.PayloadHash}
}
