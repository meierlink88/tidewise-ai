package runtimehealth

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
)

func TestRuntimeHealthFixtureMatchesOpenAPI(t *testing.T) {
	document, err := openapi3.NewLoader().LoadFromData(v1.Document())
	if err != nil {
		t.Fatal(err)
	}
	if err := document.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile("testdata/runtime-health.json")
	if err != nil {
		t.Fatal(err)
	}
	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		t.Fatal(err)
	}
	if err := document.Components.Schemas["RuntimeHealthEnvelope"].Value.VisitJSON(value); err != nil {
		t.Fatalf("runtime health fixture violates OpenAPI: %v", err)
	}
}
