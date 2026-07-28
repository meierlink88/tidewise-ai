package researchreasoningtreeimport

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/identity"
)

var (
	ErrPayloadConflict   = errors.New("Theme conflicts with the published Reason Tree payload")
	ErrPublisherConflict = errors.New("Reason Tree publisher does not own the parent Theme publication")
)

type ReferenceKind uint8

const (
	ReferenceNotFound ReferenceKind = iota + 1
	ReferenceInvalid
)

type ReferenceError struct {
	Kind                  ReferenceKind
	IndustryChainEntityID string
	Path                  string
	Reference             string
	Message               string
}

func (e *ReferenceError) Error() string {
	return fmt.Sprintf("%s: %s (%s)", e.Path, e.Message, e.Reference)
}

type ContractError struct {
	Path, Reference, Message string
}

func (e *ContractError) Error() string {
	return fmt.Sprintf("%s: %s (%s)", e.Path, e.Message, e.Reference)
}

type Result struct {
	ReceiptID                               string            `json:"receipt_id"`
	ThemeID                                 string            `json:"theme_id"`
	PayloadHash                             string            `json:"payload_hash"`
	ReasoningTreeIDsByIndustryChainEntityID map[string]string `json:"reasoning_tree_ids_by_industry_chain_entity_id"`
	Counts                                  Counts            `json:"counts"`
	PublishedAt                             time.Time         `json:"published_at"`
	ImportedAt                              time.Time         `json:"imported_at"`
	Replayed                                bool              `json:"replayed"`
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{store: store, now: func() time.Time { return time.Now().UTC() }}
}

type plan struct {
	ReceiptID                               string
	PayloadHash                             string
	ReasoningTreeIDsByIndustryChainEntityID map[string]string
	Counts                                  Counts
}

func (s *Service) Import(ctx context.Context, publisherSubject string, publication Publication) (Result, error) {
	if s == nil || s.store == nil {
		return Result{}, errors.New("Research Reason Tree import store is required")
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
	err = s.store.InResearchReasoningTreeImportTransaction(ctx, func(tx Transaction) error {
		if err := tx.LockResearchReasoningTreeImportTheme(ctx, publication.ThemeID); err != nil {
			return fmt.Errorf("lock Research Reason Tree import Theme: %w", err)
		}
		existing, err := tx.ResearchReasoningTreeImportReceipt(ctx, publication.ThemeID)
		if err != nil {
			return fmt.Errorf("load Research Reason Tree import receipt: %w", err)
		}
		if existing != nil {
			if err := validateReplay(*existing, publisherSubject, publicationPlan, publication.ThemeID); err != nil {
				return err
			}
			if err := tx.VerifyResearchReasoningTreeImportReceipt(ctx, *existing); err != nil {
				return fmt.Errorf("verify Research Reason Tree import replay: %w", err)
			}
			result = resultFromReceipt(*existing, true)
			return nil
		}

		parent, err := tx.ResearchReasoningTreeImportThemePublication(ctx, publication.ThemeID)
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
		if err := tx.LockResearchReasoningTreeAnalysisBatch(ctx, parent.AnalysisBatchID); err != nil {
			return fmt.Errorf("lock Research Reason Tree analysis batch: %w", err)
		}
		snapshots := publicationSignalSnapshots(publication)
		keys := make([]string, 0, len(snapshots))
		for key := range snapshots {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		existingSnapshots, err := tx.ResearchReasoningTreeSignalSnapshots(ctx, parent.AnalysisBatchID, keys)
		if err != nil {
			return fmt.Errorf("load existing Variable Signal snapshots: %w", err)
		}
		if err := validateExistingSignalSnapshots(snapshots, existingSnapshots); err != nil {
			return err
		}
		if err := validateReferences(ctx, tx, publication, *parent); err != nil {
			return err
		}

		publishedAt := s.now().UTC().Truncate(time.Microsecond)
		receipt := Receipt{
			ID: publicationPlan.ReceiptID, ThemeID: publication.ThemeID,
			PublisherSubject: publisherSubject, PayloadHash: publicationPlan.PayloadHash,
			ReasoningTreeIDsByIndustryChainEntityID: cloneStringMap(publicationPlan.ReasoningTreeIDsByIndustryChainEntityID),
			Counts:                                  publicationPlan.Counts, PublishedAt: publishedAt, ImportedAt: publishedAt,
		}
		if err := tx.InsertResearchReasoningTreeImportReceipt(ctx, receipt); err != nil {
			return fmt.Errorf("insert Research Reason Tree import receipt: %w", err)
		}
		for _, tree := range publication.ReasoningTrees {
			treeID := publicationPlan.ReasoningTreeIDsByIndustryChainEntityID[tree.IndustryChainEntityID]
			if err := tx.InsertResearchReasoningTree(ctx, ReasoningTreeRecord{
				ID: treeID, ThemeID: publication.ThemeID, ImportReceiptID: receipt.ID,
				IndustryChainEntityID: tree.IndustryChainEntityID, Title: tree.Title,
				DisplayOrder: tree.DisplayOrder, OneLineConclusion: tree.OneLineConclusion,
				FactSummary: tree.FactSummary, TransmissionSummary: tree.TransmissionSummary,
				ImpactDirection: tree.ImpactDirection, ImpactStrength: tree.ImpactStrength,
				ImpactSummary: tree.ImpactSummary, ConclusionBoundarySummary: tree.ConclusionBoundarySummary,
				SupportSummary: tree.SupportSummary, CounterSummary: tree.CounterSummary,
				InvalidationConditions: append([]string(nil), tree.InvalidationConditions...),
				Checkpoints:            append([]Checkpoint(nil), tree.Checkpoints...),
			}); err != nil {
				return fmt.Errorf("insert Reason Tree %q: %w", tree.IndustryChainEntityID, err)
			}
			for _, event := range tree.Events {
				if err := tx.InsertResearchReasoningTreeEvent(ctx, EventRecord{
					ReasoningTreeID: treeID, EventID: event.EventID,
					EvidenceRole: event.EvidenceRole, DisplayOrder: event.DisplayOrder,
				}); err != nil {
					return fmt.Errorf("insert Reason Tree %q Event %q: %w", tree.IndustryChainEntityID, event.EventID, err)
				}
			}
			for _, node := range tree.Nodes {
				nodeID := ReasoningTreeNodeID(treeID, node.Position, node.ChainNodeEntityID)
				if err := tx.InsertResearchReasoningTreeNode(ctx, NodeRecord{
					ID: nodeID, ReasoningTreeID: treeID, Position: node.Position,
					ChainNodeEntityID: node.ChainNodeEntityID, StateSummary: node.StateSummary,
					ImpactDirection: node.ImpactDirection, ImpactStrength: node.ImpactStrength,
					ImpactSummary: node.ImpactSummary, ReasoningBasisSummary: node.ReasoningBasisSummary,
					EvidenceGapSummary:               node.EvidenceGapSummary,
					IncomingIndustryChainGraphEdgeID: node.IncomingIndustryChainGraphEdgeID,
					IncomingTransmissionTitle:        node.IncomingTransmissionTitle,
					IncomingTransmissionMechanism:    node.IncomingTransmissionMechanism,
					IncomingConditionSummary:         node.IncomingConditionSummary,
				}); err != nil {
					return fmt.Errorf("insert Reason Tree %q Node %d: %w", tree.IndustryChainEntityID, node.Position, err)
				}
				for _, signal := range node.Signals {
					if err := tx.InsertResearchReasoningTreeNodeSignal(ctx, SignalRecord{
						ReasoningTreeNodeID: nodeID, VariableSignalKey: signal.VariableSignalKey,
						SignalRole: signal.SignalRole, SignalDirection: signal.SignalDirection,
						DisplaySummary: signal.DisplaySummary, DisplayOrder: signal.DisplayOrder,
					}); err != nil {
						return fmt.Errorf("insert Reason Tree %q Node %d Signal %q: %w", tree.IndustryChainEntityID, node.Position, signal.VariableSignalKey, err)
					}
				}
			}
		}
		if err := tx.VerifyResearchReasoningTreeImportReceipt(ctx, receipt); err != nil {
			return fmt.Errorf("verify Research Reason Tree import: %w", err)
		}
		result = resultFromReceipt(receipt, false)
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func publicationSignalSnapshots(publication Publication) map[string]SignalSnapshot {
	result := make(map[string]SignalSnapshot)
	for _, tree := range publication.ReasoningTrees {
		for _, node := range tree.Nodes {
			for _, signal := range node.Signals {
				result[signal.VariableSignalKey] = SignalSnapshot{
					SignalDirection: signal.SignalDirection,
					DisplaySummary:  signal.DisplaySummary,
				}
			}
		}
	}
	return result
}

func validateExistingSignalSnapshots(
	incoming, existing map[string]SignalSnapshot,
) error {
	for key, snapshot := range incoming {
		if current, ok := existing[key]; ok && current != snapshot {
			return &ContractError{
				Path:      "reasoning_trees.nodes.signals.variable_signal_key",
				Reference: key,
				Message:   "must keep the same direction and display summary within the analysis batch",
			}
		}
	}
	return nil
}

func buildPlan(publication Publication) (plan, error) {
	if err := publication.Validate(); err != nil {
		return plan{}, err
	}
	payloadHash, err := CanonicalHash(publication)
	if err != nil {
		return plan{}, err
	}
	treeIDs := make(map[string]string, len(publication.ReasoningTrees))
	counts := Counts{ReasoningTrees: len(publication.ReasoningTrees), Receipts: 1}
	for _, tree := range publication.ReasoningTrees {
		treeIDs[tree.IndustryChainEntityID] = ReasoningTreeID(publication.ThemeID, tree.IndustryChainEntityID)
		counts.EventAssociations += len(tree.Events)
		counts.Nodes += len(tree.Nodes)
		for _, node := range tree.Nodes {
			counts.SignalAssociations += len(node.Signals)
		}
	}
	return plan{
		ReceiptID:   identity.NormalizeUUID("research_reasoning_tree_import_receipt", publication.ThemeID),
		PayloadHash: payloadHash, ReasoningTreeIDsByIndustryChainEntityID: treeIDs, Counts: counts,
	}, nil
}

func validateReferences(ctx context.Context, tx Transaction, publication Publication, parent ThemePublication) error {
	chainIDs := make([]string, 0, len(publication.ReasoningTrees))
	graphEdgeIDs := make([]string, 0)
	for _, tree := range publication.ReasoningTrees {
		chainIDs = append(chainIDs, tree.IndustryChainEntityID)
		for _, node := range tree.Nodes {
			if node.IncomingIndustryChainGraphEdgeID != nil {
				graphEdgeIDs = append(graphEdgeIDs, *node.IncomingIndustryChainGraphEdgeID)
			}
		}
	}
	existingChains, err := tx.ExistingResearchReasoningTreeIndustryChains(ctx, chainIDs)
	if err != nil {
		return fmt.Errorf("resolve Industry Chains: %w", err)
	}
	memberships, err := tx.ResearchReasoningTreeChainMemberships(ctx, chainIDs)
	if err != nil {
		return fmt.Errorf("resolve Industry Chain memberships: %w", err)
	}
	graphEdges, err := tx.ResearchReasoningTreeGraphEdges(ctx, graphEdgeIDs)
	if err != nil {
		return fmt.Errorf("resolve Industry Chain Graph Edges: %w", err)
	}

	coveredImpacts := make(map[string]struct{})
	for treeIndex, tree := range publication.ReasoningTrees {
		treePath := fmt.Sprintf("reasoning_trees[%d]", treeIndex)
		if _, exists := existingChains[tree.IndustryChainEntityID]; !exists {
			return referenceError(ReferenceNotFound, tree.IndustryChainEntityID, treePath+".industry_chain_entity_id", tree.IndustryChainEntityID, "active approved Industry Chain does not exist")
		}
		treeImpactCount := 0
		for eventIndex, event := range tree.Events {
			if _, allowed := parent.EventIDs[event.EventID]; !allowed {
				return referenceError(ReferenceInvalid, tree.IndustryChainEntityID, fmt.Sprintf("%s.events[%d].event_id", treePath, eventIndex), event.EventID, "Event is outside the parent Theme Event set")
			}
		}
		for nodeIndex, node := range tree.Nodes {
			nodePath := fmt.Sprintf("%s.nodes[%d]", treePath, nodeIndex)
			if _, member := memberships[tree.IndustryChainEntityID][node.ChainNodeEntityID]; !member {
				return referenceError(ReferenceInvalid, tree.IndustryChainEntityID, nodePath+".chain_node_entity_id", node.ChainNodeEntityID, "Node is not an active approved member of the Industry Chain")
			}
			if _, impact := parent.ImpactNodeIDs[node.ChainNodeEntityID]; impact {
				coveredImpacts[node.ChainNodeEntityID] = struct{}{}
				treeImpactCount++
			}
			if node.IncomingIndustryChainGraphEdgeID != nil {
				edgeID := *node.IncomingIndustryChainGraphEdgeID
				edge, exists := graphEdges[edgeID]
				if !exists {
					return referenceError(ReferenceNotFound, tree.IndustryChainEntityID, nodePath+".incoming_industry_chain_graph_edge_id", edgeID, "active approved Graph Edge does not exist")
				}
				previousNodeID := tree.Nodes[nodeIndex-1].ChainNodeEntityID
				if edge.IndustryChainEntityID != tree.IndustryChainEntityID ||
					edge.FromChainNodeEntityID != previousNodeID ||
					edge.ToChainNodeEntityID != node.ChainNodeEntityID {
					return referenceError(ReferenceInvalid, tree.IndustryChainEntityID, nodePath+".incoming_industry_chain_graph_edge_id", edgeID, "Graph Edge does not match the ordered incoming path")
				}
			}
		}
		if treeImpactCount == 0 {
			return &ContractError{Path: treePath + ".nodes", Reference: tree.IndustryChainEntityID, Message: "must contain at least one parent Theme Impact"}
		}
	}
	if len(coveredImpacts) != len(parent.ImpactNodeIDs) {
		missing := make([]string, 0)
		for nodeID := range parent.ImpactNodeIDs {
			if _, covered := coveredImpacts[nodeID]; !covered {
				missing = append(missing, nodeID)
			}
		}
		sort.Strings(missing)
		reference := ""
		if len(missing) > 0 {
			reference = missing[0]
		}
		return &ContractError{Path: "reasoning_trees", Reference: reference, Message: "must cover every parent Theme Impact"}
	}
	return nil
}

func validateReplay(receipt Receipt, publisherSubject string, publicationPlan plan, themeID string) error {
	if receipt.PublisherSubject != publisherSubject {
		return ErrPublisherConflict
	}
	if receipt.PayloadHash != publicationPlan.PayloadHash {
		return ErrPayloadConflict
	}
	if receipt.ThemeID != themeID || receipt.Counts != publicationPlan.Counts ||
		!reflect.DeepEqual(receipt.ReasoningTreeIDsByIndustryChainEntityID, publicationPlan.ReasoningTreeIDsByIndustryChainEntityID) {
		return errors.New("Research Reason Tree import receipt does not match deterministic plan")
	}
	return nil
}

func resultFromReceipt(receipt Receipt, replayed bool) Result {
	return Result{
		ReceiptID: receipt.ID, ThemeID: receipt.ThemeID, PayloadHash: receipt.PayloadHash,
		ReasoningTreeIDsByIndustryChainEntityID: cloneStringMap(receipt.ReasoningTreeIDsByIndustryChainEntityID),
		Counts:                                  receipt.Counts, PublishedAt: receipt.PublishedAt.UTC(),
		ImportedAt: receipt.ImportedAt.UTC(), Replayed: replayed,
	}
}

func referenceError(kind ReferenceKind, chainID, path, reference, message string) *ReferenceError {
	return &ReferenceError{Kind: kind, IndustryChainEntityID: chainID, Path: path, Reference: reference, Message: message}
}

func cloneStringMap(input map[string]string) map[string]string {
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
