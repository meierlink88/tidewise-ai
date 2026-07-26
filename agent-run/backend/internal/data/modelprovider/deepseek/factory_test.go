package deepseek

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

func TestFactoryGenerateReadsProviderResponseBeforeExecutionCancel(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/chat/completions" {
			t.Fatalf("provider request = %s %s, want POST /chat/completions", request.Method, request.URL.Path)
		}
		response.Header().Set("Content-Type", "application/json")
		response.(http.Flusher).Flush()
		time.Sleep(25 * time.Millisecond)
		_, _ = response.Write([]byte(`{
			"id":"chatcmpl-test",
			"object":"chat.completion",
			"created":1720000000,
			"model":"deepseek-chat",
			"choices":[{
				"index":0,
				"message":{"role":"assistant","content":"{\"ok\":true}"},
				"finish_reason":"stop"
			}],
			"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
		}`))
	}))
	defer provider.Close()

	executionContext, cancelExecution := context.WithCancel(context.Background())
	defer cancelExecution()
	chatModel, err := (Factory{Timeout: time.Second}).New(executionContext, agentrun.ModelProviderConfig{
		ProviderKey: "deepseek",
		BaseURL:     provider.URL,
		Model:       "deepseek-chat",
		APIKey:      "test-key",
	})
	if err != nil {
		t.Fatalf("create DeepSeek model: %v", err)
	}

	result, err := chatModel.Generate(executionContext, []*schema.Message{
		schema.UserMessage("return JSON"),
	})
	if err != nil {
		t.Fatalf("generate DeepSeek response: %v", err)
	}
	if result == nil || result.Content != `{"ok":true}` {
		t.Fatalf("generated response = %#v, want JSON body", result)
	}
}

func TestExecutionBoundClientCancelsProviderRequestWithExecution(t *testing.T) {
	executionContext, cancelExecution := context.WithCancel(context.Background())
	requestStarted := make(chan struct{})
	client := executionBoundClient(executionContext, 10*time.Second, roundTripperFunc(
		func(request *http.Request) (*http.Response, error) {
			close(requestStarted)
			<-request.Context().Done()
			return nil, request.Context().Err()
		},
	))
	result := make(chan error, 1)
	go func() {
		request, _ := http.NewRequest(http.MethodPost, "https://deepseek.test/chat/completions", nil)
		_, err := client.Do(request)
		result <- err
	}()
	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		t.Fatal("Provider request did not start")
	}
	cancelExecution()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Provider request error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Provider request was not canceled with the Execution")
	}
}
