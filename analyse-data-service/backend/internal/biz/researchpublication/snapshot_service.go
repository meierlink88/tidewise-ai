package researchpublication

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/identity"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchreasoningtreeimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
)

func (s *Service) PublishSnapshot(ctx context.Context, publisher string, aggregate SnapshotAggregate) (Result, error) {
	if s == nil || s.store == nil {
		return Result{}, errors.New("research publication store is required")
	}
	publisher = strings.TrimSpace(publisher)
	if publisher == "" || len(publisher) > 200 {
		return Result{}, errors.New("publisher subject must contain 1..200 characters")
	}
	analysisAsOf, themeID, err := aggregate.Validate()
	if err != nil {
		return Result{}, err
	}
	payloadHash, err := CanonicalSnapshotHash(aggregate)
	if err != nil {
		return Result{}, fmt.Errorf("hash research snapshot publication: %w", err)
	}
	plan := snapshotPublicationPlan(aggregate, themeID, payloadHash)
	var result Result
	err = s.store.InResearchPublicationTransaction(ctx, func(tx Transaction) error {
		if err := tx.Lock(ctx, aggregate.AnalysisBatchID); err != nil {
			return fmt.Errorf("lock research publication: %w", err)
		}
		existing, err := tx.Receipt(ctx, aggregate.AnalysisBatchID)
		if err != nil {
			return fmt.Errorf("load research publication receipt: %w", err)
		}
		if existing != nil {
			if existing.ContractVersion != 3 || existing.PublicationMode != SnapshotPublicationMode {
				return ErrPayloadConflict
			}
			if existing.PublisherSubject != publisher {
				return ErrPublisherConflict
			}
			if existing.PayloadHash != payloadHash {
				return ErrPayloadConflict
			}
			if existing.ThemeID != plan.ThemeID ||
				!reflect.DeepEqual(existing.ReasoningTreeIDsByTreeKey, plan.ReasoningTreeIDsByTreeKey) ||
				existing.Counts != plan.Counts {
				return errors.New("research snapshot publication receipt does not match deterministic plan")
			}
			if err := tx.Verify(ctx, *existing); err != nil {
				return fmt.Errorf("verify research snapshot publication replay: %w", err)
			}
			result = resultFromReceipt(*existing, true)
			return nil
		}

		facts, err := tx.ReferenceFacts(ctx, snapshotReferenceQuery(aggregate))
		if err != nil {
			return fmt.Errorf("load research snapshot references: %w", err)
		}
		if err := validateSnapshotReferences(aggregate, facts); err != nil {
			return err
		}

		publishedAt := s.now().UTC().Truncate(time.Microsecond)
		receipt := plan
		receipt.PublisherSubject = publisher
		receipt.PublishedAt, receipt.ImportedAt = publishedAt, publishedAt
		if err := tx.InsertThemeReceipt(ctx, receipt); err != nil {
			return fmt.Errorf("insert aggregate receipt: %w", err)
		}
		windowStart, _ := time.Parse(time.RFC3339, aggregate.DiscoveryWindowStart)
		windowEnd, _ := time.Parse(time.RFC3339, aggregate.DiscoveryWindowEnd)
		theme := aggregate.Theme
		if err := tx.InsertTheme(ctx, researchthemeimport.ThemeRecord{
			ID: themeID, ImportReceiptID: receipt.ID, AnalysisBatchID: aggregate.AnalysisBatchID,
			ThemeKey: theme.ThemeKey, Title: theme.Title, OneLineConclusion: theme.OneLineConclusion,
			ConclusionDirection: theme.ConclusionDirection, ImpactStrength: theme.ImpactStrength,
			AttentionLevel: theme.AttentionLevel, ConclusionStatus: theme.ConclusionStatus,
			TransmissionStage: theme.TransmissionStage, InvestmentGuidanceAction: theme.InvestmentGuidanceAction,
			InvestmentGuidanceSummary: theme.InvestmentGuidanceSummary,
			TimeHorizonCategory:       theme.TimeHorizonCategory, TimeHorizonSummary: theme.TimeHorizonSummary,
			TransmissionSummary: theme.TransmissionSummary, CheckpointSummary: theme.CheckpointSummary,
			RiskSummary: theme.RiskSummary, AnalysisAsOf: analysisAsOf,
			WindowStart: windowStart, WindowEnd: windowEnd, PublishedAt: publishedAt,
		}); err != nil {
			return fmt.Errorf("insert Theme: %w", err)
		}
		for _, impact := range theme.Impacts {
			if err := tx.InsertSnapshotThemeImpact(ctx, SnapshotImpactRecord{
				ThemeID: themeID, NodeKey: impact.NodeKey, DisplayName: impact.DisplayName,
				RelationRole: impact.RelationRole, ImpactDirection: impact.ImpactDirection,
				ImpactSummary: impact.ImpactSummary, DisplayOrder: impact.DisplayOrder,
			}); err != nil {
				return fmt.Errorf("insert snapshot Theme Impact: %w", err)
			}
		}
		for _, event := range theme.Events {
			if err := tx.InsertThemeEvent(ctx, researchthemeimport.EventRecord{
				ThemeID: themeID, EventID: event.EventID, EvidenceRole: event.EvidenceRole,
				SupportedClaim: event.SupportedClaim, EvidenceIDs: append([]string(nil), event.EvidenceIDs...),
			}); err != nil {
				return fmt.Errorf("insert Theme Event: %w", err)
			}
		}

		treeReceipt := SnapshotTreeReceipt{
			ID:      identity.NormalizeUUID("research_reasoning_tree_import_receipt", themeID),
			ThemeID: themeID, PublisherSubject: publisher, PayloadHash: payloadHash,
			ReasoningTreeIDsByTreeKey: cloneMap(plan.ReasoningTreeIDsByTreeKey),
			Counts: researchreasoningtreeimport.Counts{
				ReasoningTrees: plan.Counts.ReasoningTrees, Nodes: plan.Counts.Nodes,
				EventAssociations:  plan.Counts.TreeEventAssociations,
				SignalAssociations: plan.Counts.SignalAssociations, Receipts: 1,
			},
			PublishedAt: publishedAt, ImportedAt: publishedAt,
		}
		if err := tx.InsertSnapshotTreeReceipt(ctx, treeReceipt); err != nil {
			return fmt.Errorf("insert snapshot Reason Tree receipt: %w", err)
		}
		for _, tree := range aggregate.ReasoningTrees {
			treeID := plan.ReasoningTreeIDsByTreeKey[tree.TreeKey]
			if err := tx.InsertSnapshotTree(ctx, SnapshotTreeRecord{
				ID: treeID, ThemeID: themeID, ImportReceiptID: treeReceipt.ID,
				TreeKey: tree.TreeKey, DisplayName: tree.DisplayName, Title: tree.Title,
				DisplayOrder: tree.DisplayOrder, OneLineConclusion: tree.OneLineConclusion,
				FactSummary: tree.FactSummary, TransmissionSummary: tree.TransmissionSummary,
				ImpactDirection: tree.ImpactDirection, ImpactStrength: tree.ImpactStrength,
				ImpactSummary: tree.ImpactSummary, ConclusionBoundarySummary: tree.ConclusionBoundarySummary,
				SupportSummary: tree.SupportSummary, CounterSummary: tree.CounterSummary,
				InvalidationConditions: append([]string(nil), tree.InvalidationConditions...),
				Checkpoints:            append([]researchreasoningtreeimport.Checkpoint(nil), tree.Checkpoints...),
			}); err != nil {
				return fmt.Errorf("insert snapshot Reason Tree: %w", err)
			}
			for _, event := range tree.Events {
				if err := tx.InsertTreeEvent(ctx, researchreasoningtreeimport.EventRecord{
					ReasoningTreeID: treeID, EventID: event.EventID, EvidenceRole: event.EvidenceRole,
					DisplayOrder: event.DisplayOrder, EvidenceIDs: append([]string(nil), event.EvidenceIDs...),
				}); err != nil {
					return fmt.Errorf("insert snapshot Reason Tree Event: %w", err)
				}
			}
			for _, node := range tree.Nodes {
				nodeID := SnapshotNodeID(treeID, node.NodeKey)
				record := SnapshotNodeRecord{
					ID: nodeID, ReasoningTreeID: treeID, NodeKey: node.NodeKey, DisplayName: node.DisplayName,
					Position: node.Position, StateSummary: node.StateSummary,
					ImpactDirection: node.ImpactDirection, ImpactStrength: node.ImpactStrength,
					ImpactSummary: node.ImpactSummary, ReasoningBasisSummary: node.ReasoningBasisSummary,
					EvidenceGapSummary: node.EvidenceGapSummary,
				}
				if node.IncomingTransmission != nil {
					record.IncomingTransmissionTitle = node.IncomingTransmission.Title
					record.IncomingTransmissionMechanism = &node.IncomingTransmission.Mechanism
					record.IncomingConditionSummary = node.IncomingTransmission.ConditionSummary
				}
				if err := tx.InsertSnapshotNode(ctx, record); err != nil {
					return fmt.Errorf("insert snapshot Reason Tree Node: %w", err)
				}
				for _, signal := range node.Signals {
					if err := tx.InsertSnapshotSignal(ctx, SnapshotSignalRecord{
						ReasoningTreeNodeID: nodeID, SignalKey: signal.SignalKey,
						SignalRole: signal.Role, DisplaySummary: signal.DisplaySummary,
						VariableName: signal.VariableName, SignalDirection: signal.Direction,
						DisplayOrder: signal.DisplayOrder,
					}); err != nil {
						return fmt.Errorf("insert snapshot Reason Tree Signal: %w", err)
					}
				}
			}
		}
		if err := tx.Verify(ctx, receipt); err != nil {
			return fmt.Errorf("verify research snapshot publication: %w", err)
		}
		result = resultFromReceipt(receipt, false)
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func snapshotPublicationPlan(a SnapshotAggregate, themeID, payloadHash string) Receipt {
	treeIDs := make(map[string]string, len(a.ReasoningTrees))
	counts := Counts{Themes: 1, Impacts: len(a.Theme.Impacts), ThemeEventAssociations: len(a.Theme.Events), Receipts: 2}
	for _, tree := range a.ReasoningTrees {
		treeIDs[tree.TreeKey] = SnapshotTreeID(themeID, tree.TreeKey)
		counts.ReasoningTrees++
		counts.TreeEventAssociations += len(tree.Events)
		counts.Nodes += len(tree.Nodes)
		for _, node := range tree.Nodes {
			counts.SignalAssociations += len(node.Signals)
		}
	}
	return Receipt{
		ID:              identity.NormalizeUUID("research_theme_import_receipt", a.AnalysisBatchID),
		AnalysisBatchID: a.AnalysisBatchID, PayloadHash: payloadHash, ThemeID: themeID,
		ThemeKey: a.Theme.ThemeKey, ContractVersion: 3, PublicationMode: SnapshotPublicationMode,
		ReasoningTreeIDsByIndustryChainEntityID: map[string]string{},
		ReasoningTreeIDsByTreeKey:               treeIDs, Counts: counts,
	}
}

func snapshotReferenceQuery(a SnapshotAggregate) ReferenceQuery {
	query := ReferenceQuery{SnapshotEventExistenceOnly: true}
	for _, event := range a.Theme.Events {
		query.EventIDs = append(query.EventIDs, event.EventID)
		query.EvidenceIDs = append(query.EvidenceIDs, event.EvidenceIDs...)
	}
	for _, tree := range a.ReasoningTrees {
		for _, event := range tree.Events {
			query.EventIDs = append(query.EventIDs, event.EventID)
			query.EvidenceIDs = append(query.EvidenceIDs, event.EvidenceIDs...)
		}
	}
	return query
}

func validateSnapshotReferences(a SnapshotAggregate, facts ReferenceFacts) error {
	validate := func(path string, eventID string, evidenceIDs []string) error {
		if _, ok := facts.Events[eventID]; !ok {
			return invalidReference(path+".event_id", eventID, "Event does not exist")
		}
		for index, evidenceID := range evidenceIDs {
			evidence, ok := facts.Evidences[evidenceID]
			if !ok || evidence.EventID != eventID {
				return invalidReference(fmt.Sprintf("%s.evidence_ids[%d]", path, index), evidenceID, "Evidence does not belong to Event")
			}
		}
		return nil
	}
	for index, event := range a.Theme.Events {
		if err := validate(fmt.Sprintf("theme.events[%d]", index), event.EventID, event.EvidenceIDs); err != nil {
			return err
		}
	}
	for treeIndex, tree := range a.ReasoningTrees {
		for eventIndex, event := range tree.Events {
			if err := validate(fmt.Sprintf("reasoning_trees[%d].events[%d]", treeIndex, eventIndex), event.EventID, event.EvidenceIDs); err != nil {
				return err
			}
		}
	}
	return nil
}
