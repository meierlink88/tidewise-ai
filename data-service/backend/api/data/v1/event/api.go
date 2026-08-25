package event

import (
	"context"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const (
	OperationListAdminEvents = "data.v1.listAdminEvents"
	OperationPublishEvent    = "data.v1.publishEvent"

	ErrorInvalidRequest                = "INVALID_REQUEST"
	ErrorDataServiceNotReady           = "DATA_SERVICE_NOT_READY"
	ErrorDataRepositoryFailure         = "DATA_REPOSITORY_FAILURE"
	ErrorEventPublishConflict          = "EVENT_PUBLISH_CONFLICT"
	ErrorEventEvidenceReferenceInvalid = "EVENT_EVIDENCE_REFERENCE_INVALID"
)

func BusinessOperations() []string { return []string{OperationListAdminEvents, OperationPublishEvent} }

type Service interface {
	ListEvents(context.Context, *ListRequest) (*v1.Response[Page], error)
	PublishEvent(context.Context, *PublicationRequest) (*v1.Response[PublicationResult], error)
}

type ListRequest struct {
	Title         string
	Modality      string
	Status        string
	OccurredFrom  string
	OccurredTo    string
	AnnouncedFrom string
	AnnouncedTo   string
	Page          string
	PageSize      string
}

type Semantic struct {
	Actors        []string `json:"actors"`
	Action        string   `json:"action"`
	Objects       []string `json:"objects"`
	Stage         string   `json:"stage"`
	Jurisdictions []string `json:"jurisdictions"`
	EffectiveAt   *string  `json:"effective_at"`
	TimePrecision string   `json:"time_precision"`
}

type PublicationRequest struct {
	PublicationKey string           `json:"publication_key"`
	Event          PublicationEvent `json:"event"`
	EvidenceIDs    []string         `json:"evidence_ids"`
}

type PublicationEvent struct {
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	Semantic    Semantic `json:"semantic"`
	Modality    string   `json:"modality"`
	OccurredAt  *string  `json:"occurred_at"`
	AnnouncedAt *string  `json:"announced_at"`
}

type PublicationResult struct {
	Event           Item     `json:"event"`
	EvidenceLinkIDs []string `json:"evidence_link_ids"`
	ReceiptID       string   `json:"receipt_id"`
	PayloadHash     string   `json:"payload_hash"`
	Replayed        bool     `json:"replayed"`
}

type Item struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	Semantic    Semantic `json:"semantic"`
	Modality    string   `json:"modality"`
	OccurredAt  *string  `json:"occurred_at"`
	AnnouncedAt *string  `json:"announced_at"`
	Status      string   `json:"status"`
}

type Page struct {
	Items    []Item `json:"items"`
	Total    int    `json:"total"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}
