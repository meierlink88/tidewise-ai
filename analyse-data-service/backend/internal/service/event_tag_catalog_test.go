package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventtagcatalog"
)

type fakeEventTagCatalogService struct {
	catalog eventtagcatalog.Catalog
	err     error
}

func (f fakeEventTagCatalogService) Active(context.Context) (eventtagcatalog.Catalog, error) {
	return f.catalog, f.err
}

func TestEventTagCatalogHTTPReturnsAgentRunSnapshot(t *testing.T) {
	const catalogHash = "1a7035195312cdf7652880308d9fffcc6aea180f7c09b5c07f6678514a1298eb"
	handler := dataServiceTestHandler(
		Dependencies{EventTagCatalog: fakeEventTagCatalogService{catalog: eventtagcatalog.Catalog{
			Revision: "event-tags:" + catalogHash,
			Hash:     catalogHash,
			Tags: []eventtagcatalog.Tag{{
				ID: "22a5afc5-20ed-55ce-bf77-54c26bbcc6ea", Kind: "news_category",
				Code: "technology_industry", Name: "科技产业", Active: true,
			}},
		}}},
		map[string]v1.Principal{
			"catalog-reader": {Identity: "agent-run", Scopes: []string{ScopeEventTagRead}},
			"event-writer":   {Identity: "agent-run", Scopes: []string{ScopeReviewedEventImport}},
		},
		"catalog-request",
	)

	for _, test := range []struct {
		name       string
		target     string
		token      string
		wantStatus int
		wantCode   string
	}{
		{name: "active catalog", target: Namespace + "/event-tags?active=true", token: "catalog-reader", wantStatus: http.StatusOK},
		{name: "missing active", target: Namespace + "/event-tags", token: "catalog-reader", wantStatus: http.StatusBadRequest, wantCode: "INVALID_REQUEST"},
		{name: "false active", target: Namespace + "/event-tags?active=false", token: "catalog-reader", wantStatus: http.StatusBadRequest, wantCode: "INVALID_REQUEST"},
		{name: "unknown query", target: Namespace + "/event-tags?active=true&other=x", token: "catalog-reader", wantStatus: http.StatusBadRequest, wantCode: "INVALID_REQUEST"},
		{name: "wrong scope", target: Namespace + "/event-tags?active=true", token: "event-writer", wantStatus: http.StatusForbidden, wantCode: "FORBIDDEN"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.target, nil)
			request.Header.Set("Authorization", "Bearer "+test.token)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.wantStatus, response.Body)
			}
			if test.wantCode != "" && !strings.Contains(response.Body.String(), `"code":"`+test.wantCode+`"`) {
				t.Fatalf("body = %s, want code %s", response.Body, test.wantCode)
			}
			if test.wantStatus != http.StatusOK {
				return
			}
			var envelope struct {
				RequestID string             `json:"request_id"`
				Result    v1.EventTagCatalog `json:"result"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.RequestID != "catalog-request" ||
				envelope.Result.CatalogHash != catalogHash ||
				envelope.Result.CatalogRevision != "event-tags:"+catalogHash ||
				len(envelope.Result.Tags) != 1 ||
				!envelope.Result.Tags[0].IsActive {
				t.Fatalf("catalog envelope = %#v", envelope)
			}
		})
	}
}
