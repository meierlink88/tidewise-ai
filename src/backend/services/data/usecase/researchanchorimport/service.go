package researchanchorimport

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	domainimport "github.com/meierlink88/tidewise-ai/backend/services/data/domain/researchanchorimport"
	"github.com/meierlink88/tidewise-ai/backend/services/data/repositories"
)

var (
	ErrPayloadConflict   = errors.New("Theme conflicts with the published Research Anchor payload")
	ErrPublisherConflict = errors.New("Theme publication belongs to another publisher subject")
)

type ReferenceKind uint8

const (
	ReferenceNotFound ReferenceKind = iota + 1
	ReferenceInvalid
)

type ReferenceError struct {
	Kind              ReferenceKind
	CenterChainNodeID string
	Path              string
	Reference         string
	Message           string
}

func (e *ReferenceError) Error() string {
	return fmt.Sprintf("%s: %s (%s)", e.Path, e.Message, e.Reference)
}

type ContractError struct {
	CenterChainNodeID string
	Path              string
	Reference         string
	Message           string
}

func (e *ContractError) Error() string {
	return fmt.Sprintf("%s: %s (%s)", e.Path, e.Message, e.Reference)
}

type Store = repositories.ResearchAnchorImportStore
type Transaction = repositories.ResearchAnchorImportTransaction
type Receipt = repositories.ResearchAnchorImportReceipt
type Counts = repositories.ResearchAnchorImportCounts

type Result struct {
	ReceiptID                    string            `json:"receipt_id"`
	ThemeID                      string            `json:"theme_id"`
	PayloadHash                  string            `json:"payload_hash"`
	AnchorIDsByCenterChainNodeID map[string]string `json:"anchor_ids_by_center_chain_node_id"`
	Counts                       Counts            `json:"counts"`
	PublishedAt                  time.Time         `json:"published_at"`
	ImportedAt                   time.Time         `json:"imported_at"`
	Replayed                     bool              `json:"replayed"`
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{store: store, now: func() time.Time { return time.Now().UTC() }}
}

type plan struct {
	ReceiptID                    string
	PayloadHash                  string
	AnchorIDsByCenterChainNodeID map[string]string
	Counts                       Counts
}

func (s *Service) Import(ctx context.Context, publisherSubject string, publication domainimport.Publication) (Result, error) {
	if s == nil || s.store == nil {
		return Result{}, errors.New("Research Anchor import store is required")
	}
	publisherSubject = strings.TrimSpace(publisherSubject)
	if publisherSubject == "" || len(publisherSubject) > 200 {
		return Result{}, errors.New("publisher subject must contain 1..200 characters")
	}
	publicationPlan, err := buildPlan(publication)
	if err != nil {
		return Result{}, err
	}

	var result Result
	err = s.store.InResearchAnchorImportTransaction(ctx, func(tx Transaction) error {
		if err := tx.LockResearchAnchorImportTheme(ctx, publication.ThemeID); err != nil {
			return fmt.Errorf("lock Research Anchor import Theme: %w", err)
		}
		existing, err := tx.ResearchAnchorImportReceipt(ctx, publication.ThemeID)
		if err != nil {
			return fmt.Errorf("load Research Anchor import receipt: %w", err)
		}
		if existing != nil {
			if err := validateReplay(*existing, publisherSubject, publicationPlan, publication.ThemeID); err != nil {
				return err
			}
			if err := tx.VerifyResearchAnchorImportReceipt(ctx, *existing); err != nil {
				return fmt.Errorf("verify Research Anchor import replay: %w", err)
			}
			result = resultFromReceipt(*existing, true)
			return nil
		}

		parent, err := tx.ResearchAnchorImportThemePublication(ctx, publication.ThemeID)
		if err != nil {
			return fmt.Errorf("load parent Theme publication: %w", err)
		}
		if parent == nil {
			return referenceError(ReferenceNotFound, "", "theme_id", publication.ThemeID, "Theme does not exist")
		}
		if parent.ThemeImportReceiptID == "" {
			return referenceError(ReferenceInvalid, "", "theme_id", publication.ThemeID, "Theme has no Theme Import V1 receipt")
		}
		if parent.PublisherSubject != publisherSubject {
			return ErrPublisherConflict
		}
		if err := validateReferences(ctx, tx, publication); err != nil {
			return err
		}

		publishedAt := s.now().UTC().Truncate(time.Microsecond)
		receipt := Receipt{
			ID: publicationPlan.ReceiptID, ThemeID: publication.ThemeID, PublisherSubject: publisherSubject,
			PayloadHash:                  publicationPlan.PayloadHash,
			AnchorIDsByCenterChainNodeID: cloneStringMap(publicationPlan.AnchorIDsByCenterChainNodeID),
			Counts:                       publicationPlan.Counts, PublishedAt: publishedAt, ImportedAt: publishedAt,
		}
		if err := tx.InsertResearchAnchorImportReceipt(ctx, receipt); err != nil {
			return fmt.Errorf("insert Research Anchor import receipt: %w", err)
		}
		for _, anchor := range publication.Anchors {
			anchorID := publicationPlan.AnchorIDsByCenterChainNodeID[anchor.CenterChainNodeID]
			if err := tx.InsertResearchAnchor(ctx, repositories.ResearchAnchorImportAnchor{
				ID: anchorID, ThemeID: publication.ThemeID, CenterChainNodeEntityID: anchor.CenterChainNodeID,
				ImportReceiptID: receipt.ID, OneLineConclusion: anchor.OneLineConclusion,
				FactSummary: anchor.FactSummary, NetDirectionSummary: anchor.NetDirectionSummary,
				SupportSummary: anchor.SupportSummary, CounterSummary: anchor.CounterSummary,
				TradingDirection: anchor.TradingDirection, NextCheckpoint: anchor.NextCheckpoint,
			}); err != nil {
				return fmt.Errorf("insert Anchor %q: %w", anchor.CenterChainNodeID, err)
			}
			for _, event := range anchor.Events {
				if err := tx.InsertResearchAnchorEvent(ctx, repositories.ResearchAnchorImportEvent{
					AnchorID: anchorID, EventID: event.EventID, EvidenceRole: event.EvidenceRole, EvidenceSummary: event.EvidenceSummary,
				}); err != nil {
					return fmt.Errorf("insert Anchor %q Event %q: %w", anchor.CenterChainNodeID, event.EventID, err)
				}
			}
			for index, node := range anchor.PathNodes {
				if err := tx.InsertResearchAnchorPathNode(ctx, repositories.ResearchAnchorImportPathNode{
					AnchorID: anchorID, Position: index + 1, ChainNodeEntityID: node.ChainNodeID,
					ChangeDirection: node.ChangeDirection, ChangeSummary: node.ChangeSummary,
					ImpactSummary: node.ImpactSummary, IncomingTransmissionMechanism: node.IncomingTransmissionMechanism,
				}); err != nil {
					return fmt.Errorf("insert Anchor %q Path Node %d: %w", anchor.CenterChainNodeID, index, err)
				}
			}
		}
		if err := tx.VerifyResearchAnchorImportReceipt(ctx, receipt); err != nil {
			return fmt.Errorf("verify Research Anchor import: %w", err)
		}
		result = resultFromReceipt(receipt, false)
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func buildPlan(publication domainimport.Publication) (plan, error) {
	if err := publication.Validate(); err != nil {
		return plan{}, err
	}
	payloadHash, err := domainimport.CanonicalHash(publication)
	if err != nil {
		return plan{}, fmt.Errorf("hash Research Anchor publication: %w", err)
	}
	anchorIDs := make(map[string]string, len(publication.Anchors))
	counts := Counts{Anchors: len(publication.Anchors), Receipts: 1}
	for _, anchor := range publication.Anchors {
		anchorIDs[anchor.CenterChainNodeID] = domainimport.AnchorID(publication.ThemeID, anchor.CenterChainNodeID)
		counts.EventAssociations += len(anchor.Events)
		counts.PathNodes += len(anchor.PathNodes)
	}
	return plan{
		ReceiptID:   repositories.NormalizeUUID("research_anchor_import_receipt", publication.ThemeID),
		PayloadHash: payloadHash, AnchorIDsByCenterChainNodeID: anchorIDs, Counts: counts,
	}, nil
}

func validateReferences(ctx context.Context, tx Transaction, publication domainimport.Publication) error {
	parentCenters, err := tx.ResearchAnchorImportThemeChainNodes(ctx, publication.ThemeID)
	if err != nil {
		return fmt.Errorf("resolve parent Theme Chain Nodes: %w", err)
	}
	parentEvents, err := tx.ResearchAnchorImportThemeEvents(ctx, publication.ThemeID)
	if err != nil {
		return fmt.Errorf("resolve parent Theme Events: %w", err)
	}
	nodeIDs, eventIDs := referencedIDs(publication)
	existingNodes, err := tx.ExistingResearchAnchorChainNodes(ctx, nodeIDs)
	if err != nil {
		return fmt.Errorf("resolve Research Anchor Chain Nodes: %w", err)
	}
	existingEvents, err := tx.ExistingResearchAnchorEvents(ctx, eventIDs)
	if err != nil {
		return fmt.Errorf("resolve Research Anchor Events: %w", err)
	}

	requestCenters := make(map[string]struct{}, len(publication.Anchors))
	for anchorIndex, anchor := range publication.Anchors {
		centerPath := fmt.Sprintf("anchors[%d].center_chain_node_id", anchorIndex)
		if _, exists := existingNodes[anchor.CenterChainNodeID]; !exists {
			return referenceError(ReferenceNotFound, anchor.CenterChainNodeID, centerPath, anchor.CenterChainNodeID, "center Chain Node does not exist")
		}
		if _, allowed := parentCenters[anchor.CenterChainNodeID]; !allowed {
			return referenceError(ReferenceInvalid, anchor.CenterChainNodeID, centerPath, anchor.CenterChainNodeID, "center Chain Node is outside the parent Theme")
		}
		requestCenters[anchor.CenterChainNodeID] = struct{}{}
		for eventIndex, event := range anchor.Events {
			path := fmt.Sprintf("anchors[%d].events[%d].event_id", anchorIndex, eventIndex)
			if _, exists := existingEvents[event.EventID]; !exists {
				return referenceError(ReferenceNotFound, anchor.CenterChainNodeID, path, event.EventID, "Event does not exist")
			}
			if _, allowed := parentEvents[event.EventID]; !allowed {
				return referenceError(ReferenceInvalid, anchor.CenterChainNodeID, path, event.EventID, "Event is outside the parent Theme evidence set")
			}
		}
		for nodeIndex, node := range anchor.PathNodes {
			path := fmt.Sprintf("anchors[%d].path_nodes[%d].chain_node_id", anchorIndex, nodeIndex)
			if _, exists := existingNodes[node.ChainNodeID]; !exists {
				return referenceError(ReferenceNotFound, anchor.CenterChainNodeID, path, node.ChainNodeID, "Path Node does not exist")
			}
		}
	}
	if len(requestCenters) != len(parentCenters) {
		missing := make([]string, 0)
		for centerID := range parentCenters {
			if _, exists := requestCenters[centerID]; !exists {
				missing = append(missing, centerID)
			}
		}
		sort.Strings(missing)
		reference := ""
		if len(missing) > 0 {
			reference = missing[0]
		}
		return &ContractError{Path: "anchors", Reference: reference, Message: "must cover every parent Theme Chain Node exactly once"}
	}
	return nil
}

func referencedIDs(publication domainimport.Publication) ([]string, []string) {
	nodes := make(map[string]struct{})
	events := make(map[string]struct{})
	for _, anchor := range publication.Anchors {
		nodes[anchor.CenterChainNodeID] = struct{}{}
		for _, event := range anchor.Events {
			events[event.EventID] = struct{}{}
		}
		for _, node := range anchor.PathNodes {
			nodes[node.ChainNodeID] = struct{}{}
		}
	}
	return sortedKeys(nodes), sortedKeys(events)
}

func sortedKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func referenceError(kind ReferenceKind, centerID, path, reference, message string) *ReferenceError {
	return &ReferenceError{Kind: kind, CenterChainNodeID: centerID, Path: path, Reference: reference, Message: message}
}

func validateReplay(receipt Receipt, publisherSubject string, publicationPlan plan, themeID string) error {
	if receipt.PublisherSubject != publisherSubject {
		return ErrPublisherConflict
	}
	if receipt.PayloadHash != publicationPlan.PayloadHash {
		return ErrPayloadConflict
	}
	if receipt.ThemeID != themeID || receipt.Counts != publicationPlan.Counts || !reflect.DeepEqual(receipt.AnchorIDsByCenterChainNodeID, publicationPlan.AnchorIDsByCenterChainNodeID) {
		return errors.New("Research Anchor import receipt does not match deterministic plan")
	}
	return nil
}

func resultFromReceipt(receipt Receipt, replayed bool) Result {
	return Result{
		ReceiptID: receipt.ID, ThemeID: receipt.ThemeID, PayloadHash: receipt.PayloadHash,
		AnchorIDsByCenterChainNodeID: cloneStringMap(receipt.AnchorIDsByCenterChainNodeID), Counts: receipt.Counts,
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
