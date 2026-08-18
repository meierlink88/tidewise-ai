package biz

import (
	"context"
	"testing"
	"time"
)

type runtimeHealthProviderStub struct {
	get func(context.Context) (ProviderRuntimeHealth, error)
}

func (s runtimeHealthProviderStub) GetRuntimeHealth(ctx context.Context) (ProviderRuntimeHealth, error) {
	return s.get(ctx)
}

func TestRuntimeHealthReturnsDataServiceStatus(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	provider := runtimeHealthProviderStub{get: func(context.Context) (ProviderRuntimeHealth, error) {
		return ProviderRuntimeHealth{CheckedAt: now, Services: []RuntimeHealthService{{
			Key: RuntimeServiceData, DisplayName: RuntimeServiceData.DisplayName(),
			Status: RuntimeStatusReady, CheckedAt: now,
		}}}, nil
	}}
	service := NewService(nil, WithRuntimeHealthProvider(provider))

	result := service.GetRuntimeHealth(context.Background())

	if result.Status != RuntimeStatusReady || len(result.Services) != 1 || result.Services[0].Key != RuntimeServiceData {
		t.Fatalf("result = %#v", result)
	}
}

func TestRuntimeHealthReturnsSafeDegradedStatusWhenDataServiceIsUnavailable(t *testing.T) {
	service := NewService(nil, WithRuntimeHealthProvider(runtimeHealthProviderStub{get: func(context.Context) (ProviderRuntimeHealth, error) {
		return ProviderRuntimeHealth{}, &RuntimeHealthProviderError{ReasonCode: RuntimeReasonTimeout}
	}}))

	result := service.GetRuntimeHealth(context.Background())

	if result.Status != RuntimeStatusDegraded || len(result.Services) != 1 ||
		result.Services[0].Key != RuntimeServiceData || result.Services[0].ReasonCode != RuntimeReasonTimeout {
		t.Fatalf("result = %#v", result)
	}
}
