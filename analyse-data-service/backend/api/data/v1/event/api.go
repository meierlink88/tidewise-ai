package event

import (
	"context"
	"encoding/json"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
)

const (
	OperationPublishReviewedEvents = "data.v1.publishReviewedEvents"
	OperationListActiveEventTags   = "data.v1.listActiveEventTags"
	OperationListAdminEvents       = "data.v1.listAdminEvents"

	ErrorInvalidRequest           = "INVALID_REQUEST"
	ErrorEventPublicationInvalid  = "EVENT_PUBLICATION_INVALID"
	ErrorEventPublicationConflict = "EVENT_PUBLICATION_CONFLICT"
	ErrorEventPublicationFailed   = "EVENT_PUBLICATION_FAILED"
	ErrorEventTagCatalogFailed    = "EVENT_TAG_CATALOG_FAILED"
	ErrorDataServiceNotReady      = "DATA_SERVICE_NOT_READY"
	ErrorDataRepositoryFailure    = "DATA_REPOSITORY_FAILURE"
)

func BusinessOperations() []string {
	return []string{OperationPublishReviewedEvents, OperationListActiveEventTags, OperationListAdminEvents}
}

type Service interface {
	PublishReviewedEvents(context.Context, *PublicationRequest) (*v1.Response[PublicationResult], error)
	ListActiveEventTags(context.Context, *TagCatalogRequest) (*v1.Response[TagCatalog], error)
	ListEvents(context.Context, *ListRequest) (*v1.Response[Page], error)
}

type PublicationRequest struct {
	PackageID    string                   `json:"package_id"`
	Provenance   PublicationProvenance    `json:"provenance"`
	RawDocuments []PublicationRawDocument `json:"raw_documents"`
	Events       []PublicationEvent       `json:"events"`
}

type PublicationProvenance struct {
	ExtractorExecutionID  string                          `json:"extractor_execution_id"`
	ExtractorAgentVersion string                          `json:"extractor_agent_version"`
	CollectorExecutions   []PublicationCollectorExecution `json:"collector_executions"`
}

type PublicationCollectorExecution struct {
	ArtifactID           string `json:"artifact_id"`
	CollectorExecutionID string `json:"collector_execution_id"`
}

type PublicationRawDocument struct {
	ArtifactID    string     `json:"artifact_id"`
	ContentSHA256 string     `json:"content_sha256"`
	SourceRef     string     `json:"source_ref"`
	SourceName    string     `json:"source_name"`
	SourceType    string     `json:"source_type"`
	SourceURL     string     `json:"source_url,omitempty"`
	Title         string     `json:"title"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	CollectedAt   time.Time  `json:"collected_at"`
	Language      string     `json:"language,omitempty"`
	MIMEType      string     `json:"mime_type,omitempty"`
}

type PublicationEvent struct {
	DedupeKey      string                `json:"dedupe_key"`
	Title          string                `json:"title"`
	FactualSummary string                `json:"factual_summary"`
	OccurredAt     *time.Time            `json:"occurred_at,omitempty"`
	FactPayload    map[string]any        `json:"fact_payload"`
	Evidence       []PublicationEvidence `json:"evidence"`
	Tags           []PublicationTag      `json:"tags"`
	Review         PublicationReview     `json:"review"`
}

type PublicationEvidence struct {
	ArtifactID        string   `json:"artifact_id"`
	EvidenceRelation  string   `json:"evidence_relation"`
	EvidenceStatement string   `json:"evidence_statement"`
	SupportsFields    []string `json:"supports_fields"`
	SourceLevel       string   `json:"source_level"`
}

type PublicationTag struct {
	TagID            string      `json:"tag_id"`
	TagKind          string      `json:"tag_kind"`
	TagCode          string      `json:"tag_code"`
	Confidence       json.Number `json:"confidence"`
	AssignmentReason string      `json:"assignment_reason"`
	AssignSource     string      `json:"assign_source"`
}

type PublicationReview struct {
	ReviewID      string   `json:"review_id"`
	EvidenceGrade string   `json:"evidence_grade"`
	Reasons       []string `json:"reasons"`
}

type PublicationResult struct {
	ReceiptID    string                         `json:"receipt_id"`
	PackageID    string                         `json:"package_id"`
	ImportedAt   time.Time                      `json:"imported_at"`
	Events       []PublicationEventResult       `json:"events"`
	RawDocuments []PublicationRawDocumentResult `json:"raw_documents"`
	Counts       PublicationCounts              `json:"counts"`
}

type PublicationEventResult struct {
	DedupeKey   string `json:"dedupe_key"`
	EventID     string `json:"event_id"`
	Disposition string `json:"disposition"`
}

type PublicationRawDocumentResult struct {
	ArtifactID    string `json:"artifact_id"`
	RawDocumentID string `json:"raw_document_id"`
	Disposition   string `json:"disposition"`
}

type PublicationCounts struct {
	EventsCreated       int `json:"events_created"`
	EventsReused        int `json:"events_reused"`
	RawDocumentsCreated int `json:"raw_documents_created"`
	RawDocumentsReused  int `json:"raw_documents_reused"`
	EventSourcesCreated int `json:"event_sources_created"`
	EventSourcesReused  int `json:"event_sources_reused"`
	EventTagsCreated    int `json:"event_tags_created"`
	EventTagsReused     int `json:"event_tags_reused"`
}

type TagCatalogRequest struct{ Active bool }

type TagCatalog struct {
	CatalogRevision string           `json:"catalog_revision"`
	CatalogHash     string           `json:"catalog_hash"`
	Tags            []TagCatalogItem `json:"tags"`
}

type TagCatalogItem struct {
	ID       string `json:"id"`
	TagKind  string `json:"tag_kind"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
}

type ListRequest struct {
	Title         string
	EventStatus   string
	FactStatus    string
	EventTimeFrom string
	EventTimeTo   string
	FirstSeenFrom string
	FirstSeenTo   string
	Page          string
	PageSize      string
}

type Item struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Summary     string  `json:"summary"`
	EventTime   *string `json:"event_time"`
	FirstSeenAt string  `json:"first_seen_at"`
	KnowableAt  *string `json:"knowable_at"`
	EventStatus string  `json:"event_status"`
	FactStatus  string  `json:"fact_status"`
	DedupeKey   string  `json:"dedupe_key"`
}

type Page struct {
	Items    []Item `json:"items"`
	Total    int    `json:"total"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}
