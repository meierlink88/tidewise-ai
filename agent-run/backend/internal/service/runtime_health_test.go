package service

import (
	"context"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform/runtimehealth"
)

type runtimeHealthStub struct{ result runtimehealth.Result }

func (s runtimeHealthStub) Get(context.Context) runtimehealth.Result { return s.result }

func TestGetRuntimeHealthConvertsOnlySafeProviderFields(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	api := &AgentRunService{runtimeHealth: runtimeHealthStub{result: runtimehealth.Result{
		CheckedAt: now,
		Services: []runtimehealth.ServiceStatus{
			{Key: "agentrun", DisplayName: "AgentRun", Status: runtimehealth.StatusReady, CheckedAt: now},
			{Key: "qdrant", DisplayName: "Qdrant", Status: runtimehealth.StatusDegraded, CheckedAt: now, ReasonCode: runtimehealth.ReasonCollectionUnhealthy},
		},
	}}}

	result, err := api.GetRuntimeHealth(context.Background(), nil)

	if err != nil {
		t.Fatal(err)
	}
	if len(result.Services) != 2 || result.Services[1].ReasonCode != "collection_unhealthy" {
		t.Fatalf("result = %#v", result)
	}
}
