package graphprojection

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

type RelationshipMapStatus string

const (
	RelationshipMapStatusProjected RelationshipMapStatus = "projected"
	RelationshipMapStatusSkipped   RelationshipMapStatus = "skipped"
	RelationshipMapStatusFailed    RelationshipMapStatus = "failed"
)

type RelationshipMapReport struct {
	EdgeID string
	Status RelationshipMapStatus
	Reason string
}

var safeRelationPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

var knownRelationTypes = map[string]string{
	"member_of":       "MEMBER_OF",
	"has_market":      "HAS_MARKET",
	"tracks_index":    "TRACKS_INDEX",
	"issues":          "ISSUES",
	"participates_in": "PARTICIPATES_IN",
	"affiliated_with": "AFFILIATED_WITH",
	"applies_to":      "APPLIES_TO",
	"related_to":      "RELATED_TO",
	"self":            "RELATED_TO",
}

func MapEntityNode(node repositories.GraphEntityNode, namespace string) (GraphNode, error) {
	if node.ID == "" {
		return GraphNode{}, fmt.Errorf("entity id is required")
	}
	if node.EntityKey == "" {
		return GraphNode{}, fmt.Errorf("entity key is required")
	}
	if node.EntityType == "" {
		return GraphNode{}, fmt.Errorf("entity type is required")
	}
	if node.LayerCode == "" {
		return GraphNode{}, fmt.Errorf("layer code is required")
	}
	if node.Name == "" {
		return GraphNode{}, fmt.Errorf("entity name is required")
	}
	if node.CanonicalName == "" {
		return GraphNode{}, fmt.Errorf("canonical name is required")
	}
	if node.Status == "" {
		return GraphNode{}, fmt.Errorf("entity status is required")
	}
	if namespace == "" {
		return GraphNode{}, fmt.Errorf("projection namespace is required")
	}

	return GraphNode{
		EntityID:      node.ID,
		EntityKey:     node.EntityKey,
		EntityType:    string(node.EntityType),
		LayerCode:     node.LayerCode,
		Name:          node.Name,
		CanonicalName: node.CanonicalName,
		Status:        string(node.Status),
		Namespace:     namespace,
		UpdatedAt:     node.UpdatedAt,
	}, nil
}

func MapRelationType(relationType string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(relationType))
	if normalized == "" {
		return "", fmt.Errorf("relation type is required")
	}
	if !safeRelationPattern.MatchString(strings.TrimSpace(relationType)) {
		return "RELATED_TO", nil
	}
	if mapped, ok := knownRelationTypes[normalized]; ok {
		return mapped, nil
	}
	return "RELATED_TO", nil
}

func MapEntityRelationship(edge repositories.GraphEntityEdge, nodes map[string]GraphNode, namespace string) (*GraphRelationship, RelationshipMapReport) {
	report := RelationshipMapReport{EdgeID: edge.ID}
	if edge.Status == domain.StatusInactive {
		report.Status = RelationshipMapStatusSkipped
		report.Reason = "inactive relationship"
		return nil, report
	}
	if _, ok := nodes[edge.FromEntityID]; !ok {
		report.Status = RelationshipMapStatusSkipped
		report.Reason = "missing endpoint: from entity"
		return nil, report
	}
	if _, ok := nodes[edge.ToEntityID]; !ok {
		report.Status = RelationshipMapStatusSkipped
		report.Reason = "missing endpoint: to entity"
		return nil, report
	}

	mappedType, err := MapRelationType(edge.RelationType)
	if err != nil {
		report.Status = RelationshipMapStatusFailed
		report.Reason = err.Error()
		return nil, report
	}

	report.Status = RelationshipMapStatusProjected
	return &GraphRelationship{
		EdgeID:               edge.ID,
		FromEntityID:         edge.FromEntityID,
		ToEntityID:           edge.ToEntityID,
		RelationshipType:     mappedType,
		OriginalRelationType: edge.RelationType,
		Source:               "postgres_entity_edges",
		Confidence:           1,
		Status:               string(edge.Status),
		Namespace:            namespace,
		UpdatedAt:            edge.UpdatedAt,
	}, report
}

func MapEntityRelationships(edges []repositories.GraphEntityEdge, nodes map[string]GraphNode, namespace string) ([]GraphRelationship, []RelationshipMapReport) {
	relationships := make([]GraphRelationship, 0, len(edges))
	reports := make([]RelationshipMapReport, 0, len(edges))
	seen := map[string]struct{}{}
	for _, edge := range edges {
		if _, ok := seen[edge.ID]; ok {
			reports = append(reports, RelationshipMapReport{
				EdgeID: edge.ID,
				Status: RelationshipMapStatusSkipped,
				Reason: "duplicate relationship edge",
			})
			continue
		}
		seen[edge.ID] = struct{}{}

		relationship, report := MapEntityRelationship(edge, nodes, namespace)
		reports = append(reports, report)
		if relationship != nil {
			relationships = append(relationships, *relationship)
		}
	}
	return relationships, reports
}
