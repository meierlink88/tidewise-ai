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

func TestHTTPClientListsResearchThemesWithIdentityAndRequestID(t *testing.T) {
	t.Parallel()
	var gotAuthorization, gotRequestID, gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotAuthorization = request.Header.Get("Authorization")
		gotRequestID = request.Header.Get(RequestIDHeader)
		gotQuery = request.URL.RawQuery
		if request.URL.Path != ResearchThemesPath {
			t.Fatalf("path = %q, want %q", request.URL.Path, ResearchThemesPath)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"request_id":"data-req-1","result":{"as_of":"2026-07-17T01:02:03Z","items":[{"id":"11111111-1111-5111-8111-111111111111","name":"theme"}],"next_cursor":null}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, server.Client(), "miniapp-service-token")
	page, err := client.ListResearchThemes(WithRequestID(context.Background(), "req-123"), ResearchListQuery{WindowHours: 24, Limit: 20, Cursor: "cursor value"})
	if err != nil {
		t.Fatal(err)
	}
	if gotAuthorization != "Bearer miniapp-service-token" || gotRequestID != "req-123" {
		t.Fatalf("auth/request ID = %q/%q", gotAuthorization, gotRequestID)
	}
	for _, fragment := range []string{"window_hours=24", "limit=20", "cursor=cursor+value"} {
		if !strings.Contains(gotQuery, fragment) {
			t.Fatalf("query = %q, want %q", gotQuery, fragment)
		}
	}
	if len(page.Items) != 1 || page.Items[0].Name != "theme" || !page.AsOf.Equal(time.Date(2026, 7, 17, 1, 2, 3, 0, time.UTC)) {
		t.Fatalf("page = %#v", page)
	}
}

func TestHTTPClientEscapesResearchDetailID(t *testing.T) {
	t.Parallel()
	var gotPath, gotQuery, gotRequestID string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotPath, gotQuery, gotRequestID = request.URL.EscapedPath(), request.URL.RawQuery, request.Header.Get(RequestIDHeader)
		_, _ = writer.Write([]byte(`{"request_id":"data-req-2","result":{"theme":{"id":"theme/id","name":"detail"},"events":[]}}`))
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, server.Client(), "token")

	result, err := client.GetResearchTheme(context.Background(), "theme/id", ResearchDetailQuery{WindowHours: 48})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != ResearchThemesPath+"/theme%2Fid" || gotQuery != "window_hours=48" || gotRequestID == "" || result.Theme.Name != "detail" {
		t.Fatalf("path/query/request ID/result = %q/%q/%q/%#v", gotPath, gotQuery, gotRequestID, result)
	}
}

func TestHTTPClientRejectsMalformedSuccessEnvelope(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"items":[]}`))
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, server.Client(), "token")

	_, err := client.ListResearchThemes(context.Background(), ResearchListQuery{})
	assertErrorKind(t, err, ErrorKindDecode)
}

func TestHTTPClientRetriesOnlySafeRetryableReads(t *testing.T) {
	t.Parallel()
	var readAttempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodGet {
			if readAttempts.Add(1) == 1 {
				writer.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			_, _ = writer.Write([]byte(`{"request_id":"data-req-3","result":{"items":[]}}`))
			return
		}
		readAttempts.Add(1)
		writer.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, server.Client(), "token")

	if _, err := client.ListResearchAnchors(context.Background(), ResearchListQuery{}); err != nil {
		t.Fatalf("safe read error = %v", err)
	}
	if got := readAttempts.Load(); got != 2 {
		t.Fatalf("safe read attempts = %d, want 2", got)
	}

	readAttempts.Store(0)
	err := client.doJSON(context.Background(), http.MethodPost, ResearchThemesPath, map[string]string{"value": "mutation"}, nil)
	if err == nil {
		t.Fatal("mutation error = nil")
	}
	if got := readAttempts.Load(); got != 1 {
		t.Fatalf("mutation attempts = %d, want 1", got)
	}
}

func TestHTTPClientClassifiesHTTPFailuresWithoutLeakingSecrets(t *testing.T) {
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

			_, err := client.GetResearchAnchor(context.Background(), "11111111-1111-5111-8111-111111111111", ResearchDetailQuery{})
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
	_, err := connectionClient.ListResearchThemes(context.Background(), ResearchListQuery{})
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
	_, err = timeoutClient.ListResearchThemes(context.Background(), ResearchListQuery{})
	assertErrorKind(t, err, ErrorKindTimeout)

	transportTimeoutClient := newTestClient(t, "http://data.invalid", &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("transport secret: %w", context.DeadlineExceeded)
	})}, "token")
	_, err = transportTimeoutClient.ListResearchThemes(context.Background(), ResearchListQuery{})
	assertErrorKind(t, err, ErrorKindTimeout)
	if strings.Contains(err.Error(), "transport secret") {
		t.Fatalf("unsafe timeout error = %q", err)
	}
}

func TestFakeImplementsPort(t *testing.T) {
	t.Parallel()
	fake := &Fake{ListResearchThemesFunc: func(context.Context, ResearchListQuery) (ResearchThemePage, error) {
		return ResearchThemePage{Items: []ResearchTheme{{Name: "fake"}}}, nil
	}}
	var client DataServiceClient = fake
	page, err := client.ListResearchThemes(context.Background(), ResearchListQuery{})
	if err != nil || len(page.Items) != 1 || page.Items[0].Name != "fake" {
		t.Fatalf("fake result/error = %#v/%v", page, err)
	}
	if _, err := client.GetResearchTheme(context.Background(), "id", ResearchDetailQuery{}); !errors.Is(err, ErrFakeMethodNotConfigured) {
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
