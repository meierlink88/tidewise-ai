package server

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	dataapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	organizationapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/organization"
	organizationbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/organization"
	organizationdata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/organization"
	organizationservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/entity/organization"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/research"
)

func TestProductionServerOrganizationCreateAndReadContract(t *testing.T) {
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	db := postgresfixture.OpenIsolated(t, "tw_organization_server", migrationDir, 0)
	catalogPublication := organizationdata.CurrentCatalog()
	if err := organizationdata.PublishCatalog(context.Background(), db, catalogPublication); err != nil {
		t.Fatal(err)
	}
	if err := organizationdata.PublishCatalog(context.Background(), db, catalogPublication); err != nil {
		t.Fatalf("idempotent catalog publication: %v", err)
	}
	if _, err := db.ExecContext(context.Background(), `
INSERT INTO countries(id,code,name,name_en) VALUES('COU_CHN','CHN','中国','China');`); err != nil {
		t.Fatal(err)
	}
	store, err := organizationdata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := organizationbiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	application, err := organizationservice.NewService(useCase)
	if err != nil {
		t.Fatal(err)
	}
	authenticator, err := NewAuthenticator([]Credential{
		{Secret: "organization-read-token", Principal: dataapi.Principal{Identity: "organization-reader", Scopes: []string{ScopeOrganizationRead}}},
		{Secret: "organization-write-token", Principal: dataapi.Principal{Identity: "organization-writer", Scopes: []string{ScopeOrganizationWrite}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewHTTPServer(
		testConfig(), serverTestDataService{}, research.Service{}, serverTestEventService{},
		serverTestEventSemanticService{}, serverTestEvidenceService{}, serverTestRawDocumentService{},
		serverTestCountryService{}, application, authenticator, nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	created := productionContractRequest(t, server, http.MethodPost, dataapi.APIPrefix+"/entities/organizations", "organization-write-token", `{
		"id":"ORG_UN","code":"UN","name":"联合国","name_en":"United Nations",
		"region_id":null,"category_code":"INTERGOVERNMENTAL","function_code":"SECURITY",
		"legal_entity_code":null,"dominant_party_id":null,"binding_power_level":"HIGH",
		"influence_rating":"S","strategic_positioning":null,"core_impact_scope":null,
		"founding_document":"联合国宪章","established_date":"1945-10-24",
		"headquarters_city":"纽约","headquarters_country_id":null,
		"headquarters_subdivision_id":null,"description":null
	}`, "organization-contract-request", http.StatusCreated)
	result := created["result"].(map[string]any)
	if result["id"] != "ORG_UN" || result["category"].(map[string]any)["name_zh"] != "政府间国际组织" {
		t.Fatalf("created Organization envelope = %#v", created)
	}
	productionContractRequest(t, server, http.MethodPost, dataapi.APIPrefix+"/entities/organizations", "organization-write-token", `{
		"id":"ORG_ASEAN","code":"ASEAN","name":"东南亚国家联盟","name_en":"Association of Southeast Asian Nations",
		"category_code":"TRADE_BLOC","function_code":"TRADE"
	}`, "organization-contract-request", http.StatusCreated)
	productionContractRequest(t, server, http.MethodPost, dataapi.APIPrefix+"/entities/organizations", "organization-write-token", `{
		"id":"ORG_INVALID","code":"INVALID","name":"无效组织","name_en":"Invalid Organization",
		"category_code":"UNKNOWN_CATEGORY","function_code":"TRADE"
	}`, "organization-contract-request", http.StatusUnprocessableEntity)

	detail := productionContractRequest(t, server, http.MethodGet, dataapi.APIPrefix+"/entities/organizations/ORG_UN", "organization-read-token", "", "organization-contract-request", http.StatusOK)
	detailResult := detail["result"].(map[string]any)
	if detailResult["function"].(map[string]any)["code"] != "SECURITY" {
		t.Fatalf("Organization detail envelope = %#v", detail)
	}

	listed := productionContractRequest(t, server, http.MethodGet, dataapi.APIPrefix+"/entities/organizations?category_code=INTERGOVERNMENTAL", "organization-read-token", "", "organization-contract-request", http.StatusOK)
	items := listed["result"].(map[string]any)["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["id"] != "ORG_UN" {
		t.Fatalf("Organization list envelope = %#v", listed)
	}
	allOrganizations := productionContractRequest(t, server, http.MethodGet, dataapi.APIPrefix+"/entities/organizations", "organization-read-token", "", "organization-contract-request", http.StatusOK)
	allItems := allOrganizations["result"].(map[string]any)["items"].([]any)
	if len(allItems) != 2 || allItems[0].(map[string]any)["id"] != "ORG_ASEAN" || allItems[1].(map[string]any)["id"] != "ORG_UN" {
		t.Fatalf("Organization list is not stably ordered by code: %#v", allOrganizations)
	}

	catalog := productionContractRequest(t, server, http.MethodGet, dataapi.APIPrefix+"/organization-catalog", "organization-read-token", "", "organization-contract-request", http.StatusOK)
	if len(catalog["result"].(map[string]any)["domain_tags"].([]any)) != 21 {
		t.Fatalf("Organization catalog = %#v", catalog)
	}

	tagged := productionContractRequest(t, server, http.MethodPut, dataapi.APIPrefix+"/entities/organizations/ORG_UN/domain-tags", "organization-write-token", `{"domain_tag_codes":["REGIONAL_SECURITY_DIALOGUE"]}`, "organization-contract-request", http.StatusOK)
	if len(tagged["result"].(map[string]any)["domain_tags"].([]any)) != 1 {
		t.Fatalf("tagged Organization = %#v", tagged)
	}
	updated := productionContractRequest(t, server, http.MethodPut, dataapi.APIPrefix+"/entities/organizations/ORG_UN", "organization-write-token", `{"name":"联合国","name_en":"United Nations","region_id":null,"category_code":"INTERGOVERNMENTAL","function_code":"TECHNOLOGY","legal_entity_code":null,"dominant_party_id":null,"binding_power_level":"HIGH","influence_rating":"S","strategic_positioning":null,"core_impact_scope":null,"founding_document":"联合国宪章","established_date":"1945-10-24","headquarters_city":"纽约","headquarters_country_id":null,"headquarters_subdivision_id":null,"description":null,"domain_tag_codes":["AI_TECHNOLOGY_AND_GOVERNANCE"]}`, "organization-contract-request", http.StatusOK)
	updatedResult := updated["result"].(map[string]any)
	if updatedResult["function"].(map[string]any)["code"] != "TECHNOLOGY" || updatedResult["domain_tags"].([]any)[0].(map[string]any)["function_code"] != "TECHNOLOGY" {
		t.Fatalf("atomically updated Organization = %#v", updated)
	}

	member := productionContractRequest(t, server, http.MethodPost, dataapi.APIPrefix+"/entities/organizations/ORG_UN/members", "organization-write-token", `{"country_id":"COU_CHN","membership_type":"FULL_MEMBER","effective_date":"2020-01-01","expiry_date":null}`, "organization-contract-request", http.StatusCreated)
	memberID := int64(member["result"].(map[string]any)["id"].(float64))
	productionContractRequest(t, server, http.MethodPost, dataapi.APIPrefix+"/entities/organizations/ORG_UN/members", "organization-write-token", `{"country_id":"COU_USA","membership_type":"FULL_MEMBER","effective_date":null,"expiry_date":null}`, "organization-contract-request", http.StatusUnprocessableEntity)
	productionContractRequest(t, server, http.MethodPost, dataapi.APIPrefix+"/entities/organizations/ORG_UN/members", "organization-write-token", `{"country_id":"COU_CHN","membership_type":"OBSERVER","effective_date":"2021-01-01","expiry_date":null}`, "organization-contract-request", http.StatusConflict)

	productionContractRequest(t, server, http.MethodPut, fmt.Sprintf("%s/entities/organizations/ORG_UN/members/%d", dataapi.APIPrefix, memberID), "organization-write-token", `{"country_id":"COU_CHN","membership_type":"FULL_MEMBER","effective_date":"2020-01-01","expiry_date":"2020-12-31"}`, "organization-contract-request", http.StatusOK)
	secondMember := productionContractRequest(t, server, http.MethodPost, dataapi.APIPrefix+"/entities/organizations/ORG_UN/members", "organization-write-token", `{"country_id":"COU_CHN","membership_type":"OBSERVER","effective_date":"2021-01-01","expiry_date":null}`, "organization-contract-request", http.StatusCreated)
	secondMemberID := int64(secondMember["result"].(map[string]any)["id"].(float64))
	asOf := productionContractRequest(t, server, http.MethodGet, dataapi.APIPrefix+"/entities/organizations/ORG_UN/members?as_of=2020-06-01", "organization-read-token", "", "organization-contract-request", http.StatusOK)
	if asOf["result"].(map[string]any)["items"].([]any)[0].(map[string]any)["membership_type"] != "FULL_MEMBER" {
		t.Fatalf("as-of members = %#v", asOf)
	}
	memberFiltered := productionContractRequest(t, server, http.MethodGet, dataapi.APIPrefix+"/entities/organizations?country_id=COU_CHN&as_of=2021-06-01", "organization-read-token", "", "organization-contract-request", http.StatusOK)
	filteredItems := memberFiltered["result"].(map[string]any)["items"].([]any)
	if len(filteredItems) != 1 || filteredItems[0].(map[string]any)["id"] != "ORG_UN" {
		t.Fatalf("member Country filter = %#v", memberFiltered)
	}
	productionContractRequest(t, server, http.MethodPut, dataapi.APIPrefix+"/entities/organizations/ORG_UN/members/999999", "organization-write-token", `{"country_id":"COU_CHN","membership_type":"OBSERVER","effective_date":null,"expiry_date":null}`, "organization-contract-request", http.StatusNotFound)
	productionContractRequest(t, server, http.MethodPut, dataapi.APIPrefix+"/entities/organizations/ORG_MISSING/domain-tags", "organization-write-token", `{"domain_tag_codes":[]}`, "organization-contract-request", http.StatusNotFound)
	productionContractRequest(t, server, http.MethodDelete, fmt.Sprintf("%s/entities/organizations/ORG_UN/members/%d", dataapi.APIPrefix, secondMemberID), "organization-write-token", "", "organization-contract-request", http.StatusOK)
}

var _ organizationapi.Service = (*organizationservice.Service)(nil)
