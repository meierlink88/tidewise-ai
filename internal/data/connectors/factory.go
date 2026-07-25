package connectors

import (
	"errors"
	"net/http"
	"time"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/biz/agents/collector"
)

const defaultTimeout = 30 * time.Second

// Factory constructs the fixed Collector V1 Connector set from one immutable
// runtime configuration snapshot.
type Factory struct {
	Client  *http.Client
	Timeout time.Duration
}

func (f Factory) New(runtime collector.RuntimeConfiguration) ([]collector.Connector, error) {
	client := f.Client
	if client == nil {
		timeout := f.Timeout
		if timeout == 0 {
			timeout = defaultTimeout
		}
		if timeout < 0 {
			return nil, errors.New("Connector timeout must be positive")
		}
		client = &http.Client{Timeout: timeout}
	}
	configured := runtime.Connectors
	return []collector.Connector{
		ParallelSearch{
			APIKey:   configured[collector.ConnectorParallelSearch].APIKey,
			Endpoint: configured[collector.ConnectorParallelSearch].BaseURL,
			Client:   client,
		},
		Tavily{
			APIKey:   configured[collector.ConnectorTavily].APIKey,
			Endpoint: configured[collector.ConnectorTavily].BaseURL,
			Client:   client,
		},
		Bocha{
			APIKey:   configured[collector.ConnectorBocha].APIKey,
			Endpoint: configured[collector.ConnectorBocha].BaseURL,
			Client:   client,
		},
		CLSTelegraph{
			Endpoint: configured[collector.ConnectorCLSTelegraph].BaseURL,
			Client:   client,
		},
		EastmoneyFastNews{
			Endpoint: configured[collector.ConnectorEastmoneyFastNews].BaseURL,
			Client:   client,
		},
		EastmoneyStockNews{
			Endpoint: configured[collector.ConnectorEastmoneyStock].BaseURL,
			Client:   client,
		},
		STCNQuickNews{
			Endpoint: configured[collector.ConnectorSTCNQuickNews].BaseURL,
			Client:   client,
		},
	}, nil
}
