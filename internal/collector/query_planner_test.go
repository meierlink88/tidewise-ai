package collector

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type fakeChatModel struct {
	generate func(context.Context, []*schema.Message) (*schema.Message, error)
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
		return schema.AssistantMessage(`{"queries":["same"," model query ","same"]}`, nil), nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	originalQueries := []string{"legacy query"}
	const objective = "prompt-owned intent\nwith exact whitespace\n"
	input := &Request{
		RunID: "run-1", Objective: objective, SearchQueries: originalQueries,
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
	if output.RunID != input.RunID || output.Objective != input.Objective || output.CandidateLimit != input.CandidateLimit || output.TimeWindowHours != input.TimeWindowHours || !output.CollectedAt.Equal(input.CollectedAt) {
		t.Fatalf("non-query fields changed: input=%+v output=%+v", input, output)
	}
	output.SearchQueries[0] = "changed"
	if input.SearchQueries[0] != "legacy query" || originalQueries[0] != "legacy query" {
		t.Fatal("output SearchQueries aliases caller input")
	}
	if len(captured) != 2 || captured[0].Role != schema.System || captured[0].Content != objective || captured[1].Role != schema.User {
		t.Fatalf("unexpected messages: %#v", captured)
	}
	for _, value := range []string{"48", "2026-07-17T17:02:03Z"} {
		if !strings.Contains(captured[1].Content, value) {
			t.Fatalf("user prompt missing %q: %s", value, captured[1].Content)
		}
	}
	for _, forbidden := range []string{objective, "legacy query", "objective", "search_queries"} {
		if strings.Contains(captured[1].Content, forbidden) {
			t.Fatalf("technical user message contains %q: %s", forbidden, captured[1].Content)
		}
	}
}

func TestDeepSeekQueryPlannerStrictJSONValidation(t *testing.T) {
	tests := map[string]string{
		"empty response":     "",
		"malformed":          `{"queries":[}`,
		"unknown field":      `{"queries":["q"],"facts":"not allowed"}`,
		"model objective":    `{"objective":"rewritten","queries":["q"]}`,
		"wrong type":         `{"queries":"q"}`,
		"empty queries":      `{"queries":[]}`,
		"empty query":        `{"queries":["  "]}`,
		"trailing object":    `{"queries":["q"]}{"queries":["other"]}`,
		"trailing primitive": `{"queries":["q"]} true`,
		"too long":           fmt.Sprintf(`{"queries":[%q]}`, strings.Repeat("界", 257)),
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			planner, err := NewDeepSeekQueryPlanner(fakeChatModel{generate: func(context.Context, []*schema.Message) (*schema.Message, error) {
				return schema.AssistantMessage(content, nil), nil
			}})
			if err != nil {
				t.Fatal(err)
			}
			_, err = planner.Plan(context.Background(), &Request{Objective: "system-prompt-secret"})
			if err == nil {
				t.Fatal("expected schema error")
			}
			if content != "" && strings.Contains(err.Error(), content) || strings.Contains(err.Error(), "system-prompt-secret") {
				t.Fatalf("error leaked raw content or prompt: %v", err)
			}
		})
	}
}

func TestDeepSeekQueryPlannerRejectsEmptyObjectiveBeforeModelCall(t *testing.T) {
	modelCalls := 0
	planner, err := NewDeepSeekQueryPlanner(fakeChatModel{generate: func(context.Context, []*schema.Message) (*schema.Message, error) {
		modelCalls++
		return schema.AssistantMessage(`{"queries":["q"]}`, nil), nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = planner.Plan(context.Background(), &Request{Objective: " \n\t"})
	if !errors.Is(err, ErrQueryPlanningSchema) || modelCalls != 0 {
		t.Fatalf("error=%v modelCalls=%d", err, modelCalls)
	}
}

func TestDeepSeekQueryPlannerUsesOnlyModelQueriesAndLimitsQueries(t *testing.T) {
	planner, err := NewDeepSeekQueryPlanner(fakeChatModel{generate: func(context.Context, []*schema.Message) (*schema.Message, error) {
		return schema.AssistantMessage(`{"queries":["model-a"," model-a ","model-b"]}`, nil), nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	output, err := planner.Plan(context.Background(), &Request{Objective: "intent", SearchQueries: []string{"legacy-query"}})
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
		return schema.AssistantMessage(`{"queries":["`+strings.Join(queries, `","`)+`"]}`, nil), nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	output, err = planner.Plan(context.Background(), &Request{Objective: "intent"})
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
	_, err = planner.Plan(context.Background(), &Request{Objective: prompt})
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
	_, err = planner.Plan(ctx, &Request{Objective: "prompt"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
