package runtimehealth

import (
	"context"
	"testing"
	"time"

	runtimehealthbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/runtimehealth"
)

func TestGetRuntimeHealthConvertsOnlySafeProviderFields(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	handler, err := NewService(runtimehealthbiz.New(func() time.Time { return now }))
	if err != nil {
		t.Fatal(err)
	}

	response, err := handler.GetRuntimeHealth(context.Background(), nil)

	if err != nil {
		t.Fatalf("get runtime health: %v", err)
	}
	if len(response.Result.Services) != 1 || response.Result.Services[0].Key != "data" {
		t.Fatalf("services = %#v", response.Result.Services)
	}
}
