package industrygraphprojection

import (
	"fmt"
	"strings"
)

func ValidateProjection(projection Projection) error {
	if !ValidPackageSHA256(projection.PackageSHA256) {
		return fmt.Errorf("package SHA-256 must contain 64 lowercase hexadecimal characters")
	}
	nodes := make(map[string]Node, len(projection.Nodes))
	entityKeys := make(map[string]string, len(projection.Nodes))
	degree := make(map[string]int, len(projection.Nodes))
	for index, node := range projection.Nodes {
		if strings.TrimSpace(node.EntityID) == "" ||
			strings.TrimSpace(node.EntityKey) == "" ||
			strings.TrimSpace(node.CanonicalName) == "" {
			return fmt.Errorf("node %d requires entity_id, entity_key and canonical_name", index)
		}
		if !validEntityType(node.EntityType) {
			return fmt.Errorf("node %q has unsupported entity_type %q", node.EntityID, node.EntityType)
		}
		if _, duplicate := nodes[node.EntityID]; duplicate {
			return fmt.Errorf("duplicate node entity_id %q", node.EntityID)
		}
		if previous, duplicate := entityKeys[node.EntityKey]; duplicate {
			return fmt.Errorf("duplicate node entity_key %q on %q and %q", node.EntityKey, previous, node.EntityID)
		}
		nodes[node.EntityID] = node
		entityKeys[node.EntityKey] = node.EntityID
	}

	relationKeys := make(map[string]struct{}, len(projection.Relationships))
	memberships := make(map[string]map[string]struct{})
	graphEdges := make(map[string][]Relationship)
	for _, relationship := range projection.Relationships {
		if strings.TrimSpace(relationship.RelationKey) == "" {
			return fmt.Errorf("relationship requires relation_key")
		}
		if _, duplicate := relationKeys[relationship.RelationKey]; duplicate {
			return fmt.Errorf("duplicate relationship relation_key %q", relationship.RelationKey)
		}
		relationKeys[relationship.RelationKey] = struct{}{}
		from, fromExists := nodes[relationship.FromEntityID]
		to, toExists := nodes[relationship.ToEntityID]
		if !fromExists || !toExists {
			return fmt.Errorf("relationship %q has a missing endpoint", relationship.RelationKey)
		}
		if relationship.FromEntityID == relationship.ToEntityID {
			return fmt.Errorf("relationship %q is a self-loop", relationship.RelationKey)
		}
		if strings.TrimSpace(relationship.Mechanism) == "" {
			return fmt.Errorf("relationship %q requires mechanism", relationship.RelationKey)
		}
		if err := validateRelationshipShape(relationship, from, to); err != nil {
			return err
		}
		if relationship.Type == RelationshipTypeHasNode {
			if memberships[relationship.ChainID] == nil {
				memberships[relationship.ChainID] = make(map[string]struct{})
			}
			memberships[relationship.ChainID][relationship.ToEntityID] = struct{}{}
		}
		if isChainGraphType(relationship.Type) {
			graphEdges[relationship.ChainID] = append(graphEdges[relationship.ChainID], relationship)
		}
		degree[relationship.FromEntityID]++
		degree[relationship.ToEntityID]++
	}
	for _, node := range projection.Nodes {
		if degree[node.EntityID] == 0 {
			return fmt.Errorf("projected node %q is isolated", node.EntityID)
		}
	}
	if err := validateHierarchies(nodes, projection.Relationships); err != nil {
		return err
	}
	if err := validateChainTopologies(nodes, memberships, graphEdges); err != nil {
		return err
	}
	return nil
}

func validateRelationshipShape(relationship Relationship, from, to Node) error {
	switch relationship.Type {
	case RelationshipTypeMappedToIndustry:
		if from.EntityType != EntityTypeIndustryChain || to.EntityType != EntityTypeIndustry {
			return fmt.Errorf("relationship %q has invalid MAPPED_TO_INDUSTRY direction", relationship.RelationKey)
		}
		if relationship.ChainID != relationship.FromEntityID {
			return fmt.Errorf("relationship %q chain_id must equal its Industry Chain endpoint", relationship.RelationKey)
		}
	case RelationshipTypeMappedToConcept:
		if from.EntityType != EntityTypeIndustryChain || to.EntityType != EntityTypeConcept {
			return fmt.Errorf("relationship %q has invalid MAPPED_TO_CONCEPT direction", relationship.RelationKey)
		}
		if relationship.ChainID != relationship.FromEntityID {
			return fmt.Errorf("relationship %q chain_id must equal its Industry Chain endpoint", relationship.RelationKey)
		}
	case RelationshipTypeHasNode:
		if from.EntityType != EntityTypeIndustryChain || to.EntityType != EntityTypeChainNode {
			return fmt.Errorf("relationship %q has invalid HAS_NODE direction", relationship.RelationKey)
		}
		if relationship.ChainID != relationship.FromEntityID {
			return fmt.Errorf("relationship %q chain_id must equal its Industry Chain endpoint", relationship.RelationKey)
		}
		if relationship.Position == nil || *relationship.Position <= 0 {
			return fmt.Errorf("relationship %q requires a positive position", relationship.RelationKey)
		}
		switch relationship.ContextualStage {
		case "upstream", "midstream", "downstream":
		default:
			return fmt.Errorf("relationship %q has invalid contextual_stage", relationship.RelationKey)
		}
	case RelationshipTypeInputTo, RelationshipTypeIsComponentOf, RelationshipTypeDependsOn:
		if from.EntityType != EntityTypeChainNode || to.EntityType != EntityTypeChainNode {
			return fmt.Errorf("relationship %q has invalid Chain Node direction", relationship.RelationKey)
		}
		if strings.TrimSpace(relationship.ChainID) == "" {
			return fmt.Errorf("relationship %q requires chain_id", relationship.RelationKey)
		}
	case RelationshipTypeIsSubcategoryOf:
		validIndustryHierarchy := from.EntityType == EntityTypeIndustry && to.EntityType == EntityTypeIndustry
		validNodeHierarchy := from.EntityType == EntityTypeChainNode && to.EntityType == EntityTypeChainNode
		if !validIndustryHierarchy && !validNodeHierarchy {
			return fmt.Errorf("relationship %q has invalid IS_SUBCATEGORY_OF direction", relationship.RelationKey)
		}
		if strings.TrimSpace(relationship.ChainID) != "" {
			return fmt.Errorf("relationship %q must not have chain_id", relationship.RelationKey)
		}
	default:
		return fmt.Errorf("relationship %q has unsupported type %q", relationship.RelationKey, relationship.Type)
	}

	switch relationship.Type {
	case RelationshipTypeMappedToIndustry,
		RelationshipTypeMappedToConcept,
		RelationshipTypeHasNode,
		RelationshipTypeInputTo,
		RelationshipTypeIsComponentOf,
		RelationshipTypeDependsOn:
		if strings.TrimSpace(relationship.ChainID) == "" {
			return fmt.Errorf("relationship %q requires chain_id", relationship.RelationKey)
		}
	}
	return nil
}

func validEntityType(value EntityType) bool {
	switch value {
	case EntityTypeIndustry, EntityTypeConcept, EntityTypeIndustryChain, EntityTypeChainNode:
		return true
	default:
		return false
	}
}

func isChainGraphType(value RelationshipType) bool {
	switch value {
	case RelationshipTypeInputTo, RelationshipTypeIsComponentOf, RelationshipTypeDependsOn:
		return true
	default:
		return false
	}
}

func validateHierarchies(nodes map[string]Node, relationships []Relationship) error {
	hierarchyNodes := map[EntityType]map[string]struct{}{
		EntityTypeIndustry:  {},
		EntityTypeChainNode: {},
	}
	hierarchyEdges := map[EntityType]map[string][]string{
		EntityTypeIndustry:  {},
		EntityTypeChainNode: {},
	}
	for _, relationship := range relationships {
		if relationship.Type != RelationshipTypeIsSubcategoryOf {
			continue
		}
		entityType := nodes[relationship.FromEntityID].EntityType
		hierarchyNodes[entityType][relationship.FromEntityID] = struct{}{}
		hierarchyNodes[entityType][relationship.ToEntityID] = struct{}{}
		hierarchyEdges[entityType][relationship.FromEntityID] = append(
			hierarchyEdges[entityType][relationship.FromEntityID],
			relationship.ToEntityID,
		)
	}
	for _, entityType := range []EntityType{EntityTypeIndustry, EntityTypeChainNode} {
		if directedCycle(hierarchyNodes[entityType], hierarchyEdges[entityType]) {
			return fmt.Errorf("%s is_subcategory_of hierarchy must be acyclic", entityType)
		}
	}
	return nil
}

func validateChainTopologies(
	nodes map[string]Node,
	memberships map[string]map[string]struct{},
	graphEdges map[string][]Relationship,
) error {
	for entityID, node := range nodes {
		if node.EntityType != EntityTypeIndustryChain {
			continue
		}
		if len(memberships[entityID]) < 2 {
			return fmt.Errorf("Industry Chain %q requires at least two active memberships", entityID)
		}
	}

	for chainID, edges := range graphEdges {
		members := memberships[chainID]
		if len(members) < 2 {
			return fmt.Errorf("chain %q graph edges require active memberships", chainID)
		}
		undirected := make(map[string]map[string]struct{}, len(members))
		directed := make(map[string][]string, len(members))
		for nodeID := range members {
			undirected[nodeID] = make(map[string]struct{})
		}
		for _, edge := range edges {
			if _, ok := members[edge.FromEntityID]; !ok {
				return fmt.Errorf("relationship %q endpoints must be active memberships in chain %q", edge.RelationKey, chainID)
			}
			if _, ok := members[edge.ToEntityID]; !ok {
				return fmt.Errorf("relationship %q endpoints must be active memberships in chain %q", edge.RelationKey, chainID)
			}
			undirected[edge.FromEntityID][edge.ToEntityID] = struct{}{}
			undirected[edge.ToEntityID][edge.FromEntityID] = struct{}{}
			directed[edge.FromEntityID] = append(directed[edge.FromEntityID], edge.ToEntityID)
		}
		for nodeID, neighbours := range undirected {
			if len(neighbours) == 0 {
				return fmt.Errorf("membership %q in chain %q has no chain graph edge", nodeID, chainID)
			}
		}
		if !weaklyConnected(members, undirected) {
			return fmt.Errorf("chain %q graph must be weakly connected", chainID)
		}
		if directedCycle(members, directed) {
			return fmt.Errorf("chain %q graph must be acyclic", chainID)
		}
	}

	for chainID := range memberships {
		if len(graphEdges[chainID]) == 0 {
			return fmt.Errorf("Industry Chain %q requires at least one graph edge", chainID)
		}
	}
	return nil
}

func weaklyConnected(nodes map[string]struct{}, adjacency map[string]map[string]struct{}) bool {
	if len(nodes) == 0 {
		return false
	}
	var first string
	for nodeID := range nodes {
		first = nodeID
		break
	}
	seen := map[string]struct{}{first: {}}
	pending := []string{first}
	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		for next := range adjacency[current] {
			if _, ok := seen[next]; ok {
				continue
			}
			seen[next] = struct{}{}
			pending = append(pending, next)
		}
	}
	return len(seen) == len(nodes)
}

func directedCycle(nodes map[string]struct{}, adjacency map[string][]string) bool {
	const (
		unseen   = 0
		visiting = 1
		visited  = 2
	)
	state := make(map[string]int, len(nodes))
	var visit func(string) bool
	visit = func(nodeID string) bool {
		switch state[nodeID] {
		case visiting:
			return true
		case visited:
			return false
		}
		state[nodeID] = visiting
		for _, next := range adjacency[nodeID] {
			if visit(next) {
				return true
			}
		}
		state[nodeID] = visited
		return false
	}
	for nodeID := range nodes {
		if visit(nodeID) {
			return true
		}
	}
	return false
}

// ValidPackageSHA256 reports whether value is a canonical lowercase SHA-256.
func ValidPackageSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}
