package planning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/biz/agents/collector"
)

type Request = collector.Request

type fakeChatModel struct {
	generate func(context.Context, []*schema.Message) (*schema.Message, error)
}

func TestDeepSeekQueryPlannerPlansCombinedQueryAndOptionalTimeWindow(t *testing.T) {
	var captured []*schema.Message
	planner, err := NewDeepSeekQueryPlanner(fakeChatModel{generate: func(_ context.Context, messages []*schema.Message) (*schema.Message, error) {
		captured = messages
		return schema.AssistantMessage(`{"queries":["A股 半导体","供应链 价格"],"combined_query":"A股 半导体 供应链 价格","time_window_hours":168}`, nil), nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	prompt := "采集最近一周中国半导体产业链资讯。\n重点关注价格与供需。"
	input := &Request{RunID: "run-v1", Prompt: prompt, CollectedAt: time.Date(2026, 7, 22, 2, 0, 0, 0, time.UTC)}

	output, err := planner.Plan(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(output.SearchQueries, []string{"A股 半导体", "供应链 价格"}) {
		t.Fatalf("queries = %#v", output.SearchQueries)
	}
	if output.CombinedQuery != "A股 半导体 供应链 价格" || output.TimeWindowHours != 168 {
		t.Fatalf("planned request = %#v", output)
	}
	if len(captured) != 2 || captured[0].Role != schema.System || captured[1].Role != schema.User {
		t.Fatalf("messages = %#v", captured)
	}
	var userPayload struct {
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal([]byte(captured[1].Content), &userPayload); err != nil {
		t.Fatalf("decode user message: %v", err)
	}
	if strings.Contains(captured[0].Content, "半导体") || userPayload.Prompt != prompt {
		t.Fatalf("business prompt crossed message roles: %#v", captured)
	}

	planner, err = NewDeepSeekQueryPlanner(fakeChatModel{generate: func(context.Context, []*schema.Message) (*schema.Message, error) {
		return schema.AssistantMessage(`{"queries":["宏观政策"],"combined_query":"宏观政策"}`, nil), nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	output, err = planner.Plan(context.Background(), &Request{Prompt: "采集宏观政策", CollectedAt: input.CollectedAt})
	if err != nil {
		t.Fatal(err)
	}
	if output.TimeWindowHours != 48 {
		t.Fatalf("default time window = %d, want 48", output.TimeWindowHours)
	}
}

func (f fakeChatModel) Generate(ctx context.Context, input []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	return f.generate(ctx, input)
}

func (f fakeChatModel) Stream(context.Context, []*schema.Message, ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("unexpected stream call")
}

func TestDeepSeekQueryPlannerBuildsPromptAndReturnsIndependentRequest(t *testing.T) {
	var captured []*schema.Message
	planner, err := NewDeepSeekQueryPlanner(fakeChatModel{generate: func(_ context.Context, messages []*schema.Message) (*schema.Message, error) {
		captured = messages
		return schema.AssistantMessage(`{"queries":["same"," model query ","same"],"combined_query":"same model query"}`, nil), nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	originalQueries := []string{"legacy query"}
	const prompt = "prompt-owned intent\nwith exact whitespace\n"
	input := &Request{
		RunID: "run-1", Prompt: prompt, SearchQueries: originalQueries,
		CandidateLimit: 7, TimeWindowHours: 48,
		CollectedAt: time.Date(2026, 7, 18, 1, 2, 3, 0, time.FixedZone("CST", 8*60*60)),
	}

	output, err := planner.Plan(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if output == input {
		t.Fatal("planner returned the caller's Request pointer")
	}
	if !reflect.DeepEqual(input.SearchQueries, []string{"legacy query"}) {
		t.Fatalf("planner mutated input queries: %#v", input.SearchQueries)
	}
	wantQueries := []string{"same", "model query"}
	if !reflect.DeepEqual(output.SearchQueries, wantQueries) {
		t.Fatalf("queries = %#v, want %#v", output.SearchQueries, wantQueries)
	}
	if output.RunID != input.RunID || output.Prompt != input.Prompt || output.CandidateLimit != input.CandidateLimit || output.TimeWindowHours != input.TimeWindowHours || !output.CollectedAt.Equal(input.CollectedAt) {
		t.Fatalf("non-query fields changed: input=%+v output=%+v", input, output)
	}
	output.SearchQueries[0] = "changed"
	if input.SearchQueries[0] != "legacy query" || originalQueries[0] != "legacy query" {
		t.Fatal("output SearchQueries aliases caller input")
	}
	if len(captured) != 2 || captured[0].Role != schema.System || captured[1].Role != schema.User {
		t.Fatalf("unexpected messages: %#v", captured)
	}
	for _, value := range []string{"2026-07-17T17:02:03Z", "prompt-owned intent"} {
		if !strings.Contains(captured[1].Content, value) {
			t.Fatalf("user prompt missing %q: %s", value, captured[1].Content)
		}
	}
	if strings.Contains(captured[0].Content, prompt) || strings.Contains(captured[1].Content, "legacy query") {
		t.Fatalf("messages crossed Planner contract: %#v", captured)
	}
}

func TestDeepSeekQueryPlannerStrictJSONValidation(t *testing.T) {
	tests := map[string]string{
		"empty response":     "",
		"malformed":          `{"queries":[}`,
		"unknown field":      `{"queries":["q"],"combined_query":"q","facts":"not allowed"}`,
		"model objective":    `{"objective":"rewritten","queries":["q"],"combined_query":"q"}`,
		"wrong type":         `{"queries":"q","combined_query":"q"}`,
		"empty queries":      `{"queries":[],"combined_query":"q"}`,
		"empty query":        `{"queries":["  "],"combined_query":"q"}`,
		"empty combined":     `{"queries":["q"],"combined_query":" "}`,
		"invalid window":     `{"queries":["q"],"combined_query":"q","time_window_hours":0}`,
		"trailing object":    `{"queries":["q"],"combined_query":"q"}{"queries":["other"]}`,
		"trailing primitive": `{"queries":["q"],"combined_query":"q"} true`,
		"too long":           fmt.Sprintf(`{"queries":[%q],"combined_query":"q"}`, strings.Repeat("界", 257)),
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			planner, err := NewDeepSeekQueryPlanner(fakeChatModel{generate: func(context.Context, []*schema.Message) (*schema.Message, error) {
				return schema.AssistantMessage(content, nil), nil
			}})
			if err != nil {
				t.Fatal(err)
			}
			_, err = planner.Plan(context.Background(), &Request{Prompt: "system-prompt-secret"})
			if err == nil {
				t.Fatal("expected schema error")
			}
			if content != "" && strings.Contains(err.Error(), content) || strings.Contains(err.Error(), "system-prompt-secret") {
				t.Fatalf("error leaked raw content or prompt: %v", err)
			}
		})
	}
}

func TestDeepSeekQueryPlannerRejectsEmptyPromptBeforeModelCall(t *testing.T) {
	modelCalls := 0
	planner, err := NewDeepSeekQueryPlanner(fakeChatModel{generate: func(context.Context, []*schema.Message) (*schema.Message, error) {
		modelCalls++
		return schema.AssistantMessage(`{"queries":["q"],"combined_query":"q"}`, nil), nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = planner.Plan(context.Background(), &Request{Prompt: " \n\t"})
	if !errors.Is(err, ErrQueryPlanningSchema) || modelCalls != 0 {
		t.Fatalf("error=%v modelCalls=%d", err, modelCalls)
	}
}

func TestDeepSeekQueryPlannerUsesOnlyModelQueriesAndLimitsQueries(t *testing.T) {
	planner, err := NewDeepSeekQueryPlanner(fakeChatModel{generate: func(context.Context, []*schema.Message) (*schema.Message, error) {
		return schema.AssistantMessage(`{"queries":["model-a"," model-a ","model-b"],"combined_query":"model-a model-b"}`, nil), nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	output, err := planner.Plan(context.Background(), &Request{Prompt: "intent", SearchQueries: []string{"legacy-query"}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(output.SearchQueries, []string{"model-a", "model-b"}) {
		t.Fatalf("planner retained a legacy query or failed stable deduplication: %#v", output.SearchQueries)
	}

	planner, err = NewDeepSeekQueryPlanner(fakeChatModel{generate: func(context.Context, []*schema.Message) (*schema.Message, error) {
		queries := make([]string, 15)
		for index := range queries {
			queries[index] = fmt.Sprintf("model-%02d", index)
		}
		return schema.AssistantMessage(`{"queries":["`+strings.Join(queries, `","`)+`"],"combined_query":"model query"}`, nil), nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	output, err = planner.Plan(context.Background(), &Request{Prompt: "intent"})
	if err != nil {
		t.Fatal(err)
	}
	if len(output.SearchQueries) != 12 || output.SearchQueries[0] != "model-00" || output.SearchQueries[11] != "model-11" {
		t.Fatalf("unexpected model limit: %#v", output.SearchQueries)
	}
}

func TestDeepSeekQueryPlannerSanitizesModelFailureAndPreservesCancellation(t *testing.T) {
	const apiKey = "secret-deepseek-api-key"
	const rawResponse = `{"error":"secret raw response"}`
	const prompt = "complete secret prompt"
	planner, err := NewDeepSeekQueryPlanner(fakeChatModel{generate: func(context.Context, []*schema.Message) (*schema.Message, error) {
		return nil, fmt.Errorf("provider failed key=%s prompt=%s response=%s", apiKey, prompt, rawResponse)
	}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = planner.Plan(context.Background(), &Request{Prompt: prompt})
	if err == nil {
		t.Fatal("expected model error")
	}
	for _, secret := range []string{apiKey, prompt, rawResponse} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("error leaked %q: %v", secret, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	planner, err = NewDeepSeekQueryPlanner(fakeChatModel{generate: func(ctx context.Context, _ []*schema.Message) (*schema.Message, error) {
		return nil, ctx.Err()
	}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = planner.Plan(ctx, &Request{Prompt: "prompt"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
