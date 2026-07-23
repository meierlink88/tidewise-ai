package transport

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/apihttp"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/runtimeconfig"
	"github.com/meierlink88/tidewise-ai/backend/services/adminportal/dataclient"
	"github.com/meierlink88/tidewise-ai/backend/services/adminportal/usecase"
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
	Status      string                    `json:"status"`
	Service     string                    `json:"service"`
	Environment runtimeconfig.Environment `json:"environment"`
	Checks      map[string]string         `json:"checks,omitempty"`
}

func NewRouter(app runtimeconfig.AppConfig, service *usecase.Service, adminToken string, allowedOrigins ...string) *gin.Engine {
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
	return router
}

func apiRecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(ctx *gin.Context, _ any) {
		requestID := apihttp.ResolveRequestID(ctx.GetHeader(apihttp.RequestIDHeader), "admin")
		ctx.Request.Header.Set(apihttp.RequestIDHeader, requestID)
		ctx.Header(apihttp.RequestIDHeader, requestID)
		writeAPIError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	})
}

func requestIDMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestID := apihttp.ResolveRequestID(ctx.GetHeader(apihttp.RequestIDHeader), "admin")
		ctx.Request.Header.Set(apihttp.RequestIDHeader, requestID)
		ctx.Header(apihttp.RequestIDHeader, requestID)
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
		ctx.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
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

func healthHandler(app runtimeconfig.AppConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, healthResponse{Status: "ok", Service: app.Name, Environment: app.Env})
	}
}

func readyHandler(app runtimeconfig.AppConfig) gin.HandlerFunc {
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

func writeInternalError(ctx *gin.Context) {
	writeAPIError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}

func writeSuccess(ctx *gin.Context, result any) {
	requestID := ctx.GetHeader(apihttp.RequestIDHeader)
	ctx.JSON(http.StatusOK, apihttp.Success(requestID, result))
}

func writeAPIError(ctx *gin.Context, status int, code, message string) {
	requestID := ctx.GetHeader(apihttp.RequestIDHeader)
	ctx.AbortWithStatusJSON(status, apihttp.Error(requestID, code, message, map[string]any{}))
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
