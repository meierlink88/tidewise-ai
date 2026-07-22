package application

import (
	"context"
	"net/http"
	"time"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/compose"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector/artifacts"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector/connectors"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector/planning"
	collectorworkflow "github.com/guanchaojia/tidewise-ai-agentrun/internal/collector/workflow"
)

type runtimeFactory struct {
	store        Repository
	artifactRoot string
	now          func() time.Time
}

func (f runtimeFactory) Build(ctx context.Context, executionID string, providerConfig collector.ProviderConfiguration) (compose.Runnable[*collector.Request, *collector.Result], error) {
	model, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey: providerConfig.DeepSeek.APIKey, Model: providerConfig.DeepSeek.Model,
		BaseURL: providerConfig.DeepSeek.BaseURL, Timeout: plannerTimeout,
		ResponseFormatType: deepseek.ResponseFormatTypeJSONObject,
	})
	if err != nil {
		return nil, err
	}
	planner, err := planning.NewDeepSeekQueryPlanner(model)
	if err != nil {
		return nil, err
	}
	plannerWithState := &trackingPlanner{executionID: executionID, store: f.store, delegate: planner, now: f.now}

	httpClient := &http.Client{Timeout: connectorTimeout}
	configured := providerConfig.Connectors
	connectorSet := []collector.Connector{
		connectors.ParallelSearch{APIKey: configured[collector.ProviderParallelSearch].APIKey, Endpoint: configured[collector.ProviderParallelSearch].BaseURL, Client: httpClient},
		connectors.Tavily{APIKey: configured[collector.ProviderTavily].APIKey, Endpoint: configured[collector.ProviderTavily].BaseURL, Client: httpClient, Topic: "finance", MaxResults: 5},
		connectors.Bocha{APIKey: configured[collector.ProviderBocha].APIKey, Endpoint: configured[collector.ProviderBocha].BaseURL, Client: httpClient},
		connectors.CLSTelegraph{Endpoint: configured[collector.ProviderCLSTelegraph].BaseURL, Client: httpClient},
		connectors.EastmoneyFastNews{Endpoint: configured[collector.ProviderEastmoneyFastNews].BaseURL, Client: httpClient},
		connectors.EastmoneyStockNews{Endpoint: configured[collector.ProviderEastmoneyStock].BaseURL, Client: httpClient},
		connectors.STCNQuickNews{Endpoint: configured[collector.ProviderSTCNQuickNews].BaseURL, Client: httpClient},
	}
	tracked := make([]collector.Connector, 0, len(connectorSet))
	for _, connector := range connectorSet {
		tracked = append(tracked, &trackingConnector{executionID: executionID, store: f.store, delegate: connector, now: f.now})
	}
	materializer := &trackingMaterializer{
		executionID: executionID, store: f.store, now: f.now,
		delegate: artifacts.File{Root: f.artifactRoot, NearDuplicateRadius: 3},
	}
	return collectorworkflow.New(ctx, plannerWithState, tracked, maxParallel, materializer)
}
