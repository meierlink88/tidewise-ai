package deepseek

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

func TestFactoryGenerateExplicitlyDisablesThinking(t *testing.T) {
	requestBody := make(chan map[string]any, 1)
	provider := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Errorf("decode provider request: %v", err)
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		requestBody <- body
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{
			"id":"chatcmpl-thinking-disabled",
			"object":"chat.completion",
			"created":1720000000,
			"model":"deepseek-v4-pro",
			"choices":[{
				"index":0,
				"message":{"role":"assistant","content":"{\"ok\":true}"},
				"finish_reason":"stop"
			}],
			"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
		}`))
	}))
	defer provider.Close()

	chatModel, err := (Factory{Timeout: time.Second}).New(context.Background(), agentrun.ModelProviderConfig{
		ProviderKey: "deepseek",
		BaseURL:     provider.URL,
		Model:       "deepseek-v4-pro",
		APIKey:      "test-key",
	})
	if err != nil {
		t.Fatalf("create DeepSeek model: %v", err)
	}
	if _, err := chatModel.Generate(context.Background(), []*schema.Message{
		schema.UserMessage("return JSON"),
	}); err != nil {
		t.Fatalf("generate DeepSeek response: %v", err)
	}

	body := <-requestBody
	thinking, ok := body["thinking"].(map[string]any)
	if !ok {
		t.Fatalf("provider request thinking = %#v, want object", body["thinking"])
	}
	if thinking["type"] != "disabled" {
		t.Fatalf("provider request thinking.type = %#v, want disabled", thinking["type"])
	}
}

func TestToolCallingFactoryUsesOfficialForcedFunctionProtocolWithoutJSONObjectMode(t *testing.T) {
	requestBody := make(chan map[string]any, 1)
	provider := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Errorf("decode provider request: %v", err)
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		requestBody <- body
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{
			"id":"chatcmpl-tool-call",
			"object":"chat.completion",
			"created":1720000000,
			"model":"deepseek-chat",
			"choices":[{
				"index":0,
				"message":{"role":"assistant","content":null,"tool_calls":[{
					"id":"call-1","type":"function",
					"function":{"name":"submit_event_candidates","arguments":"{\"documents\":[]}"}
				}]},
				"finish_reason":"tool_calls"
			}],
			"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
		}`))
	}))
	defer provider.Close()

	base, err := (Factory{Timeout: time.Second}).NewToolCalling(
		context.Background(),
		agentrun.ModelProviderConfig{
			ProviderKey: "deepseek", BaseURL: provider.URL,
			Model: "deepseek-chat", APIKey: "test-key",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	bound, err := base.WithTools([]*schema.ToolInfo{{
		Name: "submit_event_candidates", Desc: "submit results",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"documents": {Type: schema.Array, Required: true},
		}),
	}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := bound.Generate(
		context.Background(),
		[]*schema.Message{schema.UserMessage("extract")},
		model.WithToolChoice(schema.ToolChoiceForced, "submit_event_candidates"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.ToolCalls) != 1 || result.ToolCalls[0].Function.Name != "submit_event_candidates" ||
		result.ToolCalls[0].Function.Arguments != `{"documents":[]}` {
		t.Fatalf("tool response = %#v", result.ToolCalls)
	}
	body := <-requestBody
	if _, exists := body["response_format"]; exists {
		t.Fatalf("Function Calling request also set response_format: %#v", body["response_format"])
	}
	if tools, ok := body["tools"].([]any); !ok || len(tools) != 1 {
		t.Fatalf("provider tools = %#v", body["tools"])
	}
	if _, exists := body["tool_choice"]; !exists {
		t.Fatalf("provider tool_choice = %#v", body["tool_choice"])
	}
}

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
