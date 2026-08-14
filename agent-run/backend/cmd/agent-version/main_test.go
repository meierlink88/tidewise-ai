package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

func TestPublicationDocumentRoundTrip(t *testing.T) {
	want := agentrun.AgentVersionPublication{Added: []agentrun.AgentVersion{{
		AgentKey: "event-semantic-enricher",
		Version:  "event-semantic-enricher.v4",
	}}}
	document := encodePublication(want)
	got, err := decodePublication(strings.NewReader(
		`{"added":[{"agent_key":"event-semantic-enricher","version":"event-semantic-enricher.v4"}]}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) || len(document.Added) != 1 ||
		document.Added[0].Version != want.Added[0].Version {
		t.Fatalf("document = %#v, publication = %#v", document, got)
	}
}

func TestPublicationDocumentRejectsUnknownOrTrailingContent(t *testing.T) {
	for _, payload := range []string{
		`{"added":[],"unknown":true}`,
		`{"added":[]} {}`,
	} {
		if _, err := decodePublication(strings.NewReader(payload)); err == nil {
			t.Fatalf("payload %q was accepted", payload)
		}
	}
}
