package dataclient

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact"
)

const dataAPIPrefix = "/api/data/v1"

type Config struct {
	BaseURL          string
	ServiceToken     string
	Timeout          time.Duration
	MaxResponseBytes int64
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
	return &Client{
		baseURL: parsed.String(), serviceToken: config.ServiceToken,
		http: &http.Client{Timeout: config.Timeout}, timeout: config.Timeout,
		maxResponseBytes: config.MaxResponseBytes,
	}, nil
}

func (c *Client) ActiveEventTags(ctx context.Context) (eventfact.TagCatalog, error) {
	var response envelope[eventfact.TagCatalog]
	if err := c.do(ctx, http.MethodGet, dataAPIPrefix+"/event-tags?active=true", nil, &response); err != nil {
		return eventfact.TagCatalog{}, err
	}
	if response.Result.Revision == "" || len(response.Result.Hash) != 64 || len(response.Result.Tags) == 0 {
		return eventfact.TagCatalog{}, &eventfact.RemoteError{
			Code: "invalid_tag_catalog", Summary: "Data Service returned an invalid Tag Catalog",
		}
	}
	for position, tag := range response.Result.Tags {
		if tag.ID == "" || tag.Kind == "" || tag.Code == "" || tag.Name == "" || !tag.IsActive {
			return eventfact.TagCatalog{}, &eventfact.RemoteError{
				Code: "invalid_tag_catalog", Summary: "Data Service returned an invalid Tag Catalog",
			}
		}
		if position > 0 {
			previous := response.Result.Tags[position-1]
			if previous.Kind > tag.Kind ||
				(previous.Kind == tag.Kind && previous.Code > tag.Code) ||
				(previous.Kind == tag.Kind && previous.Code == tag.Code && previous.ID >= tag.ID) {
				return eventfact.TagCatalog{}, &eventfact.RemoteError{
					Code: "invalid_tag_catalog", Summary: "Data Service returned an unstable Tag Catalog",
				}
			}
		}
	}
	encoded, err := json.Marshal(response.Result.Tags)
	if err != nil {
		return eventfact.TagCatalog{}, &eventfact.RemoteError{
			Code: "invalid_tag_catalog", Summary: "Data Service returned an invalid Tag Catalog",
		}
	}
	sum := sha256.Sum256(encoded)
	hash := hex.EncodeToString(sum[:])
	if response.Result.Hash != hash || response.Result.Revision != "event-tags:"+hash {
		return eventfact.TagCatalog{}, &eventfact.RemoteError{
			Code: "invalid_tag_catalog", Summary: "Data Service returned an invalid Tag Catalog identity",
		}
	}
	return response.Result, nil
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
