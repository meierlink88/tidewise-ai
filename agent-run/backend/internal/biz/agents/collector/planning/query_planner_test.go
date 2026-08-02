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
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector"
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

func TestDeepSeekQueryPlannerProtocolBoundsCombinedQueryAndForbidsSources(t *testing.T) {
	var captured []*schema.Message
	planner, err := NewDeepSeekQueryPlanner(fakeChatModel{generate: func(_ context.Context, messages []*schema.Message) (*schema.Message, error) {
		captured = messages
		return schema.AssistantMessage(`{"queries":["global macro"],"combined_query":"global macro"}`, nil), nil
	}})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := planner.Plan(context.Background(), &Request{Prompt: "global macro news"}); err != nil {
		t.Fatal(err)
	}
	if len(captured) != 2 {
		t.Fatalf("messages = %#v", captured)
	}
	for _, required := range []string{"220 Unicode characters", "search terms, not explanatory prose", "site:", "provider"} {
		if !strings.Contains(captured[0].Content, required) {
			t.Fatalf("Planner protocol does not contain %q: %s", required, captured[0].Content)
		}
	}
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
		"overlong combined and invalid window": fmt.Sprintf(
			`{"queries":["q"],"combined_query":%q,"time_window_hours":0}`,
			strings.Repeat("界", 257),
		),
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			modelCalls := 0
			planner, err := NewDeepSeekQueryPlanner(fakeChatModel{generate: func(context.Context, []*schema.Message) (*schema.Message, error) {
				modelCalls++
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
			if modelCalls != 2 {
				t.Fatalf("model calls = %d, want one initial call and one bounded repair", modelCalls)
			}
		})
	}
}

func TestDeepSeekQueryPlannerRepairsMalformedSchemaOnce(t *testing.T) {
	modelCalls := 0
	var repairMessages []*schema.Message
	planner, err := NewDeepSeekQueryPlanner(fakeChatModel{generate: func(_ context.Context, messages []*schema.Message) (*schema.Message, error) {
		modelCalls++
		if modelCalls == 1 {
			return schema.AssistantMessage(`{"queries":"macro","combined_query":"macro"}`, nil), nil
		}
		repairMessages = messages
		return schema.AssistantMessage(`{"queries":["macro"],"combined_query":"macro"}`, nil), nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	output, err := planner.Plan(context.Background(), &Request{Prompt: "macro news"})
	if err != nil {
		t.Fatal(err)
	}
	if modelCalls != 2 || !reflect.DeepEqual(output.SearchQueries, []string{"macro"}) {
		t.Fatalf("calls=%d output=%#v", modelCalls, output)
	}
	if len(repairMessages) != 2 || !strings.Contains(repairMessages[1].Content, `"violation":"schema_invalid"`) {
		t.Fatalf("repair messages = %#v", repairMessages)
	}
}

func TestDeepSeekQueryPlannerEnforcesCombinedQueryHardLimit(t *testing.T) {
	tests := []struct {
		name    string
		runes   int
		wantErr bool
	}{
		{name: "accepts boundary", runes: 256},
		{name: "rejects over boundary", runes: 257, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			combinedQuery := strings.Repeat("界", test.runes)
			planner, err := NewDeepSeekQueryPlanner(fakeChatModel{generate: func(context.Context, []*schema.Message) (*schema.Message, error) {
				return schema.AssistantMessage(fmt.Sprintf(`{"queries":["q"],"combined_query":%q}`, combinedQuery), nil), nil
			}})
			if err != nil {
				t.Fatal(err)
			}

			output, err := planner.Plan(context.Background(), &Request{Prompt: "intent"})
			if test.wantErr {
				if !errors.Is(err, ErrQueryPlanningSchema) {
					t.Fatalf("error = %v, want schema error", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if output.CombinedQuery != combinedQuery {
				t.Fatalf("combined query changed at hard boundary")
			}
		})
	}
}

func TestDeepSeekQueryPlannerRepairsOnlyOverlongCombinedQueryOnce(t *testing.T) {
	modelCalls := 0
	var repairMessages []*schema.Message
	overlong := strings.Repeat("macro ", 50)
	planCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	wantDeadline, _ := planCtx.Deadline()
	planner, err := NewDeepSeekQueryPlanner(fakeChatModel{generate: func(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
		gotDeadline, ok := ctx.Deadline()
		if !ok || !gotDeadline.Equal(wantDeadline) {
			t.Fatalf("model call deadline = %v, want %v", gotDeadline, wantDeadline)
		}
		modelCalls++
		switch modelCalls {
		case 1:
			return schema.AssistantMessage(fmt.Sprintf(`{"queries":["global macro","central banks"],"combined_query":%q,"time_window_hours":48}`, overlong), nil), nil
		case 2:
			repairMessages = messages
			return schema.AssistantMessage(`{"queries":["global macro","central banks"],"combined_query":"global macro OR central banks","time_window_hours":48}`, nil), nil
		default:
			t.Fatalf("unexpected model call %d", modelCalls)
			return nil, nil
		}
	}})
	if err != nil {
		t.Fatal(err)
	}

	output, err := planner.Plan(planCtx, &Request{Prompt: "global macro news from the last 48 hours"})
	if err != nil {
		t.Fatal(err)
	}
	if modelCalls != 2 {
		t.Fatalf("model calls = %d, want 2", modelCalls)
	}
	if output.CombinedQuery != "global macro OR central banks" || output.TimeWindowHours != 48 {
		t.Fatalf("repaired output = %#v", output)
	}
	if len(repairMessages) != 2 ||
		!strings.Contains(repairMessages[0].Content, "repair") ||
		!strings.Contains(repairMessages[0].Content, "220 Unicode characters") ||
		!strings.Contains(repairMessages[0].Content, "search terms, not explanatory prose") ||
		!strings.Contains(repairMessages[1].Content, overlong) {
		t.Fatalf("repair messages = %#v", repairMessages)
	}
}

func TestDeepSeekQueryPlannerStopsAfterOneOverlongRepair(t *testing.T) {
	modelCalls := 0
	overlong := strings.Repeat("macro ", 50)
	planner, err := NewDeepSeekQueryPlanner(fakeChatModel{generate: func(context.Context, []*schema.Message) (*schema.Message, error) {
		modelCalls++
		return schema.AssistantMessage(fmt.Sprintf(`{"queries":["global macro"],"combined_query":%q}`, overlong), nil), nil
	}})
	if err != nil {
		t.Fatal(err)
	}

	_, err = planner.Plan(context.Background(), &Request{Prompt: "global macro news"})
	if !errors.Is(err, ErrQueryPlanningSchema) {
		t.Fatalf("error = %v, want schema error", err)
	}
	if modelCalls != 2 {
		t.Fatalf("model calls = %d, want exactly 2", modelCalls)
	}
}

func TestDeepSeekQueryPlannerRejectsRepairThatChangesOriginalPlan(t *testing.T) {
	modelCalls := 0
	overlong := strings.Repeat("macro ", 50)
	planner, err := NewDeepSeekQueryPlanner(fakeChatModel{generate: func(context.Context, []*schema.Message) (*schema.Message, error) {
		modelCalls++
		switch modelCalls {
		case 1:
			return schema.AssistantMessage(fmt.Sprintf(`{"queries":["global macro"],"combined_query":%q,"time_window_hours":72}`, overlong), nil), nil
		case 2:
			return schema.AssistantMessage(`{"queries":["changed intent"],"combined_query":"global macro","time_window_hours":48}`, nil), nil
		default:
			t.Fatalf("unexpected model call %d", modelCalls)
			return nil, nil
		}
	}})
	if err != nil {
		t.Fatal(err)
	}

	_, err = planner.Plan(context.Background(), &Request{Prompt: "global macro news"})
	if !errors.Is(err, ErrQueryPlanningSchema) {
		t.Fatalf("error = %v, want schema error", err)
	}
	if modelCalls != 2 {
		t.Fatalf("model calls = %d, want exactly 2", modelCalls)
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
	modelCalls := 0
	planner, err := NewDeepSeekQueryPlanner(fakeChatModel{generate: func(context.Context, []*schema.Message) (*schema.Message, error) {
		modelCalls++
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
	if modelCalls != 1 {
		t.Fatalf("model calls = %d, want 1 for Provider failure", modelCalls)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	modelCalls = 0
	planner, err = NewDeepSeekQueryPlanner(fakeChatModel{generate: func(ctx context.Context, _ []*schema.Message) (*schema.Message, error) {
		modelCalls++
		return nil, ctx.Err()
	}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = planner.Plan(ctx, &Request{Prompt: "prompt"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if modelCalls != 1 {
		t.Fatalf("model calls = %d, want 1 for cancellation", modelCalls)
	}

	deadlineCtx, deadlineCancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer deadlineCancel()
	modelCalls = 0
	planner, err = NewDeepSeekQueryPlanner(fakeChatModel{generate: func(ctx context.Context, _ []*schema.Message) (*schema.Message, error) {
		modelCalls++
		return nil, ctx.Err()
	}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = planner.Plan(deadlineCtx, &Request{Prompt: "prompt"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", err)
	}
	if modelCalls != 1 {
		t.Fatalf("model calls = %d, want 1 for deadline expiry", modelCalls)
	}
}
