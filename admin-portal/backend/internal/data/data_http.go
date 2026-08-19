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

	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/biz"
)

const (
	RequestIDHeader        = "X-Request-ID"
	maxSuccessBodyBytes    = 8 << 20
	maxErrorBodyBytes      = 1 << 20
	maxErrorCodeLength     = 100
	maxReadAttempts        = 3
	dataAPIPrefix          = "/api/data/v1"
	eventsPath             = dataAPIPrefix + "/events"
	evidencesPath          = dataAPIPrefix + "/evidences"
	evidenceCategoriesPath = dataAPIPrefix + "/evidence-categories"
	sourcesPath            = dataAPIPrefix + "/sources"
)

type DataHTTPConfig struct {
	BaseURL         string
	ServiceToken    string
	Timeout         time.Duration
	MaxReadAttempts int
	HTTPClient      *http.Client
}

type DataHTTPClient struct {
	serviceToken    string
	timeout         time.Duration
	maxReadAttempts int
	httpClient      *kratoshttp.Client
	closeIdle       func()
}

func NewDataHTTPClient(config DataHTTPConfig) (*DataHTTPClient, error) {
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
		kratoshttp.WithResponseDecoder(decodeDataSuccessResponse),
		kratoshttp.WithErrorDecoder(decodeDataErrorResponse),
	)
	if err != nil {
		return nil, errors.New("create data service HTTP client")
	}
	return &DataHTTPClient{
		serviceToken:    token,
		timeout:         config.Timeout,
		maxReadAttempts: attempts,
		httpClient:      httpClient,
		closeIdle:       closeIdle,
	}, nil
}

func (c *DataHTTPClient) Close() error {
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

func (c *DataHTTPClient) ListEvents(ctx context.Context, query biz.EventListQuery) (biz.EventPage, error) {
	var envelope responseEnvelope[eventPageWire]
	err := c.doJSON(ctx, http.MethodGet, "Data.ListEvents", eventsPath,
		eventListPath(query), nil, &envelope)
	wire, err := unwrapEnvelope(envelope, err)
	if err != nil {
		return biz.EventPage{}, biz.ErrDataServiceUnavailable
	}
	page, err := wire.toBiz()
	if err != nil {
		return biz.EventPage{}, biz.ErrDataServiceUnavailable
	}
	return page, nil
}

func (c *DataHTTPClient) ListEvidences(ctx context.Context, query biz.EvidenceListQuery) (biz.EvidencePage, error) {
	var envelope responseEnvelope[evidencePageWire]
	err := c.doJSON(ctx, http.MethodGet, "Data.ListEvidences", evidencesPath, evidenceListPath(query), nil, &envelope)
	wire, err := unwrapEnvelope(envelope, err)
	if err != nil {
		return biz.EvidencePage{}, classifyReadError(err)
	}
	page, err := wire.toBiz()
	if err != nil {
		return biz.EvidencePage{}, biz.ErrDataServiceUnavailable
	}
	return page, nil
}

func (c *DataHTTPClient) ListEvidenceCategories(ctx context.Context) ([]biz.EvidenceCategory, error) {
	var envelope responseEnvelope[evidenceCategoryListWire]
	err := c.doJSON(ctx, http.MethodGet, "Data.ListEvidenceCategories", evidenceCategoriesPath, evidenceCategoriesPath, nil, &envelope)
	wire, err := unwrapEnvelope(envelope, err)
	if err != nil {
		return nil, classifyReadError(err)
	}
	items, err := wire.toBiz()
	if err != nil {
		return nil, biz.ErrDataServiceUnavailable
	}
	return items, nil
}

func (c *DataHTTPClient) ListSources(ctx context.Context) ([]biz.Source, error) {
	var envelope responseEnvelope[sourceListWire]
	err := c.doJSON(ctx, http.MethodGet, "Data.ListSources", sourcesPath, sourcesPath, nil, &envelope)
	wire, err := unwrapEnvelope(envelope, err)
	if err != nil {
		return nil, classifyReadError(err)
	}
	items, err := wire.toBiz()
	if err != nil {
		return nil, biz.ErrDataServiceUnavailable
	}
	return items, nil
}

func classifyReadError(err error) error {
	var clientError *Error
	if errors.As(err, &clientError) {
		switch clientError.Kind {
		case ErrorKindCanceled:
			return context.Canceled
		case ErrorKindTimeout:
			return context.DeadlineExceeded
		}
	}
	return biz.ErrDataServiceUnavailable
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

func eventListPath(query biz.EventListQuery) string {
	values := url.Values{}
	if query.Title != "" {
		values.Set("title", query.Title)
	}
	if query.Modality != "" {
		values.Set("modality", string(query.Modality))
	}
	if query.Status != "" {
		values.Set("status", string(query.Status))
	}
	setTimeQuery(values, "occurred_from", query.OccurredFrom)
	setTimeQuery(values, "occurred_to", query.OccurredTo)
	setTimeQuery(values, "announced_from", query.AnnouncedFrom)
	setTimeQuery(values, "announced_to", query.AnnouncedTo)
	setPageQuery(values, query.Page, query.PageSize)
	return appendQuery(eventsPath, values)
}

func evidenceListPath(query biz.EvidenceListQuery) string {
	values := url.Values{}
	for name, value := range map[string]string{"title": query.Title, "summary": query.Summary, "category_id": query.CategoryID, "source_name": query.SourceName, "source_level": query.SourceLevel} {
		if value != "" {
			values.Set(name, value)
		}
	}
	if query.IsSplit != nil {
		values.Set("is_split", strconv.FormatBool(*query.IsSplit))
	}
	setTimeQuery(values, "published_from", query.PublishedFrom)
	setTimeQuery(values, "published_to", query.PublishedTo)
	setTimeQuery(values, "collected_from", query.CollectedFrom)
	setTimeQuery(values, "collected_to", query.CollectedTo)
	setPageQuery(values, query.Page, query.PageSize)
	return appendQuery(evidencesPath, values)
}

func setPageQuery(values url.Values, page int, pageSize int) {
	if page != 0 {
		values.Set("page", strconv.Itoa(page))
	}
	if pageSize != 0 {
		values.Set("page_size", strconv.Itoa(pageSize))
	}
}

func setTimeQuery(values url.Values, name string, value *time.Time) {
	if value != nil {
		values.Set(name, value.UTC().Format(time.RFC3339))
	}
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

func (c *DataHTTPClient) doJSON(
	ctx context.Context,
	method string,
	operation string,
	pathTemplate string,
	path string,
	requestBody any,
	responseBody any,
) error {
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
		err = c.doJSONAttempt(operationCtx, method, operation, pathTemplate, path, payload, requestID, responseBody)
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

func (c *DataHTTPClient) doJSONAttempt(
	ctx context.Context,
	method string,
	operation string,
	pathTemplate string,
	path string,
	payload []byte,
	requestID string,
	result any,
) error {
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
		var clientErr *Error
		if errors.As(err, &clientErr) {
			return clientErr
		}
		return transportError(ctx, err)
	}
	return nil
}

func decodeDataSuccessResponse(_ context.Context, response *http.Response, result any) error {
	defer response.Body.Close()
	bodyBytes, err := io.ReadAll(io.LimitReader(response.Body, maxSuccessBodyBytes+1))
	if err != nil {
		return err
	}
	if len(bodyBytes) > maxSuccessBodyBytes || len(bodyBytes) == 0 {
		return &Error{Kind: ErrorKindDecode, RequestID: response.Header.Get(RequestIDHeader)}
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(bodyBytes, result); err != nil {
		return &Error{Kind: ErrorKindDecode, RequestID: response.Header.Get(RequestIDHeader)}
	}
	return nil
}

func decodeDataErrorResponse(_ context.Context, response *http.Response) error {
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		return nil
	}
	defer response.Body.Close()
	bodyBytes, err := io.ReadAll(io.LimitReader(response.Body, maxErrorBodyBytes+1))
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
	if len(body) <= maxErrorBodyBytes {
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

type ErrorKind string

const (
	ErrorKindClient     ErrorKind = "client"
	ErrorKindConflict   ErrorKind = "conflict"
	ErrorKindServer     ErrorKind = "server"
	ErrorKindConnection ErrorKind = "connection"
	ErrorKindTimeout    ErrorKind = "timeout"
	ErrorKindCanceled   ErrorKind = "canceled"
	ErrorKindProtocol   ErrorKind = "protocol"
	ErrorKindEncode     ErrorKind = "encode"
	ErrorKindDecode     ErrorKind = "decode"
)

type Error struct {
	Kind       ErrorKind
	StatusCode int
	Code       string
	RequestID  string
}

func (e *Error) Error() string {
	if e == nil {
		return "data service request failed"
	}
	message := "data service request failed: kind=" + string(e.Kind)
	if e.StatusCode != 0 {
		message += " status=" + strconv.Itoa(e.StatusCode)
	}
	if code := safeMetadata(e.Code, maxErrorCodeLength); code != "" {
		message += " code=" + code
	}
	if requestID := safeMetadata(e.RequestID, 128); requestID != "" {
		message += " request_id=" + requestID
	}
	return message
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
