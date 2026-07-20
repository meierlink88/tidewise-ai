package transport

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/dataclient"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/usecase"
)

func RegisterResearchRoutes(group *gin.RouterGroup, service *usecase.ResearchService) {
	research := group.Group("/miniapp/research")
	research.GET("/themes", func(ctx *gin.Context) { handleListThemes(ctx, service) })
	research.GET("/themes/:theme_id", func(ctx *gin.Context) { handleGetTheme(ctx, service) })
}

func handleListThemes(ctx *gin.Context, service *usecase.ResearchService) {
	response, err := service.ListThemes(dataRequestContext(ctx), parseResearchListRequest(ctx))
	writeResearchResponse(ctx, response, err)
}

func handleGetTheme(ctx *gin.Context, service *usecase.ResearchService) {
	response, err := service.GetTheme(dataRequestContext(ctx), ctx.Param("theme_id"), parseResearchDetailRequest(ctx))
	writeResearchResponse(ctx, response, err)
}

func dataRequestContext(ctx *gin.Context) context.Context {
	return dataclient.WithRequestID(ctx.Request.Context(), ctx.GetHeader(dataclient.RequestIDHeader))
}

func parseResearchListRequest(ctx *gin.Context) usecase.ResearchListRequest {
	return usecase.ResearchListRequest{
		WindowHours: parseIntQuery(ctx, "window_hours"),
		Limit:       parseIntQuery(ctx, "limit"),
		Cursor:      ctx.Query("cursor"),
	}
}

func parseResearchDetailRequest(ctx *gin.Context) usecase.ResearchDetailRequest {
	return usecase.ResearchDetailRequest{WindowHours: parseIntQuery(ctx, "window_hours")}
}

func parseIntQuery(ctx *gin.Context, key string) int {
	value := strings.TrimSpace(ctx.Query(key))
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return -1
	}
	return parsed
}

func writeResearchResponse(ctx *gin.Context, response any, err error) {
	if err == nil {
		ctx.JSON(http.StatusOK, response)
		return
	}
	status := http.StatusInternalServerError
	message := usecase.ErrResearchDataService.Error()
	if errors.Is(err, usecase.ErrInvalidResearchRequest) {
		status = http.StatusBadRequest
		message = err.Error()
	} else if errors.Is(err, usecase.ErrResearchNotFound) {
		status = http.StatusNotFound
		message = usecase.ErrResearchNotFound.Error()
	}
	ctx.AbortWithStatusJSON(status, gin.H{"error": message})
}
