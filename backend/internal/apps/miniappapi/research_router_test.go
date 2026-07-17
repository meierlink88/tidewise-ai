package miniappapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

func TestResearchRoutesExposeReadOnlyEndpointsAndEmptyArrays(t *testing.T) {
	now := time.Date(2026, 7, 17, 1, 0, 0, 0, time.UTC)
	repo := &fakeResearchReadRepository{themePage: repositories.ResearchThemePage{AsOf: now, WindowStart: now.Add(-24 * time.Hour), WindowEnd: now, Items: []repositories.ResearchThemeSummary{}}}
	service := NewResearchService(repo, func() time.Time { return now })
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterResearchRoutes(router.Group("/api/v1"), service)

	for _, path := range []string{
		"/api/v1/miniapp/research/themes",
		"/api/v1/miniapp/research/anchors",
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200", path, response.Code)
		}
		var body map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		if body["items"] == nil {
			t.Fatalf("GET %s items = nil, want empty array", path)
		}
	}
}

func TestResearchRoutesRejectInvalidParametersAndNotFound(t *testing.T) {
	service := NewResearchService(&fakeResearchReadRepository{}, time.Now)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterResearchRoutes(router.Group("/api/v1"), service)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/miniapp/research/themes?limit=51", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid limit status = %d, want 400", response.Code)
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/miniapp/research/themes/11111111-1111-4111-8111-111111111111", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("missing theme status = %d, want 404", response.Code)
	}
}

var _ repositories.ResearchReadRepository = (*fakeResearchReadRepository)(nil)
