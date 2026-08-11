package event

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/identity"
)

const (
	MinEvents = 1
	MaxEvents = 10

	EventSourceLevelPrimary   = "primary"
	EventSourceLevelSecondary = "secondary"
	EventTagKindNewsCategory  = "news_category"
	EventTagKindIndexCategory = "index_category"
	EventFieldTitle           = "title"
	EventFieldFactualSummary  = "factual_summary"
	EventFieldOccurredAt      = "occurred_at"
	EventFieldFactPayload     = "fact_payload"
)

type Store interface {
	TransactionStore
	ListActiveTags(context.Context) ([]EventTag, error)
	ListEvents(context.Context, EventListFilter) (EventStorePage, error)
}

var (
	lowerSHA256 = regexp.MustCompile(`^[0-9a-f]{64}$`)
	lowerUUID   = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

type PublicationBatch struct {
	PackageID    string                `json:"package_id"`
	Provenance   Provenance            `json:"provenance"`
	RawDocuments []EventEvidenceRecord `json:"raw_documents"`
	Events       []PublicationEvent    `json:"events"`
}

type Provenance struct {
	ExtractorExecutionID  string               `json:"extractor_execution_id"`
	ExtractorAgentVersion string               `json:"extractor_agent_version"`
	CollectorExecutions   []CollectorExecution `json:"collector_executions"`
}

type CollectorExecution struct {
	ArtifactID           string `json:"artifact_id"`
	CollectorExecutionID string `json:"collector_execution_id"`
}

type EventEvidenceRecord struct {
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
	DedupeKey      string                   `json:"dedupe_key"`
	Title          string                   `json:"title"`
	FactualSummary string                   `json:"factual_summary"`
	OccurredAt     *time.Time               `json:"occurred_at,omitempty"`
	FactPayload    map[string]any           `json:"fact_payload"`
	Evidence       []EventEvidenceLinkInput `json:"evidence"`
	Tags           []EventTagInput          `json:"tags"`
	Review         Review                   `json:"review"`
}

type EventEvidenceLinkInput struct {
	ArtifactID        string   `json:"artifact_id"`
	EvidenceRelation  string   `json:"evidence_relation"`
	EvidenceStatement string   `json:"evidence_statement"`
	SupportsFields    []string `json:"supports_fields"`
	SourceLevel       string   `json:"source_level"`
}

type EventTagInput struct {
	TagID            string      `json:"tag_id"`
	TagKind          string      `json:"tag_kind"`
	TagCode          string      `json:"tag_code"`
	Confidence       json.Number `json:"confidence"`
	AssignmentReason string      `json:"assignment_reason"`
	AssignSource     string      `json:"assign_source"`
}

type Review struct {
	ReviewID      string   `json:"review_id"`
	EvidenceGrade string   `json:"evidence_grade"`
	Reasons       []string `json:"reasons"`
}

type ValidationIssue struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ValidationError struct {
	Issues []ValidationIssue
}

func (e *ValidationError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "event publication failed validation"
	}
	return fmt.Sprintf("%s: %s", e.Issues[0].Path, e.Issues[0].Message)
}

func NewValidationError(issues []ValidationIssue) *ValidationError {
	copied := append([]ValidationIssue(nil), issues...)
	sort.SliceStable(copied, func(i, j int) bool {
		if copied[i].Path != copied[j].Path {
			return copied[i].Path < copied[j].Path
		}
		if copied[i].Code != copied[j].Code {
			return copied[i].Code < copied[j].Code
		}
		return copied[i].Message < copied[j].Message
	})
	return &ValidationError{Issues: copied}
}

func DecodeStrict(reader io.Reader) (PublicationBatch, error) {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var publication PublicationBatch
	if err := decoder.Decode(&publication); err != nil {
		return PublicationBatch{}, fmt.Errorf("decode Event Publication: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return PublicationBatch{}, fmt.Errorf("Event Publication body must contain exactly one JSON object")
		}
		return PublicationBatch{}, fmt.Errorf("decode trailing Event Publication data: %w", err)
	}
	return publication, nil
}

func (p PublicationBatch) Validate() error {
	var issues []ValidationIssue
	addBoundedRequired(&issues, "package_id", p.PackageID, 256)
	addBoundedRequired(&issues, "provenance.extractor_execution_id", p.Provenance.ExtractorExecutionID, 256)
	addBoundedRequired(&issues, "provenance.extractor_agent_version", p.Provenance.ExtractorAgentVersion, 256)
	if len(p.Events) < MinEvents || len(p.Events) > MaxEvents {
		addIssue(&issues, "events", "INVALID_COUNT", "events must contain 1..10 items")
	}
	if len(p.RawDocuments) == 0 {
		addIssue(&issues, "raw_documents", "INVALID_COUNT", "raw_documents must contain at least one item")
	}

	documents := make(map[string]EventEvidenceRecord, len(p.RawDocuments))
	referencedArtifacts := make(map[string]struct{}, len(p.RawDocuments))
	for index, document := range p.RawDocuments {
		path := fmt.Sprintf("raw_documents[%d]", index)
		addBoundedRequired(&issues, path+".artifact_id", document.ArtifactID, 256)
		if document.ArtifactID != "" {
			if _, duplicate := documents[document.ArtifactID]; duplicate {
				addIssue(&issues, path+".artifact_id", "DUPLICATE_ARTIFACT", "artifact_id must be unique in the package")
			} else {
				documents[document.ArtifactID] = document
			}
		}
		if !lowerSHA256.MatchString(document.ContentSHA256) {
			addIssue(&issues, path+".content_sha256", "INVALID_SHA256", "content_sha256 must be lowercase 64-character hexadecimal")
		}
		addBoundedRequired(&issues, path+".source_ref", document.SourceRef, 256)
		addBoundedRequired(&issues, path+".source_name", document.SourceName, 300)
		addBoundedRequired(&issues, path+".source_type", document.SourceType, 64)
		addBoundedRequired(&issues, path+".title", document.Title, 1000)
		addOptionalMaxLength(&issues, path+".source_url", document.SourceURL, 2048)
		addOptionalMaxLength(&issues, path+".language", document.Language, 16)
		addOptionalMaxLength(&issues, path+".mime_type", document.MIMEType, 128)
		if document.CollectedAt.IsZero() {
			addIssue(&issues, path+".collected_at", "REQUIRED", "collected_at is required")
		} else if !isUTC(document.CollectedAt) {
			addIssue(&issues, path+".collected_at", "INVALID_TIMESTAMP", "collected_at must use UTC")
		}
		if document.PublishedAt != nil && !isUTC(*document.PublishedAt) {
			addIssue(&issues, path+".published_at", "INVALID_TIMESTAMP", "published_at must use UTC")
		}
		if document.SourceURL != "" && !absoluteHTTPURL(document.SourceURL) {
			addIssue(&issues, path+".source_url", "INVALID_URL", "source_url must be an absolute HTTP(S) URL")
		}
	}

	collectorArtifacts := make(map[string]struct{}, len(p.Provenance.CollectorExecutions))
	for index, execution := range p.Provenance.CollectorExecutions {
		path := fmt.Sprintf("provenance.collector_executions[%d]", index)
		addBoundedRequired(&issues, path+".artifact_id", execution.ArtifactID, 256)
		addBoundedRequired(&issues, path+".collector_execution_id", execution.CollectorExecutionID, 256)
		if _, duplicate := collectorArtifacts[execution.ArtifactID]; duplicate {
			addIssue(&issues, path+".artifact_id", "DUPLICATE_ARTIFACT", "collector execution artifact_id must be unique")
		} else {
			collectorArtifacts[execution.ArtifactID] = struct{}{}
		}
		if execution.ArtifactID != "" {
			if _, exists := documents[execution.ArtifactID]; !exists {
				addIssue(&issues, path+".artifact_id", "UNKNOWN_ARTIFACT", "artifact_id is not declared in raw_documents")
			}
		}
	}
	for index, document := range p.RawDocuments {
		if _, exists := collectorArtifacts[document.ArtifactID]; !exists {
			addIssue(&issues, fmt.Sprintf("raw_documents[%d].artifact_id", index), "MISSING_COLLECTOR_EXECUTION", "artifact_id must have one collector execution")
		}
	}

	seenEvents := make(map[string]struct{}, len(p.Events))
	for eventIndex, event := range p.Events {
		path := fmt.Sprintf("events[%d]", eventIndex)
		addRequired(&issues, path+".dedupe_key", event.DedupeKey)
		if event.DedupeKey != "" {
			if _, duplicate := seenEvents[event.DedupeKey]; duplicate {
				addIssue(&issues, path+".dedupe_key", "DUPLICATE_EVENT", "dedupe_key must be unique in the package")
			} else {
				seenEvents[event.DedupeKey] = struct{}{}
			}
		}
		addRequired(&issues, path+".title", event.Title)
		addRequired(&issues, path+".factual_summary", event.FactualSummary)
		if event.OccurredAt != nil && !isUTC(*event.OccurredAt) {
			addIssue(&issues, path+".occurred_at", "INVALID_TIMESTAMP", "occurred_at must use UTC")
		}
		if err := ValidateFactPayload(event.FactPayload); err != nil {
			addIssue(&issues, path+".fact_payload", "INVALID_FACT_PAYLOAD", err.Error())
		}
		validateEvidence(&issues, path, event.Evidence, documents, referencedArtifacts)
		validateTags(&issues, path, event.Tags)
		validateReview(&issues, path, event.Review)
	}

	for index, document := range p.RawDocuments {
		if _, referenced := referencedArtifacts[document.ArtifactID]; !referenced {
			addIssue(&issues, fmt.Sprintf("raw_documents[%d].artifact_id", index), "UNREFERENCED_ARTIFACT", "every raw document must be referenced by at least one Event")
		}
	}
	if len(issues) > 0 {
		return NewValidationError(issues)
	}
	return nil
}

func validateEvidence(issues *[]ValidationIssue, eventPath string, evidence []EventEvidenceLinkInput, documents map[string]EventEvidenceRecord, referenced map[string]struct{}) {
	if len(evidence) == 0 {
		addIssue(issues, eventPath+".evidence", "INVALID_COUNT", "each Event must contain at least one evidence item")
		return
	}
	seen := make(map[string]struct{}, len(evidence))
	allowedFields := map[string]struct{}{
		EventFieldTitle: {}, EventFieldFactualSummary: {}, EventFieldOccurredAt: {}, EventFieldFactPayload: {},
	}
	for index, item := range evidence {
		path := fmt.Sprintf("%s.evidence[%d]", eventPath, index)
		addRequired(issues, path+".artifact_id", item.ArtifactID)
		if _, duplicate := seen[item.ArtifactID]; duplicate {
			addIssue(issues, path+".artifact_id", "DUPLICATE_EVIDENCE", "an artifact can appear only once in one Event")
		} else {
			seen[item.ArtifactID] = struct{}{}
		}
		if _, exists := documents[item.ArtifactID]; !exists {
			addIssue(issues, path+".artifact_id", "UNKNOWN_ARTIFACT", "artifact_id is not declared in raw_documents")
		} else {
			referenced[item.ArtifactID] = struct{}{}
		}
		if item.EvidenceRelation != string(EvidenceRelationSupports) &&
			item.EvidenceRelation != string(EvidenceRelationContradicts) &&
			item.EvidenceRelation != string(EvidenceRelationContext) {
			addIssue(issues, path+".evidence_relation", "INVALID_ENUM", "evidence_relation must be supports, contradicts, or context")
		}
		addRequired(issues, path+".evidence_statement", item.EvidenceStatement)
		if item.SourceLevel != EventSourceLevelPrimary && item.SourceLevel != EventSourceLevelSecondary {
			addIssue(issues, path+".source_level", "INVALID_ENUM", "source_level must be primary or secondary")
		}
		if (item.EvidenceRelation == string(EvidenceRelationSupports) ||
			item.EvidenceRelation == string(EvidenceRelationContradicts)) && len(item.SupportsFields) == 0 {
			addIssue(issues, path+".supports_fields", "INVALID_COUNT", "supports_fields must be non-empty for supports or contradicts evidence")
		}
		seenFields := make(map[string]struct{}, len(item.SupportsFields))
		for fieldIndex, field := range item.SupportsFields {
			fieldPath := fmt.Sprintf("%s.supports_fields[%d]", path, fieldIndex)
			if _, allowed := allowedFields[field]; !allowed {
				addIssue(issues, fieldPath, "INVALID_ENUM", "supports_fields contains an unsupported Event field")
			}
			if _, duplicate := seenFields[field]; duplicate {
				addIssue(issues, fieldPath, "DUPLICATE_FIELD", "supports_fields must not contain duplicates")
			}
			seenFields[field] = struct{}{}
		}
	}
}

func validateTags(issues *[]ValidationIssue, eventPath string, tags []EventTagInput) {
	newsCount := 0
	indexCount := 0
	seen := make(map[string]struct{}, len(tags))
	for index, tag := range tags {
		path := fmt.Sprintf("%s.tags[%d]", eventPath, index)
		if !lowerUUID.MatchString(tag.TagID) {
			addIssue(issues, path+".tag_id", "INVALID_UUID", "tag_id must be a lowercase UUID")
		}
		addRequired(issues, path+".tag_code", tag.TagCode)
		switch tag.TagKind {
		case EventTagKindNewsCategory:
			newsCount++
		case EventTagKindIndexCategory:
			indexCount++
		default:
			addIssue(issues, path+".tag_kind", "INVALID_ENUM", "tag_kind must be news_category or index_category")
		}
		if _, duplicate := seen[tag.TagID]; duplicate {
			addIssue(issues, path+".tag_id", "DUPLICATE_TAG", "tag_id must be unique in one Event")
		}
		seen[tag.TagID] = struct{}{}
		if tag.AssignSource != string(TagAssignSourceAI) && tag.AssignSource != string(TagAssignSourceRule) {
			addIssue(issues, path+".assign_source", "INVALID_ENUM", "assign_source must be ai or rule")
		}
		addRequired(issues, path+".assignment_reason", tag.AssignmentReason)
		confidence, ok := new(big.Rat).SetString(string(tag.Confidence))
		if !ok || confidence.Sign() < 0 || confidence.Cmp(big.NewRat(1, 1)) > 0 {
			addIssue(issues, path+".confidence", "INVALID_CONFIDENCE", "confidence must be between 0 and 1")
		}
	}
	if newsCount < 1 || newsCount > 2 {
		addIssue(issues, eventPath+".tags", "INVALID_NEWS_TAG_COUNT", "each Event must contain 1..2 news_category tags")
	}
	if indexCount > 3 {
		addIssue(issues, eventPath+".tags", "INVALID_INDEX_TAG_COUNT", "each Event can contain at most 3 index_category tags")
	}
}

func validateReview(issues *[]ValidationIssue, eventPath string, review Review) {
	addRequired(issues, eventPath+".review.review_id", review.ReviewID)
	addRequired(issues, eventPath+".review.evidence_grade", review.EvidenceGrade)
	if len(review.Reasons) == 0 {
		addIssue(issues, eventPath+".review.reasons", "INVALID_COUNT", "review reasons must contain at least one item")
	}
	for index, reason := range review.Reasons {
		addRequired(issues, fmt.Sprintf("%s.review.reasons[%d]", eventPath, index), reason)
	}
}

func addRequired(issues *[]ValidationIssue, path string, value string) {
	if strings.TrimSpace(value) == "" {
		addIssue(issues, path, "REQUIRED", "field must be non-empty")
	}
}

func addBoundedRequired(issues *[]ValidationIssue, path string, value string, maxLength int) {
	addRequired(issues, path, value)
	addOptionalMaxLength(issues, path, value, maxLength)
}

func addOptionalMaxLength(issues *[]ValidationIssue, path string, value string, maxLength int) {
	if utf8.RuneCountInString(value) > maxLength {
		addIssue(issues, path, "MAX_LENGTH", fmt.Sprintf("field must contain at most %d characters", maxLength))
	}
}

func addIssue(issues *[]ValidationIssue, path, code, message string) {
	*issues = append(*issues, ValidationIssue{Path: path, Code: code, Message: message})
}

func absoluteHTTPURL(raw string) bool {
	parsed, err := url.Parse(raw)
	return err == nil &&
		(parsed.Scheme == "http" || parsed.Scheme == "https") &&
		parsed.Host != "" &&
		parsed.User == nil
}

func isUTC(value time.Time) bool {
	_, offset := value.Zone()
	return offset == 0
}

func SemanticJSONEqual(left, right any) bool {
	leftValue, err := decodeSemanticJSON(left)
	if err != nil {
		return false
	}
	rightValue, err := decodeSemanticJSON(right)
	if err != nil {
		return false
	}
	return equalSemanticJSON(leftValue, rightValue)
}

func decodeSemanticJSON(value any) (any, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func equalSemanticJSON(left, right any) bool {
	switch typedLeft := left.(type) {
	case nil:
		return right == nil
	case bool:
		typedRight, ok := right.(bool)
		return ok && typedLeft == typedRight
	case string:
		typedRight, ok := right.(string)
		return ok && typedLeft == typedRight
	case json.Number:
		typedRight, ok := right.(json.Number)
		if !ok {
			return false
		}
		leftNumber, leftOK := new(big.Rat).SetString(string(typedLeft))
		rightNumber, rightOK := new(big.Rat).SetString(string(typedRight))
		return leftOK && rightOK && leftNumber.Cmp(rightNumber) == 0
	case []any:
		typedRight, ok := right.([]any)
		if !ok || len(typedLeft) != len(typedRight) {
			return false
		}
		for index := range typedLeft {
			if !equalSemanticJSON(typedLeft[index], typedRight[index]) {
				return false
			}
		}
		return true
	case map[string]any:
		typedRight, ok := right.(map[string]any)
		if !ok || len(typedLeft) != len(typedRight) {
			return false
		}
		for key, leftValue := range typedLeft {
			rightValue, exists := typedRight[key]
			if !exists || !equalSemanticJSON(leftValue, rightValue) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

type EventStatus string

const (
	EventStatusCandidate EventStatus = "candidate"
	EventStatusConfirmed EventStatus = "confirmed"
	EventStatusRejected  EventStatus = "rejected"
)

type FactStatus string

const (
	FactStatusUnverified FactStatus = "unverified"
	FactStatusVerified   FactStatus = "verified"
	FactStatusDisputed   FactStatus = "disputed"
)

type FactPayload map[string]any

var forbiddenFactPayloadKeys = map[string]struct{}{
	"buy":                       {},
	"sell":                      {},
	"buy_recommendation":        {},
	"sell_recommendation":       {},
	"investment_advice":         {},
	"direct_investment_advice":  {},
	"investment_recommendation": {},
	"recommendation":            {},
	"price_prediction":          {},
	"price_forecast":            {},
	"return_prediction":         {},
	"prediction":                {},
	"forecast":                  {},
	"event_score":               {},
	"score":                     {},
	"scoring":                   {},
	"transmission_strength":     {},
	"favorable":                 {},
	"unfavorable":               {},
	"bullish":                   {},
	"bearish":                   {},
}

func ValidateFactPayload(payload any) error {
	if payload == nil {
		return fmt.Errorf("fact payload must be a JSON object")
	}

	var object map[string]any
	switch value := payload.(type) {
	case FactPayload:
		object = map[string]any(value)
	case map[string]any:
		object = value
	default:
		return fmt.Errorf("fact payload must be a JSON object")
	}
	if object == nil {
		return fmt.Errorf("fact payload must be a JSON object")
	}

	if _, err := json.Marshal(object); err != nil {
		return fmt.Errorf("fact payload must be JSON encodable: %w", err)
	}
	for key := range object {
		normalizedKey := strings.NewReplacer("-", "_", " ", "_").Replace(strings.ToLower(strings.TrimSpace(key)))
		if _, forbidden := forbiddenFactPayloadKeys[normalizedKey]; forbidden {
			return fmt.Errorf("fact payload key %q is not allowed", key)
		}
	}
	return nil
}

type Event struct {
	ID          string
	Title       string
	Summary     string
	EventTime   *time.Time
	FirstSeenAt time.Time
	KnowableAt  *time.Time
	EventStatus EventStatus
	FactStatus  FactStatus
	DedupeKey   string
	FactPayload FactPayload
}

func (e Event) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("event id is required")
	}
	if e.Title == "" {
		return fmt.Errorf("title is required")
	}
	if e.FirstSeenAt.IsZero() {
		return fmt.Errorf("first seen at is required")
	}
	if e.DedupeKey == "" {
		return fmt.Errorf("dedupe key is required")
	}
	if !validStatus(e.EventStatus, EventStatusCandidate, EventStatusConfirmed, EventStatusRejected) {
		return fmt.Errorf("unsupported event status %q", e.EventStatus)
	}
	if !validStatus(e.FactStatus, FactStatusUnverified, FactStatusVerified, FactStatusDisputed) {
		return fmt.Errorf("unsupported fact status %q", e.FactStatus)
	}
	if err := ValidateFactPayload(e.FactPayload); err != nil {
		return err
	}
	return nil
}

type EvidenceRelation string

const (
	EvidenceRelationSupports    EvidenceRelation = "supports"
	EvidenceRelationContradicts EvidenceRelation = "contradicts"
	EvidenceRelationContext     EvidenceRelation = "context"
)

type EventEvidenceLink struct {
	ID                string
	EventID           string
	RawDocumentID     string
	SourceLevel       string
	EvidenceStatement string
	EvidenceHash      string
	EvidenceRelation  EvidenceRelation
	SupportsFields    []string
}

func (s EventEvidenceLink) Validate() error {
	if s.EvidenceRelation == "" {
		return nil
	}
	if !validStatus(s.EvidenceRelation, EvidenceRelationSupports, EvidenceRelationContradicts, EvidenceRelationContext) {
		return fmt.Errorf("unsupported evidence relation %q", s.EvidenceRelation)
	}
	if s.EvidenceRelation != EvidenceRelationSupports && s.EvidenceRelation != EvidenceRelationContradicts {
		return nil
	}
	if len(s.SupportsFields) == 0 {
		return fmt.Errorf("supports fields are required for evidence relation %q", s.EvidenceRelation)
	}
	for _, field := range s.SupportsFields {
		if strings.TrimSpace(field) == "" {
			return fmt.Errorf("supports fields must not contain blank values")
		}
	}
	return nil
}

type EventTag struct {
	ID     string
	Kind   string
	Code   string
	Name   string
	Active bool
}

type ReviewStatus string

const (
	ReviewStatusCandidate ReviewStatus = "candidate"
	ReviewStatusReviewed  ReviewStatus = "reviewed"
	ReviewStatusPending   ReviewStatus = "pending"
	ReviewStatusApproved  ReviewStatus = "approved"
	ReviewStatusRejected  ReviewStatus = "rejected"
)

type EventTagAssignment struct {
	ID               string
	EventID          string
	TagID            string
	AssignSource     string
	ReviewStatus     ReviewStatus
	Confidence       *float64
	AssignmentReason string
}

const (
	TagAssignSourceAI   = "ai"
	TagAssignSourceRule = "rule"
)

func (m EventTagAssignment) Validate() error {
	if m.Confidence != nil {
		if math.IsNaN(*m.Confidence) || math.IsInf(*m.Confidence, 0) || *m.Confidence < 0 || *m.Confidence > 1 {
			return fmt.Errorf("tag confidence must be between 0 and 1")
		}
	}
	assignSource := strings.ToLower(strings.TrimSpace(m.AssignSource))
	if (assignSource == TagAssignSourceAI || assignSource == TagAssignSourceRule) && strings.TrimSpace(m.AssignmentReason) == "" {
		return fmt.Errorf("assignment reason is required for %s tag assignment", assignSource)
	}
	return nil
}

type Result struct {
	ReceiptID    string              `json:"receipt_id"`
	PackageID    string              `json:"package_id"`
	ImportedAt   time.Time           `json:"imported_at"`
	Events       []EventResult       `json:"events"`
	RawDocuments []RawDocumentResult `json:"raw_documents"`
	Counts       Counts              `json:"counts"`
}

type EventResult struct {
	DedupeKey   string `json:"dedupe_key"`
	EventID     string `json:"event_id"`
	Disposition string `json:"disposition"`
}

type RawDocumentResult struct {
	ArtifactID    string `json:"artifact_id"`
	RawDocumentID string `json:"raw_document_id"`
	Disposition   string `json:"disposition"`
}

type Disposition string

const (
	DispositionCreated Disposition = "created"
	DispositionReused  Disposition = "reused"
)

type Counts struct {
	EventsCreated       int `json:"events_created"`
	EventsReused        int `json:"events_reused"`
	RawDocumentsCreated int `json:"raw_documents_created"`
	RawDocumentsReused  int `json:"raw_documents_reused"`
	EventSourcesCreated int `json:"event_sources_created"`
	EventSourcesReused  int `json:"event_sources_reused"`
	EventTagsCreated    int `json:"event_tags_created"`
	EventTagsReused     int `json:"event_tags_reused"`
}

type ConflictIssue struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ConflictError struct {
	Issues []ConflictIssue
}

func (e *ConflictError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "Event Publication conflicts with stored data"
	}
	return fmt.Sprintf("%s: %s", e.Issues[0].Path, e.Issues[0].Message)
}

type UseCase struct {
	store   Store
	now     func() time.Time
	newUUID func() (string, error)
}

func NewUseCase(store Store) (*UseCase, error) {
	if store == nil {
		return nil, errors.New("Event store is required")
	}
	return &UseCase{
		store:   store,
		now:     func() time.Time { return time.Now().UTC() },
		newUUID: randomUUID,
	}, nil
}

type rawPlan struct {
	input       EventEvidenceRecord
	record      StoredEventEvidenceRecord
	disposition Disposition
}

type eventPlan struct {
	input       PublicationEvent
	record      StoredEvent
	disposition Disposition
	sources     []sourcePlan
	tags        []tagPlan
}

type sourcePlan struct {
	record      StoredEventEvidenceLink
	disposition Disposition
}

type tagPlan struct {
	record      StoredEventTagAssignment
	disposition Disposition
}

func (s *UseCase) Import(ctx context.Context, callerSubject string, publication PublicationBatch) (Result, error) {
	if s == nil || s.store == nil {
		return Result{}, errors.New("Event Publication store is required")
	}
	if strings.TrimSpace(callerSubject) == "" {
		return Result{}, errors.New("Event Publication caller subject is required")
	}
	if err := publication.Validate(); err != nil {
		return Result{}, err
	}

	var result Result
	err := s.store.InTransaction(ctx, func(tx Transaction) error {
		rawPlans, eventPlans, identities := planPublication(publication)
		if err := tx.LockIdentities(ctx, identities); err != nil {
			return err
		}

		validationIssues, conflicts, err := inspectExisting(ctx, tx, rawPlans, eventPlans)
		if err != nil {
			return err
		}
		if len(validationIssues) > 0 {
			return NewValidationError(validationIssues)
		}
		if len(conflicts) > 0 {
			sort.SliceStable(conflicts, func(i, j int) bool {
				if conflicts[i].Path != conflicts[j].Path {
					return conflicts[i].Path < conflicts[j].Path
				}
				return conflicts[i].Code < conflicts[j].Code
			})
			return &ConflictError{Issues: conflicts}
		}

		if err := writePublication(ctx, tx, rawPlans, eventPlans); err != nil {
			return err
		}
		receiptID, err := s.newUUID()
		if err != nil {
			return fmt.Errorf("generate Event Publication receipt ID: %w", err)
		}
		importedAt := s.now().UTC()
		result = buildResult(receiptID, importedAt, publication.PackageID, rawPlans, eventPlans)
		if err := tx.InsertEventPublicationReceipt(ctx, buildReceipt(
			receiptID, importedAt, callerSubject, publication, rawPlans, eventPlans, result.Counts,
		)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func planPublication(publication PublicationBatch) ([]*rawPlan, []*eventPlan, []string) {
	rawPlans := make([]*rawPlan, 0, len(publication.RawDocuments))
	rawByArtifact := make(map[string]*rawPlan, len(publication.RawDocuments))
	identities := make([]string, 0, len(publication.RawDocuments)+len(publication.Events))
	for _, input := range publication.RawDocuments {
		plan := &rawPlan{
			input: input,
			record: StoredEventEvidenceRecord{
				ID:         identity.NormalizeUUID("raw_document_artifact", input.ArtifactID),
				ArtifactID: input.ArtifactID, ContentSHA256: input.ContentSHA256,
				SourceRef: input.SourceRef, SourceName: input.SourceName, SourceType: input.SourceType,
				SourceURL: input.SourceURL, Title: input.Title, PublishedAt: input.PublishedAt,
				CollectedAt: input.CollectedAt, Language: input.Language, MIMEType: input.MIMEType,
			},
			disposition: DispositionCreated,
		}
		rawPlans = append(rawPlans, plan)
		rawByArtifact[input.ArtifactID] = plan
		identities = append(identities, "raw-document:"+input.ArtifactID)
	}

	eventPlans := make([]*eventPlan, 0, len(publication.Events))
	for _, input := range publication.Events {
		firstSeenAt, knowableAt := observationTimes(input, rawByArtifact)
		plan := &eventPlan{
			input: input,
			record: StoredEvent{
				ID:        identity.NormalizeUUID("event", input.DedupeKey),
				DedupeKey: input.DedupeKey, Title: input.Title, FactualSummary: input.FactualSummary,
				OccurredAt: input.OccurredAt, FactPayload: FactPayload(input.FactPayload),
				FirstSeenAt: firstSeenAt, KnowableAt: knowableAt,
				EventStatus: EventStatusConfirmed, FactStatus: FactStatusVerified,
			},
			disposition: DispositionCreated,
		}
		eventPlans = append(eventPlans, plan)
		identities = append(identities, "event:"+input.DedupeKey)
	}
	return rawPlans, eventPlans, identities
}

func observationTimes(event PublicationEvent, rawByArtifact map[string]*rawPlan) (time.Time, time.Time) {
	var firstSeenAt time.Time
	var knowableAt time.Time
	for _, evidence := range event.Evidence {
		document := rawByArtifact[evidence.ArtifactID].input
		if firstSeenAt.IsZero() || document.CollectedAt.Before(firstSeenAt) {
			firstSeenAt = document.CollectedAt
		}
		candidate := document.CollectedAt
		if document.PublishedAt != nil {
			candidate = *document.PublishedAt
		}
		if knowableAt.IsZero() || candidate.Before(knowableAt) {
			knowableAt = candidate
		}
	}
	return firstSeenAt, knowableAt
}

func inspectExisting(
	ctx context.Context,
	tx Transaction,
	rawPlans []*rawPlan,
	eventPlans []*eventPlan,
) ([]ValidationIssue, []ConflictIssue, error) {
	var validationIssues []ValidationIssue
	var conflicts []ConflictIssue
	rawByArtifact := make(map[string]*rawPlan, len(rawPlans))

	for index, plan := range rawPlans {
		existing, err := tx.StoredEventEvidenceRecord(ctx, plan.input.ArtifactID)
		if err != nil {
			return nil, nil, err
		}
		if existing != nil {
			plan.record.ID = existing.ID
			plan.disposition = DispositionReused
			if !sameRawDocument(*existing, plan.record) {
				conflicts = append(conflicts, ConflictIssue{
					Path: fmt.Sprintf("raw_documents[%d]", index), Code: "ARTIFACT_CONFLICT",
					Message: "artifact_id is already bound to different evidence metadata",
				})
			}
		}
		rawByArtifact[plan.input.ArtifactID] = plan
	}

	for eventIndex, plan := range eventPlans {
		existing, err := tx.StoredEvent(ctx, plan.input.DedupeKey)
		if err != nil {
			return nil, nil, err
		}
		if existing != nil {
			plan.record.ID = existing.ID
			plan.disposition = DispositionReused
			if !sameEventCore(*existing, plan.record) {
				conflicts = append(conflicts, ConflictIssue{
					Path: fmt.Sprintf("events[%d]", eventIndex), Code: "EVENT_CONFLICT",
					Message: "dedupe_key is already bound to different immutable Event facts",
				})
			}
		}

		for evidenceIndex, evidence := range plan.input.Evidence {
			rawPlan := rawByArtifact[evidence.ArtifactID]
			record := StoredEventEvidenceLink{
				ID:      identity.NormalizeUUID("event_source_v2", plan.record.ID, rawPlan.record.ID),
				EventID: plan.record.ID, RawDocumentID: rawPlan.record.ID,
				SourceLevel: evidence.SourceLevel, EvidenceStatement: evidence.EvidenceStatement,
				EvidenceHash:     hashEvidenceStatement(evidence.EvidenceStatement),
				EvidenceRelation: EvidenceRelation(evidence.EvidenceRelation),
				SupportsFields:   append([]string{}, evidence.SupportsFields...),
			}
			sourcePlan := sourcePlan{record: record, disposition: DispositionCreated}
			stored, err := tx.StoredEventEvidenceLink(ctx, record.EventID, record.RawDocumentID)
			if err != nil {
				return nil, nil, err
			}
			if stored != nil {
				sourcePlan.record.ID = stored.ID
				sourcePlan.disposition = DispositionReused
				if !sameEventSource(*stored, record) {
					conflicts = append(conflicts, ConflictIssue{
						Path:    fmt.Sprintf("events[%d].evidence[%d]", eventIndex, evidenceIndex),
						Code:    "EVIDENCE_CONFLICT",
						Message: "Event and artifact are already bound to different evidence semantics",
					})
				}
			}
			plan.sources = append(plan.sources, sourcePlan)
		}

		for tagIndex, input := range plan.input.Tags {
			path := fmt.Sprintf("events[%d].tags[%d]", eventIndex, tagIndex)
			tag, err := tx.PublicationTag(ctx, input.TagID)
			if err != nil {
				return nil, nil, err
			}
			if tag == nil {
				validationIssues = append(validationIssues, ValidationIssue{
					Path: path + ".tag_id", Code: "UNKNOWN_TAG", Message: "tag_id does not exist",
				})
			} else if !tag.Active {
				validationIssues = append(validationIssues, ValidationIssue{
					Path: path + ".tag_id", Code: "INACTIVE_TAG", Message: "tag_id is inactive",
				})
			} else if tag.Kind != input.TagKind || tag.Code != input.TagCode {
				validationIssues = append(validationIssues, ValidationIssue{
					Path: path, Code: "TAG_IDENTITY_MISMATCH", Message: "tag_id, tag_kind, and tag_code do not identify the same Tag",
				})
			}

			record := StoredEventTagAssignment{
				ID:      identity.NormalizeUUID("event_tag_map", plan.record.ID, input.TagID),
				EventID: plan.record.ID, TagID: input.TagID, AssignSource: input.AssignSource,
				ReviewStatus: ReviewStatusApproved, Confidence: string(input.Confidence),
				AssignmentReason: input.AssignmentReason,
			}
			tagPlan := tagPlan{record: record, disposition: DispositionCreated}
			stored, err := tx.StoredEventTagAssignment(ctx, record.EventID, record.TagID)
			if err != nil {
				return nil, nil, err
			}
			if stored != nil {
				tagPlan.record.ID = stored.ID
				tagPlan.disposition = DispositionReused
				if !sameEventTag(*stored, record) {
					conflicts = append(conflicts, ConflictIssue{
						Path: path, Code: "EVENT_TAG_CONFLICT",
						Message: "Event and Tag are already bound to different assignment semantics",
					})
				}
			}
			plan.tags = append(plan.tags, tagPlan)
		}
	}
	return validationIssues, conflicts, nil
}

func writePublication(ctx context.Context, tx Transaction, rawPlans []*rawPlan, eventPlans []*eventPlan) error {
	for _, plan := range rawPlans {
		if plan.disposition == DispositionCreated {
			if err := tx.InsertStoredEventEvidenceRecord(ctx, plan.record); err != nil {
				return err
			}
		}
	}
	for _, plan := range eventPlans {
		if plan.disposition == DispositionCreated {
			if err := tx.InsertStoredEvent(ctx, plan.record); err != nil {
				return err
			}
		} else {
			if err := tx.AdvanceStoredEventObservationTimes(
				ctx, plan.record.ID, plan.record.FirstSeenAt, plan.record.KnowableAt,
			); err != nil {
				return err
			}
		}
		for _, source := range plan.sources {
			if source.disposition == DispositionCreated {
				if err := tx.InsertStoredEventEvidenceLink(ctx, source.record); err != nil {
					return err
				}
			}
		}
		for _, tag := range plan.tags {
			if tag.disposition == DispositionCreated {
				if err := tx.InsertStoredEventTagAssignment(ctx, tag.record); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func buildResult(
	receiptID string,
	importedAt time.Time,
	packageID string,
	rawPlans []*rawPlan,
	eventPlans []*eventPlan,
) Result {
	result := Result{
		ReceiptID: receiptID, PackageID: packageID, ImportedAt: importedAt,
		Events:       make([]EventResult, 0, len(eventPlans)),
		RawDocuments: make([]RawDocumentResult, 0, len(rawPlans)),
	}
	for _, plan := range rawPlans {
		result.RawDocuments = append(result.RawDocuments, RawDocumentResult{
			ArtifactID: plan.input.ArtifactID, RawDocumentID: plan.record.ID, Disposition: string(plan.disposition),
		})
		if plan.disposition == DispositionCreated {
			result.Counts.RawDocumentsCreated++
		} else {
			result.Counts.RawDocumentsReused++
		}
	}
	for _, plan := range eventPlans {
		result.Events = append(result.Events, EventResult{
			DedupeKey: plan.input.DedupeKey, EventID: plan.record.ID, Disposition: string(plan.disposition),
		})
		if plan.disposition == DispositionCreated {
			result.Counts.EventsCreated++
		} else {
			result.Counts.EventsReused++
		}
		for _, source := range plan.sources {
			if source.disposition == DispositionCreated {
				result.Counts.EventSourcesCreated++
			} else {
				result.Counts.EventSourcesReused++
			}
		}
		for _, tag := range plan.tags {
			if tag.disposition == DispositionCreated {
				result.Counts.EventTagsCreated++
			} else {
				result.Counts.EventTagsReused++
			}
		}
	}
	return result
}

func buildReceipt(
	receiptID string,
	importedAt time.Time,
	callerSubject string,
	publication PublicationBatch,
	rawPlans []*rawPlan,
	eventPlans []*eventPlan,
	counts Counts,
) EventPublicationReceipt {
	eventIDs := make([]string, 0, len(eventPlans))
	rawIDs := make([]string, 0, len(rawPlans))
	sourceIDs := make([]string, 0)
	tagIDs := make([]string, 0)
	collectorExecutions := make([]PublicationCollectorExecution, 0, len(publication.Provenance.CollectorExecutions))
	for _, execution := range publication.Provenance.CollectorExecutions {
		collectorExecutions = append(collectorExecutions, PublicationCollectorExecution{
			ArtifactID:           execution.ArtifactID,
			CollectorExecutionID: execution.CollectorExecutionID,
		})
	}
	reviews := make([]PublicationReviewMetadata, 0, len(eventPlans))
	for _, plan := range rawPlans {
		rawIDs = append(rawIDs, plan.record.ID)
	}
	for _, plan := range eventPlans {
		eventIDs = append(eventIDs, plan.record.ID)
		for _, source := range plan.sources {
			sourceIDs = append(sourceIDs, source.record.ID)
		}
		for _, tag := range plan.tags {
			tagIDs = append(tagIDs, tag.record.ID)
		}
		reviews = append(reviews, PublicationReviewMetadata{
			DedupeKey:     plan.input.DedupeKey,
			ReviewID:      plan.input.Review.ReviewID,
			EvidenceGrade: plan.input.Review.EvidenceGrade,
			Reasons:       plan.input.Review.Reasons,
		})
	}
	writeCounts := PublicationWriteCounts{
		EventsCreated: counts.EventsCreated, EventsReused: counts.EventsReused,
		RawDocumentsCreated: counts.RawDocumentsCreated, RawDocumentsReused: counts.RawDocumentsReused,
		EventSourcesCreated: counts.EventSourcesCreated, EventSourcesReused: counts.EventSourcesReused,
		EventTagsCreated: counts.EventTagsCreated, EventTagsReused: counts.EventTagsReused,
	}
	return EventPublicationReceipt{
		ID: receiptID, PackageID: publication.PackageID, CallerSubject: callerSubject,
		ExtractorExecutionID:  publication.Provenance.ExtractorExecutionID,
		ExtractorAgentVersion: publication.Provenance.ExtractorAgentVersion,
		CollectorExecutions:   collectorExecutions,
		EventIDs:              eventIDs, RawDocumentIDs: rawIDs, EventSourceIDs: sourceIDs,
		EventTagMapIDs: tagIDs, ReviewMetadata: reviews, WriteCounts: writeCounts,
		ImportedAt: importedAt,
	}
}

func sameRawDocument(left, right StoredEventEvidenceRecord) bool {
	return left.ArtifactID == right.ArtifactID &&
		left.ContentSHA256 == right.ContentSHA256 &&
		left.SourceRef == right.SourceRef &&
		left.SourceName == right.SourceName &&
		left.SourceType == right.SourceType &&
		left.SourceURL == right.SourceURL &&
		left.Title == right.Title &&
		sameOptionalTime(left.PublishedAt, right.PublishedAt) &&
		left.CollectedAt.Equal(right.CollectedAt) &&
		left.Language == right.Language &&
		left.MIMEType == right.MIMEType
}

func sameEventCore(left, right StoredEvent) bool {
	return left.Title == right.Title &&
		left.FactualSummary == right.FactualSummary &&
		sameOptionalTime(left.OccurredAt, right.OccurredAt) &&
		SemanticJSONEqual(left.FactPayload, right.FactPayload) &&
		left.EventStatus == EventStatusConfirmed &&
		left.FactStatus == FactStatusVerified
}

func sameEventSource(left, right StoredEventEvidenceLink) bool {
	return left.SourceLevel == right.SourceLevel &&
		left.EvidenceStatement == right.EvidenceStatement &&
		left.EvidenceHash == right.EvidenceHash &&
		left.EvidenceRelation == right.EvidenceRelation &&
		sameStrings(left.SupportsFields, right.SupportsFields)
}

func sameEventTag(left, right StoredEventTagAssignment) bool {
	return left.AssignSource == right.AssignSource &&
		left.ReviewStatus == ReviewStatusApproved &&
		sameNumber(left.Confidence, right.Confidence) &&
		left.AssignmentReason == right.AssignmentReason
}

func sameOptionalTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	counts := make(map[string]int, len(left))
	for _, value := range left {
		counts[value]++
	}
	for _, value := range right {
		counts[value]--
		if counts[value] < 0 {
			return false
		}
	}
	return true
}

func sameNumber(left, right string) bool {
	leftRat, leftOK := new(big.Rat).SetString(left)
	rightRat, rightOK := new(big.Rat).SetString(right)
	return leftOK && rightOK && leftRat.Cmp(rightRat) == 0
}

func hashEvidenceStatement(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func randomUUID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		value[0:4], value[4:6], value[6:8], value[8:10], value[10:16],
	), nil
}

type EventTagCatalog struct {
	Revision string
	Hash     string
	Tags     []EventTag
}

type EventListRequest struct {
	Title         string
	EventStatus   EventStatus
	FactStatus    FactStatus
	EventTimeFrom *time.Time
	EventTimeTo   *time.Time
	FirstSeenFrom *time.Time
	FirstSeenTo   *time.Time
	Page          int
	PageSize      int
}

type EventPage struct {
	Items    []EventListItem
	Total    int
	Page     int
	PageSize int
}

type EventListFilter struct {
	Title                      string
	EventStatus                EventStatus
	FactStatus                 FactStatus
	EventTimeFrom, EventTimeTo *time.Time
	FirstSeenFrom, FirstSeenTo *time.Time
	Page, PageSize             int
}

type EventListItem struct {
	ID          string
	Title       string
	Summary     string
	EventTime   *time.Time
	FirstSeenAt time.Time
	KnowableAt  *time.Time
	EventStatus EventStatus
	FactStatus  FactStatus
	DedupeKey   string
}

type EventStorePage struct {
	Items          []EventListItem
	Total          int
	Page, PageSize int
}

func (s *UseCase) ActiveTags(ctx context.Context) (EventTagCatalog, error) {
	if s == nil || s.store == nil {
		return EventTagCatalog{}, errors.New("Event Tag Catalog store is required")
	}
	tags, err := s.store.ListActiveTags(ctx)
	if err != nil {
		return EventTagCatalog{}, fmt.Errorf("list active Event Tags: %w", err)
	}
	if len(tags) == 0 {
		return EventTagCatalog{}, errors.New("active Event Tag Catalog is empty")
	}
	for position := range tags {
		tags[position].ID = strings.TrimSpace(tags[position].ID)
		tags[position].Kind = strings.TrimSpace(tags[position].Kind)
		tags[position].Code = strings.TrimSpace(tags[position].Code)
		tags[position].Name = strings.TrimSpace(tags[position].Name)
		if err := validateEventTag(tags[position]); err != nil {
			return EventTagCatalog{}, fmt.Errorf("invalid Event Tag Catalog row: %w", err)
		}
	}
	sort.Slice(tags, func(left, right int) bool {
		if tags[left].Kind != tags[right].Kind {
			return tags[left].Kind < tags[right].Kind
		}
		if tags[left].Code != tags[right].Code {
			return tags[left].Code < tags[right].Code
		}
		return tags[left].ID < tags[right].ID
	})
	for position := 1; position < len(tags); position++ {
		if tags[position-1].ID == tags[position].ID ||
			(tags[position-1].Kind == tags[position].Kind && tags[position-1].Code == tags[position].Code) {
			return EventTagCatalog{}, errors.New("Event Tag Catalog contains duplicate identity")
		}
	}
	encoded, err := json.Marshal(tags)
	if err != nil {
		return EventTagCatalog{}, fmt.Errorf("encode Event Tag Catalog: %w", err)
	}
	digest := sha256.Sum256(encoded)
	hash := hex.EncodeToString(digest[:])
	return EventTagCatalog{Revision: "event-tags:" + hash, Hash: hash, Tags: tags}, nil
}

func validateEventTag(tag EventTag) error {
	if _, err := uuid.Parse(tag.ID); err != nil {
		return errors.New("Tag ID is invalid")
	}
	if tag.Kind != EventTagKindNewsCategory && tag.Kind != EventTagKindIndexCategory {
		return errors.New("Tag kind is invalid")
	}
	if tag.Code == "" || tag.Name == "" {
		return errors.New("Tag code and name are required")
	}
	if !tag.Active {
		return errors.New("Tag must be active")
	}
	return nil
}

func (s *UseCase) ListEvents(ctx context.Context, request EventListRequest) (EventPage, error) {
	if s == nil || s.store == nil {
		return EventPage{}, errors.New("Event store is required")
	}
	page, err := s.store.ListEvents(ctx, EventListFilter{
		Title: request.Title, EventStatus: request.EventStatus, FactStatus: request.FactStatus,
		EventTimeFrom: request.EventTimeFrom, EventTimeTo: request.EventTimeTo,
		FirstSeenFrom: request.FirstSeenFrom, FirstSeenTo: request.FirstSeenTo,
		Page: request.Page, PageSize: request.PageSize,
	})
	if err != nil {
		return EventPage{}, err
	}
	return EventPage{Items: page.Items, Total: page.Total, Page: page.Page, PageSize: page.PageSize}, nil
}

func validStatus[T comparable](value T, allowed ...T) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
