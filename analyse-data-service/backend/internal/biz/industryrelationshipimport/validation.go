package industryrelationshipimport

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/identity"
)

func (p Package) Validate() error {
	if len(p.IndustryChains) != 708 {
		return fmt.Errorf("industry chain coverage is %d/708", len(p.IndustryChains))
	}
	if len(p.ConceptDispositions) != 194 {
		return fmt.Errorf("Concept disposition coverage is %d/194", len(p.ConceptDispositions))
	}
	if len(p.NodeDispositions) != 588 {
		return fmt.Errorf("Chain Node disposition coverage is %d/588", len(p.NodeDispositions))
	}
	if len(p.UnmappedCandidates) != 0 {
		return fmt.Errorf("unmapped relation candidate count is %d, want zero", len(p.UnmappedCandidates))
	}
	if p.ValidationReport.SchemaVersion != "industry_relationship_validation_report_v1" ||
		p.ValidationReport.Status != "passed" ||
		p.ValidationReport.ApprovalBasis != ApprovalBasis {
		return fmt.Errorf("relationship validation report is not passed under %s", ApprovalBasis)
	}
	if p.ValidationReport.VerifiedAt.IsZero() {
		return fmt.Errorf("relationship validation report verified_at is required")
	}
	if err := validateHardGates(p.ValidationReport); err != nil {
		return err
	}
	if err := validateCounts(p); err != nil {
		return err
	}

	chains, err := validateChains(p.IndustryChains)
	if err != nil {
		return err
	}
	additions, err := validateAdditions(p.ChainNodeAdditions)
	if err != nil {
		return err
	}
	if err := validateMappings(p.IndustryMappings, "mapped_to_industry", chains); err != nil {
		return err
	}
	if err := validateMappings(p.ConceptMappings, "mapped_to_concept", chains); err != nil {
		return err
	}
	memberships, err := validateMemberships(p.Memberships, chains, additions)
	if err != nil {
		return err
	}
	if err := validateGraphEdges(p.GraphEdges, chains, memberships); err != nil {
		return err
	}
	if err := validateGlobalRelations(p.GlobalRelations); err != nil {
		return err
	}
	if err := validateDispositions(p, additions); err != nil {
		return err
	}
	mappedChains := make(map[string]struct{}, len(chains))
	for _, relation := range p.IndustryMappings {
		mappedChains[relation.FromKey] = struct{}{}
	}
	for _, relation := range p.ConceptMappings {
		mappedChains[relation.FromKey] = struct{}{}
	}
	if len(mappedChains) != len(chains) {
		return fmt.Errorf("%d Industry Chains have no Industry or Concept mapping", len(chains)-len(mappedChains))
	}
	return nil
}

func validateHardGates(report ValidationReport) error {
	if strings.TrimSpace(report.ClosedWorldNote) == "" {
		return fmt.Errorf("relationship validation report closed_world_note is required")
	}
	requiredStrings := map[string]string{
		"chain_topology_decision_coverage": "708/708",
		"chain_mapping_coverage":           "708/708",
		"topology_semantic_audit_status":   "passed",
		"topology_global_closure_status":   "passed",
	}
	for gate, expected := range requiredStrings {
		if report.HardGates[gate] != expected {
			return fmt.Errorf("relationship hard gate %s=%v, want %s", gate, report.HardGates[gate], expected)
		}
	}
	for _, gate := range []string{
		"chain_topologies_weakly_connected",
		"chain_topologies_acyclic",
	} {
		if report.HardGates[gate] != true {
			return fmt.Errorf("relationship hard gate %s did not pass", gate)
		}
	}
	for _, gate := range []string{
		"topology_semantic_error_count",
		"topology_semantic_unreviewed_warning_count",
		"orphan_membership_count",
		"frozen_chain_node_without_formal_relation_count",
		"orphan_industry_count",
		"projected_concept_without_mapping_count",
		"neo4j_projected_orphan_count",
		"unresolved_endpoint_count",
		"unresolved_evidence_count",
		"duplicate_semantic_relation_count",
		"candidate_relation_count",
		"unmapped_relation_candidate_count",
	} {
		if !zeroNumber(report.HardGates[gate]) {
			return fmt.Errorf("relationship hard gate %s=%v, want zero", gate, report.HardGates[gate])
		}
	}
	return nil
}

func zeroNumber(value any) bool {
	switch typed := value.(type) {
	case int:
		return typed == 0
	case int64:
		return typed == 0
	case float64:
		return typed == 0
	case json.Number:
		return typed.String() == "0"
	default:
		return false
	}
}

func validateCounts(p Package) error {
	actual := countsMap(p.Counts())
	for key, count := range actual {
		if p.Manifest.PackageCounts[key] != count {
			return fmt.Errorf("manifest package count %s=%d, actual %d", key, p.Manifest.PackageCounts[key], count)
		}
		if p.ValidationReport.PackageCounts[key] != count {
			return fmt.Errorf("validation report package count %s=%d, actual %d", key, p.ValidationReport.PackageCounts[key], count)
		}
	}
	if p.Manifest.PackageCounts["industry_chain"] != 708 {
		return fmt.Errorf("manifest Industry Chain count must be 708")
	}
	return nil
}

func validateChains(items []IndustryChain) (map[string]IndustryChain, error) {
	result := make(map[string]IndustryChain, len(items))
	ids := make(map[string]struct{}, len(items))
	for index, item := range items {
		path := fmt.Sprintf("industry_chains[%d]", index)
		if item.EntityType != "industry_chain" || item.LayerCode != "industry_chain" {
			return nil, fmt.Errorf("%s has invalid entity_type/layer_code", path)
		}
		if item.EntityID != identity.NormalizeUUID("entity", item.EntityKey) {
			return nil, fmt.Errorf("%s has non-deterministic entity_id", path)
		}
		if !strings.HasPrefix(item.EntityKey, "industry_chain:") {
			return nil, fmt.Errorf("%s has invalid entity_key", path)
		}
		if item.Name != item.CanonicalName || !allNonblank(
			item.Name, item.Scope, item.TargetOutput, item.EndUse, item.Geography,
			item.AsOfDate, item.ReviewNote,
		) || !allSliceNonblank(item.ObservableVariables) {
			return nil, fmt.Errorf("%s has incomplete identity/definition", path)
		}
		if item.Status != "active" || item.ReviewStatus != "approved" ||
			item.RelationshipApprovalBasis != ApprovalBasis {
			return nil, fmt.Errorf("%s is not approved/active", path)
		}
		if _, exists := result[item.EntityKey]; exists {
			return nil, fmt.Errorf("%s duplicates entity_key %s", path, item.EntityKey)
		}
		if _, exists := ids[item.EntityID]; exists {
			return nil, fmt.Errorf("%s duplicates entity_id %s", path, item.EntityID)
		}
		result[item.EntityKey] = item
		ids[item.EntityID] = struct{}{}
	}
	return result, nil
}

func validateAdditions(items []ChainNodeAddition) (map[string]ChainNodeAddition, error) {
	requiredGates := []string{
		"stable_object", "clear_boundary", "economic_independence", "observable_state",
		"chain_connectivity", "distinguishable_granularity",
	}
	result := make(map[string]ChainNodeAddition, len(items))
	names := make(map[string]string, len(items))
	for index, item := range items {
		path := fmt.Sprintf("chain_node_additions[%d]", index)
		if item.EntityType != "chain_node" || item.LayerCode != "industry_chain" ||
			!strings.HasPrefix(item.EntityKey, "chain_node:v2_") {
			return nil, fmt.Errorf("%s has invalid Chain Node identity", path)
		}
		if item.EntityID != identity.NormalizeUUID("entity", item.EntityKey) {
			return nil, fmt.Errorf("%s has non-deterministic entity_id", path)
		}
		if item.Name != item.CanonicalName ||
			!allNonblank(item.Name, item.Definition, item.BoundaryNote) {
			return nil, fmt.Errorf("%s has incomplete definition/boundary", path)
		}
		if item.Status != "active" || item.ReviewStatus != "approved" ||
			item.RelationshipApprovalBasis != ApprovalBasis || item.VerifiedAt.IsZero() {
			return nil, fmt.Errorf("%s is not approved/active with verified_at", path)
		}
		if !allSliceNonblank(item.EvidenceIDs) {
			return nil, fmt.Errorf("%s has no evidence", path)
		}
		for _, gate := range requiredGates {
			if item.GateResults[gate] != "pass" {
				return nil, fmt.Errorf("%s gate %s did not pass", path, gate)
			}
		}
		if _, exists := result[item.EntityKey]; exists {
			return nil, fmt.Errorf("%s duplicates entity_key %s", path, item.EntityKey)
		}
		foldedName := strings.ToLower(strings.TrimSpace(item.CanonicalName))
		if previous, exists := names[foldedName]; exists && previous != item.EntityKey {
			return nil, fmt.Errorf("%s duplicates canonical name under keys %s and %s", path, previous, item.EntityKey)
		}
		result[item.EntityKey] = item
		names[foldedName] = item.EntityKey
	}
	return result, nil
}

func validateMappings(items []EntityMapping, relationType string, chains map[string]IndustryChain) error {
	seenKeys := make(map[string]struct{}, len(items))
	seenTuples := make(map[string]struct{}, len(items))
	for index, item := range items {
		path := fmt.Sprintf("%s[%d]", relationType, index)
		chain, exists := chains[item.FromKey]
		if !exists || chain.EntityID != item.FromEntityID {
			return fmt.Errorf("%s has unresolved Industry Chain endpoint", path)
		}
		if item.RelationType != relationType ||
			item.RelationKey != strings.Join([]string{item.FromKey, relationType, item.ToKey}, "|") {
			return fmt.Errorf("%s has invalid type/key", path)
		}
		if item.RelationID != identity.NormalizeUUID("entity_relationship", item.RelationKey) {
			return fmt.Errorf("%s has non-deterministic relation_id", path)
		}
		if !allNonblank(item.ToKey, item.ToEntityID, item.MappingReason, item.EvidenceNote) ||
			!allSliceNonblank(item.EvidenceIDs) {
			return fmt.Errorf("%s has incomplete target/reason/evidence", path)
		}
		if item.ReviewStatus != "approved" || item.Status != "active" || item.VerifiedAt.IsZero() {
			return fmt.Errorf("%s is not approved/active with verified_at", path)
		}
		tuple := strings.Join([]string{item.FromEntityID, relationType, item.ToEntityID}, "|")
		if _, exists := seenKeys[item.RelationKey]; exists {
			return fmt.Errorf("%s duplicates relation_key", path)
		}
		if _, exists := seenTuples[tuple]; exists {
			return fmt.Errorf("%s duplicates semantic tuple", path)
		}
		seenKeys[item.RelationKey] = struct{}{}
		seenTuples[tuple] = struct{}{}
	}
	return nil
}

type membershipIdentity struct {
	ChainID, NodeID, ChainKey, NodeKey string
}

func validateMemberships(
	items []Membership,
	chains map[string]IndustryChain,
	additions map[string]ChainNodeAddition,
) (map[string]map[string]membershipIdentity, error) {
	result := make(map[string]map[string]membershipIdentity, len(chains))
	for index, item := range items {
		path := fmt.Sprintf("industry_chain_node_memberships[%d]", index)
		chain, exists := chains[item.ChainKey]
		if !exists || chain.EntityID != item.IndustryChainEntityID {
			return nil, fmt.Errorf("%s has unresolved Industry Chain endpoint", path)
		}
		if item.RelationKey != strings.Join([]string{item.ChainKey, "has_node", item.NodeKey}, "|") ||
			item.ChainNodeEntityID != identity.NormalizeUUID("entity", item.NodeKey) {
			return nil, fmt.Errorf("%s has invalid deterministic identity", path)
		}
		if addition, isAddition := additions[item.NodeKey]; isAddition &&
			addition.EntityID != item.ChainNodeEntityID {
			return nil, fmt.Errorf("%s does not match its new Chain Node", path)
		}
		if item.ContextualStage != "upstream" &&
			item.ContextualStage != "midstream" &&
			item.ContextualStage != "downstream" {
			return nil, fmt.Errorf("%s has invalid contextual_stage", path)
		}
		if item.Position <= 0 || !allNonblank(
			item.NodeKey, item.InclusionReason, item.SourceName, item.SourceURL,
		) || !allSliceNonblank(item.EvidenceIDs) {
			return nil, fmt.Errorf("%s has incomplete membership/provenance", path)
		}
		if !validSourceLocator(item.SourceURL) || item.VerifiedAt.IsZero() ||
			item.ReviewStatus != "approved" || item.Status != "active" {
			return nil, fmt.Errorf("%s is not approved/active with valid provenance", path)
		}
		byNode := result[item.ChainKey]
		if byNode == nil {
			byNode = make(map[string]membershipIdentity)
			result[item.ChainKey] = byNode
		}
		if _, exists := byNode[item.NodeKey]; exists {
			return nil, fmt.Errorf("%s duplicates a Chain membership", path)
		}
		byNode[item.NodeKey] = membershipIdentity{
			ChainID: item.IndustryChainEntityID, NodeID: item.ChainNodeEntityID,
			ChainKey: item.ChainKey, NodeKey: item.NodeKey,
		}
	}
	for chainKey := range chains {
		if len(result[chainKey]) < 2 {
			return nil, fmt.Errorf("%s has %d memberships, want at least two", chainKey, len(result[chainKey]))
		}
	}
	return result, nil
}

func validateGraphEdges(
	items []GraphEdge,
	chains map[string]IndustryChain,
	memberships map[string]map[string]membershipIdentity,
) error {
	edgesByChain := make(map[string][]GraphEdge, len(chains))
	seenIDs := make(map[string]struct{}, len(items))
	seenTuples := make(map[string]struct{}, len(items))
	for index, item := range items {
		path := fmt.Sprintf("industry_chain_graph_edges[%d]", index)
		chain, exists := chains[item.ChainKey]
		if !exists || chain.EntityID != item.IndustryChainEntityID {
			return fmt.Errorf("%s has unresolved Industry Chain endpoint", path)
		}
		from, fromExists := memberships[item.ChainKey][item.FromNodeKey]
		to, toExists := memberships[item.ChainKey][item.ToNodeKey]
		if !fromExists || !toExists ||
			from.NodeID != item.FromChainNodeEntityID ||
			to.NodeID != item.ToChainNodeEntityID {
			return fmt.Errorf("%s endpoints are not same-chain memberships", path)
		}
		if item.FromNodeKey == item.ToNodeKey ||
			(item.RelationType != "input_to" &&
				item.RelationType != "is_component_of" &&
				item.RelationType != "depends_on") {
			return fmt.Errorf("%s has invalid endpoint/type", path)
		}
		semanticKey := strings.Join(
			[]string{item.ChainKey, item.FromNodeKey, item.RelationType, item.ToNodeKey}, "|",
		)
		if item.RelationKey != semanticKey ||
			item.ID != identity.NormalizeUUID("industry_chain_graph_edge", semanticKey) {
			return fmt.Errorf("%s has non-deterministic relation identity", path)
		}
		if !allNonblank(item.Mechanism, item.SourceName, item.SourceURL) ||
			!allSliceNonblank(item.EvidenceIDs) || !validSourceLocator(item.SourceURL) {
			return fmt.Errorf("%s has incomplete mechanism/provenance", path)
		}
		if item.SegmentKind == "direct_candidate" {
			if item.OmittedStepNote != nil {
				return fmt.Errorf("%s direct edge has omitted_step_note", path)
			}
		} else if item.SegmentKind != "compressed_candidate" ||
			item.OmittedStepNote == nil || strings.TrimSpace(*item.OmittedStepNote) == "" {
			return fmt.Errorf("%s has invalid segment_kind/omitted_step_note", path)
		}
		if item.VerifiedAt.IsZero() || item.ReviewStatus != "approved" || item.Status != "active" {
			return fmt.Errorf("%s is not approved/active with verified_at", path)
		}
		if _, exists := seenIDs[item.ID]; exists {
			return fmt.Errorf("%s duplicates ID %s", path, item.ID)
		}
		if _, exists := seenTuples[semanticKey]; exists {
			return fmt.Errorf("%s duplicates semantic tuple", path)
		}
		seenIDs[item.ID] = struct{}{}
		seenTuples[semanticKey] = struct{}{}
		edgesByChain[item.ChainKey] = append(edgesByChain[item.ChainKey], item)
	}
	for chainKey := range chains {
		if len(edgesByChain[chainKey]) == 0 {
			return fmt.Errorf("%s has no graph edges", chainKey)
		}
		if err := validateChainTopology(chainKey, memberships[chainKey], edgesByChain[chainKey]); err != nil {
			return err
		}
	}
	return nil
}

func validateChainTopology(
	chainKey string,
	members map[string]membershipIdentity,
	edges []GraphEdge,
) error {
	undirected := make(map[string][]string, len(members))
	directed := make(map[string][]string, len(members))
	incident := make(map[string]int, len(members))
	for key := range members {
		undirected[key] = nil
		directed[key] = nil
	}
	for _, edge := range edges {
		undirected[edge.FromNodeKey] = append(undirected[edge.FromNodeKey], edge.ToNodeKey)
		undirected[edge.ToNodeKey] = append(undirected[edge.ToNodeKey], edge.FromNodeKey)
		directed[edge.FromNodeKey] = append(directed[edge.FromNodeKey], edge.ToNodeKey)
		incident[edge.FromNodeKey]++
		incident[edge.ToNodeKey]++
	}
	keys := make([]string, 0, len(members))
	for key := range members {
		keys = append(keys, key)
		if incident[key] == 0 {
			return fmt.Errorf("%s has orphan membership %s", chainKey, key)
		}
	}
	sort.Strings(keys)
	seen := make(map[string]struct{}, len(keys))
	stack := []string{keys[0]}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, exists := seen[current]; exists {
			continue
		}
		seen[current] = struct{}{}
		stack = append(stack, undirected[current]...)
	}
	if len(seen) != len(members) {
		return fmt.Errorf("%s graph is not weakly connected", chainKey)
	}
	state := make(map[string]uint8, len(keys))
	var visit func(string) bool
	visit = func(node string) bool {
		if state[node] == 1 {
			return true
		}
		if state[node] == 2 {
			return false
		}
		state[node] = 1
		for _, next := range directed[node] {
			if visit(next) {
				return true
			}
		}
		state[node] = 2
		return false
	}
	for _, key := range keys {
		if visit(key) {
			return fmt.Errorf("%s graph contains a cycle", chainKey)
		}
	}
	return nil
}

func validateGlobalRelations(items []GlobalChainNodeRelation) error {
	seen := make(map[string]struct{}, len(items))
	hierarchy := make(map[string][]string)
	for index, item := range items {
		path := fmt.Sprintf("global_chain_node_relations[%d]", index)
		if item.RelationType != "is_subcategory_of" ||
			item.FromChainNodeEntityID == item.ToChainNodeEntityID {
			return fmt.Errorf("%s has invalid global hierarchy semantics", path)
		}
		if item.FromChainNodeEntityID != identity.NormalizeUUID("entity", item.FromNodeKey) ||
			item.ToChainNodeEntityID != identity.NormalizeUUID("entity", item.ToNodeKey) {
			return fmt.Errorf("%s has non-deterministic endpoint UUID", path)
		}
		tuple := strings.Join(
			[]string{item.FromChainNodeEntityID, item.RelationType, item.ToChainNodeEntityID}, "|",
		)
		if item.ID != identity.NormalizeUUID("chain_node_relation", tuple) {
			return fmt.Errorf("%s has non-deterministic relation ID", path)
		}
		if item.RelationKey != "chain_node_relation:"+item.ID {
			return fmt.Errorf("%s has non-deterministic relation_key", path)
		}
		if !allNonblank(
			item.FromNodeKey, item.FromName, item.ToNodeKey, item.ToName,
			item.Mechanism, item.EvidenceNote, item.Provenance, item.Confidence,
		) || item.VerifiedAt.IsZero() {
			return fmt.Errorf("%s has incomplete mechanism/evidence", path)
		}
		if item.ReviewStatus != "approved" || item.Status != "active" {
			return fmt.Errorf("%s is not approved/active", path)
		}
		if _, exists := seen[tuple]; exists {
			return fmt.Errorf("%s duplicates semantic tuple", path)
		}
		seen[tuple] = struct{}{}
		hierarchy[item.FromNodeKey] = append(hierarchy[item.FromNodeKey], item.ToNodeKey)
		if _, exists := hierarchy[item.ToNodeKey]; !exists {
			hierarchy[item.ToNodeKey] = nil
		}
	}
	state := make(map[string]uint8, len(hierarchy))
	var visit func(string) bool
	visit = func(nodeKey string) bool {
		if state[nodeKey] == 1 {
			return true
		}
		if state[nodeKey] == 2 {
			return false
		}
		state[nodeKey] = 1
		for _, parentKey := range hierarchy[nodeKey] {
			if visit(parentKey) {
				return true
			}
		}
		state[nodeKey] = 2
		return false
	}
	for nodeKey := range hierarchy {
		if visit(nodeKey) {
			return fmt.Errorf("global Chain Node hierarchy contains a cycle at %s", nodeKey)
		}
	}
	return nil
}

type conceptDisposition struct {
	ConceptKey   string `json:"concept_key"`
	Disposition  string `json:"disposition"`
	ReviewStatus string `json:"review_status"`
	Status       string `json:"status"`
}

type nodeDisposition struct {
	NodeKey      string `json:"node_key"`
	Disposition  string `json:"disposition"`
	ReviewStatus string `json:"review_status"`
	Status       string `json:"status"`
}

func validateDispositions(p Package, additions map[string]ChainNodeAddition) error {
	mappedConcepts := make(map[string]struct{}, len(p.ConceptMappings))
	for _, relation := range p.ConceptMappings {
		mappedConcepts[relation.ToKey] = struct{}{}
	}
	conceptKeys := make(map[string]struct{}, len(p.ConceptDispositions))
	for index, raw := range p.ConceptDispositions {
		var disposition conceptDisposition
		if err := json.Unmarshal(raw, &disposition); err != nil {
			return fmt.Errorf("concept_dispositions[%d] is invalid: %w", index, err)
		}
		if !allNonblank(disposition.ConceptKey, disposition.Disposition) ||
			disposition.ReviewStatus != "approved" || disposition.Status != "active" {
			return fmt.Errorf("concept_dispositions[%d] is not approved/active", index)
		}
		if _, exists := conceptKeys[disposition.ConceptKey]; exists {
			return fmt.Errorf("concept_dispositions[%d] duplicates %s", index, disposition.ConceptKey)
		}
		conceptKeys[disposition.ConceptKey] = struct{}{}
		_, mapped := mappedConcepts[disposition.ConceptKey]
		switch disposition.Disposition {
		case "mapped":
			if !mapped {
				return fmt.Errorf("mapped Concept %s has no formal relation", disposition.ConceptKey)
			}
		case "needs_chain_expansion":
			if mapped {
				return fmt.Errorf("Concept %s is both mapped and excluded for expansion", disposition.ConceptKey)
			}
		default:
			return fmt.Errorf(
				"concept_dispositions[%d] has unsupported disposition %s",
				index, disposition.Disposition,
			)
		}
	}
	for conceptKey := range mappedConcepts {
		if _, exists := conceptKeys[conceptKey]; !exists {
			return fmt.Errorf("mapped Concept %s has no disposition", conceptKey)
		}
	}

	formalNodeKeys := make(map[string]struct{})
	membershipNodeKeys := make(map[string]struct{})
	for _, membership := range p.Memberships {
		formalNodeKeys[membership.NodeKey] = struct{}{}
		membershipNodeKeys[membership.NodeKey] = struct{}{}
	}
	for _, relation := range p.GlobalRelations {
		formalNodeKeys[relation.FromNodeKey] = struct{}{}
		formalNodeKeys[relation.ToNodeKey] = struct{}{}
	}
	nodeKeys := make(map[string]struct{}, len(p.NodeDispositions))
	allowedNodeDispositions := map[string]struct{}{
		"connected_by_discovery": {},
		"hierarchy_child":        {},
		"hierarchy_parent":       {},
		"membership_required":    {},
	}
	for index, raw := range p.NodeDispositions {
		var disposition nodeDisposition
		if err := json.Unmarshal(raw, &disposition); err != nil {
			return fmt.Errorf("node_dispositions[%d] is invalid: %w", index, err)
		}
		if !allNonblank(disposition.NodeKey, disposition.Disposition) ||
			disposition.ReviewStatus != "approved" || disposition.Status != "active" {
			return fmt.Errorf("node_dispositions[%d] is not approved/active", index)
		}
		if _, allowed := allowedNodeDispositions[disposition.Disposition]; !allowed {
			return fmt.Errorf(
				"node_dispositions[%d] has unsupported disposition %s",
				index, disposition.Disposition,
			)
		}
		if _, exists := nodeKeys[disposition.NodeKey]; exists {
			return fmt.Errorf("node_dispositions[%d] duplicates %s", index, disposition.NodeKey)
		}
		if _, isAddition := additions[disposition.NodeKey]; isAddition {
			return fmt.Errorf("node disposition %s collides with a package addition", disposition.NodeKey)
		}
		if _, connected := formalNodeKeys[disposition.NodeKey]; !connected {
			return fmt.Errorf("frozen Chain Node %s has no membership/hierarchy", disposition.NodeKey)
		}
		if disposition.Disposition == "membership_required" {
			if _, admitted := membershipNodeKeys[disposition.NodeKey]; !admitted {
				return fmt.Errorf(
					"membership_required frozen Chain Node %s has no M2 membership",
					disposition.NodeKey,
				)
			}
		}
		nodeKeys[disposition.NodeKey] = struct{}{}
	}
	for nodeKey := range formalNodeKeys {
		if _, isAddition := additions[nodeKey]; isAddition {
			continue
		}
		if _, exists := nodeKeys[nodeKey]; !exists {
			return fmt.Errorf("formal Chain Node endpoint %s has no frozen disposition or addition", nodeKey)
		}
	}
	for nodeKey := range additions {
		if _, connected := formalNodeKeys[nodeKey]; !connected {
			return fmt.Errorf("new Chain Node %s has no membership/hierarchy", nodeKey)
		}
	}
	return nil
}

func validSourceLocator(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil &&
		(parsed.Scheme == "http" || parsed.Scheme == "https" || parsed.Scheme == "artifact") &&
		parsed.Host != ""
}

func allNonblank(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return false
		}
	}
	return true
}

func allSliceNonblank(values []string) bool {
	if len(values) == 0 {
		return false
	}
	return allNonblank(values...)
}

func countsMap(counts Counts) map[string]int {
	return map[string]int{
		"industry_chain":                    counts.IndustryChains,
		"chain_node_additions":              counts.ChainNodeAdditions,
		"industry_chain_industry_relations": counts.IndustryMappings,
		"industry_chain_concept_relations":  counts.ConceptMappings,
		"industry_chain_node_memberships":   counts.Memberships,
		"industry_chain_graph_edges":        counts.GraphEdges,
		"global_chain_node_relations":       counts.GlobalRelations,
		"relationship_evidence":             counts.Evidence,
		"concept_dispositions":              counts.ConceptDispositions,
		"node_dispositions":                 counts.NodeDispositions,
		"unmapped_relation_candidates":      counts.UnmappedCandidates,
	}
}

func countsEqual(left, right Counts) bool {
	return reflect.DeepEqual(left, right)
}
