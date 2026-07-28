package data

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/transport"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"

	"github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz"
)

const (
	RequestIDHeader      = "X-Request-ID"
	DataAPIPrefix        = "/api/data/v1"
	ResearchThemesPath   = DataAPIPrefix + "/research/themes"
	maxResponseBodyBytes = 1 << 20
	maxErrorCodeLength   = 100
	maxReadAttempts      = 3
)

type HTTPConfig struct {
	BaseURL         string
	ServiceToken    string
	Timeout         time.Duration
	MaxReadAttempts int
	HTTPClient      *http.Client
}

type HTTPClient struct {
	serviceToken    string
	timeout         time.Duration
	maxReadAttempts int
	httpClient      *kratoshttp.Client
	closeIdle       func()
}

func NewHTTPClient(config HTTPConfig) (*HTTPClient, error) {
	parsed, err := url.Parse(strings.TrimSpace(config.BaseURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, errors.New("data service base URL must be an absolute HTTP(S) URL")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return nil, errors.New("data service base URL must not contain credentials, a path, query, or fragment")
	}
	token := strings.TrimSpace(config.ServiceToken)
	if token == "" {
		return nil, errors.New("data service identity token is required")
	}
	if config.Timeout <= 0 {
		return nil, errors.New("data service timeout must be positive")
	}
	attempts := config.MaxReadAttempts
	if attempts == 0 {
		attempts = 2
	}
	if attempts < 1 || attempts > maxReadAttempts {
		return nil, fmt.Errorf("data service read attempts must be between 1 and %d", maxReadAttempts)
	}
	transport := http.DefaultTransport
	closeIdle := http.DefaultClient.CloseIdleConnections
	if config.HTTPClient != nil && config.HTTPClient.Transport != nil {
		transport = config.HTTPClient.Transport
		closeIdle = config.HTTPClient.CloseIdleConnections
	}
	httpClient, err := kratoshttp.NewClient(
		context.Background(),
		kratoshttp.WithEndpoint(parsed.Scheme+"://"+parsed.Host),
		kratoshttp.WithTimeout(config.Timeout),
		kratoshttp.WithTransport(transport),
		kratoshttp.WithResponseDecoder(decodeSuccessResponse),
		kratoshttp.WithErrorDecoder(decodeErrorResponse),
	)
	if err != nil {
		return nil, errors.New("create data service HTTP client")
	}
	return &HTTPClient{
		serviceToken:    token,
		timeout:         config.Timeout,
		maxReadAttempts: attempts,
		httpClient:      httpClient,
		closeIdle:       closeIdle,
	}, nil
}

func (c *HTTPClient) Close() error {
	if c == nil {
		return nil
	}
	if c.closeIdle != nil {
		c.closeIdle()
	}
	if c.httpClient == nil {
		return nil
	}
	return c.httpClient.Close()
}

func (c *HTTPClient) ListResearchThemes(ctx context.Context, query biz.ResearchListQuery) (biz.ResearchThemePage, error) {
	var envelope responseEnvelope[wireResearchThemePage]
	err := c.doJSON(ctx, http.MethodGet, researchListPath(ResearchThemesPath, query), nil, &envelope)
	value, err := unwrapEnvelope(envelope, err)
	if err != nil {
		return biz.ResearchThemePage{}, mapThemeDataError(err)
	}
	return value.toBiz(), nil
}

func (c *HTTPClient) GetResearchTheme(ctx context.Context, id string, query biz.ResearchDetailQuery) (biz.ResearchThemeDetail, error) {
	var envelope responseEnvelope[wireResearchThemeDetail]
	err := c.doJSON(ctx, http.MethodGet, researchDetailPath(ResearchThemesPath, id, query), nil, &envelope)
	value, err := unwrapEnvelope(envelope, err)
	if err != nil {
		return biz.ResearchThemeDetail{}, mapThemeDataError(err)
	}
	return value.toBiz(), nil
}

func (c *HTTPClient) ListResearchThemeReasoningTrees(ctx context.Context, themeID string) (biz.ResearchReasoningTreeList, error) {
	var envelope responseEnvelope[wireResearchReasoningTreeList]
	err := c.doJSON(ctx, http.MethodGet, researchReasoningTreeListPath(themeID), nil, &envelope)
	value, err := unwrapEnvelope(envelope, err)
	if err != nil {
		return biz.ResearchReasoningTreeList{}, mapReasoningTreeDataError(err)
	}
	return value.toBiz(), nil
}

func (c *HTTPClient) GetResearchThemeReasoningTree(ctx context.Context, themeID, reasoningTreeID string) (biz.ResearchReasoningTreeDetail, error) {
	var envelope responseEnvelope[wireResearchReasoningTreeDetail]
	err := c.doJSON(ctx, http.MethodGet, researchReasoningTreeDetailPath(themeID, reasoningTreeID), nil, &envelope)
	value, err := unwrapEnvelope(envelope, err)
	if err != nil {
		return biz.ResearchReasoningTreeDetail{}, mapReasoningTreeDataError(err)
	}
	return value.toBiz(), nil
}

type responseEnvelope[T any] struct {
	RequestID string `json:"request_id"`
	Result    *T     `json:"result"`
}

func unwrapEnvelope[T any](envelope responseEnvelope[T], err error) (T, error) {
	var zero T
	if err != nil {
		return zero, err
	}
	if envelope.Result == nil || safeMetadata(envelope.RequestID, 128) == "" {
		return zero, &Error{Kind: ErrorKindDecode}
	}
	return *envelope.Result, nil
}

func researchListPath(path string, query biz.ResearchListQuery) string {
	values := url.Values{}
	if query.WindowHours != 0 {
		values.Set("window_hours", strconv.Itoa(query.WindowHours))
	}
	if query.Limit != 0 {
		values.Set("limit", strconv.Itoa(query.Limit))
	}
	if query.Cursor != "" {
		values.Set("cursor", query.Cursor)
	}
	return appendQuery(path, values)
}

func researchDetailPath(path string, id string, query biz.ResearchDetailQuery) string {
	values := url.Values{}
	if query.WindowHours != 0 {
		values.Set("window_hours", strconv.Itoa(query.WindowHours))
	}
	return appendQuery(path+"/"+url.PathEscape(id), values)
}

func researchReasoningTreeListPath(themeID string) string {
	return ResearchThemesPath + "/" + url.PathEscape(themeID) + "/reasoning-trees"
}

func researchReasoningTreeDetailPath(themeID, reasoningTreeID string) string {
	return researchReasoningTreeListPath(themeID) + "/" + url.PathEscape(reasoningTreeID)
}

func appendQuery(path string, values url.Values) string {
	if encoded := values.Encode(); encoded != "" {
		return path + "?" + encoded
	}
	return path
}

type requestIDContextKey struct{}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requestIDContextKey{}, safeMetadata(requestID, 128))
}

func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	if requestID == "" {
		if serverTransport, ok := transport.FromServerContext(ctx); ok {
			requestID = serverTransport.RequestHeader().Get(RequestIDHeader)
		}
	}
	return safeMetadata(requestID, 128)
}

func (c *HTTPClient) doJSON(ctx context.Context, method string, path string, requestBody any, responseBody any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	payload, err := marshalRequest(requestBody)
	if err != nil {
		return err
	}
	requestID := RequestIDFromContext(ctx)
	if requestID == "" {
		requestID = newRequestID()
	}
	operationCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	attempts := 1
	if method == http.MethodGet {
		attempts = c.maxReadAttempts
	}
	for attempt := 1; attempt <= attempts; attempt++ {
		err = c.doJSONAttempt(operationCtx, method, path, payload, requestID, responseBody)
		if err == nil {
			return nil
		}
		if attempt == attempts || !retryableReadFailure(method, err) {
			return err
		}
	}
	return err
}

func marshalRequest(body any) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, &Error{Kind: ErrorKindEncode}
	}
	return payload, nil
}

func (c *HTTPClient) doJSONAttempt(ctx context.Context, method string, path string, payload []byte, requestID string, result any) error {
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
	)
	if err != nil {
		var clientErr *Error
		if errors.As(err, &clientErr) {
			return clientErr
		}
		return transportError(ctx, err)
	}
	return nil
}

func decodeSuccessResponse(_ context.Context, response *http.Response, result any) error {
	bodyBytes, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBodyBytes+1))
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}
	if len(bodyBytes) > maxResponseBodyBytes || len(bodyBytes) == 0 {
		return &Error{Kind: ErrorKindDecode, RequestID: response.Header.Get(RequestIDHeader)}
	}
	if err := json.Unmarshal(bodyBytes, result); err != nil {
		return &Error{Kind: ErrorKindDecode, RequestID: response.Header.Get(RequestIDHeader)}
	}
	return nil
}

func decodeErrorResponse(_ context.Context, response *http.Response) error {
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		return nil
	}
	defer response.Body.Close()
	bodyBytes, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBodyBytes+1))
	if err != nil {
		return &Error{
			Kind:       ErrorKindProtocol,
			StatusCode: response.StatusCode,
			RequestID:  safeMetadata(response.Header.Get(RequestIDHeader), 128),
		}
	}
	return httpStatusError(response.StatusCode, response.Header.Get(RequestIDHeader), bodyBytes)
}

func retryableReadFailure(method string, err error) bool {
	if method != http.MethodGet {
		return false
	}
	var clientErr *Error
	if !errors.As(err, &clientErr) {
		return false
	}
	return clientErr.Kind == ErrorKindConnection || clientErr.Kind == ErrorKindServer
}

func transportError(ctx context.Context, cause error) *Error {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded), errors.Is(cause, context.DeadlineExceeded):
		return &Error{Kind: ErrorKindTimeout}
	case errors.Is(ctx.Err(), context.Canceled), errors.Is(cause, context.Canceled):
		return &Error{Kind: ErrorKindCanceled}
	}
	var networkError net.Error
	if errors.As(cause, &networkError) && networkError.Timeout() {
		return &Error{Kind: ErrorKindTimeout}
	}
	return &Error{Kind: ErrorKindConnection}
}

func httpStatusError(status int, headerRequestID string, body []byte) *Error {
	kind := ErrorKindProtocol
	switch {
	case status == http.StatusConflict:
		kind = ErrorKindConflict
	case status >= 400 && status < 500:
		kind = ErrorKindClient
	case status >= 500:
		kind = ErrorKindServer
	}
	var envelope struct {
		RequestID string `json:"request_id"`
		Error     struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if len(body) <= maxResponseBodyBytes {
		_ = json.Unmarshal(body, &envelope)
	}
	requestID := safeMetadata(headerRequestID, 128)
	if requestID == "" {
		requestID = safeMetadata(envelope.RequestID, 128)
	}
	return &Error{
		Kind:       kind,
		StatusCode: status,
		Code:       safeMetadata(envelope.Error.Code, maxErrorCodeLength),
		RequestID:  requestID,
	}
}

func newRequestID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err == nil {
		return "req-" + hex.EncodeToString(value)
	}
	return "req-" + strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
}
