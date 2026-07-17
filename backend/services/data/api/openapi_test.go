package api_test

import (
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const namespace = "/internal/data/v1"

type operationContract struct {
	method      string
	operationID string
	scope       string
}

func TestOpenAPIContractFreezesNamespacePathsOperationsAndScopes(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	want := map[string]operationContract{
		namespace + "/research/themes":                        {method: "get", operationID: "listResearchThemes", scope: "data.research.read"},
		namespace + "/research/themes/{theme_id}":             {method: "get", operationID: "getResearchTheme", scope: "data.research.read"},
		namespace + "/research/anchors":                       {method: "get", operationID: "listResearchAnchors", scope: "data.research.read"},
		namespace + "/research/anchors/{anchor_id}":           {method: "get", operationID: "getResearchAnchor", scope: "data.research.read"},
		namespace + "/admin/raw-documents":                    {method: "get", operationID: "listAdminRawDocuments", scope: "data.admin.read"},
		namespace + "/admin/events":                           {method: "get", operationID: "listAdminEvents", scope: "data.admin.read"},
		namespace + "/admin/source-catalogs":                  {method: "get", operationID: "listAdminSourceCatalogs", scope: "data.admin.read"},
		namespace + "/agent-run/source-metadata":              {method: "get", operationID: "listAgentSourceMetadata", scope: "data.source-metadata.read"},
		namespace + "/raw-document-imports":                   {method: "post", operationID: "importRawDocuments", scope: "data.raw-documents.import"},
		namespace + "/raw-document-imports/{idempotency_key}": {method: "get", operationID: "getRawDocumentImportStatus", scope: "data.raw-documents.import"},
		namespace + "/reviewed-event-imports":                 {method: "post", operationID: "importReviewedEvent", scope: "data.reviewed-events.import"},
	}

	if len(paths) != len(want) {
		t.Fatalf("path count = %d, want %d; got %v", len(paths), len(want), sortedKeys(paths))
	}
	for path, expected := range want {
		if !strings.HasPrefix(path, namespace+"/") {
			t.Fatalf("path %q escapes namespace %q", path, namespace)
		}
		pathItem := object(t, paths[path], "path "+path)
		operation := object(t, pathItem[expected.method], expected.method+" "+path)
		assertString(t, operation, "operationId", expected.operationID)
		assertString(t, operation, "x-client-drift-anchor", "data.v1."+expected.operationID)
		assertString(t, operation, "x-required-service-scope", expected.scope)
		for _, method := range []string{"get", "post", "put", "patch", "delete"} {
			if method != expected.method {
				if _, exists := pathItem[method]; exists {
					t.Fatalf("path %q unexpectedly defines %s", path, method)
				}
			}
		}
	}
}

func TestOpenAPIContractFreezesBearerIdentityRequestIDAndStructuredErrors(t *testing.T) {
	document := loadContract(t)
	if document["openapi"] != "3.0.3" {
		t.Fatalf("openapi = %v, want 3.0.3", document["openapi"])
	}
	assertString(t, document, "x-contract-id", "tidewise-data-v1")
	assertString(t, document, "x-handwritten-client-policy", "consumer-owned-small-typed-clients")

	security := array(t, document["security"], "security")
	if len(security) != 1 {
		t.Fatalf("global security length = %d, want 1", len(security))
	}
	globalScheme := object(t, security[0], "security[0]")
	if _, ok := globalScheme["ServiceBearer"]; !ok {
		t.Fatalf("global security = %v, want ServiceBearer", globalScheme)
	}

	components := object(t, document["components"], "components")
	schemes := object(t, components["securitySchemes"], "securitySchemes")
	bearer := object(t, schemes["ServiceBearer"], "ServiceBearer")
	assertString(t, bearer, "type", "http")
	assertString(t, bearer, "scheme", "bearer")
	assertString(t, bearer, "bearerFormat", "opaque service identity token")

	parameters := object(t, components["parameters"], "parameters")
	requestID := object(t, parameters["RequestID"], "RequestID")
	assertString(t, requestID, "name", "X-Request-ID")
	assertString(t, requestID, "in", "header")

	errorEnvelope := schema(t, document, "ErrorEnvelope")
	assertRequired(t, errorEnvelope, "error", "request_id")
	errorDetail := schema(t, document, "ErrorDetail")
	assertRequired(t, errorDetail, "code", "message", "details")

	paths := object(t, document["paths"], "paths")
	for path, rawPathItem := range paths {
		pathItem := object(t, rawPathItem, "path "+path)
		for _, method := range []string{"get", "post"} {
			rawOperation, exists := pathItem[method]
			if !exists {
				continue
			}
			operation := object(t, rawOperation, method+" "+path)
			responses := object(t, operation["responses"], "responses for "+method+" "+path)
			for _, status := range []string{"401", "403", "500"} {
				if _, ok := responses[status]; !ok {
					t.Fatalf("%s %s is missing structured %s response", method, path, status)
				}
			}
		}
	}
}

func TestOpenAPIContractFreezesBoundedAtomicRawImportAndStatus(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	importOperation := object(t, object(t, paths[namespace+"/raw-document-imports"], "raw import path")["post"], "raw import operation")
	assertString(t, importOperation, "x-canonicalization", "raw-document-import-v1")
	assertString(t, importOperation, "x-atomicity", "whole-batch-single-postgresql-transaction")
	assertString(t, importOperation, "x-retry-policy", "idempotent-with-key")
	requestBody := object(t, importOperation["requestBody"], "raw import requestBody")
	content := object(t, requestBody["content"], "raw import content")
	media := object(t, content["application/json"], "raw import application/json")
	requestSchemaRef := object(t, media["schema"], "raw import request schema")
	assertInt(t, requestSchemaRef, "maxLength", 1048576)
	assertInt(t, requestSchemaRef, "x-max-body-bytes", 1048576)

	request := schema(t, document, "RawDocumentBatchImportRequest")
	assertRequired(t, request, "idempotency_key", "items")
	requestProperties := object(t, request["properties"], "RawDocumentBatchImportRequest properties")
	if _, exists := requestProperties["caller_identity"]; exists {
		t.Fatal("raw import request must derive caller_identity from bearer principal")
	}
	if _, exists := requestProperties["payload_hash"]; exists {
		t.Fatal("raw import request must not trust a caller-supplied payload_hash")
	}
	key := object(t, requestProperties["idempotency_key"], "idempotency_key")
	assertInt(t, key, "minLength", 1)
	assertInt(t, key, "maxLength", 200)
	items := object(t, requestProperties["items"], "raw import items")
	assertInt(t, items, "minItems", 1)
	assertInt(t, items, "maxItems", 100)

	statusPath := object(t, paths[namespace+"/raw-document-imports/{idempotency_key}"], "raw import status path")
	parameters := array(t, statusPath["parameters"], "status parameters")
	if len(parameters) != 1 {
		t.Fatalf("status parameter count = %d, want 1", len(parameters))
	}
	statusKey := object(t, parameters[0], "status idempotency key")
	assertString(t, statusKey, "name", "idempotency_key")
	assertString(t, statusKey, "in", "path")
	keySchema := object(t, statusKey["schema"], "status idempotency key schema")
	assertInt(t, keySchema, "minLength", 1)
	assertInt(t, keySchema, "maxLength", 200)
	statusOperation := object(t, statusPath["get"], "raw import status operation")
	statusResponses := object(t, statusOperation["responses"], "raw import status responses")
	if _, ok := statusResponses["400"]; !ok {
		t.Fatal("raw import status must reject out-of-bounds idempotency_key with 400")
	}

	result := schema(t, document, "RawDocumentImportResult")
	assertRequired(t, result, "receipt_id", "payload_hash", "raw_document_ids", "items", "imported_at")
	resultProperties := object(t, result["properties"], "RawDocumentImportResult properties")
	payloadHash := object(t, resultProperties["payload_hash"], "payload_hash")
	assertString(t, payloadHash, "pattern", "^[0-9a-f]{64}$")
	resultItems := object(t, resultProperties["items"], "result items")
	itemSchema := schema(t, document, refName(t, object(t, resultItems["items"], "result item schema")))
	disposition := object(t, object(t, itemSchema["properties"], "RawDocumentImportItem properties")["disposition"], "disposition")
	if _, referenced := disposition["$ref"]; referenced {
		disposition = schema(t, document, refName(t, disposition))
	}
	assertStringSet(t, disposition["enum"], "created", "reused")

	status := schema(t, document, "RawDocumentImportStatusResponse")
	statusProperties := object(t, status["properties"], "RawDocumentImportStatusResponse properties")
	statusEnum := object(t, statusProperties["status"], "raw import status")
	assertStringSet(t, statusEnum["enum"], "completed", "unknown")

	responses := object(t, importOperation["responses"], "raw import responses")
	if _, ok := responses["409"]; !ok {
		t.Fatal("raw import must define same caller/key changed-payload 409")
	}
}

func TestOpenAPIContractFreezesDTOFormatsEnumsAndSensitiveMetadataBoundary(t *testing.T) {
	document := loadContract(t)
	for _, name := range []string{
		"ResearchThemeCollection", "ResearchThemeDetail", "ResearchAnchorCollection", "ResearchAnchorDetail",
		"AdminRawDocumentPage", "AdminEventPage", "AdminSourceCatalogCollection",
		"AgentSourceMetadataCollection", "RawDocumentBatchImportRequest", "RawDocumentImportStatusResponse",
		"ReviewedEventImportRequest", "ReviewedEventImportResult", "ErrorEnvelope",
	} {
		contractSchema := schema(t, document, name)
		assertString(t, contractSchema, "x-client-drift-anchor", "data.v1.schema."+name)
	}

	uuid := schema(t, document, "UUID")
	assertString(t, uuid, "type", "string")
	assertString(t, uuid, "format", "uuid")
	utc := schema(t, document, "UTCTimestamp")
	assertString(t, utc, "type", "string")
	assertString(t, utc, "format", "date-time")
	if !strings.Contains(strings.ToUpper(stringValue(t, utc["description"], "UTCTimestamp description")), "UTC") || !strings.Contains(stringValue(t, utc["description"], "UTCTimestamp description"), "RFC3339") {
		t.Fatalf("UTCTimestamp description must freeze UTC RFC3339 semantics: %v", utc["description"])
	}

	assertStringSet(t, schema(t, document, "SourceStatus")["enum"], "active", "inactive", "disabled")
	assertStringSet(t, schema(t, document, "RawDocumentDisposition")["enum"], "created", "reused")
	assertStringSet(t, schema(t, document, "EventStatus")["enum"], "candidate", "confirmed", "rejected")
	assertStringSet(t, schema(t, document, "FactStatus")["enum"], "unverified", "verified", "disputed")

	agentMetadata := schema(t, document, "AgentSourceMetadata")
	properties := object(t, agentMetadata["properties"], "AgentSourceMetadata properties")
	forbidden := []string{"secret", "api_key", "token", "cookie", "authorization", "password", "credential_value"}
	for key := range properties {
		lower := strings.ToLower(key)
		for _, fragment := range forbidden {
			if strings.Contains(lower, fragment) {
				t.Fatalf("AgentSourceMetadata exposes forbidden property %q", key)
			}
		}
	}
	if _, ok := properties["credential_ref"]; !ok {
		t.Fatal("AgentSourceMetadata must expose only the credential reference name")
	}
	approved := schema(t, document, "ApprovedSourceConfig")
	if value, ok := approved["additionalProperties"]; !ok || value != false {
		t.Fatalf("ApprovedSourceConfig additionalProperties = %v, want false", value)
	}

	reviewed := schema(t, document, "ReviewedEventImportRequest")
	assertRequired(t, reviewed, "idempotency_key", "package_id", "raw_documents", "event", "event_sources", "event_tags", "review")
	paths := object(t, document["paths"], "paths")
	reviewedOperation := object(t, object(t, paths[namespace+"/reviewed-event-imports"], "reviewed event path")["post"], "reviewed event operation")
	assertString(t, reviewedOperation, "x-receipt-schema", "event_import_receipts")
	assertString(t, reviewedOperation, "x-transaction-membership", "raw-document,event,event-source,event-tag,event-import-receipt")
	responses := object(t, reviewedOperation["responses"], "reviewed event responses")
	if _, ok := responses["409"]; !ok {
		t.Fatal("reviewed event import must define idempotency conflict 409")
	}
}

func TestOpenAPIContractHasNoDanglingLocalReferences(t *testing.T) {
	document := loadContract(t)
	walkContract(t, document, document, "document")
}

func loadContract(t *testing.T) map[string]any {
	t.Helper()
	content, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatalf("read openapi.yaml: %v", err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(content, &document); err != nil {
		t.Fatalf("parse openapi.yaml: %v", err)
	}
	return document
}

func schema(t *testing.T, document map[string]any, name string) map[string]any {
	t.Helper()
	components := object(t, document["components"], "components")
	schemas := object(t, components["schemas"], "schemas")
	return object(t, schemas[name], "schema "+name)
}

func refName(t *testing.T, value map[string]any) string {
	t.Helper()
	ref := stringValue(t, value["$ref"], "$ref")
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		t.Fatalf("$ref = %q, want schema reference", ref)
	}
	return strings.TrimPrefix(ref, prefix)
}

func object(t *testing.T, value any, label string) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s = %#v, want object", label, value)
	}
	return result
}

func array(t *testing.T, value any, label string) []any {
	t.Helper()
	result, ok := value.([]any)
	if !ok {
		t.Fatalf("%s = %#v, want array", label, value)
	}
	return result
}

func assertString(t *testing.T, value map[string]any, key, want string) {
	t.Helper()
	if got := stringValue(t, value[key], key); got != want {
		t.Fatalf("%s = %q, want %q", key, got, want)
	}
}

func stringValue(t *testing.T, value any, label string) string {
	t.Helper()
	result, ok := value.(string)
	if !ok {
		t.Fatalf("%s = %#v, want string", label, value)
	}
	return result
}

func assertInt(t *testing.T, value map[string]any, key string, want int) {
	t.Helper()
	got, ok := value[key].(int)
	if !ok || got != want {
		t.Fatalf("%s = %#v, want %d", key, value[key], want)
	}
}

func assertRequired(t *testing.T, value map[string]any, wanted ...string) {
	t.Helper()
	assertStringSet(t, value["required"], wanted...)
}

func assertStringSet(t *testing.T, value any, wanted ...string) {
	t.Helper()
	items := array(t, value, "string set")
	got := make([]string, 0, len(items))
	for _, item := range items {
		got = append(got, stringValue(t, item, "string set item"))
	}
	sort.Strings(got)
	want := append([]string(nil), wanted...)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("string set = %v, want %v", got, want)
	}
}

func sortedKeys(value map[string]any) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func walkContract(t *testing.T, document map[string]any, value any, path string) {
	t.Helper()
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			childPath := path + "." + key
			if key == "$ref" {
				ref := stringValue(t, child, childPath)
				const prefix = "#/components/"
				if !strings.HasPrefix(ref, prefix) {
					t.Fatalf("%s = %q, only local component references are allowed", childPath, ref)
				}
				parts := strings.Split(strings.TrimPrefix(ref, prefix), "/")
				if len(parts) != 2 {
					t.Fatalf("%s = %q, want #/components/<section>/<name>", childPath, ref)
				}
				components := object(t, document["components"], "components")
				section := object(t, components[parts[0]], "components."+parts[0])
				if _, exists := section[parts[1]]; !exists {
					t.Fatalf("%s references missing component %q", childPath, ref)
				}
				continue
			}
			walkContract(t, document, child, childPath)
		}
	case []any:
		for _, child := range typed {
			walkContract(t, document, child, path+"[]")
		}
	}
}
