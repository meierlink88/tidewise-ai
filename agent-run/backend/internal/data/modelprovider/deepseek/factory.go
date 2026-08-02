package deepseek

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

const defaultTimeout = 30 * time.Second

// Factory adapts AgentRun's current model configuration to the official Eino
// DeepSeek component while exposing only the Eino core model interface.
type Factory struct {
	Timeout   time.Duration
	Transport http.RoundTripper
}

func (f Factory) New(ctx context.Context, config agentrun.ModelProviderConfig) (model.BaseChatModel, error) {
	return f.new(ctx, config, true)
}

// NewToolCalling creates the official Eino OpenAI-compatible model without a
// competing json_object response-format requirement. Event Fact binds one
// immutable result Function per deterministic workflow stage.
func (f Factory) NewToolCalling(
	ctx context.Context,
	config agentrun.ModelProviderConfig,
) (model.ToolCallingChatModel, error) {
	chatModel, err := f.new(ctx, config, false)
	if err != nil {
		return nil, err
	}
	toolModel, ok := chatModel.(model.ToolCallingChatModel)
	if !ok {
		return nil, errors.New("model provider does not support Function Calling")
	}
	return toolModel, nil
}

func (f Factory) new(
	ctx context.Context,
	config agentrun.ModelProviderConfig,
	jsonObject bool,
) (model.BaseChatModel, error) {
	timeout := f.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	if timeout < 0 {
		return nil, errors.New("model timeout must be positive")
	}
	modelConfig := &einoopenai.ChatModelConfig{
		APIKey:     config.APIKey,
		Model:      config.Model,
		BaseURL:    config.BaseURL,
		Timeout:    timeout,
		HTTPClient: executionBoundClient(ctx, timeout, f.Transport),
		ExtraFields: map[string]any{
			"thinking": map[string]string{"type": "disabled"},
		},
	}
	if jsonObject {
		modelConfig.ResponseFormat = &einoopenai.ChatCompletionResponseFormat{
			Type: einoopenai.ChatCompletionResponseFormatTypeJSONObject,
		}
	}
	return einoopenai.NewChatModel(ctx, modelConfig)
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
			response, err := transport.RoundTrip(request.WithContext(requestContext))
			if err != nil {
				stop()
				cancel()
				return nil, err
			}
			if response.Body == nil {
				stop()
				cancel()
				return response, nil
			}
			response.Body = &executionBoundBody{
				ReadCloser: response.Body,
				stop:       stop,
				cancel:     cancel,
			}
			return response, nil
		}),
	}
}

type executionBoundBody struct {
	io.ReadCloser
	stop   func() bool
	cancel context.CancelFunc
	once   sync.Once
}

func (b *executionBoundBody) Read(payload []byte) (int, error) {
	read, err := b.ReadCloser.Read(payload)
	if err != nil {
		b.finish()
	}
	return read, err
}

func (b *executionBoundBody) Close() error {
	err := b.ReadCloser.Close()
	b.finish()
	return err
}

func (b *executionBoundBody) finish() {
	b.once.Do(func() {
		b.stop()
		b.cancel()
	})
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
