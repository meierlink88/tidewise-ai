package adminapi

import (
	"crypto/subtle"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

type schedulerRepository interface {
	repositories.SchedulerRepository
	repositories.AdminQueryRepository
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

type rawDocumentListResponse struct {
	Items    []rawDocumentResponse `json:"items"`
	Total    int                   `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}

type rawDocumentResponse struct {
	ID               string `json:"id"`
	SourceID         string `json:"source_id"`
	IngestChannel    string `json:"ingest_channel"`
	SourceType       string `json:"source_type"`
	SourceName       string `json:"source_name"`
	SourceURL        string `json:"source_url"`
	SourceExternalID string `json:"source_external_id,omitempty"`
	Title            string `json:"title"`
	ContentText      string `json:"content_text"`
	RawObjectURI     string `json:"raw_object_uri"`
	RawMIMEType      string `json:"raw_mime_type"`
	Language         string `json:"language"`
	PublishedAt      string `json:"published_at,omitempty"`
	CollectedAt      string `json:"collected_at"`
	IngestStatus     string `json:"ingest_status"`
}

type eventListResponse struct {
	Items    []eventResponse `json:"items"`
	Total    int             `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

type eventResponse struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Summary         string `json:"summary"`
	EventTime       string `json:"event_time,omitempty"`
	FirstSeenAt     string `json:"first_seen_at"`
	KnowableAt      string `json:"knowable_at,omitempty"`
	EventStatus     string `json:"event_status"`
	FactStatus      string `json:"fact_status"`
	DedupeKey       string `json:"dedupe_key"`
	PrimarySourceID string `json:"primary_source_id,omitempty"`
}

type sourceCatalogListResponse struct {
	Items []sourceCatalogResponse `json:"items"`
}

type sourceCatalogResponse struct {
	ID            string `json:"id"`
	IngestChannel string `json:"ingest_channel"`
	ProviderKey   string `json:"provider_key"`
	ConnectorKey  string `json:"connector_key"`
	SourceType    string `json:"source_type"`
	SourceName    string `json:"source_name"`
	SourceURL     string `json:"source_url"`
	SourceLevel   string `json:"source_level"`
	TopicHint     string `json:"topic_hint"`
	UsagePolicy   string `json:"usage_policy"`
	Status        string `json:"status"`
}

func NewRouter(repository schedulerRepository, adminToken string) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	admin := router.Group("/admin")
	admin.Use(adminTokenMiddleware(adminToken))
	admin.GET("/raw-documents", listRawDocuments(repository))
	admin.GET("/events", listEvents(repository))
	admin.GET("/source-catalogs", listSourceCatalogs(repository))
	admin.GET("/scheduler/config", getSchedulerConfig(repository))
	admin.PUT("/scheduler/config", updateSchedulerConfig(repository))
	admin.GET("/scheduler/runs", listSchedulerRuns(repository))
	return router
}

func listRawDocuments(repository schedulerRepository) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		filter, err := rawDocumentListFilterFromQuery(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		page, err := repository.ListRawDocuments(ctx.Request.Context(), filter)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items := make([]rawDocumentResponse, 0, len(page.Items))
		for _, doc := range page.Items {
			items = append(items, rawDocumentDTO(doc))
		}
		ctx.JSON(http.StatusOK, rawDocumentListResponse{
			Items:    items,
			Total:    page.Total,
			Page:     page.Page,
			PageSize: page.PageSize,
		})
	}
}

func listEvents(repository schedulerRepository) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		filter, err := eventListFilterFromQuery(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		page, err := repository.ListEvents(ctx.Request.Context(), filter)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items := make([]eventResponse, 0, len(page.Items))
		for _, event := range page.Items {
			items = append(items, eventDTO(event))
		}
		ctx.JSON(http.StatusOK, eventListResponse{
			Items:    items,
			Total:    page.Total,
			Page:     page.Page,
			PageSize: page.PageSize,
		})
	}
}

func listSourceCatalogs(repository schedulerRepository) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		status := domain.SourceCatalogStatus(ctx.Query("status"))
		if status != "" && status != domain.SourceCatalogStatusActive && status != domain.SourceCatalogStatusInactive && status != domain.SourceCatalogStatusDisabled {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "unsupported source status"})
			return
		}
		sources, err := repository.ListSourceCatalogs(ctx.Request.Context(), repositories.SourceCatalogListFilter{Status: status})
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items := make([]sourceCatalogResponse, 0, len(sources))
		for _, source := range sources {
			items = append(items, sourceCatalogDTO(source))
		}
		ctx.JSON(http.StatusOK, sourceCatalogListResponse{Items: items})
	}
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

func rawDocumentListFilterFromQuery(ctx *gin.Context) (repositories.RawDocumentListFilter, error) {
	page, pageSize, err := pageFromQuery(ctx)
	if err != nil {
		return repositories.RawDocumentListFilter{}, err
	}
	return repositories.RawDocumentListFilter{
		Title:    ctx.Query("title"),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func eventListFilterFromQuery(ctx *gin.Context) (repositories.EventListFilter, error) {
	page, pageSize, err := pageFromQuery(ctx)
	if err != nil {
		return repositories.EventListFilter{}, err
	}
	eventTimeFrom, err := parseOptionalTime(ctx.Query("event_time_from"))
	if err != nil {
		return repositories.EventListFilter{}, err
	}
	eventTimeTo, err := parseOptionalTime(ctx.Query("event_time_to"))
	if err != nil {
		return repositories.EventListFilter{}, err
	}
	firstSeenFrom, err := parseOptionalTime(ctx.Query("first_seen_from"))
	if err != nil {
		return repositories.EventListFilter{}, err
	}
	firstSeenTo, err := parseOptionalTime(ctx.Query("first_seen_to"))
	if err != nil {
		return repositories.EventListFilter{}, err
	}
	eventStatus := domain.EventStatus(ctx.Query("event_status"))
	if eventStatus != "" && eventStatus != domain.EventStatusCandidate && eventStatus != domain.EventStatusConfirmed && eventStatus != domain.EventStatusRejected {
		return repositories.EventListFilter{}, errBadRequest("unsupported event status")
	}
	factStatus := domain.FactStatus(ctx.Query("fact_status"))
	if factStatus != "" && factStatus != domain.FactStatusUnverified && factStatus != domain.FactStatusVerified && factStatus != domain.FactStatusDisputed {
		return repositories.EventListFilter{}, errBadRequest("unsupported fact status")
	}
	return repositories.EventListFilter{
		Title:         ctx.Query("title"),
		EventStatus:   eventStatus,
		FactStatus:    factStatus,
		EventTimeFrom: eventTimeFrom,
		EventTimeTo:   eventTimeTo,
		FirstSeenFrom: firstSeenFrom,
		FirstSeenTo:   firstSeenTo,
		Page:          page,
		PageSize:      pageSize,
	}, nil
}

func pageFromQuery(ctx *gin.Context) (int, int, error) {
	page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if err != nil || page <= 0 {
		return 0, 0, errBadRequest("page must be positive")
	}
	pageSize, err := strconv.Atoi(ctx.DefaultQuery("page_size", "50"))
	if err != nil || pageSize <= 0 {
		return 0, 0, errBadRequest("page_size must be positive")
	}
	return page, pageSize, nil
}

func parseOptionalTime(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, errBadRequest("time query must use RFC3339")
	}
	return &parsed, nil
}

type errBadRequest string

func (e errBadRequest) Error() string {
	return string(e)
}

func rawDocumentDTO(doc domain.RawDocument) rawDocumentResponse {
	response := rawDocumentResponse{
		ID:               doc.ID,
		SourceID:         doc.SourceID,
		IngestChannel:    doc.IngestChannel,
		SourceType:       doc.SourceType,
		SourceName:       doc.SourceName,
		SourceURL:        doc.SourceURL,
		SourceExternalID: doc.SourceExternalID,
		Title:            doc.Title,
		ContentText:      doc.ContentText,
		RawObjectURI:     doc.RawObjectURI,
		RawMIMEType:      doc.RawMIMEType,
		Language:         doc.Language,
		CollectedAt:      doc.CollectedAt.Format(time.RFC3339),
		IngestStatus:     string(doc.IngestStatus),
	}
	if doc.PublishedAt != nil {
		response.PublishedAt = doc.PublishedAt.Format(time.RFC3339)
	}
	return response
}

func eventDTO(event domain.Event) eventResponse {
	response := eventResponse{
		ID:              event.ID,
		Title:           event.Title,
		Summary:         event.Summary,
		FirstSeenAt:     event.FirstSeenAt.Format(time.RFC3339),
		EventStatus:     string(event.EventStatus),
		FactStatus:      string(event.FactStatus),
		DedupeKey:       event.DedupeKey,
		PrimarySourceID: event.PrimarySourceID,
	}
	if event.EventTime != nil {
		response.EventTime = event.EventTime.Format(time.RFC3339)
	}
	if event.KnowableAt != nil {
		response.KnowableAt = event.KnowableAt.Format(time.RFC3339)
	}
	return response
}

func sourceCatalogDTO(source domain.SourceCatalog) sourceCatalogResponse {
	return sourceCatalogResponse{
		ID:            source.ID,
		IngestChannel: source.IngestChannel,
		ProviderKey:   source.ProviderKey,
		ConnectorKey:  source.ConnectorKey,
		SourceType:    source.SourceType,
		SourceName:    source.SourceName,
		SourceURL:     source.SourceURL,
		SourceLevel:   source.SourceLevel,
		TopicHint:     source.TopicHint,
		UsagePolicy:   source.UsagePolicy,
		Status:        string(source.Status),
	}
}
