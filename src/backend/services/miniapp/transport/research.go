package transport

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/apihttp"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/dataclient"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/usecase"
)

func RegisterResearchRoutes(group *gin.RouterGroup, service *usecase.ResearchService) {
	group.Use(requestIDMiddleware())
	research := group.Group("/research")
	research.GET("/themes", func(ctx *gin.Context) { handleListThemes(ctx, service) })
	research.GET("/themes/:theme_id/reasoning-trees", func(ctx *gin.Context) { handleListReasoningTrees(ctx, service) })
	research.GET("/themes/:theme_id/reasoning-trees/:anchor_id", func(ctx *gin.Context) { handleGetReasoningTree(ctx, service) })
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

func handleListReasoningTrees(ctx *gin.Context, service *usecase.ResearchService) {
	if ctx.Request.URL.RawQuery != "" {
		writeReasoningTreeResponse(ctx, nil, usecase.ErrInvalidResearchRequest)
		return
	}
	response, err := service.ListReasoningTrees(dataRequestContext(ctx), ctx.Param("theme_id"))
	writeReasoningTreeResponse(ctx, response, err)
}

func handleGetReasoningTree(ctx *gin.Context, service *usecase.ResearchService) {
	if ctx.Request.URL.RawQuery != "" {
		writeReasoningTreeResponse(ctx, nil, usecase.ErrInvalidResearchRequest)
		return
	}
	response, err := service.GetReasoningTree(dataRequestContext(ctx), ctx.Param("theme_id"), ctx.Param("anchor_id"))
	writeReasoningTreeResponse(ctx, response, err)
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
		writeSuccess(ctx, response)
		return
	}
	status := http.StatusInternalServerError
	code := "RESEARCH_DATA_UNAVAILABLE"
	message := usecase.ErrResearchDataService.Error()
	if errors.Is(err, usecase.ErrInvalidResearchRequest) {
		status = http.StatusBadRequest
		code = "INVALID_REQUEST"
		message = err.Error()
	} else if errors.Is(err, usecase.ErrResearchNotFound) {
		status = http.StatusNotFound
		code = "RESEARCH_RESULT_NOT_FOUND"
		message = usecase.ErrResearchNotFound.Error()
	}
	writeAPIError(ctx, status, code, message)
}

func writeReasoningTreeResponse(ctx *gin.Context, response any, err error) {
	if err == nil {
		writeSuccess(ctx, response)
		return
	}
	status := http.StatusBadGateway
	code := "RESEARCH_DATA_UNAVAILABLE"
	message := "research data is temporarily unavailable"
	switch {
	case errors.Is(err, usecase.ErrInvalidResearchRequest):
		status = http.StatusBadRequest
		code = "INVALID_REQUEST"
		message = "invalid research request"
	case errors.Is(err, usecase.ErrResearchThemeNotFound):
		status = http.StatusNotFound
		code = "RESEARCH_THEME_NOT_FOUND"
		message = "research Theme was not found"
	case errors.Is(err, usecase.ErrResearchReasoningTreesNotFound):
		status = http.StatusNotFound
		code = "RESEARCH_REASONING_TREES_NOT_FOUND"
		message = "research Theme has no published reasoning trees"
	case errors.Is(err, usecase.ErrResearchReasoningTreeNotFound):
		status = http.StatusNotFound
		code = "RESEARCH_REASONING_TREE_NOT_FOUND"
		message = "research reasoning tree was not found for the Theme"
	}
	writeAPIError(ctx, status, code, message)
}

func writeSuccess(ctx *gin.Context, result any) {
	requestID := ctx.GetHeader(apihttp.RequestIDHeader)
	ctx.JSON(http.StatusOK, apihttp.Success(requestID, result))
}

func writeAPIError(ctx *gin.Context, status int, code, message string) {
	requestID := ctx.GetHeader(apihttp.RequestIDHeader)
	ctx.AbortWithStatusJSON(status, apihttp.Error(requestID, code, message, map[string]any{}))
}
