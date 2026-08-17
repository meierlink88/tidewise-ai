package industrychain_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	industrychainapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/industrychain"
	industrychainbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/industrychain"
	industrychaindata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/industrychain"
	industrychainservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/entity/industrychain"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestIndustryChainHTTPContractPersistsIndependentIndustryChainFacts(t *testing.T) {
	db := openTestDatabase(t)
	handler := newHandler(t, db)

	created := request[industrychainapi.IndustryChain](t, handler, http.MethodPost, v1.APIPrefix+"/entities/industry-chains", `{
		"name":"先进制程产业链","aliases":["先进逻辑"],"scope":"先进逻辑芯片",
		"target_output":"先进制程芯片","end_use":"人工智能计算","geography":"全球",
		"primary_country_id":null,"as_of_date":"2026-08-17","review_status":"candidate",
		"review_note":null,"technology_route_qualifier":"EUV","observable_variables":["wafer_price"]
	}`, http.StatusCreated)
	if !industrychainbiz.IsID(created.ID) || created.Name != "先进制程产业链" || created.AsOfDate != "2026-08-17" || len(created.ObservableVariables) != 1 || created.CreatedAt == "" || created.UpdatedAt == "" {
		t.Fatalf("created IndustryChain = %#v", created)
	}

	updated := request[industrychainapi.IndustryChain](t, handler, http.MethodPut, v1.APIPrefix+"/entities/industry-chains/"+created.ID, `{
		"name":"先进逻辑芯片产业链","aliases":["先进逻辑"],"scope":"先进逻辑芯片全链条",
		"target_output":"先进逻辑芯片","end_use":"人工智能与高性能计算","geography":"全球",
		"primary_country_id":null,"as_of_date":"2026-08-18","review_status":"approved",
		"review_note":"已复核","technology_route_qualifier":null,"observable_variables":["wafer_price","lead_time"]
	}`, http.StatusOK)
	if updated.Name != "先进逻辑芯片产业链" || updated.ReviewStatus != "approved" || updated.ReviewNote == nil || len(updated.ObservableVariables) != 2 {
		t.Fatalf("updated IndustryChain = %#v", updated)
	}

	detail := request[industrychainapi.IndustryChain](t, handler, http.MethodGet, v1.APIPrefix+"/entities/industry-chains/"+created.ID, "", http.StatusOK)
	if detail.Name != updated.Name || detail.Scope != updated.Scope || detail.TechnologyRouteQualifier != nil {
		t.Fatalf("IndustryChain detail = %#v", detail)
	}
	second := request[industrychainapi.IndustryChain](t, handler, http.MethodPost, v1.APIPrefix+"/entities/industry-chains", `{
		"name":"成熟制程产业链","aliases":[],"scope":"成熟制程","target_output":"成熟制程芯片",
		"end_use":"汽车与工业","geography":"全球","primary_country_id":null,"as_of_date":"2026-08-17",
		"review_status":"candidate","review_note":null,"technology_route_qualifier":null,
		"observable_variables":["capacity_utilization"]
	}`, http.StatusCreated)
	firstPage := request[industrychainapi.IndustryChainList](t, handler, http.MethodGet, v1.APIPrefix+"/entities/industry-chains?page_size=1", "", http.StatusOK)
	if len(firstPage.Items) != 1 || firstPage.NextCursor == nil || len(*firstPage.NextCursor) > 256 {
		t.Fatalf("first IndustryChain page = %#v", firstPage)
	}
	secondPage := request[industrychainapi.IndustryChainList](t, handler, http.MethodGet, v1.APIPrefix+"/entities/industry-chains?page_size=1&cursor="+url.QueryEscape(*firstPage.NextCursor), "", http.StatusOK)
	if len(secondPage.Items) != 1 || secondPage.NextCursor != nil {
		t.Fatalf("second IndustryChain page = %#v", secondPage)
	}
	listedIDs := map[string]bool{firstPage.Items[0].ID: true, secondPage.Items[0].ID: true}
	if len(listedIDs) != 2 || !listedIDs[created.ID] || !listedIDs[second.ID] {
		t.Fatalf("listed IndustryChain IDs = %#v", listedIDs)
	}
	requestError(t, handler, http.MethodGet, v1.APIPrefix+"/entities/industry-chains?cursor=not-a-cursor", "", http.StatusBadRequest, industrychainapi.ErrorInvalidRequest)
	requestError(t, handler, http.MethodPost, v1.APIPrefix+"/entities/industry-chains", `{
		"industry_chain_id":"ENT99999999-9999-4999-8999-999999999999","name":"旧合同","aliases":[],
		"scope":"旧合同","target_output":"旧合同","end_use":"旧合同","geography":"全球",
		"primary_country_id":null,"as_of_date":"2026-08-17","review_status":"candidate","review_note":null,
		"technology_route_qualifier":null,"observable_variables":["legacy"]
	}`, http.StatusBadRequest, "INVALID_REQUEST")
	requestError(t, handler, http.MethodGet, v1.APIPrefix+"/entities/industry-chains/ENT99999999-9999-4999-8999-999999999999", "", http.StatusNotFound, industrychainapi.ErrorNotFound)

	var shadowRows int
	if err := db.QueryRowContext(context.Background(), `SELECT count(*) FROM entity_nodes WHERE id = ANY($1::text[])`, []string{created.ID, second.ID}).Scan(&shadowRows); err != nil {
		t.Fatal(err)
	}
	if shadowRows != 0 {
		t.Fatalf("IndustryChain writes created %d shadow Entity rows", shadowRows)
	}
}

func newHandler(t *testing.T, db *sql.DB) http.Handler {
	t.Helper()
	store, err := industrychaindata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := industrychainbiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	application, err := industrychainservice.NewService(useCase)
	if err != nil {
		t.Fatal(err)
	}
	server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(testErrorEncoder))
	industrychainapi.RegisterHTTPServer(server, application)
	return server
}

func openTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_industry_chain_http", migrationDir, 0)
}

func request[T any](t *testing.T, handler http.Handler, method, path, body string, wantStatus int) T {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(recorder, request)
	if recorder.Code != wantStatus {
		t.Fatalf("%s %s status = %d, want %d; body=%s", method, path, recorder.Code, wantStatus, recorder.Body.String())
	}
	var result T
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, recorder.Body.String())
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

func testErrorEncoder(response http.ResponseWriter, _ *http.Request, err error) {
	if public, ok := err.(*v1.PublicError); ok {
		response.WriteHeader(public.Status)
		_ = json.NewEncoder(response).Encode(public)
		return
	}
	response.WriteHeader(http.StatusInternalServerError)
}
