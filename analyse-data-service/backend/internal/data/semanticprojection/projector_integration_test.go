package semanticprojection

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	projection "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/semanticprojection"
)

type integrationEmbedder struct{}

func (integrationEmbedder) Embed(_ context.Context, documents []string) ([][]float32, error) {
	result := make([][]float32, len(documents))
	for index := range documents {
		result[index] = make([]float32, projection.VectorSize)
		result[index][0] = float32(index + 1)
	}
	return result, nil
}

func TestPostgresToQdrantProjectorRealRebuildIsCurrentIdempotentAndRemovesStalePoints(t *testing.T) {
	if os.Getenv("TIDEWISE_EVENT_SEMANTIC_PROJECTOR_E2E") != "1" {
		t.Skip("set TIDEWISE_EVENT_SEMANTIC_PROJECTOR_E2E=1 for the local PostgreSQL/Qdrant seam")
	}
	databaseURL := os.Getenv("TIDEWISE_TEST_DATABASE_URL")
	qdrantURL := os.Getenv("QDRANT_URL")
	parsed, err := url.Parse(databaseURL)
	if err != nil || (parsed.Hostname() != "localhost" && parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "::1") {
		t.Fatal("projector integration database must be an explicit loopback URL")
	}
	if qdrantURL != "http://127.0.0.1:6333" && qdrantURL != "http://localhost:6333" {
		t.Fatal("projector integration Qdrant must be the explicit local endpoint")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	admin, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("tw_semantic_projection_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = admin.ExecContext(context.Background(), `DROP SCHEMA IF EXISTS `+schema+` CASCADE`) }()
	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	config.RuntimeParams["search_path"] = schema
	database := stdlib.OpenDB(*config)
	defer database.Close()
	if _, err := database.ExecContext(ctx, `
		CREATE TABLE entity_nodes(
			id uuid PRIMARY KEY, entity_type text NOT NULL, layer_code text NOT NULL,
			name text NOT NULL, canonical_name text NOT NULL, aliases text[] NOT NULL, status text NOT NULL
		);
		CREATE TABLE variable_definitions(
			variable_key text NOT NULL, version integer NOT NULL, name_zh text NOT NULL, name_en text NOT NULL,
			business_definition text NOT NULL, domain text NOT NULL, value_type text NOT NULL,
			allowed_units text[] NOT NULL, allowed_directions text[] NOT NULL, status text NOT NULL,
			PRIMARY KEY(variable_key, version)
		);
		CREATE TABLE variable_definition_entity_types(
			variable_key text NOT NULL, variable_version integer NOT NULL, entity_type text NOT NULL
		);
		INSERT INTO entity_nodes VALUES
		('11111111-1111-4111-8111-111111111111','company','micro','NVIDIA','英伟达',ARRAY['NVIDIA'],'active'),
		('22222222-2222-4222-8222-222222222222','company','micro','Amkor','安靠科技',ARRAY['Amkor'],'active'),
		('33333333-3333-4333-8333-333333333333','company','micro','Inactive','Inactive',ARRAY[]::text[],'inactive');
		INSERT INTO variable_definitions VALUES
		('revenue',1,'收入旧版','Revenue old','旧定义','finance','narrative',ARRAY[]::text[],ARRAY['increase'],'active'),
		('revenue',2,'收入','Revenue','当前定义','finance','narrative',ARRAY['CNY'],ARRAY['increase','decrease'],'active'),
		('inactive_variable',1,'停用','Inactive','停用定义','finance','narrative',ARRAY[]::text[],ARRAY['increase'],'deprecated');
		INSERT INTO variable_definition_entity_types VALUES ('revenue',2,'company');
	`); err != nil {
		t.Fatal(err)
	}
	source, err := NewPostgresSource(database)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewQdrantStore(HTTPConfig{
		Endpoint: qdrantURL, Timeout: 10 * time.Second, MaxResponseBytes: 8 << 20, BatchSize: 16,
	})
	if err != nil {
		t.Fatal(err)
	}
	suffix := fmt.Sprintf("_%d", time.Now().UnixNano())
	store.entityCollection = "test_entity_semantic" + suffix
	store.variableCollection = "test_variable_semantic" + suffix
	defer deleteTestCollection(store, store.entityCollection)
	defer deleteTestCollection(store, store.variableCollection)
	service, err := projection.New(source, integrationEmbedder{}, store)
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.Rebuild(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if first.EntityCount != 2 || first.VariableCount != 1 {
		t.Fatalf("first rebuild = %#v", first)
	}
	variablePoints := testVariableCollectionPoints(t, store)
	if len(variablePoints) != 1 || variablePoints[0].Payload.SourceIdentity != "revenue@2" ||
		variablePoints[0].Payload.VariableKey != "revenue" || variablePoints[0].Payload.VariableVersion != 2 ||
		variablePoints[0].Payload.ProjectionVersion != projection.ProjectionVersion ||
		variablePoints[0].Payload.EmbeddingModel != projection.EmbeddingModel {
		t.Fatalf("current Variable Definition points = %#v", variablePoints)
	}
	firstIDs := testCollectionPointIDs(t, store, store.entityCollection)
	if len(firstIDs) != 2 || firstIDs[0] != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("first Entity point IDs = %#v", firstIDs)
	}
	if _, err := database.ExecContext(ctx, `UPDATE entity_nodes SET status='inactive' WHERE id='22222222-2222-4222-8222-222222222222'`); err != nil {
		t.Fatal(err)
	}
	second, err := service.Rebuild(ctx)
	if err != nil {
		t.Fatal(err)
	}
	secondIDs := testCollectionPointIDs(t, store, store.entityCollection)
	if second.EntityCount != 1 || second.VariableCount != 1 || len(secondIDs) != 1 || secondIDs[0] != firstIDs[0] {
		t.Fatalf("second rebuild=%#v Entity IDs=%#v", second, secondIDs)
	}
	var info struct {
		Result struct {
			Status string `json:"status"`
			Config struct {
				Params struct {
					Vectors struct {
						Size     int    `json:"size"`
						Distance string `json:"distance"`
					} `json:"vectors"`
				} `json:"params"`
			} `json:"config"`
		} `json:"result"`
	}
	if err := requestJSON(ctx, store.http, http.MethodGet, store.endpoint+"/collections/"+store.entityCollection, "", store.maxResponseBytes, nil, &info); err != nil {
		t.Fatal(err)
	}
	if info.Result.Status != "green" || info.Result.Config.Params.Vectors.Size != projection.VectorSize ||
		info.Result.Config.Params.Vectors.Distance != "Cosine" {
		t.Fatalf("Qdrant collection info = %#v", info)
	}
}

type testVariablePoint struct {
	ID      string `json:"id"`
	Payload struct {
		SourceIdentity    string `json:"source_identity"`
		VariableKey       string `json:"variable_key"`
		VariableVersion   int    `json:"variable_version"`
		ProjectionVersion string `json:"projection_version"`
		EmbeddingModel    string `json:"embedding_model"`
	} `json:"payload"`
}

func testVariableCollectionPoints(t *testing.T, store *QdrantStore) []testVariablePoint {
	t.Helper()
	var response struct {
		Result struct {
			Points []testVariablePoint `json:"points"`
		} `json:"result"`
	}
	payload := []byte(`{"limit":100,"with_payload":true,"with_vector":false}`)
	if err := requestJSON(context.Background(), store.http, http.MethodPost,
		store.endpoint+"/collections/"+store.variableCollection+"/points/scroll", "",
		store.maxResponseBytes, payload, &response); err != nil {
		t.Fatal(err)
	}
	return response.Result.Points
}

func testCollectionPointIDs(t *testing.T, store *QdrantStore, collection string) []string {
	t.Helper()
	var response struct {
		Result struct {
			Points []struct {
				ID string `json:"id"`
			} `json:"points"`
		} `json:"result"`
	}
	payload := []byte(`{"limit":100,"with_payload":false,"with_vector":false}`)
	if err := requestJSON(context.Background(), store.http, http.MethodPost,
		store.endpoint+"/collections/"+collection+"/points/scroll", "", store.maxResponseBytes, payload, &response); err != nil {
		t.Fatal(err)
	}
	result := make([]string, 0, len(response.Result.Points))
	for _, point := range response.Result.Points {
		result = append(result, point.ID)
	}
	return result
}

func deleteTestCollection(store *QdrantStore, collection string) {
	_ = requestJSON(context.Background(), store.http, http.MethodDelete,
		store.endpoint+"/collections/"+collection, "", store.maxResponseBytes, nil, nil, http.StatusNotFound)
}
