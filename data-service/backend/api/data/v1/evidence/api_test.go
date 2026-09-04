package evidence

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	"gopkg.in/yaml.v3"
)

func TestEvidenceCategoryCatalogProviderFixtureMatchesExactDTO(t *testing.T) {
	payload, err := os.ReadFile("testdata/evidence-category-catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		RequestID string                  `json:"request_id"`
		Result    EvidenceCategoryCatalog `json:"result"`
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		t.Fatalf("fixture contains trailing JSON: %v", err)
	}
	if envelope.RequestID == "" || len(envelope.Result.Categories) != 11 {
		t.Fatalf("fixture envelope = request_id %q, category count %d", envelope.RequestID, len(envelope.Result.Categories))
	}
	wantCodes := []string{
		"COMMENTARY_EDITORIAL_OPINION", "EVENT_BRIEF", "FINANCIAL_REPORT_DATA_SUMMARY",
		"FORECAST_PLAN_OUTLOOK", "INDUSTRY_THEME_ANALYSIS", "INTERVIEW_OR_STATEMENT",
		"IN_DEPTH_REPORT", "MARKET_MOVEMENT_ANALYSIS", "MARKET_MOVEMENT_BRIEF",
		"POLICY_DOCUMENT_SUMMARY", "SOCIAL_MEDIA_BRIEF",
	}
	seenIDs := make(map[string]struct{}, len(wantCodes))
	for index, category := range envelope.Result.Categories {
		if category.Code != wantCodes[index] || category.ID == "" || category.Name == "" || category.Description == "" {
			t.Fatalf("fixture category[%d] = %#v, want code %s with complete fields", index, category, wantCodes[index])
		}
		if _, duplicate := seenIDs[category.ID]; duplicate {
			t.Fatalf("fixture contains duplicate category ID %q", category.ID)
		}
		seenIDs[category.ID] = struct{}{}
	}
}

func TestEvidencePublicationProviderFixturesAreContractNeutralAndTwoPhase(t *testing.T) {
	rawPayload, err := os.ReadFile("testdata/raw-evidence-publication.json")
	if err != nil {
		t.Fatal(err)
	}
	evidencePayload, err := os.ReadFile("testdata/evidence-publication.json")
	if err != nil {
		t.Fatal(err)
	}
	for name, payload := range map[string][]byte{"raw": rawPayload, "evidence": evidencePayload} {
		lower := strings.ToLower(string(payload))
		for _, forbidden := range []string{"agentrun", "agentos", "group_id", "idempotency-key"} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("%s fixture couples the Data contract to %q", name, forbidden)
			}
		}
	}
	rawRequest, err := decodeRawEvidence(rawPayload)
	if err != nil {
		t.Fatalf("decode Raw Evidence fixture: %v", err)
	}
	evidenceRequest, err := decodeEvidence(evidencePayload)
	if err != nil {
		t.Fatalf("decode Evidence fixture: %v", err)
	}
	if rawRequest.RawEvidence.PublicationKey == "" || !strings.HasPrefix(evidenceRequest.RawEvidenceID, "RAW") {
		t.Fatalf("fixture identities = publication key %q, Raw Evidence %q", rawRequest.RawEvidence.PublicationKey, evidenceRequest.RawEvidenceID)
	}
	if rawRequest.RawEvidence.RawText != "/raw-evidence/documents/2026/08/11/1111111111111111111111111111111111111111111111111111111111111111.md" {
		t.Fatalf("fixture raw_text = %q, want environment-neutral object path", rawRequest.RawEvidence.RawText)
	}
	if len(rawRequest.RawEvidence.CategoryIDs) != 2 || rawRequest.RawEvidence.CategoryIDs[0] != "EVCed6e9380-8b20-53d5-b748-fa45c774fa67" || rawRequest.RawEvidence.CategoryIDs[1] != "EVC5b12ffce-178d-56ed-a54f-c01696c486f4" {
		t.Fatalf("fixture Raw Evidence categories = %#v", rawRequest.RawEvidence.CategoryIDs)
	}
	if len(evidenceRequest.Evidences) != 2 || evidenceRequest.Evidences[0].Summary == "" ||
		evidenceRequest.Evidences[0].Semantic.Action == "" || evidenceRequest.Evidences[1].Semantic.Action == "" {
		t.Fatalf("fixture must contain complete summary and semantic Evidence items: %#v", evidenceRequest.Evidences)
	}
}

func TestEvidencePublicationProviderFixtureMapsEveryCurrentRequestItem(t *testing.T) {
	requestPayload, err := os.ReadFile("testdata/evidence-publication.json")
	if err != nil {
		t.Fatal(err)
	}
	responsePayload, err := os.ReadFile("testdata/evidence-publication-response.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	request, err := decodeEvidence(requestPayload)
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		RequestID string                    `json:"request_id"`
		Result    EvidencePublicationResult `json:"result"`
	}
	decoder := json.NewDecoder(bytes.NewReader(responsePayload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&response); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		t.Fatalf("fixture contains trailing JSON: %v", err)
	}
	if response.RequestID == "" || response.Result.RawEvidenceID != request.RawEvidenceID ||
		len(request.Evidences) != 2 || len(response.Result.IDs) != 2 || len(response.Result.Items) != 2 {
		t.Fatalf("request/response fixture mismatch: request=%#v response=%#v", request, response)
	}
	if response.Result.IDs[0] >= response.Result.IDs[1] {
		t.Fatalf("compatibility ids are not formally sorted: %#v", response.Result.IDs)
	}
	if response.Result.Items[0].InputIndex != 0 || response.Result.Items[1].InputIndex != 1 ||
		response.Result.Items[0].ID != response.Result.IDs[1] || response.Result.Items[1].ID != response.Result.IDs[0] {
		t.Fatalf("fixture does not prove request-indexed association independently of ids order: %#v", response.Result)
	}
}

func TestEvidencePublicationOpenAPIUsesInternalThreeSecondBudget(t *testing.T) {
	var document map[string]any
	if err := yaml.Unmarshal(v1.Document(), &document); err != nil {
		t.Fatal(err)
	}
	paths := document["paths"].(map[string]any)
	for _, path := range []string{
		v1.APIPrefix + "/raw-evidence-publications",
		v1.APIPrefix + "/evidence-publications",
	} {
		operation := paths[path].(map[string]any)["post"].(map[string]any)
		if got := operation["x-timeout-budget-ms"]; got != 3000 {
			t.Errorf("%s x-timeout-budget-ms = %#v, want 3000", path, got)
		}
	}
}

func TestEvidencePublicationOpenAPISuccessResultsContainOnlyFormalIdentities(t *testing.T) {
	var document map[string]any
	if err := yaml.Unmarshal(v1.Document(), &document); err != nil {
		t.Fatal(err)
	}
	components := document["components"].(map[string]any)["schemas"].(map[string]any)
	for name, expected := range map[string][]string{
		"RawEvidencePublicationResult": {"id"},
		"EvidencePublicationResult":    {"raw_evidence_id", "ids", "items"},
	} {
		schema := components[name].(map[string]any)
		properties := schema["properties"].(map[string]any)
		if len(properties) != len(expected) {
			t.Fatalf("%s properties = %#v, want only %#v", name, properties, expected)
		}
		for _, field := range expected {
			if _, ok := properties[field]; !ok {
				t.Fatalf("%s is missing %q", name, field)
			}
		}
		required := schema["required"].([]any)
		if len(required) != len(expected) {
			t.Fatalf("%s required fields = %#v, want %#v", name, required, expected)
		}
		requiredFields := make(map[string]struct{}, len(required))
		for _, field := range required {
			requiredFields[field.(string)] = struct{}{}
		}
		for _, field := range expected {
			if _, ok := requiredFields[field]; !ok {
				t.Fatalf("%s required fields = %#v, want %#v", name, required, expected)
			}
		}
		if schema["additionalProperties"] != false {
			t.Fatalf("%s must reject additional response properties", name)
		}
	}
	item := components["EvidencePublicationResultItem"].(map[string]any)
	itemProperties := item["properties"].(map[string]any)
	if len(itemProperties) != 2 || item["additionalProperties"] != false {
		t.Fatalf("EvidencePublicationResultItem contract = %#v", item)
	}
	required := item["required"].([]any)
	requiredFields := make(map[string]struct{}, len(required))
	for _, field := range required {
		requiredFields[field.(string)] = struct{}{}
	}
	if len(requiredFields) != 2 {
		t.Fatalf("EvidencePublicationResultItem required fields = %#v", required)
	}
	for _, field := range []string{"input_index", "id"} {
		if _, ok := requiredFields[field]; !ok {
			t.Fatalf("EvidencePublicationResultItem required fields = %#v", required)
		}
	}
	inputIndex := itemProperties["input_index"].(map[string]any)
	if inputIndex["type"] != "integer" || inputIndex["minimum"] != 0 {
		t.Fatalf("EvidencePublicationResultItem input_index = %#v", inputIndex)
	}
	id := itemProperties["id"].(map[string]any)
	if id["type"] != "string" || id["pattern"] == nil {
		t.Fatalf("EvidencePublicationResultItem id = %#v", id)
	}
}

func TestAtomicEvidenceOpenAPIPublishesBusinessPropositionSemanticAndEvidenceKeywords(t *testing.T) {
	var document map[string]any
	if err := yaml.Unmarshal(v1.Document(), &document); err != nil {
		t.Fatal(err)
	}
	components := document["components"].(map[string]any)["schemas"].(map[string]any)
	atomic := components["AtomicEvidence"].(map[string]any)
	atomicProperties := atomic["properties"].(map[string]any)
	if len(atomicProperties) != 3 || atomicProperties["summary"] == nil || atomicProperties["keywords"] == nil || atomicProperties["semantic"] == nil || atomic["additionalProperties"] != false {
		t.Fatalf("AtomicEvidence contract = %#v", atomic)
	}
	keywords := atomicProperties["keywords"].(map[string]any)
	if keywords["type"] != "array" || keywords["minItems"] != 1 || keywords["maxItems"] != 5 || keywords["uniqueItems"] != true {
		t.Fatalf("AtomicEvidence keywords = %#v", keywords)
	}
	semantic := components["EvidenceSemantic"].(map[string]any)
	semanticProperties := semantic["properties"].(map[string]any)
	for _, field := range []string{"actors", "action", "objects", "stage", "modality", "time", "jurisdictions", "reason", "method", "metrics", "attribution"} {
		if _, exists := semanticProperties[field]; !exists {
			t.Fatalf("EvidenceSemantic is missing %q", field)
		}
	}
	if len(semanticProperties) != 11 || semantic["additionalProperties"] != false {
		t.Fatalf("EvidenceSemantic contract = %#v", semantic)
	}
	for _, schema := range []string{"EvidenceTime", "EvidenceMetric", "EvidenceAttribution"} {
		if components[schema] == nil {
			t.Fatalf("Evidence contract is missing %q", schema)
		}
	}
	raw := components["RawEvidence"].(map[string]any)
	if _, exists := raw["properties"].(map[string]any)["keywords"]; exists {
		t.Fatalf("RawEvidence must not own keywords: %#v", raw)
	}
	for _, removed := range []string{"EvidenceLayerType", "expression_key", "fingerprint_version", "source_what", "source_what_core", "split_order"} {
		if _, exists := components[removed]; exists {
			t.Fatalf("removed Evidence schema %q is still published", removed)
		}
		if _, exists := atomicProperties[removed]; exists {
			t.Fatalf("removed AtomicEvidence property %q is still published", removed)
		}
	}
}

func TestRawEvidenceCategoryOpenAPIContract(t *testing.T) {
	var document map[string]any
	if err := yaml.Unmarshal(v1.Document(), &document); err != nil {
		t.Fatal(err)
	}
	components := document["components"].(map[string]any)["schemas"].(map[string]any)
	raw := components["RawEvidence"].(map[string]any)
	categoryIDs := raw["properties"].(map[string]any)["category_ids"].(map[string]any)
	if categoryIDs["type"] != "array" || categoryIDs["uniqueItems"] != true {
		t.Fatalf("RawEvidence category_ids = %#v", categoryIDs)
	}
	category := components["EvidenceCategory"].(map[string]any)
	properties := category["properties"].(map[string]any)
	for _, field := range []string{"id", "code", "name", "description"} {
		if _, exists := properties[field]; !exists {
			t.Fatalf("EvidenceCategory is missing %q", field)
		}
	}
	if len(properties) != 4 || category["additionalProperties"] != false {
		t.Fatalf("EvidenceCategory contract = %#v", category)
	}
	read := components["RawEvidenceRead"].(map[string]any)
	categories := read["properties"].(map[string]any)["categories"].(map[string]any)
	items := categories["items"].(map[string]any)
	if items["$ref"] != "#/components/schemas/EvidenceCategory" {
		t.Fatalf("RawEvidenceRead categories = %#v", categories)
	}
}

func TestEvidenceCategoryCatalogOpenAPIContract(t *testing.T) {
	var document map[string]any
	if err := yaml.Unmarshal(v1.Document(), &document); err != nil {
		t.Fatal(err)
	}
	paths := document["paths"].(map[string]any)
	path, exists := paths[v1.APIPrefix+"/evidence-categories"].(map[string]any)
	if !exists {
		t.Fatal("Evidence Category Catalog path is missing")
	}
	operation := path["get"].(map[string]any)
	for field, want := range map[string]any{
		"operationId":              "listEvidenceCategories",
		"x-client-drift-anchor":    "data.v1.listEvidenceCategories",
		"x-required-service-scope": "data.evidence-categories.read",
		"x-retry-policy":           "safe-get",
		"x-timeout-budget-ms":      3000,
	} {
		if got := operation[field]; got != want {
			t.Fatalf("%s = %#v, want %#v", field, got, want)
		}
	}
	parameters := operation["parameters"].([]any)
	if len(parameters) != 1 || parameters[0].(map[string]any)["$ref"] != "#/components/parameters/RequestID" {
		t.Fatalf("parameters = %#v, want only RequestID", parameters)
	}
	responses := operation["responses"].(map[string]any)
	for _, status := range []string{"200", "400", "401", "403", "500", "503"} {
		if _, exists := responses[status]; !exists {
			t.Fatalf("response %s is missing", status)
		}
	}
	if responses["500"].(map[string]any)["$ref"] != "#/components/responses/EvidenceCategoryCatalogFailed" ||
		responses["503"].(map[string]any)["$ref"] != "#/components/responses/EvidenceCategoryCatalogTimeout" {
		t.Fatalf("catalog error responses = 500:%#v 503:%#v", responses["500"], responses["503"])
	}
	components := document["components"].(map[string]any)["schemas"].(map[string]any)
	for schemaName, wantCode := range map[string]string{
		"EvidenceCategoryCatalogFailedErrorDetail":  ErrorEvidenceCategoryCatalogFailed,
		"EvidenceCategoryCatalogTimeoutErrorDetail": ErrorEvidenceCategoryCatalogTimeout,
	} {
		detail := components[schemaName].(map[string]any)
		code := detail["properties"].(map[string]any)["code"].(map[string]any)
		enum := code["enum"].([]any)
		if len(enum) != 1 || enum[0] != wantCode {
			t.Fatalf("%s code enum = %#v", schemaName, enum)
		}
	}
	category := components["EvidenceCategory"].(map[string]any)
	if category["additionalProperties"] != false {
		t.Fatal("EvidenceCategory must reject additional properties")
	}
	code := category["properties"].(map[string]any)["code"].(map[string]any)
	if code["pattern"] != "^[A-Z][A-Z0-9_]*$" {
		t.Fatalf("EvidenceCategory code pattern = %#v", code["pattern"])
	}
	result := components["EvidenceCategoryCatalog"].(map[string]any)
	if result["additionalProperties"] != false {
		t.Fatal("EvidenceCategoryCatalog must reject additional properties")
	}
	categories := result["properties"].(map[string]any)["categories"].(map[string]any)
	if categories["minItems"] != 1 {
		t.Fatalf("categories minItems = %#v, want 1", categories["minItems"])
	}
	envelope := components["EvidenceCategoryCatalogEnvelope"].(map[string]any)
	if envelope["additionalProperties"] != false {
		t.Fatal("EvidenceCategoryCatalogEnvelope must reject additional properties")
	}
}

func TestAdminEvidenceListOpenAPIContract(t *testing.T) {
	var document map[string]any
	if err := yaml.Unmarshal(v1.Document(), &document); err != nil {
		t.Fatal(err)
	}
	paths := document["paths"].(map[string]any)
	path, exists := paths[v1.APIPrefix+"/evidences"].(map[string]any)
	if !exists {
		t.Fatal("Admin Evidence list path is missing")
	}
	operation := path["get"].(map[string]any)
	for field, want := range map[string]any{
		"operationId":              "listAdminEvidence",
		"x-client-drift-anchor":    "data.v1.listAdminEvidence",
		"x-required-service-scope": "data.admin.read",
		"x-retry-policy":           "safe-get",
		"x-timeout-budget-ms":      3000,
	} {
		if got := operation[field]; got != want {
			t.Fatalf("%s = %#v, want %#v", field, got, want)
		}
	}
	wantParameters := []string{"X-Request-ID", "title", "summary", "category_id", "source_id", "source_name", "source_level", "is_split", "published_from", "published_to", "collected_from", "collected_to", "page", "page_size"}
	parameters := operation["parameters"].([]any)
	if len(parameters) != len(wantParameters) {
		t.Fatalf("parameters = %#v, want %d", parameters, len(wantParameters))
	}
	for index, want := range wantParameters {
		parameter := parameters[index].(map[string]any)
		if ref, ok := parameter["$ref"].(string); ok {
			wantRef := map[string]string{"X-Request-ID": "#/components/parameters/RequestID", "page": "#/components/parameters/Page", "page_size": "#/components/parameters/PageSize"}[want]
			if ref != wantRef {
				t.Fatalf("parameter[%d] = %#v, want %s", index, parameter, want)
			}
			continue
		}
		if parameter["name"] != want || parameter["in"] != "query" || parameter["required"] != false {
			t.Fatalf("parameter[%d] = %#v, want optional query %s", index, parameter, want)
		}
	}
	responses := operation["responses"].(map[string]any)
	for _, status := range []string{"200", "400", "401", "403", "500", "503"} {
		if _, exists := responses[status]; !exists {
			t.Fatalf("response %s is missing", status)
		}
	}
	components := document["components"].(map[string]any)["schemas"].(map[string]any)
	item := components["AdminEvidence"].(map[string]any)
	wantFields := []string{"id", "raw_evidence_id", "title", "summary", "semantic", "categories", "source_id", "source_name", "source_level", "source_url", "is_original", "quoted_source_name", "keywords", "is_split", "published_at", "collected_at"}
	properties := item["properties"].(map[string]any)
	if len(properties) != len(wantFields) || item["additionalProperties"] != false {
		t.Fatalf("AdminEvidence = %#v", item)
	}
	for _, field := range wantFields {
		if _, exists := properties[field]; !exists {
			t.Fatalf("AdminEvidence is missing %q", field)
		}
	}
	semantic := properties["semantic"].(map[string]any)
	if semantic["$ref"] != "#/components/schemas/EvidenceSemantic" {
		t.Fatalf("AdminEvidence semantic = %#v", semantic)
	}
	keywords := properties["keywords"].(map[string]any)
	if keywords["type"] != "array" {
		t.Fatalf("AdminEvidence keywords = %#v", keywords)
	}
	page := components["AdminEvidencePage"].(map[string]any)
	pageProperties := page["properties"].(map[string]any)
	if len(pageProperties) != 4 || page["additionalProperties"] != false {
		t.Fatalf("AdminEvidencePage = %#v", page)
	}
}
