package semanticretrieval

import (
	"context"
	"os"
	"testing"
	"time"

	openaiembedding "github.com/cloudwego/eino-ext/components/embedding/openai"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
)

func TestDashScopeQdrantFixedEventRetrieval(t *testing.T) {
	if os.Getenv("TIDEWISE_EVENT_SEMANTIC_RETRIEVAL_E2E") != "1" {
		t.Skip("set TIDEWISE_EVENT_SEMANTIC_RETRIEVAL_E2E=1 for the local DashScope/Qdrant seam")
	}
	apiKey := os.Getenv("EMBEDDING_API_KEY")
	if apiKey == "" {
		t.Fatal("EMBEDDING_API_KEY is required")
	}
	dimensions := VectorSize
	encoding := openaiembedding.EmbeddingEncodingFormatFloat
	embedder, err := openaiembedding.NewEmbedder(context.Background(), &openaiembedding.EmbeddingConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", APIKey: apiKey,
		Model: EmbeddingModel, Dimensions: &dimensions, EncodingFormat: &encoding,
		Timeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	client, err := New(Config{
		QdrantURL: "http://127.0.0.1:6333", Embedder: embedder,
		EntityCollection: EntityCollection, VectorSize: VectorSize,
		Timeout: 30 * time.Second, MaxResponseBytes: 8 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	lookups := []eventsemantic.EntityLookup{
		{CandidateKey: "nvidia", Mention: "英伟达"},
		{CandidateKey: "amkor", Mention: "安靠科技"},
		{CandidateKey: "packaging", Mention: "第三方封测产能"},
	}
	exact, err := client.ExactEntities(context.Background(), lookups)
	if err != nil {
		t.Fatal(err)
	}
	if len(exact) != 3 || len(exact[0].Candidates) != 1 ||
		exact[0].Candidates[0].Entity.CanonicalName != "英伟达" || len(exact[1].Candidates) != 0 {
		t.Fatalf("exact candidates = %#v", exact)
	}
	fallback, err := client.SearchEntities(context.Background(), lookups[1:], 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(fallback) != 2 {
		t.Fatalf("fallback = %#v", fallback)
	}
	for _, set := range fallback {
		for _, candidate := range set.Candidates {
			if candidate.Entity.EntityID == "" || candidate.Entity.Status != "active" {
				t.Fatalf("invalid formal candidate = %#v", candidate)
			}
		}
	}
	t.Logf("fixed Event exact=%#v fallback=%#v", exact, fallback)
}
