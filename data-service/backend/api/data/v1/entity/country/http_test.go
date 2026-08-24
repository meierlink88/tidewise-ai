package country_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	countryapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/country"
	entitybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity"
	countrybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/country"
	entitydata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity"
	countrydata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/country"
	countryservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/entity/country"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestCountryHTTPContractPersistsCountryAndRegionRelationships(t *testing.T) {
	db := openCountryTestDatabase(t)
	seedRegion(t, db, "REG88d53cc8-1c75-57e6-a02c-56f9a4bc13c4", "APAC", "亚太地区", "Asia Pacific", "GEOGRAPHIC")
	seedRegion(t, db, "REG3d30569b-9ea9-5949-96a5-3c1ee26655d8", "EM", "新兴市场", "Emerging Markets", "INVESTMENT")
	handler := newCountryHandler(t, db)

	created := request[countryapi.Country](t, handler, http.MethodPost, v1.APIPrefix+"/entities/countries", `{
		"code":"CN",
		"name":"中国",
		"name_en":"China",
		"strategic_positioning":"全球制造业与供应链枢纽",
		"key_resources":null
	}`, http.StatusCreated)
	if !entitybiz.IsCountryID(created.ID) || created.Code != "CN" || created.Name != "中国" || created.NameEn != "China" {
		t.Fatalf("created Country = %#v", created)
	}
	if created.CreatedAt == "" || created.UpdatedAt == "" || created.KeyResources != nil {
		t.Fatalf("created timestamps/resources = %#v", created)
	}

	related := request[countryapi.Country](t, handler, http.MethodPut, v1.APIPrefix+"/entities/countries/"+created.ID+"/regions", `{
		"region_ids":["REG88d53cc8-1c75-57e6-a02c-56f9a4bc13c4","REG3d30569b-9ea9-5949-96a5-3c1ee26655d8"]
	}`, http.StatusOK)
	if len(related.Regions) != 2 || related.Regions[0].ID != "REG88d53cc8-1c75-57e6-a02c-56f9a4bc13c4" || related.Regions[1].ID != "REG3d30569b-9ea9-5949-96a5-3c1ee26655d8" {
		t.Fatalf("Country regions = %#v", related.Regions)
	}

	updated := request[countryapi.Country](t, handler, http.MethodPut, v1.APIPrefix+"/entities/countries/"+created.ID, `{
		"name":"中华人民共和国",
		"name_en":"China",
		"strategic_positioning":null,
		"key_resources":"稀土、制造业基础与完整供应链"
	}`, http.StatusOK)
	if updated.Name != "中华人民共和国" || updated.StrategicPositioning != nil || updated.KeyResources == nil {
		t.Fatalf("updated Country = %#v", updated)
	}
	if updated.UpdatedAt == created.UpdatedAt {
		t.Fatalf("updated_at did not advance: %q", updated.UpdatedAt)
	}

	detail := request[countryapi.Country](t, handler, http.MethodGet, v1.APIPrefix+"/entities/countries/"+created.ID, "", http.StatusOK)
	if detail.Name != updated.Name || len(detail.Regions) != 2 {
		t.Fatalf("Country detail = %#v", detail)
	}

	page := request[countryapi.CountryList](t, handler, http.MethodGet, v1.APIPrefix+"/entities/countries?region_id=REG88d53cc8-1c75-57e6-a02c-56f9a4bc13c4", "", http.StatusOK)
	if len(page.Items) != 1 || page.Items[0].ID != created.ID || len(page.Items[0].Regions) != 2 {
		t.Fatalf("Country list = %#v", page)
	}

	entityStore, err := entitydata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := entityStore.SearchResearchGraph(context.Background(), entitybiz.ResearchGraphQuery{
		AnalysisAsOf: time.Now().UTC().Add(time.Second), SeedEntityIDs: []string{created.ID},
		RelationFilters: []entitybiz.ResearchGraphRelationFilter{{
			RelationType: "belongs_to_region", Direction: entitybiz.ResearchGraphDirectionOutgoing,
		}},
		MaxDepth: 1, NodeBudget: 10, EdgeBudget: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if graph.ActualDepth != 1 || len(graph.Entities) != 3 || len(graph.EntityRelations) != 2 ||
		graph.Entities[0].EntityID != created.ID || graph.Entities[0].EntityType != "country" {
		t.Fatalf("Country Research Graph = %#v", graph)
	}

	unchanged := request[countryapi.Country](t, handler, http.MethodPut, v1.APIPrefix+"/entities/countries/"+created.ID, `{
		"name":"中华人民共和国",
		"name_en":"China",
		"strategic_positioning":null,
		"key_resources":"稀土、制造业基础与完整供应链"
	}`, http.StatusOK)
	if unchanged.UpdatedAt != updated.UpdatedAt {
		t.Fatalf("no-op updated_at = %q, want %q", unchanged.UpdatedAt, updated.UpdatedAt)
	}

	var genericCountryRows int
	if err := db.QueryRowContext(context.Background(), `SELECT count(*) FROM entity_nodes WHERE entity_type = 'country' OR name = '中华人民共和国'`).Scan(&genericCountryRows); err != nil {
		t.Fatal(err)
	}
	if genericCountryRows != 0 {
		t.Fatalf("Country created %d generic Entity rows", genericCountryRows)
	}
	var oldProfileTable *string
	if err := db.QueryRowContext(context.Background(), `SELECT to_regclass('economy_profiles')::text`).Scan(&oldProfileTable); err != nil {
		t.Fatal(err)
	}
	if oldProfileTable != nil {
		t.Fatalf("retired profile table remains: %s", *oldProfileTable)
	}

	requestError(t, handler, http.MethodPost, v1.APIPrefix+"/entities/countries", `{
		"code":"CHN","name":"美国","name_en":"United States"
	}`, http.StatusUnprocessableEntity, "COUNTRY_INVALID")
	requestError(t, handler, http.MethodPost, v1.APIPrefix+"/entities/countries", `{
		"code":"US","name":" ","name_en":"United States"
	}`, http.StatusUnprocessableEntity, "COUNTRY_INVALID")
	requestError(t, handler, http.MethodPost, v1.APIPrefix+"/entities/countries", `{
		"code":"US","name":"美国","name_en":"United States","key_resources":" "
	}`, http.StatusUnprocessableEntity, "COUNTRY_INVALID")
	requestError(t, handler, http.MethodPost, v1.APIPrefix+"/entities/countries", `{
		"code":"CN","name":"中国","name_en":"China"
	}`, http.StatusConflict, "COUNTRY_CONFLICT")
	requestError(t, handler, http.MethodPut, v1.APIPrefix+"/entities/countries/"+created.ID+"/regions", `{
		"region_ids":["REG7a0c86e7-c95d-5d23-9eb9-a73bfa47d9a6"]
	}`, http.StatusUnprocessableEntity, "COUNTRY_REFERENCE_INVALID")
	requestError(t, handler, http.MethodPut, v1.APIPrefix+"/entities/countries/"+created.ID+"/regions", `{
		"region_ids":["REG88d53cc8-1c75-57e6-a02c-56f9a4bc13c4","REG88d53cc8-1c75-57e6-a02c-56f9a4bc13c4"]
	}`, http.StatusUnprocessableEntity, "COUNTRY_INVALID")
	requestError(t, handler, http.MethodGet, v1.APIPrefix+"/entities/countries/COUd92518af-6eec-55be-9b60-237c38f98719", "", http.StatusNotFound, "COUNTRY_NOT_FOUND")
	requestError(t, handler, http.MethodPost, v1.APIPrefix+"/entities/countries", `{
		"code":"US","name":"美国","name_en":"United States","unknown":true
	}`, http.StatusBadRequest, "INVALID_REQUEST")
	requestError(t, handler, http.MethodPut, v1.APIPrefix+"/entities/countries/"+created.ID, `{
		"code":"US","name":"中国","name_en":"China","strategic_positioning":null,"key_resources":null
	}`, http.StatusBadRequest, "INVALID_REQUEST")
	preserved := request[countryapi.Country](t, handler, http.MethodGet, v1.APIPrefix+"/entities/countries/"+created.ID, "", http.StatusOK)
	if len(preserved.Regions) != 2 {
		t.Fatalf("failed Region replacement changed links: %#v", preserved.Regions)
	}
	empty := request[countryapi.Country](t, handler, http.MethodPut, v1.APIPrefix+"/entities/countries/"+created.ID+"/regions", `{
		"region_ids":[]
	}`, http.StatusOK)
	if len(empty.Regions) != 0 {
		t.Fatalf("empty Region replacement retained links: %#v", empty.Regions)
	}
}

func request[T any](t *testing.T, handler http.Handler, method, path, body string, wantStatus int) T {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, path, bytes.NewBufferString(body)))
	if recorder.Code != wantStatus {
		t.Fatalf("%s %s status = %d, want %d; body=%s", method, path, recorder.Code, wantStatus, recorder.Body.String())
	}
	var result T
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode %s %s response: %v; body=%s", method, path, err, recorder.Body.String())
	}
	return result
}

func requestError(t *testing.T, handler http.Handler, method, path, body string, wantStatus int, wantCode string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(recorder, request)
	if recorder.Code != wantStatus || !bytes.Contains(recorder.Body.Bytes(), []byte(wantCode)) {
		t.Fatalf("%s %s status=%d want=%d code=%s body=%s", method, path, recorder.Code, wantStatus, wantCode, recorder.Body.String())
	}
}

func newCountryHandler(t *testing.T, db *sql.DB) http.Handler {
	t.Helper()
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
	server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(func(response http.ResponseWriter, _ *http.Request, err error) {
		if public, ok := err.(*v1.PublicError); ok {
			response.WriteHeader(public.Status)
			_ = json.NewEncoder(response).Encode(public)
			return
		}
		response.WriteHeader(http.StatusInternalServerError)
	}))
	countryapi.RegisterHTTPServer(server, application)
	return server
}

func openCountryTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_country_http", migrationDir, 0)
}

func seedRegion(t *testing.T, db *sql.DB, id, code, name, nameEn, regionType string) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), `
INSERT INTO regions (id, code, name, name_en, region_type)
VALUES ($1, $2, $3, $4, $5)`, id, code, name, nameEn, regionType); err != nil {
		t.Fatal(err)
	}
}
