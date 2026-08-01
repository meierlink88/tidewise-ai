package embeddingopenai

import (
	"context"
	"testing"
	"time"
)

func TestNewValidatesAndConstructsOfficialEmbedder(t *testing.T) {
	if value, err := New(context.Background(), Config{}); err == nil || value != nil {
		t.Fatalf("invalid config returned value=%#v err=%v", value, err)
	}
	value, err := New(context.Background(), Config{
		BaseURL: "https://embedding.example/v1", APIKey: "test-key",
		Model: "text-embedding-v4", Dimensions: 1024, Timeout: time.Second,
	})
	if err != nil || value == nil {
		t.Fatalf("valid config returned value=%#v err=%v", value, err)
	}
}
