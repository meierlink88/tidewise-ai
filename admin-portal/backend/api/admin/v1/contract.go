package v1

import (
	"context"
	"time"
)

const APIPrefix = "/api/admin/v1"

const (
	OperationListEvents       = "admin.events.list"
	OperationGetRuntimeHealth = "admin.runtimeHealth.get"
)

type AdminHTTPServer interface {
	ListEvents(context.Context, *ListEventsRequest) (*EventListResponse, error)
	GetRuntimeHealth(context.Context, *EmptyRequest) (*RuntimeHealth, error)
}

type EmptyRequest struct{}

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
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Summary     string        `json:"summary"`
	Semantic    EventSemantic `json:"semantic"`
	Modality    string        `json:"modality"`
	OccurredAt  string        `json:"occurred_at,omitempty"`
	AnnouncedAt string        `json:"announced_at,omitempty"`
	Status      string        `json:"status"`
}

type EventSemantic struct {
	Who   *string `json:"who"`
	What  *string `json:"what"`
	When  *string `json:"when"`
	Where *string `json:"where"`
	Why   *string `json:"why"`
	How   *string `json:"how"`
}
