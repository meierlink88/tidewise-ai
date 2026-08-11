package biz

import (
	"context"
	"errors"
	"time"
)

const (
	runtimeHealthTotalBudget    = 3 * time.Second
	runtimeHealthProviderBudget = 1500 * time.Millisecond
)

type RuntimeStatus string

const (
	RuntimeStatusReady    RuntimeStatus = "ready"
	RuntimeStatusDegraded RuntimeStatus = "degraded"
	RuntimeStatusDown     RuntimeStatus = "down"
	RuntimeStatusUnknown  RuntimeStatus = "unknown"
)

type RuntimeReasonCode string

const (
	RuntimeReasonTimeout              RuntimeReasonCode = "timeout"
	RuntimeReasonUnreachable          RuntimeReasonCode = "unreachable"
	RuntimeReasonNotReady             RuntimeReasonCode = "not_ready"
	RuntimeReasonCollectionUnhealthy  RuntimeReasonCode = "collection_unhealthy"
	RuntimeReasonAuthenticationFailed RuntimeReasonCode = "authentication_failed"
	RuntimeReasonInvalidResponse      RuntimeReasonCode = "invalid_response"
)

type RuntimeServiceKey string

const (
	RuntimeServiceData     RuntimeServiceKey = "data"
	RuntimeServiceAgentRun RuntimeServiceKey = "agentrun"
	RuntimeServiceQdrant   RuntimeServiceKey = "qdrant"
)

type RuntimeHealthService struct {
	Key         RuntimeServiceKey
	DisplayName string
	Status      RuntimeStatus
	CheckedAt   time.Time
	LatencyMS   *int64
	ReasonCode  RuntimeReasonCode
}

type ProviderRuntimeHealth struct {
	CheckedAt time.Time
	Services  []RuntimeHealthService
}

type RuntimeHealth struct {
	Status    RuntimeStatus
	CheckedAt time.Time
	Services  []RuntimeHealthService
}

type RuntimeHealthProvider interface {
	GetRuntimeHealth(context.Context) (ProviderRuntimeHealth, error)
}

type RuntimeHealthProviderError struct {
	ReasonCode RuntimeReasonCode
}

func (e *RuntimeHealthProviderError) Error() string { return "runtime health provider unavailable" }

type providerResult struct {
	name   string
	health ProviderRuntimeHealth
	err    error
}

func (s *Service) GetRuntimeHealth(ctx context.Context) RuntimeHealth {
	totalContext, cancel := context.WithTimeout(ctx, runtimeHealthTotalBudget)
	defer cancel()
	results := make(chan providerResult, 2)
	go callRuntimeProvider(totalContext, "data", s.dataHealth, results)
	go callRuntimeProvider(totalContext, "agentrun", s.agentRunHealth, results)

	providers := map[string]providerResult{
		"data":     {name: "data", err: &RuntimeHealthProviderError{ReasonCode: RuntimeReasonTimeout}},
		"agentrun": {name: "agentrun", err: &RuntimeHealthProviderError{ReasonCode: RuntimeReasonTimeout}},
	}
collect:
	for received := 0; received < 2; received++ {
		select {
		case result := <-results:
			providers[result.name] = result
		case <-totalContext.Done():
			break collect
		}
	}

	checkedAt := s.now().UTC()
	serviceByKey := make(map[RuntimeServiceKey]RuntimeHealthService, 3)
	mergeProvider(serviceByKey, providers["data"], []RuntimeServiceKey{RuntimeServiceData}, checkedAt)
	mergeProvider(serviceByKey, providers["agentrun"], []RuntimeServiceKey{RuntimeServiceAgentRun, RuntimeServiceQdrant}, checkedAt)
	order := []RuntimeServiceKey{RuntimeServiceData, RuntimeServiceAgentRun, RuntimeServiceQdrant}
	services := make([]RuntimeHealthService, 0, len(order))
	status := RuntimeStatusReady
	for _, key := range order {
		item := serviceByKey[key]
		services = append(services, item)
		if item.Status != RuntimeStatusReady {
			status = RuntimeStatusDegraded
		}
	}
	return RuntimeHealth{Status: status, CheckedAt: checkedAt, Services: services}
}

func callRuntimeProvider(ctx context.Context, name string, provider RuntimeHealthProvider, results chan<- providerResult) {
	if provider == nil {
		results <- providerResult{name: name, err: &RuntimeHealthProviderError{ReasonCode: RuntimeReasonNotReady}}
		return
	}
	providerContext, cancel := context.WithTimeout(ctx, runtimeHealthProviderBudget)
	defer cancel()
	health, err := provider.GetRuntimeHealth(providerContext)
	if err == nil && providerContext.Err() != nil {
		err = providerContext.Err()
	}
	results <- providerResult{name: name, health: health, err: err}
}

func mergeProvider(target map[RuntimeServiceKey]RuntimeHealthService, result providerResult, expected []RuntimeServiceKey, checkedAt time.Time) {
	reason := providerFailureReason(result.err)
	if reason == "" && !validProviderHealth(result.health, expected) {
		reason = RuntimeReasonInvalidResponse
	}
	if reason != "" {
		for _, key := range expected {
			target[key] = RuntimeHealthService{
				Key: key, DisplayName: key.DisplayName(), Status: RuntimeStatusUnknown,
				CheckedAt: checkedAt, ReasonCode: reason,
			}
		}
		return
	}
	for _, item := range result.health.Services {
		target[item.Key] = item
	}
}

func providerFailureReason(err error) RuntimeReasonCode {
	if err == nil {
		return ""
	}
	var providerError *RuntimeHealthProviderError
	if errors.As(err, &providerError) && providerError.ReasonCode.Valid() {
		return providerError.ReasonCode
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return RuntimeReasonTimeout
	}
	return RuntimeReasonUnreachable
}

func validProviderHealth(health ProviderRuntimeHealth, expected []RuntimeServiceKey) bool {
	if health.CheckedAt.IsZero() || len(health.Services) != len(expected) {
		return false
	}
	seen := make(map[RuntimeServiceKey]bool, len(expected))
	for _, item := range health.Services {
		if item.CheckedAt.IsZero() || item.DisplayName != item.Key.DisplayName() || !item.Key.Valid() || !item.Status.Valid() ||
			item.Status == RuntimeStatusReady && item.ReasonCode != "" ||
			item.Status != RuntimeStatusReady && !item.ReasonCode.Valid() {
			return false
		}
		seen[item.Key] = true
	}
	for _, key := range expected {
		if !seen[key] {
			return false
		}
	}
	return true
}

func (status RuntimeStatus) Valid() bool {
	return status == RuntimeStatusReady || status == RuntimeStatusDegraded || status == RuntimeStatusDown || status == RuntimeStatusUnknown
}

func (reason RuntimeReasonCode) Valid() bool {
	return reason == RuntimeReasonTimeout || reason == RuntimeReasonUnreachable || reason == RuntimeReasonNotReady ||
		reason == RuntimeReasonCollectionUnhealthy || reason == RuntimeReasonAuthenticationFailed || reason == RuntimeReasonInvalidResponse
}

func (key RuntimeServiceKey) Valid() bool {
	return key == RuntimeServiceData || key == RuntimeServiceAgentRun || key == RuntimeServiceQdrant
}

func (key RuntimeServiceKey) DisplayName() string {
	switch key {
	case RuntimeServiceData:
		return "Data Service"
	case RuntimeServiceAgentRun:
		return "AgentRun"
	case RuntimeServiceQdrant:
		return "Qdrant"
	default:
		return ""
	}
}
