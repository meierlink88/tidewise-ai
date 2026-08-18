package v1

import (
	"context"
	"time"
)

const APIPrefix = "/api/admin/v1"

const (
	OperationListRawDocuments = "admin.rawDocuments.list"
	OperationListEvents       = "admin.events.list"
	OperationGetRuntimeHealth = "admin.runtimeHealth.get"
)

type AdminHTTPServer interface {
	ListRawDocuments(context.Context, *ListRawDocumentsRequest) (*RawDocumentListResponse, error)
	ListEvents(context.Context, *ListEventsRequest) (*EventListResponse, error)
	GetRuntimeHealth(context.Context, *EmptyRequest) (*RuntimeHealth, error)
}

type EmptyRequest struct{}

type ListRawDocumentsRequest struct {
	Title     string
	SourceRef string
	Page      int
	PageSize  int
}

type ListEventsRequest struct {
	Title         string
	EventStatus   string
	FactStatus    string
	EventTimeFrom string
	EventTimeTo   string
	FirstSeenFrom string
	FirstSeenTo   string
	Page          int
	PageSize      int
}

type RuntimeHealth struct {
	Status    string                 `json:"status"`
	CheckedAt time.Time              `json:"checked_at"`
	Services  []RuntimeHealthService `json:"services"`
}

type RuntimeHealthService struct {
	Key         string    `json:"key"`
	DisplayName string    `json:"display_name"`
	Status      string    `json:"status"`
	CheckedAt   time.Time `json:"checked_at"`
	LatencyMS   *int64    `json:"latency_ms,omitempty"`
	ReasonCode  string    `json:"reason_code,omitempty"`
}

type RawDocumentListResponse struct {
	Items    []RawDocument `json:"items"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

type RawDocument struct {
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

type EventListResponse struct {
	Items    []Event `json:"items"`
	Total    int     `json:"total"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
}

type Event struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	EventTime   string `json:"event_time,omitempty"`
	FirstSeenAt string `json:"first_seen_at"`
	KnowableAt  string `json:"knowable_at,omitempty"`
	EventStatus string `json:"event_status"`
	FactStatus  string `json:"fact_status"`
	DedupeKey   string `json:"dedupe_key"`
}
