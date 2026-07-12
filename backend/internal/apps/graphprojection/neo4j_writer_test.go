package graphprojection

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNeo4jGraphWriterUpsertsEntityNodes(t *testing.T) {
	driver := &recordingCypherDriver{}
	writer := NewNeo4jGraphWriter(driver, "neo4j")
	localTime := time.Date(2026, 7, 10, 13, 0, 0, 0, time.Local)

	result, err := writer.UpsertEntities(context.Background(), []GraphNode{{
		EntityID:      "entity-1",
		EntityKey:     "economy:cn",
		EntityType:    "economy",
		LayerCode:     "economy",
		Name:          "中国",
		CanonicalName: "中国",
		Status:        "active",
		Namespace:     "tidewise",
		UpdatedAt:     localTime,
	}})
	if err != nil {
		t.Fatalf("UpsertEntities() error = %v", err)
	}

	if result.Projected != 1 {
		t.Fatalf("Projected = %d, want 1", result.Projected)
	}
	if len(driver.calls) != 1 {
		t.Fatalf("driver calls = %d, want 1", len(driver.calls))
	}
	call := driver.calls[0]
	for _, fragment := range []string{
		"unwind $nodes as row",
		"merge (entity:entity",
		"projection_namespace",
		"entity.entity_key = row.entity_key",
	} {
		if !strings.Contains(strings.ToLower(call.query), fragment) {
			t.Fatalf("node upsert query missing %q: %s", fragment, call.query)
		}
	}
	if strings.Contains(strings.ToLower(call.query), "tidewiseentity") {
		t.Fatalf("node upsert query contains obsolete TidewiseEntity label: %s", call.query)
	}
	if call.database != "neo4j" {
		t.Fatalf("database = %q, want neo4j", call.database)
	}
	nodes := call.params["nodes"].([]map[string]any)
	if got, ok := nodes[0]["updated_at"].(string); !ok || got != localTime.UTC().Format(time.RFC3339Nano) {
		t.Fatalf("updated_at param = %#v, want UTC RFC3339 string", nodes[0]["updated_at"])
	}
}

func TestNeo4jGraphWriterUpsertsRelationshipsByType(t *testing.T) {
	driver := &recordingCypherDriver{}
	writer := NewNeo4jGraphWriter(driver, "neo4j")

	result, err := writer.UpsertRelationships(context.Background(), []GraphRelationship{{
		EdgeID:               "edge-1",
		FromEntityID:         "entity-1",
		ToEntityID:           "entity-2",
		RelationshipType:     "MEMBER_OF",
		OriginalRelationType: "member_of",
		Source:               "postgres_entity_edges",
		Confidence:           1,
		Status:               "active",
		Namespace:            "tidewise",
	}})
	if err != nil {
		t.Fatalf("UpsertRelationships() error = %v", err)
	}
	if result.Projected != 1 {
		t.Fatalf("Projected = %d, want 1", result.Projected)
	}
	if len(driver.calls) != 1 {
		t.Fatalf("driver calls = %d, want 1", len(driver.calls))
	}
	query := driver.calls[0].query
	for _, fragment := range []string{
		"unwind $relationships as row",
		"match (from:Entity",
		"match (to:Entity",
		"merge (from)-[relationship:MEMBER_OF",
		"relationship.original_relation_type = row.original_relation_type",
	} {
		if !strings.Contains(strings.ToLower(query), strings.ToLower(fragment)) {
			t.Fatalf("relationship upsert query missing %q: %s", fragment, query)
		}
	}
	if strings.Contains(strings.ToLower(query), "tidewiseentity") {
		t.Fatalf("relationship upsert query contains obsolete TidewiseEntity label: %s", query)
	}
}

func TestNeo4jGraphWriterDeletesOnlyNamespace(t *testing.T) {
	driver := &recordingCypherDriver{}
	writer := NewNeo4jGraphWriter(driver, "neo4j")

	if err := writer.DeleteNamespace(context.Background(), "tidewise"); err != nil {
		t.Fatalf("DeleteNamespace() error = %v", err)
	}
	query := driver.calls[0].query
	for _, fragment := range []string{
		"match (entity:entity",
		"projection_namespace: $namespace",
		"detach delete entity",
	} {
		if !strings.Contains(strings.ToLower(query), fragment) {
			t.Fatalf("delete query missing %q: %s", fragment, query)
		}
	}
	if strings.Contains(strings.ToLower(query), "tidewiseentity") {
		t.Fatalf("delete query contains obsolete TidewiseEntity label: %s", query)
	}
}

type recordingCypherDriver struct {
	calls []cypherCall
}

func (d *recordingCypherDriver) ExecuteWrite(_ context.Context, database string, query string, params map[string]any) error {
	d.calls = append(d.calls, cypherCall{database: database, query: query, params: params})
	return nil
}

type cypherCall struct {
	database string
	query    string
	params   map[string]any
}
