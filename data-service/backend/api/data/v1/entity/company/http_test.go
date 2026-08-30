package company_test

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
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	companyapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/company"
	companybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/company"
	companydata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/company"
	companyservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/entity/company"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

type fixtureService struct {
	request *companyapi.ListRequest
}

func (s *fixtureService) List(_ context.Context, request *companyapi.ListRequest) (*v1.Response[companyapi.CompanyProjectionPage], error) {
	s.request = request
	return &v1.Response[companyapi.CompanyProjectionPage]{
		Status: http.StatusOK,
		Result: companyapi.CompanyProjectionPage{
			SchemaVersion: companyapi.ProjectionSchemaVersion,
			SnapshotID:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Items:         []companyapi.Company{},
			NextCursor:    nil,
		},
	}, nil
}

func TestCompanyProjectionHTTPBindsOnlyTheFrozenQuery(t *testing.T) {
	application := &fixtureService{}
	server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(testErrorEncoder))
	companyapi.RegisterHTTPServer(server, application)

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, v1.APIPrefix+"/entities/companies?page_size=25&cursor=opaque", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if application.request == nil || application.request.PageSize != "25" || application.request.Cursor != "opaque" {
		t.Fatalf("bound request = %#v", application.request)
	}

	for _, target := range []string{
		v1.APIPrefix + "/entities/companies?unknown=value",
		v1.APIPrefix + "/entities/companies?page_size=1&page_size=2",
		v1.APIPrefix + "/entities/companies?page_size=",
		v1.APIPrefix + "/entities/companies?page_size=%20",
		v1.APIPrefix + "/entities/companies?cursor=",
	} {
		response = httptest.NewRecorder()
		server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("GET %s status=%d body=%s", target, response.Code, response.Body.String())
		}
	}
}

func TestCompanyProjectionHTTPReturnsFormalLinksAndRejectsSnapshotDrift(t *testing.T) {
	db := openTestDatabase(t)
	ctx := context.Background()
	store, err := companydata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	industryID := companybiz.IndustryID("IND33333333-3333-4333-8333-333333333333")
	if _, err := db.ExecContext(ctx, `INSERT INTO industry (
    id, classification_system, industry_code, hierarchy_path_codes,
    definition, review_status, name, aliases
) VALUES ($1, 'TIDEWISE', 'SEMICONDUCTOR', ARRAY['SEMICONDUCTOR'],
    '半导体行业', 'approved', '半导体', '{}')`, industryID); err != nil {
		t.Fatal(err)
	}
	founding := time.Date(1987, time.February, 21, 0, 0, 0, 0, time.UTC)
	firstID := companybiz.ID("COM11111111-1111-4111-8111-111111111111")
	secondID := companybiz.ID("COM22222222-2222-4222-8222-222222222222")
	for _, input := range []companybiz.Company{
		{ID: firstID, Code: "000001.SZ", Name: "First", Aliases: []string{}, FoundingDate: &founding, Status: companybiz.StatusActive},
		{ID: secondID, Code: "000002.SZ", Name: "Second", Aliases: []string{}, Status: companybiz.StatusActive},
	} {
		if _, err := store.Create(ctx, input); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.ReplaceIndustries(ctx, firstID, []companybiz.IndustryLink{{
		ID: "CIL44444444-4444-4444-8444-444444444444", IndustryID: industryID,
	}}); err != nil {
		t.Fatal(err)
	}
	useCase, err := companybiz.NewProjectionUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	application, err := companyservice.NewService(useCase)
	if err != nil {
		t.Fatal(err)
	}
	server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(testErrorEncoder))
	companyapi.RegisterHTTPServer(server, application)

	firstPage := request[companyapi.CompanyProjectionPage](t, server, v1.APIPrefix+"/entities/companies?page_size=1", http.StatusOK)
	if firstPage.SchemaVersion != companyapi.ProjectionSchemaVersion || len(firstPage.SnapshotID) != 64 || len(firstPage.Items) != 1 || firstPage.NextCursor == nil {
		t.Fatalf("first page = %#v", firstPage)
	}
	company := firstPage.Items[0]
	if company.ID != string(firstID) || company.FoundingDate == nil || *company.FoundingDate != "1987-02-21" || len(company.IndustryLinks) != 1 || company.IndustryLinks[0].CompanyID != string(firstID) {
		t.Fatalf("projected Company = %#v", company)
	}
	secondPage := request[companyapi.CompanyProjectionPage](t, server, v1.APIPrefix+"/entities/companies?page_size=1&cursor="+url.QueryEscape(*firstPage.NextCursor), http.StatusOK)
	if len(secondPage.Items) != 1 || secondPage.Items[0].ID != string(secondID) || secondPage.NextCursor != nil || secondPage.SnapshotID != firstPage.SnapshotID {
		t.Fatalf("second page = %#v", secondPage)
	}
	if _, err := db.ExecContext(ctx, `UPDATE company SET name = 'Changed', updated_at = now() WHERE id = $1`, secondID); err != nil {
		t.Fatal(err)
	}
	requestError(t, server, v1.APIPrefix+"/entities/companies?page_size=1&cursor="+url.QueryEscape(*firstPage.NextCursor), http.StatusConflict, companyapi.ErrorSnapshotChanged)
}

func openTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_company_projection_http", migrationDir, 0)
}

func request[T any](t *testing.T, handler http.Handler, path string, wantStatus int) T {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Code != wantStatus {
		t.Fatalf("GET %s status=%d want=%d body=%s", path, response.Code, wantStatus, response.Body.String())
	}
	var result T
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, response.Body.String())
	}
	return result
}

func requestError(t *testing.T, handler http.Handler, path string, wantStatus int, wantCode string) {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Code != wantStatus || !bytes.Contains(response.Body.Bytes(), []byte(wantCode)) {
		t.Fatalf("GET %s status=%d want=%d code=%s body=%s", path, response.Code, wantStatus, wantCode, response.Body.String())
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

var _ companyapi.Service = (*fixtureService)(nil)
