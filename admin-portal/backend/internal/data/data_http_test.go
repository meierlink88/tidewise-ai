package data

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/biz"
)

func TestHTTPClientListsAdminDataWithIdentityRequestIDAndTypedQueries(t *testing.T) {
	t.Parallel()
	requests := make(chan *http.Request, 2)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests <- request.Clone(context.Background())
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case rawDocumentsPath:
			_, _ = writer.Write([]byte(`{"request_id":"data-req-1","result":{"items":[{"id":"11111111-1111-5111-8111-111111111111","contract_version":2,"artifact_id":"artifact-1","source_ref":"source:reuters:world","title":"raw","content_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","collected_at":"2026-07-17T01:02:03Z","ingest_status":"collected"}],"total":1,"page":2,"page_size":10}}`))
		case eventsPath:
			_, _ = writer.Write([]byte(`{"request_id":"data-req-2","result":{"items":[{"id":"22222222-2222-5222-8222-222222222222","title":"event","first_seen_at":"2026-07-17T01:02:03Z","event_status":"confirmed","fact_status":"verified"}],"total":1,"page":1,"page_size":20}}`))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, server.Client(), "admin-service-token")
	ctx := WithRequestID(context.Background(), "admin-req-123")

	rawPage, err := client.ListRawDocuments(ctx, biz.RawDocumentListQuery{Title: "央行 data", SourceRef: "source:reuters:world", IngestStatus: "collected", Page: 2, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	eventFrom := time.Date(2026, 7, 1, 2, 3, 4, 0, time.UTC)
	eventPage, err := client.ListEvents(ctx, biz.EventListQuery{Title: "event title", EventStatus: "confirmed", FactStatus: "verified", EventTimeFrom: &eventFrom, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(rawPage.Items) != 1 || rawPage.Items[0].Title != "raw" || rawPage.Items[0].ContractVersion != 2 || rawPage.Items[0].ArtifactID != "artifact-1" || rawPage.Items[0].SourceRef != "source:reuters:world" || rawPage.Page != 2 || !rawPage.Items[0].CollectedAt.Equal(time.Date(2026, 7, 17, 1, 2, 3, 0, time.UTC)) {
		t.Fatalf("raw page = %#v", rawPage)
	}
	if len(eventPage.Items) != 1 || eventPage.Items[0].EventStatus != "confirmed" {
		t.Fatalf("events = %#v", eventPage)
	}

	rawRequest, eventRequest := <-requests, <-requests
	for _, request := range []*http.Request{rawRequest, eventRequest} {
		if request.Header.Get("Authorization") != "Bearer admin-service-token" || request.Header.Get(RequestIDHeader) != "admin-req-123" {
			t.Fatalf("auth/request ID for %q = %q/%q", request.URL.Path, request.Header.Get("Authorization"), request.Header.Get(RequestIDHeader))
		}
	}
	if rawRequest.URL.Path != rawDocumentsPath || rawRequest.URL.Query().Get("title") != "央行 data" || rawRequest.URL.Query().Get("source_ref") != "source:reuters:world" || rawRequest.URL.Query().Get("ingest_status") != "collected" || rawRequest.URL.Query().Get("page") != "2" || rawRequest.URL.Query().Get("page_size") != "10" {
		t.Fatalf("raw request = %s?%s", rawRequest.URL.Path, rawRequest.URL.RawQuery)
	}
	if eventRequest.URL.Path != eventsPath || eventRequest.URL.Query().Get("event_time_from") != eventFrom.Format(time.RFC3339) || eventRequest.URL.Query().Get("event_status") != "confirmed" {
		t.Fatalf("event request = %s?%s", eventRequest.URL.Path, eventRequest.URL.RawQuery)
	}
}

func TestHTTPClientRetriesOnlySafeRetryableReads(t *testing.T) {
	t.Parallel()
	var attempts atomic.Int32
	var gotRequestID string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotRequestID = request.Header.Get(RequestIDHeader)
		if request.Method == http.MethodGet {
			if attempts.Add(1) == 1 {
				writer.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			_, _ = writer.Write([]byte(`{"request_id":"data-req-4","result":{"items":[],"total":0,"page":1,"page_size":50}}`))
			return
		}
		attempts.Add(1)
		writer.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, server.Client(), "token")

	if _, err := client.ListRawDocuments(context.Background(), biz.RawDocumentListQuery{}); err != nil {
		t.Fatalf("safe read error = %v", err)
	}
	if got := attempts.Load(); got != 2 {
		t.Fatalf("safe read attempts = %d, want 2", got)
	}
	if gotRequestID == "" {
		t.Fatal("generated request ID is empty")
	}

	attempts.Store(0)
	err := client.doJSON(
		context.Background(), http.MethodPost, "Data.TestMutation", rawDocumentsPath,
		rawDocumentsPath, map[string]string{"value": "mutation"}, nil,
	)
	if err == nil {
		t.Fatal("mutation error = nil")
	}
	if got := attempts.Load(); got != 1 {
		t.Fatalf("mutation attempts = %d, want 1", got)
	}
}

func TestHTTPClientRejectsMalformedSuccessEnvelope(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"items":[]}`))
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, server.Client(), "token")

	_, err := client.ListEvents(context.Background(), biz.EventListQuery{})
	assertDataUnavailable(t, err)
}

func TestHTTPClientClassifiesFailuresWithoutLeakingSecrets(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		status int
		kind   ErrorKind
	}{
		{name: "client", status: http.StatusBadRequest, kind: ErrorKindClient},
		{name: "conflict", status: http.StatusConflict, kind: ErrorKindConflict},
		{name: "server", status: http.StatusInternalServerError, kind: ErrorKindServer},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var attempts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				attempts.Add(1)
				writer.Header().Set(RequestIDHeader, "response-request-id")
				writer.WriteHeader(test.status)
				_, _ = writer.Write([]byte(`{"error":{"code":"UPSTREAM_CODE","message":"secret-response-body"}}`))
			}))
			defer server.Close()
			client := newTestClient(t, server.URL, server.Client(), "secret-service-token")

			_, err := client.ListRawDocuments(context.Background(), biz.RawDocumentListQuery{})
			if !errors.Is(err, biz.ErrDataServiceUnavailable) {
				t.Fatalf("error = %#v", err)
			}
			if strings.Contains(err.Error(), "secret-service-token") || strings.Contains(err.Error(), "secret-response-body") {
				t.Fatalf("unsafe error = %q", err)
			}
			wantAttempts := int32(1)
			if test.status >= 500 {
				wantAttempts = 2
			}
			if attempts.Load() != wantAttempts {
				t.Fatalf("attempts = %d, want %d", attempts.Load(), wantAttempts)
			}
		})
	}
}

func TestHTTPClientClassifiesConnectionFailureAndDeadline(t *testing.T) {
	t.Parallel()
	var connectionAttempts atomic.Int32
	connectionClient := newTestClient(t, "http://data.invalid", &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		connectionAttempts.Add(1)
		return nil, fmt.Errorf("dial failed with secret-service-token")
	})}, "secret-service-token")
	_, err := connectionClient.ListEvents(context.Background(), biz.EventListQuery{})
	assertDataUnavailable(t, err)
	if connectionAttempts.Load() != 2 || strings.Contains(err.Error(), "secret-service-token") {
		t.Fatalf("connection attempts/error = %d/%q", connectionAttempts.Load(), err)
	}

	timeoutClient, err := NewDataHTTPClient(DataHTTPConfig{
		BaseURL:         "http://data.invalid",
		ServiceToken:    "token",
		Timeout:         10 * time.Millisecond,
		MaxReadAttempts: 2,
		HTTPClient: &http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
			<-request.Context().Done()
			return nil, request.Context().Err()
		})},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = timeoutClient.ListEvents(context.Background(), biz.EventListQuery{})
	assertDataUnavailable(t, err)

	transportTimeoutClient := newTestClient(t, "http://data.invalid", &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("transport secret: %w", context.DeadlineExceeded)
	})}, "token")
	_, err = transportTimeoutClient.ListEvents(context.Background(), biz.EventListQuery{})
	assertDataUnavailable(t, err)
	if strings.Contains(err.Error(), "transport secret") {
		t.Fatalf("unsafe timeout error = %q", err)
	}
}

func newTestClient(t *testing.T, baseURL string, httpClient *http.Client, token string) *DataHTTPClient {
	t.Helper()
	client, err := NewDataHTTPClient(DataHTTPConfig{BaseURL: baseURL, ServiceToken: token, Timeout: time.Second, MaxReadAttempts: 2, HTTPClient: httpClient})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func assertDataUnavailable(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, biz.ErrDataServiceUnavailable) {
		t.Fatalf("error = %#v, want data service unavailable", err)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}
