package server

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"

	dataapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	countryapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/country"
	countrybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/country"
	countrydata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/country"
	countryservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/entity/country"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/research"
)

func TestProductionServerCountryContractUsesPostgresEnvelopeRequestIDAndScopes(t *testing.T) {
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	db := postgresfixture.OpenIsolated(t, "tw_country_server", migrationDir, 0)
	if _, err := db.ExecContext(context.Background(), `
INSERT INTO regions (id, code, name, name_en, region_type)
VALUES ('REG_APAC', 'APAC', '亚太地区', 'Asia Pacific', 'GEOGRAPHIC')`); err != nil {
		t.Fatal(err)
	}
	store, err := countrydata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := countrybiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	application, err := countryservice.NewService(useCase)
	if err != nil {
		t.Fatal(err)
	}
	authenticator, err := NewAuthenticator([]Credential{
		{Secret: "country-read-token", Principal: dataapi.Principal{Identity: "country-reader", Scopes: []string{ScopeCountryRead}}},
		{Secret: "country-write-token", Principal: dataapi.Principal{Identity: "country-writer", Scopes: []string{ScopeCountryWrite}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewHTTPServer(
		testConfig(), serverTestDataService{}, research.Service{}, serverTestEventService{},
		serverTestEventSemanticService{}, serverTestEvidenceService{}, serverTestRawDocumentService{},
		application, serverTestOrganizationService{}, authenticator, nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	created := productionCountryRequest(t, server, http.MethodPost, dataapi.APIPrefix+"/entities/countries", "country-write-token", `{
		"id":"COU_CHN","code":"CHN","name":"中国","name_en":"China",
		"strategic_positioning":null,"key_resources":null
	}`, http.StatusCreated)
	result := created["result"].(map[string]any)
	if result["id"] != "COU_CHN" || result["code"] != "CHN" {
		t.Fatalf("created Country envelope = %#v", created)
	}
	productionCountryRequest(t, server, http.MethodPut, dataapi.APIPrefix+"/entities/countries/COU_CHN/regions", "country-write-token", `{
		"region_ids":["REG_APAC"]
	}`, http.StatusOK)
	updated := productionCountryRequest(t, server, http.MethodPut, dataapi.APIPrefix+"/entities/countries/COU_CHN", "country-write-token", `{
		"name":"中华人民共和国","name_en":"China",
		"strategic_positioning":"全球制造业与供应链枢纽","key_resources":null
	}`, http.StatusOK)
	if updated["result"].(map[string]any)["name"] != "中华人民共和国" {
		t.Fatalf("updated Country envelope = %#v", updated)
	}
	detail := productionCountryRequest(t, server, http.MethodGet, dataapi.APIPrefix+"/entities/countries/COU_CHN", "country-read-token", "", http.StatusOK)
	detailResult := detail["result"].(map[string]any)
	if regions, ok := detailResult["regions"].([]any); !ok || len(regions) != 1 {
		t.Fatalf("Country detail envelope = %#v", detail)
	}
	listed := productionCountryRequest(t, server, http.MethodGet, dataapi.APIPrefix+"/entities/countries?region_id=REG_APAC", "country-read-token", "", http.StatusOK)
	items := listed["result"].(map[string]any)["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["id"] != "COU_CHN" {
		t.Fatalf("Region-filtered Country envelope = %#v", listed)
	}
	productionCountryError(t, server, http.MethodPut, dataapi.APIPrefix+"/entities/countries/COU_CHN", "country-write-token", `{
		"code":"USA","name":"中国","name_en":"China","strategic_positioning":null,"key_resources":null
	}`, http.StatusBadRequest, "INVALID_REQUEST")
	productionCountryError(t, server, http.MethodGet, dataapi.APIPrefix+"/entities/countries/COU_USA", "country-read-token", "", http.StatusNotFound, "COUNTRY_NOT_FOUND")
	productionCountryError(t, server, http.MethodPost, dataapi.APIPrefix+"/entities/countries", "country-write-token", `{
		"id":"COU_CHN","code":"CHN","name":"中国","name_en":"China",
		"strategic_positioning":null,"key_resources":null
	}`, http.StatusConflict, "COUNTRY_CONFLICT")
	productionCountryError(t, server, http.MethodPost, dataapi.APIPrefix+"/entities/countries", "country-write-token", `{
		"id":"COU_USA","code":"CHN","name":"美国","name_en":"United States",
		"strategic_positioning":null,"key_resources":null
	}`, http.StatusUnprocessableEntity, "COUNTRY_INVALID")
	forbidden := productionCountryRequest(t, server, http.MethodGet, dataapi.APIPrefix+"/entities/countries", "country-write-token", "", http.StatusForbidden)
	if forbidden["error"].(map[string]any)["code"] != "FORBIDDEN" {
		t.Fatalf("forbidden Country envelope = %#v", forbidden)
	}
	unauthorized := productionCountryRequest(t, server, http.MethodGet, dataapi.APIPrefix+"/entities/countries", "", "", http.StatusUnauthorized)
	if unauthorized["error"].(map[string]any)["code"] != "UNAUTHENTICATED" {
		t.Fatalf("unauthorized Country envelope = %#v", unauthorized)
	}
}

func productionCountryRequest(t *testing.T, handler http.Handler, method, path, token, body string, wantStatus int) map[string]any {
	t.Helper()
	return productionContractRequest(t, handler, method, path, token, body, "country-contract-request", wantStatus)
}

func productionCountryError(t *testing.T, handler http.Handler, method, path, token, body string, wantStatus int, wantCode string) {
	t.Helper()
	_ = productionContractError(t, handler, method, path, token, body, "country-contract-request", wantStatus, wantCode)
}

var _ countryapi.Service = (*countryservice.Service)(nil)
