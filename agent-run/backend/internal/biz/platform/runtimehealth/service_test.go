package runtimehealth

import (
	"context"
	"errors"
	"testing"
	"time"
)

type readinessStub struct{ err error }

func (s readinessStub) Ready(context.Context) error { return s.err }

type probeStub struct{ check Check }

func (s probeStub) Check(context.Context) Check { return s.check }

func TestServiceReturnsAgentRunAndQdrantInStableOrder(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	service := New(
		readinessStub{err: errors.New("database detail")},
		probeStub{check: Check{Status: StatusDegraded, ReasonCode: ReasonCollectionUnhealthy, Latency: 8 * time.Millisecond}},
		func() time.Time { return now },
	)

	result := service.Get(context.Background())

	if len(result.Services) != 2 || result.Services[0].Key != "agentrun" || result.Services[1].Key != "qdrant" {
		t.Fatalf("services = %#v", result.Services)
	}
	if result.Services[0].Status != StatusDegraded || result.Services[0].ReasonCode != ReasonNotReady {
		t.Fatalf("AgentRun status = %#v", result.Services[0])
	}
	if result.Services[1].Status != StatusDegraded || result.Services[1].ReasonCode != ReasonCollectionUnhealthy {
		t.Fatalf("Qdrant status = %#v", result.Services[1])
	}
}
