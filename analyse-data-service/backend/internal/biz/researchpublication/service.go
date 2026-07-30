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

var (
	ErrPayloadConflict   = errors.New("analysis batch conflicts with the published aggregate")
	ErrPublisherConflict = errors.New("analysis batch belongs to another publisher subject")
)

type Result struct {
	ReceiptID, AnalysisBatchID, PayloadHash, ThemeID string
	ReasoningTreeIDsByIndustryChainEntityID          map[string]string
	Counts                                           Counts
	PublishedAt, ImportedAt                          time.Time
	Replayed                                         bool
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{store: store, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) Publish(ctx context.Context, publisher string, aggregate Aggregate) (Result, error) {
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
	payloadHash, err := CanonicalHash(aggregate)
	if err != nil {
		return Result{}, fmt.Errorf("hash research publication: %w", err)
	}
	plan := publicationPlan(aggregate, themeID, payloadHash)
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
			if existing.ContractVersion != 2 {
				return ErrPayloadConflict
			}
			if existing.PublisherSubject != publisher {
				return ErrPublisherConflict
			}
			if existing.PayloadHash != payloadHash {
				return ErrPayloadConflict
			}
			if existing.ThemeID != plan.ThemeID ||
				!reflect.DeepEqual(existing.ReasoningTreeIDsByIndustryChainEntityID, plan.ReasoningTreeIDsByIndustryChainEntityID) ||
				existing.Counts != plan.Counts {
				return errors.New("research publication receipt does not match deterministic plan")
			}
			if err := tx.Verify(ctx, *existing); err != nil {
				return fmt.Errorf("verify research publication replay: %w", err)
			}
			result = resultFromReceipt(*existing, true)
			return nil
		}

		query := referenceQuery(aggregate)
		facts, err := tx.ReferenceFacts(ctx, query)
		if err != nil {
			return fmt.Errorf("load research publication references: %w", err)
		}
		if err := validateReferences(aggregate, analysisAsOf, facts); err != nil {
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
			TransmissionStage:         theme.TransmissionStage,
			InvestmentGuidanceAction:  theme.InvestmentGuidanceAction,
			InvestmentGuidanceSummary: theme.InvestmentGuidanceSummary,
			TimeHorizonCategory:       theme.TimeHorizonCategory, TimeHorizonSummary: theme.TimeHorizonSummary,
			TransmissionSummary: theme.TransmissionSummary, CheckpointSummary: theme.CheckpointSummary,
			RiskSummary: theme.RiskSummary, AnalysisAsOf: analysisAsOf,
			WindowStart: windowStart, WindowEnd: windowEnd, PublishedAt: publishedAt,
		}); err != nil {
			return fmt.Errorf("insert Theme: %w", err)
		}
		for _, impact := range theme.Impacts {
			if err := tx.InsertThemeImpact(ctx, researchthemeimport.ImpactRecord{
				ThemeID: themeID, ChainNodeEntityID: impact.ChainNodeEntityID,
				RelationRole: impact.RelationRole, ImpactDirection: impact.ImpactDirection,
				ImpactSummary:               impact.ImpactSummary,
				PrimarySignalDisplaySummary: impact.PrimarySignalDisplaySummary,
				DisplayOrder:                impact.DisplayOrder,
			}); err != nil {
				return fmt.Errorf("insert Theme Impact: %w", err)
			}
		}
		for _, event := range theme.Events {
			if err := tx.InsertThemeEvent(ctx, researchthemeimport.EventRecord{
				ThemeID: themeID, EventID: event.EventID, EvidenceRole: event.EvidenceRole,
				SupportedClaim: event.SupportedClaim,
			}); err != nil {
				return fmt.Errorf("insert Theme Event: %w", err)
			}
		}

		treeReceipt := researchreasoningtreeimport.Receipt{
			ID:      identity.NormalizeUUID("research_reasoning_tree_import_receipt", themeID),
			ThemeID: themeID, PublisherSubject: publisher, PayloadHash: payloadHash,
			ReasoningTreeIDsByIndustryChainEntityID: cloneMap(plan.ReasoningTreeIDsByIndustryChainEntityID),
			Counts: researchreasoningtreeimport.Counts{
				ReasoningTrees: plan.Counts.ReasoningTrees, Nodes: plan.Counts.Nodes,
				EventAssociations:  plan.Counts.TreeEventAssociations,
				SignalAssociations: plan.Counts.SignalAssociations, Receipts: 1,
			},
			PublishedAt: publishedAt, ImportedAt: publishedAt,
		}
		if err := tx.InsertTreeReceipt(ctx, treeReceipt); err != nil {
			return fmt.Errorf("insert Reason Tree receipt: %w", err)
		}
		for _, tree := range aggregate.ReasoningTrees {
			treeID := plan.ReasoningTreeIDsByIndustryChainEntityID[tree.IndustryChainEntityID]
			if err := tx.InsertTree(ctx, researchreasoningtreeimport.ReasoningTreeRecord{
				ID: treeID, ThemeID: themeID, ImportReceiptID: treeReceipt.ID,
				IndustryChainEntityID: tree.IndustryChainEntityID, Title: tree.Title,
				DisplayOrder: tree.DisplayOrder, OneLineConclusion: tree.OneLineConclusion,
				FactSummary: tree.FactSummary, TransmissionSummary: tree.TransmissionSummary,
				ImpactDirection: tree.ImpactDirection, ImpactStrength: tree.ImpactStrength,
				ImpactSummary: tree.ImpactSummary, ConclusionBoundarySummary: tree.ConclusionBoundarySummary,
				SupportSummary: tree.SupportSummary, CounterSummary: tree.CounterSummary,
				InvalidationConditions: append([]string(nil), tree.InvalidationConditions...),
				Checkpoints:            append([]researchreasoningtreeimport.Checkpoint(nil), tree.Checkpoints...),
			}); err != nil {
				return fmt.Errorf("insert Reason Tree: %w", err)
			}
			for _, event := range tree.Events {
				if err := tx.InsertTreeEvent(ctx, researchreasoningtreeimport.EventRecord{
					ReasoningTreeID: treeID, EventID: event.EventID,
					EvidenceRole: event.EvidenceRole, DisplayOrder: event.DisplayOrder,
				}); err != nil {
					return fmt.Errorf("insert Reason Tree Event: %w", err)
				}
			}
			for _, node := range tree.Nodes {
				nodeID := researchreasoningtreeimport.ReasoningTreeNodeID(treeID, node.Position, node.ChainNodeEntityID)
				record := NodeRecord{NodeRecord: researchreasoningtreeimport.NodeRecord{
					ID: nodeID, ReasoningTreeID: treeID, Position: node.Position,
					ChainNodeEntityID: node.ChainNodeEntityID, StateSummary: node.StateSummary,
					ImpactDirection: node.ImpactDirection, ImpactStrength: node.ImpactStrength,
					ImpactSummary: node.ImpactSummary, ReasoningBasisSummary: node.ReasoningBasisSummary,
					EvidenceGapSummary:               node.EvidenceGapSummary,
					IncomingIndustryChainGraphEdgeID: node.IncomingIndustryChainGraphEdgeID,
					IncomingTransmissionTitle:        node.IncomingTransmissionTitle,
					IncomingTransmissionMechanism:    node.IncomingTransmissionMechanism,
					IncomingConditionSummary:         node.IncomingConditionSummary,
				}}
				if node.IncomingLineage != nil {
					lineage := node.IncomingLineage
					record.IncomingSourceKind = &lineage.SourceKind
					record.DirectImpactAssertionID = lineage.DirectImpactAssertionID
					record.DirectImpactSemanticSubmissionID = lineage.SemanticSubmissionID
					record.DirectImpactEvidenceID = lineage.EvidenceID
					record.DirectImpactEvidenceHash = lineage.EvidenceHash
					record.DirectImpactAffectedVariableKey = lineage.AffectedVariableKey
					record.DirectImpactAffectedDirection = lineage.AffectedDirection
					record.InferenceUpstreamVariableSignalID = lineage.UpstreamVariableSignalID
					record.InferenceUpstreamDirectImpactAssertionID = lineage.UpstreamDirectImpactAssertionID
					record.InferenceEntityRelationID = lineage.EntityRelationID
				}
				if err := tx.InsertNode(ctx, record); err != nil {
					return fmt.Errorf("insert Reason Tree Node: %w", err)
				}
				for _, signal := range node.Signals {
					if err := tx.InsertSignal(ctx, SignalRecord{
						SignalRecord: researchreasoningtreeimport.SignalRecord{
							ReasoningTreeNodeID: nodeID, VariableSignalKey: signal.VariableSignalKey,
							SignalRole: signal.SignalRole, SignalDirection: signal.SignalDirection,
							DisplaySummary: signal.DisplaySummary, DisplayOrder: signal.DisplayOrder,
						},
						SourceKind:           signal.Lineage.SourceKind,
						VariableSignalID:     signal.Lineage.VariableSignalID,
						SemanticSubmissionID: signal.Lineage.SemanticSubmissionID,
						EvidenceID:           signal.Lineage.EvidenceID, EvidenceHash: signal.Lineage.EvidenceHash,
						UpstreamVariableSignalID:        signal.Lineage.UpstreamVariableSignalID,
						UpstreamDirectImpactAssertionID: signal.Lineage.UpstreamDirectImpactAssertionID,
						EntityRelationID:                signal.Lineage.EntityRelationID,
						IndustryChainGraphEdgeID:        signal.Lineage.IndustryChainGraphEdgeID,
					}); err != nil {
						return fmt.Errorf("insert Reason Tree Signal: %w", err)
					}
				}
			}
		}
		if err := tx.Verify(ctx, receipt); err != nil {
			return fmt.Errorf("verify research publication: %w", err)
		}
		result = resultFromReceipt(receipt, false)
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func publicationPlan(a Aggregate, themeID, payloadHash string) Receipt {
	treeIDs := make(map[string]string, len(a.ReasoningTrees))
	counts := Counts{Themes: 1, Impacts: len(a.Theme.Impacts), ThemeEventAssociations: len(a.Theme.Events), Receipts: 2}
	for _, tree := range a.ReasoningTrees {
		treeIDs[tree.IndustryChainEntityID] = researchreasoningtreeimport.ReasoningTreeID(themeID, tree.IndustryChainEntityID)
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
		ThemeKey: a.Theme.ThemeKey, ContractVersion: 2,
		ReasoningTreeIDsByIndustryChainEntityID: treeIDs, Counts: counts,
	}
}

func referenceQuery(a Aggregate) ReferenceQuery {
	q := ReferenceQuery{}
	for _, impact := range a.Theme.Impacts {
		q.ChainNodeIDs = append(q.ChainNodeIDs, impact.ChainNodeEntityID)
	}
	for _, event := range a.Theme.Events {
		q.EventIDs = append(q.EventIDs, event.EventID)
	}
	for _, tree := range a.ReasoningTrees {
		q.IndustryChainIDs = append(q.IndustryChainIDs, tree.IndustryChainEntityID)
		for _, event := range tree.Events {
			q.EventIDs = append(q.EventIDs, event.EventID)
		}
		for _, node := range tree.Nodes {
			q.ChainNodeIDs = append(q.ChainNodeIDs, node.ChainNodeEntityID)
			if node.IncomingIndustryChainGraphEdgeID != nil {
				q.GraphEdgeIDs = append(q.GraphEdgeIDs, *node.IncomingIndustryChainGraphEdgeID)
			}
			if l := node.IncomingLineage; l != nil {
				appendLineageQuery(&q, l.DirectImpactAssertionID, l.UpstreamVariableSignalID, l.UpstreamDirectImpactAssertionID, l.EvidenceID, l.EntityRelationID)
			}
			for _, signal := range node.Signals {
				l := signal.Lineage
				appendLineageQuery(&q, nil, choose(l.VariableSignalID, l.UpstreamVariableSignalID), l.UpstreamDirectImpactAssertionID, l.EvidenceID, l.EntityRelationID)
				if l.IndustryChainGraphEdgeID != nil {
					q.GraphEdgeIDs = append(q.GraphEdgeIDs, *l.IndustryChainGraphEdgeID)
				}
			}
		}
	}
	return q
}

func appendLineageQuery(q *ReferenceQuery, impact, signal, upstreamImpact, evidence, relation *string) {
	if impact != nil {
		q.ImpactIDs = append(q.ImpactIDs, *impact)
	}
	if signal != nil {
		q.SignalIDs = append(q.SignalIDs, *signal)
	}
	if upstreamImpact != nil {
		q.ImpactIDs = append(q.ImpactIDs, *upstreamImpact)
	}
	if evidence != nil {
		q.EvidenceIDs = append(q.EvidenceIDs, *evidence)
	}
	if relation != nil {
		q.EntityRelationIDs = append(q.EntityRelationIDs, *relation)
	}
}

func choose(first, second *string) *string {
	if first != nil {
		return first
	}
	return second
}

func validateReferences(a Aggregate, asOf time.Time, facts ReferenceFacts) error {
	windowStart, _ := time.Parse(time.RFC3339, a.DiscoveryWindowStart)
	windowEnd, _ := time.Parse(time.RFC3339, a.DiscoveryWindowEnd)
	for index, impact := range a.Theme.Impacts {
		temporal, ok := facts.ChainNodeIDs[impact.ChainNodeEntityID]
		if !ok || !temporalFactAvailableAt(temporal, asOf) {
			return invalidReference(fmt.Sprintf("theme.impacts[%d].chain_node_entity_id", index), impact.ChainNodeEntityID, "active approved Chain Node does not exist")
		}
	}
	for index, event := range a.Theme.Events {
		fact, ok := facts.Events[event.EventID]
		if !ok {
			return invalidReference(fmt.Sprintf("theme.events[%d].event_id", index), event.EventID, "confirmed verified Event does not exist")
		}
		if fact.KnowledgeAvailableAt.Before(windowStart) ||
			!fact.KnowledgeAvailableAt.Before(windowEnd) ||
			fact.KnowledgeAvailableAt.After(asOf) {
			return invalidReference(
				fmt.Sprintf("theme.events[%d].event_id", index), event.EventID,
				"Event was not knowable inside the declared discovery window by analysis_as_of",
			)
		}
	}
	for treeIndex, tree := range a.ReasoningTrees {
		treePath := fmt.Sprintf("reasoning_trees[%d]", treeIndex)
		chainTemporal, ok := facts.IndustryChainIDs[tree.IndustryChainEntityID]
		if !ok || !temporalFactAvailableAt(chainTemporal, asOf) {
			return invalidReference(treePath+".industry_chain_entity_id", tree.IndustryChainEntityID, "active approved Industry Chain does not exist")
		}
		treeEvents := make(map[string]struct{}, len(tree.Events))
		for _, event := range tree.Events {
			treeEvents[event.EventID] = struct{}{}
		}
		for nodeIndex, node := range tree.Nodes {
			nodePath := fmt.Sprintf("%s.nodes[%d]", treePath, nodeIndex)
			membership, ok := facts.Memberships[tree.IndustryChainEntityID][node.ChainNodeEntityID]
			if !ok || !temporalFactAvailableAt(membership, asOf) {
				return invalidReference(nodePath+".chain_node_entity_id", node.ChainNodeEntityID, "Node is not an active approved member of the Industry Chain")
			}
			if node.IncomingIndustryChainGraphEdgeID != nil {
				edge, ok := facts.GraphEdges[*node.IncomingIndustryChainGraphEdgeID]
				if !ok || !temporalFactAvailableAt(edge.TemporalFact, asOf) ||
					nodeIndex == 0 || edge.IndustryChainEntityID != tree.IndustryChainEntityID ||
					edge.FromChainNodeEntityID != tree.Nodes[nodeIndex-1].ChainNodeEntityID ||
					edge.ToChainNodeEntityID != node.ChainNodeEntityID {
					return invalidReference(nodePath+".incoming_industry_chain_graph_edge_id", *node.IncomingIndustryChainGraphEdgeID, "Graph Edge does not match the ordered path")
				}
			}
			if node.IncomingLineage != nil {
				if err := validateIncomingLineage(
					nodePath+".incoming_lineage",
					*node.IncomingLineage,
					tree.Nodes[nodeIndex-1].ChainNodeEntityID,
					node.ChainNodeEntityID,
					treeEvents,
					asOf,
					facts,
				); err != nil {
					return err
				}
			}
			for signalIndex, signal := range node.Signals {
				if err := validateSignalLineage(
					fmt.Sprintf("%s.signals[%d].lineage", nodePath, signalIndex),
					signal, node.ChainNodeEntityID, treeEvents, asOf, facts,
				); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateSignalLineage(path string, input Signal, nodeEntityID string, treeEvents map[string]struct{}, asOf time.Time, facts ReferenceFacts) error {
	l := input.Lineage
	if l.SourceKind == "formal_signal" {
		fact, ok := facts.Signals[*l.VariableSignalID]
		if !ok {
			return invalidReference(path+".variable_signal_id", *l.VariableSignalID, "accepted latest Signal does not exist")
		}
		if fact.SemanticSubmissionID != *l.SemanticSubmissionID {
			return invalidReference(path+".semantic_submission_id", *l.SemanticSubmissionID, "does not own the Signal")
		}
		if fact.SubjectEntityID != nodeEntityID {
			return invalidReference(path+".variable_signal_id", fact.ID, "Signal subject must equal the Node entity")
		}
		if fact.VariableKey != input.VariableSignalKey || fact.Direction != input.SignalDirection {
			return invalidReference(path+".variable_signal_id", fact.ID, "Signal display snapshot does not match the formal Signal")
		}
		if _, ok := treeEvents[fact.EventID]; !ok {
			return invalidReference(path+".variable_signal_id", fact.ID, "Signal source Event is not covered by Reason Tree events")
		}
		if err := validateEvidence(path, *l.EvidenceID, *l.EvidenceHash, fact.EventID, fact.EvidenceIDs, asOf, facts); err != nil {
			return err
		}
		if fact.AcceptedAt.After(asOf) {
			return invalidReference(path+".variable_signal_id", fact.ID, "Signal was not accepted by analysis_as_of")
		}
		return nil
	}
	return validateInference(
		path, l.UpstreamVariableSignalID, l.UpstreamDirectImpactAssertionID,
		l.EntityRelationID, l.IndustryChainGraphEdgeID, nodeEntityID,
		treeEvents, asOf, facts,
	)
}

func validateIncomingLineage(
	path string,
	l IncomingLineage,
	sourceEntityID, targetEntityID string,
	treeEvents map[string]struct{},
	asOf time.Time,
	facts ReferenceFacts,
) error {
	if l.SourceKind == "formal_direct_impact" {
		fact, ok := facts.Impacts[*l.DirectImpactAssertionID]
		if !ok {
			return invalidReference(path+".direct_impact_assertion_id", *l.DirectImpactAssertionID, "accepted latest Direct Impact does not exist")
		}
		if fact.SemanticSubmissionID != *l.SemanticSubmissionID {
			return invalidReference(path+".semantic_submission_id", *l.SemanticSubmissionID, "does not own the Direct Impact")
		}
		if fact.TargetEntityID != targetEntityID {
			return invalidReference(path+".direct_impact_assertion_id", fact.ID, "Direct Impact target must equal the downstream Node")
		}
		if fact.SourceEntityID != sourceEntityID {
			return invalidReference(path+".direct_impact_assertion_id", fact.ID, "Direct Impact source Signal subject must equal the previous Node")
		}
		if fact.AffectedVariableKey != *l.AffectedVariableKey || fact.AffectedDirection != *l.AffectedDirection {
			return invalidReference(path+".direct_impact_assertion_id", fact.ID, "affected-variable snapshot does not match the formal Direct Impact")
		}
		if _, ok := treeEvents[fact.SourceEventID]; !ok {
			return invalidReference(path+".direct_impact_assertion_id", fact.ID, "Direct Impact source Event is not covered by Reason Tree events")
		}
		if err := validateEvidence(path, *l.EvidenceID, *l.EvidenceHash, fact.SourceEventID, fact.EvidenceIDs, asOf, facts); err != nil {
			return err
		}
		if fact.AcceptedAt.After(asOf) {
			return invalidReference(path+".direct_impact_assertion_id", fact.ID, "Direct Impact was not accepted by analysis_as_of")
		}
		return nil
	}
	return validateInference(
		path, l.UpstreamVariableSignalID, l.UpstreamDirectImpactAssertionID,
		l.EntityRelationID, nil, targetEntityID, treeEvents, asOf, facts,
	)
}

func validateInference(
	path string,
	signalID, impactID, entityRelationID, graphEdgeID *string,
	targetEntityID string,
	treeEvents map[string]struct{},
	asOf time.Time,
	facts ReferenceFacts,
) error {
	upstreamEntityIDs := make(map[string]struct{}, 2)
	if signalID != nil {
		fact, ok := facts.Signals[*signalID]
		if !ok || fact.AcceptedAt.After(asOf) {
			return invalidReference(path+".upstream_variable_signal_id", *signalID, "accepted latest upstream Signal does not exist at analysis_as_of")
		}
		if _, ok := treeEvents[fact.EventID]; !ok {
			return invalidReference(path+".upstream_variable_signal_id", *signalID, "upstream Signal source Event is not covered by Reason Tree events")
		}
		upstreamEntityIDs[fact.SubjectEntityID] = struct{}{}
	}
	if impactID != nil {
		fact, ok := facts.Impacts[*impactID]
		if !ok || fact.AcceptedAt.After(asOf) {
			return invalidReference(path+".upstream_direct_impact_assertion_id", *impactID, "accepted latest upstream Direct Impact does not exist at analysis_as_of")
		}
		if _, ok := treeEvents[fact.SourceEventID]; !ok {
			return invalidReference(path+".upstream_direct_impact_assertion_id", *impactID, "upstream Direct Impact source Event is not covered by Reason Tree events")
		}
		upstreamEntityIDs[fact.SourceEntityID] = struct{}{}
		upstreamEntityIDs[fact.TargetEntityID] = struct{}{}
	}
	if entityRelationID != nil {
		relation, ok := facts.EntityRelations[*entityRelationID]
		_, connectsUpstream := upstreamEntityIDs[relation.FromEntityID]
		if !ok || !temporalFactAvailableAt(relation.TemporalFact, asOf) ||
			!connectsUpstream || relation.ToEntityID != targetEntityID {
			return invalidReference(path+".entity_relation_id", *entityRelationID, "active Entity Relation does not connect the upstream fact to the inferred target")
		}
	}
	if graphEdgeID != nil {
		edge, ok := facts.GraphEdges[*graphEdgeID]
		_, connectsUpstream := upstreamEntityIDs[edge.FromChainNodeEntityID]
		if !ok || !temporalFactAvailableAt(edge.TemporalFact, asOf) ||
			!connectsUpstream || edge.ToChainNodeEntityID != targetEntityID {
			return invalidReference(path+".industry_chain_graph_edge_id", *graphEdgeID, "active approved Industry Chain Graph Edge does not connect the upstream fact to the inferred target")
		}
	}
	return nil
}

func temporalFactAvailableAt(fact TemporalFact, asOf time.Time) bool {
	return !fact.CreatedAt.IsZero() &&
		!fact.UpdatedAt.IsZero() &&
		!fact.CreatedAt.After(asOf) &&
		!fact.UpdatedAt.After(asOf)
}

func validateEvidence(path, evidenceID, hash, eventID string, allowed map[string]struct{}, asOf time.Time, facts ReferenceFacts) error {
	if _, ok := allowed[evidenceID]; !ok {
		return invalidReference(path+".evidence_id", evidenceID, "Evidence is not attached to the formal fact")
	}
	evidence, ok := facts.Evidences[evidenceID]
	if !ok || evidence.EventID != eventID || evidence.Hash != hash {
		return invalidReference(path+".evidence_id", evidenceID, "Evidence ID/hash/Event lineage does not match")
	}
	if evidence.KnowledgeAvailableAt.After(asOf) {
		return invalidReference(path+".evidence_id", evidenceID, "Evidence was not knowable by analysis_as_of")
	}
	return nil
}

func resultFromReceipt(r Receipt, replayed bool) Result {
	return Result{
		ReceiptID: r.ID, AnalysisBatchID: r.AnalysisBatchID, PayloadHash: r.PayloadHash,
		ThemeID: r.ThemeID, ReasoningTreeIDsByIndustryChainEntityID: cloneMap(r.ReasoningTreeIDsByIndustryChainEntityID),
		Counts: r.Counts, PublishedAt: r.PublishedAt, ImportedAt: r.ImportedAt, Replayed: replayed,
	}
}

func cloneMap(input map[string]string) map[string]string {
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
