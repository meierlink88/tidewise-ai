package data

import (
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/biz"
)

type eventPageWire struct {
	Items    []eventWire `json:"items"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

func (w eventPageWire) toBiz() (biz.EventPage, error) {
	if w.Total < 0 || w.Page < 1 || w.PageSize < 1 {
		return biz.EventPage{}, &Error{Kind: ErrorKindDecode}
	}
	items := make([]biz.Event, 0, len(w.Items))
	for _, item := range w.Items {
		mapped, err := item.toBiz()
		if err != nil {
			return biz.EventPage{}, err
		}
		items = append(items, mapped)
	}
	return biz.EventPage{Items: items, Total: w.Total, Page: w.Page, PageSize: w.PageSize}, nil
}

type eventSemanticWire struct {
	Who   *string `json:"who"`
	What  *string `json:"what"`
	When  *string `json:"when"`
	Where *string `json:"where"`
	Why   *string `json:"why"`
	How   *string `json:"how"`
}

type eventWire struct {
	ID          string                   `json:"id"`
	Title       string                   `json:"title"`
	Summary     string                   `json:"summary"`
	Semantic    eventSemanticWire        `json:"semantic"`
	Modality    biz.EventModality        `json:"modality"`
	OccurredAt  *time.Time               `json:"occurred_at"`
	AnnouncedAt *time.Time               `json:"announced_at"`
	Status      biz.EventLifecycleStatus `json:"status"`
}

func (w eventWire) toBiz() (biz.Event, error) {
	if strings.TrimSpace(w.ID) == "" || !validEventModality(w.Modality) || !validEventStatus(w.Status) {
		return biz.Event{}, &Error{Kind: ErrorKindDecode}
	}
	return biz.Event{
		ID: w.ID, Title: w.Title, Summary: w.Summary,
		Semantic: biz.EventSemantic{
			Who: w.Semantic.Who, What: w.Semantic.What, When: w.Semantic.When,
			Where: w.Semantic.Where, Why: w.Semantic.Why, How: w.Semantic.How,
		},
		Modality: w.Modality, OccurredAt: w.OccurredAt, AnnouncedAt: w.AnnouncedAt, Status: w.Status,
	}, nil
}

func validEventModality(modality biz.EventModality) bool {
	return modality == biz.EventModalityFact || modality == biz.EventModalityPlan || modality == biz.EventModalitySpec
}

func validEventStatus(status biz.EventLifecycleStatus) bool {
	return status == biz.EventLifecycleActive || status == biz.EventLifecycleDeprecated || status == biz.EventLifecycleArchived
}
