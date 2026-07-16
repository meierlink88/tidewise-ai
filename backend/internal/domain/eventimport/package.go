package eventimport

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Package struct {
	IdempotencyKey string             `json:"idempotency_key"`
	PackageID      string             `json:"package_id"`
	RawDocuments   []RawDocumentInput `json:"raw_documents"`
	Event          EventInput         `json:"event"`
	EventSources   []EventSourceInput `json:"event_sources"`
	EventTags      []EventTagInput    `json:"event_tags"`
	Review         ReviewInput        `json:"review"`
	raw            []byte
}

type RawDocumentInput struct {
	DocumentID   string     `json:"document_id"`
	SourceName   string     `json:"source_name"`
	SourceURL    string     `json:"source_url"`
	Title        string     `json:"title"`
	ContentText  string     `json:"content_text"`
	ContentLevel string     `json:"content_level"`
	PublishedAt  *time.Time `json:"published_at"`
	CollectedAt  time.Time  `json:"collected_at"`
	ContentHash  string     `json:"content_hash"`
}

type EventInput struct {
	DedupeKey      string         `json:"dedupe_key"`
	Title          string         `json:"title"`
	FactualSummary string         `json:"factual_summary"`
	OccurredAt     *time.Time     `json:"occurred_at"`
	FactStatus     string         `json:"fact_status"`
	EventStatus    string         `json:"event_status"`
	FactPayload    map[string]any `json:"fact_payload"`
}

type EventSourceInput struct {
	DocumentID       string   `json:"document_id"`
	EvidenceExcerpt  string   `json:"evidence_excerpt"`
	SourceURL        string   `json:"source_url"`
	EvidenceRelation string   `json:"evidence_relation"`
	SupportsFields   []string `json:"supports_fields"`
	SourceLevel      string   `json:"source_level"`
	ContentLevel     string   `json:"content_level"`
	EvidenceHash     string   `json:"evidence_hash"`
}

type EventTagInput struct {
	TagID            string      `json:"tag_id"`
	TagKind          string      `json:"tag_kind"`
	TagCode          string      `json:"tag_code"`
	Confidence       json.Number `json:"confidence"`
	ReviewStatus     string      `json:"review_status"`
	AssignmentReason string      `json:"assignment_reason"`
	AssignSource     string      `json:"assign_source"`
}

type ReviewInput struct {
	ReviewID          string            `json:"review_id"`
	PackageID         string            `json:"package_id"`
	Decision          string            `json:"decision"`
	EventStatus       string            `json:"event_status"`
	FactStatus        string            `json:"fact_status"`
	EvidenceGrade     string            `json:"evidence_grade"`
	Reasons           []string          `json:"reasons"`
	ComponentVersions map[string]string `json:"component_versions"`
}

type Mapping struct {
	EventStatus string
	FactStatus  string
	FirstSeenAt time.Time
	KnowableAt  time.Time
}

func DecodeStrict(reader io.Reader) (Package, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return Package{}, fmt.Errorf("read reviewed outbox: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var pkg Package
	if err := decoder.Decode(&pkg); err != nil {
		return Package{}, fmt.Errorf("decode reviewed outbox: %w", err)
	}
	if err := requireEOF(decoder); err != nil {
		return Package{}, err
	}
	if strings.TrimSpace(pkg.PackageID) == "" || strings.TrimSpace(pkg.Review.PackageID) == "" {
		return Package{}, fmt.Errorf("package_id and review.package_id are required")
	}
	if len(pkg.EventTags) == 0 {
		return Package{}, fmt.Errorf("event_tags must include at least one news_category tag")
	}
	pkg.raw = append([]byte(nil), content...)
	return pkg, nil
}

func (p Package) CanonicalHash() (string, error) {
	if len(p.raw) > 0 {
		return CanonicalHash(p.raw)
	}
	content, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return CanonicalHash(content)
}

func requireEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode trailing reviewed outbox data: %w", err)
	}
	return fmt.Errorf("reviewed outbox must contain exactly one JSON object")
}

func (p Package) Validate() (Mapping, error) {
	if strings.TrimSpace(p.IdempotencyKey) == "" {
		return Mapping{}, fmt.Errorf("idempotency_key is required")
	}
	if p.Review.PackageID != p.PackageID {
		return Mapping{}, fmt.Errorf("review.package_id must equal package_id")
	}
	if strings.TrimSpace(p.Review.ReviewID) == "" || strings.TrimSpace(p.Review.EvidenceGrade) == "" {
		return Mapping{}, fmt.Errorf("review_id and evidence_grade are required")
	}
	if len(p.Review.Reasons) == 0 || len(p.Review.ComponentVersions) == 0 {
		return Mapping{}, fmt.Errorf("review reasons and component_versions are required")
	}

	eventStatus, factStatus, err := reviewStatuses(p.Review.Decision)
	if err != nil {
		return Mapping{}, err
	}
	if p.Review.EventStatus != eventStatus || p.Event.EventStatus != eventStatus || p.Review.FactStatus != factStatus || p.Event.FactStatus != factStatus {
		return Mapping{}, fmt.Errorf("review decision/status mapping mismatch")
	}
	if strings.TrimSpace(p.Event.DedupeKey) == "" || strings.TrimSpace(p.Event.Title) == "" || strings.TrimSpace(p.Event.FactualSummary) == "" || p.Event.FactPayload == nil {
		return Mapping{}, fmt.Errorf("event dedupe_key, title, factual_summary and fact_payload are required")
	}

	documents := make(map[string]RawDocumentInput, len(p.RawDocuments))
	var firstSeen time.Time
	var knowable time.Time
	for index, document := range p.RawDocuments {
		if strings.TrimSpace(document.DocumentID) == "" || strings.TrimSpace(document.SourceName) == "" || strings.TrimSpace(document.Title) == "" || strings.TrimSpace(document.ContentHash) == "" || strings.TrimSpace(document.ContentLevel) == "" || document.CollectedAt.IsZero() {
			return Mapping{}, fmt.Errorf("raw_documents[%d] is incomplete", index)
		}
		if _, duplicate := documents[document.DocumentID]; duplicate {
			return Mapping{}, fmt.Errorf("duplicate raw document %q", document.DocumentID)
		}
		documents[document.DocumentID] = document
		firstSeen = earliest(firstSeen, document.CollectedAt)
		candidate := document.CollectedAt
		if document.PublishedAt != nil {
			candidate = *document.PublishedAt
		}
		knowable = earliest(knowable, candidate)
	}
	if len(documents) == 0 {
		return Mapping{}, fmt.Errorf("at least one raw document is required")
	}

	if len(p.EventSources) == 0 {
		return Mapping{}, fmt.Errorf("at least one event source is required")
	}
	seenEvidence := map[string]struct{}{}
	for index, source := range p.EventSources {
		document, exists := documents[source.DocumentID]
		if !exists {
			return Mapping{}, fmt.Errorf("event_sources[%d].document_id is unknown", index)
		}
		relation := source.EvidenceRelation
		if relation == "" {
			relation = "supports"
		}
		if relation != "supports" && relation != "contradicts" && relation != "context" {
			return Mapping{}, fmt.Errorf("event_sources[%d].evidence_relation is invalid", index)
		}
		if strings.TrimSpace(source.SourceLevel) == "" || strings.TrimSpace(source.ContentLevel) == "" || strings.TrimSpace(source.EvidenceHash) == "" || strings.TrimSpace(source.EvidenceExcerpt) == "" {
			return Mapping{}, fmt.Errorf("event_sources[%d] evidence metadata is incomplete", index)
		}
		if source.ContentLevel != document.ContentLevel {
			return Mapping{}, fmt.Errorf("event_sources[%d].content_level does not match raw document", index)
		}
		if (relation == "supports" || relation == "contradicts") && !nonBlankStrings(source.SupportsFields) {
			return Mapping{}, fmt.Errorf("event_sources[%d].supports_fields must be non-empty", index)
		}
		key := source.DocumentID + "\x00" + source.EvidenceHash
		if _, duplicate := seenEvidence[key]; duplicate {
			return Mapping{}, fmt.Errorf("duplicate event source evidence")
		}
		seenEvidence[key] = struct{}{}
	}

	newsCount := 0
	indexCount := 0
	seenTags := map[string]struct{}{}
	for index, tag := range p.EventTags {
		if strings.TrimSpace(tag.TagID) == "" || strings.TrimSpace(tag.TagCode) == "" || strings.TrimSpace(tag.AssignmentReason) == "" {
			return Mapping{}, fmt.Errorf("event_tags[%d] is incomplete", index)
		}
		if tag.AssignSource != "ai" && tag.AssignSource != "rule" {
			return Mapping{}, fmt.Errorf("event_tags[%d].assign_source is invalid", index)
		}
		if tag.ReviewStatus != "approved" && tag.ReviewStatus != "pending" && tag.ReviewStatus != "rejected" {
			return Mapping{}, fmt.Errorf("event_tags[%d].review_status is invalid", index)
		}
		confidence, ok := new(big.Rat).SetString(string(tag.Confidence))
		if !ok || confidence.Sign() < 0 || confidence.Cmp(big.NewRat(1, 1)) > 0 {
			return Mapping{}, fmt.Errorf("event_tags[%d].confidence must be between 0 and 1", index)
		}
		key := tag.TagKind + "\x00" + tag.TagCode
		if _, duplicate := seenTags[key]; duplicate {
			return Mapping{}, fmt.Errorf("duplicate event tag %q", tag.TagCode)
		}
		seenTags[key] = struct{}{}
		if _, err := LookupFrozenTag(tag.TagKind, tag.TagID, tag.TagCode); err != nil {
			return Mapping{}, fmt.Errorf("event_tags[%d]: %w", index, err)
		}
		switch tag.TagKind {
		case "news_category":
			newsCount++
		case "index_category":
			indexCount++
		default:
			return Mapping{}, fmt.Errorf("event_tags[%d].tag_kind is invalid", index)
		}
	}
	if newsCount < 1 || newsCount > 2 || indexCount > 3 {
		return Mapping{}, fmt.Errorf("event tag counts must be news_category=1..2 and index_category=0..3")
	}

	return Mapping{EventStatus: eventStatus, FactStatus: factStatus, FirstSeenAt: firstSeen, KnowableAt: knowable}, nil
}

func reviewStatuses(decision string) (string, string, error) {
	switch decision {
	case "auto_approved":
		return "confirmed", "verified", nil
	case "pending_evidence", "manual_review":
		return "candidate", "unverified", nil
	case "rejected":
		return "", "", fmt.Errorf("rejected review cannot be imported")
	default:
		return "", "", fmt.Errorf("unsupported review decision %q", decision)
	}
}

func earliest(current, candidate time.Time) time.Time {
	if current.IsZero() || candidate.Before(current) {
		return candidate
	}
	return current
}

func nonBlankStrings(values []string) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return false
		}
	}
	return true
}

func CanonicalHash(payload []byte) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "", fmt.Errorf("decode canonical JSON: %w", err)
	}
	if err := requireEOF(decoder); err != nil {
		return "", err
	}
	var canonical bytes.Buffer
	if err := writeCanonicalJSON(&canonical, value); err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical.Bytes())
	return hex.EncodeToString(sum[:]), nil
}

func writeCanonicalJSON(writer *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case nil:
		writer.WriteString("null")
	case bool:
		writer.WriteString(strconv.FormatBool(typed))
	case string:
		encoded, _ := json.Marshal(typed)
		writer.Write(encoded)
	case json.Number:
		normalized, err := normalizeJSONNumber(string(typed))
		if err != nil {
			return err
		}
		writer.WriteString(normalized)
	case []any:
		writer.WriteByte('[')
		for index, item := range typed {
			if index > 0 {
				writer.WriteByte(',')
			}
			if err := writeCanonicalJSON(writer, item); err != nil {
				return err
			}
		}
		writer.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		writer.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				writer.WriteByte(',')
			}
			encoded, _ := json.Marshal(key)
			writer.Write(encoded)
			writer.WriteByte(':')
			if err := writeCanonicalJSON(writer, typed[key]); err != nil {
				return err
			}
		}
		writer.WriteByte('}')
	default:
		return fmt.Errorf("unsupported canonical JSON value %T", value)
	}
	return nil
}

func normalizeJSONNumber(value string) (string, error) {
	negative := strings.HasPrefix(value, "-")
	unsigned := strings.TrimPrefix(value, "-")
	parts := strings.FieldsFunc(unsigned, func(r rune) bool { return r == 'e' || r == 'E' })
	if len(parts) > 2 {
		return "", fmt.Errorf("invalid JSON number %q", value)
	}
	exponent := 0
	if len(parts) == 2 {
		parsed, err := strconv.Atoi(parts[1])
		if err != nil {
			return "", fmt.Errorf("invalid JSON number %q", value)
		}
		exponent = parsed
	}
	mantissa := strings.Split(parts[0], ".")
	digits := strings.Join(mantissa, "")
	fractionDigits := 0
	if len(mantissa) == 2 {
		fractionDigits = len(mantissa[1])
	}
	digits = strings.TrimLeft(digits, "0")
	if digits == "" {
		return "0", nil
	}
	scale := exponent - fractionDigits
	for strings.HasSuffix(digits, "0") {
		digits = strings.TrimSuffix(digits, "0")
		scale++
	}

	plain := digits
	if scale >= 0 {
		plain += strings.Repeat("0", scale)
	} else if point := len(digits) + scale; point > 0 {
		plain = digits[:point] + "." + digits[point:]
	} else {
		plain = "0." + strings.Repeat("0", -point) + digits
	}
	scientific := digits[:1]
	if len(digits) > 1 {
		scientific += "." + digits[1:]
	}
	scientific += "e" + strconv.Itoa(scale+len(digits)-1)
	if len(scientific) < len(plain) {
		plain = scientific
	}
	if negative {
		plain = "-" + plain
	}
	return plain, nil
}
