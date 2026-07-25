package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz"
	"github.com/meierlink88/tidewise-ai/miniapp/backend/internal/data"
	appservice "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/service"
)

func TestMiniappResearchRequestTraversesKratosDataClient(t *testing.T) {
	const requestID = "miniapp-data-smoke"
	var gotAuthorization, gotRequestID string
	dataService := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotAuthorization = request.Header.Get("Authorization")
		gotRequestID = request.Header.Get(requestIDHeader)
		if request.Method != http.MethodGet || request.URL.Path != data.ResearchThemesPath {
			t.Fatalf("upstream request = %s %s", request.Method, request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"request_id":"data-smoke","result":{"theme_count":0,"event_count":0,"items":[],"next_cursor":null}}`))
	}))
	defer dataService.Close()

	repository, err := data.NewHTTPClient(data.HTTPConfig{
		BaseURL: dataService.URL, ServiceToken: "miniapp-service-token",
		Timeout: time.Second, MaxReadAttempts: 1, HTTPClient: dataService.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	research := appservice.NewResearchService(biz.NewResearchService(repository))
	server := NewHTTPServer(testRuntimeConfig(), research, testLogger())
	request := httptest.NewRequest(http.MethodGet, "/api/miniapp/v1/research/themes", nil)
	request.Header.Set(requestIDHeader, requestID)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if gotAuthorization != "Bearer miniapp-service-token" || gotRequestID != requestID {
		t.Fatalf("upstream auth/request ID = %q/%q", gotAuthorization, gotRequestID)
	}
	var envelope struct {
		RequestID string `json:"request_id"`
		Result    struct {
			ThemeCount int   `json:"theme_count"`
			Items      []any `json:"items"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode Miniapp response: %v", err)
	}
	if envelope.RequestID != requestID || envelope.Result.ThemeCount != 0 || envelope.Result.Items == nil {
		t.Fatalf("Miniapp response = %#v", envelope)
	}
}
