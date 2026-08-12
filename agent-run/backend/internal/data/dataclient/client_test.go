package dataclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientReadsTagCatalogAndPublishesExactBytes(t *testing.T) {
	payload := []byte(`{"package_id":"package-1"}`)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer service-token" {
			t.Fatalf("Authorization = %q", request.Header.Get("Authorization"))
		}
		switch request.URL.Path {
		case "/api/data/v1/event-tags":
			if request.Method != http.MethodGet || request.URL.RawQuery != "active=true" {
				t.Fatalf("Tag Catalog request = %s %s?%s", request.Method, request.URL.Path, request.URL.RawQuery)
			}
			response.Header().Set("Content-Type", "application/json")
			_, _ = response.Write([]byte(`{"request_id":"data-1","result":{"tags":[{"id":"11111111-1111-4111-8111-111111111111","tag_kind":"news_category","code":"technology","name":"科技","is_active":true}]}}`))
		case "/api/data/v1/reviewed-event-imports":
			body, _ := io.ReadAll(request.Body)
			if string(body) != string(payload) {
				t.Fatalf("publication bytes = %s", body)
			}
			response.WriteHeader(http.StatusCreated)
			_, _ = response.Write([]byte(`{"request_id":"data-2","result":{"receipt_id":"receipt-1","package_id":"package-1","imported_at":"2026-07-26T00:00:00Z","events":[],"raw_documents":[],"counts":{"events_created":0,"events_reused":0,"raw_documents_created":0,"raw_documents_reused":0,"event_sources_created":0,"event_sources_reused":0,"event_tags_created":0,"event_tags_reused":0}}}`))
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	client, err := New(Config{
		BaseURL: server.URL, ServiceToken: "service-token",
		Timeout: time.Second, MaxResponseBytes: 64 * 1024,
	})
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := client.ActiveEventTags(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Tags) != 1 || catalog.Tags[0].Code != "technology" {
		t.Fatalf("catalog = %#v", catalog)
	}
	receipt, err := client.PublishReviewedEvents(context.Background(), payload)
	if err != nil {
		t.Fatal(err)
	}
	if receipt != "receipt-1" {
		t.Fatalf("receipt = %q", receipt)
	}
}

func TestClientRejectsMalformedCurrentTagCollections(t *testing.T) {
	for _, test := range []struct {
		name   string
		result string
	}{
		{name: "empty", result: `{"tags":[]}`},
		{name: "inactive", result: `{"tags":[{"id":"11111111-1111-4111-8111-111111111111","tag_kind":"news_category","code":"technology","name":"科技","is_active":false}]}`},
		{name: "unknown kind", result: `{"tags":[{"id":"11111111-1111-4111-8111-111111111111","tag_kind":"other","code":"technology","name":"科技","is_active":true}]}`},
		{name: "unstable order", result: `{"tags":[{"id":"11111111-1111-4111-8111-111111111111","tag_kind":"news_category","code":"z","name":"Z","is_active":true},{"id":"22222222-2222-4222-8222-222222222222","tag_kind":"news_category","code":"a","name":"A","is_active":true}]}`},
		{name: "duplicate identity", result: `{"tags":[{"id":"11111111-1111-4111-8111-111111111111","tag_kind":"news_category","code":"technology","name":"科技","is_active":true},{"id":"22222222-2222-4222-8222-222222222222","tag_kind":"news_category","code":"technology","name":"科技产业","is_active":true}]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.Header().Set("Content-Type", "application/json")
				_, _ = response.Write([]byte(`{"request_id":"data-1","result":` + test.result + `}`))
			}))
			defer server.Close()

			client, err := New(Config{
				BaseURL: server.URL, ServiceToken: "service-token",
				Timeout: time.Second, MaxResponseBytes: 64 * 1024,
			})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.ActiveEventTags(context.Background())
			remote, ok := AsRemoteError(err)
			if !ok || remote.Code != "invalid_tag_catalog" {
				t.Fatalf("error = %#v, want invalid_tag_catalog", err)
			}
		})
	}
}

func TestClientClassifiesRetryableAndBlockedResponses(t *testing.T) {
	for _, test := range []struct {
		name      string
		status    int
		retryable bool
	}{
		{name: "too many requests", status: http.StatusTooManyRequests, retryable: true},
		{name: "server failure", status: http.StatusBadGateway, retryable: true},
		{name: "unauthorized", status: http.StatusUnauthorized, retryable: false},
		{name: "conflict", status: http.StatusConflict, retryable: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.WriteHeader(test.status)
				_, _ = response.Write([]byte(`{"error":{"code":"safe_code","message":"safe message","details":{}}}`))
			}))
			defer server.Close()
			client, err := New(Config{
				BaseURL: server.URL, ServiceToken: "token",
				Timeout: time.Second, MaxResponseBytes: 4096,
			})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.PublishReviewedEvents(context.Background(), []byte(`{"package_id":"package-1"}`))
			remote, ok := AsRemoteError(err)
			if !ok || remote.Retryable != test.retryable || !strings.Contains(remote.Error(), "safe_code") {
				t.Fatalf("remote error = %#v, ok=%v", remote, ok)
			}
		})
	}
}
