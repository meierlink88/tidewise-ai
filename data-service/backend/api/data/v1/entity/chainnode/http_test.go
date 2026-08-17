package chainnode_test

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
	chainnodeapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/chainnode"
	chainnodebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/chainnode"
	chainnodedata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/chainnode"
	chainnodeservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/entity/chainnode"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestChainNodeHTTPContractPersistsIndependentChainNodeFacts(t *testing.T) {
	db := openTestDatabase(t)
	handler := newHandler(t, db)

	created := request[chainnodeapi.ChainNode](t, handler, http.MethodPost, v1.APIPrefix+"/entities/chain-nodes", `{
		"name":"晶圆制造","aliases":["Wafer Fab"],
		"definition":"将设计版图制造为晶圆的产业链环节","review_status":"candidate"
	}`, http.StatusCreated)
	if !chainnodebiz.IsID(created.ID) || created.Name != "晶圆制造" || len(created.Aliases) != 1 || created.CreatedAt == "" || created.UpdatedAt == "" {
		t.Fatalf("created ChainNode = %#v", created)
	}

	updated := request[chainnodeapi.ChainNode](t, handler, http.MethodPut, v1.APIPrefix+"/entities/chain-nodes/"+created.ID, `{
		"name":"晶圆制造环节","aliases":["Wafer Fabrication"],
		"definition":"将芯片设计转化为晶圆产品的制造环节","review_status":"approved"
	}`, http.StatusOK)
	if updated.Name != "晶圆制造环节" || updated.ReviewStatus != "approved" || len(updated.Aliases) != 1 {
		t.Fatalf("updated ChainNode = %#v", updated)
	}

	detail := request[chainnodeapi.ChainNode](t, handler, http.MethodGet, v1.APIPrefix+"/entities/chain-nodes/"+created.ID, "", http.StatusOK)
	if detail.Name != updated.Name || detail.Definition != updated.Definition {
		t.Fatalf("ChainNode detail = %#v", detail)
	}
	second := request[chainnodeapi.ChainNode](t, handler, http.MethodPost, v1.APIPrefix+"/entities/chain-nodes", `{
		"name":"封装测试","aliases":[],"definition":"芯片封装与测试环节","review_status":"candidate"
	}`, http.StatusCreated)
	firstPage := request[chainnodeapi.ChainNodeList](t, handler, http.MethodGet, v1.APIPrefix+"/entities/chain-nodes?page_size=1", "", http.StatusOK)
	if len(firstPage.Items) != 1 || firstPage.NextCursor == nil || len(*firstPage.NextCursor) > 256 {
		t.Fatalf("first ChainNode page = %#v", firstPage)
	}
	secondPage := request[chainnodeapi.ChainNodeList](t, handler, http.MethodGet, v1.APIPrefix+"/entities/chain-nodes?page_size=1&cursor="+url.QueryEscape(*firstPage.NextCursor), "", http.StatusOK)
	if len(secondPage.Items) != 1 || secondPage.NextCursor != nil {
		t.Fatalf("second ChainNode page = %#v", secondPage)
	}
	listedIDs := map[string]bool{firstPage.Items[0].ID: true, secondPage.Items[0].ID: true}
	if len(listedIDs) != 2 || !listedIDs[created.ID] || !listedIDs[second.ID] {
		t.Fatalf("listed ChainNode IDs = %#v", listedIDs)
	}
	requestError(t, handler, http.MethodGet, v1.APIPrefix+"/entities/chain-nodes?cursor=not-a-cursor", "", http.StatusBadRequest, chainnodeapi.ErrorInvalidRequest)
	requestError(t, handler, http.MethodPost, v1.APIPrefix+"/entities/chain-nodes", `{
		"name":"旧合同","aliases":[],"definition":"旧合同不得继续写入",
		"boundary_note":"retired","review_status":"candidate"
	}`, http.StatusBadRequest, "INVALID_REQUEST")
	requestError(t, handler, http.MethodGet, v1.APIPrefix+"/entities/chain-nodes/CND99999999-9999-4999-8999-999999999999", "", http.StatusNotFound, chainnodeapi.ErrorNotFound)

	var shadowRows int
	if err := db.QueryRowContext(context.Background(), `SELECT count(*) FROM entity_nodes WHERE id = ANY($1::text[])`, []string{created.ID, second.ID}).Scan(&shadowRows); err != nil {
		t.Fatal(err)
	}
	if shadowRows != 0 {
		t.Fatalf("ChainNode writes created %d shadow Entity rows", shadowRows)
	}
}

func newHandler(t *testing.T, db *sql.DB) http.Handler {
	t.Helper()
	store, err := chainnodedata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := chainnodebiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	application, err := chainnodeservice.NewService(useCase)
	if err != nil {
		t.Fatal(err)
	}
	server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(testErrorEncoder))
	chainnodeapi.RegisterHTTPServer(server, application)
	return server
}

func openTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_chain_node_http", migrationDir, 0)
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
