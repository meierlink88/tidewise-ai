package adminapi

import (
	"crypto/subtle"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

type schedulerRepository interface {
	repositories.SchedulerRepository
}

type schedulerSourceFilterRequest struct {
	ProviderKey   string `json:"provider_key"`
	IngestChannel string `json:"ingest_channel"`
	SourceType    string `json:"source_type"`
}

type schedulerConfigRequest struct {
	Enabled         bool                         `json:"enabled"`
	Mode            string                       `json:"mode"`
	IntervalMinutes int                          `json:"interval_minutes"`
	FixedTimes      []string                     `json:"fixed_times"`
	Concurrency     int                          `json:"concurrency"`
	BatchSize       int                          `json:"batch_size"`
	TimeoutSeconds  int                          `json:"timeout_seconds"`
	SourceFilter    schedulerSourceFilterRequest `json:"source_filter"`
	Timezone        string                       `json:"timezone"`
}

type schedulerConfigResponse struct {
	ID              string                       `json:"id"`
	Enabled         bool                         `json:"enabled"`
	Mode            string                       `json:"mode"`
	IntervalMinutes int                          `json:"interval_minutes"`
	FixedTimes      []string                     `json:"fixed_times"`
	Concurrency     int                          `json:"concurrency"`
	BatchSize       int                          `json:"batch_size"`
	TimeoutSeconds  int                          `json:"timeout_seconds"`
	SourceFilter    schedulerSourceFilterRequest `json:"source_filter"`
	Timezone        string                       `json:"timezone"`
	ConfigVersion   int                          `json:"config_version"`
	RecentRun       *schedulerRunResponse        `json:"recent_run,omitempty"`
}

type schedulerRunResponse struct {
	ID               string `json:"id"`
	TriggerType      string `json:"trigger_type"`
	Status           string `json:"status"`
	StartedAt        string `json:"started_at"`
	FinishedAt       string `json:"finished_at,omitempty"`
	TotalSources     int    `json:"total_sources"`
	SucceededSources int    `json:"succeeded_sources"`
	FailedSources    int    `json:"failed_sources"`
	SkippedSources   int    `json:"skipped_sources"`
	ErrorSummary     string `json:"error_summary"`
}

func NewRouter(repository schedulerRepository, adminToken string) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	admin := router.Group("/admin")
	admin.Use(adminTokenMiddleware(adminToken))
	admin.GET("/scheduler/config", getSchedulerConfig(repository))
	admin.PUT("/scheduler/config", updateSchedulerConfig(repository))
	admin.GET("/scheduler/runs", listSchedulerRuns(repository))
	return router
}

func adminTokenMiddleware(adminToken string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if adminToken == "" {
			ctx.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "admin token is not configured"})
			return
		}
		header := ctx.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		value := strings.TrimPrefix(header, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(value), []byte(adminToken)) != 1 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		ctx.Next()
	}
}

func getSchedulerConfig(repository schedulerRepository) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		config, err := repository.LoadSchedulerConfig(ctx.Request.Context())
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		response := schedulerConfigDTO(config)
		runs, err := repository.RecentIngestionRuns(ctx.Request.Context(), 1)
		if err == nil && len(runs) > 0 {
			recent := schedulerRunDTO(runs[0])
			response.RecentRun = &recent
		}
		ctx.JSON(http.StatusOK, response)
	}
}

func updateSchedulerConfig(repository schedulerRepository) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var request schedulerConfigRequest
		if err := ctx.ShouldBindJSON(&request); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		config := schedulerConfigFromRequest(request)
		if err := config.Validate(); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		saved, err := repository.SaveSchedulerConfig(ctx.Request.Context(), config)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, schedulerConfigDTO(saved))
	}
}

func listSchedulerRuns(repository schedulerRepository) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		limit, err := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
		if err != nil || limit <= 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "limit must be positive"})
			return
		}
		runs, err := repository.RecentIngestionRuns(ctx.Request.Context(), limit)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		response := make([]schedulerRunResponse, 0, len(runs))
		for _, run := range runs {
			response = append(response, schedulerRunDTO(run))
		}
		ctx.JSON(http.StatusOK, response)
	}
}

func schedulerConfigFromRequest(request schedulerConfigRequest) domain.SchedulerConfig {
	return domain.SchedulerConfig{
		ID:              "default",
		Enabled:         request.Enabled,
		Mode:            domain.SchedulerMode(request.Mode),
		IntervalMinutes: request.IntervalMinutes,
		FixedTimes:      append([]string(nil), request.FixedTimes...),
		Concurrency:     request.Concurrency,
		BatchSize:       request.BatchSize,
		TimeoutSeconds:  request.TimeoutSeconds,
		SourceFilter: domain.SchedulerSourceFilter{
			ProviderKey:   request.SourceFilter.ProviderKey,
			IngestChannel: request.SourceFilter.IngestChannel,
			SourceType:    request.SourceFilter.SourceType,
		},
		Timezone: request.Timezone,
	}
}

func schedulerConfigDTO(config domain.SchedulerConfig) schedulerConfigResponse {
	return schedulerConfigResponse{
		ID:              config.ID,
		Enabled:         config.Enabled,
		Mode:            string(config.Mode),
		IntervalMinutes: config.IntervalMinutes,
		FixedTimes:      append([]string(nil), config.FixedTimes...),
		Concurrency:     config.Concurrency,
		BatchSize:       config.BatchSize,
		TimeoutSeconds:  config.TimeoutSeconds,
		SourceFilter: schedulerSourceFilterRequest{
			ProviderKey:   config.SourceFilter.ProviderKey,
			IngestChannel: config.SourceFilter.IngestChannel,
			SourceType:    config.SourceFilter.SourceType,
		},
		Timezone:      config.Timezone,
		ConfigVersion: config.ConfigVersion,
	}
}

func schedulerRunDTO(run domain.IngestionRun) schedulerRunResponse {
	response := schedulerRunResponse{
		ID:               run.ID,
		TriggerType:      string(run.TriggerType),
		Status:           string(run.Status),
		StartedAt:        run.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		TotalSources:     run.TotalSources,
		SucceededSources: run.SucceededSources,
		FailedSources:    run.FailedSources,
		SkippedSources:   run.SkippedSources,
		ErrorSummary:     run.ErrorSummary,
	}
	if run.FinishedAt != nil {
		response.FinishedAt = run.FinishedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return response
}
