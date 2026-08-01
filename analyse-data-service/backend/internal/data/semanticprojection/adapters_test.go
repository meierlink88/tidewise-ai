package semanticprojection

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	projection "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/semanticprojection"
)

func TestPostgresSourceRejectsMidStreamIterationFailure(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, entity_type, layer_code, name, canonical_name, array_to_json(aliases)::text, status")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "entity_type", "layer_code", "name", "canonical_name", "aliases", "status"}).
			AddRow("33333333-3333-4333-8333-333333333333", "company", "micro", "NVIDIA", "英伟达", `[]`, "active").
			RowError(0, errors.New("stream interrupted")))
	source, err := NewPostgresSource(database)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.Current(context.Background()); err == nil {
		t.Fatal("PostgresSource accepted a truncated Entity snapshot")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenAIEmbedderUsesFrozenDashScopeCompatibleContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/compatible-mode/v1/embeddings" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer runtime-secret" {
			t.Fatalf("authorization header was not injected")
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != projection.EmbeddingModel ||
			int(body["dimensions"].(float64)) != projection.VectorSize ||
			body["encoding_format"] != "float" {
			t.Fatalf("embedding request = %#v", body)
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{"data": []any{
			map[string]any{"index": 0, "embedding": make([]float32, projection.VectorSize)},
		}})
	}))
	defer server.Close()

	embedder, err := NewOpenAIEmbedder(HTTPConfig{
		Endpoint: server.URL + "/compatible-mode/v1", APIKey: "runtime-secret",
		Timeout: time.Second, MaxResponseBytes: 1 << 20, BatchSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	vectors, err := embedder.Embed(context.Background(), []string{"英伟达"})
	if err != nil {
		t.Fatal(err)
	}
	if len(vectors) != 1 || len(vectors[0]) != projection.VectorSize {
		t.Fatalf("vectors = %dx%d", len(vectors), len(vectors[0]))
	}
}

func TestDecodePostgresProjectionArraysFromJSON(t *testing.T) {
	var values []string
	if err := decodeStringArray(`["英伟达","NVIDIA"]`, &values); err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0] != "英伟达" || values[1] != "NVIDIA" {
		t.Fatalf("values = %#v", values)
	}
	for _, invalid := range []string{"null", `{"value":"NVIDIA"}`, "{NVIDIA}"} {
		if err := decodeStringArray(invalid, &values); err == nil {
			t.Fatalf("decodeStringArray accepted %q", invalid)
		}
	}
}
