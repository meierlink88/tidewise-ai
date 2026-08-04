package runtimehealth

import (
	"context"
	"testing"
	"time"
)

type probeStub struct {
	result Check
}

func (p probeStub) Check(context.Context) Check { return p.result }

func TestServiceReturnsDataAndNeo4jInStableSafeOrder(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	service := New(probeStub{result: Check{
		Status: StatusDown, ReasonCode: ReasonAuthenticationFailed, Latency: 12 * time.Millisecond,
	}}, func() time.Time { return now })

	result := service.Get(context.Background())

	if !result.CheckedAt.Equal(now) || len(result.Services) != 2 {
		t.Fatalf("result = %#v", result)
	}
	if result.Services[0].Key != "data" || result.Services[0].Status != StatusReady {
		t.Fatalf("Data status = %#v", result.Services[0])
	}
	if result.Services[1].Key != "neo4j" || result.Services[1].Status != StatusDown ||
		result.Services[1].ReasonCode != ReasonAuthenticationFailed || result.Services[1].Latency != 12*time.Millisecond {
		t.Fatalf("Neo4j status = %#v", result.Services[1])
	}
}

func TestServiceReturnsUnknownWhenNeo4jProbeIsNotConfigured(t *testing.T) {
	result := New(nil, time.Now).Get(context.Background())

	if result.Services[1].Status != StatusUnknown || result.Services[1].ReasonCode != ReasonNotReady {
		t.Fatalf("Neo4j status = %#v", result.Services[1])
	}
}
