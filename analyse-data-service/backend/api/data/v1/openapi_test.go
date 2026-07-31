package v1_test

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
		namespace + "/event-tags":      {method: "get", operationID: "listActiveEventTags", driftAnchor: "data.v1.listActiveEventTags", scope: "data.event-tags.read"},
		namespace + "/research/themes": {method: "get", operationID: "listResearchThemes", driftAnchor: "data.v1.listResearchThemes", scope: "data.research.read"},
		namespace + "/research/themes/{theme_id}":                                     {method: "get", operationID: "getResearchTheme", driftAnchor: "data.v1.getResearchTheme", scope: "data.research.read"},
		namespace + "/research/themes/{theme_id}/reasoning-trees":                     {method: "get", operationID: "listResearchThemeReasoningTrees", driftAnchor: "data.v1.listResearchThemeReasoningTrees", scope: "data.research.read"},
		namespace + "/research/themes/{theme_id}/reasoning-trees/{reasoning_tree_id}": {method: "get", operationID: "getResearchThemeReasoningTree", driftAnchor: "data.v1.getResearchThemeReasoningTree", scope: "data.research.read"},
		namespace + "/raw-documents":                                                  {method: "get", operationID: "listAdminRawDocuments", driftAnchor: "data.v1.listAdminRawDocuments", scope: "data.admin.read"},
		namespace + "/events":                                                         {method: "get", operationID: "listAdminEvents", driftAnchor: "data.v1.listAdminEvents", scope: "data.admin.read"},
		namespace + "/event-semantics/eligible-events":                                {method: "get", operationID: "listEligibleEventSemanticEvents", driftAnchor: "data.v1.listEligibleEventSemanticEvents", scope: "data.event-semantics.read"},
		namespace + "/event-semantics/context-leases":                                 {method: "post", operationID: "createEventSemanticContextLease", driftAnchor: "data.v1.createEventSemanticContextLease", scope: "data.event-semantics.write"},
		namespace + "/event-semantics/context-leases/{context_lease_id}/context":      {method: "get", operationID: "getEventSemanticContext", driftAnchor: "data.v1.getEventSemanticContext", scope: "data.event-semantics.read"},
		namespace + "/event-semantics/entity-resolutions":                             {method: "post", operationID: "resolveEventSemanticEntities", driftAnchor: "data.v1.resolveEventSemanticEntities", scope: "data.event-semantics.write"},
		namespace + "/event-semantics/direct-targets:search":                          {method: "post", operationID: "searchEventSemanticDirectTargets", driftAnchor: "data.v1.searchEventSemanticDirectTargets", scope: "data.event-semantics.write"},
		namespace + "/event-semantics/resolution-routes:list":                         {method: "post", operationID: "listEventSemanticResolutionRoutes", driftAnchor: "data.v1.listEventSemanticResolutionRoutes", scope: "data.event-semantics.write"},
		namespace + "/event-semantics/resolution-anchors:list":                        {method: "post", operationID: "listEventSemanticResolutionAnchors", driftAnchor: "data.v1.listEventSemanticResolutionAnchors", scope: "data.event-semantics.write"},
		namespace + "/event-semantics/chain-node-candidates:resolve":                  {method: "post", operationID: "resolveEventSemanticChainNodeCandidates", driftAnchor: "data.v1.resolveEventSemanticChainNodeCandidates", scope: "data.event-semantics.write"},
		namespace + "/event-semantics/submissions":                                    {method: "post", operationID: "createEventSemanticSubmission", driftAnchor: "data.v1.createEventSemanticSubmission", scope: "data.event-semantics.write"},
		namespace + "/event-semantics/submissions/{submission_id}/reviews":            {method: "post", operationID: "submitEventSemanticReview", driftAnchor: "data.v1.submitEventSemanticReview", scope: "data.event-semantics.write"},
		namespace + "/events/{event_id}/semantics":                                    {method: "get", operationID: "getEventSemantics", driftAnchor: "data.v1.getEventSemantics", scope: "data.event-semantics.read"},
		namespace + "/research-analysis-context":                                      {method: "get", operationID: "listResearchAnalysisContext", driftAnchor: "data.v1.listResearchAnalysisContext", scope: "data.research.read"},
		namespace + "/research-graph:search":                                          {method: "post", operationID: "searchResearchGraph", driftAnchor: "data.v1.searchResearchGraph", scope: "data.research.read"},
		namespace + "/reviewed-event-imports":                                         {method: "post", operationID: "publishReviewedEvents", driftAnchor: "data.v1.publishReviewedEvents", scope: "data.reviewed-events.import"},
		namespace + "/research-theme-imports":                                         {method: "post", operationID: "publishResearchTheme", driftAnchor: "data.v1.publishResearchTheme", scope: "data.research.import"},
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

func TestEventSemanticManifestReadersDeclareRequestIDAndContextDrift(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	operations := map[string]string{
		namespace + "/event-semantics/context-leases/{context_lease_id}/context": "get",
		namespace + "/event-semantics/entity-resolutions":                        "post",
		namespace + "/event-semantics/direct-targets:search":                     "post",
		namespace + "/event-semantics/resolution-routes:list":                    "post",
		namespace + "/event-semantics/resolution-anchors:list":                   "post",
		namespace + "/event-semantics/chain-node-candidates:resolve":             "post",
	}
	for path, method := range operations {
		operation := object(t, object(t, paths[path], path)[method], method+" "+path)
		parameters := array(t, operation["parameters"], path+" parameters")
		if stringValue(t, object(t, parameters[0], "request ID parameter")["$ref"], "$ref") != "#/components/parameters/RequestID" {
			t.Fatalf("%s %s must accept the unified X-Request-ID parameter first: %v", method, path, parameters)
		}
		responses := object(t, operation["responses"], path+" responses")
		drift := object(t, responses["409"], "409 response")
		if stringValue(t, drift["$ref"], "$ref") != "#/components/responses/EventSemanticContextDrift" {
			t.Fatalf("%s %s 409 response = %v", method, path, drift)
		}
	}
	detail := schema(t, document, "EventSemanticContextDriftErrorDetail")
	code := object(t, object(t, detail["properties"], "drift properties")["code"], "drift code")
	assertStringSet(t, code["enum"], "EVENT_SEMANTIC_CONTEXT_DRIFT")
}

func TestOpenAPIContractFreezesActiveEventTagCatalog(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	operation := object(t, object(t, paths[namespace+"/event-tags"], "Event Tag Catalog path")["get"], "Event Tag Catalog operation")
	assertString(t, operation, "x-retry-policy", "safe-get")
	assertString(t, operation, "x-required-service-scope", "data.event-tags.read")

	parameters := array(t, operation["parameters"], "Event Tag Catalog parameters")
	if len(parameters) != 2 {
		t.Fatalf("Event Tag Catalog parameter count = %d, want 2", len(parameters))
	}
	active := object(t, parameters[1], "active parameter")
	assertString(t, active, "name", "active")
	assertStringSet(t, object(t, active["schema"], "active schema")["enum"], "true")

	catalog := schema(t, document, "EventTagCatalog")
	assertRequired(t, catalog, "catalog_revision", "catalog_hash", "tags")
	properties := object(t, catalog["properties"], "EventTagCatalog properties")
	assertString(t, object(t, properties["catalog_hash"], "catalog_hash"), "$ref", "#/components/schemas/PayloadHash")
	tags := object(t, properties["tags"], "tags")
	assertInt(t, tags, "minItems", 1)

	tag := schema(t, document, "EventTagCatalogItem")
	assertRequired(t, tag, "id", "tag_kind", "code", "name", "is_active")
	assertString(t, object(t, object(t, tag["properties"], "tag properties")["tag_kind"], "tag_kind"), "$ref", "#/components/schemas/TagKind")
}

func TestOpenAPIContractFreezesResearchReasoningTreeReadV1(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	for _, legacy := range []string{namespace + "/research/anchors", namespace + "/research/anchors/{anchor_id}", namespace + "/research-anchor-imports"} {
		if _, exists := paths[legacy]; exists {
			t.Fatalf("legacy research Anchor path remains in OpenAPI: %s", legacy)
		}
	}

	for _, path := range []string{
		namespace + "/research/themes/{theme_id}/reasoning-trees",
		namespace + "/research/themes/{theme_id}/reasoning-trees/{reasoning_tree_id}",
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
	assertString(t, object(t, trees["items"], "reasoning tree summary items"), "$ref", "#/components/schemas/ResearchReasoningTreeSummary")

	tree := schema(t, document, "ResearchReasoningTree")
	assertRequired(t, tree,
		"reasoning_tree_id", "theme_id", "industry_chain_entity_id", "industry_chain_name", "title",
		"display_order", "one_line_conclusion", "fact_summary", "transmission_summary",
		"impact_direction", "impact_strength", "impact_summary", "conclusion_boundary_summary",
		"support_summary", "counter_summary", "invalidation_conditions", "checkpoints",
		"published_at", "event_count", "events", "nodes",
	)
	node := schema(t, document, "ResearchReasoningTreeNode")
	assertRequired(t, node,
		"id", "position", "chain_node_entity_id", "name", "state_summary", "impact_direction",
		"impact_strength", "impact_summary", "reasoning_basis_summary", "evidence_gap_summary",
		"incoming_industry_chain_graph_edge_id", "incoming_transmission_title",
		"incoming_transmission_mechanism", "incoming_condition_summary", "incoming_graph_edge",
		"signals", "primary_signal", "signal_display_summary",
	)
	detail := schema(t, document, "ResearchReasoningTreeDetail")
	assertRequired(t, detail, "theme_id", "impact_node_ids", "reasoning_tree")
}

func TestOpenAPIContractFreezesAtomicResearchPublicationV2(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	if _, exists := paths[namespace+"/research-reasoning-tree-imports"]; exists {
		t.Fatal("Reason Tree must not have an independent publication endpoint")
	}
	operation := object(t, object(t, paths[namespace+"/research-theme-imports"], "research Theme import path")["post"], "research Theme import operation")
	assertString(t, operation, "x-canonicalization", "rfc8785-sha256")
	assertString(t, operation, "x-atomicity", "one-theme-and-all-reason-trees-single-postgresql-transaction")
	assertString(t, operation, "x-receipt-schema", "research_theme_import_receipts")
	assertString(t, operation, "x-retry-policy", "idempotent-with-analysis-batch-id")

	request := schema(t, document, "ResearchThemeImportRequest")
	assertRequired(t, request, "analysis_batch_id", "analysis_as_of", "discovery_window_start", "discovery_window_end", "theme", "reasoning_trees")
	tree := schema(t, document, "ResearchReasoningTreeImportItem")
	assertRequired(t, tree,
		"industry_chain_entity_id", "title", "display_order", "one_line_conclusion", "fact_summary",
		"transmission_summary", "impact_direction", "impact_strength", "impact_summary",
		"conclusion_boundary_summary", "support_summary", "counter_summary",
		"invalidation_conditions", "checkpoints", "events", "nodes",
	)
	node := schema(t, document, "ResearchReasoningTreeImportNode")
	assertRequired(t, node,
		"position", "chain_node_entity_id", "state_summary", "impact_direction", "impact_strength",
		"impact_summary", "reasoning_basis_summary", "evidence_gap_summary",
		"incoming_industry_chain_graph_edge_id", "incoming_transmission_title",
		"incoming_transmission_mechanism", "incoming_condition_summary", "incoming_lineage", "signals",
	)
	signal := schema(t, document, "ResearchReasoningTreeImportSignal")
	assertRequired(t, signal, "variable_signal_key", "signal_role", "signal_direction", "display_summary", "display_order", "lineage")
	assertRequired(t, schema(t, document, "ResearchReasoningTreeSignalLineage"),
		"source_kind", "variable_signal_id", "semantic_submission_id", "evidence_id", "evidence_hash",
		"upstream_variable_signal_id", "upstream_direct_impact_assertion_id", "entity_relation_id",
		"industry_chain_graph_edge_id",
	)
	result := schema(t, document, "ResearchThemeImportResult")
	assertRequired(t, result, "receipt_id", "analysis_batch_id", "theme_id", "payload_hash", "reasoning_tree_ids_by_industry_chain_entity_id", "counts", "published_at", "imported_at", "replayed")
}

func TestOpenAPIContractFreezesCorrectedResearchAnalysisContextV1(t *testing.T) {
	document := loadContract(t)
	contextSchema := schema(t, document, "ResearchAnalysisContext")
	assertRequired(t, contextSchema,
		"contract_version", "tbox_contract_version", "temporal_semantics", "temporal_limitation",
		"event_page_fingerprint", "reference_closure_fingerprint", "discovery_window_start",
		"discovery_window_end", "analysis_as_of", "event_semantic_bundles",
		"dictionaries", "has_more",
	)
	contextProperties := object(t, contextSchema["properties"], "ResearchAnalysisContext properties")
	assertStringSet(t, object(t, contextProperties["contract_version"], "contract_version")["enum"], "research-analysis-context.v1")
	if _, exists := contextProperties["dictionary_fingerprint"]; exists {
		t.Fatal("corrected Research Analysis Context v1 must not expose the global dictionary fingerprint")
	}
	paths := object(t, document["paths"], "paths")
	operation := object(t, object(t, paths[namespace+"/research-analysis-context"], "Analysis Context path")["get"], "Analysis Context operation")
	responses := object(t, operation["responses"], "Analysis Context responses")
	assertString(t, object(t, responses["409"], "Analysis Context 409"), "$ref", "#/components/responses/ResearchAnalysisContextInconsistent")
	dictionaries := schema(t, document, "ResearchAnalysisDictionaries")
	assertRequired(t, dictionaries,
		"entities", "relation_definitions", "entity_relations", "industry_chains",
		"industry_chain_memberships", "industry_chain_graph_edges",
		"entity_type_definitions", "variable_definitions",
		"direct_transmission_rules", "acceptance_policies",
	)
	if additional, ok := dictionaries["additionalProperties"].(bool); !ok || additional {
		t.Fatalf("ResearchAnalysisDictionaries additionalProperties = %#v, want false", dictionaries["additionalProperties"])
	}
}

func TestOpenAPIContractFreezesControlledResearchGraphSearchV1(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	operation := object(t, object(t, paths[namespace+"/research-graph:search"], "Research Graph path")["post"], "Research Graph operation")
	assertString(t, operation, "x-required-service-scope", "data.research.read")
	assertString(t, operation, "x-retry-policy", "safe-idempotent-read-reduce-scope-on-resource-limit")
	responses := object(t, operation["responses"], "Research Graph responses")
	assertString(t, object(t, responses["413"], "Research Graph 413"), "$ref", "#/components/responses/PayloadTooLarge")
	assertString(t, object(t, responses["429"], "Research Graph 429"), "$ref", "#/components/responses/ResearchResourceLimit")

	request := schema(t, document, "ResearchGraphSearchRequest")
	assertRequired(t, request,
		"analysis_as_of", "seed_entity_ids", "relation_filters",
		"max_depth", "node_budget", "edge_budget",
	)
	requestProperties := object(t, request["properties"], "ResearchGraphSearchRequest properties")
	assertInt(t, object(t, requestProperties["max_depth"], "max_depth"), "maximum", 5)
	assertInt(t, object(t, requestProperties["node_budget"], "node_budget"), "maximum", 500)
	assertInt(t, object(t, requestProperties["edge_budget"], "edge_budget"), "maximum", 1000)
	filter := schema(t, document, "ResearchGraphRelationFilter")
	assertRequired(t, filter, "relation_type", "direction")
	assertStringSet(t, object(t, object(t, filter["properties"], "filter properties")["direction"], "direction")["enum"], "outgoing", "incoming", "both")

	result := schema(t, document, "ResearchGraphSearchResult")
	assertRequired(t, result,
		"contract_version", "analysis_as_of", "query_fingerprint", "graph_fingerprint",
		"actual_depth", "entities", "relation_definitions", "entity_relations",
		"industry_chains", "industry_chain_memberships", "industry_chain_graph_edges",
	)
	details := schema(t, document, "ResearchResourceLimitDetails")
	assertRequired(t, details, "component", "retry_guidance")
}

func TestOpenAPIContractFreezesResearchThemeBatchPublicationV1(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	operation := object(t, object(t, paths[namespace+"/research-theme-imports"], "research Theme import path")["post"], "research Theme import operation")
	assertString(t, operation, "x-canonicalization", "rfc8785-sha256")
	assertString(t, operation, "x-atomicity", "one-theme-and-all-reason-trees-single-postgresql-transaction")
	assertString(t, operation, "x-receipt-schema", "research_theme_import_receipts")
	assertString(t, operation, "x-retry-policy", "idempotent-with-analysis-batch-id")

	request := schema(t, document, "ResearchThemeImportRequest")
	assertRequired(t, request, "analysis_batch_id", "analysis_as_of", "discovery_window_start", "discovery_window_end", "theme", "reasoning_trees")
	properties := object(t, request["properties"], "ResearchThemeImportRequest properties")
	for _, forbidden := range []string{"idempotency_key", "publisher_subject", "published_at", "confidence", "market_confirmation"} {
		if _, exists := properties[forbidden]; exists {
			t.Fatalf("ResearchThemeImportRequest must not expose %q", forbidden)
		}
	}
	reasoningTrees := object(t, properties["reasoning_trees"], "reasoning_trees")
	assertInt(t, reasoningTrees, "minItems", 1)

	theme := schema(t, document, "ResearchThemeImportItem")
	assertRequired(t, theme,
		"theme_key", "title", "one_line_conclusion", "conclusion_direction", "impact_strength",
		"attention_level", "conclusion_status", "transmission_stage", "investment_guidance_action",
		"investment_guidance_summary", "time_horizon_category", "time_horizon_summary",
		"transmission_summary", "checkpoint_summary", "risk_summary", "impacts", "events",
	)
	themeProperties := object(t, theme["properties"], "ResearchThemeImportItem properties")
	for _, forbidden := range []string{"id", "event_ids", "chain_node_ids", "indices", "index_entity_ids", "confidence", "causal_chain", "research_direction", "confirmation_conditions"} {
		if _, exists := themeProperties[forbidden]; exists {
			t.Fatalf("ResearchThemeImportItem must not expose %q", forbidden)
		}
	}
	assertString(t, object(t, themeProperties["theme_key"], "theme_key"), "pattern", "^[a-z0-9][a-z0-9._:-]{0,127}$")
	assertStringSet(t, object(t, themeProperties["transmission_stage"], "transmission_stage")["enum"], "identification", "validation", "diffusion", "dampening")

	impact := schema(t, document, "ResearchThemeImportImpact")
	assertRequired(t, impact, "chain_node_entity_id", "relation_role", "impact_direction", "impact_summary", "display_order")
	lowercaseUUID := schema(t, document, "LowercaseUUID")
	assertString(t, lowercaseUUID, "pattern", "^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$")
	assertString(t, object(t, object(t, impact["properties"], "impact properties")["chain_node_entity_id"], "chain_node_entity_id"), "$ref", "#/components/schemas/LowercaseUUID")
	assertStringSet(t, object(t, object(t, impact["properties"], "impact properties")["relation_role"], "relation_role")["enum"], "driver", "beneficiary", "constraint", "exposure")
	event := schema(t, document, "ResearchThemeImportEvent")
	assertRequired(t, event, "event_id", "evidence_role", "supported_claim")
	assertString(t, object(t, object(t, event["properties"], "event properties")["event_id"], "event_id"), "$ref", "#/components/schemas/LowercaseUUID")
	assertStringSet(t, object(t, object(t, event["properties"], "event properties")["evidence_role"], "evidence_role")["enum"], "driver", "supporting", "contradicting", "context")

	result := schema(t, document, "ResearchThemeImportResult")
	assertRequired(t, result, "receipt_id", "analysis_batch_id", "payload_hash", "theme_id", "reasoning_tree_ids_by_industry_chain_entity_id", "counts", "published_at", "imported_at", "replayed")
	resultProperties := object(t, result["properties"], "ResearchThemeImportResult properties")
	if value := object(t, resultProperties["reasoning_tree_ids_by_industry_chain_entity_id"], "reasoning tree IDs")["additionalProperties"]; value == nil {
		t.Fatal("reasoning_tree_ids_by_industry_chain_entity_id must define UUID map values")
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
