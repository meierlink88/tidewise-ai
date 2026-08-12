package dataclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact"
)

const dataAPIPrefix = "/api/data/v1"

type Config struct {
	BaseURL          string
	ServiceToken     string
	Timeout          time.Duration
	MaxResponseBytes int64
	HTTPClient       *http.Client
}

type Client struct {
	baseURL          string
	serviceToken     string
	http             *http.Client
	timeout          time.Duration
	maxResponseBytes int64
}

type envelope[T any] struct {
	RequestID string `json:"request_id"`
	Result    T      `json:"result"`
}

type publicationResult struct {
	ReceiptID    string          `json:"receipt_id"`
	PackageID    string          `json:"package_id"`
	ImportedAt   time.Time       `json:"imported_at"`
	Events       json.RawMessage `json:"events"`
	RawDocuments json.RawMessage `json:"raw_documents"`
	Counts       json.RawMessage `json:"counts"`
}

type tagCatalogWire struct {
	CatalogRevision string          `json:"catalog_revision,omitempty"`
	CatalogHash     string          `json:"catalog_hash,omitempty"`
	Tags            []eventfact.Tag `json:"tags"`
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func New(config Config) (*Client, error) {
	parsed, err := url.Parse(strings.TrimRight(config.BaseURL, "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") ||
		(parsed.Path != "" && parsed.Path != "/") ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("Data Service base URL is invalid")
	}
	if strings.TrimSpace(config.ServiceToken) == "" {
		return nil, errors.New("Data Service token is required")
	}
	if config.Timeout <= 0 {
		return nil, errors.New("Data Service timeout must be positive")
	}
	if config.MaxResponseBytes <= 0 {
		return nil, errors.New("Data Service response limit must be positive")
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: config.Timeout}
	}
	return &Client{
		baseURL: parsed.String(), serviceToken: config.ServiceToken,
		http: httpClient, timeout: config.Timeout,
		maxResponseBytes: config.MaxResponseBytes,
	}, nil
}

func (c *Client) ActiveEventTags(ctx context.Context) (eventfact.TagCatalog, error) {
	var response envelope[tagCatalogWire]
	if err := c.do(ctx, http.MethodGet, dataAPIPrefix+"/event-tags?active=true", nil, &response); err != nil {
		return eventfact.TagCatalog{}, err
	}
	if len(response.Result.Tags) == 0 {
		return eventfact.TagCatalog{}, invalidTagCatalog("Data Service returned an invalid Tag Catalog")
	}
	seenIDs := make(map[string]struct{}, len(response.Result.Tags))
	seenIdentities := make(map[string]struct{}, len(response.Result.Tags))
	for position, tag := range response.Result.Tags {
		if _, err := uuid.Parse(tag.ID); err != nil ||
			tag.ID != strings.TrimSpace(tag.ID) ||
			tag.Kind != strings.TrimSpace(tag.Kind) ||
			tag.Code != strings.TrimSpace(tag.Code) || tag.Code == "" || utf8.RuneCountInString(tag.Code) > 100 ||
			tag.Name != strings.TrimSpace(tag.Name) || tag.Name == "" || utf8.RuneCountInString(tag.Name) > 200 ||
			!tag.IsActive ||
			(tag.Kind != eventfact.TagKindNewsCategory && tag.Kind != eventfact.TagKindIndexCategory) {
			return eventfact.TagCatalog{}, invalidTagCatalog("Data Service returned an invalid Tag Catalog")
		}
		identity := tag.Kind + "\x00" + tag.Code
		if _, exists := seenIDs[tag.ID]; exists {
			return eventfact.TagCatalog{}, invalidTagCatalog("Data Service returned duplicate Event Tags")
		}
		if _, exists := seenIdentities[identity]; exists {
			return eventfact.TagCatalog{}, invalidTagCatalog("Data Service returned duplicate Event Tags")
		}
		seenIDs[tag.ID] = struct{}{}
		seenIdentities[identity] = struct{}{}
		if position > 0 {
			previous := response.Result.Tags[position-1]
			if previous.Kind > tag.Kind ||
				(previous.Kind == tag.Kind && previous.Code > tag.Code) ||
				(previous.Kind == tag.Kind && previous.Code == tag.Code && previous.ID >= tag.ID) {
				return eventfact.TagCatalog{}, invalidTagCatalog("Data Service returned an unstable Tag Catalog")
			}
		}
	}
	return eventfact.TagCatalog{Tags: response.Result.Tags}, nil
}

func invalidTagCatalog(summary string) *eventfact.RemoteError {
	return &eventfact.RemoteError{Code: "invalid_tag_catalog", Summary: summary}
}

func (c *Client) PublishReviewedEvents(ctx context.Context, payload []byte) (string, error) {
	var identity struct {
		PackageID string `json:"package_id"`
	}
	if err := json.Unmarshal(payload, &identity); err != nil || identity.PackageID == "" {
		return "", &eventfact.RemoteError{
			Code: "publication_payload_invalid", Summary: "Stored Event publication payload is invalid",
		}
	}
	var response envelope[publicationResult]
	if err := c.do(ctx, http.MethodPost, dataAPIPrefix+"/reviewed-event-imports", payload, &response); err != nil {
		return "", err
	}
	if response.Result.ReceiptID == "" || response.Result.PackageID != identity.PackageID ||
		response.Result.ImportedAt.IsZero() {
		return "", &eventfact.RemoteError{
			Code: "invalid_publication_receipt", Summary: "Data Service returned an invalid publication receipt",
		}
	}
	return response.Result.ReceiptID, nil
}

func (c *Client) do(ctx context.Context, method, path string, payload []byte, target any) error {
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return &eventfact.RemoteError{Code: "data_request_invalid", Summary: "Data Service request is invalid"}
	}
	request.Header.Set("Authorization", "Bearer "+c.serviceToken)
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.http.Do(request)
	if err != nil {
		return &eventfact.RemoteError{
			Code: "data_transport_unavailable", Summary: "Data Service is unavailable", Retryable: true,
		}
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, c.maxResponseBytes+1))
	if err != nil || int64(len(body)) > c.maxResponseBytes {
		return &eventfact.RemoteError{
			Code: "data_response_invalid", Summary: "Data Service response is unavailable",
			Retryable: response.StatusCode >= 500,
		}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var public errorEnvelope
		_ = json.Unmarshal(body, &public)
		code := public.Error.Code
		summary := public.Error.Message
		if code == "" {
			code = fmt.Sprintf("data_http_%d", response.StatusCode)
		}
		if summary == "" {
			summary = "Data Service rejected the request"
		}
		return &eventfact.RemoteError{
			Code: code, Summary: summary,
			Retryable: response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500,
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil || decoder.Decode(&struct{}{}) == nil {
		return &eventfact.RemoteError{
			Code: "data_response_invalid", Summary: "Data Service response contract is invalid",
		}
	}
	return nil
}

func AsRemoteError(err error) (*eventfact.RemoteError, bool) {
	var target *eventfact.RemoteError
	ok := errors.As(err, &target)
	return target, ok
}
