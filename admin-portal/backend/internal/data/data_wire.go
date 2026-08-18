package data

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/biz"
)

type eventPageWire struct {
	Items    []eventWire `json:"items"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

func (w *eventPageWire) UnmarshalJSON(payload []byte) error {
	if !hasExactJSONFields(payload, "items", "total", "page", "page_size") {
		return &Error{Kind: ErrorKindDecode}
	}
	type alias eventPageWire
	var decoded alias
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return &Error{Kind: ErrorKindDecode}
	}
	*w = eventPageWire(decoded)
	return nil
}

func (w eventPageWire) toBiz() (biz.EventPage, error) {
	if w.Total < 0 || w.Page < 1 || w.PageSize < 1 || w.PageSize > 100 || len(w.Items) > w.PageSize || len(w.Items) > w.Total {
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

func (w *eventSemanticWire) UnmarshalJSON(payload []byte) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(payload, &object); err != nil || !hasExactJSONFields(payload, "who", "what", "when", "where", "why", "how") {
		return &Error{Kind: ErrorKindDecode}
	}
	values := map[string]**string{
		"who": &w.Who, "what": &w.What, "when": &w.When,
		"where": &w.Where, "why": &w.Why, "how": &w.How,
	}
	for name, target := range values {
		raw, present := object[name]
		if !present || json.Unmarshal(raw, target) != nil {
			return &Error{Kind: ErrorKindDecode}
		}
	}
	return nil
}

type eventWire struct {
	ID          string                   `json:"id"`
	Title       string                   `json:"title"`
	Summary     string                   `json:"summary"`
	Semantic    *eventSemanticWire       `json:"semantic"`
	Modality    biz.EventModality        `json:"modality"`
	OccurredAt  *time.Time               `json:"occurred_at"`
	AnnouncedAt *time.Time               `json:"announced_at"`
	Status      biz.EventLifecycleStatus `json:"status"`
}

func (w *eventWire) UnmarshalJSON(payload []byte) error {
	if !hasExactJSONFields(payload, "id", "title", "summary", "semantic", "modality", "occurred_at", "announced_at", "status") {
		return &Error{Kind: ErrorKindDecode}
	}
	type alias eventWire
	var decoded alias
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return &Error{Kind: ErrorKindDecode}
	}
	*w = eventWire(decoded)
	return nil
}

func (w eventWire) toBiz() (biz.Event, error) {
	if !eventIDPattern.MatchString(w.ID) || strings.TrimSpace(w.Title) == "" || utf8.RuneCountInString(w.Title) > 200 ||
		strings.TrimSpace(w.Summary) == "" || w.Semantic == nil || !validEventModality(w.Modality) || !validEventStatus(w.Status) {
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

var eventIDPattern = regexp.MustCompile(`^EVT[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func hasExactJSONFields(payload []byte, names ...string) bool {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(payload, &object); err != nil || len(object) != len(names) {
		return false
	}
	for _, name := range names {
		if _, present := object[name]; !present {
			return false
		}
	}
	return true
}

func validEventModality(modality biz.EventModality) bool {
	return modality == biz.EventModalityFact || modality == biz.EventModalityPlan || modality == biz.EventModalitySpec
}

func validEventStatus(status biz.EventLifecycleStatus) bool {
	return status == biz.EventLifecycleActive || status == biz.EventLifecycleDeprecated || status == biz.EventLifecycleArchived
}
