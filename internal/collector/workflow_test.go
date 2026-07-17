package collector

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
)

type fakeConnector struct {
	name        string
	fail        bool
	active      *int32
	maxObserved *int32
}

func (f fakeConnector) Name() string { return f.name }

func (f fakeConnector) Collect(context.Context, Request) ([]Candidate, error) {
	active := atomic.AddInt32(f.active, 1)
	defer atomic.AddInt32(f.active, -1)
	for {
		observed := atomic.LoadInt32(f.maxObserved)
		if active <= observed || atomic.CompareAndSwapInt32(f.maxObserved, observed, active) {
			break
		}
	}
	time.Sleep(15 * time.Millisecond)
	if f.fail {
		return nil, errors.New("connector unavailable")
	}
	return []Candidate{{Title: f.name, URL: "https://example.com/" + f.name, Content: f.name, ContentLevel: LevelSnippet}}, nil
}

type captureMaterializer struct {
	runs  map[string]ConnectorRun
	calls int32
}

func (m *captureMaterializer) Materialize(_ context.Context, request Request, runs map[string]ConnectorRun) (*Result, error) {
	atomic.AddInt32(&m.calls, 1)
	m.runs = runs
	return &Result{RunID: request.RunID, Stats: Stats{ResultsPending: 0}}, nil
}

type plannerFunc func(context.Context, *Request) (*Request, error)

func (f plannerFunc) Plan(ctx context.Context, request *Request) (*Request, error) {
	return f(ctx, request)
}

func passthroughPlanner(_ context.Context, request *Request) (*Request, error) {
	copyRequest := *request
	copyRequest.SearchQueries = append([]string(nil), request.SearchQueries...)
	return &copyRequest, nil
}

type recordingConnector struct {
	name       string
	calls      *int32
	mu         *sync.Mutex
	requests   *[]Request
	candidates []Candidate
}

func (c recordingConnector) Name() string { return c.name }

func (c recordingConnector) Collect(_ context.Context, request Request) ([]Candidate, error) {
	atomic.AddInt32(c.calls, 1)
	c.mu.Lock()
	*c.requests = append(*c.requests, request)
	c.mu.Unlock()
	return append([]Candidate(nil), c.candidates...), nil
}

func TestWorkflowCapsConcurrencyAndPreservesConnectorFailure(t *testing.T) {
	var active, maxObserved int32
	var connectors []Connector
	for index := 0; index < 5; index++ {
		connectors = append(connectors, fakeConnector{
			name: fmt.Sprintf("connector_%d", index), fail: index == 3,
			active: &active, maxObserved: &maxObserved,
		})
	}
	materializer := &captureMaterializer{}
	workflow, err := NewWorkflow(context.Background(), plannerFunc(passthroughPlanner), connectors, 2, materializer)
	if err != nil {
		t.Fatal(err)
	}
	result, err := workflow.Invoke(context.Background(), &Request{RunID: "run-1"})
	if err != nil {
		t.Fatal(err)
	}
	if result.RunID != "run-1" || len(materializer.runs) != 5 {
		t.Fatalf("unexpected aggregation: result=%+v runs=%d", result, len(materializer.runs))
	}
	if maxObserved > 2 {
		t.Fatalf("parallel limit exceeded: %d", maxObserved)
	}
	if materializer.runs["connector_3"].Error != "connector unavailable" {
		t.Fatalf("connector failure was not preserved: %+v", materializer.runs["connector_3"])
	}
	if got := materializer.runs["connector_0"].Results[0].ContentOrigin; got != ContentOrigin {
		t.Fatalf("content origin = %q", got)
	}
}

func TestWorkflowPlanningRunsBeforeConnectorsAndSharesPlannedRequest(t *testing.T) {
	var connectorCalls int32
	var requests []Request
	var mu sync.Mutex
	plannerCalls := int32(0)
	planner := plannerFunc(func(_ context.Context, request *Request) (*Request, error) {
		atomic.AddInt32(&plannerCalls, 1)
		if atomic.LoadInt32(&connectorCalls) != 0 {
			t.Fatal("connector ran before planner completed")
		}
		if request.Objective != "runtime prompt intent" || len(request.SearchQueries) != 0 {
			t.Fatalf("planner input = %+v", request)
		}
		planned := *request
		planned.SearchQueries = []string{"planned-one", "planned-two"}
		return &planned, nil
	})
	connectors := []Connector{
		recordingConnector{name: "one", calls: &connectorCalls, mu: &mu, requests: &requests},
		recordingConnector{name: "two", calls: &connectorCalls, mu: &mu, requests: &requests},
	}
	materializer := &captureMaterializer{}
	workflow, err := NewWorkflow(context.Background(), planner, connectors, 2, materializer)
	if err != nil {
		t.Fatal(err)
	}
	input := &Request{RunID: "run-planned", Objective: "runtime prompt intent"}
	if _, err := workflow.Invoke(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if plannerCalls != 1 || connectorCalls != 2 || materializer.calls != 1 {
		t.Fatalf("calls planner=%d connectors=%d materializer=%d", plannerCalls, connectorCalls, materializer.calls)
	}
	for _, request := range requests {
		if request.Objective != input.Objective {
			t.Fatalf("connector objective = %q, want %q", request.Objective, input.Objective)
		}
		if !reflect.DeepEqual(request.SearchQueries, []string{"planned-one", "planned-two"}) {
			t.Fatalf("connector got %#v", request.SearchQueries)
		}
	}
	if input.SearchQueries != nil {
		t.Fatalf("workflow mutated input: %#v", input.SearchQueries)
	}
}

func TestWorkflowPlanningFailureStopsConnectorsAndMaterializer(t *testing.T) {
	tests := map[string]func(*testing.T) (context.Context, QueryPlanner){
		"API failure": func(t *testing.T) (context.Context, QueryPlanner) {
			planner, err := NewDeepSeekQueryPlanner(fakeChatModel{generate: func(context.Context, []*schema.Message) (*schema.Message, error) {
				return nil, errors.New("provider unavailable")
			}})
			if err != nil {
				t.Fatal(err)
			}
			return context.Background(), planner
		},
		"schema failure": func(t *testing.T) (context.Context, QueryPlanner) {
			planner, err := NewDeepSeekQueryPlanner(fakeChatModel{generate: func(context.Context, []*schema.Message) (*schema.Message, error) {
				return schema.AssistantMessage(`{"queries":["q"]} trailing`, nil), nil
			}})
			if err != nil {
				t.Fatal(err)
			}
			return context.Background(), planner
		},
		"cancellation": func(t *testing.T) (context.Context, QueryPlanner) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			planner, err := NewDeepSeekQueryPlanner(fakeChatModel{generate: func(ctx context.Context, _ []*schema.Message) (*schema.Message, error) {
				return nil, ctx.Err()
			}})
			if err != nil {
				t.Fatal(err)
			}
			return ctx, planner
		},
	}
	for name, setup := range tests {
		t.Run(name, func(t *testing.T) {
			var connectorCalls int32
			var requests []Request
			var mu sync.Mutex
			materializer := &captureMaterializer{}
			ctx, planner := setup(t)
			workflow, err := NewWorkflow(context.Background(), planner, []Connector{
				recordingConnector{name: "one", calls: &connectorCalls, mu: &mu, requests: &requests},
			}, 1, materializer)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = workflow.Invoke(ctx, &Request{Objective: "prompt intent"}); err == nil {
				t.Fatal("expected planner failure")
			}
			if connectorCalls != 0 || materializer.calls != 0 {
				t.Fatalf("calls connectors=%d materializer=%d", connectorCalls, materializer.calls)
			}
		})
	}
}

func TestWorkflowContentOriginAndFactsComeOnlyFromConnector(t *testing.T) {
	var connectorCalls int32
	var requests []Request
	var mu sync.Mutex
	want := Candidate{
		Title: "connector title", URL: "https://example.com/fact", PublishedAtHint: "2026-07-18",
		SourceName: "connector source", SourceExternalID: "connector-id", SourceType: "news",
		Content: "connector evidence", ContentLevel: LevelSnippet,
	}
	planner := plannerFunc(func(_ context.Context, request *Request) (*Request, error) {
		planned := *request
		planned.SearchQueries = []string{"model claimed fact"}
		return &planned, nil
	})
	materializer := &captureMaterializer{}
	workflow, err := NewWorkflow(context.Background(), planner, []Connector{
		recordingConnector{name: "facts", calls: &connectorCalls, mu: &mu, requests: &requests, candidates: []Candidate{want}},
	}, 1, materializer)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = workflow.Invoke(context.Background(), &Request{RunID: "facts"}); err != nil {
		t.Fatal(err)
	}
	got := materializer.runs["facts"].Results[0]
	if got.Title != want.Title || got.URL != want.URL || got.PublishedAtHint != want.PublishedAtHint || got.SourceName != want.SourceName || got.SourceExternalID != want.SourceExternalID || got.SourceType != want.SourceType || got.Content != want.Content || got.ContentLevel != want.ContentLevel {
		t.Fatalf("connector facts changed: %+v", got)
	}
	if got.ContentOrigin != ContentOrigin || strings.Contains(got.Content, "model claimed fact") {
		t.Fatalf("model content crossed fact boundary: %+v", got)
	}
}
