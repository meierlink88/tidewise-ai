package semanticretrieval

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
)

type recordingEmbedder struct {
	calls int
	texts [][]string
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func (e *recordingEmbedder) EmbedStrings(_ context.Context, texts []string, _ ...embedding.Option) ([][]float64, error) {
	e.calls++
	e.texts = append(e.texts, append([]string(nil), texts...))
	result := make([][]float64, len(texts))
	for index := range result {
		result[index] = make([]float64, VectorSize)
	}
	return result, nil
}

func TestClientUsesOneExactAndOneVectorBatchPerEvent(t *testing.T) {
	var exactCalls, vectorCalls int
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(request.URL.Path, "/points/scroll"):
			exactCalls++
			var body map[string]any
			_ = json.NewDecoder(request.Body).Decode(&body)
			filter := body["filter"].(map[string]any)
			if _, exists := filter["should"]; exists || len(filter["must"].([]any)) != 2 {
				t.Fatalf("exact filter = %#v", filter)
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"result": map[string]any{"points": []any{
				map[string]any{"id": "33333333-3333-4333-8333-333333333333", "payload": entityPayload("33333333-3333-4333-8333-333333333333", "英伟达", []string{"英伟达"})},
			}}})
		case strings.HasSuffix(request.URL.Path, "/points/query/batch"):
			vectorCalls++
			var body map[string]any
			_ = json.NewDecoder(request.Body).Decode(&body)
			if len(body["searches"].([]any)) != 2 {
				t.Fatalf("query batch = %#v", body)
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"result": []any{
				map[string]any{"points": []any{
					map[string]any{"id": "44444444-4444-4444-8444-444444444444", "score": 0.82, "payload": entityPayload("44444444-4444-4444-8444-444444444444", "安靠科技", []string{"安靠科技"})},
				}},
				map[string]any{"points": []any{}},
			}})
		default:
			t.Fatalf("unexpected path %s", request.URL.Path)
		}
	}))
	defer server.Close()
	embedder := &recordingEmbedder{}
	client, err := New(Config{
		QdrantURL: server.URL, Embedder: embedder,
		EntityCollection: EntityCollection, VectorSize: VectorSize, Timeout: time.Second,
		MaxResponseBytes: 1 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	lookups := []eventsemantic.EntityLookup{
		{CandidateKey: "nvidia", Mention: "英伟达"},
		{CandidateKey: "amkor", Mention: "安靠科技"},
		{CandidateKey: "capacity", Mention: "第三方封测产能"},
	}
	exact, err := client.ExactEntities(context.Background(), lookups)
	if err != nil {
		t.Fatal(err)
	}
	if len(exact[0].Candidates) != 1 || len(exact[1].Candidates) != 0 || len(exact[2].Candidates) != 0 {
		t.Fatalf("exact = %#v", exact)
	}
	vector, err := client.SearchEntities(context.Background(), lookups[1:], 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(vector) != 2 || len(vector[0].Candidates) != 1 || len(vector[1].Candidates) != 0 {
		t.Fatalf("vector = %#v", vector)
	}
	if exactCalls != 1 || embedder.calls != 1 || vectorCalls != 1 || len(embedder.texts[0]) != 2 {
		t.Fatalf("calls exact=%d embedding=%d vector=%d texts=%v", exactCalls, embedder.calls, vectorCalls, embedder.texts)
	}
}

func entityPayload(id, name string, normalized []string) map[string]any {
	return map[string]any{
		"entity_id": id, "entity_type": "company", "name": name, "canonical_name": name,
		"aliases": []string{name}, "normalized_names": normalized, "description": "", "status": "active",
		"source_identity": id, "projection_version": ProjectionVersion, "embedding_model": EmbeddingModel,
		"content_fingerprint": strings.Repeat("a", 64),
	}
}

func TestNormalizeNameIsSharedExactKeyContract(t *testing.T) {
	if got := NormalizeName(" NVIDIA（英伟达） "); got != "nvidia英伟达" {
		t.Fatalf("NormalizeName = %q", got)
	}
}

func TestClientRejectsMalformedQdrantCandidateIdentity(t *testing.T) {
	tests := []struct {
		name   string
		exact  bool
		point  map[string]any
		lookup eventsemantic.EntityLookup
	}{
		{
			name: "payload entity ID is not a UUID",
			point: map[string]any{
				"id": "33333333-3333-4333-8333-333333333333", "score": 0.9,
				"payload": entityPayload("garbage", "英伟达", []string{"英伟达"}),
			},
			lookup: eventsemantic.EntityLookup{CandidateKey: "nvidia", Mention: "英伟达"},
		},
		{
			name:  "point and payload IDs differ",
			exact: true,
			point: map[string]any{
				"id":      "33333333-3333-4333-8333-333333333333",
				"payload": entityPayload("44444444-4444-4444-8444-444444444444", "英伟达", []string{"英伟达"}),
			},
			lookup: eventsemantic.EntityLookup{CandidateKey: "nvidia", Mention: "英伟达"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				if test.exact {
					_ = json.NewEncoder(writer).Encode(map[string]any{"result": map[string]any{"points": []any{test.point}}})
					return
				}
				_ = json.NewEncoder(writer).Encode(map[string]any{"result": []any{map[string]any{"points": []any{test.point}}}})
			}))
			defer server.Close()
			client, err := New(Config{
				QdrantURL: server.URL, Embedder: &recordingEmbedder{}, EntityCollection: EntityCollection,
				VectorSize: VectorSize, Timeout: time.Second, MaxResponseBytes: 1 << 20,
			})
			if err != nil {
				t.Fatal(err)
			}
			if test.exact {
				_, err = client.ExactEntities(context.Background(), []eventsemantic.EntityLookup{test.lookup})
			} else {
				_, err = client.SearchEntities(context.Background(), []eventsemantic.EntityLookup{test.lookup}, 5)
			}
			var remote *eventsemantic.RemoteError
			if !errors.As(err, &remote) || remote.Code != "qdrant_response_invalid" || remote.Retryable {
				t.Fatalf("err = %#v", err)
			}
		})
	}
}

func TestClientRejectsQdrantPointWithInvalidProjectionProvenance(t *testing.T) {
	entityID := "33333333-3333-4333-8333-333333333333"
	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "missing source identity", mutate: func(payload map[string]any) { delete(payload, "source_identity") }},
		{name: "foreign source identity", mutate: func(payload map[string]any) { payload["source_identity"] = "44444444-4444-4444-8444-444444444444" }},
		{name: "missing projection version", mutate: func(payload map[string]any) { delete(payload, "projection_version") }},
		{name: "stale projection version", mutate: func(payload map[string]any) { payload["projection_version"] = "event-semantic-projection.v0" }},
		{name: "wrong embedding model", mutate: func(payload map[string]any) { payload["embedding_model"] = "other-model" }},
		{name: "missing fingerprint", mutate: func(payload map[string]any) { delete(payload, "content_fingerprint") }},
		{name: "invalid fingerprint", mutate: func(payload map[string]any) { payload["content_fingerprint"] = strings.Repeat("G", 64) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload := entityPayload(entityID, "英伟达", []string{"英伟达"})
			test.mutate(payload)
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(writer).Encode(map[string]any{"result": map[string]any{"points": []any{
					map[string]any{"id": entityID, "payload": payload},
				}}})
			}))
			defer server.Close()
			client, err := New(Config{
				QdrantURL: server.URL, Embedder: &recordingEmbedder{}, EntityCollection: EntityCollection,
				VectorSize: VectorSize, Timeout: time.Second, MaxResponseBytes: 1 << 20,
			})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.ExactEntities(context.Background(), []eventsemantic.EntityLookup{{CandidateKey: "nvidia", Mention: "英伟达"}})
			var remote *eventsemantic.RemoteError
			if !errors.As(err, &remote) || remote.Code != "qdrant_response_invalid" || remote.Retryable {
				t.Fatalf("err = %#v", err)
			}
		})
	}
}

func TestClientRejectsMalformedSuccessfulQdrantEnvelope(t *testing.T) {
	tests := []struct {
		name, body, code string
	}{
		{name: "missing result points", body: `{"status":"ok"}`, code: "qdrant_response_invalid"},
		{name: "trailing JSON value", body: `{"result":{"points":[]}} {}`, code: "semantic_retrieval_response_invalid"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()
			client, err := New(Config{
				QdrantURL: server.URL, Embedder: &recordingEmbedder{}, EntityCollection: EntityCollection,
				VectorSize: VectorSize, Timeout: time.Second, MaxResponseBytes: 1 << 20,
			})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.ExactEntities(context.Background(), []eventsemantic.EntityLookup{{
				CandidateKey: "nvidia", Mention: "英伟达",
			}})
			var remote *eventsemantic.RemoteError
			if !errors.As(err, &remote) || remote.Code != test.code || remote.Retryable {
				t.Fatalf("err = %#v", err)
			}
		})
	}
}

func TestClientPreservesHTTPContextCancellationAndDeadline(t *testing.T) {
	tests := []struct {
		name string
		ctx  func() (context.Context, context.CancelFunc)
		want error
	}{
		{name: "canceled", ctx: func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx, func() {}
		}, want: context.Canceled},
		{name: "deadline", ctx: func() (context.Context, context.CancelFunc) {
			return context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		}, want: context.DeadlineExceeded},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, err := New(Config{
				QdrantURL: "http://qdrant.invalid", Embedder: &recordingEmbedder{}, EntityCollection: EntityCollection,
				VectorSize: VectorSize, Timeout: time.Second, MaxResponseBytes: 1 << 20,
				HTTPClient: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
					return nil, request.Context().Err()
				})},
			})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := test.ctx()
			defer cancel()
			_, err = client.ExactEntities(ctx, []eventsemantic.EntityLookup{{CandidateKey: "entity", Mention: "实体"}})
			if !errors.Is(err, test.want) {
				t.Fatalf("err=%#v want=%v", err, test.want)
			}
			var remote *eventsemantic.RemoteError
			if errors.As(err, &remote) {
				t.Fatalf("context error was wrapped as RemoteError: %#v", remote)
			}
		})
	}
}
