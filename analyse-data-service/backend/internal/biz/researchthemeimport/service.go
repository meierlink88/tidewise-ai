package researchthemeimport

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/identity"
)

var (
	ErrPayloadConflict   = errors.New("analysis batch conflicts with the published payload")
	ErrPublisherConflict = errors.New("analysis batch belongs to another publisher subject")
)

type Result struct {
	ReceiptID       string            `json:"receipt_id"`
	AnalysisBatchID string            `json:"analysis_batch_id"`
	PayloadHash     string            `json:"payload_hash"`
	ThemeIDsByKey   map[string]string `json:"theme_ids_by_key"`
	Counts          Counts            `json:"counts"`
	PublishedAt     time.Time         `json:"published_at"`
	ImportedAt      time.Time         `json:"imported_at"`
	Replayed        bool              `json:"replayed"`
}

type ReferenceError struct {
	ThemeKey  string
	Path      string
	Reference string
}

func (e *ReferenceError) Error() string {
	return fmt.Sprintf("%s: %s references missing master data %q", e.ThemeKey, e.Path, e.Reference)
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{store: store, now: func() time.Time { return time.Now().UTC() }}
}

type plan struct {
	ReceiptID     string
	PayloadHash   string
	ThemeIDsByKey map[string]string
	Counts        Counts
	AnalysisAsOf  time.Time
	WindowStart   time.Time
	WindowEnd     time.Time
}

func (s *Service) Import(ctx context.Context, publisherSubject string, batch Batch) (Result, error) {
	if s == nil || s.store == nil {
		return Result{}, errors.New("research theme import store is required")
	}
	publisherSubject = strings.TrimSpace(publisherSubject)
	if publisherSubject == "" || len(publisherSubject) > 200 {
		return Result{}, errors.New("publisher subject must contain 1..200 characters")
	}
	publication, err := buildPlan(batch)
	if err != nil {
		return Result{}, err
	}

	var result Result
	err = s.store.InResearchThemeImportTransaction(ctx, func(tx Transaction) error {
		if err := tx.LockResearchThemeImportBatch(ctx, batch.AnalysisBatchID); err != nil {
			return fmt.Errorf("lock research theme import batch: %w", err)
		}
		existing, err := tx.ResearchThemeImportReceipt(ctx, batch.AnalysisBatchID)
		if err != nil {
			return fmt.Errorf("load research theme import receipt: %w", err)
		}
		if existing != nil {
			if err := validateReplay(*existing, publisherSubject, publication); err != nil {
				return err
			}
			if err := tx.VerifyResearchThemeImportReceipt(ctx, *existing); err != nil {
				return fmt.Errorf("verify research theme import replay: %w", err)
			}
			result = resultFromReceipt(*existing, true)
			return nil
		}

		if err := validateReferences(ctx, tx, batch); err != nil {
			return err
		}
		publishedAt := s.now().UTC().Truncate(time.Microsecond)
		receipt := Receipt{
			ID: publication.ReceiptID, AnalysisBatchID: batch.AnalysisBatchID, PublisherSubject: publisherSubject,
			PayloadHash: publication.PayloadHash, ThemeIDsByKey: cloneStringMap(publication.ThemeIDsByKey), Counts: publication.Counts,
			PublishedAt: publishedAt, ImportedAt: publishedAt,
		}
		if err := tx.InsertResearchThemeImportReceipt(ctx, receipt); err != nil {
			return fmt.Errorf("insert research theme import receipt: %w", err)
		}
		for _, theme := range batch.Themes {
			themeID := publication.ThemeIDsByKey[theme.ThemeKey]
			if err := tx.InsertResearchTheme(ctx, ThemeRecord{
				ID: themeID, ImportReceiptID: receipt.ID, AnalysisBatchID: batch.AnalysisBatchID, ThemeKey: theme.ThemeKey,
				Title: theme.Title, OneLineConclusion: theme.OneLineConclusion,
				ConclusionDirection: theme.ConclusionDirection, ImpactStrength: theme.ImpactStrength,
				AttentionLevel: theme.AttentionLevel, ConclusionStatus: theme.ConclusionStatus,
				TransmissionStage:         theme.TransmissionStage,
				InvestmentGuidanceAction:  theme.InvestmentGuidanceAction,
				InvestmentGuidanceSummary: theme.InvestmentGuidanceSummary,
				TimeHorizonCategory:       theme.TimeHorizonCategory, TimeHorizonSummary: theme.TimeHorizonSummary,
				TransmissionSummary: theme.TransmissionSummary, CheckpointSummary: theme.CheckpointSummary,
				RiskSummary: theme.RiskSummary, AnalysisAsOf: publication.AnalysisAsOf,
				WindowStart: publication.WindowStart, WindowEnd: publication.WindowEnd, PublishedAt: publishedAt,
			}); err != nil {
				return fmt.Errorf("insert Theme %q: %w", theme.ThemeKey, err)
			}
			for impactIndex, impact := range theme.Impacts {
				if err := tx.InsertResearchThemeImpact(ctx, ImpactRecord{
					ThemeID: themeID, ChainNodeEntityID: impact.ChainNodeEntityID,
					RelationRole: impact.RelationRole, ImpactDirection: impact.ImpactDirection,
					ImpactSummary: impact.ImpactSummary, DisplayOrder: impact.DisplayOrder,
				}); err != nil {
					return fmt.Errorf("insert %s Impact %d: %w", theme.ThemeKey, impactIndex, err)
				}
			}
			for eventIndex, event := range theme.Events {
				if err := tx.InsertResearchThemeEvent(ctx, EventRecord{
					ThemeID: themeID, EventID: event.EventID, EvidenceRole: event.EvidenceRole, SupportedClaim: event.SupportedClaim,
				}); err != nil {
					return fmt.Errorf("insert %s Event %d: %w", theme.ThemeKey, eventIndex, err)
				}
			}
		}
		if err := tx.VerifyResearchThemeImportReceipt(ctx, receipt); err != nil {
			return fmt.Errorf("verify research Theme import receipt: %w", err)
		}
		result = resultFromReceipt(receipt, false)
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func buildPlan(batch Batch) (plan, error) {
	window, err := batch.Validate()
	if err != nil {
		return plan{}, err
	}
	payloadHash, err := CanonicalHash(batch)
	if err != nil {
		return plan{}, fmt.Errorf("hash research theme publication batch: %w", err)
	}
	themeIDs := make(map[string]string, len(batch.Themes))
	counts := Counts{Themes: len(batch.Themes), Receipts: 1}
	for _, theme := range batch.Themes {
		themeIDs[theme.ThemeKey] = ThemeID(batch.AnalysisBatchID, theme.ThemeKey)
		counts.Impacts += len(theme.Impacts)
		counts.EventAssociations += len(theme.Events)
	}
	return plan{
		ReceiptID:   identity.NormalizeUUID("research_theme_import_receipt", batch.AnalysisBatchID),
		PayloadHash: payloadHash, ThemeIDsByKey: themeIDs, Counts: counts,
		AnalysisAsOf: window.AnalysisAsOf, WindowStart: window.Start, WindowEnd: window.End,
	}, nil
}

func validateReferences(ctx context.Context, tx Transaction, batch Batch) error {
	chainNodeIDs := make([]string, 0)
	eventIDs := make([]string, 0)
	for _, theme := range batch.Themes {
		for _, impact := range theme.Impacts {
			chainNodeIDs = append(chainNodeIDs, impact.ChainNodeEntityID)
		}
		for _, event := range theme.Events {
			eventIDs = append(eventIDs, event.EventID)
		}
	}
	existingNodes, err := tx.ExistingResearchThemeImpactNodes(ctx, chainNodeIDs)
	if err != nil {
		return fmt.Errorf("resolve research theme chain nodes: %w", err)
	}
	existingEvents, err := tx.ExistingResearchThemeEvents(ctx, eventIDs)
	if err != nil {
		return fmt.Errorf("resolve research theme Events: %w", err)
	}
	for themeIndex, theme := range batch.Themes {
		for impactIndex, impact := range theme.Impacts {
			if _, exists := existingNodes[impact.ChainNodeEntityID]; !exists {
				return &ReferenceError{ThemeKey: theme.ThemeKey, Path: fmt.Sprintf("themes[%d].impacts[%d].chain_node_entity_id", themeIndex, impactIndex), Reference: impact.ChainNodeEntityID}
			}
		}
		for eventIndex, event := range theme.Events {
			if _, exists := existingEvents[event.EventID]; !exists {
				return &ReferenceError{ThemeKey: theme.ThemeKey, Path: fmt.Sprintf("themes[%d].events[%d].event_id", themeIndex, eventIndex), Reference: event.EventID}
			}
		}
	}
	return nil
}

func validateReplay(receipt Receipt, publisherSubject string, publication plan) error {
	if receipt.PublisherSubject != publisherSubject {
		return ErrPublisherConflict
	}
	if receipt.PayloadHash != publication.PayloadHash {
		return ErrPayloadConflict
	}
	if receipt.ID != publication.ReceiptID || receipt.Counts != publication.Counts || !reflect.DeepEqual(receipt.ThemeIDsByKey, publication.ThemeIDsByKey) {
		return errors.New("research theme import receipt does not match deterministic plan")
	}
	return nil
}

func resultFromReceipt(receipt Receipt, replayed bool) Result {
	return Result{
		ReceiptID: receipt.ID, AnalysisBatchID: receipt.AnalysisBatchID, PayloadHash: receipt.PayloadHash,
		ThemeIDsByKey: cloneStringMap(receipt.ThemeIDsByKey), Counts: receipt.Counts,
		PublishedAt: receipt.PublishedAt.UTC(), ImportedAt: receipt.ImportedAt.UTC(), Replayed: replayed,
	}
}

func cloneStringMap(input map[string]string) map[string]string {
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
