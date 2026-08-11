package service

import (
	"context"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/runtimehealth"
)

func TestGetRuntimeHealthConvertsOnlySafeProviderFields(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	handler := NewDataService(Dependencies{RuntimeHealth: runtimehealth.New(func() time.Time { return now })})

	response, err := handler.GetRuntimeHealth(context.Background(), nil)

	if err != nil {
		t.Fatalf("get runtime health: %v", err)
	}
	if len(response.Result.Services) != 1 || response.Result.Services[0].Key != "data" {
		t.Fatalf("services = %#v", response.Result.Services)
	}
}
