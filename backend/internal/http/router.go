package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/meierlink88/tidewise-ai/backend/internal/apps/miniappapi"
	"github.com/meierlink88/tidewise-ai/backend/internal/config"
)

type HealthResponse struct {
	Status      string             `json:"status"`
	Service     string             `json:"service"`
	Environment config.Environment `json:"environment"`
}

type ReadyResponse struct {
	Status      string             `json:"status"`
	Service     string             `json:"service"`
	Environment config.Environment `json:"environment"`
	Checks      map[string]string  `json:"checks"`
}

func NewRouter(cfg config.Config, researchServices ...*miniappapi.ResearchService) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/healthz", healthHandler(cfg))
	router.GET("/readyz", readyHandler(cfg))

	registerV1Routes(router.Group("/api/v1"), firstResearchService(researchServices))

	return router
}

func registerV1Routes(group *gin.RouterGroup, researchService *miniappapi.ResearchService) {
	if researchService != nil {
		miniappapi.RegisterResearchRoutes(group, researchService)
	}
}

func firstResearchService(services []*miniappapi.ResearchService) *miniappapi.ResearchService {
	if len(services) == 0 {
		return nil
	}
	return services[0]
}

func healthHandler(cfg config.Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, HealthResponse{
			Status:      "ok",
			Service:     cfg.App.Name,
			Environment: cfg.App.Env,
		})
	}
}

func readyHandler(cfg config.Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, ReadyResponse{
			Status:      "ready",
			Service:     cfg.App.Name,
			Environment: cfg.App.Env,
			Checks: map[string]string{
				"config": "ok",
			},
		})
	}
}
