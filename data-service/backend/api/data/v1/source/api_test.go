package source

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

func TestSourceSnapshotProviderFixtureMatchesOpenAPIAndExactDTO(t *testing.T) {
	payload, err := os.ReadFile("testdata/source-snapshot.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) > 500_000 {
		t.Fatalf("fixture size = %d, exceeds snapshot envelope budget", len(payload))
	}
	var envelope struct {
		RequestID string         `json:"request_id"`
		Result    SourceSnapshot `json:"result"`
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		t.Fatalf("fixture contains trailing JSON: %v", err)
	}
	if envelope.RequestID == "" || len(envelope.Result.Sources) != 2 {
		t.Fatalf("fixture envelope = request_id %q, Source count %d", envelope.RequestID, len(envelope.Result.Sources))
	}
	if envelope.Result.Sources[0].ChannelType != "api" || envelope.Result.Sources[1].ChannelType != "web_search" {
		t.Fatalf("fixture is not in stable snapshot order: %+v", envelope.Result.Sources)
	}
	if envelope.Result.Sources[1].AppKey == nil || *envelope.Result.Sources[1].AppKey != "plaintext-provider-key" {
		t.Fatalf("fixture does not freeze plaintext app_key: %+v", envelope.Result.Sources[1])
	}
	for _, item := range envelope.Result.Sources {
		wantID, err := coreid.Derive(coreid.Source, "source", item.Code)
		if err != nil {
			t.Fatal(err)
		}
		if item.ID != wantID {
			t.Fatalf("fixture Source %s id = %q, want runtime-derived %q", item.Code, item.ID, wantID)
		}
	}

	document, err := openapi3.NewLoader().LoadFromData(v1.Document())
	if err != nil {
		t.Fatal(err)
	}
	if err := document.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		t.Fatal(err)
	}
	if err := document.Components.Schemas["SourceSnapshotEnvelope"].Value.VisitJSON(value); err != nil {
		t.Fatalf("Source snapshot fixture violates OpenAPI: %v", err)
	}
}
