package biz

import (
	"context"
	"errors"
	"time"
)

const runtimeHealthProviderBudget = 1500 * time.Millisecond

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
	RuntimeReasonAuthenticationFailed RuntimeReasonCode = "authentication_failed"
	RuntimeReasonInvalidResponse      RuntimeReasonCode = "invalid_response"
)

type RuntimeServiceKey string

const RuntimeServiceData RuntimeServiceKey = "data"

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

func (s *Service) GetRuntimeHealth(ctx context.Context) RuntimeHealth {
	checkedAt := time.Now().UTC()
	if s != nil && s.now != nil {
		checkedAt = s.now().UTC()
	}
	service := RuntimeHealthService{
		Key: RuntimeServiceData, DisplayName: RuntimeServiceData.DisplayName(),
		Status: RuntimeStatusUnknown, CheckedAt: checkedAt, ReasonCode: RuntimeReasonNotReady,
	}
	if s == nil || s.dataHealth == nil {
		return RuntimeHealth{Status: RuntimeStatusDegraded, CheckedAt: checkedAt, Services: []RuntimeHealthService{service}}
	}

	providerContext, cancel := context.WithTimeout(ctx, runtimeHealthProviderBudget)
	defer cancel()
	health, err := s.dataHealth.GetRuntimeHealth(providerContext)
	if err == nil && providerContext.Err() != nil {
		err = providerContext.Err()
	}
	if reason := providerFailureReason(err); reason != "" {
		service.ReasonCode = reason
		return RuntimeHealth{Status: RuntimeStatusDegraded, CheckedAt: checkedAt, Services: []RuntimeHealthService{service}}
	}
	if !validProviderHealth(health) {
		service.ReasonCode = RuntimeReasonInvalidResponse
		return RuntimeHealth{Status: RuntimeStatusDegraded, CheckedAt: checkedAt, Services: []RuntimeHealthService{service}}
	}
	service = health.Services[0]
	status := RuntimeStatusReady
	if service.Status != RuntimeStatusReady {
		status = RuntimeStatusDegraded
	}
	return RuntimeHealth{Status: status, CheckedAt: checkedAt, Services: []RuntimeHealthService{service}}
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

func validProviderHealth(health ProviderRuntimeHealth) bool {
	return len(health.Services) == 1 && health.Services[0].Key == RuntimeServiceData &&
		health.Services[0].DisplayName == RuntimeServiceData.DisplayName() &&
		health.Services[0].Status.Valid() && health.Services[0].CheckedAt.Equal(health.CheckedAt) &&
		(health.Services[0].ReasonCode == "" || health.Services[0].ReasonCode.Valid())
}

func (status RuntimeStatus) Valid() bool {
	return status == RuntimeStatusReady || status == RuntimeStatusDegraded ||
		status == RuntimeStatusDown || status == RuntimeStatusUnknown
}

func (reason RuntimeReasonCode) Valid() bool {
	return reason == RuntimeReasonTimeout || reason == RuntimeReasonUnreachable ||
		reason == RuntimeReasonNotReady || reason == RuntimeReasonAuthenticationFailed ||
		reason == RuntimeReasonInvalidResponse
}

func (key RuntimeServiceKey) Valid() bool { return key == RuntimeServiceData }

func (key RuntimeServiceKey) DisplayName() string {
	if key == RuntimeServiceData {
		return "Data Service"
	}
	return ""
}
