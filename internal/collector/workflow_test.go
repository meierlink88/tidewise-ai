package collector

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
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
	runs map[string]ConnectorRun
}

func (m *captureMaterializer) Materialize(_ context.Context, request Request, runs map[string]ConnectorRun) (*Result, error) {
	m.runs = runs
	return &Result{RunID: request.RunID, Stats: Stats{ResultsPending: 0}}, nil
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
	workflow, err := NewWorkflow(context.Background(), connectors, 2, materializer)
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
