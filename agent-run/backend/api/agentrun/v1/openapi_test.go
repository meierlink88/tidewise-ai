package v1

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestOpenAPIParsesWithLocalReferencesAndFreezesTidewiseEnvelopes(t *testing.T) {
	t.Parallel()

	loader := openapi3.NewLoader()
	document, err := loader.LoadFromData(Document())
	if err != nil {
		t.Fatal(err)
	}
	if err := document.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if document.OpenAPI != "3.0.4" {
		t.Fatalf("OpenAPI version = %q", document.OpenAPI)
	}
	for _, schemaName := range []string{
		"CollectorRunEnvelope",
		"ModelProviderListEnvelope",
		"ModelProviderConfigurationEnvelope",
		"ConnectorListEnvelope",
		"ConnectorConfigurationEnvelope",
		"AgentScheduleListEnvelope",
		"AgentScheduleEnvelope",
		"AgentExecutionPageEnvelope",
		"AgentStatusListEnvelope",
		"MonitoringSummaryEnvelope",
		"CollectorMonitoringPageEnvelope",
		"ArtifactMonitoringPageEnvelope",
		"SemanticMonitoringPageEnvelope",
		"EventSemanticWorkItemEnvelope",
	} {
		schema := document.Components.Schemas[schemaName]
		if schema == nil || schema.Value == nil {
			t.Fatalf("success envelope schema %s is missing", schemaName)
		}
	}
	errorSchema := document.Components.Schemas["ErrorResponse"]
	if errorSchema == nil || errorSchema.Value == nil {
		t.Fatal("ErrorResponse is missing")
	}
	if _, ok := errorSchema.Value.Properties["request_id"]; !ok {
		t.Fatal("ErrorResponse does not require a request_id")
	}
	if _, ok := errorSchema.Value.Properties["error"]; !ok {
		t.Fatal("ErrorResponse does not contain structured error details")
	}
	for fixture, schemaName := range map[string]string{
		"testdata/collector-run-success.json":     "CollectorRunEnvelope",
		"testdata/admin-model-provider-list.json": "ModelProviderListEnvelope",
		"testdata/error-response.json":            "ErrorResponse",
	} {
		payload, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatal(err)
		}
		var value any
		if err := json.Unmarshal(payload, &value); err != nil {
			t.Fatalf("decode fixture %s: %v", fixture, err)
		}
		schema := document.Components.Schemas[schemaName]
		if schema == nil || schema.Value == nil {
			t.Fatalf("fixture schema %s is missing", schemaName)
		}
		if err := schema.Value.VisitJSON(value); err != nil {
			t.Fatalf("fixture %s violates %s: %v", fixture, schemaName, err)
		}
	}
}

func TestOpenAPIIncludesDependentEventExtractorExecutionStates(t *testing.T) {
	loader := openapi3.NewLoader()
	document, err := loader.LoadFromData(Document())
	if err != nil {
		t.Fatal(err)
	}
	execution := document.Components.Schemas["AgentExecutionListItem"].Value
	trigger := execution.Properties["trigger_source"].Value.Enum
	if !slices.Contains(trigger, any("dependent")) {
		t.Fatalf("trigger_source enum = %#v, want dependent", trigger)
	}
	status := document.Components.Schemas["ExecutionStatus"].Value.Enum
	if !slices.Contains(status, any("running")) {
		t.Fatalf("ExecutionStatus enum = %#v, want running", status)
	}
}
