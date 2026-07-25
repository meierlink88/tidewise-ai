package deepseek

import (
	"context"
	"errors"
	"net/http"
	"time"

	einodeepseek "github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/components/model"
	agentrun "github.com/guanchaojia/tidewise-ai-agentrun/internal/biz/platform"
)

const defaultTimeout = 30 * time.Second

// Factory adapts AgentRun's current model configuration to the official Eino
// DeepSeek component while exposing only the Eino core model interface.
type Factory struct {
	Timeout   time.Duration
	Transport http.RoundTripper
}

func (f Factory) New(ctx context.Context, config agentrun.ModelProviderConfig) (model.BaseChatModel, error) {
	timeout := f.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	if timeout < 0 {
		return nil, errors.New("model timeout must be positive")
	}
	return einodeepseek.NewChatModel(ctx, &einodeepseek.ChatModelConfig{
		APIKey:             config.APIKey,
		Model:              config.Model,
		BaseURL:            config.BaseURL,
		Timeout:            timeout,
		HTTPClient:         executionBoundClient(ctx, timeout, f.Transport),
		ResponseFormatType: einodeepseek.ResponseFormatTypeJSONObject,
	})
}

func executionBoundClient(executionContext context.Context, timeout time.Duration, transport http.RoundTripper) *http.Client {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &http.Client{
		Timeout: timeout,
		Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
			requestContext, cancel := context.WithCancel(request.Context())
			stop := context.AfterFunc(executionContext, cancel)
			defer func() {
				stop()
				cancel()
			}()
			return transport.RoundTrip(request.WithContext(requestContext))
		}),
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
