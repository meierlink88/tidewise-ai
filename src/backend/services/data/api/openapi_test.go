package api_test

import (
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const namespace = "/api/data/v1"

type operationContract struct {
	method      string
	operationID string
	driftAnchor string
	scope       string
}

func TestOpenAPIContractFreezesNamespacePathsOperationsAndScopes(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	want := map[string]operationContract{
		"/healthz":                     {method: "get", operationID: "getDataServiceHealth"},
		"/readyz":                      {method: "get", operationID: "getDataServiceReadiness"},
		namespace + "/research/themes": {method: "get", operationID: "listResearchThemes", driftAnchor: "data.v1.listResearchThemes", scope: "data.research.read"},
		namespace + "/research/themes/{theme_id}":                             {method: "get", operationID: "getResearchTheme", driftAnchor: "data.v1.getResearchTheme", scope: "data.research.read"},
		namespace + "/research/themes/{theme_id}/reasoning-trees":             {method: "get", operationID: "listResearchThemeReasoningTrees", driftAnchor: "data.v1.listResearchThemeReasoningTrees", scope: "data.research.read"},
		namespace + "/research/themes/{theme_id}/reasoning-trees/{anchor_id}": {method: "get", operationID: "getResearchThemeReasoningTree", driftAnchor: "data.v1.getResearchThemeReasoningTree", scope: "data.research.read"},
		namespace + "/raw-documents":                                          {method: "get", operationID: "listAdminRawDocuments", driftAnchor: "data.v1.listAdminRawDocuments", scope: "data.admin.read"},
		namespace + "/events":                                                 {method: "get", operationID: "listAdminEvents", driftAnchor: "data.v1.listAdminEvents", scope: "data.admin.read"},
		namespace + "/reviewed-event-imports":                                 {method: "post", operationID: "publishReviewedEvents", driftAnchor: "data.v1.publishReviewedEvents", scope: "data.reviewed-events.import"},
		namespace + "/research-theme-imports":                                 {method: "post", operationID: "importResearchThemes", driftAnchor: "data.v1.importResearchThemes", scope: "data.research.import"},
		namespace + "/research-anchor-imports":                                {method: "post", operationID: "importResearchAnchors", driftAnchor: "data.v1.importResearchAnchors", scope: "data.research.import"},
	}

	if len(paths) != len(want) {
		t.Fatalf("path count = %d, want %d; got %v", len(paths), len(want), sortedKeys(paths))
	}
	for path, expected := range want {
		if path != "/healthz" && path != "/readyz" && !strings.HasPrefix(path, namespace+"/") {
			t.Fatalf("path %q escapes supported Data namespaces", path)
		}
		pathItem := object(t, paths[path], "path "+path)
		operation := object(t, pathItem[expected.method], expected.method+" "+path)
		assertString(t, operation, "operationId", expected.operationID)
		if expected.driftAnchor != "" {
			assertString(t, operation, "x-client-drift-anchor", expected.driftAnchor)
		}
		if expected.scope != "" {
			assertString(t, operation, "x-required-service-scope", expected.scope)
		}
		for _, method := range []string{"get", "post", "put", "patch", "delete"} {
			if method != expected.method {
				if _, exists := pathItem[method]; exists {
					t.Fatalf("path %q unexpectedly defines %s", path, method)
				}
			}
		}
	}
}

func TestOpenAPIContractFreezesResearchReasoningTreeReadV1(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	for _, legacy := range []string{namespace + "/research/anchors", namespace + "/research/anchors/{anchor_id}"} {
		if _, exists := paths[legacy]; exists {
			t.Fatalf("legacy research Anchor path remains in OpenAPI: %s", legacy)
		}
	}

	for _, path := range []string{
		namespace + "/research/themes/{theme_id}/reasoning-trees",
		namespace + "/research/themes/{theme_id}/reasoning-trees/{anchor_id}",
	} {
		operation := object(t, object(t, paths[path], "path "+path)["get"], "GET "+path)
		parameters := array(t, operation["parameters"], "reasoning tree operation parameters")
		if len(parameters) != 1 || stringValue(t, object(t, parameters[0], "request ID parameter")["$ref"], "$ref") != "#/components/parameters/RequestID" {
			t.Fatalf("GET %s must accept only X-Request-ID at operation level: %v", path, parameters)
		}
		responses := object(t, operation["responses"], "reasoning tree responses")
		for _, status := range []string{"200", "400", "401", "403", "404", "500"} {
			if _, exists := responses[status]; !exists {
				t.Fatalf("GET %s missing response %s", path, status)
			}
		}
	}

	list := schema(t, document, "ResearchReasoningTreeList")
	assertRequired(t, list, "theme", "reasoning_trees")
	listProperties := object(t, list["properties"], "ResearchReasoningTreeList properties")
	trees := object(t, listProperties["reasoning_trees"], "reasoning_trees")
	assertInt(t, trees, "minItems", 1)
	assertString(t, object(t, trees["items"], "reasoning tree summary items"), "$ref", "#/components/schemas/ResearchReasoningTreeSummary")

	tree := schema(t, document, "ResearchReasoningTree")
	assertRequired(t, tree, "anchor_id", "center_chain_node", "one_line_conclusion", "fact_summary", "net_direction_summary", "support_summary", "counter_summary", "trading_direction", "next_checkpoint", "event_count", "events", "path_nodes")
	treeProperties := object(t, tree["properties"], "ResearchReasoningTree properties")
	assertInt(t, object(t, treeProperties["events"], "events"), "minItems", 1)
	assertInt(t, object(t, treeProperties["path_nodes"], "path_nodes"), "minItems", 2)

	pathNode := schema(t, document, "ResearchReasoningTreePathNode")
	pathProperties := object(t, pathNode["properties"], "ResearchReasoningTreePathNode properties")
	assertStringSet(t, object(t, pathProperties["change_direction"], "change_direction")["enum"], "increase", "decrease", "mixed", "unchanged", "uncertain")
	event := schema(t, document, "ResearchReasoningTreeEvent")
	eventProperties := object(t, event["properties"], "ResearchReasoningTreeEvent properties")
	assertStringSet(t, object(t, eventProperties["evidence_role"], "evidence_role")["enum"], "driver", "supporting", "contradicting", "context")
}

func TestOpenAPIContractFreezesResearchAnchorPublicationV1(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	operation := object(t, object(t, paths[namespace+"/research-anchor-imports"], "research Anchor import path")["post"], "research Anchor import operation")
	assertString(t, operation, "x-canonicalization", "rfc8785-sha256")
	assertString(t, operation, "x-atomicity", "whole-theme-single-postgresql-transaction")
	assertString(t, operation, "x-receipt-schema", "research_anchor_import_receipts")
	assertString(t, operation, "x-retry-policy", "idempotent-with-theme-id")

	request := schema(t, document, "ResearchAnchorImportRequest")
	assertRequired(t, request, "theme_id", "anchors")
	properties := object(t, request["properties"], "ResearchAnchorImportRequest properties")
	for _, forbidden := range []string{"publisher_subject", "published_at", "imported_at", "idempotency_key"} {
		if _, exists := properties[forbidden]; exists {
			t.Fatalf("ResearchAnchorImportRequest must not expose %q", forbidden)
		}
	}
	anchors := object(t, properties["anchors"], "anchors")
	assertInt(t, anchors, "minItems", 1)

	anchor := schema(t, document, "ResearchAnchorImportItem")
	assertRequired(t, anchor, "center_chain_node_id", "one_line_conclusion", "fact_summary", "net_direction_summary", "support_summary", "counter_summary", "trading_direction", "next_checkpoint", "events", "path_nodes")
	anchorProperties := object(t, anchor["properties"], "ResearchAnchorImportItem properties")
	for _, forbidden := range []string{"anchor_id", "anchor_type", "importance", "indices", "transmission_path"} {
		if _, exists := anchorProperties[forbidden]; exists {
			t.Fatalf("ResearchAnchorImportItem must not expose %q", forbidden)
		}
	}

	event := schema(t, document, "ResearchAnchorImportEvent")
	assertRequired(t, event, "event_id", "evidence_role", "evidence_summary")
	assertStringSet(t, object(t, object(t, event["properties"], "event properties")["evidence_role"], "evidence_role")["enum"], "driver", "supporting", "contradicting", "context")
	pathNode := schema(t, document, "ResearchAnchorImportPathNode")
	assertRequired(t, pathNode, "chain_node_id", "change_direction", "change_summary", "impact_summary", "incoming_transmission_mechanism")
	assertStringSet(t, object(t, object(t, pathNode["properties"], "path node properties")["change_direction"], "change_direction")["enum"], "increase", "decrease", "mixed", "unchanged", "uncertain")

	result := schema(t, document, "ResearchAnchorImportResult")
	assertRequired(t, result, "receipt_id", "theme_id", "payload_hash", "anchor_ids_by_center_chain_node_id", "counts", "published_at", "imported_at", "replayed")
}

func TestOpenAPIContractFreezesResearchThemeBatchPublicationV1(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	operation := object(t, object(t, paths[namespace+"/research-theme-imports"], "research Theme import path")["post"], "research Theme import operation")
	assertString(t, operation, "x-canonicalization", "rfc8785-sha256")
	assertString(t, operation, "x-atomicity", "whole-batch-single-postgresql-transaction")
	assertString(t, operation, "x-receipt-schema", "research_theme_import_receipts")
	assertString(t, operation, "x-retry-policy", "idempotent-with-analysis-batch-id")

	request := schema(t, document, "ResearchThemeImportRequest")
	assertRequired(t, request, "analysis_batch_id", "window_start", "window_end", "themes")
	properties := object(t, request["properties"], "ResearchThemeImportRequest properties")
	for _, forbidden := range []string{"idempotency_key", "publisher_subject", "published_at", "confidence", "market_confirmation"} {
		if _, exists := properties[forbidden]; exists {
			t.Fatalf("ResearchThemeImportRequest must not expose %q", forbidden)
		}
	}
	themes := object(t, properties["themes"], "themes")
	assertInt(t, themes, "minItems", 1)

	theme := schema(t, document, "ResearchThemeImportItem")
	assertRequired(t, theme,
		"theme_key", "name", "one_line_conclusion", "impact_level", "transmission_path",
		"trading_direction", "transmission_stage", "next_checkpoint", "market_confirmation_summary",
		"chain_nodes", "events",
	)
	themeProperties := object(t, theme["properties"], "ResearchThemeImportItem properties")
	for _, forbidden := range []string{"id", "event_ids", "chain_node_ids", "indices", "index_entity_ids", "confidence", "causal_chain", "research_direction", "confirmation_conditions"} {
		if _, exists := themeProperties[forbidden]; exists {
			t.Fatalf("ResearchThemeImportItem must not expose %q", forbidden)
		}
	}
	assertString(t, object(t, themeProperties["theme_key"], "theme_key"), "pattern", "^[a-z0-9][a-z0-9._:-]{0,127}$")
	assertStringSet(t, object(t, themeProperties["impact_level"], "impact_level")["enum"], "high", "focus", "watch")
	assertStringSet(t, object(t, themeProperties["transmission_stage"], "transmission_stage")["enum"], "identification", "validation", "diffusion", "dampening")

	chainNode := schema(t, document, "ResearchThemeImportChainNode")
	assertRequired(t, chainNode, "chain_node_id", "relation_role", "impact_summary")
	lowercaseUUID := schema(t, document, "LowercaseUUID")
	assertString(t, lowercaseUUID, "pattern", "^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$")
	assertString(t, object(t, object(t, chainNode["properties"], "chain node properties")["chain_node_id"], "chain_node_id"), "$ref", "#/components/schemas/LowercaseUUID")
	assertStringSet(t, object(t, object(t, chainNode["properties"], "chain node properties")["relation_role"], "relation_role")["enum"], "driver", "beneficiary", "constraint", "exposure")
	event := schema(t, document, "ResearchThemeImportEvent")
	assertRequired(t, event, "event_id", "evidence_role", "supported_claim")
	assertString(t, object(t, object(t, event["properties"], "event properties")["event_id"], "event_id"), "$ref", "#/components/schemas/LowercaseUUID")
	assertStringSet(t, object(t, object(t, event["properties"], "event properties")["evidence_role"], "evidence_role")["enum"], "driver", "supporting", "contradicting", "context")

	result := schema(t, document, "ResearchThemeImportResult")
	assertRequired(t, result, "receipt_id", "analysis_batch_id", "payload_hash", "theme_ids_by_key", "counts", "published_at", "imported_at", "replayed")
	resultProperties := object(t, result["properties"], "ResearchThemeImportResult properties")
	if value := object(t, resultProperties["theme_ids_by_key"], "theme_ids_by_key")["additionalProperties"]; value == nil {
		t.Fatal("theme_ids_by_key must define UUID map values")
	}
}

func TestOpenAPIContractFreezesBearerIdentityRequestIDAndStructuredErrors(t *testing.T) {
	document := loadContract(t)
	if document["openapi"] != "3.0.4" {
		t.Fatalf("openapi = %v, want 3.0.4", document["openapi"])
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
		if path == "/healthz" || path == "/readyz" {
			continue
		}
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

func TestOpenAPIContractFreezesEventPublication(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	operation := object(t, object(t, paths[namespace+"/reviewed-event-imports"], "Event Publication path")["post"], "Event Publication operation")
	assertString(t, operation, "x-atomicity", "whole-batch-single-postgresql-transaction")
	assertString(t, operation, "x-receipt-schema", "event_publication_receipts")
	assertString(t, operation, "x-retry-policy", "retry-failed-call-with-natural-identities")
	requestBody := object(t, operation["requestBody"], "Event Publication request body")
	media := object(t, object(t, requestBody["content"], "request content")["application/json"], "request media")
	requestSchema := object(t, media["schema"], "request schema")
	assertInt(t, requestSchema, "x-max-body-bytes", 1048576)

	request := schema(t, document, "EventPublicationRequest")
	assertRequired(t, request, "package_id", "provenance", "raw_documents", "events")
	requestProperties := object(t, request["properties"], "EventPublicationRequest properties")
	for _, forbidden := range []string{"idempotency_key", "payload_hash", "caller_subject", "content_text", "artifact_uri"} {
		if _, exists := requestProperties[forbidden]; exists {
			t.Fatalf("EventPublicationRequest exposes forbidden field %q", forbidden)
		}
	}
	events := object(t, requestProperties["events"], "publication events")
	assertInt(t, events, "minItems", 1)
	assertInt(t, events, "maxItems", 10)

	raw := schema(t, document, "EventPublicationRawDocument")
	assertRequired(t, raw, "artifact_id", "content_sha256", "source_ref", "source_name", "source_type", "title", "collected_at")
	rawProperties := object(t, raw["properties"], "raw document properties")
	for _, forbidden := range []string{"content_text", "artifact_uri", "ingest_channel", "ingest_status", "content_level", "source_external_id"} {
		if _, exists := rawProperties[forbidden]; exists {
			t.Fatalf("EventPublicationRawDocument exposes forbidden field %q", forbidden)
		}
	}
	assertInt(t, object(t, rawProperties["source_type"], "source_type"), "maxLength", 64)
	assertInt(t, object(t, rawProperties["language"], "language"), "maxLength", 16)
	assertInt(t, object(t, rawProperties["mime_type"], "mime_type"), "maxLength", 128)
	event := schema(t, document, "EventPublicationEvent")
	assertRequired(t, event, "dedupe_key", "title", "factual_summary", "fact_payload", "evidence", "tags", "review")
	for _, forbidden := range []string{"event_status", "fact_status"} {
		if _, exists := object(t, event["properties"], "event properties")[forbidden]; exists {
			t.Fatalf("EventPublicationEvent lets caller submit %q", forbidden)
		}
	}
	result := schema(t, document, "EventPublicationResult")
	assertRequired(t, result, "receipt_id", "package_id", "imported_at", "events", "raw_documents", "counts")
}

func TestOpenAPIContractFreezesDTOFormatsEnumsAndSensitiveMetadataBoundary(t *testing.T) {
	document := loadContract(t)
	for _, name := range []string{
		"ResearchThemeCollection", "ResearchThemeDetail", "ResearchReasoningTreeList", "ResearchReasoningTreeDetail",
		"AdminRawDocumentPage", "AdminEventPage", "ErrorEnvelope",
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

	assertStringSet(t, schema(t, document, "EventStatus")["enum"], "candidate", "confirmed", "rejected")
	assertStringSet(t, schema(t, document, "FactStatus")["enum"], "unverified", "verified", "disputed")
	assertStringSet(t, schema(t, document, "EventPublicationDisposition")["enum"], "created", "reused")
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
