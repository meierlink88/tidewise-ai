package service

import (
	"context"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/runtimehealth"
)

func TestGetRuntimeHealthConvertsOnlySafeProviderFields(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	handler := NewDataService(Dependencies{RuntimeHealth: runtimehealth.New(nil, func() time.Time { return now })})

	response, err := handler.GetRuntimeHealth(context.Background(), nil)

	if err != nil {
		t.Fatalf("get runtime health: %v", err)
	}
	if len(response.Result.Services) != 2 || response.Result.Services[0].Key != "data" || response.Result.Services[1].Key != "neo4j" {
		t.Fatalf("services = %#v", response.Result.Services)
	}
	if response.Result.Services[1].Status != "unknown" || response.Result.Services[1].ReasonCode != "not_ready" {
		t.Fatalf("Neo4j service = %#v", response.Result.Services[1])
	}
}
