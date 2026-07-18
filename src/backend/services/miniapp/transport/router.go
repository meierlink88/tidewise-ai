package transport

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/runtimeconfig"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/usecase"
)

type HealthResponse struct {
	Status      string                    `json:"status"`
	Service     string                    `json:"service"`
	Environment runtimeconfig.Environment `json:"environment"`
}

type ReadyResponse struct {
	Status      string                    `json:"status"`
	Service     string                    `json:"service"`
	Environment runtimeconfig.Environment `json:"environment"`
	Checks      map[string]string         `json:"checks"`
}

func NewRouter(app runtimeconfig.AppConfig, researchServices ...*usecase.ResearchService) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/healthz", healthHandler(app))
	router.GET("/readyz", readyHandler(app))

	registerV1Routes(router.Group("/api/v1"), firstResearchService(researchServices))

	return router
}

func registerV1Routes(group *gin.RouterGroup, researchService *usecase.ResearchService) {
	if researchService != nil {
		RegisterResearchRoutes(group, researchService)
	}
}

func firstResearchService(services []*usecase.ResearchService) *usecase.ResearchService {
	if len(services) == 0 {
		return nil
	}
	return services[0]
}

func healthHandler(app runtimeconfig.AppConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, HealthResponse{
			Status:      "ok",
			Service:     app.Name,
			Environment: app.Env,
		})
	}
}

func readyHandler(app runtimeconfig.AppConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, ReadyResponse{
			Status:      "ready",
			Service:     app.Name,
			Environment: app.Env,
			Checks: map[string]string{
				"config": "ok",
			},
		})
	}
}
