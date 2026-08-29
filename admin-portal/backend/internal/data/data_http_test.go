package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/biz"
)

func TestHTTPClientListsAdminDataWithIdentityRequestIDAndTypedQueries(t *testing.T) {
	t.Parallel()
	requests := make(chan *http.Request, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests <- request.Clone(context.Background())
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case eventsPath:
			_, _ = writer.Write([]byte(`{"request_id":"data-req-2","result":{"items":[{"id":"EVT22222222-2222-5222-8222-222222222222","title":"event","summary":"summary","semantic":{"actors":["Federal Reserve"],"action":"holds target rate","objects":["federal funds rate"],"stage":"ANNOUNCED","jurisdictions":["United States"],"effective_at":null,"time_precision":"DAY"},"modality":"FACT","occurred_at":"2026-07-17T01:02:03Z","announced_at":null,"status":"ACTIVE"}],"total":1,"page":1,"page_size":20}}`))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, server.Client(), "admin-service-token")
	ctx := WithRequestID(context.Background(), "admin-req-123")

	eventFrom := time.Date(2026, 7, 1, 2, 3, 4, 0, time.UTC)
	eventPage, err := client.ListEvents(ctx, biz.EventListQuery{Title: "event title", Modality: biz.EventModalityFact, Status: biz.EventLifecycleActive, OccurredFrom: &eventFrom, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(eventPage.Items) != 1 || eventPage.Items[0].Status != biz.EventLifecycleActive ||
		!reflect.DeepEqual(eventPage.Items[0].Semantic.Actors, []string{"Federal Reserve"}) ||
		eventPage.Items[0].Semantic.Action != "holds target rate" ||
		!reflect.DeepEqual(eventPage.Items[0].Semantic.Objects, []string{"federal funds rate"}) ||
		eventPage.Items[0].Semantic.Stage != biz.EventStageAnnounced ||
		!reflect.DeepEqual(eventPage.Items[0].Semantic.Jurisdictions, []string{"United States"}) ||
		eventPage.Items[0].Semantic.EffectiveAt != nil || eventPage.Items[0].Semantic.TimePrecision != biz.EventTimePrecisionDay {
		t.Fatalf("events = %#v", eventPage)
	}

	eventRequest := <-requests
	if eventRequest.Header.Get("Authorization") != "Bearer admin-service-token" || eventRequest.Header.Get(RequestIDHeader) != "admin-req-123" {
		t.Fatalf("auth/request ID for %q = %q/%q", eventRequest.URL.Path, eventRequest.Header.Get("Authorization"), eventRequest.Header.Get(RequestIDHeader))
	}
	if eventRequest.URL.Path != eventsPath || eventRequest.URL.Query().Get("occurred_from") != eventFrom.Format(time.RFC3339) || eventRequest.URL.Query().Get("modality") != "FACT" || eventRequest.URL.Query().Get("status") != "ACTIVE" {
		t.Fatalf("event request = %s?%s", eventRequest.URL.Path, eventRequest.URL.RawQuery)
	}
}

func TestHTTPClientListsEvidenceCategoriesAndSourcesWithStrictProjection(t *testing.T) {
	t.Parallel()
	requests := make(chan *http.Request, 3)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests <- request.Clone(context.Background())
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case evidencesPath:
			_, _ = writer.Write([]byte(`{"request_id":"data-evidence","result":{"items":[{"id":"EVD22222222-2222-5222-8222-222222222222","raw_evidence_id":"RAW22222222-2222-5222-8222-222222222222","title":"raw title","summary":"summary","semantic":{"who":"Official","what":"announced a policy","when":null,"where":"China","why":null,"how":null},"categories":[{"id":"EVC22222222-2222-5222-8222-222222222222","code":"EVENT_BRIEF","name":"Event brief","description":"description"}],"source_id":"SRC_example_00000000000000000000","source_name":"Official","source_level":"L1_OFFICIAL","source_url":"https://example.com/report","is_original":false,"quoted_source_name":"Agency filing","keywords":["policy","exports"],"is_split":true,"published_at":"2026-08-19T01:00:00Z","collected_at":"2026-08-19T02:00:00Z"}],"total":1,"page":1,"page_size":50}}`))
		case evidenceCategoriesPath:
			_, _ = writer.Write([]byte(`{"request_id":"data-category","result":{"categories":[{"id":"EVC22222222-2222-5222-8222-222222222222","code":"EVENT_BRIEF","name":"Event brief","description":"description"}]}}`))
		case sourcesPath:
			_, _ = writer.Write([]byte(`{"request_id":"data-source","result":{"sources":[{"id":"SRC22222222-2222-5222-8222-222222222222","code":"official","name":"Official","ownership_type":"fixed","channel_type":"api","adapter_key":"tavily","enabled":true,"endpoint":"https://provider.example.test","app_key":"must-not-leak","config":{"secret":"must-not-leak"},"priority":1,"timeout_seconds":10,"max_results":20,"default_source_level":"L1_OFFICIAL","created_at":"2026-08-19T01:00:00Z","updated_at":"2026-08-19T02:00:00Z"}]}}`))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, server.Client(), "admin-service-token")
	isSplit := true
	page, err := client.ListEvidences(context.Background(), biz.EvidenceListQuery{Title: "raw", CategoryID: "EVC22222222-2222-5222-8222-222222222222", SourceID: "SRC_example_00000000000000000000", IsSplit: &isSplit, Page: 1, PageSize: 50})
	if err != nil || len(page.Items) != 1 || len(page.Items[0].Categories) != 1 || page.Items[0].Semantic.What != "announced a policy" ||
		page.Items[0].SourceID != "SRC_example_00000000000000000000" || page.Items[0].SourceURL != "https://example.com/report" || page.Items[0].IsOriginal || page.Items[0].QuotedSourceName == nil ||
		*page.Items[0].QuotedSourceName != "Agency filing" || len(page.Items[0].Keywords) != 2 {
		t.Fatalf("Evidence page/error = %#v/%v", page, err)
	}
	categories, err := client.ListEvidenceCategories(context.Background())
	if err != nil || len(categories) != 1 {
		t.Fatalf("categories/error = %#v/%v", categories, err)
	}
	sources, err := client.ListSources(context.Background())
	if err != nil || len(sources) != 1 || sources[0].Code != "official" {
		t.Fatalf("sources/error = %#v/%v", sources, err)
	}
	first, second, third := <-requests, <-requests, <-requests
	if first.URL.Query().Get("title") != "raw" || first.URL.Query().Get("category_id") == "" || first.URL.Query().Get("source_id") != "SRC_example_00000000000000000000" || first.URL.Query().Get("is_split") != "true" || second.URL.RawQuery != "" || third.URL.RawQuery != "" {
		t.Fatalf("requests = %s / %s / %s", first.URL.String(), second.URL.String(), third.URL.String())
	}
}

func TestHTTPClientGetsRawEvidenceDocumentWithStrictContract(t *testing.T) {
	t.Parallel()
	const id = "RAW22222222-2222-5222-8222-222222222222"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != rawEvidencesPath+"/"+id || request.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("request = %s auth=%q", request.URL.Path, request.Header.Get("Authorization"))
		}
		_, _ = writer.Write([]byte(`{"request_id":"raw-document","result":{"raw_evidence":{"id":"RAW22222222-2222-5222-8222-222222222222","source_id":"SRC_example_00000000000000000000","source_name":"Official","source_level":"L1_OFFICIAL","source_url":"https://example.test/article","is_original":true,"quoted_source_id":null,"quoted_source_name":null,"title":"Title","raw_text":"/raw-evidence/documents/2026/08/17/11f0864fc4078b47a4cc758149a2b0b7923654d2c7c8a694ad5b2d5ced4fc998.md","published_at":null,"collected_at":"2026-08-19T02:00:00Z","keywords":[],"categories":[]}}}`))
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, server.Client(), "token")
	document, err := client.GetRawEvidenceDocument(context.Background(), id)
	if err != nil || !strings.HasPrefix(document.RawText, "/raw-evidence/documents/") {
		t.Fatalf("document/error = %#v/%v", document, err)
	}
}

func TestHTTPClientMapsRawEvidenceNotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNotFound)
		_, _ = writer.Write([]byte(`{"request_id":"missing","error":{"code":"RAW_EVIDENCE_NOT_FOUND","message":"not found","details":{}}}`))
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, server.Client(), "token")
	_, err := client.GetRawEvidenceDocument(context.Background(), "RAW22222222-2222-5222-8222-222222222222")
	if !errors.Is(err, biz.ErrRawEvidenceNotFound) {
		t.Fatalf("error = %v", err)
	}
}

func TestHTTPClientRejectsInvalidRawEvidenceDocumentProjection(t *testing.T) {
	t.Parallel()
	const id = "RAW22222222-2222-5222-8222-222222222222"
	const valid = `{"request_id":"raw-document","result":{"raw_evidence":{"id":"RAW22222222-2222-5222-8222-222222222222","source_id":"SRC_example_00000000000000000000","source_name":"Official","source_level":"L1_OFFICIAL","source_url":"https://example.test/article","is_original":true,"quoted_source_id":null,"quoted_source_name":null,"title":"Title","raw_text":"/raw-evidence/documents/2026/08/17/11f0864fc4078b47a4cc758149a2b0b7923654d2c7c8a694ad5b2d5ced4fc998.md","published_at":null,"collected_at":"2026-08-19T02:00:00Z","keywords":[],"categories":[]}}}`
	for _, test := range []struct {
		name    string
		payload string
	}{
		{name: "oversized source identity", payload: strings.Replace(valid, "SRC_example_00000000000000000000", strings.Repeat("s", 33), 1)},
		{name: "unsupported source level", payload: strings.Replace(valid, "L1_OFFICIAL", "UNKNOWN", 1)},
		{name: "non-http source URL", payload: strings.Replace(valid, "https://example.test/article", "file:///secret", 1)},
		{name: "invalid category", payload: strings.Replace(valid, `"categories":[]`, `"categories":[{"id":"invalid","code":"EVENT_BRIEF","name":"Event brief","description":"description"}]`, 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				_, _ = writer.Write([]byte(test.payload))
			}))
			defer server.Close()
			client := newTestClient(t, server.URL, server.Client(), "token")
			_, err := client.GetRawEvidenceDocument(context.Background(), id)
			assertDataUnavailable(t, err)
		})
	}
}

func TestHTTPClientAcceptsMaximumCompleteSourceListLargerThanOneMiB(t *testing.T) {
	t.Parallel()
	sources := make([]map[string]any, 200)
	for index := range sources {
		sources[index] = map[string]any{
			"id":   fmt.Sprintf("SRC%08x-0000-5000-8000-%012x", index+1, index+1),
			"code": fmt.Sprintf("source-%03d", index), "name": fmt.Sprintf("Source %03d", index),
			"ownership_type": "fixed", "channel_type": "api", "adapter_key": "tavily", "enabled": true,
			"endpoint": "https://provider.example.test/" + strings.Repeat("a", 2000), "app_key": strings.Repeat("k", 512),
			"config": map[string]any{"blob": strings.Repeat("x", 4080)}, "priority": 1, "timeout_seconds": 300,
			"max_results": 100, "default_source_level": "L1_OFFICIAL", "created_at": "2026-08-19T01:00:00Z", "updated_at": "2026-08-19T02:00:00Z",
		}
	}
	payload, err := json.Marshal(map[string]any{"request_id": "large-source-list", "result": map[string]any{"sources": sources}})
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) <= 1<<20 || len(payload) > maxSuccessBodyBytes {
		t.Fatalf("payload size = %d", len(payload))
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) { _, _ = writer.Write(payload) }))
	defer server.Close()
	client := newTestClient(t, server.URL, server.Client(), "token")
	result, err := client.ListSources(context.Background())
	if err != nil || len(result) != 200 {
		t.Fatalf("Source count/error = %d/%v", len(result), err)
	}
}

func TestClassifyReadErrorPreservesCancellationAndTimeout(t *testing.T) {
	if !errors.Is(classifyReadError(&Error{Kind: ErrorKindCanceled}), context.Canceled) {
		t.Fatal("cancellation was not preserved")
	}
	if !errors.Is(classifyReadError(&Error{Kind: ErrorKindTimeout}), context.DeadlineExceeded) {
		t.Fatal("timeout was not preserved")
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

	if _, err := client.ListEvents(context.Background(), biz.EventListQuery{}); err != nil {
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
		context.Background(), http.MethodPost, "Data.TestMutation", eventsPath,
		eventsPath, map[string]string{"value": "mutation"}, nil,
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

func TestHTTPClientRejectsEventContractDrift(t *testing.T) {
	t.Parallel()
	valid := `{"request_id":"data-req","result":{"items":[{"id":"EVT22222222-2222-5222-8222-222222222222","title":"event","summary":"summary","semantic":{"actors":["Federal Reserve"],"action":"holds target rate","objects":["federal funds rate"],"stage":"ANNOUNCED","jurisdictions":["United States"],"effective_at":null,"time_precision":"DAY"},"modality":"FACT","occurred_at":null,"announced_at":null,"status":"ACTIVE"}],"total":1,"page":1,"page_size":50}}`
	for _, test := range []struct {
		name    string
		payload string
	}{
		{name: "missing semantic", payload: strings.Replace(valid, `,"semantic":{"actors":["Federal Reserve"],"action":"holds target rate","objects":["federal funds rate"],"stage":"ANNOUNCED","jurisdictions":["United States"],"effective_at":null,"time_precision":"DAY"}`, "", 1)},
		{name: "missing semantic key", payload: strings.Replace(valid, `,"time_precision":"DAY"`, "", 1)},
		{name: "extra semantic key", payload: strings.Replace(valid, `,"time_precision":"DAY"`, `,"time_precision":"DAY","extra":null`, 1)},
		{name: "legacy semantic keys", payload: strings.Replace(valid, `"semantic":{"actors":["Federal Reserve"],"action":"holds target rate","objects":["federal funds rate"],"stage":"ANNOUNCED","jurisdictions":["United States"],"effective_at":null,"time_precision":"DAY"}`, `"semantic":{"who":null,"what":"fact","when":null,"where":null,"why":null,"how":null}`, 1)},
		{name: "invalid semantic stage", payload: strings.Replace(valid, `"stage":"ANNOUNCED"`, `"stage":"INVALID"`, 1)},
		{name: "missing nullable time", payload: strings.Replace(valid, `,"occurred_at":null`, "", 1)},
		{name: "wrong ID", payload: strings.Replace(valid, "EVT22222222-2222-5222-8222-222222222222", "not-an-event", 1)},
		{name: "blank title", payload: strings.Replace(valid, `"title":"event"`, `"title":" "`, 1)},
		{name: "oversized page", payload: strings.Replace(valid, `"page_size":50`, `"page_size":101`, 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				_, _ = writer.Write([]byte(test.payload))
			}))
			defer server.Close()
			client := newTestClient(t, server.URL, server.Client(), "token")
			_, err := client.ListEvents(context.Background(), biz.EventListQuery{})
			assertDataUnavailable(t, err)
		})
	}
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

			_, err := client.ListEvents(context.Background(), biz.EventListQuery{})
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
