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
	PackageID      string   `json:"package_id"`
	ReceiptID      string   `json:"receipt_id"`
	EventID        string   `json:"event_id"`
	RawDocumentIDs []string `json:"raw_document_ids"`
	EventSourceIDs []string `json:"event_source_ids"`
	EventTagMapIDs []string `json:"event_tag_map_ids"`
	PayloadHash    string   `json:"payload_hash"`
}

type Plan struct {
	PackageID      string   `json:"package_id"`
	PayloadHash    string   `json:"payload_hash"`
	ReceiptID      string   `json:"receipt_id"`
	EventID        string   `json:"event_id"`
	RawDocumentIDs []string `json:"raw_document_ids"`
	EventSourceIDs []string `json:"event_source_ids"`
	EventTagMapIDs []string `json:"event_tag_map_ids"`
	Counts         Counts   `json:"counts"`
}

type Counts struct {
	RawDocuments int `json:"raw_documents"`
	Events       int `json:"events"`
	EventSources int `json:"event_sources"`
	EventTags    int `json:"event_tags"`
	Receipts     int `json:"receipts"`
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{store: store, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) Plan(pkg domainimport.Package) (Plan, error) {
	if _, err := pkg.Validate(); err != nil {
		return Plan{}, err
	}
	hash, err := packageHash(pkg)
	if err != nil {
		return Plan{}, err
	}
	eventID := repositories.NormalizeUUID("event", pkg.Event.DedupeKey)
	rawIDs := make([]string, 0, len(pkg.RawDocuments))
	for _, input := range pkg.RawDocuments {
		rawIDs = append(rawIDs, repositories.RawDocumentUUID(FixedSourceID, input.DocumentID, input.DocumentID, input.ContentHash))
	}
	sourceIDs := make([]string, 0, len(pkg.EventSources))
	for _, input := range pkg.EventSources {
		rawIndex := -1
		for index, document := range pkg.RawDocuments {
			if document.DocumentID == input.DocumentID {
				rawIndex = index
				break
			}
		}
		if rawIndex < 0 {
			return Plan{}, fmt.Errorf("event source document %q is not planned", input.DocumentID)
		}
		sourceIDs = append(sourceIDs, repositories.NormalizeUUID("event_source", eventID, rawIDs[rawIndex], input.EvidenceHash))
	}
	tagIDs := make([]string, 0, len(pkg.EventTags))
	for _, input := range pkg.EventTags {
		tagIDs = append(tagIDs, repositories.NormalizeUUID("event_tag_map", eventID, input.TagID))
	}
	plan := Plan{
		PackageID: pkg.PackageID, PayloadHash: hash,
		ReceiptID: repositories.NormalizeUUID("event_import_receipt", pkg.IdempotencyKey), EventID: eventID,
		RawDocumentIDs: rawIDs, EventSourceIDs: sourceIDs, EventTagMapIDs: tagIDs,
		Counts: Counts{RawDocuments: len(rawIDs), Events: 1, EventSources: len(sourceIDs), EventTags: len(tagIDs), Receipts: 1},
	}
	if err := validatePlan(plan); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func validatePlan(plan Plan) error {
	if plan.EventID == "" || plan.ReceiptID == "" {
		return errors.New("planned event and receipt IDs are required")
	}
	for name, ids := range map[string][]string{"raw_documents": plan.RawDocumentIDs, "event_sources": plan.EventSourceIDs, "event_tag_maps": plan.EventTagMapIDs} {
		seen := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			if id == "" {
				return fmt.Errorf("planned %s contains empty ID", name)
			}
			if _, ok := seen[id]; ok {
				return fmt.Errorf("planned %s contains duplicate ID %q", name, id)
			}
			seen[id] = struct{}{}
		}
	}
	return nil
}

func (s *Service) Import(ctx context.Context, pkg domainimport.Package) (Result, error) {
	if s == nil || s.store == nil {
		return Result{}, errors.New("event import store is required")
	}
	if _, err := pkg.Validate(); err != nil {
		return Result{}, err
	}
	plan, err := s.Plan(pkg)
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
			if err := validateReceiptReplay(*existing, pkg, plan); err != nil {
				return err
			}
			if err := tx.VerifyReceiptResults(ctx, *existing); err != nil {
				return fmt.Errorf("verify replay receipt results: %w", err)
			}
			if existing.PayloadHash != plan.PayloadHash {
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
		for index, input := range pkg.RawDocuments {
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
			if err := ensureUniqueNonEmpty(rawIDs, "raw document"); err != nil {
				return err
			}
			if id != plan.RawDocumentIDs[index] {
				return fmt.Errorf("raw document %q resolved to unexpected ID %q", input.DocumentID, id)
			}
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
		if storedEventID != plan.EventID {
			return fmt.Errorf("event resolved to unexpected ID %q", storedEventID)
		}

		sourceIDs := make([]string, 0, len(pkg.EventSources))
		for index, input := range pkg.EventSources {
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
			if storedSourceID != plan.EventSourceIDs[index] {
				return fmt.Errorf("event source resolved to unexpected ID %q", storedSourceID)
			}
			sourceIDs = append(sourceIDs, storedSourceID)
			if err := ensureUniqueNonEmpty(sourceIDs, "event source"); err != nil {
				return err
			}
		}

		tagIDs := make([]string, 0, len(pkg.EventTags))
		for index, input := range pkg.EventTags {
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
			if storedTagID != plan.EventTagMapIDs[index] {
				return fmt.Errorf("event tag map resolved to unexpected ID %q", storedTagID)
			}
			tagIDs = append(tagIDs, storedTagID)
			if err := ensureUniqueNonEmpty(tagIDs, "event tag map"); err != nil {
				return err
			}
		}

		if storedEventID == "" {
			return errors.New("repository returned empty event ID")
		}
		receipt := Receipt{ID: plan.ReceiptID, IdempotencyKey: pkg.IdempotencyKey, PackageID: pkg.PackageID, ReviewID: pkg.Review.ReviewID, ReviewDecision: pkg.Review.Decision, PayloadHash: plan.PayloadHash, EventID: storedEventID, RawDocumentIDs: rawIDs, EventSourceIDs: sourceIDs, EventTagMapIDs: tagIDs, ReviewMetadata: map[string]any{"evidence_grade": pkg.Review.EvidenceGrade, "reasons": pkg.Review.Reasons, "component_versions": pkg.Review.ComponentVersions}, ImportedAt: s.now()}
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

func ensureUniqueNonEmpty(ids []string, label string) error {
	if len(ids) == 0 {
		return fmt.Errorf("repository returned no %s IDs", label)
	}
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			return fmt.Errorf("repository returned empty %s ID", label)
		}
		if _, exists := seen[id]; exists {
			return fmt.Errorf("repository returned duplicate %s ID %q", label, id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func validateReceiptReplay(receipt Receipt, pkg domainimport.Package, plan Plan) error {
	if receipt.ID != plan.ReceiptID || receipt.IdempotencyKey != pkg.IdempotencyKey || receipt.PackageID != pkg.PackageID || receipt.ReviewID != pkg.Review.ReviewID || receipt.ReviewDecision != pkg.Review.Decision || receipt.EventID != plan.EventID {
		return fmt.Errorf("replay receipt identity does not match deterministic plan")
	}
	if receipt.PayloadHash != plan.PayloadHash {
		return ErrIdempotencyConflict
	}
	if err := ensureUniqueNonEmpty(receipt.RawDocumentIDs, "replay raw document"); err != nil {
		return err
	}
	if err := ensureUniqueNonEmpty(receipt.EventSourceIDs, "replay event source"); err != nil {
		return err
	}
	if err := ensureUniqueNonEmpty(receipt.EventTagMapIDs, "replay event tag map"); err != nil {
		return err
	}
	if !sameStrings(receipt.RawDocumentIDs, plan.RawDocumentIDs) || !sameStrings(receipt.EventSourceIDs, plan.EventSourceIDs) || !sameStrings(receipt.EventTagMapIDs, plan.EventTagMapIDs) {
		return fmt.Errorf("replay receipt result IDs do not match deterministic plan")
	}
	return nil
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func packageHash(pkg domainimport.Package) (string, error) {
	hash, err := pkg.CanonicalHash()
	if err != nil {
		return "", fmt.Errorf("canonicalize reviewed outbox for hash: %w", err)
	}
	return hash, nil
}

func resultFromReceipt(receipt Receipt) Result {
	return Result{PackageID: receipt.PackageID, ReceiptID: receipt.ID, EventID: receipt.EventID, RawDocumentIDs: append([]string(nil), receipt.RawDocumentIDs...), EventSourceIDs: append([]string(nil), receipt.EventSourceIDs...), EventTagMapIDs: append([]string(nil), receipt.EventTagMapIDs...), PayloadHash: receipt.PayloadHash}
}
