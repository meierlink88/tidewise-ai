package adminapi

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/meierlink88/tidewise-ai/backend/internal/config"
	"github.com/meierlink88/tidewise-ai/backend/services/adminportal/dataclient"
)

const schedulerRetiredCode = "ADMIN_SCHEDULER_RETIRED"

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

type healthResponse struct {
	Status      string             `json:"status"`
	Service     string             `json:"service"`
	Environment config.Environment `json:"environment"`
	Checks      map[string]string  `json:"checks,omitempty"`
}

func NewRouter(cfg config.Config, client dataclient.DataServiceClient, adminToken string) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/healthz", healthHandler(cfg))
	router.GET("/readyz", readyHandler(cfg))

	admin := router.Group("/admin")
	admin.Use(adminTokenMiddleware(adminToken))
	admin.GET("/raw-documents", listRawDocuments(client))
	admin.GET("/events", listEvents(client))
	admin.GET("/source-catalogs", listSourceCatalogs(client))
	admin.GET("/scheduler/config", retiredScheduler())
	admin.PUT("/scheduler/config", retiredScheduler())
	admin.GET("/scheduler/runs", retiredScheduler())
	return router
}

func healthHandler(cfg config.Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, healthResponse{Status: "ok", Service: cfg.App.Name, Environment: cfg.App.Env})
	}
}

func readyHandler(cfg config.Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, healthResponse{
			Status: "ready", Service: cfg.App.Name, Environment: cfg.App.Env,
			Checks: map[string]string{"config": "ok"},
		})
	}
}

func listRawDocuments(client dataclient.DataServiceClient) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		query, err := rawDocumentListQueryFromRequest(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if client == nil {
			writeInternalError(ctx)
			return
		}
		page, err := client.ListRawDocuments(dataRequestContext(ctx), query)
		if err != nil {
			writeInternalError(ctx)
			return
		}
		items := make([]rawDocumentResponse, 0, len(page.Items))
		for _, document := range page.Items {
			items = append(items, rawDocumentDTO(document))
		}
		ctx.JSON(http.StatusOK, rawDocumentListResponse{Items: items, Total: page.Total, Page: page.Page, PageSize: page.PageSize})
	}
}

func listEvents(client dataclient.DataServiceClient) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		query, err := eventListQueryFromRequest(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if client == nil {
			writeInternalError(ctx)
			return
		}
		page, err := client.ListEvents(dataRequestContext(ctx), query)
		if err != nil {
			writeInternalError(ctx)
			return
		}
		items := make([]eventResponse, 0, len(page.Items))
		for _, event := range page.Items {
			items = append(items, eventDTO(event))
		}
		ctx.JSON(http.StatusOK, eventListResponse{Items: items, Total: page.Total, Page: page.Page, PageSize: page.PageSize})
	}
}

func listSourceCatalogs(client dataclient.DataServiceClient) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		status := dataclient.SourceStatus(ctx.Query("status"))
		if status != "" && status != dataclient.SourceStatusActive && status != dataclient.SourceStatusInactive && status != dataclient.SourceStatusDisabled {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "unsupported source status"})
			return
		}
		if client == nil {
			writeInternalError(ctx)
			return
		}
		collection, err := client.ListSourceCatalogs(dataRequestContext(ctx), dataclient.SourceCatalogListQuery{Status: status})
		if err != nil {
			writeInternalError(ctx)
			return
		}
		items := make([]sourceCatalogResponse, 0, len(collection.Items))
		for _, source := range collection.Items {
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

func retiredScheduler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestID := requestIDForResponse(ctx)
		ctx.Header(dataclient.RequestIDHeader, requestID)
		ctx.JSON(http.StatusGone, gin.H{
			"request_id": requestID,
			"error": gin.H{
				"code":    schedulerRetiredCode,
				"message": "scheduler control has moved out of Tidewise",
				"details": gin.H{},
			},
		})
	}
}

func requestIDForResponse(ctx *gin.Context) string {
	requestID := dataclient.RequestIDFromContext(dataclient.WithRequestID(ctx.Request.Context(), ctx.GetHeader(dataclient.RequestIDHeader)))
	if requestID != "" {
		return requestID
	}
	value := make([]byte, 16)
	if _, err := rand.Read(value); err == nil {
		return "req-" + hex.EncodeToString(value)
	}
	return "req-" + strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
}

func dataRequestContext(ctx *gin.Context) context.Context {
	return dataclient.WithRequestID(ctx.Request.Context(), ctx.GetHeader(dataclient.RequestIDHeader))
}

func writeInternalError(ctx *gin.Context) {
	ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}

func rawDocumentListQueryFromRequest(ctx *gin.Context) (dataclient.RawDocumentListQuery, error) {
	page, pageSize, err := pageFromQuery(ctx)
	if err != nil {
		return dataclient.RawDocumentListQuery{}, err
	}
	return dataclient.RawDocumentListQuery{Title: ctx.Query("title"), Page: page, PageSize: pageSize}, nil
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
		ID: document.ID, SourceID: document.SourceID, IngestChannel: document.IngestChannel, SourceType: document.SourceType,
		SourceName: document.SourceName, SourceURL: document.SourceURL, SourceExternalID: document.SourceExternalID,
		Title: document.Title, ContentText: document.ContentText, RawObjectURI: document.RawObjectURI,
		RawMIMEType: document.RawMIMEType, Language: document.Language, CollectedAt: document.CollectedAt.Format(time.RFC3339),
		IngestStatus: string(document.IngestStatus),
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

func sourceCatalogDTO(source dataclient.SourceCatalog) sourceCatalogResponse {
	return sourceCatalogResponse{
		ID: source.ID, IngestChannel: source.IngestChannel, ProviderKey: source.ProviderKey, ConnectorKey: source.ConnectorKey,
		SourceType: source.SourceType, SourceName: source.SourceName, SourceURL: source.SourceURL, SourceLevel: source.SourceLevel,
		TopicHint: source.TopicHint, UsagePolicy: source.UsagePolicy, Status: string(source.Status),
	}
}
