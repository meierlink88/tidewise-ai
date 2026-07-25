package dataclient

import (
	"bytes"
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
	httpClient      *http.Client
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
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &HTTPClient{
		baseURL:         parsed.Scheme + "://" + parsed.Host,
		serviceToken:    token,
		timeout:         config.Timeout,
		maxReadAttempts: attempts,
		httpClient:      httpClient,
	}, nil
}

func (c *HTTPClient) ListRawDocuments(ctx context.Context, query RawDocumentListQuery) (RawDocumentPage, error) {
	var envelope responseEnvelope[RawDocumentPage]
	err := c.doJSON(ctx, http.MethodGet, rawDocumentListPath(query), nil, &envelope)
	return unwrapEnvelope(envelope, err)
}

func (c *HTTPClient) ListEvents(ctx context.Context, query EventListQuery) (EventPage, error) {
	var envelope responseEnvelope[EventPage]
	err := c.doJSON(ctx, http.MethodGet, eventListPath(query), nil, &envelope)
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

func rawDocumentListPath(query RawDocumentListQuery) string {
	values := url.Values{}
	if query.Title != "" {
		values.Set("title", query.Title)
	}
	if query.SourceRef != "" {
		values.Set("source_ref", query.SourceRef)
	}
	if query.IngestStatus != "" {
		values.Set("ingest_status", string(query.IngestStatus))
	}
	setPageQuery(values, query.Page, query.PageSize)
	return appendQuery(AdminRawDocumentsPath, values)
}

func eventListPath(query EventListQuery) string {
	values := url.Values{}
	if query.Title != "" {
		values.Set("title", query.Title)
	}
	if query.EventStatus != "" {
		values.Set("event_status", string(query.EventStatus))
	}
	if query.FactStatus != "" {
		values.Set("fact_status", string(query.FactStatus))
	}
	setTimeQuery(values, "event_time_from", query.EventTimeFrom)
	setTimeQuery(values, "event_time_to", query.EventTimeTo)
	setTimeQuery(values, "first_seen_from", query.FirstSeenFrom)
	setTimeQuery(values, "first_seen_to", query.FirstSeenTo)
	setPageQuery(values, query.Page, query.PageSize)
	return appendQuery(AdminEventsPath, values)
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
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return &Error{Kind: ErrorKindProtocol}
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+c.serviceToken)
	request.Header.Set(RequestIDHeader, requestID)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return transportError(ctx, err)
	}
	bodyBytes, readErr := io.ReadAll(io.LimitReader(response.Body, maxResponseBodyBytes+1))
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil {
		if readErr != nil {
			return transportError(ctx, readErr)
		}
		return transportError(ctx, closeErr)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return httpStatusError(response.StatusCode, response.Header.Get(RequestIDHeader), bodyBytes)
	}
	if len(bodyBytes) > maxResponseBodyBytes || len(bodyBytes) == 0 {
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
