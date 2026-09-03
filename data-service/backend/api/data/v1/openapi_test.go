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
		"/healthz":                                                                  {method: "get", operationID: "getDataServiceHealth"},
		"/readyz":                                                                   {method: "get", operationID: "getDataServiceReadiness"},
		namespace + "/evidence-categories":                                          {method: "get", operationID: "listEvidenceCategories", driftAnchor: "data.v1.listEvidenceCategories", scope: "data.evidence-categories.read"},
		namespace + "/evidences":                                                    {method: "get", operationID: "listAdminEvidence", driftAnchor: "data.v1.listAdminEvidence", scope: "data.admin.read"},
		namespace + "/events":                                                       {method: "get", operationID: "listAdminEvents", driftAnchor: "data.v1.listAdminEvents", scope: "data.admin.read"},
		namespace + "/runtime-health":                                               {method: "get", operationID: "getDataRuntimeHealth", driftAnchor: "data.v1.getRuntimeHealth", scope: "data.admin.read"},
		namespace + "/research-graph:search":                                        {method: "post", operationID: "searchResearchGraph", driftAnchor: "data.v1.searchResearchGraph", scope: "data.research.read"},
		namespace + "/raw-evidence-publications":                                    {method: "post", operationID: "publishRawEvidence", driftAnchor: "data.v1.publishRawEvidence", scope: "data.raw-evidences.import"},
		namespace + "/raw-evidences/{id}":                                           {method: "get", operationID: "getRawEvidence", driftAnchor: "data.v1.getRawEvidence", scope: "data.raw-evidences.read"},
		namespace + "/evidence-publications":                                        {method: "post", operationID: "publishEvidence", driftAnchor: "data.v1.publishEvidence", scope: "data.evidences.import"},
		namespace + "/entities/countries":                                           {method: "get", operationID: "listCountries", driftAnchor: "data.v1.listCountries", scope: "data.countries.read"},
		namespace + "/entities/countries/{country_id}":                              {method: "get", operationID: "getCountry", driftAnchor: "data.v1.getCountry", scope: "data.countries.read"},
		namespace + "/entities/countries/{country_id}/regions":                      {method: "put", operationID: "replaceCountryRegions", driftAnchor: "data.v1.replaceCountryRegions", scope: "data.countries.write"},
		namespace + "/entities/industries":                                          {method: "get", operationID: "listIndustries", driftAnchor: "data.v1.listIndustries", scope: "data.industries.read"},
		namespace + "/entities/companies":                                           {method: "get", operationID: "listCompanies", driftAnchor: "data.v1.listCompanies", scope: "data.companies.read"},
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
		namespace + "/sources":                                                      {method: "get", operationID: "listSources", driftAnchor: "data.v1.listSources", scope: "data.sources.read"},
		namespace + "/sources/{source_id}":                                          {method: "put", operationID: "updateSource", driftAnchor: "data.v1.updateSource", scope: "data.sources.write"},
		namespace + "/source-snapshot":                                              {method: "get", operationID: "getSourceSnapshot", driftAnchor: "data.v1.getSourceSnapshot", scope: "data.sources.read"},
		namespace + "/report-publications":                                          {method: "post", operationID: "publishReport", driftAnchor: "data.v1.publishReport", scope: "data.reports.publish"},
		namespace + "/reports":                                                      {method: "get", operationID: "listReports", driftAnchor: "data.v1.listReports", scope: "data.reports.read"},
		namespace + "/reports/{report_id}/home":                                     {method: "get", operationID: "getReportHome", driftAnchor: "data.v1.getReportHome", scope: "data.reports.read"},
		namespace + "/reports/{report_id}/layers/{layer_key}":                       {method: "get", operationID: "getReportLayer", driftAnchor: "data.v1.getReportLayer", scope: "data.reports.read"},
		namespace + "/reports/{report_id}/industry-chains":                          {method: "get", operationID: "listReportIndustryChains", driftAnchor: "data.v1.listReportIndustryChains", scope: "data.reports.read"},
		namespace + "/reports/{report_id}/industry-chains/{chain_key}":              {method: "get", operationID: "getReportIndustryChain", driftAnchor: "data.v1.getReportIndustryChain", scope: "data.reports.read"},
		namespace + "/reports/{report_id}/evidences":                                {method: "get", operationID: "listReportEvidence", driftAnchor: "data.v1.listReportEvidence", scope: "data.reports.read"},
	}
	additionalMethods := map[string]map[string]struct{}{
		namespace + "/events":                                                       {"post": {}},
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
		namespace + "/sources":                                                      {"post": {}},
		namespace + "/sources/{source_id}":                                          {"delete": {}},
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

func TestOpenAPIContractFreezesImmutableReportPublicationAndReadModels(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	schemas := object(t, object(t, document["components"], "components")["schemas"], "schemas")
	for _, retired := range []string{
		"ReportPublicationRequestV2", "ReportContentV2", "ReportCard", "ReportCompanyBoundary",
		"ReportLayerRead", "ReportIndustryChainRead",
	} {
		if _, exists := schemas[retired]; exists {
			t.Errorf("OpenAPI retains retired report-publication.v1 schema %q", retired)
		}
	}
	publish := object(t, object(t, paths[namespace+"/report-publications"], "Report publication path")["post"], "Report publication operation")
	assertInt(t, publish, "x-timeout-budget-ms", 20000)
	requestBody := object(t, publish["requestBody"], "Report publication request body")
	mediaTypes := object(t, requestBody["content"], "Report publication media types")
	mediaType := object(t, mediaTypes["application/json"], "Report publication JSON media type")
	bodySchema := object(t, mediaType["schema"], "Report publication body schema")
	assertInt(t, bodySchema, "maxLength", 1048576)
	assertInt(t, bodySchema, "x-max-body-bytes", 1048576)
	bodySchemaParts := array(t, bodySchema["allOf"], "Report publication body allOf")
	if len(bodySchemaParts) != 1 {
		t.Fatalf("Report publication body allOf count = %d, want 1", len(bodySchemaParts))
	}
	assertString(t, object(t, bodySchemaParts[0], "Report publication request reference"), "$ref", "#/components/schemas/ReportPublicationRequest")
	request := schema(t, document, "ReportPublicationRequest")
	assertRequired(t, request, "publisher_report_id", "report")
	requestProperties := object(t, request["properties"], "ReportPublicationRequest properties")
	for _, retired := range []string{"publication_key", "receipt_id", "contract_version", "content", "publisher"} {
		if _, exists := requestProperties[retired]; exists {
			t.Errorf("Report publication request exposes retired field %q", retired)
		}
	}
	assertRequired(t, schema(t, document, "Report"), "report_type", "generated_at", "timezone", "industry_chains")
	assertRequired(t, schema(t, document, "ReportIndustryChain"),
		"local_key", "name", "conclusion", "result", "time_window", "confidence",
		"path_summary", "accepted_hypothesis_summary", "nodes", "edges", "uncertainty", "evidence_refs",
	)
	assertRequired(t, schema(t, document, "ReportIndustryChainGraph"), "nodes", "edges")
	assertRequired(t, schema(t, document, "ReportReasoningStep"), "local_key", "input", "mechanism", "output", "confidence", "evidence_refs")
	assertRequired(t, schema(t, document, "ReportTransmission"),
		"local_key", "source_conclusion", "targets", "transmission_logic", "transmission_kind",
		"confidence", "status",
	)
	assertRequired(t, schema(t, document, "ReportPublicationResult"), "report_id", "published_at", "replayed")
	assertRequired(t, schema(t, document, "ReportCollection"), "items", "next_cursor")
	resultProperties := object(t, schema(t, document, "ReportPublicationResult")["properties"], "ReportPublicationResult properties")
	for _, retired := range []string{"payload_hash", "content_hash", "receipt_id"} {
		if _, exists := resultProperties[retired]; exists {
			t.Errorf("Report publication response exposes retired field %q", retired)
		}
	}
	assertRequired(t, schema(t, document, "ReportIndustryChainCollection"), "items", "next_cursor")
	assertRequired(t, schema(t, document, "ReportLayerSnapshot"), "key", "title", "summary")
	assertRequired(t, schema(t, document, "ReportEvidenceItem"), "published_at", "summary", "keywords")
	assertRequired(t, schema(t, document, "ReportEvidenceCollection"), "report_id", "scope_token", "items")

	for _, path := range []string{
		namespace + "/reports", namespace + "/reports/{report_id}/home",
		namespace + "/reports/{report_id}/layers/{layer_key}",
		namespace + "/reports/{report_id}/industry-chains",
		namespace + "/reports/{report_id}/industry-chains/{chain_key}",
		namespace + "/reports/{report_id}/evidences",
	} {
		operation := object(t, object(t, paths[path], path)["get"], "GET "+path)
		assertInt(t, operation, "x-timeout-budget-ms", 5000)
	}
	evidenceOperation := object(t, object(t, paths[namespace+"/reports/{report_id}/evidences"], "Report Evidence path")["get"], "Report Evidence operation")
	assertStringSet(t, evidenceOperation["x-error-codes"],
		"INVALID_REQUEST", "REPORT_NOT_FOUND", "REPORT_EVIDENCE_SCOPE_NOT_FOUND", "REPORT_REPOSITORY_FAILURE", "DATA_SERVICE_NOT_READY",
	)
}

func TestOpenAPIContractFreezesCompanyProjectionSnapshot(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	operation := object(t, object(t, paths[namespace+"/entities/companies"], "Company projection path")["get"], "Company projection operation")
	assertString(t, operation, "x-retry-policy", "safe-get-restart-from-first-page-on-snapshot-change")
	assertInt(t, operation, "x-timeout-budget-ms", 5000)
	assertStringSet(t, operation["x-error-codes"],
		"INVALID_REQUEST", "COMPANY_PROJECTION_SNAPSHOT_CHANGED", "COMPANY_PROJECTION_TIMEOUT",
		"COMPANY_PROJECTION_PERSISTENCE_FAILED", "COMPANY_PROJECTION_FAILED",
	)
	parameters := array(t, operation["parameters"], "Company projection parameters")
	if len(parameters) != 3 || stringValue(t, object(t, parameters[2], "Company cursor parameter")["$ref"], "Company cursor ref") != "#/components/parameters/CompanyProjectionCursor" {
		t.Fatalf("Company projection cursor parameter = %#v", parameters)
	}
	responses := object(t, operation["responses"], "Company projection responses")
	for _, status := range []string{"200", "400", "401", "403", "409", "500", "503"} {
		if _, exists := responses[status]; !exists {
			t.Errorf("Company projection response %s is missing", status)
		}
	}
	page := schema(t, document, "CompanyProjectionPage")
	assertRequired(t, page, "schema_version", "snapshot_id", "items", "next_cursor")
	properties := object(t, page["properties"], "CompanyProjectionPage properties")
	assertStringSet(t, object(t, properties["schema_version"], "schema_version")["enum"], "company-projection-snapshot.v1")
	assertString(t, object(t, properties["snapshot_id"], "snapshot_id"), "$ref", "#/components/schemas/PayloadHash")
	company := schema(t, document, "CompanyProjectionItem")
	assertRequired(t, company,
		"id", "code", "name", "name_en", "legal_name", "aliases", "registration_country_id",
		"operating_area", "headquarters_city", "founding_date", "ipo_date", "legal_form", "ownership_type",
		"strategic_positioning", "description", "status", "created_at", "updated_at", "industry_links",
	)
	assertRequired(t, schema(t, document, "CompanyIndustryLink"), "id", "company_id", "industry_id", "created_at")
}

func TestOpenAPIContractFreezesSourceManagementAndSnapshot(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	type sourceOperation struct {
		operationContract
		errorCodes []string
	}
	for _, expected := range []sourceOperation{
		{operationContract: operationContract{method: "get", operationID: "listSources", driftAnchor: "data.v1.listSources", scope: "data.sources.read"}, errorCodes: []string{"SOURCE_FAILED", "SOURCE_TIMEOUT"}},
		{operationContract: operationContract{method: "post", operationID: "createSource", driftAnchor: "data.v1.createSource", scope: "data.sources.write"}, errorCodes: []string{"SOURCE_INVALID", "SOURCE_CONFLICT", "SOURCE_CAPACITY_EXCEEDED", "SOURCE_TIMEOUT", "SOURCE_FAILED"}},
		{operationContract: operationContract{method: "put", operationID: "updateSource", driftAnchor: "data.v1.updateSource", scope: "data.sources.write"}, errorCodes: []string{"SOURCE_INVALID", "SOURCE_NOT_FOUND", "SOURCE_CONFLICT", "SOURCE_CAPACITY_EXCEEDED", "SOURCE_TIMEOUT", "SOURCE_FAILED"}},
		{operationContract: operationContract{method: "delete", operationID: "deleteSource", driftAnchor: "data.v1.deleteSource", scope: "data.sources.write"}, errorCodes: []string{"SOURCE_INVALID", "SOURCE_NOT_FOUND", "SOURCE_FIXED_DELETE_FORBIDDEN", "SOURCE_TIMEOUT", "SOURCE_FAILED"}},
		{operationContract: operationContract{method: "get", operationID: "getSourceSnapshot", driftAnchor: "data.v1.getSourceSnapshot", scope: "data.sources.read"}, errorCodes: []string{"SOURCE_SNAPSHOT_FAILED", "SOURCE_TIMEOUT", "SOURCE_FAILED"}},
	} {
		path := namespace + "/sources"
		if expected.method == "put" || expected.method == "delete" {
			path += "/{source_id}"
		}
		if expected.operationID == "getSourceSnapshot" {
			path = namespace + "/source-snapshot"
		}
		operation := object(t, object(t, paths[path], path)[expected.method], expected.method+" "+path)
		assertString(t, operation, "operationId", expected.operationID)
		assertString(t, operation, "x-client-drift-anchor", expected.driftAnchor)
		assertString(t, operation, "x-required-service-scope", expected.scope)
		assertInt(t, operation, "x-timeout-budget-ms", 3000)
		assertStringSet(t, operation["x-error-codes"], expected.errorCodes...)
	}
	createResponses := object(t, object(t, object(t, paths[namespace+"/sources"], "Source collection path")["post"], "Source create")["responses"], "Source create responses")
	updateResponses := object(t, object(t, object(t, paths[namespace+"/sources/{source_id}"], "Source item path")["put"], "Source update")["responses"], "Source update responses")
	deleteResponses := object(t, object(t, object(t, paths[namespace+"/sources/{source_id}"], "Source item path")["delete"], "Source delete")["responses"], "Source delete responses")
	if _, exists := createResponses["413"]; !exists {
		t.Fatal("Source create must document 413")
	}
	if _, exists := updateResponses["413"]; !exists {
		t.Fatal("Source update must document 413")
	}
	if _, exists := deleteResponses["422"]; !exists {
		t.Fatal("Source delete must document 422")
	}

	snapshot := object(t, object(t, paths[namespace+"/source-snapshot"], "snapshot path")["get"], "snapshot operation")
	assertInt(t, snapshot, "x-max-response-bytes", 500000)
	assertRequired(t, schema(t, document, "Source"),
		"id", "code", "name", "ownership_type", "channel_type", "adapter_key", "enabled", "endpoint", "app_key", "config",
		"priority", "timeout_seconds", "max_results", "default_source_level", "created_at", "updated_at",
	)
	assertStringSet(t, schema(t, document, "SourceLevel")["enum"], "L1_OFFICIAL", "L2_WIRE", "L3_MEDIA", "L4_SOCIAL")
	for _, schemaName := range []string{"SourceCreateRequest", "SourceUpdateRequest"} {
		properties := object(t, schema(t, document, schemaName)["properties"], schemaName+" properties")
		for _, immutable := range []string{"id", "ownership_type", "channel_type"} {
			if _, exists := properties[immutable]; exists {
				t.Errorf("%s exposes system-owned or immutable property %q", schemaName, immutable)
			}
		}
	}
	updateProperties := object(t, schema(t, document, "SourceUpdateRequest")["properties"], "SourceUpdateRequest properties")
	if _, exists := updateProperties["code"]; exists {
		t.Fatal("SourceUpdateRequest must not expose immutable code")
	}
	if _, exists := updateProperties["adapter_key"]; !exists {
		t.Fatal("SourceUpdateRequest must expose mutable adapter_key")
	}

	collectionProperties := object(t, schema(t, document, "SourceCollection")["properties"], "SourceCollection properties")
	sources := object(t, collectionProperties["sources"], "SourceCollection sources")
	assertInt(t, sources, "maxItems", 200)
	assertInt(t, schema(t, document, "SourceSnapshotEnvelope"), "x-max-utf8-bytes", 500000)
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
		"CountryCreateRequest":           {"id", "country_id"},
		"IndustryCreateRequest":          {"id", "industry_id"},
		"ConceptWriteRequest":            {"id", "concept_id"},
		"ChainNodeWriteRequest":          {"id", "chain_node_id"},
		"IndustryChainWriteRequest":      {"id", "industry_chain_id"},
		"CountryRegionsReplaceRequest":   {"id", "country_region_link_id"},
		"OrganizationCreateRequest":      {"id", "organization_id"},
		"OrganizationMemberWriteRequest": {"id", "member_id"},
		"RawEvidence":                    {"id", "raw_evidence_id"},
		"AtomicEvidence":                 {"id", "evidence_id"},
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
	assertRequired(t, event, "id", "title", "summary", "semantic", "status")
	semantic := schema(t, document, "EventSemantic")
	assertRequired(t, semantic, "actors", "action", "objects", "stage", "modality", "time", "jurisdictions", "reason", "method", "metrics")
	if semantic["additionalProperties"] != false {
		t.Fatal("EventSemantic must reject additional properties")
	}
	eventTimeVariants := array(t, schema(t, document, "EventTime")["oneOf"], "EventTime oneOf")
	if len(eventTimeVariants) != 2 {
		t.Fatalf("EventTime oneOf = %d, want business and observed-only time", len(eventTimeVariants))
	}
	assertRequired(t, schema(t, document, "EventBusinessTime"), "occurred_at", "announced_at", "effective_at", "observed_at", "precision")
	assertRequired(t, schema(t, document, "EventObservedTime"), "occurred_at", "announced_at", "effective_at", "observed_at", "precision")
	assertRequired(t, schema(t, document, "EventLegacyTime"), "occurred_at", "announced_at", "effective_at", "precision")
	publicationTime := schema(t, document, "EventPublicationTime")
	publicationTimeVariants := array(t, publicationTime["oneOf"], "EventPublicationTime oneOf")
	if len(publicationTimeVariants) != 2 {
		t.Fatalf("EventPublicationTime oneOf = %d, want canonical and legacy time", len(publicationTimeVariants))
	}
	assertRequired(t, schema(t, document, "EventMetric"), "name", "value", "unit", "change", "period")
	assertStringSet(t, schema(t, document, "EventModality")["enum"], "FACT", "PLAN", "SPEC")
	assertStringSet(t, schema(t, document, "EventLifecycleStatus")["enum"], "ACTIVE", "DEPRECATED", "ARCHIVED")

	for _, retired := range []string{namespace + "/event-tags", namespace + "/raw-documents", namespace + "/reviewed-event-imports"} {
		if _, exists := paths[retired]; exists {
			t.Errorf("retired path %q remains in the Data contract", retired)
		}
	}
}

func TestOpenAPIContractFreezesEvidenceSourceIDProjection(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	operation := object(t, object(t, paths[namespace+"/evidences"], "Evidence path")["get"], "Evidence operation")
	parameters := array(t, operation["parameters"], "Evidence parameters")
	foundSourceID := false
	for _, value := range parameters {
		parameter := object(t, value, "Evidence parameter")
		if parameter["name"] == "source_id" {
			foundSourceID = true
			assertInt(t, parameter, "x-trimmed-max-length", 32)
			parameterSchema := object(t, parameter["schema"], "source_id schema")
			if _, exists := parameterSchema["maxLength"]; exists {
				t.Fatal("source_id query maxLength must apply after trimming, not to the raw query value")
			}
		}
	}
	if !foundSourceID {
		t.Fatal("Evidence parameters do not contain source_id")
	}
	assertRequired(t, schema(t, document, "AdminEvidence"),
		"id", "raw_evidence_id", "title", "summary", "semantic", "categories", "source_id", "source_name",
		"source_level", "source_url", "is_original", "quoted_source_name", "keywords", "is_split", "published_at", "collected_at",
	)
}

func TestOpenAPIContractDoesNotPublishRetiredResearchThemeContracts(t *testing.T) {
	document := loadContract(t)
	paths := object(t, document["paths"], "paths")
	for _, retiredPath := range []string{
		namespace + "/research-theme-imports",
		namespace + "/research/themes",
		namespace + "/research/themes/{theme_id}",
		namespace + "/research/themes/{theme_id}/reasoning-trees",
		namespace + "/research/themes/{theme_id}/reasoning-trees/{reasoning_tree_id}",
	} {
		if _, exists := paths[retiredPath]; exists {
			t.Fatalf("retired Research Theme path remains in OpenAPI: %s", retiredPath)
		}
	}
	schemas := object(t, object(t, document["components"], "components")["schemas"], "schemas")
	for name := range schemas {
		if strings.HasPrefix(name, "ResearchTheme") || strings.HasPrefix(name, "ResearchReasoningTree") ||
			name == "ResearchSnapshotIncomingTransmission" || name == "ResearchDirection" ||
			name == "ResearchLocalKey" || name == "ResearchImpactStrength" || name == "ResearchEvent" {
			t.Fatalf("retired Research Theme schema remains in OpenAPI: %s", name)
		}
	}
}

func TestOpenAPIContractFreezesControlledResearchGraphSearchV2(t *testing.T) {
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
	resultProperties := object(t, result["properties"], "ResearchGraphSearchResult properties")
	assertStringSet(t, object(t, resultProperties["contract_version"], "contract_version")["enum"], "research-graph-search.v2")
	memberships := object(t, object(t, resultProperties["industry_chain_memberships"], "industry_chain_memberships")["items"], "industry_chain_memberships items")
	assertRequired(t, memberships, "industry_chain_id", "chain_node_id", "position", "contextual_stage")
	edges := object(t, object(t, resultProperties["industry_chain_graph_edges"], "industry_chain_graph_edges")["items"], "industry_chain_graph_edges items")
	assertRequired(t, edges, "industry_chain_graph_edge_id", "industry_chain_id", "from_chain_node_id", "to_chain_node_id", "relation_type")
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
