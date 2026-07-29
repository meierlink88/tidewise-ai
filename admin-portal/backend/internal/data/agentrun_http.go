package data

import (
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

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"

	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/biz"
)

const (
	defaultMaxReadAttempts = 2
	agentRunAdminPrefix    = "/api/admin/v1"
)

type AgentRunHTTPConfig struct {
	BaseURL         string
	ServiceToken    string
	Timeout         time.Duration
	MaxReadAttempts int
	HTTPClient      *http.Client
}

type AgentRunHTTPClient struct {
	serviceToken    string
	timeout         time.Duration
	maxReadAttempts int
	httpClient      *kratoshttp.Client
	closeIdle       func()
}

func NewAgentRunHTTPClient(config AgentRunHTTPConfig) (*AgentRunHTTPClient, error) {
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
	httpTransport := http.DefaultTransport
	closeIdle := http.DefaultClient.CloseIdleConnections
	if config.HTTPClient != nil && config.HTTPClient.Transport != nil {
		httpTransport = config.HTTPClient.Transport
		closeIdle = config.HTTPClient.CloseIdleConnections
	}
	httpClient, err := kratoshttp.NewClient(
		context.Background(),
		kratoshttp.WithEndpoint(parsed.Scheme+"://"+parsed.Host),
		kratoshttp.WithTimeout(config.Timeout),
		kratoshttp.WithTransport(httpTransport),
		kratoshttp.WithResponseDecoder(decodeAgentRunSuccessResponse),
		kratoshttp.WithErrorDecoder(decodeAgentRunErrorResponse),
	)
	if err != nil {
		return nil, errors.New("create AgentRun HTTP client")
	}
	return &AgentRunHTTPClient{
		serviceToken:    token,
		timeout:         config.Timeout,
		maxReadAttempts: readAttempts,
		httpClient:      httpClient,
		closeIdle:       closeIdle,
	}, nil
}

func (c *AgentRunHTTPClient) Close() error {
	if c == nil {
		return nil
	}
	if c.closeIdle != nil {
		c.closeIdle()
	}
	if c.httpClient != nil {
		return c.httpClient.Close()
	}
	return nil
}

func (c *AgentRunHTTPClient) GetAgentSchedule(ctx context.Context, agentKey string) (biz.AgentSchedule, error) {
	var wire agentScheduleWire
	err := c.doJSON(ctx, http.MethodGet, "AgentRun.GetAgentSchedule", agentRunAdminPrefix+"/agent-schedules/{agent_key}",
		agentRunAdminPrefix+"/agent-schedules/"+url.PathEscape(agentKey), nil, &wire)
	if err != nil {
		return biz.AgentSchedule{}, err
	}
	return wire.toBiz()
}

func (c *AgentRunHTTPClient) PutAgentSchedule(ctx context.Context, agentKey string, input biz.PutAgentScheduleInput) (biz.AgentSchedule, error) {
	var wire agentScheduleWire
	err := c.doJSON(ctx, http.MethodPut, "AgentRun.PutAgentSchedule", agentRunAdminPrefix+"/agent-schedules/{agent_key}",
		agentRunAdminPrefix+"/agent-schedules/"+url.PathEscape(agentKey), newPutAgentScheduleWire(input), &wire)
	if err != nil {
		return biz.AgentSchedule{}, err
	}
	return wire.toBiz()
}

func (c *AgentRunHTTPClient) PatchAgentSchedule(ctx context.Context, agentKey string, input biz.PatchAgentScheduleInput) (biz.AgentSchedule, error) {
	var wire agentScheduleWire
	err := c.doJSON(ctx, http.MethodPatch, "AgentRun.PatchAgentSchedule", agentRunAdminPrefix+"/agent-schedules/{agent_key}",
		agentRunAdminPrefix+"/agent-schedules/"+url.PathEscape(agentKey), newPatchAgentScheduleWire(input), &wire)
	if err != nil {
		return biz.AgentSchedule{}, err
	}
	return wire.toBiz()
}

func (c *AgentRunHTTPClient) ListAgentExecutions(ctx context.Context, query biz.AgentExecutionQuery) (biz.AgentExecutionPage, error) {
	values := url.Values{}
	values.Set("agent_key", query.AgentKey)
	values.Set("page", strconv.Itoa(query.Page))
	values.Set("page_size", strconv.Itoa(query.PageSize))
	var wire agentExecutionPageWire
	err := c.doJSON(ctx, http.MethodGet, "AgentRun.ListAgentExecutions", agentRunAdminPrefix+"/agent-executions",
		agentRunAdminPrefix+"/agent-executions?"+values.Encode(), nil, &wire)
	if err != nil {
		return biz.AgentExecutionPage{}, err
	}
	return wire.toBiz()
}

func (c *AgentRunHTTPClient) ListAgentStatuses(ctx context.Context) ([]biz.AgentStatus, error) {
	var wire agentStatusListWire
	err := c.doJSON(ctx, http.MethodGet, "AgentRun.ListAgentStatuses",
		agentRunAdminPrefix+"/agent-statuses", agentRunAdminPrefix+"/agent-statuses", nil, &wire)
	if err != nil {
		return nil, err
	}
	items := make([]biz.AgentStatus, 0, len(wire.Items))
	for _, item := range wire.Items {
		mapped, mapErr := item.toBiz()
		if mapErr != nil {
			return nil, mapErr
		}
		items = append(items, mapped)
	}
	return items, nil
}

func (c *AgentRunHTTPClient) ListModelProviders(ctx context.Context) ([]biz.ModelProviderConfiguration, error) {
	var wire modelProviderListWire
	err := c.doJSON(ctx, http.MethodGet, "AgentRun.ListModelProviders", agentRunAdminPrefix+"/model-providers",
		agentRunAdminPrefix+"/model-providers", nil, &wire)
	if err != nil {
		return nil, err
	}
	items := make([]biz.ModelProviderConfiguration, 0, len(wire.Items))
	for _, item := range wire.Items {
		mapped, mapErr := item.toBiz()
		if mapErr != nil {
			return nil, mapErr
		}
		items = append(items, mapped)
	}
	return items, nil
}

func (c *AgentRunHTTPClient) GetModelProvider(ctx context.Context, providerKey string) (biz.ModelProviderConfiguration, error) {
	var wire modelProviderWire
	err := c.doJSON(ctx, http.MethodGet, "AgentRun.GetModelProvider", agentRunAdminPrefix+"/model-providers/{provider_key}",
		agentRunAdminPrefix+"/model-providers/"+url.PathEscape(providerKey), nil, &wire)
	if err != nil {
		return biz.ModelProviderConfiguration{}, err
	}
	return wire.toBiz()
}

func (c *AgentRunHTTPClient) PatchModelProvider(
	ctx context.Context,
	providerKey string,
	patch biz.ModelProviderPatch,
) (biz.ModelProviderConfiguration, error) {
	var wire modelProviderWire
	request := modelProviderPatchWire{BaseURL: patch.BaseURL, Model: patch.Model, APIKey: patch.APIKey}
	err := c.doJSON(ctx, http.MethodPatch, "AgentRun.PatchModelProvider", agentRunAdminPrefix+"/model-providers/{provider_key}",
		agentRunAdminPrefix+"/model-providers/"+url.PathEscape(providerKey), request, &wire)
	if err != nil {
		return biz.ModelProviderConfiguration{}, err
	}
	return wire.toBiz()
}

func (c *AgentRunHTTPClient) ListConnectors(ctx context.Context) ([]biz.ConnectorConfiguration, error) {
	var wire connectorListWire
	err := c.doJSON(ctx, http.MethodGet, "AgentRun.ListConnectors", agentRunAdminPrefix+"/connectors",
		agentRunAdminPrefix+"/connectors", nil, &wire)
	if err != nil {
		return nil, err
	}
	items := make([]biz.ConnectorConfiguration, 0, len(wire.Items))
	for _, item := range wire.Items {
		mapped, mapErr := item.toBiz()
		if mapErr != nil {
			return nil, mapErr
		}
		items = append(items, mapped)
	}
	return items, nil
}

func (c *AgentRunHTTPClient) GetConnector(ctx context.Context, connectorKey string) (biz.ConnectorConfiguration, error) {
	var wire connectorWire
	err := c.doJSON(ctx, http.MethodGet, "AgentRun.GetConnector", agentRunAdminPrefix+"/connectors/{connector_key}",
		agentRunAdminPrefix+"/connectors/"+url.PathEscape(connectorKey), nil, &wire)
	if err != nil {
		return biz.ConnectorConfiguration{}, err
	}
	return wire.toBiz()
}

func (c *AgentRunHTTPClient) PatchConnector(
	ctx context.Context,
	connectorKey string,
	patch biz.ConnectorPatch,
) (biz.ConnectorConfiguration, error) {
	var wire connectorWire
	request := connectorPatchWire{BaseURL: patch.BaseURL, APIKey: patch.APIKey}
	err := c.doJSON(ctx, http.MethodPatch, "AgentRun.PatchConnector", agentRunAdminPrefix+"/connectors/{connector_key}",
		agentRunAdminPrefix+"/connectors/"+url.PathEscape(connectorKey), request, &wire)
	if err != nil {
		return biz.ConnectorConfiguration{}, err
	}
	return wire.toBiz()
}

func (c *AgentRunHTTPClient) doJSON(
	ctx context.Context,
	method string,
	operation string,
	pathTemplate string,
	path string,
	requestBody any,
	result any,
) error {
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
	requestID := RequestIDFromContext(ctx)
	if requestID == "" {
		requestID = newRequestID()
	}

	attempts := 1
	if method == http.MethodGet {
		attempts = c.maxReadAttempts
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		retryable, err := c.doJSONAttempt(operationCtx, method, operation, pathTemplate, path, payload, requestID, result)
		if err == nil {
			return nil
		}
		lastErr = err
		if !retryable {
			return mapAgentRunAdapterError(err)
		}
	}
	return mapAgentRunAdapterError(lastErr)
}

func (c *AgentRunHTTPClient) doJSONAttempt(
	ctx context.Context,
	method string,
	operation string,
	pathTemplate string,
	path string,
	payload []byte,
	requestID string,
	result any,
) (bool, error) {
	var body any
	if payload != nil {
		body = json.RawMessage(payload)
	}
	headers := http.Header{
		"Accept":        []string{"application/json"},
		"Authorization": []string{"Bearer " + c.serviceToken},
		RequestIDHeader: []string{requestID},
		"Content-Type":  []string{"application/json"},
	}
	err := c.httpClient.Invoke(
		ctx,
		method,
		path,
		body,
		result,
		kratoshttp.Header(&headers),
		kratoshttp.Accept("application/json"),
		kratoshttp.Operation(operation),
		kratoshttp.PathTemplate(pathTemplate),
	)
	if err != nil {
		var upstream *agentRunHTTPError
		if errors.As(err, &upstream) {
			return upstream.status >= http.StatusInternalServerError, upstream
		}
		var decodeErr *agentRunDecodeError
		if errors.As(err, &decodeErr) {
			return false, decodeErr
		}
		return true, fmt.Errorf("call AgentRun")
	}
	return false, nil
}

func mapAgentRunAdapterError(err error) error {
	var upstream *agentRunHTTPError
	if errors.As(err, &upstream) {
		return agentRunBusinessError(upstream.status)
	}
	if err == nil {
		return nil
	}
	return biz.ErrAgentRunUnavailable
}

func decodeAgentRunSuccessResponse(_ context.Context, response *http.Response, result any) error {
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBodyBytes+1))
	if err != nil {
		return &agentRunDecodeError{}
	}
	if len(responseBody) > maxResponseBodyBytes {
		return &agentRunDecodeError{}
	}
	var envelope struct {
		Result json.RawMessage `json:"result"`
	}
	if len(responseBody) == 0 || result == nil || json.Unmarshal(responseBody, &envelope) != nil ||
		len(envelope.Result) == 0 || string(envelope.Result) == "null" ||
		json.Unmarshal(envelope.Result, result) != nil {
		return &agentRunDecodeError{}
	}
	return nil
}

func decodeAgentRunErrorResponse(_ context.Context, response *http.Response) error {
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		return nil
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBodyBytes+1))
	if err != nil {
		return &agentRunHTTPError{status: response.StatusCode}
	}
	var errorBody struct {
		Code string `json:"error_code"`
	}
	if len(responseBody) <= maxResponseBodyBytes {
		_ = json.Unmarshal(responseBody, &errorBody)
	}
	return &agentRunHTTPError{status: response.StatusCode, code: safeMetadata(errorBody.Code, maxErrorCodeLength)}
}

type agentRunDecodeError struct{}

func (*agentRunDecodeError) Error() string {
	return "AgentRun returned an invalid response"
}

type agentRunHTTPError struct {
	status int
	code   string
}

func (*agentRunHTTPError) Error() string {
	return "AgentRun rejected the request"
}
