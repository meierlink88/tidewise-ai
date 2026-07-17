package dataclient

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
)

func TestHTTPClientListsAdminDataWithIdentityRequestIDAndTypedQueries(t *testing.T) {
	t.Parallel()
	requests := make(chan *http.Request, 3)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests <- request.Clone(context.Background())
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case AdminRawDocumentsPath:
			_, _ = writer.Write([]byte(`{"request_id":"data-req-1","result":{"items":[{"id":"11111111-1111-5111-8111-111111111111","title":"raw","content_level":"full","collected_at":"2026-07-17T01:02:03Z"}],"total":1,"page":2,"page_size":10}}`))
		case AdminEventsPath:
			_, _ = writer.Write([]byte(`{"request_id":"data-req-2","result":{"items":[{"id":"22222222-2222-5222-8222-222222222222","title":"event","first_seen_at":"2026-07-17T01:02:03Z","event_status":"confirmed","fact_status":"verified"}],"total":1,"page":1,"page_size":20}}`))
		case AdminSourceCatalogsPath:
			_, _ = writer.Write([]byte(`{"request_id":"data-req-3","result":{"items":[{"id":"source-1","source_name":"source","parser_key":"rss_item","status":"inactive"}]}}`))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, server.Client(), "admin-service-token")
	ctx := WithRequestID(context.Background(), "admin-req-123")

	rawPage, err := client.ListRawDocuments(ctx, RawDocumentListQuery{Title: "央行 data", SourceID: "source-id", IngestStatus: "collected", Page: 2, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	eventFrom := time.Date(2026, 7, 1, 2, 3, 4, 0, time.UTC)
	eventPage, err := client.ListEvents(ctx, EventListQuery{Title: "event title", EventStatus: "confirmed", FactStatus: "verified", EventTimeFrom: &eventFrom, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	sources, err := client.ListSourceCatalogs(ctx, SourceCatalogListQuery{Status: "inactive"})
	if err != nil {
		t.Fatal(err)
	}

	if len(rawPage.Items) != 1 || rawPage.Items[0].Title != "raw" || rawPage.Items[0].ContentLevel != "full" || rawPage.Page != 2 || !rawPage.Items[0].CollectedAt.Equal(time.Date(2026, 7, 17, 1, 2, 3, 0, time.UTC)) {
		t.Fatalf("raw page = %#v", rawPage)
	}
	if len(eventPage.Items) != 1 || eventPage.Items[0].EventStatus != "confirmed" || len(sources.Items) != 1 || sources.Items[0].Status != "inactive" || sources.Items[0].ParserKey != "rss_item" {
		t.Fatalf("events/sources = %#v/%#v", eventPage, sources)
	}

	rawRequest, eventRequest, sourceRequest := <-requests, <-requests, <-requests
	for _, request := range []*http.Request{rawRequest, eventRequest, sourceRequest} {
		if request.Header.Get("Authorization") != "Bearer admin-service-token" || request.Header.Get(RequestIDHeader) != "admin-req-123" {
			t.Fatalf("auth/request ID for %q = %q/%q", request.URL.Path, request.Header.Get("Authorization"), request.Header.Get(RequestIDHeader))
		}
	}
	if rawRequest.URL.Path != AdminRawDocumentsPath || rawRequest.URL.Query().Get("title") != "央行 data" || rawRequest.URL.Query().Get("source_id") != "source-id" || rawRequest.URL.Query().Get("ingest_status") != "collected" || rawRequest.URL.Query().Get("page") != "2" || rawRequest.URL.Query().Get("page_size") != "10" {
		t.Fatalf("raw request = %s?%s", rawRequest.URL.Path, rawRequest.URL.RawQuery)
	}
	if eventRequest.URL.Path != AdminEventsPath || eventRequest.URL.Query().Get("event_time_from") != eventFrom.Format(time.RFC3339) || eventRequest.URL.Query().Get("event_status") != "confirmed" {
		t.Fatalf("event request = %s?%s", eventRequest.URL.Path, eventRequest.URL.RawQuery)
	}
	if sourceRequest.URL.Path != AdminSourceCatalogsPath || sourceRequest.URL.Query().Get("status") != "inactive" {
		t.Fatalf("source request = %s?%s", sourceRequest.URL.Path, sourceRequest.URL.RawQuery)
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
			_, _ = writer.Write([]byte(`{"request_id":"data-req-4","result":{"items":[]}}`))
			return
		}
		attempts.Add(1)
		writer.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, server.Client(), "token")

	if _, err := client.ListSourceCatalogs(context.Background(), SourceCatalogListQuery{}); err != nil {
		t.Fatalf("safe read error = %v", err)
	}
	if got := attempts.Load(); got != 2 {
		t.Fatalf("safe read attempts = %d, want 2", got)
	}
	if gotRequestID == "" {
		t.Fatal("generated request ID is empty")
	}

	attempts.Store(0)
	err := client.doJSON(context.Background(), http.MethodPost, AdminRawDocumentsPath, map[string]string{"value": "mutation"}, nil)
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

	_, err := client.ListEvents(context.Background(), EventListQuery{})
	assertErrorKind(t, err, ErrorKindDecode)
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

			_, err := client.ListRawDocuments(context.Background(), RawDocumentListQuery{})
			var clientErr *Error
			if !errors.As(err, &clientErr) || clientErr.Kind != test.kind || clientErr.StatusCode != test.status || clientErr.Code != "UPSTREAM_CODE" || clientErr.RequestID != "response-request-id" {
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
	_, err := connectionClient.ListEvents(context.Background(), EventListQuery{})
	assertErrorKind(t, err, ErrorKindConnection)
	if connectionAttempts.Load() != 2 || strings.Contains(err.Error(), "secret-service-token") {
		t.Fatalf("connection attempts/error = %d/%q", connectionAttempts.Load(), err)
	}

	timeoutClient, err := NewHTTPClient(HTTPConfig{
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
	_, err = timeoutClient.ListEvents(context.Background(), EventListQuery{})
	assertErrorKind(t, err, ErrorKindTimeout)

	transportTimeoutClient := newTestClient(t, "http://data.invalid", &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("transport secret: %w", context.DeadlineExceeded)
	})}, "token")
	_, err = transportTimeoutClient.ListEvents(context.Background(), EventListQuery{})
	assertErrorKind(t, err, ErrorKindTimeout)
	if strings.Contains(err.Error(), "transport secret") {
		t.Fatalf("unsafe timeout error = %q", err)
	}
}

func TestFakeImplementsPort(t *testing.T) {
	t.Parallel()
	fake := &Fake{ListRawDocumentsFunc: func(context.Context, RawDocumentListQuery) (RawDocumentPage, error) {
		return RawDocumentPage{Items: []RawDocument{{Title: "fake"}}}, nil
	}}
	var client DataServiceClient = fake
	page, err := client.ListRawDocuments(context.Background(), RawDocumentListQuery{})
	if err != nil || len(page.Items) != 1 || page.Items[0].Title != "fake" {
		t.Fatalf("fake result/error = %#v/%v", page, err)
	}
	if _, err := client.ListEvents(context.Background(), EventListQuery{}); !errors.Is(err, ErrFakeMethodNotConfigured) {
		t.Fatalf("unconfigured fake error = %v", err)
	}
}

func newTestClient(t *testing.T, baseURL string, httpClient *http.Client, token string) *HTTPClient {
	t.Helper()
	client, err := NewHTTPClient(HTTPConfig{BaseURL: baseURL, ServiceToken: token, Timeout: time.Second, MaxReadAttempts: 2, HTTPClient: httpClient})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func assertErrorKind(t *testing.T, err error, want ErrorKind) {
	t.Helper()
	var clientErr *Error
	if !errors.As(err, &clientErr) || clientErr.Kind != want {
		t.Fatalf("error = %#v, want kind %q", err, want)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}
