package data

import (
	"encoding/json"
	"net/url"
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
	Actors        []string          `json:"actors"`
	Action        string            `json:"action"`
	Objects       []string          `json:"objects"`
	Stage         biz.EventStage    `json:"stage"`
	Modality      biz.EventModality `json:"modality"`
	Time          *eventTimeWire    `json:"time"`
	Jurisdictions []string          `json:"jurisdictions"`
	Reason        *string           `json:"reason"`
	Method        *string           `json:"method"`
	Metrics       []eventMetricWire `json:"metrics"`
}

func (w *eventSemanticWire) UnmarshalJSON(payload []byte) error {
	if !hasExactJSONFields(payload, "actors", "action", "objects", "stage", "modality", "time", "jurisdictions", "reason", "method", "metrics") {
		return &Error{Kind: ErrorKindDecode}
	}
	type alias eventSemanticWire
	var decoded alias
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return &Error{Kind: ErrorKindDecode}
	}
	*w = eventSemanticWire(decoded)
	return nil
}

type eventWire struct {
	ID       string                   `json:"id"`
	Title    string                   `json:"title"`
	Summary  string                   `json:"summary"`
	Semantic *eventSemanticWire       `json:"semantic"`
	Status   biz.EventLifecycleStatus `json:"status"`
}

func (w *eventWire) UnmarshalJSON(payload []byte) error {
	if !hasExactJSONFields(payload, "id", "title", "summary", "semantic", "status") {
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
		strings.TrimSpace(w.Summary) == "" || w.Semantic == nil || !validEventStatus(w.Status) {
		return biz.Event{}, &Error{Kind: ErrorKindDecode}
	}
	semantic, err := w.Semantic.toBiz()
	if err != nil {
		return biz.Event{}, err
	}
	return biz.Event{
		ID: w.ID, Title: w.Title, Summary: w.Summary,
		Semantic: semantic, Status: w.Status,
	}, nil
}

func (w eventSemanticWire) toBiz() (biz.EventSemantic, error) {
	if !validEventSemanticStringArray(w.Actors, 1) || strings.TrimSpace(w.Action) == "" ||
		!validEventSemanticStringArray(w.Objects, 1) || !validEventStage(w.Stage) || !validEventModality(w.Modality) ||
		w.Time == nil || !w.Time.valid() || !validEventSemanticStringArray(w.Jurisdictions, 0) || w.Metrics == nil {
		return biz.EventSemantic{}, &Error{Kind: ErrorKindDecode}
	}
	metrics := make([]biz.EventMetric, len(w.Metrics))
	for index, metric := range w.Metrics {
		if !metric.valid() {
			return biz.EventSemantic{}, &Error{Kind: ErrorKindDecode}
		}
		metrics[index] = biz.EventMetric{Name: metric.Name, Value: metric.Value, Unit: metric.Unit,
			Change: metric.Change, Period: metric.Period}
	}
	return biz.EventSemantic{
		Actors: append([]string{}, w.Actors...), Action: w.Action, Objects: append([]string{}, w.Objects...),
		Stage: w.Stage, Modality: w.Modality,
		Time: biz.EventTime{OccurredAt: w.Time.OccurredAt, AnnouncedAt: w.Time.AnnouncedAt,
			EffectiveAt: w.Time.EffectiveAt, ObservedAt: w.Time.ObservedAt, Precision: w.Time.Precision},
		Jurisdictions: append([]string{}, w.Jurisdictions...), Reason: w.Reason, Method: w.Method, Metrics: metrics,
	}, nil
}

type eventTimeWire struct {
	OccurredAt  *time.Time             `json:"occurred_at"`
	AnnouncedAt *time.Time             `json:"announced_at"`
	EffectiveAt *time.Time             `json:"effective_at"`
	ObservedAt  *time.Time             `json:"observed_at"`
	Precision   biz.EventTimePrecision `json:"precision"`
}

func (w *eventTimeWire) UnmarshalJSON(payload []byte) error {
	if !hasExactJSONFields(payload, "occurred_at", "announced_at", "effective_at", "precision") &&
		!hasExactJSONFields(payload, "occurred_at", "announced_at", "effective_at", "observed_at", "precision") {
		return &Error{Kind: ErrorKindDecode}
	}
	type alias eventTimeWire
	var decoded alias
	if json.Unmarshal(payload, &decoded) != nil {
		return &Error{Kind: ErrorKindDecode}
	}
	*w = eventTimeWire(decoded)
	return nil
}

func (w eventTimeWire) valid() bool {
	return (w.OccurredAt != nil || w.AnnouncedAt != nil || w.EffectiveAt != nil || w.ObservedAt != nil) &&
		(w.OccurredAt == nil || w.OccurredAt.Location() == time.UTC) &&
		(w.AnnouncedAt == nil || w.AnnouncedAt.Location() == time.UTC) &&
		(w.EffectiveAt == nil || w.EffectiveAt.Location() == time.UTC) &&
		(w.ObservedAt == nil || w.ObservedAt.Location() == time.UTC) && validEventTimePrecision(w.Precision)
}

type eventMetricWire struct {
	Name   string  `json:"name"`
	Value  *string `json:"value"`
	Unit   *string `json:"unit"`
	Change *string `json:"change"`
	Period *string `json:"period"`
}

func (w *eventMetricWire) UnmarshalJSON(payload []byte) error {
	if !hasExactJSONFields(payload, "name", "value", "unit", "change", "period") {
		return &Error{Kind: ErrorKindDecode}
	}
	type alias eventMetricWire
	var decoded alias
	if json.Unmarshal(payload, &decoded) != nil {
		return &Error{Kind: ErrorKindDecode}
	}
	*w = eventMetricWire(decoded)
	return nil
}

func (w eventMetricWire) valid() bool {
	return strings.TrimSpace(w.Name) != "" && (w.Value != nil || w.Change != nil)
}

func validEventSemanticStringArray(values []string, minimumItems int) bool {
	if values == nil || len(values) < minimumItems {
		return false
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return false
		}
		if _, exists := seen[value]; exists {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
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

func validEventStage(stage biz.EventStage) bool {
	return stage == biz.EventStageOccurred || stage == biz.EventStageAnnounced || stage == biz.EventStageEffective ||
		stage == biz.EventStageImplemented || stage == biz.EventStageUpdated || stage == biz.EventStageSuspended ||
		stage == biz.EventStageTerminated || stage == biz.EventStageExpected
}

func validEventTimePrecision(precision biz.EventTimePrecision) bool {
	return precision == biz.EventTimePrecisionInstant || precision == biz.EventTimePrecisionDay ||
		precision == biz.EventTimePrecisionRange || precision == biz.EventTimePrecisionMonth || precision == biz.EventTimePrecisionQuarter ||
		precision == biz.EventTimePrecisionYear || precision == biz.EventTimePrecisionUnknown
}

type evidencePageWire struct {
	Items    []evidenceWire `json:"items"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

type rawEvidenceResultWire struct {
	RawEvidence rawEvidenceWire `json:"raw_evidence"`
}

func (w *rawEvidenceResultWire) UnmarshalJSON(payload []byte) error {
	if !hasExactJSONFields(payload, "raw_evidence") {
		return &Error{Kind: ErrorKindDecode}
	}
	type alias rawEvidenceResultWire
	var decoded alias
	if json.Unmarshal(payload, &decoded) != nil {
		return &Error{Kind: ErrorKindDecode}
	}
	*w = rawEvidenceResultWire(decoded)
	return nil
}

func (w rawEvidenceResultWire) toBiz(expectedID string) (biz.RawEvidenceDocument, error) {
	raw := w.RawEvidence
	if raw.ID != expectedID || !rawEvidenceIDPattern.MatchString(raw.ID) ||
		strings.TrimSpace(raw.SourceID) == "" || utf8.RuneCountInString(raw.SourceID) > 32 ||
		strings.TrimSpace(raw.SourceName) == "" || utf8.RuneCountInString(raw.SourceName) > 100 ||
		!validSourceLevel(raw.SourceLevel) || strings.TrimSpace(raw.SourceURL) == "" || utf8.RuneCountInString(raw.SourceURL) > 2048 ||
		strings.TrimSpace(raw.RawText) == "" || raw.Categories == nil ||
		raw.CollectedAt.IsZero() || raw.CollectedAt.Location() != time.UTC ||
		raw.Title != nil && (strings.TrimSpace(*raw.Title) == "" || utf8.RuneCountInString(*raw.Title) > 500) ||
		raw.QuotedSourceID != nil && (strings.TrimSpace(*raw.QuotedSourceID) == "" || utf8.RuneCountInString(*raw.QuotedSourceID) > 32) ||
		raw.QuotedSourceName != nil && (strings.TrimSpace(*raw.QuotedSourceName) == "" || utf8.RuneCountInString(*raw.QuotedSourceName) > 100) ||
		raw.PublishedAt != nil && raw.PublishedAt.Location() != time.UTC ||
		raw.IsOriginal && (raw.QuotedSourceID != nil || raw.QuotedSourceName != nil) ||
		!raw.IsOriginal && raw.QuotedSourceName == nil {
		return biz.RawEvidenceDocument{}, &Error{Kind: ErrorKindDecode}
	}
	parsedSourceURL, err := url.Parse(raw.SourceURL)
	if err != nil || parsedSourceURL.Host == "" || parsedSourceURL.Scheme != "http" && parsedSourceURL.Scheme != "https" {
		return biz.RawEvidenceDocument{}, &Error{Kind: ErrorKindDecode}
	}
	seenCategories := make(map[string]struct{}, len(raw.Categories))
	for _, value := range raw.Categories {
		category, err := value.toBiz()
		if err != nil {
			return biz.RawEvidenceDocument{}, err
		}
		if _, exists := seenCategories[category.ID]; exists {
			return biz.RawEvidenceDocument{}, &Error{Kind: ErrorKindDecode}
		}
		seenCategories[category.ID] = struct{}{}
	}
	return biz.RawEvidenceDocument{RawText: raw.RawText}, nil
}

type rawEvidenceWire struct {
	ID               string                 `json:"id"`
	SourceID         string                 `json:"source_id"`
	SourceName       string                 `json:"source_name"`
	SourceLevel      string                 `json:"source_level"`
	SourceURL        string                 `json:"source_url"`
	IsOriginal       bool                   `json:"is_original"`
	QuotedSourceID   *string                `json:"quoted_source_id"`
	QuotedSourceName *string                `json:"quoted_source_name"`
	Title            *string                `json:"title"`
	RawText          string                 `json:"raw_text"`
	PublishedAt      *time.Time             `json:"published_at"`
	CollectedAt      time.Time              `json:"collected_at"`
	Categories       []evidenceCategoryWire `json:"categories"`
}

func (w *rawEvidenceWire) UnmarshalJSON(payload []byte) error {
	if !hasExactJSONFields(payload, "id", "source_id", "source_name", "source_level", "source_url", "is_original", "quoted_source_id", "quoted_source_name", "title", "raw_text", "published_at", "collected_at", "categories") {
		return &Error{Kind: ErrorKindDecode}
	}
	type alias rawEvidenceWire
	var decoded alias
	if json.Unmarshal(payload, &decoded) != nil {
		return &Error{Kind: ErrorKindDecode}
	}
	*w = rawEvidenceWire(decoded)
	return nil
}

func (w *evidencePageWire) UnmarshalJSON(payload []byte) error {
	if !hasExactJSONFields(payload, "items", "total", "page", "page_size") {
		return &Error{Kind: ErrorKindDecode}
	}
	type alias evidencePageWire
	var decoded alias
	if json.Unmarshal(payload, &decoded) != nil {
		return &Error{Kind: ErrorKindDecode}
	}
	*w = evidencePageWire(decoded)
	return nil
}
func (w evidencePageWire) toBiz() (biz.EvidencePage, error) {
	if w.Total < 0 || w.Page < 1 || w.PageSize < 1 || w.PageSize > 100 || len(w.Items) > w.PageSize || len(w.Items) > w.Total {
		return biz.EvidencePage{}, &Error{Kind: ErrorKindDecode}
	}
	items := make([]biz.Evidence, 0, len(w.Items))
	seen := make(map[string]struct{}, len(w.Items))
	for _, value := range w.Items {
		item, err := value.toBiz()
		if err != nil {
			return biz.EvidencePage{}, err
		}
		if _, exists := seen[item.ID]; exists {
			return biz.EvidencePage{}, &Error{Kind: ErrorKindDecode}
		}
		seen[item.ID] = struct{}{}
		items = append(items, item)
	}
	return biz.EvidencePage{Items: items, Total: w.Total, Page: w.Page, PageSize: w.PageSize}, nil
}

type evidenceWire struct {
	ID               string                 `json:"id"`
	RawEvidenceID    string                 `json:"raw_evidence_id"`
	Title            *string                `json:"title"`
	Summary          string                 `json:"summary"`
	Semantic         *evidenceSemanticWire  `json:"semantic"`
	Categories       []evidenceCategoryWire `json:"categories"`
	SourceID         string                 `json:"source_id"`
	SourceName       string                 `json:"source_name"`
	SourceLevel      string                 `json:"source_level"`
	SourceURL        string                 `json:"source_url"`
	IsOriginal       bool                   `json:"is_original"`
	QuotedSourceName *string                `json:"quoted_source_name"`
	Keywords         []string               `json:"keywords"`
	IsSplit          bool                   `json:"is_split"`
	PublishedAt      *time.Time             `json:"published_at"`
	CollectedAt      time.Time              `json:"collected_at"`
}

func (w *evidenceWire) UnmarshalJSON(payload []byte) error {
	if !hasExactJSONFields(payload, "id", "raw_evidence_id", "title", "summary", "semantic", "categories", "source_id", "source_name", "source_level", "source_url", "is_original", "quoted_source_name", "keywords", "is_split", "published_at", "collected_at") {
		return &Error{Kind: ErrorKindDecode}
	}
	type alias evidenceWire
	var decoded alias
	if json.Unmarshal(payload, &decoded) != nil {
		return &Error{Kind: ErrorKindDecode}
	}
	*w = evidenceWire(decoded)
	return nil
}
func (w evidenceWire) toBiz() (biz.Evidence, error) {
	if !evidenceIDPattern.MatchString(w.ID) || !rawEvidenceIDPattern.MatchString(w.RawEvidenceID) || strings.TrimSpace(w.Summary) == "" || utf8.RuneCountInString(w.Summary) > 200 ||
		strings.TrimSpace(w.SourceID) == "" || utf8.RuneCountInString(w.SourceID) > 32 || strings.TrimSpace(w.SourceName) == "" || utf8.RuneCountInString(w.SourceName) > 100 || !validSourceLevel(w.SourceLevel) || w.CollectedAt.IsZero() || w.CollectedAt.Location() != time.UTC ||
		w.Semantic == nil || !validEvidenceSemantic(*w.Semantic) || !validEvidenceKeywords(w.Keywords) || utf8.RuneCountInString(w.SourceURL) > 2048 ||
		(w.Title != nil && (strings.TrimSpace(*w.Title) == "" || utf8.RuneCountInString(*w.Title) > 500)) || (w.PublishedAt != nil && w.PublishedAt.Location() != time.UTC) {
		return biz.Evidence{}, &Error{Kind: ErrorKindDecode}
	}
	parsedURL, err := url.Parse(w.SourceURL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") ||
		w.IsOriginal && w.QuotedSourceName != nil || !w.IsOriginal && (w.QuotedSourceName == nil || strings.TrimSpace(*w.QuotedSourceName) == "") ||
		w.QuotedSourceName != nil && utf8.RuneCountInString(*w.QuotedSourceName) > 100 {
		return biz.Evidence{}, &Error{Kind: ErrorKindDecode}
	}
	categories := make([]biz.EvidenceCategory, 0, len(w.Categories))
	seen := make(map[string]struct{}, len(w.Categories))
	for _, value := range w.Categories {
		category, err := value.toBiz()
		if err != nil {
			return biz.Evidence{}, err
		}
		if _, exists := seen[category.ID]; exists {
			return biz.Evidence{}, &Error{Kind: ErrorKindDecode}
		}
		seen[category.ID] = struct{}{}
		categories = append(categories, category)
	}
	metrics := make([]biz.EvidenceMetric, 0, len(w.Semantic.Metrics))
	for _, metric := range w.Semantic.Metrics {
		metrics = append(metrics, biz.EvidenceMetric{Name: metric.Name, Value: metric.Value, Unit: metric.Unit, Change: metric.Change, Period: metric.Period})
	}
	return biz.Evidence{ID: w.ID, RawEvidenceID: w.RawEvidenceID, Title: w.Title, Summary: w.Summary,
		Semantic: biz.EvidenceSemantic{Actors: append([]string{}, w.Semantic.Actors...), Action: w.Semantic.Action,
			Objects: append([]string{}, w.Semantic.Objects...), Stage: w.Semantic.Stage, Modality: w.Semantic.Modality,
			Time:          biz.EvidenceTime{Raw: w.Semantic.Time.Raw, StartAt: w.Semantic.Time.StartAt, EndAt: w.Semantic.Time.EndAt, Precision: w.Semantic.Time.Precision},
			Jurisdictions: append([]string{}, w.Semantic.Jurisdictions...), Reason: w.Semantic.Reason, Method: w.Semantic.Method,
			Metrics: metrics, Attribution: biz.EvidenceAttribution{ReportedBy: w.Semantic.Attribution.ReportedBy, ClaimedBy: w.Semantic.Attribution.ClaimedBy}},
		Categories: categories, SourceID: w.SourceID, SourceName: w.SourceName, SourceLevel: w.SourceLevel, SourceURL: w.SourceURL,
		IsOriginal: w.IsOriginal, QuotedSourceName: w.QuotedSourceName, Keywords: append([]string{}, w.Keywords...),
		IsSplit: w.IsSplit, PublishedAt: w.PublishedAt, CollectedAt: w.CollectedAt}, nil
}

type evidenceSemanticWire struct {
	Actors        []string                `json:"actors"`
	Action        string                  `json:"action"`
	Objects       []string                `json:"objects"`
	Stage         string                  `json:"stage"`
	Modality      string                  `json:"modality"`
	Time          evidenceTimeWire        `json:"time"`
	Jurisdictions []string                `json:"jurisdictions"`
	Reason        *string                 `json:"reason"`
	Method        *string                 `json:"method"`
	Metrics       []evidenceMetricWire    `json:"metrics"`
	Attribution   evidenceAttributionWire `json:"attribution"`
}

func (w *evidenceSemanticWire) UnmarshalJSON(payload []byte) error {
	if !hasExactJSONFields(payload, "actors", "action", "objects", "stage", "modality", "time", "jurisdictions", "reason", "method", "metrics", "attribution") {
		return &Error{Kind: ErrorKindDecode}
	}
	type alias evidenceSemanticWire
	var decoded alias
	if json.Unmarshal(payload, &decoded) != nil {
		return &Error{Kind: ErrorKindDecode}
	}
	*w = evidenceSemanticWire(decoded)
	return nil
}

type evidenceTimeWire struct {
	Raw       *string    `json:"raw"`
	StartAt   *time.Time `json:"start_at"`
	EndAt     *time.Time `json:"end_at"`
	Precision string     `json:"precision"`
}

func (w *evidenceTimeWire) UnmarshalJSON(payload []byte) error {
	if !hasExactJSONFields(payload, "raw", "start_at", "end_at", "precision") {
		return &Error{Kind: ErrorKindDecode}
	}
	type alias evidenceTimeWire
	var decoded alias
	if json.Unmarshal(payload, &decoded) != nil {
		return &Error{Kind: ErrorKindDecode}
	}
	*w = evidenceTimeWire(decoded)
	return nil
}

type evidenceMetricWire struct {
	Name   string  `json:"name"`
	Value  *string `json:"value"`
	Unit   *string `json:"unit"`
	Change *string `json:"change"`
	Period *string `json:"period"`
}

func (w *evidenceMetricWire) UnmarshalJSON(payload []byte) error {
	if !hasExactJSONFields(payload, "name", "value", "unit", "change", "period") {
		return &Error{Kind: ErrorKindDecode}
	}
	type alias evidenceMetricWire
	var decoded alias
	if json.Unmarshal(payload, &decoded) != nil {
		return &Error{Kind: ErrorKindDecode}
	}
	*w = evidenceMetricWire(decoded)
	return nil
}

type evidenceAttributionWire struct {
	ReportedBy *string `json:"reported_by"`
	ClaimedBy  *string `json:"claimed_by"`
}

func (w *evidenceAttributionWire) UnmarshalJSON(payload []byte) error {
	if !hasExactJSONFields(payload, "reported_by", "claimed_by") {
		return &Error{Kind: ErrorKindDecode}
	}
	type alias evidenceAttributionWire
	var decoded alias
	if json.Unmarshal(payload, &decoded) != nil {
		return &Error{Kind: ErrorKindDecode}
	}
	*w = evidenceAttributionWire(decoded)
	return nil
}

func validEvidenceSemantic(value evidenceSemanticWire) bool {
	if !validUniqueStrings(value.Actors, 1, 20, 100) || strings.TrimSpace(value.Action) == "" || utf8.RuneCountInString(value.Action) > 200 ||
		!validUniqueStrings(value.Objects, 1, 20, 200) || !validEvidenceStage(value.Stage) || !validEvidenceModality(value.Modality) ||
		!validUniqueStrings(value.Jurisdictions, 0, 20, 100) || !validOptionalString(value.Reason, 500) || !validOptionalString(value.Method, 500) ||
		!validOptionalString(value.Attribution.ReportedBy, 100) || !validOptionalString(value.Attribution.ClaimedBy, 100) || !validEvidenceTime(value.Time) {
		return false
	}
	for _, metric := range value.Metrics {
		if strings.TrimSpace(metric.Name) == "" || utf8.RuneCountInString(metric.Name) > 100 ||
			!validOptionalString(metric.Value, 100) || !validOptionalString(metric.Unit, 50) || !validOptionalString(metric.Change, 100) || !validOptionalString(metric.Period, 100) ||
			metric.Value == nil && metric.Change == nil {
			return false
		}
	}
	return true
}

func validEvidenceTime(value evidenceTimeWire) bool {
	if !validOptionalString(value.Raw, 200) || !validEvidenceTimePrecision(value.Precision) ||
		value.StartAt != nil && value.StartAt.Location() != time.UTC || value.EndAt != nil && value.EndAt.Location() != time.UTC ||
		(value.StartAt == nil) != (value.EndAt == nil) || value.StartAt != nil && value.EndAt != nil && value.StartAt.After(*value.EndAt) {
		return false
	}
	return true
}

func validEvidenceKeywords(values []string) bool { return validUniqueStrings(values, 1, 5, 6) }

func validUniqueStrings(values []string, minimum, maximum, maxRunes int) bool {
	if len(values) < minimum || len(values) > maximum {
		return false
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || utf8.RuneCountInString(trimmed) > maxRunes {
			return false
		}
		key := trimmed
		if _, exists := seen[key]; exists {
			return false
		}
		seen[key] = struct{}{}
	}
	return true
}

func validOptionalString(value *string, maxRunes int) bool {
	return value == nil || strings.TrimSpace(*value) != "" && utf8.RuneCountInString(*value) <= maxRunes
}

func validEvidenceStage(value string) bool {
	return value == "OCCURRED" || value == "ANNOUNCED" || value == "EFFECTIVE" || value == "IMPLEMENTED" || value == "UPDATED" || value == "SUSPENDED" || value == "TERMINATED" || value == "EXPECTED"
}

func validEvidenceModality(value string) bool {
	return value == "FACT" || value == "PLAN" || value == "SPEC"
}

func validEvidenceTimePrecision(value string) bool {
	return value == "INSTANT" || value == "DAY" || value == "RANGE" || value == "MONTH" || value == "QUARTER" || value == "YEAR" || value == "UNKNOWN"
}

type evidenceCategoryWire struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (w *evidenceCategoryWire) UnmarshalJSON(payload []byte) error {
	if !hasExactJSONFields(payload, "id", "code", "name", "description") {
		return &Error{Kind: ErrorKindDecode}
	}
	type alias evidenceCategoryWire
	var decoded alias
	if json.Unmarshal(payload, &decoded) != nil {
		return &Error{Kind: ErrorKindDecode}
	}
	*w = evidenceCategoryWire(decoded)
	return nil
}
func (w evidenceCategoryWire) toBiz() (biz.EvidenceCategory, error) {
	if !evidenceCategoryIDPattern.MatchString(w.ID) || !evidenceCategoryCodePattern.MatchString(w.Code) || strings.TrimSpace(w.Name) == "" || strings.TrimSpace(w.Description) == "" || utf8.RuneCountInString(w.Name) > 50 {
		return biz.EvidenceCategory{}, &Error{Kind: ErrorKindDecode}
	}
	return biz.EvidenceCategory{ID: w.ID, Code: w.Code, Name: w.Name, Description: w.Description}, nil
}

type evidenceCategoryListWire struct {
	Categories []evidenceCategoryWire `json:"categories"`
}

func (w *evidenceCategoryListWire) UnmarshalJSON(payload []byte) error {
	if !hasExactJSONFields(payload, "categories") {
		return &Error{Kind: ErrorKindDecode}
	}
	type alias evidenceCategoryListWire
	var decoded alias
	if json.Unmarshal(payload, &decoded) != nil {
		return &Error{Kind: ErrorKindDecode}
	}
	*w = evidenceCategoryListWire(decoded)
	return nil
}
func (w evidenceCategoryListWire) toBiz() ([]biz.EvidenceCategory, error) {
	if len(w.Categories) == 0 {
		return nil, &Error{Kind: ErrorKindDecode}
	}
	items := make([]biz.EvidenceCategory, 0, len(w.Categories))
	seen := map[string]struct{}{}
	previous := ""
	for _, value := range w.Categories {
		item, err := value.toBiz()
		if err != nil {
			return nil, err
		}
		key := item.Code + "\x00" + item.ID
		if _, exists := seen[item.ID]; exists || (previous != "" && key < previous) {
			return nil, &Error{Kind: ErrorKindDecode}
		}
		seen[item.ID] = struct{}{}
		previous = key
		items = append(items, item)
	}
	return items, nil
}

type sourceWire struct {
	ID                 string          `json:"id"`
	Code               string          `json:"code"`
	Name               string          `json:"name"`
	OwnershipType      string          `json:"ownership_type"`
	ChannelType        string          `json:"channel_type"`
	AdapterKey         string          `json:"adapter_key"`
	Enabled            bool            `json:"enabled"`
	Endpoint           string          `json:"endpoint"`
	AppKey             *string         `json:"app_key"`
	Config             json.RawMessage `json:"config"`
	Priority           int             `json:"priority"`
	TimeoutSeconds     int             `json:"timeout_seconds"`
	MaxResults         int             `json:"max_results"`
	DefaultSourceLevel string          `json:"default_source_level"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

func (w *sourceWire) UnmarshalJSON(payload []byte) error {
	if !hasExactJSONFields(payload, "id", "code", "name", "ownership_type", "channel_type", "adapter_key", "enabled", "endpoint", "app_key", "config", "priority", "timeout_seconds", "max_results", "default_source_level", "created_at", "updated_at") {
		return &Error{Kind: ErrorKindDecode}
	}
	type alias sourceWire
	var decoded alias
	if json.Unmarshal(payload, &decoded) != nil {
		return &Error{Kind: ErrorKindDecode}
	}
	*w = sourceWire(decoded)
	return nil
}
func (w sourceWire) toBiz() (biz.Source, error) {
	var config map[string]any
	parsedEndpoint, endpointErr := url.Parse(w.Endpoint)
	if !sourceIDPattern.MatchString(w.ID) || !sourceCodePattern.MatchString(w.Code) || strings.TrimSpace(w.Name) == "" || utf8.RuneCountInString(w.Name) > 100 ||
		(w.OwnershipType != "fixed" && w.OwnershipType != "dynamic") || (w.ChannelType != "web_search" && w.ChannelType != "api" && w.ChannelType != "rss") || !validAdapterKey(w.AdapterKey) || endpointErr != nil || parsedEndpoint.Scheme == "" || parsedEndpoint.Host == "" || utf8.RuneCountInString(w.Endpoint) > 2048 ||
		(w.AppKey != nil && utf8.RuneCountInString(*w.AppKey) > 512) || json.Unmarshal(w.Config, &config) != nil || config == nil || w.Priority < 1 || w.Priority > 5 || w.TimeoutSeconds < 1 || w.TimeoutSeconds > 300 || w.MaxResults < 1 || w.MaxResults > 100 || !validSourceLevel(w.DefaultSourceLevel) || w.CreatedAt.IsZero() || w.UpdatedAt.IsZero() || w.CreatedAt.Location() != time.UTC || w.UpdatedAt.Location() != time.UTC || w.CreatedAt.After(w.UpdatedAt) {
		return biz.Source{}, &Error{Kind: ErrorKindDecode}
	}
	return biz.Source{ID: w.ID, Code: w.Code, Name: w.Name, OwnershipType: w.OwnershipType, ChannelType: w.ChannelType, Enabled: w.Enabled, Priority: w.Priority, DefaultSourceLevel: w.DefaultSourceLevel, UpdatedAt: w.UpdatedAt}, nil
}

type sourceListWire struct {
	Sources []sourceWire `json:"sources"`
}

func (w *sourceListWire) UnmarshalJSON(payload []byte) error {
	if !hasExactJSONFields(payload, "sources") {
		return &Error{Kind: ErrorKindDecode}
	}
	type alias sourceListWire
	var decoded alias
	if json.Unmarshal(payload, &decoded) != nil {
		return &Error{Kind: ErrorKindDecode}
	}
	*w = sourceListWire(decoded)
	return nil
}
func (w sourceListWire) toBiz() ([]biz.Source, error) {
	if len(w.Sources) > 200 {
		return nil, &Error{Kind: ErrorKindDecode}
	}
	items := make([]biz.Source, 0, len(w.Sources))
	seen := map[string]struct{}{}
	var previous *biz.Source
	for _, value := range w.Sources {
		item, err := value.toBiz()
		if err != nil {
			return nil, err
		}
		if _, exists := seen[item.ID]; exists || previous != nil && (previous.Priority > item.Priority || previous.Priority == item.Priority && (previous.Code > item.Code || previous.Code == item.Code && previous.ID >= item.ID)) {
			return nil, &Error{Kind: ErrorKindDecode}
		}
		seen[item.ID] = struct{}{}
		items = append(items, item)
		previous = &items[len(items)-1]
	}
	return items, nil
}

var evidenceIDPattern = regexp.MustCompile(`^EVD[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var rawEvidenceIDPattern = regexp.MustCompile(`^RAW[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var evidenceCategoryIDPattern = regexp.MustCompile(`^EVC[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var evidenceCategoryCodePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
var sourceIDPattern = regexp.MustCompile(`^SRC[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var sourceCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

func validSourceLevel(value string) bool {
	return value == "L1_OFFICIAL" || value == "L2_WIRE" || value == "L3_MEDIA" || value == "L4_SOCIAL"
}

func validAdapterKey(value string) bool {
	switch value {
	case "bocha", "tavily", "parallel", "cls", "eastmoney_fast", "eastmoney_stock", "stcn", "generic_rss":
		return true
	default:
		return false
	}
}
