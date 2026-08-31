package v1

import (
	"context"
	"time"
)

const APIPrefix = "/api/admin/v1"

const (
	OperationListEvents             = "admin.events.list"
	OperationListEvidences          = "admin.evidences.list"
	OperationGetCollectionDocument  = "admin.collectionDocument.get"
	OperationListEvidenceCategories = "admin.evidenceCategories.list"
	OperationListSources            = "admin.sources.list"
	OperationGetRuntimeHealth       = "admin.runtimeHealth.get"
)

type AdminHTTPServer interface {
	ListEvents(context.Context, *ListEventsRequest) (*EventListResponse, error)
	ListEvidences(context.Context, *ListEvidencesRequest) (*EvidenceListResponse, error)
	GetCollectionDocument(context.Context, *GetCollectionDocumentRequest) (*CollectionDocumentResponse, error)
	ListEvidenceCategories(context.Context, *EmptyRequest) (*EvidenceCategoryListResponse, error)
	ListSources(context.Context, *ListSourcesRequest) (*SourceListResponse, error)
	GetRuntimeHealth(context.Context, *EmptyRequest) (*RuntimeHealth, error)
}

type EmptyRequest struct{}

type GetCollectionDocumentRequest struct {
	RawEvidenceID string
}

type CollectionDocumentResponse struct {
	Available bool    `json:"available"`
	URL       *string `json:"url"`
}

type ListEventsRequest struct {
	Title         string
	Modality      string
	Status        string
	OccurredFrom  string
	OccurredTo    string
	AnnouncedFrom string
	AnnouncedTo   string
	Page          int
	PageSize      int
}

type ListEvidencesRequest struct {
	Title, Summary, CategoryID, SourceID, SourceName, SourceLevel, IsSplit string
	PublishedFrom, PublishedTo, CollectedFrom, CollectedTo                 string
	Page, PageSize                                                         int
}

type ListSourcesRequest struct {
	Query, OwnershipType, ChannelType, Enabled, Priority, DefaultSourceLevel string
	UpdatedFrom, UpdatedTo                                                   string
	Page, PageSize                                                           int
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

type EventListResponse struct {
	Items    []Event `json:"items"`
	Total    int     `json:"total"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
}

type Event struct {
	ID       string        `json:"id"`
	Title    string        `json:"title"`
	Summary  string        `json:"summary"`
	Semantic EventSemantic `json:"semantic"`
	Status   string        `json:"status"`
}

type EventSemantic struct {
	Actors        []string      `json:"actors"`
	Action        string        `json:"action"`
	Objects       []string      `json:"objects"`
	Stage         string        `json:"stage"`
	Modality      string        `json:"modality"`
	Time          EventTime     `json:"time"`
	Jurisdictions []string      `json:"jurisdictions"`
	Reason        *string       `json:"reason"`
	Method        *string       `json:"method"`
	Metrics       []EventMetric `json:"metrics"`
}

type EventTime struct {
	OccurredAt  *string `json:"occurred_at"`
	AnnouncedAt *string `json:"announced_at"`
	EffectiveAt *string `json:"effective_at"`
	ObservedAt  *string `json:"observed_at"`
	Precision   string  `json:"precision"`
}

type EventMetric struct {
	Name   string  `json:"name"`
	Value  *string `json:"value"`
	Unit   *string `json:"unit"`
	Change *string `json:"change"`
	Period *string `json:"period"`
}

type EvidenceListResponse struct {
	Items    []Evidence `json:"items"`
	Total    int        `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
}

type Evidence struct {
	ID               string             `json:"id"`
	RawEvidenceID    string             `json:"raw_evidence_id"`
	Title            *string            `json:"title"`
	Summary          string             `json:"summary"`
	Semantic         EvidenceSemantic   `json:"semantic"`
	Categories       []EvidenceCategory `json:"categories"`
	SourceID         string             `json:"source_id"`
	SourceName       string             `json:"source_name"`
	SourceLevel      string             `json:"source_level"`
	SourceURL        string             `json:"source_url"`
	IsOriginal       bool               `json:"is_original"`
	QuotedSourceName *string            `json:"quoted_source_name"`
	Keywords         []string           `json:"keywords"`
	IsSplit          bool               `json:"is_split"`
	PublishedAt      *string            `json:"published_at"`
	CollectedAt      string             `json:"collected_at"`
}

type EvidenceSemantic struct {
	Actors        []string            `json:"actors"`
	Action        string              `json:"action"`
	Objects       []string            `json:"objects"`
	Stage         string              `json:"stage"`
	Modality      string              `json:"modality"`
	Time          EvidenceTime        `json:"time"`
	Jurisdictions []string            `json:"jurisdictions"`
	Reason        *string             `json:"reason"`
	Method        *string             `json:"method"`
	Metrics       []EvidenceMetric    `json:"metrics"`
	Attribution   EvidenceAttribution `json:"attribution"`
}

type EvidenceTime struct {
	Raw       *string `json:"raw"`
	StartAt   *string `json:"start_at"`
	EndAt     *string `json:"end_at"`
	Precision string  `json:"precision"`
}

type EvidenceMetric struct {
	Name   string  `json:"name"`
	Value  *string `json:"value"`
	Unit   *string `json:"unit"`
	Change *string `json:"change"`
	Period *string `json:"period"`
}

type EvidenceAttribution struct {
	ReportedBy *string `json:"reported_by"`
	ClaimedBy  *string `json:"claimed_by"`
}

type EvidenceCategory struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type EvidenceCategoryListResponse struct {
	Categories []EvidenceCategory `json:"categories"`
}

type SourceListResponse struct {
	Items    []Source `json:"items"`
	Total    int      `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
}

type Source struct {
	ID                 string `json:"id"`
	Code               string `json:"code"`
	Name               string `json:"name"`
	OwnershipType      string `json:"ownership_type"`
	ChannelType        string `json:"channel_type"`
	Enabled            bool   `json:"enabled"`
	Priority           int    `json:"priority"`
	DefaultSourceLevel string `json:"default_source_level"`
	UpdatedAt          string `json:"updated_at"`
}
