package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/industryrelationshipimport"
)

func TestVerifyPersistedIndustryRelationshipContentDecodesTextArrays(t *testing.T) {
	db := openIndustryRelationshipImportTestDatabase(t)
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = tx.Rollback()
	})

	item := biz.IndustryChain{
		EntityID:                  "11111111-1111-4111-8111-111111111111",
		EntityKey:                 "industry_chain:test_array_round_trip",
		EntityType:                "industry_chain",
		LayerCode:                 "industry_chain",
		Name:                      "数组往返测试产业链",
		CanonicalName:             "数组往返测试产业链",
		Aliases:                   []string{"别名一", "alias,two"},
		Status:                    "active",
		Scope:                     "验证 PostgreSQL TEXT[] 的扫描边界",
		TargetOutput:              "可验证数组",
		EndUse:                    "仓库回归测试",
		TechnologyRouteQualifier:  stringPointer("array-json"),
		ObservableVariables:       []string{"订单", "良率,分位"},
		Geography:                 "test",
		AsOfDate:                  "2026-07-27",
		ReviewStatus:              "approved",
		ReviewNote:                "真实 PostgreSQL TEXT[] 必须在写后内容校验中无损往返。",
		RelationshipApprovalBasis: "user_explicit_delegated_review",
	}
	verifiedAt := time.Date(2026, 7, 27, 1, 2, 3, 0, time.UTC)
	nodeA := biz.ChainNodeAddition{
		EntityID:      "22222222-2222-4222-8222-222222222222",
		EntityKey:     "chain_node:v2_array_a",
		EntityType:    "chain_node",
		LayerCode:     "industry_chain",
		Name:          "数组节点A",
		CanonicalName: "数组节点A",
		Aliases:       []string{"节点A", "node,a"},
		Status:        "active",
		Definition:    "测试节点A",
		BoundaryNote:  "仅用于事务内数组往返测试。",
		ReviewStatus:  "approved",
	}
	nodeB := nodeA
	nodeB.EntityID = "33333333-3333-4333-8333-333333333333"
	nodeB.EntityKey = "chain_node:v2_array_b"
	nodeB.Name = "数组节点B"
	nodeB.CanonicalName = "数组节点B"
	nodeB.Aliases = []string{"节点B", "node,b"}
	nodeB.Definition = "测试节点B"
	membership := biz.Membership{
		RelationKey:           "test-membership",
		IndustryChainEntityID: item.EntityID,
		ChainNodeEntityID:     nodeA.EntityID,
		ContextualStage:       "upstream",
		Position:              1,
		InclusionReason:       "测试 TEXT[] evidence_ids 往返。",
		EvidenceIDs:           []string{"evidence:a", "evidence:comma,value"},
		SourceName:            "integration-test",
		SourceURL:             "artifact://integration-test",
		VerifiedAt:            verifiedAt,
		ReviewStatus:          "approved",
		Status:                "active",
	}
	graphEdge := biz.GraphEdge{
		ID:                    "44444444-4444-4444-8444-444444444444",
		RelationKey:           "test-graph-edge",
		IndustryChainEntityID: item.EntityID,
		FromChainNodeEntityID: nodeA.EntityID,
		RelationType:          "input_to",
		ToChainNodeEntityID:   nodeB.EntityID,
		Mechanism:             "测试数组扫描。",
		SegmentKind:           "direct_candidate",
		EvidenceIDs:           []string{"evidence:edge", "evidence:edge,comma"},
		SourceName:            "integration-test",
		SourceURL:             "artifact://integration-test",
		VerifiedAt:            verifiedAt,
		ReviewStatus:          "approved",
		Status:                "active",
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO entity_nodes (
	    id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status
	) VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8)`,
		item.EntityID, item.EntityKey, item.EntityType, item.LayerCode,
		item.Name, item.CanonicalName, item.Aliases, item.Status,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO industry_chain_definitions (
	    entity_id, scope, target_output, end_use, technology_route_qualifier,
	    observable_variables, geography, as_of_date, review_status, review_note
	) VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8::date,$9,$10)`,
		item.EntityID, item.Scope, item.TargetOutput, item.EndUse,
		item.TechnologyRouteQualifier, item.ObservableVariables, item.Geography,
		item.AsOfDate, item.ReviewStatus, item.ReviewNote,
	); err != nil {
		t.Fatal(err)
	}
	for _, node := range []biz.ChainNodeAddition{nodeA, nodeB} {
		if _, err := tx.ExecContext(ctx, `INSERT INTO entity_nodes (
		    id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status
		) VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8)`,
			node.EntityID, node.EntityKey, node.EntityType, node.LayerCode,
			node.Name, node.CanonicalName, node.Aliases, node.Status,
		); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO chain_node_profiles (
		    entity_id, definition, boundary_note, review_status
		) VALUES ($1::uuid,$2,$3,$4)`,
			node.EntityID, node.Definition, node.BoundaryNote, node.ReviewStatus,
		); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO industry_chain_node_memberships (
	    industry_chain_entity_id, chain_node_entity_id, position, contextual_stage,
	    review_status, status, inclusion_reason, evidence_ids, source_name, source_url, verified_at
	) VALUES ($1::uuid,$2::uuid,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		membership.IndustryChainEntityID, membership.ChainNodeEntityID,
		membership.Position, membership.ContextualStage, membership.ReviewStatus,
		membership.Status, membership.InclusionReason, membership.EvidenceIDs,
		membership.SourceName, membership.SourceURL, membership.VerifiedAt,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO industry_chain_graph_edges (
	    id, industry_chain_entity_id, from_chain_node_entity_id, to_chain_node_entity_id,
	    relation_type, mechanism, condition_note, segment_kind, omitted_step_note,
	    review_status, status, evidence_ids, source_name, source_url, verified_at
	) VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		graphEdge.ID, graphEdge.IndustryChainEntityID, graphEdge.FromChainNodeEntityID,
		graphEdge.ToChainNodeEntityID, graphEdge.RelationType, graphEdge.Mechanism,
		graphEdge.ConditionNote, graphEdge.SegmentKind, graphEdge.OmittedStepNote,
		graphEdge.ReviewStatus, graphEdge.Status, graphEdge.EvidenceIDs,
		graphEdge.SourceName, graphEdge.SourceURL, graphEdge.VerifiedAt,
	); err != nil {
		t.Fatal(err)
	}

	repositoryTx := &postgresIndustryRelationshipImportTx{tx: tx}
	if err := repositoryTx.verifyPersistedContent(
		ctx,
		biz.Package{
			IndustryChains:     []biz.IndustryChain{item},
			ChainNodeAdditions: []biz.ChainNodeAddition{nodeA, nodeB},
			Memberships:        []biz.Membership{membership},
			GraphEdges:         []biz.GraphEdge{graphEdge},
		},
	); err != nil {
		t.Fatalf("verifyPersistedContent() error = %v", err)
	}
}

func openIndustryRelationshipImportTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	databaseURL := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TIDEWISE_TEST_DATABASE_URL to run Industry relationship import integration tests")
	}
	admin, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	var databaseName string
	if err := admin.QueryRow(`SELECT current_database()`).Scan(&databaseName); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	if databaseName != "tidewise_local" {
		admin.Close()
		t.Fatalf("integration database = %q, want tidewise_local", databaseName)
	}
	schema := fmt.Sprintf("tw_industry_relationship_import_%d", time.Now().UnixNano())
	if _, err := admin.Exec(`CREATE SCHEMA ` + schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		admin.Close()
		t.Fatal(err)
	}
	config.RuntimeParams["search_path"] = schema
	db := stdlib.OpenDB(*config)
	if err := db.Ping(); err != nil {
		db.Close()
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		_, _ = admin.Exec(`DROP SCHEMA IF EXISTS ` + schema + ` CASCADE`)
		admin.Close()
	})
	prepareIndustryRelationshipArraySchema(t, db)
	return db
}

func prepareIndustryRelationshipArraySchema(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, statement := range []string{
		`CREATE TABLE entity_nodes (
		    id UUID PRIMARY KEY,
		    entity_key TEXT NOT NULL UNIQUE,
		    entity_type TEXT NOT NULL,
		    layer_code TEXT NOT NULL,
		    name TEXT NOT NULL,
		    canonical_name TEXT NOT NULL,
		    aliases TEXT[] NOT NULL,
		    status TEXT NOT NULL
		)`,
		`CREATE TABLE industry_chain_definitions (
		    entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id),
		    scope TEXT NOT NULL,
		    target_output TEXT NOT NULL,
		    end_use TEXT NOT NULL,
		    technology_route_qualifier TEXT,
		    observable_variables TEXT[] NOT NULL,
		    geography TEXT NOT NULL,
		    as_of_date DATE NOT NULL,
		    review_status TEXT NOT NULL,
		    review_note TEXT NOT NULL
		)`,
		`CREATE TABLE chain_node_profiles (
		    entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id),
		    definition TEXT NOT NULL,
		    boundary_note TEXT NOT NULL,
		    review_status TEXT NOT NULL
		)`,
		`CREATE TABLE industry_chain_node_memberships (
		    industry_chain_entity_id UUID NOT NULL REFERENCES entity_nodes(id),
		    chain_node_entity_id UUID NOT NULL REFERENCES entity_nodes(id),
		    position INTEGER NOT NULL,
		    contextual_stage TEXT NOT NULL,
		    review_status TEXT NOT NULL,
		    status TEXT NOT NULL,
		    inclusion_reason TEXT NOT NULL,
		    evidence_ids TEXT[] NOT NULL,
		    source_name TEXT NOT NULL,
		    source_url TEXT NOT NULL,
		    verified_at TIMESTAMPTZ NOT NULL,
		    PRIMARY KEY (industry_chain_entity_id, chain_node_entity_id)
		)`,
		`CREATE TABLE industry_chain_graph_edges (
		    id UUID PRIMARY KEY,
		    industry_chain_entity_id UUID NOT NULL REFERENCES entity_nodes(id),
		    from_chain_node_entity_id UUID NOT NULL REFERENCES entity_nodes(id),
		    to_chain_node_entity_id UUID NOT NULL REFERENCES entity_nodes(id),
		    relation_type TEXT NOT NULL,
		    mechanism TEXT NOT NULL,
		    condition_note TEXT,
		    segment_kind TEXT NOT NULL,
		    omitted_step_note TEXT,
		    review_status TEXT NOT NULL,
		    status TEXT NOT NULL,
		    evidence_ids TEXT[] NOT NULL,
		    source_name TEXT NOT NULL,
		    source_url TEXT NOT NULL,
		    verified_at TIMESTAMPTZ NOT NULL
		)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
}

func stringPointer(value string) *string {
	return &value
}
