package neo4j

import (
	"strings"
	"testing"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/industrygraphprojection"
)

func TestProjectionCypherRegistryUsesOnlyFrozenStaticLabelsAndTypes(t *testing.T) {
	nodeQueries := nodeCreateQueries()
	for _, entityType := range []biz.EntityType{
		biz.EntityTypeIndustry,
		biz.EntityTypeConcept,
		biz.EntityTypeIndustryChain,
		biz.EntityTypeChainNode,
	} {
		query, ok := nodeQueries[entityType]
		if !ok || !strings.Contains(query, ":TidewiseEntity:") {
			t.Fatalf("node query %q = %q", entityType, query)
		}
	}
	if len(nodeQueries) != 4 {
		t.Fatalf("node query count = %d, want 4", len(nodeQueries))
	}

	relationshipQueries := relationshipCreateQueries()
	for _, relationshipType := range []biz.RelationshipType{
		biz.RelationshipTypeMappedToIndustry,
		biz.RelationshipTypeMappedToConcept,
		biz.RelationshipTypeHasNode,
		biz.RelationshipTypeInputTo,
		biz.RelationshipTypeIsComponentOf,
		biz.RelationshipTypeDependsOn,
		biz.RelationshipTypeIsSubcategoryOf,
	} {
		query, ok := relationshipQueries[relationshipType]
		if !ok || !strings.Contains(query, "projection_namespace: $namespace") {
			t.Fatalf("relationship query %q = %q", relationshipType, query)
		}
	}
	if len(relationshipQueries) != 7 {
		t.Fatalf("relationship query count = %d, want 7", len(relationshipQueries))
	}

	constraints := constraintDefinitions()
	if len(constraints) != 9 {
		t.Fatalf("constraint count = %d, want 9", len(constraints))
	}
	names := make(map[string]struct{}, len(constraints))
	for _, constraint := range constraints {
		if constraint.Name == "" || constraint.LabelOrType == "" || len(constraint.Properties) == 0 {
			t.Fatalf("incomplete constraint = %#v", constraint)
		}
		if _, duplicate := names[constraint.Name]; duplicate {
			t.Fatalf("duplicate constraint name %q", constraint.Name)
		}
		names[constraint.Name] = struct{}{}
		if !strings.Contains(constraint.Query, "IF NOT EXISTS") {
			t.Fatalf("constraint %q is not idempotent: %s", constraint.Name, constraint.Query)
		}
	}
}

func TestProjectionRowsPreserveTechnicalAndChainScopedIdentity(t *testing.T) {
	position := 2
	projection := biz.Projection{
		PackageSHA256: strings.Repeat("a", 64),
		Nodes: []biz.Node{{
			EntityID: "chain", EntityKey: "industry_chain:test",
			EntityType: biz.EntityTypeIndustryChain, CanonicalName: "测试产业链",
			Aliases: []string{"别名"},
		}},
		Relationships: []biz.Relationship{{
			FromEntityID: "chain", ToEntityID: "node",
			Type: biz.RelationshipTypeHasNode, ChainID: "chain",
			RelationKey:     "industry_chain:test|has_node|chain_node:test",
			ContextualStage: "midstream", Position: &position, Mechanism: "正式成员",
		}},
	}

	nodes := projectionNodeRows(projection, biz.EntityTypeIndustryChain)
	if len(nodes) != 1 ||
		nodes[0]["projection_namespace"] != biz.Namespace ||
		nodes[0]["projection_contract_version"] != biz.ContractVersion ||
		nodes[0]["source_package_sha256"] != projection.PackageSHA256 {
		t.Fatalf("node rows = %#v", nodes)
	}

	relationships := projectionRelationshipRows(projection, biz.RelationshipTypeHasNode)
	if len(relationships) != 1 {
		t.Fatalf("relationship row count = %d, want 1", len(relationships))
	}
	properties, ok := relationships[0]["properties"].(map[string]any)
	if !ok ||
		properties["chain_id"] != "chain" ||
		properties["contextual_stage"] != "midstream" ||
		properties["position"] != 2 ||
		properties["projection_namespace"] != biz.Namespace {
		t.Fatalf("relationship properties = %#v", relationships[0]["properties"])
	}
}

func TestDeleteCypherIsScopedToNewProjectionNamespaceLabel(t *testing.T) {
	if !strings.Contains(deleteProjectionNodesQuery, ":TidewiseEntity") ||
		!strings.Contains(deleteProjectionNodesQuery, "projection_namespace: $namespace") ||
		strings.Contains(deleteProjectionNodesQuery, "MATCH (n) DETACH DELETE n") {
		t.Fatalf("unsafe delete query: %s", deleteProjectionNodesQuery)
	}
	if !strings.Contains(deleteProjectionMetadataQuery, ":TidewiseProjection") ||
		!strings.Contains(deleteProjectionMetadataQuery, "projection_namespace: $namespace") {
		t.Fatalf("unsafe metadata delete query: %s", deleteProjectionMetadataQuery)
	}
}
