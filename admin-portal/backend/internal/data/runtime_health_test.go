package data

import (
	"context"
	"errors"
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
		_, _ = fmt.Fprint(response, `{"request_id":"req-data","result":{"checked_at":"2026-08-04T10:00:00Z","services":[{"key":"data","display_name":"Data Service","status":"ready","checked_at":"2026-08-04T10:00:00Z"},{"key":"neo4j","display_name":"Neo4j","status":"down","checked_at":"2026-08-04T10:00:00Z","reason_code":"authentication_failed"}]}}`)
	}))
	defer server.Close()
	client, err := NewDataHTTPClient(DataHTTPConfig{BaseURL: server.URL, ServiceToken: "data-token", Timeout: time.Second, MaxReadAttempts: 3})
	if err != nil {
		t.Fatal(err)
	}

	result, err := client.GetRuntimeHealth(context.Background())

	if err != nil || calls.Load() != 1 || len(result.Services) != 2 || result.Services[1].ReasonCode != biz.RuntimeReasonAuthenticationFailed {
		t.Fatalf("calls=%d result=%#v err=%v", calls.Load(), result, err)
	}
}

func TestAgentRunRuntimeHealthRejectsProviderContractDriftWithoutRetry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = fmt.Fprint(response, `{"request_id":"req-agent","result":{"checked_at":"2026-08-04T10:00:00Z","services":[{"key":"agentrun","display_name":"AgentRun","status":"ready","checked_at":"2026-08-04T10:00:00Z","unexpected":true},{"key":"qdrant","display_name":"Qdrant","status":"ready","checked_at":"2026-08-04T10:00:00Z"}]}}`)
	}))
	defer server.Close()
	client, err := NewAgentRunHTTPClient(AgentRunHTTPConfig{BaseURL: server.URL, ServiceToken: "agent-token", Timeout: time.Second, MaxReadAttempts: 3})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.GetRuntimeHealth(context.Background())

	var providerError *biz.RuntimeHealthProviderError
	if calls.Load() != 1 || !errors.As(err, &providerError) || providerError.ReasonCode != biz.RuntimeReasonInvalidResponse {
		t.Fatalf("calls=%d err=%#v", calls.Load(), err)
	}
}
