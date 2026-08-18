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
		"/healthz":                                {method: "get", operationID: "getDataServiceHealth"},
		"/readyz":                                 {method: "get", operationID: "getDataServiceReadiness"},
		namespace + "/evidence-categories":        {method: "get", operationID: "listEvidenceCategories", driftAnchor: "data.v1.listEvidenceCategories", scope: "data.evidence-categories.read"},
		namespace + "/research/themes":            {method: "get", operationID: "listResearchThemes", driftAnchor: "data.v1.listResearchThemes", scope: "data.research.read"},
		namespace + "/research/themes/{theme_id}": {method: "get", operationID: "getResearchTheme", driftAnchor: "data.v1.getResearchTheme", scope: "data.research.read"},
		namespace + "/research/themes/{theme_id}/reasoning-trees":                     {method: "get", operationID: "listResearchThemeReasoningTrees", driftAnchor: "data.v1.listResearchThemeReasoningTrees", scope: "data.research.read"},
		namespace + "/research/themes/{theme_id}/reasoning-trees/{reasoning_tree_id}": {method: "get", operationID: "getResearchThemeReasoningTree", driftAnchor: "data.v1.getResearchThemeReasoningTree", scope: "data.research.read"},
		namespace + "/events":                                                       {method: "get", operationID: "listAdminEvents", driftAnchor: "data.v1.listAdminEvents", scope: "data.admin.read"},
		namespace + "/runtime-health":                                               {method: "get", operationID: "getDataRuntimeHealth", driftAnchor: "data.v1.getRuntimeHealth", scope: "data.admin.read"},
		namespace + "/research-graph:search":                                        {method: "post", operationID: "searchResearchGraph", driftAnchor: "data.v1.searchResearchGraph", scope: "data.research.read"},
		namespace + "/raw-evidence-publications":                                    {method: "post", operationID: "publishRawEvidence", driftAnchor: "data.v1.publishRawEvidence", scope: "data.raw-evidences.import"},
		namespace + "/raw-evidences/{id}":                                           {method: "get", operationID: "getRawEvidence", driftAnchor: "data.v1.getRawEvidence", scope: "data.raw-evidences.read"},
		namespace + "/evidence-publications":                                        {method: "post", operationID: "publishEvidence", driftAnchor: "data.v1.publishEvidence", scope: "data.evidences.import"},
		namespace + "/research-theme-imports":                                       {method: "post", operationID: "publishResearchTheme", driftAnchor: "data.v1.publishResearchTheme", scope: "data.research.import"},
		namespace + "/entities/countries":                                           {method: "get", operationID: "listCountries", driftAnchor: "data.v1.listCountries", scope: "data.countries.read"},
		namespace + "/entities/countries/{country_id}":                              {method: "get", operationID: "getCountry", driftAnchor: "data.v1.getCountry", scope: "data.countries.read"},
		namespace + "/entities/countries/{country_id}/regions":                      {method: "put", operationID: "replaceCountryRegions", driftAnchor: "data.v1.replaceCountryRegions", scope: "data.countries.write"},
		namespace + "/entities/industries":                                          {method: "get", operationID: "listIndustries", driftAnchor: "data.v1.listIndustries", scope: "data.industries.read"},
		namespace + "/entities/industries/{industry_id}":                            {method: "get", operationID: "getIndustry", driftAnchor: "data.v1.getIndustry", scope: "data.industries.read"},
		namespace + "/entities/concepts":                                            {method: "get", operationID: "listConcepts", driftAnchor: "data.v1.listConcepts", scope: "data.concepts.read"},
		namespace + "/entities/concepts/{concept_id}":                               {method: "get", operationID: "getConcept", driftAnchor: "data.v1.getConcept", scope: "data.concepts.read"},
		namespace + "/entities/chain-nodes":                                         {method: "get", operationID: "listChainNodes", driftAnchor: "data.v1.listChainNodes", scope: "data.chain-nodes.read"},
		namespace + "/entities/chain-nodes/{chain_node_id}":                         {method: "get", operationID: "getChainNode", driftAnchor: "data.v1.getChainNode", scope: "data.chain-nodes.read"},
		namespace + "/entities/industry-chains":                                     {method: "get", operationID: "listIndustryChains", driftAnchor: "data.v1.listIndustryChains", scope: "data.industry-chains.read"},
		namespace + "/entities/industry-chains/{industry_chain_id}":                 {method: "get", operationID: "getIndustryChain", driftAnchor: "data.v1.getIndustryChain", scope: "data.industry-chains.read"},
		namespace + "/entities/organizations":                                       {method: "get", operationID: "listOrganizations", driftAnchor: "data.v1.listOrganizations", scope: "data.organizations.read"},
		namespace + "/entities/organizations/{organization_id}":                     {method: "get", operationID: "getOrganization", driftAnchor: "data.v1.getOrganization", scope: "data.organizations.read"},
		namespace + "/entities/organizations/{organization_id}/domain-tags":         {method: "put", operationID: "replaceOrganizationDomainTags", driftAnchor: "data.v1.replaceOrganizationDomainTags", scope: "data.organizations.write"},
		namespace + "/organization-catalog":                                         {method: "get", operationID: "getOrganizationCatalog", driftAnchor: "data.v1.getOrganizationCatalog", scope: "data.organizations.read"},
		namespace + "/entities/organizations/{organization_id}/members":             {method: "get", operationID: "listOrganizationMembers", driftAnchor: "data.v1.listOrganizationMembers", scope: "data.organizations.read"},
		namespace + "/entities/organizations/{organization_id}/members/{member_id}": {method: "put", operationID: "updateOrganizationMember", driftAnchor: "data.v1.updateOrganizationMember", scope: "data.organizations.write"},
	}
	additionalMethods := map[string]map[string]struct{}{
		namespace + "/entities/countries":                                           {"post": {}},
		namespace + "/entities/countries/{country_id}":                              {"put": {}},
		namespace + "/entities/industries":                                          {"post": {}},
		namespace + "/entities/industries/{industry_id}":                            {"put": {}},
		namespace + "/entities/concepts":                                            {"post": {}},
		namespace + "/entities/concepts/{concept_id}":                               {"put": {}},
		namespace + "/entities/chain-nodes":                                         {"post": {}},
		namespace + "/entities/chain-nodes/{chain_node_id}":                         {"put": {}},
		namespace + "/entities/industry-chains":                                     {"post": {}},
		namespace + "/entities/industry-chains/{industry_chain_id}":                 {"put": {}},
		namespace + "/entities/organizations":                                       {"post": {}},
		namespace + "/entities/organizations/{organization_id}":                     {"put": {}},
		namespace + "/entities/organizations/{organization_id}/members":             {"post": {}},
		namespace + "/entities/organizations/{organization_id}/members/{member_id}": {"delete": {}},
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
					if _, allowed := additionalMethods[path][method]; !allowed {
						t.Fatalf("path %q unexpectedly defines %s", path, method)
					}
				}
			}
		}
	}
}

func TestOpenAPIContractDoesNotPublishRetiredEventSemanticContracts(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	for _, path := range []string{
		namespace + "/event-semantics/eligible-events",
		namespace + "/event-semantics/context-leases",
		namespace + "/event-semantics/context-leases/{context_lease_id}/context",
		namespace + "/event-semantics/submissions",
		namespace + "/event-semantics/submissions/{submission_id}/reviews",
		namespace + "/events/{event_id}/semantics",
		namespace + "/research-analysis-context",
	} {
		if _, exists := paths[path]; exists {
			t.Errorf("retired path %q remains in the Data contract", path)
		}
	}

	schemas := object(t, object(t, document["components"], "components")["schemas"], "schemas")
	for name := range schemas {
		if (strings.HasPrefix(name, "EventSemantic") && name != "EventSemantic") || strings.HasPrefix(name, "ResearchAnalysis") ||
			strings.Contains(name, "VariableSignal") || strings.Contains(name, "DirectImpact") {
			t.Errorf("retired schema %q remains in the Data contract", name)
		}
	}
}

func TestOpenAPIContractFreezesCountryWriteOperations(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	for path, expected := range map[string]operationContract{
		namespace + "/entities/countries":              {method: "post", operationID: "createCountry", driftAnchor: "data.v1.createCountry", scope: "data.countries.write"},
		namespace + "/entities/countries/{country_id}": {method: "put", operationID: "updateCountry", driftAnchor: "data.v1.updateCountry", scope: "data.countries.write"},
	} {
		operation := object(t, object(t, paths[path], path)[expected.method], expected.method+" "+path)
		assertString(t, operation, "operationId", expected.operationID)
		assertString(t, operation, "x-client-drift-anchor", expected.driftAnchor)
		assertString(t, operation, "x-required-service-scope", expected.scope)
		assertInt(t, operation, "x-timeout-budget-ms", 5000)
	}
}

func TestOpenAPIContractFreezesIndustryAndConceptWriteOperations(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	for path, expected := range map[string]operationContract{
		namespace + "/entities/industries":                          {method: "post", operationID: "createIndustry", driftAnchor: "data.v1.createIndustry", scope: "data.industries.write"},
		namespace + "/entities/industries/{industry_id}":            {method: "put", operationID: "updateIndustry", driftAnchor: "data.v1.updateIndustry", scope: "data.industries.write"},
		namespace + "/entities/concepts":                            {method: "post", operationID: "createConcept", driftAnchor: "data.v1.createConcept", scope: "data.concepts.write"},
		namespace + "/entities/concepts/{concept_id}":               {method: "put", operationID: "updateConcept", driftAnchor: "data.v1.updateConcept", scope: "data.concepts.write"},
		namespace + "/entities/chain-nodes":                         {method: "post", operationID: "createChainNode", driftAnchor: "data.v1.createChainNode", scope: "data.chain-nodes.write"},
		namespace + "/entities/chain-nodes/{chain_node_id}":         {method: "put", operationID: "updateChainNode", driftAnchor: "data.v1.updateChainNode", scope: "data.chain-nodes.write"},
		namespace + "/entities/industry-chains":                     {method: "post", operationID: "createIndustryChain", driftAnchor: "data.v1.createIndustryChain", scope: "data.industry-chains.write"},
		namespace + "/entities/industry-chains/{industry_chain_id}": {method: "put", operationID: "updateIndustryChain", driftAnchor: "data.v1.updateIndustryChain", scope: "data.industry-chains.write"},
	} {
		operation := object(t, object(t, paths[path], path)[expected.method], expected.method+" "+path)
		assertString(t, operation, "operationId", expected.operationID)
		assertString(t, operation, "x-client-drift-anchor", expected.driftAnchor)
		assertString(t, operation, "x-required-service-scope", expected.scope)
		assertInt(t, operation, "x-timeout-budget-ms", 5000)
	}
	assertRequired(t, schema(t, document, "Industry"), "id", "name", "aliases", "classification_system", "industry_code", "parent_industry_id", "hierarchy_path_codes", "definition", "review_status", "created_at", "updated_at")
	assertRequired(t, schema(t, document, "Concept"), "id", "name", "aliases", "concept_type", "definition", "review_status", "created_at", "updated_at")
	assertRequired(t, schema(t, document, "ChainNode"), "id", "name", "aliases", "definition", "review_status", "created_at", "updated_at")
	assertRequired(t, schema(t, document, "IndustryChain"), "id", "name", "aliases", "scope", "target_output", "end_use", "geography", "primary_country_id", "as_of_date", "review_status", "review_note", "technology_route_qualifier", "observable_variables", "created_at", "updated_at")
}

func TestOpenAPIContractFreezesIndustryAndConceptKeysetPagination(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	for _, path := range []string{namespace + "/entities/industries", namespace + "/entities/concepts", namespace + "/entities/chain-nodes", namespace + "/entities/industry-chains"} {
		operation := object(t, object(t, paths[path], path)["get"], "get "+path)
		refs := make(map[string]bool)
		for _, parameter := range array(t, operation["parameters"], path+" parameters") {
			ref := stringValue(t, object(t, parameter, path+" parameter")["$ref"], path+" parameter ref")
			refs[ref] = true
		}
		for _, want := range []string{"#/components/parameters/PageSize", "#/components/parameters/EntityListCursor"} {
			if !refs[want] {
				t.Errorf("%s does not declare %s", path, want)
			}
		}
	}
	for _, schemaName := range []string{"IndustryList", "ConceptList", "ChainNodeList", "IndustryChainList"} {
		page := schema(t, document, schemaName)
		assertRequired(t, page, "items", "next_cursor")
		nextCursor := object(t, object(t, page["properties"], schemaName+" properties")["next_cursor"], schemaName+" next_cursor")
		if nullable, ok := nextCursor["nullable"].(bool); !ok || !nullable {
			t.Errorf("%s next_cursor must be nullable", schemaName)
		}
	}
}

func TestOpenAPICreateContractsDoNotAcceptSystemOwnedPrimaryKeys(t *testing.T) {
	document := loadContract(t)
	for schemaName, forbidden := range map[string][]string{
		"CountryCreateRequest":                    {"id", "country_id"},
		"IndustryCreateRequest":                   {"id", "industry_id"},
		"ConceptWriteRequest":                     {"id", "concept_id"},
		"ChainNodeWriteRequest":                   {"id", "chain_node_id"},
		"IndustryChainWriteRequest":               {"id", "industry_chain_id"},
		"CountryRegionsReplaceRequest":            {"id", "country_region_link_id"},
		"OrganizationCreateRequest":               {"id", "organization_id"},
		"OrganizationMemberWriteRequest":          {"id", "member_id"},
		"RawEvidence":                             {"id", "raw_evidence_id"},
		"AtomicEvidence":                          {"id", "evidence_id"},
		"ResearchThemeImportRequest":              {"id", "receipt_id", "theme_id", "reasoning_tree_receipt_id"},
		"ResearchThemeSnapshotItem":               {"id", "theme_id"},
		"ResearchReasoningTreeSnapshotImportItem": {"id", "reasoning_tree_id"},
		"ResearchReasoningTreeSnapshotNode":       {"id", "node_id"},
	} {
		properties := object(t, schema(t, document, schemaName)["properties"], schemaName+" properties")
		for _, property := range forbidden {
			if _, exists := properties[property]; exists {
				t.Errorf("%s exposes system-owned primary key %q", schemaName, property)
			}
		}
	}
}

func TestOpenAPIContractFreezesOrganizationNullsErrorsAndRequestIDs(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	operations := []struct {
		path     string
		method   string
		statuses []string
	}{
		{namespace + "/entities/organizations", "get", []string{"200", "401", "403", "422", "500", "503"}},
		{namespace + "/entities/organizations", "post", []string{"201", "400", "401", "403", "409", "422", "500", "503"}},
		{namespace + "/entities/organizations/{organization_id}", "get", []string{"200", "401", "403", "404", "422", "500", "503"}},
		{namespace + "/entities/organizations/{organization_id}", "put", []string{"200", "400", "401", "403", "404", "422", "500", "503"}},
		{namespace + "/entities/organizations/{organization_id}/domain-tags", "put", []string{"200", "400", "401", "403", "404", "422", "500", "503"}},
		{namespace + "/organization-catalog", "get", []string{"200", "401", "403", "500", "503"}},
		{namespace + "/entities/organizations/{organization_id}/members", "get", []string{"200", "401", "403", "404", "422", "500", "503"}},
		{namespace + "/entities/organizations/{organization_id}/members", "post", []string{"201", "400", "401", "403", "409", "422", "500", "503"}},
		{namespace + "/entities/organizations/{organization_id}/members/{member_id}", "put", []string{"200", "400", "401", "403", "404", "409", "422", "500", "503"}},
		{namespace + "/entities/organizations/{organization_id}/members/{member_id}", "delete", []string{"200", "401", "403", "404", "422", "500", "503"}},
	}
	for _, expected := range operations {
		operation := object(t, object(t, paths[expected.path], expected.path)[expected.method], expected.method+" "+expected.path)
		parameters := array(t, operation["parameters"], expected.method+" parameters")
		if len(parameters) == 0 || stringValue(t, object(t, parameters[0], "request ID parameter")["$ref"], "$ref") != "#/components/parameters/RequestID" {
			t.Errorf("%s %s does not accept X-Request-ID first: %v", expected.method, expected.path, parameters)
		}
		responses := object(t, operation["responses"], expected.method+" responses")
		for _, status := range expected.statuses {
			if _, exists := responses[status]; !exists {
				t.Errorf("%s %s missing response %s", expected.method, expected.path, status)
			}
		}
	}

	assertRequired(t, schema(t, document, "Organization"),
		"id", "code", "name", "name_en", "region_id", "category", "function",
		"legal_entity_code", "dominant_party_id", "binding_power_level", "influence_rating",
		"strategic_positioning", "core_impact_scope", "founding_document", "established_date",
		"headquarters_city", "headquarters_country_id", "headquarters_subdivision_id", "description",
		"domain_tags", "created_at", "updated_at",
	)
	assertRequired(t, schema(t, document, "OrganizationMember"),
		"id", "organization_id", "country_id", "membership_type", "effective_date", "expiry_date", "created_at", "updated_at",
	)
	assertRequired(t, schema(t, document, "OrganizationMemberDeleteEnvelope"), "request_id", "result")
	assertRequired(t, schema(t, document, "OrganizationCategory"), "id", "code", "name_zh")
	assertRequired(t, schema(t, document, "OrganizationFunction"), "id", "code", "name_zh")
	assertRequired(t, schema(t, document, "OrganizationDomainTag"), "id", "code", "function_code", "name_zh")
}

func TestOpenAPIContractFreezesCurrentEventReadContract(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	operation := object(t, object(t, paths[namespace+"/events"], "Event path")["get"], "Event operation")
	assertString(t, operation, "x-retry-policy", "safe-get")
	assertString(t, operation, "x-required-service-scope", "data.admin.read")

	parameters := array(t, operation["parameters"], "Event parameters")
	if len(parameters) != 10 {
		t.Fatalf("Event parameter count = %d, want 10", len(parameters))
	}
	wantNames := []string{"title", "modality", "status", "occurred_from", "occurred_to", "announced_from", "announced_to"}
	for index, want := range wantNames {
		assertString(t, object(t, parameters[index+3], want+" parameter"), "name", want)
	}

	event := schema(t, document, "AdminEvent")
	assertRequired(t, event, "id", "title", "summary", "semantic", "modality", "occurred_at", "announced_at", "status")
	semantic := schema(t, document, "EventSemantic")
	assertRequired(t, semantic, "who", "what", "when", "where", "why", "how")
	if semantic["additionalProperties"] != false {
		t.Fatal("EventSemantic must reject additional properties")
	}
	assertStringSet(t, schema(t, document, "EventModality")["enum"], "FACT", "PLAN", "SPEC")
	assertStringSet(t, schema(t, document, "EventLifecycleStatus")["enum"], "ACTIVE", "DEPRECATED", "ARCHIVED")

	for _, retired := range []string{namespace + "/event-tags", namespace + "/raw-documents", namespace + "/reviewed-event-imports"} {
		if _, exists := paths[retired]; exists {
			t.Errorf("retired path %q remains in the Data contract", retired)
		}
	}
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
		"tree_key", "display_name",
		"reasoning_tree_id", "theme_id", "title",
		"display_order", "one_line_conclusion", "fact_summary", "transmission_summary",
		"impact_direction", "impact_strength", "impact_summary", "conclusion_boundary_summary",
		"support_summary", "counter_summary", "invalidation_conditions", "checkpoints",
		"published_at", "event_count", "events", "nodes",
	)
	node := schema(t, document, "ResearchReasoningTreeNode")
	assertRequired(t, node,
		"node_key", "display_name",
		"id", "position", "state_summary", "impact_direction",
		"impact_strength", "impact_summary", "reasoning_basis_summary", "evidence_gap_summary",
		"incoming_transmission_title", "incoming_transmission_mechanism", "incoming_condition_summary",
		"signals", "primary_signal", "signal_display_summary",
	)
	detail := schema(t, document, "ResearchReasoningTreeDetail")
	assertRequired(t, detail, "theme_id", "theme_key", "publication_mode", "publication_contract_version", "impact_node_ids", "reasoning_tree")
}

func TestOpenAPIContractFreezesSnapshotOnlyResearchPublication(t *testing.T) {
	document := loadContract(t)
	snapshot := schema(t, document, "ResearchThemeImportRequest")
	assertRequired(t, snapshot, "publication_mode", "analysis_batch_id", "analysis_as_of", "discovery_window_start", "discovery_window_end", "theme", "reasoning_trees")
	impact := schema(t, document, "ResearchThemeSnapshotImpact")
	assertRequired(t, impact, "node_key", "display_name", "relation_role", "impact_direction", "impact_summary", "display_order")
	properties := object(t, impact["properties"], "snapshot impact properties")
	if _, exists := properties["chain_node_id"]; exists {
		t.Fatal("analyst_snapshot impact must not expose a formal Entity ID")
	}
	node := schema(t, document, "ResearchReasoningTreeSnapshotNode")
	assertRequired(t, node, "node_key", "display_name", "position", "state_summary", "impact_direction", "impact_strength", "impact_summary", "reasoning_basis_summary", "evidence_gap_summary", "incoming_transmission", "signals")
	signal := schema(t, document, "ResearchReasoningTreeSnapshotSignal")
	assertRequired(t, signal, "signal_key", "display_summary", "role", "display_order", "variable_name", "direction")
	if _, exists := object(t, signal["properties"], "snapshot signal properties")["variable_signal_id"]; exists {
		t.Fatal("analyst_snapshot signal must not expose a formal VariableSignal ID")
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

func TestOpenAPIContractFreezesDTOFormatsEnumsAndSensitiveMetadataBoundary(t *testing.T) {
	document := loadContract(t)
	for _, name := range []string{
		"ResearchThemeCollection", "ResearchThemeDetail", "ResearchReasoningTreeList", "ResearchReasoningTreeDetail",
		"AdminEventPage", "ErrorEnvelope",
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

	assertStringSet(t, schema(t, document, "EventModality")["enum"], "FACT", "PLAN", "SPEC")
	assertStringSet(t, schema(t, document, "EventLifecycleStatus")["enum"], "ACTIVE", "DEPRECATED", "ARCHIVED")
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
