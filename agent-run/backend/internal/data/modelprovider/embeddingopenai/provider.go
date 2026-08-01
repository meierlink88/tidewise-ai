package embeddingopenai

import (
	"context"
	"errors"
	"time"

	openaiembedding "github.com/cloudwego/eino-ext/components/embedding/openai"
	"github.com/cloudwego/eino/components/embedding"
)

type Config struct {
	BaseURL    string
	APIKey     string
	Model      string
	Dimensions int
	Timeout    time.Duration
}

// New constructs the official Eino OpenAI-compatible Embedder. The provider
// package owns concrete eino-ext configuration while callers depend only on
// Eino's embedding.Embedder interface.
func New(ctx context.Context, config Config) (embedding.Embedder, error) {
	if config.BaseURL == "" || config.APIKey == "" || config.Model == "" ||
		config.Dimensions <= 0 || config.Timeout <= 0 {
		return nil, errors.New("embedding provider configuration is invalid")
	}
	encoding := openaiembedding.EmbeddingEncodingFormatFloat
	return openaiembedding.NewEmbedder(ctx, &openaiembedding.EmbeddingConfig{
		BaseURL: config.BaseURL, APIKey: config.APIKey, Model: config.Model,
		Dimensions: &config.Dimensions, EncodingFormat: &encoding, Timeout: config.Timeout,
	})
}
