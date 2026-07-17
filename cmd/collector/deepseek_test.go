package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/config"
)

type stubChatModel struct{}

func (stubChatModel) Generate(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error) {
	return schema.AssistantMessage(`{"queries":["planned"]}`, nil), nil
}

func (stubChatModel) Stream(context.Context, []*schema.Message, ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("unexpected stream call")
}

func TestBuildDeepSeekPlannerMapsConfigThroughInjectedFactory(t *testing.T) {
	input := config.DeepSeekConfig{
		APIKey: "test-key", Model: "deepseek-chat", BaseURL: "https://deepseek.test/v1", Timeout: 45 * time.Second,
	}
	var captured *deepseek.ChatModelConfig
	factoryCalls := 0
	factory := func(_ context.Context, got *deepseek.ChatModelConfig) (model.BaseChatModel, error) {
		factoryCalls++
		copyConfig := *got
		captured = &copyConfig
		return stubChatModel{}, nil
	}
	planner, err := buildDeepSeekPlanner(context.Background(), input, "prompt-v1", factory)
	if err != nil {
		t.Fatal(err)
	}
	if planner == nil || factoryCalls != 1 {
		t.Fatalf("planner=%v factoryCalls=%d", planner, factoryCalls)
	}
	if captured.APIKey != input.APIKey || captured.Model != input.Model || captured.BaseURL != input.BaseURL || captured.Timeout != input.Timeout {
		t.Fatalf("mapped config = %+v", captured)
	}
	if captured.ResponseFormatType != deepseek.ResponseFormatTypeJSONObject {
		t.Fatalf("response format = %q", captured.ResponseFormatType)
	}
}

func TestBuildDeepSeekPlannerSanitizesFactoryError(t *testing.T) {
	const apiKey = "secret-deepseek-key"
	const prompt = "complete secret prompt"
	const rawResponse = `{"raw":"secret"}`
	factory := func(context.Context, *deepseek.ChatModelConfig) (model.BaseChatModel, error) {
		return nil, fmt.Errorf("factory failed key=%s prompt=%s response=%s", apiKey, prompt, rawResponse)
	}
	_, err := buildDeepSeekPlanner(context.Background(), config.DeepSeekConfig{APIKey: apiKey, Model: "deepseek-chat", Timeout: time.Second}, prompt, factory)
	if err == nil {
		t.Fatal("expected factory error")
	}
	for _, secret := range []string{apiKey, prompt, rawResponse} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("error leaked %q: %v", secret, err)
		}
	}
}
