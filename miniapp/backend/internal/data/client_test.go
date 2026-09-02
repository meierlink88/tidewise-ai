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
)

const testDataPath = DataAPIPrefix + "/test-resource"

type testPayload struct {
	Value string `json:"value"`
}

func TestHTTPClientAddsIdentityAndRequestID(t *testing.T) {
	t.Parallel()
	var gotAuthorization, gotRequestID string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotAuthorization = request.Header.Get("Authorization")
		gotRequestID = request.Header.Get(RequestIDHeader)
		if request.Method != http.MethodGet || request.URL.Path != testDataPath {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"request_id":"data-req-1","result":{"value":"ok"}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, server.Client(), "miniapp-service-token")
	var envelope responseEnvelope[testPayload]
	err := client.doJSON(WithRequestID(context.Background(), "req-123"), http.MethodGet, testDataPath, nil, &envelope)
	value, err := unwrapEnvelope(envelope, err)
	if err != nil {
		t.Fatal(err)
	}
	if gotAuthorization != "Bearer miniapp-service-token" || gotRequestID != "req-123" || value.Value != "ok" {
		t.Fatalf("auth/request ID/value = %q/%q/%q", gotAuthorization, gotRequestID, value.Value)
	}
}

func TestHTTPClientRejectsMalformedSuccessEnvelope(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"value":"missing-envelope"}`))
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, server.Client(), "token")

	var envelope responseEnvelope[testPayload]
	err := client.doJSON(context.Background(), http.MethodGet, testDataPath, nil, &envelope)
	_, err = unwrapEnvelope(envelope, err)
	assertErrorKind(t, err, ErrorKindDecode)
}

func TestHTTPClientRetriesOnlySafeRetryableReads(t *testing.T) {
	t.Parallel()
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodGet {
			if attempts.Add(1) == 1 {
				writer.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			writer.WriteHeader(http.StatusOK)
			return
		}
		attempts.Add(1)
		writer.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, server.Client(), "token")

	if err := client.doJSON(context.Background(), http.MethodGet, testDataPath, nil, nil); err != nil {
		t.Fatalf("safe read error = %v", err)
	}
	if got := attempts.Load(); got != 2 {
		t.Fatalf("safe read attempts = %d, want 2", got)
	}

	attempts.Store(0)
	err := client.doJSON(context.Background(), http.MethodPost, testDataPath, map[string]string{"value": "mutation"}, nil)
	if err == nil {
		t.Fatal("mutation error = nil")
	}
	if got := attempts.Load(); got != 1 {
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

			err := client.doJSON(context.Background(), http.MethodGet, testDataPath, nil, nil)
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
	err := connectionClient.doJSON(context.Background(), http.MethodGet, testDataPath, nil, nil)
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
	err = timeoutClient.doJSON(context.Background(), http.MethodGet, testDataPath, nil, nil)
	assertErrorKind(t, err, ErrorKindTimeout)

	transportTimeoutClient := newTestClient(t, "http://data.invalid", &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("transport secret: %w", context.DeadlineExceeded)
	})}, "token")
	err = transportTimeoutClient.doJSON(context.Background(), http.MethodGet, testDataPath, nil, nil)
	assertErrorKind(t, err, ErrorKindTimeout)
	if strings.Contains(err.Error(), "transport secret") {
		t.Fatalf("unsafe timeout error = %q", err)
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
