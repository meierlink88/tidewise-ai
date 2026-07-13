package seed

import (
	"fmt"
	"sort"
	"strings"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type FirstBatchNodeDraft struct {
	EntityKey     string   `json:"entity_key"`
	CanonicalName string   `json:"canonical_name"`
	OriginalNames []string `json:"original_names"`
	Definition    string   `json:"definition"`
	BoundaryNote  string   `json:"boundary_note,omitempty"`
	WideBoundary  bool     `json:"wide_boundary"`
}

type FirstBatchMappingDraft struct {
	CanonicalName      string `json:"canonical_name"`
	SourceSystem       string `json:"source_system"`
	SourceTaxonomyType string `json:"source_taxonomy_type,omitempty"`
	ExternalCode       string `json:"external_code"`
	ExternalName       string `json:"external_name"`
	TaxonomyResolved   bool   `json:"taxonomy_resolved"`
}

type FirstBatchDraft struct {
	Nodes    []FirstBatchNodeDraft    `json:"nodes"`
	Mappings []FirstBatchMappingDraft `json:"mappings"`
}

type FirstBatchExpectations struct {
	Nodes             int `json:"nodes"`
	OriginalNames     int `json:"original_names"`
	WideBoundaryNodes int `json:"wide_boundary_nodes"`
	Mappings          int `json:"mappings"`
	EastmoneyMappings int `json:"eastmoney_mappings"`
	THSMappings       int `json:"ths_mappings"`
	DualSourceNodes   int `json:"dual_source_nodes"`
}

func ApprovedFirstBatchExpectations() FirstBatchExpectations {
	return FirstBatchExpectations{
		Nodes:             842,
		OriginalNames:     950,
		WideBoundaryNodes: 79,
		Mappings:          1156,
		EastmoneyMappings: 811,
		THSMappings:       345,
		DualSourceNodes:   241,
	}
}

type FirstBatchIdentity struct {
	EntityID      string            `json:"entity_id"`
	EntityKey     string            `json:"entity_key"`
	EntityType    domain.EntityType `json:"entity_type"`
	CanonicalName string            `json:"canonical_name"`
	Aliases       []string          `json:"aliases"`
	Definition    string            `json:"definition"`
	BoundaryNote  string            `json:"boundary_note,omitempty"`
	Status        domain.Status     `json:"status"`
	Action        string            `json:"action"`
}

type FirstBatchIdentitySnapshot struct {
	ByEntityID          map[string]FirstBatchIdentity
	ByEntityKey         map[string]FirstBatchIdentity
	ByCanonicalName     map[string]FirstBatchIdentity
	ExternalIdentifiers FirstBatchExternalIdentifierSnapshot
}

type FirstBatchExternalIdentifier struct {
	ID                 string        `json:"id"`
	EntityID           string        `json:"entity_id"`
	SourceSystem       string        `json:"source_system"`
	SourceTaxonomyType string        `json:"source_taxonomy_type"`
	ExternalCode       string        `json:"external_code"`
	ExternalName       string        `json:"external_name"`
	Status             domain.Status `json:"status"`
}

type FirstBatchExternalIdentifierSnapshot struct {
	ByIdentity map[string]FirstBatchExternalIdentifier
	ByID       map[string]FirstBatchExternalIdentifier
}

type FirstBatchMappingReport struct {
	ID                 string `json:"id"`
	EntityID           string `json:"entity_id"`
	CanonicalName      string `json:"canonical_name"`
	SourceSystem       string `json:"source_system"`
	SourceTaxonomyType string `json:"source_taxonomy_type"`
	ExternalCode       string `json:"external_code"`
	ExternalName       string `json:"external_name"`
	Action             string `json:"action"`
}

type FirstBatchDryRunReport struct {
	Ready                 bool                      `json:"ready"`
	NodeCount             int                       `json:"node_count"`
	OriginalNameCount     int                       `json:"original_name_count"`
	WideBoundaryNodeCount int                       `json:"wide_boundary_node_count"`
	MappingCount          int                       `json:"mapping_count"`
	ProviderCounts        map[string]int            `json:"provider_counts"`
	DualSourceNodeCount   int                       `json:"dual_source_node_count"`
	Nodes                 []FirstBatchIdentity      `json:"nodes"`
	Mappings              []FirstBatchMappingReport `json:"mappings"`
	Blockers              []string                  `json:"blockers"`
	Conflicts             []string                  `json:"conflicts"`
}

func BuildFirstBatchDryRun(draft FirstBatchDraft, snapshot FirstBatchIdentitySnapshot, expectations FirstBatchExpectations) FirstBatchDryRunReport {
	report := FirstBatchDryRunReport{ProviderCounts: map[string]int{ExternalSourceEastmoney: 0, ExternalSourceTHS: 0}}
	byCanonical := make(map[string]FirstBatchIdentity, len(draft.Nodes))
	seenKeys := make(map[string]struct{}, len(draft.Nodes))
	seenIDs := make(map[string]struct{}, len(draft.Nodes))
	originalOwners := map[string]string{}

	for index, node := range draft.Nodes {
		identity := buildFirstBatchIdentity(node)
		if identity.EntityKey == "" || !strings.HasPrefix(identity.EntityKey, "chain_node:") || strings.TrimPrefix(identity.EntityKey, "chain_node:") == "" {
			report.Blockers = append(report.Blockers, fmt.Sprintf("node %d entity_key must use a nonblank chain_node prefix", index+1))
		}
		if identity.CanonicalName == "" {
			report.Blockers = append(report.Blockers, fmt.Sprintf("node %d canonical_name is required", index+1))
		}
		if _, exists := seenKeys[identity.EntityKey]; exists {
			report.Conflicts = append(report.Conflicts, fmt.Sprintf("duplicate entity_key %q", identity.EntityKey))
		}
		if _, exists := seenIDs[identity.EntityID]; exists {
			report.Conflicts = append(report.Conflicts, fmt.Sprintf("duplicate entity_id %q", identity.EntityID))
		}
		if _, exists := byCanonical[identity.CanonicalName]; exists {
			report.Conflicts = append(report.Conflicts, fmt.Sprintf("duplicate canonical_name %q", identity.CanonicalName))
		}
		seenKeys[identity.EntityKey] = struct{}{}
		seenIDs[identity.EntityID] = struct{}{}

		originals := normalizeOriginalNames(node.OriginalNames)
		if len(originals) == 0 {
			report.Blockers = append(report.Blockers, fmt.Sprintf("node %q original_names are required", identity.CanonicalName))
		}
		for _, original := range originals {
			if owner, exists := originalOwners[original]; exists && owner != identity.CanonicalName {
				report.Conflicts = append(report.Conflicts, fmt.Sprintf("original name %q belongs to both %q and %q", original, owner, identity.CanonicalName))
			} else if !exists {
				originalOwners[original] = identity.CanonicalName
				report.OriginalNameCount++
			}
		}
		if err := validateFirstBatchDefinition(identity.CanonicalName, originals, identity.Definition); err != nil {
			report.Blockers = append(report.Blockers, fmt.Sprintf("node %q definition: %v", identity.CanonicalName, err))
		}
		if node.WideBoundary && strings.TrimSpace(identity.BoundaryNote) == "" {
			report.Blockers = append(report.Blockers, fmt.Sprintf("node %q boundary_note is required for a wide boundary", identity.CanonicalName))
		}
		if node.WideBoundary {
			report.WideBoundaryNodeCount++
		}
		if node.BoundaryNote != "" && strings.TrimSpace(node.BoundaryNote) == "" {
			report.Blockers = append(report.Blockers, fmt.Sprintf("node %q boundary_note must be nonblank when present", identity.CanonicalName))
		}
		identity.Action = firstBatchIdentityAction(identity, snapshot, &report)
		byCanonical[identity.CanonicalName] = identity
		report.Nodes = append(report.Nodes, identity)
	}
	report.NodeCount = len(report.Nodes)

	seenMappingIdentities := map[string]string{}
	providersByCanonical := map[string]map[string]struct{}{}
	for index, draftMapping := range draft.Mappings {
		mapping := normalizeFirstBatchMappingDraft(draftMapping)
		identity, exists := byCanonical[mapping.CanonicalName]
		if !exists {
			report.Blockers = append(report.Blockers, fmt.Sprintf("mapping %d references unknown canonical_name %q", index+1, mapping.CanonicalName))
			continue
		}
		if !mapping.TaxonomyResolved {
			report.Blockers = append(report.Blockers, fmt.Sprintf("mapping %s:%s taxonomy is unresolved", mapping.SourceSystem, mapping.ExternalCode))
			continue
		}
		external := domain.EntityExternalIdentifier{
			ID:                 externalIdentifierSeedUUID(externalIdentifierIdentity(mapping.SourceSystem, mapping.SourceTaxonomyType, mapping.ExternalCode)),
			EntityID:           identity.EntityID,
			SourceSystem:       mapping.SourceSystem,
			SourceTaxonomyType: mapping.SourceTaxonomyType,
			ExternalCode:       mapping.ExternalCode,
			ExternalName:       mapping.ExternalName,
			Status:             domain.StatusActive,
		}
		if err := validateFirstBatchExternalIdentifier(external); err != nil {
			report.Blockers = append(report.Blockers, fmt.Sprintf("mapping %s:%s: %v", mapping.SourceSystem, mapping.ExternalCode, err))
			continue
		}
		externalIdentity := externalIdentifierIdentity(mapping.SourceSystem, mapping.SourceTaxonomyType, mapping.ExternalCode)
		if owner, duplicate := seenMappingIdentities[externalIdentity]; duplicate {
			report.Conflicts = append(report.Conflicts, fmt.Sprintf("external identity %q is duplicated for %q and %q", externalIdentity, owner, mapping.CanonicalName))
			continue
		}
		seenMappingIdentities[externalIdentity] = mapping.CanonicalName
		if providersByCanonical[mapping.CanonicalName] == nil {
			providersByCanonical[mapping.CanonicalName] = map[string]struct{}{}
		}
		providersByCanonical[mapping.CanonicalName][mapping.SourceSystem] = struct{}{}
		report.ProviderCounts[mapping.SourceSystem]++
		action := firstBatchMappingAction(external, snapshot.ExternalIdentifiers, &report)
		report.Mappings = append(report.Mappings, FirstBatchMappingReport{
			ID:                 external.ID,
			EntityID:           external.EntityID,
			CanonicalName:      mapping.CanonicalName,
			SourceSystem:       external.SourceSystem,
			SourceTaxonomyType: external.SourceTaxonomyType,
			ExternalCode:       external.ExternalCode,
			ExternalName:       external.ExternalName,
			Action:             action,
		})
	}
	report.MappingCount = len(report.Mappings)
	for _, providers := range providersByCanonical {
		if _, eastmoney := providers[ExternalSourceEastmoney]; eastmoney {
			if _, ths := providers[ExternalSourceTHS]; ths {
				report.DualSourceNodeCount++
			}
		}
	}
	validateFirstBatchCounts(&report, expectations)
	report.Ready = len(report.Blockers) == 0 && len(report.Conflicts) == 0
	return report
}

func buildFirstBatchIdentity(node FirstBatchNodeDraft) FirstBatchIdentity {
	key := strings.TrimSpace(node.EntityKey)
	canonical := strings.TrimSpace(node.CanonicalName)
	return FirstBatchIdentity{
		EntityID:      entitySeedUUID(key),
		EntityKey:     key,
		EntityType:    domain.EntityTypeChainNode,
		CanonicalName: canonical,
		Aliases:       aliasesFromOriginalNames(canonical, node.OriginalNames),
		Definition:    strings.TrimSpace(node.Definition),
		BoundaryNote:  strings.TrimSpace(node.BoundaryNote),
		Status:        domain.StatusActive,
	}
}

func normalizeOriginalNames(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func aliasesFromOriginalNames(canonical string, originals []string) []string {
	aliases := make([]string, 0, len(originals))
	for _, original := range normalizeOriginalNames(originals) {
		if original != canonical {
			aliases = append(aliases, original)
		}
	}
	return aliases
}

func validateFirstBatchDefinition(canonical string, originals []string, definition string) error {
	definition = strings.TrimSpace(definition)
	if definition == "" {
		return fmt.Errorf("is required")
	}
	if definition == canonical {
		return fmt.Errorf("must not copy canonical_name")
	}
	for _, original := range originals {
		if definition == original {
			return fmt.Errorf("must not copy an alias")
		}
	}
	for _, vague := range []string{"相关的产业链节点", "相关产业", "相关概念"} {
		if definition == canonical+vague || strings.Contains(definition, "与"+canonical+vague) {
			return fmt.Errorf("must not use a circular template")
		}
	}
	return nil
}

func firstBatchIdentityAction(identity FirstBatchIdentity, snapshot FirstBatchIdentitySnapshot, report *FirstBatchDryRunReport) string {
	conflictsBefore := len(report.Conflicts)
	existingStates := make([]FirstBatchIdentity, 0, 3)
	for _, lookup := range []struct {
		label    string
		value    string
		existing FirstBatchIdentity
		found    bool
	}{
		{label: "entity_key", value: identity.EntityKey, existing: snapshot.ByEntityKey[identity.EntityKey], found: snapshotHasIdentity(snapshot.ByEntityKey, identity.EntityKey)},
		{label: "entity_id", value: identity.EntityID, existing: snapshot.ByEntityID[identity.EntityID], found: snapshotHasIdentity(snapshot.ByEntityID, identity.EntityID)},
		{label: "canonical_name", value: identity.CanonicalName, existing: snapshot.ByCanonicalName[identity.CanonicalName], found: snapshotHasIdentity(snapshot.ByCanonicalName, identity.CanonicalName)},
	} {
		if !lookup.found {
			continue
		}
		existingStates = append(existingStates, lookup.existing)
		if !sameFirstBatchIdentityKey(lookup.existing, identity) {
			report.Conflicts = append(report.Conflicts, fmt.Sprintf("%s %q conflicts with existing identity", lookup.label, lookup.value))
		}
	}
	if len(existingStates) == 0 {
		return string(WriteCreated)
	}
	if len(existingStates) != 3 {
		report.Conflicts = append(report.Conflicts, fmt.Sprintf("snapshot indexes are incomplete for entity_key %q", identity.EntityKey))
	}
	for _, existing := range existingStates {
		if !sameFirstBatchIdentityState(existingStates[0], existing) {
			report.Conflicts = append(report.Conflicts, fmt.Sprintf("snapshot indexes disagree for entity_key %q", identity.EntityKey))
			break
		}
	}
	existing := existingStates[0]
	if existing.EntityType != domain.EntityTypeChainNode {
		report.Conflicts = append(report.Conflicts, fmt.Sprintf("entity_key %q has existing entity_type %q, want chain_node", identity.EntityKey, existing.EntityType))
	}
	if existing.Status != domain.StatusActive {
		report.Conflicts = append(report.Conflicts, fmt.Sprintf("entity_key %q has existing status %q, want active", identity.EntityKey, existing.Status))
	}
	if len(report.Conflicts) > conflictsBefore {
		return "conflict"
	}
	if !sameFirstBatchIdentityMutable(existing, identity) {
		return string(WriteUpdated)
	}
	return string(WriteUnchanged)
}

func snapshotHasIdentity(values map[string]FirstBatchIdentity, key string) bool {
	if values == nil {
		return false
	}
	_, ok := values[key]
	return ok
}

func sameFirstBatchIdentityKey(left, right FirstBatchIdentity) bool {
	return left.EntityID == right.EntityID && left.EntityKey == right.EntityKey && left.CanonicalName == right.CanonicalName
}

func sameFirstBatchIdentityState(left, right FirstBatchIdentity) bool {
	return sameFirstBatchIdentityKey(left, right) && left.EntityType == right.EntityType && left.Status == right.Status && sameFirstBatchIdentityMutable(left, right)
}

func sameFirstBatchIdentityMutable(left, right FirstBatchIdentity) bool {
	if left.Definition != right.Definition || left.BoundaryNote != right.BoundaryNote || len(left.Aliases) != len(right.Aliases) {
		return false
	}
	for index := range left.Aliases {
		if left.Aliases[index] != right.Aliases[index] {
			return false
		}
	}
	return true
}

func firstBatchMappingAction(identifier domain.EntityExternalIdentifier, snapshot FirstBatchExternalIdentifierSnapshot, report *FirstBatchDryRunReport) string {
	conflictsBefore := len(report.Conflicts)
	wanted := FirstBatchExternalIdentifier{
		ID: identifier.ID, EntityID: identifier.EntityID, SourceSystem: identifier.SourceSystem,
		SourceTaxonomyType: identifier.SourceTaxonomyType, ExternalCode: identifier.ExternalCode,
		ExternalName: identifier.ExternalName, Status: identifier.Status,
	}
	identity := externalIdentifierIdentity(wanted.SourceSystem, wanted.SourceTaxonomyType, wanted.ExternalCode)
	existingStates := make([]FirstBatchExternalIdentifier, 0, 2)
	if existing, ok := snapshot.ByIdentity[identity]; ok {
		existingStates = append(existingStates, existing)
		if externalIdentifierIdentity(existing.SourceSystem, existing.SourceTaxonomyType, existing.ExternalCode) != identity || existing.ID != wanted.ID || existing.EntityID != wanted.EntityID {
			report.Conflicts = append(report.Conflicts, fmt.Sprintf("external identity %q conflicts with existing binding or deterministic id", identity))
		}
	}
	if existing, ok := snapshot.ByID[wanted.ID]; ok {
		existingStates = append(existingStates, existing)
		if externalIdentifierIdentity(existing.SourceSystem, existing.SourceTaxonomyType, existing.ExternalCode) != identity || existing.EntityID != wanted.EntityID {
			report.Conflicts = append(report.Conflicts, fmt.Sprintf("external identifier id %q conflicts with existing identity or binding", wanted.ID))
		}
	}
	if len(existingStates) == 0 {
		return string(WriteCreated)
	}
	if len(existingStates) != 2 {
		report.Conflicts = append(report.Conflicts, fmt.Sprintf("external identifier snapshot indexes are incomplete for %q", identity))
	}
	for _, existing := range existingStates {
		if existing != existingStates[0] {
			report.Conflicts = append(report.Conflicts, fmt.Sprintf("external identifier snapshot indexes disagree for %q", identity))
			break
		}
	}
	if len(report.Conflicts) > conflictsBefore {
		return "conflict"
	}
	existing := existingStates[0]
	if existing.ExternalName != wanted.ExternalName || existing.Status != wanted.Status {
		return string(WriteUpdated)
	}
	return string(WriteUnchanged)
}

func normalizeFirstBatchMappingDraft(mapping FirstBatchMappingDraft) FirstBatchMappingDraft {
	mapping.CanonicalName = strings.TrimSpace(mapping.CanonicalName)
	mapping.SourceSystem = strings.ToLower(strings.TrimSpace(mapping.SourceSystem))
	mapping.SourceTaxonomyType = strings.ToLower(strings.TrimSpace(mapping.SourceTaxonomyType))
	mapping.ExternalCode = strings.TrimSpace(mapping.ExternalCode)
	mapping.ExternalName = strings.TrimSpace(mapping.ExternalName)
	return mapping
}

func validateFirstBatchCounts(report *FirstBatchDryRunReport, expectations FirstBatchExpectations) {
	checks := []struct {
		name string
		got  int
		want int
	}{
		{"nodes", report.NodeCount, expectations.Nodes},
		{"original_names", report.OriginalNameCount, expectations.OriginalNames},
		{"wide_boundary_nodes", report.WideBoundaryNodeCount, expectations.WideBoundaryNodes},
		{"mappings", report.MappingCount, expectations.Mappings},
		{"eastmoney_mappings", report.ProviderCounts[ExternalSourceEastmoney], expectations.EastmoneyMappings},
		{"ths_mappings", report.ProviderCounts[ExternalSourceTHS], expectations.THSMappings},
		{"dual_source_nodes", report.DualSourceNodeCount, expectations.DualSourceNodes},
	}
	for _, check := range checks {
		if check.got != check.want {
			report.Blockers = append(report.Blockers, fmt.Sprintf("%s count %d does not match approved %d", check.name, check.got, check.want))
		}
	}
	sort.Strings(report.Blockers)
	sort.Strings(report.Conflicts)
}
