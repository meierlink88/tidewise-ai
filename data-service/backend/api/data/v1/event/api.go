package event

import (
	"context"
	"encoding/json"
	"errors"

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
	Modality      string   `json:"modality"`
	Time          Time     `json:"time"`
	Jurisdictions []string `json:"jurisdictions"`
	Reason        *string  `json:"reason"`
	Method        *string  `json:"method"`
	Metrics       []Metric `json:"metrics"`
}

func (s *Semantic) UnmarshalJSON(payload []byte) error {
	if !hasExactFields(payload, "actors", "action", "objects", "stage", "modality", "time", "jurisdictions", "reason", "method", "metrics") {
		return errors.New("Event semantic must contain the exact business proposition fields")
	}
	type semanticAlias Semantic
	return json.Unmarshal(payload, (*semanticAlias)(s))
}

type Time struct {
	OccurredAt  *string `json:"occurred_at"`
	AnnouncedAt *string `json:"announced_at"`
	EffectiveAt *string `json:"effective_at"`
	ObservedAt  *string `json:"observed_at"`
	Precision   string  `json:"precision"`
}

func (t *Time) UnmarshalJSON(payload []byte) error {
	// Accept the previous four-field request during the coordinated rollout.
	// Every response and persisted semantic is emitted with observed_at.
	if !hasExactFields(payload, "occurred_at", "announced_at", "effective_at", "precision") &&
		!hasExactFields(payload, "occurred_at", "announced_at", "effective_at", "observed_at", "precision") {
		return errors.New("Event time must contain the exact business time fields")
	}
	type timeAlias Time
	return json.Unmarshal(payload, (*timeAlias)(t))
}

type Metric struct {
	Name   string  `json:"name"`
	Value  *string `json:"value"`
	Unit   *string `json:"unit"`
	Change *string `json:"change"`
	Period *string `json:"period"`
}

func (m *Metric) UnmarshalJSON(payload []byte) error {
	if !hasExactFields(payload, "name", "value", "unit", "change", "period") {
		return errors.New("Event metric must contain the exact metric fields")
	}
	type metricAlias Metric
	return json.Unmarshal(payload, (*metricAlias)(m))
}

func hasExactFields(payload []byte, fields ...string) bool {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(payload, &object); err != nil || len(object) != len(fields) {
		return false
	}
	for _, field := range fields {
		if _, exists := object[field]; !exists {
			return false
		}
	}
	return true
}

type PublicationRequest struct {
	PublicationKey string           `json:"publication_key"`
	Event          PublicationEvent `json:"event"`
	EvidenceIDs    []string         `json:"evidence_ids"`
}

type PublicationEvent struct {
	Title    string   `json:"title"`
	Summary  string   `json:"summary"`
	Semantic Semantic `json:"semantic"`
}

type PublicationResult struct {
	Event           Item     `json:"event"`
	EvidenceLinkIDs []string `json:"evidence_link_ids"`
	ReceiptID       string   `json:"receipt_id"`
	PayloadHash     string   `json:"payload_hash"`
	Replayed        bool     `json:"replayed"`
}

type Item struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Summary  string   `json:"summary"`
	Semantic Semantic `json:"semantic"`
	Status   string   `json:"status"`
}

type Page struct {
	Items    []Item `json:"items"`
	Total    int    `json:"total"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}
