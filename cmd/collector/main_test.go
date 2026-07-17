package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/components/model"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/connectors"
)

func TestCollectorPromptFlagsContainOnlyTechnicalInputs(t *testing.T) {
	options, err := parseCollectorOptions(nil)
	if err != nil {
		t.Fatal(err)
	}
	if options.promptFile != defaultCollectorPromptFile {
		t.Fatalf("default prompt file = %q", options.promptFile)
	}
	if defaultCollectorPromptFile != "agents/collector/prompts/query_planner_v1.md" {
		t.Fatalf("default prompt path changed to %q", defaultCollectorPromptFile)
	}
	if options.dataRoot != "data" || options.envFile != ".env" || options.candidateLimit != 10 || options.tavilyTopic != "general" || options.tavilyMaxResults != 5 || options.timeWindow != 48 || options.maxParallel != 3 {
		t.Fatalf("unexpected technical defaults: %+v", options)
	}

	want := collectorOptions{
		promptFile: "runtime/intent.md", dataRoot: "runtime-data", envFile: "runtime.env",
		candidateLimit: 6, tavilyTopic: "finance", tavilyMaxResults: 8, timeWindow: 24, maxParallel: 2,
	}
	got, err := parseCollectorOptions([]string{
		"-prompt-file", want.promptFile,
		"-data-root", want.dataRoot,
		"-env-file", want.envFile,
		"-candidate-limit", "6",
		"-tavily-topic", "finance",
		"-tavily-max-results", "8",
		"-time-window-hours", "24",
		"-max-parallel", "2",
	})
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("options = %+v, %v; want %+v", got, err, want)
	}
	for _, removed := range []string{"-objective", "-queries"} {
		if _, err = parseCollectorOptions([]string{removed, "legacy business input"}); err == nil {
			t.Fatalf("removed flag %s was accepted", removed)
		}
	}
}

func TestTavilyTopicOptionAcceptsSupportedValuesAndRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"general", "news", "finance"} {
		options, err := parseCollectorOptions([]string{"-tavily-topic", value})
		if err != nil {
			t.Fatalf("value %s: %v", value, err)
		}
		if options.tavilyTopic != value {
			t.Fatalf("value %s parsed as %q", value, options.tavilyTopic)
		}
	}
	for _, value := range []string{"", "sports", "GENERAL"} {
		if _, err := parseCollectorOptions([]string{"-tavily-topic", value}); err == nil {
			t.Fatalf("invalid value %q was accepted", value)
		}
	}
}

func TestTavilyMaxResultsOptionAcceptsRangeAndRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"1", "20"} {
		options, err := parseCollectorOptions([]string{"-tavily-max-results", value})
		if err != nil {
			t.Fatalf("value %s: %v", value, err)
		}
		if options.tavilyMaxResults != map[string]int{"1": 1, "20": 20}[value] {
			t.Fatalf("value %s parsed as %d", value, options.tavilyMaxResults)
		}
	}
	for _, value := range []string{"0", "21"} {
		if _, err := parseCollectorOptions([]string{"-tavily-max-results", value}); err == nil {
			t.Fatalf("invalid value %s was accepted", value)
		}
	}
}

func TestTavilyTechnicalOptionsOnlyConfigureTavily(t *testing.T) {
	t.Setenv("PARALLEL_API_KEY", "parallel-test")
	t.Setenv("TAVILY_API_KEY", "tavily-test")
	t.Setenv("BOCHA_API_KEY", "bocha-test")
	connectorSet := buildSearchConnectors(collectorOptions{tavilyTopic: "finance", tavilyMaxResults: 8})
	if len(connectorSet) != 3 {
		t.Fatalf("connector count = %d", len(connectorSet))
	}
	if _, ok := connectorSet[0].(connectors.ParallelSearch); !ok {
		t.Fatalf("connector 0 = %T", connectorSet[0])
	}
	tavily, ok := connectorSet[1].(connectors.Tavily)
	if !ok || tavily.Topic != "finance" || tavily.MaxResults != 8 {
		t.Fatalf("connector 1 = %#v", connectorSet[1])
	}
	if _, ok = connectorSet[2].(connectors.Bocha); !ok {
		t.Fatalf("connector 2 = %T", connectorSet[2])
	}
}

func TestCollectorPromptPreparationLoadsObjectiveBeforeProvider(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "test-key")
	promptPath := filepath.Join(t.TempDir(), "intent.md")
	const promptA = "runtime intent A\n"
	if err := os.WriteFile(promptPath, []byte(promptA), 0o600); err != nil {
		t.Fatal(err)
	}
	options := collectorOptions{promptFile: promptPath, candidateLimit: 7, timeWindow: 36}
	now := time.Date(2026, 7, 18, 2, 3, 4, 0, time.FixedZone("CST", 8*60*60))
	factoryCalls := 0
	factory := func(context.Context, *deepseek.ChatModelConfig) (model.BaseChatModel, error) {
		factoryCalls++
		return stubChatModel{}, nil
	}

	requestA, planner, err := prepareCollector(context.Background(), options, now, factory)
	if err != nil {
		t.Fatal(err)
	}
	if planner == nil || factoryCalls != 1 {
		t.Fatalf("planner=%v factoryCalls=%d", planner, factoryCalls)
	}
	if requestA.Objective != promptA || len(requestA.SearchQueries) != 0 || requestA.CandidateLimit != 7 || requestA.TimeWindowHours != 36 || requestA.RunID != "20260717T180304Z" || !requestA.CollectedAt.Equal(now.UTC()) {
		t.Fatalf("request A = %+v", requestA)
	}

	const promptB = "runtime intent B\n"
	if err = os.WriteFile(promptPath, []byte(promptB), 0o600); err != nil {
		t.Fatal(err)
	}
	requestB, _, err := prepareCollector(context.Background(), options, now, factory)
	if err != nil {
		t.Fatal(err)
	}
	if requestA.Objective != promptA || requestB.Objective != promptB {
		t.Fatalf("requestA=%q requestB=%q", requestA.Objective, requestB.Objective)
	}
}

func TestCollectorPromptPreparationFailsBeforeProviderAndSanitizesContent(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "test-key")
	tempDir := t.TempDir()
	factoryCalls := 0
	factory := func(context.Context, *deepseek.ChatModelConfig) (model.BaseChatModel, error) {
		factoryCalls++
		return stubChatModel{}, nil
	}

	missingPath := filepath.Join(tempDir, "missing.md")
	_, _, err := prepareCollector(context.Background(), collectorOptions{promptFile: missingPath}, time.Now(), factory)
	if err == nil || !strings.Contains(err.Error(), missingPath) {
		t.Fatalf("missing prompt error = %v", err)
	}
	if factoryCalls != 0 {
		t.Fatalf("provider called %d times for missing prompt", factoryCalls)
	}

	const promptSecret = "complete secret prompt"
	const apiKey = "test-key"
	const rawResponse = `{"raw":"secret response"}`
	invalidPath := filepath.Join(tempDir, "invalid.md")
	invalidContent := append([]byte(promptSecret+rawResponse), 0xff)
	if err = os.WriteFile(invalidPath, invalidContent, 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err = prepareCollector(context.Background(), collectorOptions{promptFile: invalidPath}, time.Now(), factory)
	if err == nil {
		t.Fatal("expected invalid prompt error")
	}
	for _, secret := range []string{promptSecret, apiKey, rawResponse} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("error leaked %q: %v", secret, err)
		}
	}
	if factoryCalls != 0 {
		t.Fatalf("provider called %d times for invalid prompt", factoryCalls)
	}
}
