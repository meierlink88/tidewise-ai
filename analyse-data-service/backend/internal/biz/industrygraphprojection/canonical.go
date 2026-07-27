package industrygraphprojection

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sort"
)

type ProjectionSummary struct {
	NodeCount                  int                      `json:"node_count"`
	RelationshipCount          int                      `json:"relationship_count"`
	NodeFingerprint            string                   `json:"node_fingerprint"`
	RelationshipFingerprint    string                   `json:"relationship_fingerprint"`
	NodeTypeCounts             map[EntityType]int       `json:"node_type_counts"`
	RelationshipTypeCounts     map[RelationshipType]int `json:"relationship_type_counts"`
	OrphanCount                int                      `json:"orphan_count"`
	DuplicateNodeCount         int                      `json:"duplicate_node_count"`
	DuplicateRelationshipCount int                      `json:"duplicate_relationship_count"`
	SelfLoopCount              int                      `json:"self_loop_count"`
	MissingChainIdentityCount  int                      `json:"missing_chain_identity_count"`
}

func ProjectionsEqual(left, right Projection) bool {
	return reflect.DeepEqual(canonicalProjection(left), canonicalProjection(right))
}

func SummarizeProjection(projection Projection) ProjectionSummary {
	canonical := canonicalProjection(projection)
	nodeTypeCounts := make(map[EntityType]int)
	nodeIDs := make(map[string]struct{}, len(canonical.Nodes))
	nodeKeys := make(map[string]struct{}, len(canonical.Nodes))
	degree := make(map[string]int, len(canonical.Nodes))
	duplicateNodeCount := 0
	for _, node := range canonical.Nodes {
		nodeTypeCounts[node.EntityType]++
		_, duplicateID := nodeIDs[node.EntityID]
		_, duplicateKey := nodeKeys[node.EntityKey]
		if duplicateID || duplicateKey {
			duplicateNodeCount++
		}
		nodeIDs[node.EntityID] = struct{}{}
		nodeKeys[node.EntityKey] = struct{}{}
	}
	relationshipTypeCounts := make(map[RelationshipType]int)
	relationKeys := make(map[string]struct{}, len(canonical.Relationships))
	duplicateRelationshipCount := 0
	selfLoopCount := 0
	missingChainIdentityCount := 0
	for _, relationship := range canonical.Relationships {
		relationshipTypeCounts[relationship.Type]++
		if _, duplicate := relationKeys[relationship.RelationKey]; duplicate {
			duplicateRelationshipCount++
		}
		relationKeys[relationship.RelationKey] = struct{}{}
		if relationship.FromEntityID == relationship.ToEntityID {
			selfLoopCount++
		}
		if requiresChainIdentity(relationship.Type) && relationship.ChainID == "" {
			missingChainIdentityCount++
		}
		degree[relationship.FromEntityID]++
		degree[relationship.ToEntityID]++
	}
	orphanCount := 0
	for _, node := range canonical.Nodes {
		if degree[node.EntityID] == 0 {
			orphanCount++
		}
	}
	return ProjectionSummary{
		NodeCount:                  len(canonical.Nodes),
		RelationshipCount:          len(canonical.Relationships),
		NodeFingerprint:            semanticFingerprint(canonical.Nodes),
		RelationshipFingerprint:    semanticFingerprint(canonical.Relationships),
		NodeTypeCounts:             nodeTypeCounts,
		RelationshipTypeCounts:     relationshipTypeCounts,
		OrphanCount:                orphanCount,
		DuplicateNodeCount:         duplicateNodeCount,
		DuplicateRelationshipCount: duplicateRelationshipCount,
		SelfLoopCount:              selfLoopCount,
		MissingChainIdentityCount:  missingChainIdentityCount,
	}
}

func requiresChainIdentity(value RelationshipType) bool {
	switch value {
	case RelationshipTypeMappedToIndustry,
		RelationshipTypeMappedToConcept,
		RelationshipTypeHasNode,
		RelationshipTypeInputTo,
		RelationshipTypeIsComponentOf,
		RelationshipTypeDependsOn:
		return true
	default:
		return false
	}
}

func canonicalProjection(value Projection) Projection {
	canonical := value
	canonical.Nodes = append([]Node(nil), value.Nodes...)
	for index := range canonical.Nodes {
		canonical.Nodes[index].Aliases = append([]string(nil), canonical.Nodes[index].Aliases...)
		sort.Strings(canonical.Nodes[index].Aliases)
	}
	canonical.Relationships = append([]Relationship(nil), value.Relationships...)
	sort.Slice(canonical.Nodes, func(left, right int) bool {
		if canonical.Nodes[left].EntityID != canonical.Nodes[right].EntityID {
			return canonical.Nodes[left].EntityID < canonical.Nodes[right].EntityID
		}
		return canonical.Nodes[left].EntityKey < canonical.Nodes[right].EntityKey
	})
	sort.Slice(canonical.Relationships, func(left, right int) bool {
		leftValue := canonical.Relationships[left]
		rightValue := canonical.Relationships[right]
		if leftValue.RelationKey != rightValue.RelationKey {
			return leftValue.RelationKey < rightValue.RelationKey
		}
		if leftValue.Type != rightValue.Type {
			return leftValue.Type < rightValue.Type
		}
		if leftValue.FromEntityID != rightValue.FromEntityID {
			return leftValue.FromEntityID < rightValue.FromEntityID
		}
		return leftValue.ToEntityID < rightValue.ToEntityID
	})
	return canonical
}

func semanticFingerprint(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}
