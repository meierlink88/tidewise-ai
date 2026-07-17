package main

import (
	"context"
	"errors"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/components/model"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector"
	projectconfig "github.com/guanchaojia/tidewise-ai-agentrun/internal/config"
)

type deepSeekModelFactory func(context.Context, *deepseek.ChatModelConfig) (model.BaseChatModel, error)

func newDeepSeekChatModel(ctx context.Context, config *deepseek.ChatModelConfig) (model.BaseChatModel, error) {
	return deepseek.NewChatModel(ctx, config)
}

func buildDeepSeekPlanner(
	ctx context.Context,
	config projectconfig.DeepSeekConfig,
	systemPrompt string,
	factory deepSeekModelFactory,
) (collector.QueryPlanner, error) {
	if factory == nil {
		return nil, errors.New("DeepSeek provider factory is required")
	}
	chatModel, err := factory(ctx, &deepseek.ChatModelConfig{
		APIKey:             config.APIKey,
		Model:              config.Model,
		BaseURL:            config.BaseURL,
		Timeout:            config.Timeout,
		ResponseFormatType: deepseek.ResponseFormatTypeJSONObject,
	})
	if err != nil {
		return nil, errors.New("initialize DeepSeek provider failed")
	}
	planner, err := collector.NewDeepSeekQueryPlanner(chatModel, systemPrompt)
	if err != nil {
		return nil, errors.New("initialize DeepSeek query planner failed")
	}
	return planner, nil
}
