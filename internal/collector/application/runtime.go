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

func (f runtimeFactory) Build(ctx context.Context, executionID string, runtimeConfig collector.RuntimeConfiguration) (compose.Runnable[*collector.Request, *collector.Result], error) {
	model, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey: runtimeConfig.ModelProvider.APIKey, Model: runtimeConfig.ModelProvider.Model,
		BaseURL: runtimeConfig.ModelProvider.BaseURL, Timeout: plannerTimeout,
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
	configured := runtimeConfig.Connectors
	connectorSet := []collector.Connector{
		connectors.ParallelSearch{APIKey: configured[collector.ConnectorParallelSearch].APIKey, Endpoint: configured[collector.ConnectorParallelSearch].BaseURL, Client: httpClient},
		connectors.Tavily{APIKey: configured[collector.ConnectorTavily].APIKey, Endpoint: configured[collector.ConnectorTavily].BaseURL, Client: httpClient},
		connectors.Bocha{APIKey: configured[collector.ConnectorBocha].APIKey, Endpoint: configured[collector.ConnectorBocha].BaseURL, Client: httpClient},
		connectors.CLSTelegraph{Endpoint: configured[collector.ConnectorCLSTelegraph].BaseURL, Client: httpClient},
		connectors.EastmoneyFastNews{Endpoint: configured[collector.ConnectorEastmoneyFastNews].BaseURL, Client: httpClient},
		connectors.EastmoneyStockNews{Endpoint: configured[collector.ConnectorEastmoneyStock].BaseURL, Client: httpClient},
		connectors.STCNQuickNews{Endpoint: configured[collector.ConnectorSTCNQuickNews].BaseURL, Client: httpClient},
	}
	tracked := make([]collector.Connector, 0, len(connectorSet))
	for _, connector := range connectorSet {
		tracked = append(tracked, &trackingConnector{executionID: executionID, store: f.store, delegate: connector, now: f.now})
	}
	materializer := &trackingMaterializer{
		executionID: executionID, store: f.store, now: f.now,
		delegate: artifacts.File{Root: f.artifactRoot, NearDuplicateRadius: 3, Publications: f.store, Now: f.now},
	}
	return collectorworkflow.New(ctx, plannerWithState, tracked, maxParallel, materializer)
}
