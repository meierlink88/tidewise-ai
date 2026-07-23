package eventpublication

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

const (
	MinEvents = 1
	MaxEvents = 10
)

var (
	lowerSHA256 = regexp.MustCompile(`^[0-9a-f]{64}$`)
	lowerUUID   = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

type Publication struct {
	PackageID    string        `json:"package_id"`
	Provenance   Provenance    `json:"provenance"`
	RawDocuments []RawDocument `json:"raw_documents"`
	Events       []Event       `json:"events"`
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

type RawDocument struct {
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

type Event struct {
	DedupeKey      string         `json:"dedupe_key"`
	Title          string         `json:"title"`
	FactualSummary string         `json:"factual_summary"`
	OccurredAt     *time.Time     `json:"occurred_at,omitempty"`
	FactPayload    map[string]any `json:"fact_payload"`
	Evidence       []Evidence     `json:"evidence"`
	Tags           []Tag          `json:"tags"`
	Review         Review         `json:"review"`
}

type Evidence struct {
	ArtifactID       string   `json:"artifact_id"`
	EvidenceRelation string   `json:"evidence_relation"`
	EvidenceExcerpt  string   `json:"evidence_excerpt"`
	SupportsFields   []string `json:"supports_fields"`
	SourceLevel      string   `json:"source_level"`
	IsPrimary        bool     `json:"is_primary"`
}

type Tag struct {
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

func DecodeStrict(reader io.Reader) (Publication, error) {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var publication Publication
	if err := decoder.Decode(&publication); err != nil {
		return Publication{}, fmt.Errorf("decode Event Publication V2: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return Publication{}, fmt.Errorf("Event Publication V2 body must contain exactly one JSON object")
		}
		return Publication{}, fmt.Errorf("decode trailing Event Publication V2 data: %w", err)
	}
	return publication, nil
}

func (p Publication) Validate() error {
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

	documents := make(map[string]RawDocument, len(p.RawDocuments))
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
		if err := domain.ValidateFactPayload(event.FactPayload); err != nil {
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

func validateEvidence(issues *[]ValidationIssue, eventPath string, evidence []Evidence, documents map[string]RawDocument, referenced map[string]struct{}) {
	if len(evidence) == 0 {
		addIssue(issues, eventPath+".evidence", "INVALID_COUNT", "each Event must contain at least one evidence item")
		return
	}
	seen := make(map[string]struct{}, len(evidence))
	primaryCount := 0
	allowedFields := map[string]struct{}{
		"title": {}, "factual_summary": {}, "occurred_at": {}, "fact_payload": {},
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
		if item.EvidenceRelation != "supports" && item.EvidenceRelation != "contradicts" && item.EvidenceRelation != "context" {
			addIssue(issues, path+".evidence_relation", "INVALID_ENUM", "evidence_relation must be supports, contradicts, or context")
		}
		addRequired(issues, path+".evidence_excerpt", item.EvidenceExcerpt)
		if item.SourceLevel != "primary" && item.SourceLevel != "secondary" {
			addIssue(issues, path+".source_level", "INVALID_ENUM", "source_level must be primary or secondary")
		}
		if item.IsPrimary {
			primaryCount++
		}
		if (item.EvidenceRelation == "supports" || item.EvidenceRelation == "contradicts") && len(item.SupportsFields) == 0 {
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
	if primaryCount != 1 {
		addIssue(issues, eventPath+".evidence", "INVALID_PRIMARY_EVIDENCE", "each Event must contain exactly one primary evidence item")
	}
}

func validateTags(issues *[]ValidationIssue, eventPath string, tags []Tag) {
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
		case "news_category":
			newsCount++
		case "index_category":
			indexCount++
		default:
			addIssue(issues, path+".tag_kind", "INVALID_ENUM", "tag_kind must be news_category or index_category")
		}
		if _, duplicate := seen[tag.TagID]; duplicate {
			addIssue(issues, path+".tag_id", "DUPLICATE_TAG", "tag_id must be unique in one Event")
		}
		seen[tag.TagID] = struct{}{}
		if tag.AssignSource != "ai" && tag.AssignSource != "rule" {
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
