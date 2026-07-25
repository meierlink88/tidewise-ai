package graphprojection

import (
	"context"
	"fmt"
	"sort"
	"time"
)

type CypherWriter interface {
	ExecuteWrite(context.Context, string, string, map[string]any) error
}

type Neo4jGraphWriter struct {
	driver   CypherWriter
	database string
}

func NewNeo4jGraphWriter(driver CypherWriter, database string) Neo4jGraphWriter {
	return Neo4jGraphWriter{driver: driver, database: database}
}

func (w Neo4jGraphWriter) UpsertEntities(ctx context.Context, nodes []GraphNode) (GraphWriteResult, error) {
	if len(nodes) == 0 {
		return GraphWriteResult{}, nil
	}
	params := map[string]any{"nodes": graphNodeParams(nodes)}
	query := `
UNWIND $nodes AS row
MERGE (entity:Entity {
    entity_id: row.entity_id,
    projection_namespace: row.projection_namespace
})
SET entity.entity_key = row.entity_key,
    entity.entity_type = row.entity_type,
    entity.layer_code = row.layer_code,
    entity.name = row.name,
    entity.canonical_name = row.canonical_name,
    entity.aliases = row.aliases,
    entity.status = row.status,
    entity.updated_at = row.updated_at
FOREACH (_ IN CASE WHEN row.classification_code IS NULL THEN [1] ELSE [] END |
    REMOVE entity.classification_code)
FOREACH (_ IN CASE WHEN row.classification_code IS NULL THEN [] ELSE [1] END |
    SET entity.classification_code = row.classification_code)
`
	if err := w.driver.ExecuteWrite(ctx, w.database, query, params); err != nil {
		return GraphWriteResult{}, err
	}
	return GraphWriteResult{Projected: len(nodes)}, nil
}

func (w Neo4jGraphWriter) UpsertRelationships(ctx context.Context, relationships []GraphRelationship) (GraphWriteResult, error) {
	if len(relationships) == 0 {
		return GraphWriteResult{}, nil
	}
	grouped := map[string][]GraphRelationship{}
	for _, relationship := range relationships {
		grouped[relationship.RelationshipType] = append(grouped[relationship.RelationshipType], relationship)
	}

	types := make([]string, 0, len(grouped))
	for relationshipType := range grouped {
		types = append(types, relationshipType)
	}
	sort.Strings(types)

	for _, relationshipType := range types {
		params := map[string]any{"relationships": graphRelationshipParams(grouped[relationshipType])}
		query := fmt.Sprintf(`
UNWIND $relationships AS row
MATCH (from:Entity {
    entity_id: row.from_entity_id,
    projection_namespace: row.projection_namespace
})
MATCH (to:Entity {
    entity_id: row.to_entity_id,
    projection_namespace: row.projection_namespace
})
MERGE (from)-[relationship:%s {
    edge_id: row.edge_id,
    projection_namespace: row.projection_namespace
}]->(to)
SET relationship.original_relation_type = row.original_relation_type,
    relationship.source = row.source,
    relationship.confidence = row.confidence,
    relationship.status = row.status,
    relationship.updated_at = row.updated_at
`, relationshipType)
		if err := w.driver.ExecuteWrite(ctx, w.database, query, params); err != nil {
			return GraphWriteResult{}, err
		}
	}
	return GraphWriteResult{Projected: len(relationships)}, nil
}

func (w Neo4jGraphWriter) DeleteNamespace(ctx context.Context, namespace string) error {
	query := `
MATCH (entity:Entity {projection_namespace: $namespace})
DETACH DELETE entity
`
	return w.driver.ExecuteWrite(ctx, w.database, query, map[string]any{"namespace": namespace})
}

func graphNodeParams(nodes []GraphNode) []map[string]any {
	params := make([]map[string]any, 0, len(nodes))
	for _, node := range nodes {
		item := map[string]any{
			"entity_id":            node.EntityID,
			"entity_key":           node.EntityKey,
			"entity_type":          node.EntityType,
			"layer_code":           node.LayerCode,
			"name":                 node.Name,
			"canonical_name":       node.CanonicalName,
			"aliases":              append([]string(nil), node.Aliases...),
			"status":               node.Status,
			"projection_namespace": node.Namespace,
			"updated_at":           neo4jTimeParam(node.UpdatedAt),
		}
		if node.ClassificationCode != "" {
			item["classification_code"] = string(node.ClassificationCode)
		}
		params = append(params, item)
	}
	return params
}

func graphRelationshipParams(relationships []GraphRelationship) []map[string]any {
	params := make([]map[string]any, 0, len(relationships))
	for _, relationship := range relationships {
		params = append(params, map[string]any{
			"edge_id":                relationship.EdgeID,
			"from_entity_id":         relationship.FromEntityID,
			"to_entity_id":           relationship.ToEntityID,
			"original_relation_type": relationship.OriginalRelationType,
			"source":                 relationship.Source,
			"confidence":             relationship.Confidence,
			"status":                 relationship.Status,
			"projection_namespace":   relationship.Namespace,
			"updated_at":             neo4jTimeParam(relationship.UpdatedAt),
		})
	}
	return params
}

func neo4jTimeParam(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
