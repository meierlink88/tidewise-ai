package biz

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type runtimeHealthProviderStub struct {
	get func(context.Context) (ProviderRuntimeHealth, error)
}

func (s runtimeHealthProviderStub) GetRuntimeHealth(ctx context.Context) (ProviderRuntimeHealth, error) {
	return s.get(ctx)
}

func TestRuntimeHealthAggregatesProvidersConcurrentlyInCanonicalOrder(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	started := make(chan string, 2)
	release := make(chan struct{})
	var once sync.Once
	provider := func(name string, services []RuntimeHealthService) runtimeHealthProviderStub {
		return runtimeHealthProviderStub{get: func(context.Context) (ProviderRuntimeHealth, error) {
			started <- name
			<-release
			return ProviderRuntimeHealth{CheckedAt: now, Services: services}, nil
		}}
	}
	service := NewService(nil, nil, WithRuntimeHealthProviders(
		provider("data", []RuntimeHealthService{
			{Key: "data", DisplayName: "Data Service", Status: RuntimeStatusReady, CheckedAt: now},
		}),
		provider("agentrun", []RuntimeHealthService{
			{Key: "agentrun", DisplayName: "AgentRun", Status: RuntimeStatusReady, CheckedAt: now},
			{Key: "qdrant", DisplayName: "Qdrant", Status: RuntimeStatusReady, CheckedAt: now},
		}),
	))

	resultChannel := make(chan RuntimeHealth, 1)
	go func() { resultChannel <- service.GetRuntimeHealth(context.Background()) }()
	first, second := <-started, <-started
	once.Do(func() { close(release) })
	result := <-resultChannel

	if first == second || result.Status != RuntimeStatusReady || len(result.Services) != 3 {
		t.Fatalf("starts=%q/%q result=%#v", first, second, result)
	}
	want := []RuntimeServiceKey{RuntimeServiceData, RuntimeServiceAgentRun, RuntimeServiceQdrant}
	for index, key := range want {
		if result.Services[index].Key != key {
			t.Fatalf("service order = %#v", result.Services)
		}
	}
}

func TestRuntimeHealthReturnsPartialSuccessAsDegraded(t *testing.T) {
	now := time.Now().UTC()
	service := NewService(nil, nil, WithRuntimeHealthProviders(
		runtimeHealthProviderStub{get: func(context.Context) (ProviderRuntimeHealth, error) {
			return ProviderRuntimeHealth{}, &RuntimeHealthProviderError{ReasonCode: RuntimeReasonTimeout}
		}},
		runtimeHealthProviderStub{get: func(context.Context) (ProviderRuntimeHealth, error) {
			return ProviderRuntimeHealth{CheckedAt: now, Services: []RuntimeHealthService{
				{Key: "agentrun", DisplayName: "AgentRun", Status: RuntimeStatusReady, CheckedAt: now},
				{Key: "qdrant", DisplayName: "Qdrant", Status: RuntimeStatusReady, CheckedAt: now},
			}}, nil
		}},
	))

	result := service.GetRuntimeHealth(context.Background())

	if result.Status != RuntimeStatusDegraded || result.Services[0].ReasonCode != RuntimeReasonTimeout {
		t.Fatalf("result = %#v", result)
	}
	if errors.Is(&RuntimeHealthProviderError{ReasonCode: RuntimeReasonTimeout}, context.DeadlineExceeded) {
		t.Fatal("provider error must not expose transport causes")
	}
}
