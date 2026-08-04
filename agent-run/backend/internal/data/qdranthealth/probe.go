package qdranthealth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform/runtimehealth"
)

const (
	entitySemanticCollection             = "entity_semantic_v1"
	variableDefinitionSemanticCollection = "variable_definition_semantic_v1"
)

type Config struct {
	BaseURL          string
	APIKey           string
	Timeout          time.Duration
	MaxResponseBytes int64
	HTTPClient       *http.Client
}

type Probe struct {
	baseURL, apiKey string
	http            *http.Client
	maxBytes        int64
	now             func() time.Time
}

func New(config Config) (*Probe, error) {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(config.BaseURL), "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.RawQuery != "" || parsed.Fragment != "" ||
		config.Timeout <= 0 || config.MaxResponseBytes <= 0 {
		return nil, errors.New("Qdrant health configuration is invalid")
	}
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: config.Timeout}
	}
	return &Probe{baseURL: parsed.String(), apiKey: config.APIKey, http: client, maxBytes: config.MaxResponseBytes, now: time.Now}, nil
}

func (p *Probe) Check(ctx context.Context) runtimehealth.Check {
	startedAt := p.now()
	requiredCollections := [...]string{entitySemanticCollection, variableDefinitionSemanticCollection}
	results := make(chan runtimehealth.Check, len(requiredCollections))
	for _, collection := range requiredCollections {
		go func(name string) { results <- p.checkCollection(ctx, name) }(collection)
	}
	checks := make([]runtimehealth.Check, 0, len(requiredCollections))
	for range requiredCollections {
		checks = append(checks, <-results)
	}
	result := runtimehealth.Check{Status: runtimehealth.StatusReady}
	for _, check := range checks {
		if severity(check.Status) > severity(result.Status) {
			result = check
		}
	}
	result.Latency = p.now().Sub(startedAt)
	return result
}

func (p *Probe) checkCollection(ctx context.Context, collection string) runtimehealth.Check {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/collections/"+url.PathEscape(collection), nil)
	if err != nil {
		return runtimehealth.Check{Status: runtimehealth.StatusUnknown, ReasonCode: runtimehealth.ReasonInvalidResponse}
	}
	request.Header.Set("Accept", "application/json")
	if p.apiKey != "" {
		request.Header.Set("api-key", p.apiKey)
		request.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	response, err := p.http.Do(request)
	if err != nil {
		if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return runtimehealth.Check{Status: runtimehealth.StatusDown, ReasonCode: runtimehealth.ReasonTimeout}
		}
		return runtimehealth.Check{Status: runtimehealth.StatusDown, ReasonCode: runtimehealth.ReasonUnreachable}
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return runtimehealth.Check{Status: runtimehealth.StatusDown, ReasonCode: runtimehealth.ReasonAuthenticationFailed}
	}
	if response.StatusCode == http.StatusNotFound {
		return runtimehealth.Check{Status: runtimehealth.StatusDegraded, ReasonCode: runtimehealth.ReasonCollectionUnhealthy}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return runtimehealth.Check{Status: runtimehealth.StatusDown, ReasonCode: runtimehealth.ReasonUnreachable}
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, p.maxBytes+1))
	if err != nil || int64(len(body)) > p.maxBytes {
		return runtimehealth.Check{Status: runtimehealth.StatusUnknown, ReasonCode: runtimehealth.ReasonInvalidResponse}
	}
	var envelope struct {
		Result *struct {
			Status          string `json:"status"`
			OptimizerStatus any    `json:"optimizer_status"`
		} `json:"result"`
		Status string `json:"status"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	if err := decoder.Decode(&envelope); err != nil || envelope.Result == nil || envelope.Status != "ok" {
		return runtimehealth.Check{Status: runtimehealth.StatusUnknown, ReasonCode: runtimehealth.ReasonInvalidResponse}
	}
	optimizer, ok := envelope.Result.OptimizerStatus.(string)
	if envelope.Result.Status != "green" || !ok || optimizer != "ok" {
		return runtimehealth.Check{Status: runtimehealth.StatusDegraded, ReasonCode: runtimehealth.ReasonCollectionUnhealthy}
	}
	return runtimehealth.Check{Status: runtimehealth.StatusReady}
}

func severity(status runtimehealth.Status) int {
	switch status {
	case runtimehealth.StatusDown:
		return 3
	case runtimehealth.StatusDegraded:
		return 2
	case runtimehealth.StatusUnknown:
		return 1
	default:
		return 0
	}
}

var _ runtimehealth.Probe = (*Probe)(nil)
