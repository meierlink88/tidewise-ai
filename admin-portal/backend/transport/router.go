package transport

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/agentrunclient"
	adminconfig "github.com/meierlink88/tidewise-ai/admin-portal/backend/config"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/dataclient"
	"github.com/meierlink88/tidewise-ai/admin-portal/backend/usecase"
)

type rawDocumentListResponse struct {
	Items    []rawDocumentResponse `json:"items"`
	Total    int                   `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}

type rawDocumentResponse struct {
	ID               string `json:"id"`
	ContractVersion  int    `json:"contract_version"`
	ArtifactID       string `json:"artifact_id,omitempty"`
	SourceRef        string `json:"source_ref,omitempty"`
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
	ContentSHA256    string `json:"content_sha256"`
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

type healthResponse struct {
	Status      string                  `json:"status"`
	Service     string                  `json:"service"`
	Environment adminconfig.Environment `json:"environment"`
	Checks      map[string]string       `json:"checks,omitempty"`
}

func NewRouter(app adminconfig.AppConfig, service *usecase.Service, adminToken string, allowedOrigins ...string) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(apiRecoveryMiddleware())

	router.GET("/healthz", healthHandler(app))
	router.GET("/readyz", readyHandler(app))

	admin := router.Group("/api/admin/v1")
	admin.Use(requestIDMiddleware())
	admin.Use(adminCORSMiddleware(firstAllowedOrigin(allowedOrigins)))
	admin.Use(adminTokenMiddleware(adminToken))
	admin.OPTIONS("/*path", func(*gin.Context) {})
	admin.GET("/raw-documents", listRawDocuments(service))
	admin.GET("/events", listEvents(service))
	admin.GET("/agent-schedules/:agent_key", getAgentSchedule(service))
	admin.PUT("/agent-schedules/:agent_key", saveAgentSchedule(service))
	admin.PATCH("/agent-schedules/:agent_key", setAgentScheduleEnabled(service))
	admin.GET("/agent-executions", listCollectorExecutions(service))
	admin.GET("/model-providers", listModelProviders(service))
	admin.GET("/model-providers/:provider_key", getModelProvider(service))
	admin.PATCH("/model-providers/:provider_key", patchModelProvider(service))
	admin.GET("/connectors", listConnectors(service))
	admin.GET("/connectors/:connector_key", getConnector(service))
	admin.PATCH("/connectors/:connector_key", patchConnector(service))
	return router
}

func getAgentSchedule(service *usecase.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		schedule, err := service.GetAgentSchedule(agentRunRequestContext(ctx), ctx.Param("agent_key"))
		if err != nil {
			writeAgentRunError(ctx, err)
			return
		}
		writeSuccess(ctx, schedule)
	}
}

func saveAgentSchedule(service *usecase.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var input struct {
			AgentVersion   string                      `json:"agent_version"`
			ScheduleType   agentrunclient.ScheduleType `json:"schedule_type"`
			CronExpression string                      `json:"cron_expression"`
			DailyTimes     []string                    `json:"daily_times"`
			Input          json.RawMessage             `json:"input"`
		}
		if err := decodeStrictJSON(ctx, &input); err != nil {
			writeAPIError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "request body is invalid")
			return
		}
		schedule, err := service.SaveAgentSchedule(agentRunRequestContext(ctx), ctx.Param("agent_key"), usecase.SaveAgentScheduleInput{
			AgentVersion: input.AgentVersion, ScheduleType: input.ScheduleType,
			CronExpression: input.CronExpression, DailyTimes: input.DailyTimes, Input: input.Input,
		})
		if err != nil {
			writeAgentRunError(ctx, err)
			return
		}
		writeSuccess(ctx, schedule)
	}
}

func setAgentScheduleEnabled(service *usecase.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var input struct {
			Enabled *bool `json:"enabled"`
		}
		if err := decodeStrictJSON(ctx, &input); err != nil || input.Enabled == nil {
			writeAPIError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "enabled is required")
			return
		}
		schedule, err := service.SetAgentScheduleEnabled(agentRunRequestContext(ctx), ctx.Param("agent_key"), *input.Enabled)
		if err != nil {
			writeAgentRunError(ctx, err)
			return
		}
		writeSuccess(ctx, schedule)
	}
}

func listCollectorExecutions(service *usecase.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
		if err != nil || page <= 0 {
			writeAPIError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "page must be positive")
			return
		}
		result, err := service.ListCollectorExecutions(agentRunRequestContext(ctx), page)
		if err != nil {
			writeAgentRunError(ctx, err)
			return
		}
		writeSuccess(ctx, result)
	}
}

func listModelProviders(service *usecase.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		items, err := service.ListModelProviders(agentRunRequestContext(ctx))
		if err != nil {
			writeAgentRunError(ctx, err)
			return
		}
		writeSuccess(ctx, map[string]any{"items": items})
	}
}

func getModelProvider(service *usecase.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		result, err := service.GetModelProvider(agentRunRequestContext(ctx), ctx.Param("provider_key"))
		if err != nil {
			writeAgentRunError(ctx, err)
			return
		}
		writeSuccess(ctx, result)
	}
}

func patchModelProvider(service *usecase.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var input struct {
			BaseURL *string `json:"base_url"`
			Model   *string `json:"model"`
			APIKey  *string `json:"api_key"`
		}
		if err := decodeStrictJSON(ctx, &input); err != nil || (input.BaseURL == nil && input.Model == nil && input.APIKey == nil) {
			writeAPIError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "request body is invalid")
			return
		}
		result, err := service.PatchModelProvider(agentRunRequestContext(ctx), ctx.Param("provider_key"), agentrunclient.ModelProviderPatch{
			BaseURL: input.BaseURL, Model: input.Model, APIKey: input.APIKey,
		})
		if err != nil {
			writeAgentRunError(ctx, err)
			return
		}
		writeSuccess(ctx, result)
	}
}

func listConnectors(service *usecase.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		items, err := service.ListConnectors(agentRunRequestContext(ctx))
		if err != nil {
			writeAgentRunError(ctx, err)
			return
		}
		writeSuccess(ctx, map[string]any{"items": items})
	}
}

func getConnector(service *usecase.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		result, err := service.GetConnector(agentRunRequestContext(ctx), ctx.Param("connector_key"))
		if err != nil {
			writeAgentRunError(ctx, err)
			return
		}
		writeSuccess(ctx, result)
	}
}

func patchConnector(service *usecase.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var input struct {
			BaseURL *string `json:"base_url"`
			APIKey  *string `json:"api_key"`
		}
		if err := decodeStrictJSON(ctx, &input); err != nil || (input.BaseURL == nil && input.APIKey == nil) {
			writeAPIError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "request body is invalid")
			return
		}
		result, err := service.PatchConnector(agentRunRequestContext(ctx), ctx.Param("connector_key"), agentrunclient.ConnectorPatch{
			BaseURL: input.BaseURL, APIKey: input.APIKey,
		})
		if err != nil {
			writeAgentRunError(ctx, err)
			return
		}
		writeSuccess(ctx, result)
	}
}

func apiRecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(ctx *gin.Context, _ any) {
		requestID := resolveRequestID(ctx.GetHeader(requestIDHeader), "admin")
		ctx.Request.Header.Set(requestIDHeader, requestID)
		ctx.Header(requestIDHeader, requestID)
		writeAPIError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	})
}

func requestIDMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestID := resolveRequestID(ctx.GetHeader(requestIDHeader), "admin")
		ctx.Request.Header.Set(requestIDHeader, requestID)
		ctx.Header(requestIDHeader, requestID)
		ctx.Next()
	}
}

func firstAllowedOrigin(origins []string) string {
	if len(origins) == 0 {
		return ""
	}
	return strings.TrimSpace(origins[0])
}

func adminCORSMiddleware(allowedOrigin string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		origin := strings.TrimSpace(ctx.GetHeader("Origin"))
		if origin == "" {
			ctx.Next()
			return
		}
		if allowedOrigin == "" || origin != allowedOrigin {
			writeAPIError(ctx, http.StatusForbidden, "FORBIDDEN", "origin is not allowed")
			return
		}
		ctx.Header("Access-Control-Allow-Origin", allowedOrigin)
		ctx.Header("Access-Control-Allow-Methods", "GET, PUT, PATCH, OPTIONS")
		ctx.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
		ctx.Header("Access-Control-Max-Age", "600")
		ctx.Header("Vary", "Origin")
		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	}
}

func healthHandler(app adminconfig.AppConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, healthResponse{Status: "ok", Service: app.Name, Environment: app.Env})
	}
}

func readyHandler(app adminconfig.AppConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, healthResponse{
			Status: "ready", Service: app.Name, Environment: app.Env,
			Checks: map[string]string{"config": "ok"},
		})
	}
}

func listRawDocuments(service *usecase.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		query, err := rawDocumentListQueryFromRequest(ctx)
		if err != nil {
			writeAPIError(ctx, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
			return
		}
		page, err := service.ListRawDocuments(dataRequestContext(ctx), query)
		if err != nil {
			writeInternalError(ctx)
			return
		}
		items := make([]rawDocumentResponse, 0, len(page.Items))
		for _, document := range page.Items {
			items = append(items, rawDocumentDTO(document))
		}
		writeSuccess(ctx, rawDocumentListResponse{Items: items, Total: page.Total, Page: page.Page, PageSize: page.PageSize})
	}
}

func listEvents(service *usecase.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		query, err := eventListQueryFromRequest(ctx)
		if err != nil {
			writeAPIError(ctx, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
			return
		}
		page, err := service.ListEvents(dataRequestContext(ctx), query)
		if err != nil {
			writeInternalError(ctx)
			return
		}
		items := make([]eventResponse, 0, len(page.Items))
		for _, event := range page.Items {
			items = append(items, eventDTO(event))
		}
		writeSuccess(ctx, eventListResponse{Items: items, Total: page.Total, Page: page.Page, PageSize: page.PageSize})
	}
}

func adminTokenMiddleware(adminToken string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if adminToken == "" {
			writeAPIError(ctx, http.StatusServiceUnavailable, "ADMIN_NOT_CONFIGURED", "admin token is not configured")
			return
		}
		header := ctx.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeAPIError(ctx, http.StatusUnauthorized, "UNAUTHENTICATED", "valid admin identity is required")
			return
		}
		value := strings.TrimPrefix(header, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(value), []byte(adminToken)) != 1 {
			writeAPIError(ctx, http.StatusUnauthorized, "UNAUTHENTICATED", "valid admin identity is required")
			return
		}
		ctx.Next()
	}
}

func dataRequestContext(ctx *gin.Context) context.Context {
	return dataclient.WithRequestID(ctx.Request.Context(), ctx.GetHeader(dataclient.RequestIDHeader))
}

func agentRunRequestContext(ctx *gin.Context) context.Context {
	return agentrunclient.WithRequestID(ctx.Request.Context(), ctx.GetHeader(agentrunclient.RequestIDHeader))
}

func writeAgentRunError(ctx *gin.Context, err error) {
	var clientError *agentrunclient.Error
	if !errors.As(err, &clientError) {
		writeAPIError(ctx, http.StatusServiceUnavailable, "AGENTRUN_UNAVAILABLE", "AgentRun is unavailable")
		return
	}
	switch clientError.StatusCode {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		writeAPIError(ctx, http.StatusBadRequest, "AGENTRUN_INVALID_REQUEST", "AgentRun rejected the request")
	case http.StatusUnauthorized, http.StatusForbidden:
		writeAPIError(ctx, http.StatusServiceUnavailable, "AGENTRUN_AUTHENTICATION_FAILED", "AgentRun authentication is unavailable")
	case http.StatusNotFound:
		writeAPIError(ctx, http.StatusNotFound, "AGENTRUN_NOT_FOUND", "AgentRun resource was not found")
	case http.StatusConflict:
		writeAPIError(ctx, http.StatusConflict, "AGENTRUN_CONFLICT", "AgentRun rejected the conflicting request")
	default:
		writeAPIError(ctx, http.StatusServiceUnavailable, "AGENTRUN_UNAVAILABLE", "AgentRun is unavailable")
	}
}

func writeInternalError(ctx *gin.Context) {
	writeAPIError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}

func writeSuccess(ctx *gin.Context, result any) {
	requestID := ctx.GetHeader(requestIDHeader)
	ctx.JSON(http.StatusOK, successResponse(requestID, result))
}

func writeAPIError(ctx *gin.Context, status int, code, message string) {
	requestID := ctx.GetHeader(requestIDHeader)
	ctx.AbortWithStatusJSON(status, errorResponse(requestID, code, message, map[string]any{}))
}

func decodeStrictJSON(ctx *gin.Context, target any) error {
	body, err := io.ReadAll(io.LimitReader(ctx.Request.Body, 128*1024+1))
	if err != nil || len(body) == 0 || len(body) > 128*1024 {
		return errors.New("invalid request body")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("trailing request content")
	}
	return nil
}

func rawDocumentListQueryFromRequest(ctx *gin.Context) (dataclient.RawDocumentListQuery, error) {
	page, pageSize, err := pageFromQuery(ctx)
	if err != nil {
		return dataclient.RawDocumentListQuery{}, err
	}
	return dataclient.RawDocumentListQuery{
		Title: ctx.Query("title"), SourceRef: ctx.Query("source_ref"), Page: page, PageSize: pageSize,
	}, nil
}

func eventListQueryFromRequest(ctx *gin.Context) (dataclient.EventListQuery, error) {
	page, pageSize, err := pageFromQuery(ctx)
	if err != nil {
		return dataclient.EventListQuery{}, err
	}
	eventTimeFrom, err := parseOptionalTime(ctx.Query("event_time_from"))
	if err != nil {
		return dataclient.EventListQuery{}, err
	}
	eventTimeTo, err := parseOptionalTime(ctx.Query("event_time_to"))
	if err != nil {
		return dataclient.EventListQuery{}, err
	}
	firstSeenFrom, err := parseOptionalTime(ctx.Query("first_seen_from"))
	if err != nil {
		return dataclient.EventListQuery{}, err
	}
	firstSeenTo, err := parseOptionalTime(ctx.Query("first_seen_to"))
	if err != nil {
		return dataclient.EventListQuery{}, err
	}
	eventStatus := dataclient.EventStatus(ctx.Query("event_status"))
	if eventStatus != "" && eventStatus != dataclient.EventStatusCandidate && eventStatus != dataclient.EventStatusConfirmed && eventStatus != dataclient.EventStatusRejected {
		return dataclient.EventListQuery{}, errBadRequest("unsupported event status")
	}
	factStatus := dataclient.FactStatus(ctx.Query("fact_status"))
	if factStatus != "" && factStatus != dataclient.FactStatusUnverified && factStatus != dataclient.FactStatusVerified && factStatus != dataclient.FactStatusDisputed {
		return dataclient.EventListQuery{}, errBadRequest("unsupported fact status")
	}
	return dataclient.EventListQuery{
		Title: ctx.Query("title"), EventStatus: eventStatus, FactStatus: factStatus,
		EventTimeFrom: eventTimeFrom, EventTimeTo: eventTimeTo, FirstSeenFrom: firstSeenFrom, FirstSeenTo: firstSeenTo,
		Page: page, PageSize: pageSize,
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

func (e errBadRequest) Error() string { return string(e) }

func rawDocumentDTO(document dataclient.RawDocument) rawDocumentResponse {
	response := rawDocumentResponse{
		ID: document.ID, ContractVersion: document.ContractVersion, ArtifactID: document.ArtifactID,
		SourceRef: document.SourceRef, IngestChannel: document.IngestChannel, SourceType: document.SourceType,
		SourceName: document.SourceName, SourceURL: document.SourceURL, SourceExternalID: document.SourceExternalID,
		Title: document.Title, ContentText: document.ContentText, RawObjectURI: document.RawObjectURI,
		RawMIMEType: document.RawMIMEType, Language: document.Language, CollectedAt: document.CollectedAt.Format(time.RFC3339),
		IngestStatus: string(document.IngestStatus), ContentSHA256: document.ContentSHA256,
	}
	if document.PublishedAt != nil {
		response.PublishedAt = document.PublishedAt.Format(time.RFC3339)
	}
	return response
}

func eventDTO(event dataclient.Event) eventResponse {
	response := eventResponse{
		ID: event.ID, Title: event.Title, Summary: event.Summary, FirstSeenAt: event.FirstSeenAt.Format(time.RFC3339),
		EventStatus: string(event.EventStatus), FactStatus: string(event.FactStatus), DedupeKey: event.DedupeKey,
	}
	if event.EventTime != nil {
		response.EventTime = event.EventTime.Format(time.RFC3339)
	}
	if event.KnowableAt != nil {
		response.KnowableAt = event.KnowableAt.Format(time.RFC3339)
	}
	if event.PrimarySourceID != nil {
		response.PrimarySourceID = *event.PrimarySourceID
	}
	return response
}
