package industry_test

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
	industryapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/industry"
	industrybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/industry"
	industrydata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/industry"
	industryservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/entity/industry"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestIndustryHTTPContractPersistsIndependentIndustryFacts(t *testing.T) {
	db := openTestDatabase(t)
	handler := newHandler(t, db)

	root := request[industryapi.Industry](t, handler, http.MethodPost, v1.APIPrefix+"/entities/industries", `{
		"name":"半导体","aliases":["芯片产业"],"classification_system":"TIDEWISE",
		"industry_code":"SEMICONDUCTOR","hierarchy_path_codes":["SEMICONDUCTOR"],
		"definition":"半导体材料、设计、制造及相关活动","review_status":"candidate"
	}`, http.StatusCreated)
	if !industrybiz.IsID(root.ID) || root.ParentIndustryID != nil || len(root.Aliases) != 1 || root.CreatedAt == "" || root.UpdatedAt == "" {
		t.Fatalf("created root Industry = %#v", root)
	}

	child := request[industryapi.Industry](t, handler, http.MethodPost, v1.APIPrefix+"/entities/industries", `{
		"name":"集成电路","aliases":[],"classification_system":"TIDEWISE","industry_code":"IC",
		"parent_industry_id":"`+root.ID+`","hierarchy_path_codes":["SEMICONDUCTOR","IC"],
		"definition":"集成电路设计、制造及封测活动","review_status":"candidate"
	}`, http.StatusCreated)
	if child.ParentIndustryID == nil || *child.ParentIndustryID != root.ID || len(child.Aliases) != 0 {
		t.Fatalf("created child Industry = %#v", child)
	}

	updated := request[industryapi.Industry](t, handler, http.MethodPut, v1.APIPrefix+"/entities/industries/"+child.ID, `{
		"name":"集成电路产业","aliases":["IC"],"parent_industry_id":"`+root.ID+`",
		"hierarchy_path_codes":["SEMICONDUCTOR","IC"],"definition":"集成电路完整产业活动",
		"review_status":"approved"
	}`, http.StatusOK)
	if updated.Name != "集成电路产业" || updated.ReviewStatus != "approved" || updated.ClassificationSystem != "TIDEWISE" || updated.IndustryCode != "IC" {
		t.Fatalf("updated Industry = %#v", updated)
	}

	detail := request[industryapi.Industry](t, handler, http.MethodGet, v1.APIPrefix+"/entities/industries/"+child.ID, "", http.StatusOK)
	if detail.Name != updated.Name || detail.IndustryCode != child.IndustryCode {
		t.Fatalf("Industry detail = %#v", detail)
	}
	firstPage := request[industryapi.IndustryList](t, handler, http.MethodGet, v1.APIPrefix+"/entities/industries?page_size=1", "", http.StatusOK)
	if len(firstPage.Items) != 1 || firstPage.Items[0].ID != root.ID || firstPage.NextCursor == nil {
		t.Fatalf("first Industry page = %#v", firstPage)
	}
	secondPage := request[industryapi.IndustryList](t, handler, http.MethodGet, v1.APIPrefix+"/entities/industries?page_size=1&cursor="+url.QueryEscape(*firstPage.NextCursor), "", http.StatusOK)
	if len(secondPage.Items) != 1 || secondPage.Items[0].ID != child.ID || secondPage.NextCursor != nil {
		t.Fatalf("second Industry page = %#v", secondPage)
	}

	requestError(t, handler, http.MethodPost, v1.APIPrefix+"/entities/industries", `{
		"name":"重复半导体","aliases":[],"classification_system":"TIDEWISE",
		"industry_code":"SEMICONDUCTOR","hierarchy_path_codes":["SEMICONDUCTOR"],
		"definition":"重复分类身份","review_status":"candidate"
	}`, http.StatusConflict, "INDUSTRY_CONFLICT")
	requestError(t, handler, http.MethodPost, v1.APIPrefix+"/entities/industries", `{
		"name":"未知父级","aliases":[],"classification_system":"TIDEWISE","industry_code":"UNKNOWN_CHILD",
		"parent_industry_id":"ENT99999999-9999-4999-8999-999999999999",
		"hierarchy_path_codes":["UNKNOWN","UNKNOWN_CHILD"],"definition":"无效层级","review_status":"candidate"
	}`, http.StatusUnprocessableEntity, "INDUSTRY_REFERENCE_INVALID")
	requestError(t, handler, http.MethodPost, v1.APIPrefix+"/entities/industries", `{
		"id":"ENT99999999-9999-4999-8999-999999999999","name":"调用方 ID","aliases":[],
		"classification_system":"TIDEWISE","industry_code":"CALLER_ID","hierarchy_path_codes":["CALLER_ID"],
		"definition":"禁止调用方 ID","review_status":"candidate"
	}`, http.StatusBadRequest, "INVALID_REQUEST")

	var shadowRows int
	if err := db.QueryRowContext(context.Background(), `SELECT count(*) FROM entity_nodes WHERE id = ANY($1::text[])`, []string{root.ID, child.ID}).Scan(&shadowRows); err != nil {
		t.Fatal(err)
	}
	if shadowRows != 0 {
		t.Fatalf("Industry writes created %d shadow Entity rows", shadowRows)
	}
}

func newHandler(t *testing.T, db *sql.DB) http.Handler {
	t.Helper()
	store, err := industrydata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := industrybiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	application, err := industryservice.NewService(useCase)
	if err != nil {
		t.Fatal(err)
	}
	server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(testErrorEncoder))
	industryapi.RegisterHTTPServer(server, application)
	return server
}

func openTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_industry_http", migrationDir, 0)
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
