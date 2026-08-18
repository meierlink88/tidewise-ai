package data

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/biz"
)

func TestDataRuntimeHealthUsesSingleStrictProviderRequest(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		if request.URL.Path != "/api/data/v1/runtime-health" || request.Header.Get("Authorization") != "Bearer data-token" {
			t.Fatalf("request = %s %s auth=%q", request.Method, request.URL.Path, request.Header.Get("Authorization"))
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(response, `{"request_id":"req-data","result":{"checked_at":"2026-08-04T10:00:00Z","services":[{"key":"data","display_name":"Data Service","status":"ready","checked_at":"2026-08-04T10:00:00Z"}]}}`)
	}))
	defer server.Close()
	client, err := NewDataHTTPClient(DataHTTPConfig{BaseURL: server.URL, ServiceToken: "data-token", Timeout: time.Second, MaxReadAttempts: 3})
	if err != nil {
		t.Fatal(err)
	}

	result, err := client.GetRuntimeHealth(context.Background())

	if err != nil || calls.Load() != 1 || len(result.Services) != 1 || result.Services[0].Key != biz.RuntimeServiceData {
		t.Fatalf("calls=%d result=%#v err=%v", calls.Load(), result, err)
	}
}
