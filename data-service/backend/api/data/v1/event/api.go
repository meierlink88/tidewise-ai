package event

import (
	"context"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const (
	OperationListAdminEvents = "data.v1.listAdminEvents"

	ErrorInvalidRequest        = "INVALID_REQUEST"
	ErrorDataServiceNotReady   = "DATA_SERVICE_NOT_READY"
	ErrorDataRepositoryFailure = "DATA_REPOSITORY_FAILURE"
)

func BusinessOperations() []string { return []string{OperationListAdminEvents} }

type Service interface {
	ListEvents(context.Context, *ListRequest) (*v1.Response[Page], error)
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
	Who   *string `json:"who"`
	What  *string `json:"what"`
	When  *string `json:"when"`
	Where *string `json:"where"`
	Why   *string `json:"why"`
	How   *string `json:"how"`
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
