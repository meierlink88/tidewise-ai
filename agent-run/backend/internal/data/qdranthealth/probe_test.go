package qdranthealth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform/runtimehealth"
)

func TestProbeRequiresBothFixedCollectionsAndSendsServiceCredential(t *testing.T) {
	seen := map[string]bool{}
	var lock sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("api-key") != "qdrant-secret" || request.Header.Get("Authorization") != "Bearer qdrant-secret" {
			response.WriteHeader(http.StatusUnauthorized)
			return
		}
		lock.Lock()
		seen[request.URL.Path] = true
		lock.Unlock()
		response.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(response, `{"result":{"status":"green","optimizer_status":"ok"},"status":"ok","time":0.01}`)
	}))
	defer server.Close()
	probe, err := New(Config{BaseURL: server.URL, APIKey: "qdrant-secret", Timeout: time.Second, MaxResponseBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}

	result := probe.Check(context.Background())

	if result.Status != runtimehealth.StatusReady || result.ReasonCode != runtimehealth.ReasonNone {
		t.Fatalf("result = %#v", result)
	}
	for _, collection := range []string{"entity_semantic_v1", "variable_definition_semantic_v1"} {
		if !seen["/collections/"+collection] {
			t.Fatalf("collection %q was not checked; seen=%v", collection, seen)
		}
	}
}

func TestProbeMapsUnhealthyCollectionWithoutLeakingQdrantBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(response, `{"result":{"status":"red","optimizer_status":"error secret detail"},"status":"ok"}`)
	}))
	defer server.Close()
	probe, err := New(Config{BaseURL: server.URL, Timeout: time.Second, MaxResponseBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}

	result := probe.Check(context.Background())

	if result.Status != runtimehealth.StatusDegraded || result.ReasonCode != runtimehealth.ReasonCollectionUnhealthy {
		t.Fatalf("result = %#v", result)
	}
}

func TestProbeMapsMissingCollection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/collections/variable_definition_semantic_v1" {
			response.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = fmt.Fprint(response, `{"result":{"status":"green","optimizer_status":"ok"},"status":"ok"}`)
	}))
	defer server.Close()
	probe, err := New(Config{BaseURL: server.URL, Timeout: time.Second, MaxResponseBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}

	result := probe.Check(context.Background())

	if result.Status != runtimehealth.StatusDegraded || result.ReasonCode != runtimehealth.ReasonCollectionUnhealthy {
		t.Fatalf("result = %#v", result)
	}
}

func TestProbeMapsTimeoutAndCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		<-request.Context().Done()
	}))
	defer server.Close()

	t.Run("timeout", func(t *testing.T) {
		probe, err := New(Config{BaseURL: server.URL, Timeout: time.Millisecond, MaxResponseBytes: 4096})
		if err != nil {
			t.Fatal(err)
		}
		result := probe.Check(context.Background())
		if result.Status != runtimehealth.StatusDown || result.ReasonCode != runtimehealth.ReasonTimeout {
			t.Fatalf("result = %#v", result)
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		probe, err := New(Config{BaseURL: server.URL, Timeout: time.Second, MaxResponseBytes: 4096})
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		result := probe.Check(ctx)
		if result.Status != runtimehealth.StatusDown || result.ReasonCode != runtimehealth.ReasonTimeout {
			t.Fatalf("result = %#v", result)
		}
	})
}
