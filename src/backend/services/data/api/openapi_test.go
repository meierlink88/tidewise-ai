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
		namespace + "/research/themes":                                        {method: "get", operationID: "listResearchThemes", scope: "data.research.read"},
		namespace + "/research/themes/{theme_id}":                             {method: "get", operationID: "getResearchTheme", scope: "data.research.read"},
		namespace + "/research/themes/{theme_id}/reasoning-trees":             {method: "get", operationID: "listResearchThemeReasoningTrees", scope: "data.research.read"},
		namespace + "/research/themes/{theme_id}/reasoning-trees/{anchor_id}": {method: "get", operationID: "getResearchThemeReasoningTree", scope: "data.research.read"},
		namespace + "/admin/raw-documents":                                    {method: "get", operationID: "listAdminRawDocuments", scope: "data.admin.read"},
		namespace + "/admin/events":                                           {method: "get", operationID: "listAdminEvents", scope: "data.admin.read"},
		namespace + "/admin/source-catalogs":                                  {method: "get", operationID: "listAdminSourceCatalogs", scope: "data.admin.read"},
		namespace + "/agent-run/source-metadata":                              {method: "get", operationID: "listAgentSourceMetadata", scope: "data.source-metadata.read"},
		namespace + "/raw-document-imports":                                   {method: "post", operationID: "importRawDocuments", scope: "data.raw-documents.import"},
		namespace + "/raw-document-imports/{idempotency_key}":                 {method: "get", operationID: "getRawDocumentImportStatus", scope: "data.raw-documents.import"},
		namespace + "/reviewed-event-imports":                                 {method: "post", operationID: "importReviewedEvent", scope: "data.reviewed-events.import"},
		namespace + "/research-theme-imports":                                 {method: "post", operationID: "importResearchThemes", scope: "data.research.import"},
		namespace + "/research-anchor-imports":                                {method: "post", operationID: "importResearchAnchors", scope: "data.research.import"},
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
		"ResearchThemeCollection", "ResearchThemeDetail", "ResearchReasoningTreeList", "ResearchReasoningTreeDetail",
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
