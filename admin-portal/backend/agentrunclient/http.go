package agentrunclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultMaxReadAttempts = 2
	maxReadAttempts        = 3
	maxResponseBodyBytes   = 1 << 20
)

type HTTPConfig struct {
	BaseURL         string
	ServiceToken    string
	Timeout         time.Duration
	MaxReadAttempts int
	HTTPClient      *http.Client
}

type HTTPClient struct {
	baseURL         string
	serviceToken    string
	timeout         time.Duration
	maxReadAttempts int
	httpClient      *http.Client
}

func NewHTTPClient(config HTTPConfig) (*HTTPClient, error) {
	parsed, err := url.Parse(strings.TrimSpace(config.BaseURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, errors.New("AgentRun base URL must be an absolute HTTP(S) URL")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return nil, errors.New("AgentRun base URL must not contain credentials, a path, query, or fragment")
	}
	token := strings.TrimSpace(config.ServiceToken)
	if token == "" {
		return nil, errors.New("AgentRun service token is required")
	}
	if config.Timeout <= 0 {
		return nil, errors.New("AgentRun timeout must be positive")
	}
	readAttempts := config.MaxReadAttempts
	if readAttempts == 0 {
		readAttempts = defaultMaxReadAttempts
	}
	if readAttempts < 1 || readAttempts > maxReadAttempts {
		return nil, fmt.Errorf("AgentRun max read attempts must be between 1 and %d", maxReadAttempts)
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &HTTPClient{
		baseURL:         parsed.Scheme + "://" + parsed.Host,
		serviceToken:    token,
		timeout:         config.Timeout,
		maxReadAttempts: readAttempts,
		httpClient:      httpClient,
	}, nil
}

func (c *HTTPClient) GetAgentSchedule(ctx context.Context, agentKey string) (AgentSchedule, error) {
	var schedule AgentSchedule
	err := c.doJSON(ctx, http.MethodGet, AdminAPIPrefix+"/agent-schedules/"+url.PathEscape(agentKey), nil, &schedule)
	return schedule, err
}

func (c *HTTPClient) PutAgentSchedule(ctx context.Context, agentKey string, input PutAgentScheduleInput) (AgentSchedule, error) {
	var schedule AgentSchedule
	err := c.doJSON(ctx, http.MethodPut, AdminAPIPrefix+"/agent-schedules/"+url.PathEscape(agentKey), input, &schedule)
	return schedule, err
}

func (c *HTTPClient) PatchAgentSchedule(ctx context.Context, agentKey string, input PatchAgentScheduleInput) (AgentSchedule, error) {
	var schedule AgentSchedule
	err := c.doJSON(ctx, http.MethodPatch, AdminAPIPrefix+"/agent-schedules/"+url.PathEscape(agentKey), input, &schedule)
	return schedule, err
}

func (c *HTTPClient) ListAgentExecutions(ctx context.Context, query AgentExecutionQuery) (AgentExecutionPage, error) {
	values := url.Values{}
	values.Set("agent_key", query.AgentKey)
	values.Set("page", strconv.Itoa(query.Page))
	values.Set("page_size", strconv.Itoa(query.PageSize))
	var page AgentExecutionPage
	err := c.doJSON(ctx, http.MethodGet, AdminAPIPrefix+"/agent-executions?"+values.Encode(), nil, &page)
	return page, err
}

func (c *HTTPClient) ListModelProviders(ctx context.Context) ([]ModelProviderConfiguration, error) {
	var response struct {
		Items []ModelProviderConfiguration `json:"items"`
	}
	err := c.doJSON(ctx, http.MethodGet, AdminAPIPrefix+"/model-providers", nil, &response)
	return response.Items, err
}

func (c *HTTPClient) GetModelProvider(ctx context.Context, providerKey string) (ModelProviderConfiguration, error) {
	var configuration ModelProviderConfiguration
	err := c.doJSON(ctx, http.MethodGet, AdminAPIPrefix+"/model-providers/"+url.PathEscape(providerKey), nil, &configuration)
	return configuration, err
}

func (c *HTTPClient) PatchModelProvider(
	ctx context.Context,
	providerKey string,
	patch ModelProviderPatch,
) (ModelProviderConfiguration, error) {
	var configuration ModelProviderConfiguration
	err := c.doJSON(ctx, http.MethodPatch, AdminAPIPrefix+"/model-providers/"+url.PathEscape(providerKey), patch, &configuration)
	return configuration, err
}

func (c *HTTPClient) ListConnectors(ctx context.Context) ([]ConnectorConfiguration, error) {
	var response struct {
		Items []ConnectorConfiguration `json:"items"`
	}
	err := c.doJSON(ctx, http.MethodGet, AdminAPIPrefix+"/connectors", nil, &response)
	return response.Items, err
}

func (c *HTTPClient) GetConnector(ctx context.Context, connectorKey string) (ConnectorConfiguration, error) {
	var configuration ConnectorConfiguration
	err := c.doJSON(ctx, http.MethodGet, AdminAPIPrefix+"/connectors/"+url.PathEscape(connectorKey), nil, &configuration)
	return configuration, err
}

func (c *HTTPClient) PatchConnector(
	ctx context.Context,
	connectorKey string,
	patch ConnectorPatch,
) (ConnectorConfiguration, error) {
	var configuration ConnectorConfiguration
	err := c.doJSON(ctx, http.MethodPatch, AdminAPIPrefix+"/connectors/"+url.PathEscape(connectorKey), patch, &configuration)
	return configuration, err
}

func (c *HTTPClient) doJSON(ctx context.Context, method string, path string, requestBody any, result any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	var payload []byte
	if requestBody != nil {
		var err error
		payload, err = json.Marshal(requestBody)
		if err != nil {
			return errors.New("encode AgentRun request")
		}
	}
	operationCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	attempts := 1
	if method == http.MethodGet {
		attempts = c.maxReadAttempts
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		retryable, err := c.doJSONAttempt(operationCtx, method, path, payload, result)
		if err == nil {
			return nil
		}
		lastErr = err
		if !retryable {
			return err
		}
	}
	return lastErr
}

func (c *HTTPClient) doJSONAttempt(
	ctx context.Context,
	method string,
	path string,
	payload []byte,
	result any,
) (bool, error) {
	var requestReader io.Reader
	if payload != nil {
		requestReader = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, requestReader)
	if err != nil {
		return false, fmt.Errorf("create AgentRun request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+c.serviceToken)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		request.Header.Set(RequestIDHeader, requestID)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return true, fmt.Errorf("call AgentRun: %w", err)
	}
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBodyBytes+1))
	_ = response.Body.Close()
	if err != nil {
		return false, fmt.Errorf("read AgentRun response: %w", err)
	}
	if len(responseBody) > maxResponseBodyBytes {
		return false, errors.New("AgentRun response is too large")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		var errorBody struct {
			Code string `json:"error_code"`
		}
		_ = json.Unmarshal(responseBody, &errorBody)
		return response.StatusCode >= http.StatusInternalServerError, &Error{StatusCode: response.StatusCode, Code: errorBody.Code}
	}
	if len(responseBody) == 0 || json.Unmarshal(responseBody, result) != nil {
		return false, errors.New("AgentRun returned an invalid response")
	}
	return false, nil
}

type requestIDContextKey struct{}

const RequestIDHeader = "X-Request-ID"

func WithRequestID(ctx context.Context, requestID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requestIDContextKey{}, strings.TrimSpace(requestID))
}

func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return strings.TrimSpace(requestID)
}
