package concept_test

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
	conceptapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/concept"
	conceptbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/concept"
	conceptdata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/concept"
	conceptservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/entity/concept"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestConceptHTTPContractPersistsIndependentConceptFacts(t *testing.T) {
	db := openTestDatabase(t)
	handler := newHandler(t, db)

	created := request[conceptapi.Concept](t, handler, http.MethodPost, v1.APIPrefix+"/entities/concepts", `{
		"name":"人工智能","aliases":["AI"],"concept_type":"technology",
		"definition":"跨行业人工智能技术主题","review_status":"candidate"
	}`, http.StatusCreated)
	if !conceptbiz.IsID(created.ID) || created.ConceptType != "technology" || len(created.Aliases) != 1 || created.CreatedAt == "" || created.UpdatedAt == "" {
		t.Fatalf("created Concept = %#v", created)
	}

	updated := request[conceptapi.Concept](t, handler, http.MethodPut, v1.APIPrefix+"/entities/concepts/"+created.ID, `{
		"name":"生成式人工智能","aliases":["GenAI"],"concept_type":"market_theme",
		"definition":"生成式人工智能市场主题","review_status":"approved"
	}`, http.StatusOK)
	if updated.Name != "生成式人工智能" || updated.ConceptType != "market_theme" || updated.ReviewStatus != "approved" {
		t.Fatalf("updated Concept = %#v", updated)
	}

	detail := request[conceptapi.Concept](t, handler, http.MethodGet, v1.APIPrefix+"/entities/concepts/"+created.ID, "", http.StatusOK)
	if detail.Name != updated.Name || len(detail.Aliases) != 1 || detail.Aliases[0] != "GenAI" {
		t.Fatalf("Concept detail = %#v", detail)
	}
	second := request[conceptapi.Concept](t, handler, http.MethodPost, v1.APIPrefix+"/entities/concepts", `{
		"name":"算力","aliases":[],"concept_type":"demand",
		"definition":"人工智能算力需求主题","review_status":"candidate"
	}`, http.StatusCreated)
	firstPage := request[conceptapi.ConceptList](t, handler, http.MethodGet, v1.APIPrefix+"/entities/concepts?page_size=1", "", http.StatusOK)
	if len(firstPage.Items) != 1 || firstPage.Items[0].ID != created.ID || firstPage.NextCursor == nil || len(*firstPage.NextCursor) > 256 {
		t.Fatalf("first Concept page = %#v", firstPage)
	}
	secondPage := request[conceptapi.ConceptList](t, handler, http.MethodGet, v1.APIPrefix+"/entities/concepts?page_size=1&cursor="+url.QueryEscape(*firstPage.NextCursor), "", http.StatusOK)
	if len(secondPage.Items) != 1 || secondPage.Items[0].ID != second.ID || secondPage.NextCursor != nil {
		t.Fatalf("second Concept page = %#v", secondPage)
	}
	requestError(t, handler, http.MethodGet, v1.APIPrefix+"/entities/concepts?cursor=not-a-cursor", "", http.StatusBadRequest, conceptapi.ErrorInvalidRequest)

	requestError(t, handler, http.MethodPost, v1.APIPrefix+"/entities/concepts", `{
		"name":"错误概念","aliases":[],"concept_type":"sector",
		"definition":"错误类型","review_status":"candidate"
	}`, http.StatusUnprocessableEntity, "CONCEPT_INVALID")
	requestError(t, handler, http.MethodPost, v1.APIPrefix+"/entities/concepts", `{
		"id":"ENT99999999-9999-4999-8999-999999999999","name":"调用方 ID","aliases":[],
		"concept_type":"technology","definition":"禁止调用方 ID","review_status":"candidate"
	}`, http.StatusBadRequest, "INVALID_REQUEST")
	requestError(t, handler, http.MethodGet, v1.APIPrefix+"/entities/concepts/ENT99999999-9999-4999-8999-999999999999", "", http.StatusNotFound, "CONCEPT_NOT_FOUND")

	var shadowRows int
	if err := db.QueryRowContext(context.Background(), `SELECT count(*) FROM entity_nodes WHERE id = ANY($1::text[])`, []string{created.ID, second.ID}).Scan(&shadowRows); err != nil {
		t.Fatal(err)
	}
	if shadowRows != 0 {
		t.Fatalf("Concept write created %d shadow Entity rows", shadowRows)
	}
}

func newHandler(t *testing.T, db *sql.DB) http.Handler {
	t.Helper()
	store, err := conceptdata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := conceptbiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	application, err := conceptservice.NewService(useCase)
	if err != nil {
		t.Fatal(err)
	}
	server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(testErrorEncoder))
	conceptapi.RegisterHTTPServer(server, application)
	return server
}

func openTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_concept_http", migrationDir, 0)
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
