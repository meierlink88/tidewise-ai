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

	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/biz"
)

const (
	RequestIDHeader      = "X-Request-ID"
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
	baseURL         string
	serviceToken    string
	timeout         time.Duration
	maxReadAttempts int
	httpClient      *kratoshttp.Client
	closeIdle       func()
}

type DataServiceClient = biz.ResearchRepo
type ResearchListQuery = biz.ResearchListQuery
type ResearchDetailQuery = biz.ResearchDetailQuery
type ResearchThemePage = biz.ResearchThemePage
type ResearchTheme = biz.ResearchTheme
type ResearchThemeDetail = biz.ResearchThemeDetail
type ResearchThemeChainNode = biz.ResearchThemeChainNode
type ResearchIndex = biz.ResearchIndex
type ResearchEvent = biz.ResearchEvent
type ResearchReasoningTreeChainNode = biz.ResearchReasoningTreeChainNode
type ResearchReasoningTreeSummary = biz.ResearchReasoningTreeSummary
type ResearchReasoningTreeList = biz.ResearchReasoningTreeList
type ResearchReasoningTreeEvent = biz.ResearchReasoningTreeEvent
type ResearchReasoningTreePathNode = biz.ResearchReasoningTreePathNode
type ResearchReasoningTree = biz.ResearchReasoningTree
type ResearchReasoningTreeDetail = biz.ResearchReasoningTreeDetail
type ImpactLevel = biz.ImpactLevel
type TransmissionStage = biz.TransmissionStage
type EvidenceRole = biz.EvidenceRole
type ImpactDirection = biz.ImpactDirection
type ChangeDirection = biz.ChangeDirection
type ErrorKind = biz.ErrorKind
type Error = biz.Error
type Fake = biz.Fake

const (
	ResearchThemesPath = biz.ResearchThemesPath

	ImpactLevelHigh  = biz.ImpactLevelHigh
	ImpactLevelFocus = biz.ImpactLevelFocus
	ImpactLevelWatch = biz.ImpactLevelWatch

	TransmissionStageIdentification = biz.TransmissionStageIdentification
	TransmissionStageValidation     = biz.TransmissionStageValidation
	TransmissionStageDiffusion      = biz.TransmissionStageDiffusion
	TransmissionStageDampening      = biz.TransmissionStageDampening

	EvidenceRoleDriver        = biz.EvidenceRoleDriver
	EvidenceRoleSupporting    = biz.EvidenceRoleSupporting
	EvidenceRoleContradicting = biz.EvidenceRoleContradicting
	EvidenceRoleContext       = biz.EvidenceRoleContext

	ImpactDirectionPositive = biz.ImpactDirectionPositive
	ImpactDirectionNegative = biz.ImpactDirectionNegative
	ImpactDirectionMixed    = biz.ImpactDirectionMixed
	ImpactDirectionNeutral  = biz.ImpactDirectionNeutral

	ChangeDirectionIncrease  = biz.ChangeDirectionIncrease
	ChangeDirectionDecrease  = biz.ChangeDirectionDecrease
	ChangeDirectionMixed     = biz.ChangeDirectionMixed
	ChangeDirectionUnchanged = biz.ChangeDirectionUnchanged
	ChangeDirectionUncertain = biz.ChangeDirectionUncertain

	ErrorKindClient     = biz.ErrorKindClient
	ErrorKindConflict   = biz.ErrorKindConflict
	ErrorKindServer     = biz.ErrorKindServer
	ErrorKindConnection = biz.ErrorKindConnection
	ErrorKindTimeout    = biz.ErrorKindTimeout
	ErrorKindCanceled   = biz.ErrorKindCanceled
	ErrorKindProtocol   = biz.ErrorKindProtocol
	ErrorKindEncode     = biz.ErrorKindEncode
	ErrorKindDecode     = biz.ErrorKindDecode
)

var ErrFakeMethodNotConfigured = biz.ErrFakeMethodNotConfigured

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
		baseURL:         parsed.Scheme + "://" + parsed.Host,
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

func (c *HTTPClient) ListResearchThemes(ctx context.Context, query ResearchListQuery) (ResearchThemePage, error) {
	var envelope responseEnvelope[ResearchThemePage]
	err := c.doJSON(ctx, http.MethodGet, researchListPath(ResearchThemesPath, query), nil, &envelope)
	return unwrapEnvelope(envelope, err)
}

func (c *HTTPClient) GetResearchTheme(ctx context.Context, id string, query ResearchDetailQuery) (ResearchThemeDetail, error) {
	var envelope responseEnvelope[ResearchThemeDetail]
	err := c.doJSON(ctx, http.MethodGet, researchDetailPath(ResearchThemesPath, id, query), nil, &envelope)
	return unwrapEnvelope(envelope, err)
}

func (c *HTTPClient) ListResearchThemeReasoningTrees(ctx context.Context, themeID string) (ResearchReasoningTreeList, error) {
	var envelope responseEnvelope[ResearchReasoningTreeList]
	err := c.doJSON(ctx, http.MethodGet, researchReasoningTreeListPath(themeID), nil, &envelope)
	return unwrapEnvelope(envelope, err)
}

func (c *HTTPClient) GetResearchThemeReasoningTree(ctx context.Context, themeID, anchorID string) (ResearchReasoningTreeDetail, error) {
	var envelope responseEnvelope[ResearchReasoningTreeDetail]
	err := c.doJSON(ctx, http.MethodGet, researchReasoningTreeDetailPath(themeID, anchorID), nil, &envelope)
	return unwrapEnvelope(envelope, err)
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

func researchListPath(path string, query ResearchListQuery) string {
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

func researchDetailPath(path string, id string, query ResearchDetailQuery) string {
	values := url.Values{}
	if query.WindowHours != 0 {
		values.Set("window_hours", strconv.Itoa(query.WindowHours))
	}
	return appendQuery(path+"/"+url.PathEscape(id), values)
}

func researchReasoningTreeListPath(themeID string) string {
	return ResearchThemesPath + "/" + url.PathEscape(themeID) + "/reasoning-trees"
}

func researchReasoningTreeDetailPath(themeID, anchorID string) string {
	return researchReasoningTreeListPath(themeID) + "/" + url.PathEscape(anchorID)
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

func safeMetadata(value string, maxLength int) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maxLength {
		return ""
	}
	for _, character := range value {
		switch {
		case character >= 'a' && character <= 'z':
		case character >= 'A' && character <= 'Z':
		case character >= '0' && character <= '9':
		case character == '-', character == '_', character == '.', character == ':':
		default:
			return ""
		}
	}
	return value
}

func newRequestID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err == nil {
		return "req-" + hex.EncodeToString(value)
	}
	return "req-" + strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
}
