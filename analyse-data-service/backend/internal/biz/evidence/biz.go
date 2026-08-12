package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

type SourceLevel string
type LayerType string
type IssueCode string

const (
	SourceLevelOfficial SourceLevel = "L1_OFFICIAL"
	SourceLevelWire     SourceLevel = "L2_WIRE"
	SourceLevelMedia    SourceLevel = "L3_MEDIA"
	SourceLevelSocial   SourceLevel = "L4_SOCIAL"

	LayerTypeSingle LayerType = "SINGLE"
	LayerTypeDouble LayerType = "DOUBLE"

	IssueRequired                IssueCode = "REQUIRED"
	IssueTooLong                 IssueCode = "TOO_LONG"
	IssueInvalidEnum             IssueCode = "INVALID_ENUM"
	IssueInvalidURL              IssueCode = "INVALID_URL"
	IssueInvalidOrigin           IssueCode = "INVALID_ORIGIN"
	IssueInvalidTimestamp        IssueCode = "INVALID_TIMESTAMP"
	IssueDuplicate               IssueCode = "DUPLICATE"
	IssueOutOfRange              IssueCode = "OUT_OF_RANGE"
	IssueInvalidLayer            IssueCode = "INVALID_LAYER"
	IssueNonContinuousSplitOrder IssueCode = "NON_CONTINUOUS_SPLIT_ORDER"
	IssueRawEvidenceConflict     IssueCode = "RAW_EVIDENCE_CONFLICT"
	IssueRawEvidenceNotFound     IssueCode = "RAW_EVIDENCE_NOT_FOUND"
	IssueEvidenceIDConflict      IssueCode = "EVIDENCE_ID_CONFLICT"
	IssueEvidenceSetConflict     IssueCode = "EVIDENCE_SET_CONFLICT"
)

type RawEvidence struct {
	RawEvidenceID    string
	SourceID         string
	SourceName       string
	SourceLevel      SourceLevel
	SourceURL        string
	IsOriginal       bool
	QuotedSourceID   *string
	QuotedSourceName *string
	Title            *string
	RawText          string
	PublishedAt      *time.Time
	CollectedAt      time.Time
	Keywords         []string
}

type StoredRawEvidence struct {
	RawEvidence
	ContentHash string
}

type Evidence struct {
	EvidenceID            string
	SplitOrder            int
	LayerType             LayerType
	SourceWho             *string
	SourceWhat            string
	SourceWhen            *time.Time
	SourceWhenRaw         *string
	SourceWhere           *string
	SourceWhy             *string
	SourceHow             *string
	SourceWhoCore         *string
	SourceWhatCore        *string
	SourceWhenCore        *time.Time
	SourceWhenRawCore     *string
	SourceWhereCore       *string
	SourceWhyCore         *string
	SourceHowCore         *string
	ExpressionFingerprint string
	ExpressionKey         string
	FingerprintVersion    string
}

type StoredEvidence struct {
	Evidence
	RawEvidenceID string
	IsSplit       bool
}

type RawEvidenceResult struct {
	RawEvidenceID string
}

type EvidenceResult struct {
	RawEvidenceID string
	EvidenceIDs   []string
}

type Issue struct {
	Path    string    `json:"path"`
	Code    IssueCode `json:"code"`
	Message string    `json:"message"`
}

type ValidationError struct{ Issues []Issue }

func (e *ValidationError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "Evidence Publication failed validation"
	}
	return e.Issues[0].Path + ": " + e.Issues[0].Message
}

type ConflictError struct{ Issues []Issue }

func (e *ConflictError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "Evidence Publication conflicts with stored data"
	}
	return e.Issues[0].Path + ": " + e.Issues[0].Message
}

type ReferenceError struct{ Issues []Issue }

func (e *ReferenceError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "Evidence Publication references unavailable data"
	}
	return e.Issues[0].Path + ": " + e.Issues[0].Message
}

var allowedSourceLevels = map[SourceLevel]struct{}{
	SourceLevelOfficial: {},
	SourceLevelWire:     {},
	SourceLevelMedia:    {},
	SourceLevelSocial:   {},
}

type UseCase struct {
	store Store
}

func NewUseCase(store Store) (*UseCase, error) {
	if store == nil {
		return nil, errors.New("Evidence Publication store is required")
	}
	return &UseCase{store: store}, nil
}

func (s *UseCase) PublishRawEvidence(ctx context.Context, input RawEvidence) (RawEvidenceResult, error) {
	if s == nil || s.store == nil {
		return RawEvidenceResult{}, errors.New("Evidence Publication store is required")
	}
	if err := validateRawEvidence(input); err != nil {
		return RawEvidenceResult{}, err
	}

	record := StoredRawEvidence{RawEvidence: cloneRawEvidence(input), ContentHash: contentHash(input.RawText)}
	var result RawEvidenceResult
	err := s.store.InTransaction(ctx, func(tx Transaction) error {
		if err := tx.LockIdentities(ctx, []string{"raw-evidence:" + input.RawEvidenceID}); err != nil {
			return err
		}
		existing, err := tx.RawEvidence(ctx, input.RawEvidenceID)
		if err != nil {
			return err
		}
		if existing != nil {
			if !sameRawEvidence(*existing, record) {
				return &ConflictError{Issues: []Issue{{
					Path: "raw_evidence.raw_evidence_id", Code: IssueRawEvidenceConflict,
					Message: "raw_evidence_id conflicts with stored content",
				}}}
			}
		} else if err := tx.InsertRawEvidence(ctx, record); err != nil {
			return err
		}
		result = RawEvidenceResult{RawEvidenceID: record.RawEvidenceID}
		return nil
	})
	if err != nil {
		return RawEvidenceResult{}, err
	}
	return result, nil
}

func (s *UseCase) PublishEvidence(ctx context.Context, rawEvidenceID string, input []Evidence) (EvidenceResult, error) {
	if s == nil || s.store == nil {
		return EvidenceResult{}, errors.New("Evidence Publication store is required")
	}
	if err := validateEvidencePublication(rawEvidenceID, input); err != nil {
		return EvidenceResult{}, err
	}

	isSplit := len(input) > 1
	records := make([]StoredEvidence, len(input))
	for index, item := range input {
		records[index] = StoredEvidence{Evidence: cloneEvidence(item), RawEvidenceID: rawEvidenceID, IsSplit: isSplit}
	}
	sort.Slice(records, func(i, j int) bool { return records[i].SplitOrder < records[j].SplitOrder })
	ids := make([]string, len(records))
	for index, record := range records {
		ids[index] = record.EvidenceID
	}
	locks := make([]string, 0, len(ids)+1)
	locks = append(locks, "raw-evidence:"+rawEvidenceID)
	for _, id := range ids {
		locks = append(locks, "evidence:"+id)
	}
	sort.Strings(locks)

	var result EvidenceResult
	err := s.store.InTransaction(ctx, func(tx Transaction) error {
		if err := tx.LockIdentities(ctx, locks); err != nil {
			return err
		}
		raw, err := tx.RawEvidence(ctx, rawEvidenceID)
		if err != nil {
			return err
		}
		if raw == nil {
			return &ReferenceError{Issues: []Issue{{
				Path: "raw_evidence_id", Code: IssueRawEvidenceNotFound,
				Message: "raw_evidence_id does not reference a published Raw Evidence",
			}}}
		}

		existing, err := tx.EvidencesByRawEvidence(ctx, rawEvidenceID)
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			if !sameEvidenceSet(existing, records) {
				return evidenceSetConflict()
			}
		} else {
			collisions, err := tx.EvidencesByIDs(ctx, ids)
			if err != nil {
				return err
			}
			if len(collisions) > 0 {
				return &ConflictError{Issues: []Issue{{
					Path: "evidences", Code: IssueEvidenceIDConflict,
					Message: "an evidence_id is already assigned to different content",
				}}}
			}
			for _, record := range records {
				if err := tx.InsertEvidence(ctx, record); err != nil {
					return err
				}
			}
		}

		result = EvidenceResult{RawEvidenceID: rawEvidenceID, EvidenceIDs: append([]string(nil), ids...)}
		return nil
	})
	if err != nil {
		return EvidenceResult{}, err
	}
	return result, nil
}

func validateEvidencePublication(rawEvidenceID string, input []Evidence) error {
	var issues []Issue
	required(&issues, "raw_evidence_id", rawEvidenceID, 32)
	if len(input) == 0 {
		issues = append(issues, Issue{Path: "evidences", Code: IssueRequired, Message: "at least one Evidence is required"})
	}
	seenIDs := make(map[string]struct{}, len(input))
	seenOrders := make(map[int]struct{}, len(input))
	for index, item := range input {
		prefix := fmt.Sprintf("evidences[%d]", index)
		required(&issues, prefix+".evidence_id", item.EvidenceID, 32)
		if _, ok := seenIDs[item.EvidenceID]; ok {
			issues = append(issues, Issue{Path: prefix + ".evidence_id", Code: IssueDuplicate, Message: "evidence_id must be unique within the publication"})
		}
		seenIDs[item.EvidenceID] = struct{}{}
		if item.SplitOrder < 0 {
			issues = append(issues, Issue{Path: prefix + ".split_order", Code: IssueOutOfRange, Message: "split_order must be non-negative"})
		}
		if _, ok := seenOrders[item.SplitOrder]; ok {
			issues = append(issues, Issue{Path: prefix + ".split_order", Code: IssueDuplicate, Message: "split_order must be unique within the publication"})
		}
		seenOrders[item.SplitOrder] = struct{}{}
		required(&issues, prefix+".source_what", item.SourceWhat, 0)
		required(&issues, prefix+".expression_fingerprint", item.ExpressionFingerprint, 200)
		required(&issues, prefix+".expression_key", item.ExpressionKey, 64)
		required(&issues, prefix+".fingerprint_version", item.FingerprintVersion, 64)
		validateOptionalEvidenceFields(&issues, prefix, item)
		switch item.LayerType {
		case LayerTypeSingle:
			if hasCoreFields(item) {
				issues = append(issues, Issue{Path: prefix + ".layer_type", Code: IssueInvalidLayer, Message: "SINGLE Evidence cannot declare core fields"})
			}
		case LayerTypeDouble:
			if item.SourceWhatCore == nil || strings.TrimSpace(*item.SourceWhatCore) == "" {
				issues = append(issues, Issue{Path: prefix + ".source_what_core", Code: IssueRequired, Message: "DOUBLE Evidence requires source_what_core"})
			}
		default:
			issues = append(issues, Issue{Path: prefix + ".layer_type", Code: IssueInvalidEnum, Message: "layer_type is invalid"})
		}
	}
	for expected := 0; expected < len(input); expected++ {
		if _, ok := seenOrders[expected]; !ok {
			issues = append(issues, Issue{Path: "evidences", Code: IssueNonContinuousSplitOrder, Message: "split_order must be continuous from zero"})
			break
		}
	}
	if len(issues) == 0 {
		return nil
	}
	sortIssues(issues)
	return &ValidationError{Issues: issues}
}

func validateOptionalEvidenceFields(issues *[]Issue, prefix string, item Evidence) {
	if item.SourceWhen != nil && !isUTC(*item.SourceWhen) {
		*issues = append(*issues, Issue{Path: prefix + ".source_when", Code: IssueInvalidTimestamp, Message: "source_when must use UTC"})
	}
	if item.SourceWhenCore != nil && !isUTC(*item.SourceWhenCore) {
		*issues = append(*issues, Issue{Path: prefix + ".source_when_core", Code: IssueInvalidTimestamp, Message: "source_when_core must use UTC"})
	}
}

func hasCoreFields(item Evidence) bool {
	return item.SourceWhoCore != nil || item.SourceWhatCore != nil || item.SourceWhenCore != nil ||
		item.SourceWhenRawCore != nil || item.SourceWhereCore != nil || item.SourceWhyCore != nil || item.SourceHowCore != nil
}

func evidenceSetConflict() error {
	return &ConflictError{Issues: []Issue{{
		Path: "evidences", Code: IssueEvidenceSetConflict,
		Message: "Raw Evidence already has a different immutable Evidence set",
	}}}
}

func validateRawEvidence(input RawEvidence) error {
	var issues []Issue
	required(&issues, "raw_evidence.raw_evidence_id", input.RawEvidenceID, 32)
	required(&issues, "raw_evidence.source_id", input.SourceID, 32)
	required(&issues, "raw_evidence.source_name", input.SourceName, 100)
	required(&issues, "raw_evidence.source_url", input.SourceURL, 0)
	required(&issues, "raw_evidence.raw_text", input.RawText, 0)
	if _, ok := allowedSourceLevels[input.SourceLevel]; !ok {
		issues = append(issues, Issue{Path: "raw_evidence.source_level", Code: IssueInvalidEnum, Message: "source_level is invalid"})
	}
	if parsed, err := url.Parse(input.SourceURL); err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		issues = append(issues, Issue{Path: "raw_evidence.source_url", Code: IssueInvalidURL, Message: "source_url must be an absolute HTTP(S) URL"})
	}
	optional(&issues, "raw_evidence.quoted_source_id", input.QuotedSourceID, 32)
	optional(&issues, "raw_evidence.quoted_source_name", input.QuotedSourceName, 100)
	optional(&issues, "raw_evidence.title", input.Title, 500)
	if input.IsOriginal && (input.QuotedSourceID != nil || input.QuotedSourceName != nil) {
		issues = append(issues, Issue{Path: "raw_evidence.is_original", Code: IssueInvalidOrigin, Message: "original content cannot declare a quoted source"})
	}
	if !input.IsOriginal && (input.QuotedSourceName == nil || strings.TrimSpace(*input.QuotedSourceName) == "") {
		issues = append(issues, Issue{Path: "raw_evidence.quoted_source_name", Code: IssueRequired, Message: "reposted content requires quoted_source_name"})
	}
	if input.CollectedAt.IsZero() || !isUTC(input.CollectedAt) {
		issues = append(issues, Issue{Path: "raw_evidence.collected_at", Code: IssueInvalidTimestamp, Message: "collected_at must be a UTC timestamp"})
	}
	if input.PublishedAt != nil && !isUTC(*input.PublishedAt) {
		issues = append(issues, Issue{Path: "raw_evidence.published_at", Code: IssueInvalidTimestamp, Message: "published_at must use UTC"})
	}
	if len(issues) == 0 {
		return nil
	}
	sortIssues(issues)
	return &ValidationError{Issues: issues}
}

func required(issues *[]Issue, path, value string, max int) {
	length := len([]rune(value))
	if strings.TrimSpace(value) == "" {
		*issues = append(*issues, Issue{Path: path, Code: IssueRequired, Message: "value is required"})
		return
	}
	if max > 0 && length > max {
		*issues = append(*issues, Issue{Path: path, Code: IssueTooLong, Message: fmt.Sprintf("value must contain at most %d characters", max)})
	}
}

func optional(issues *[]Issue, path string, value *string, max int) {
	if value == nil {
		return
	}
	required(issues, path, *value, max)
}

func sortIssues(issues []Issue) {
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Path != issues[j].Path {
			return issues[i].Path < issues[j].Path
		}
		return issues[i].Code < issues[j].Code
	})
}

func isUTC(value time.Time) bool {
	_, offset := value.Zone()
	return offset == 0
}

func contentHash(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func cloneRawEvidence(input RawEvidence) RawEvidence {
	input.QuotedSourceID = cloneString(input.QuotedSourceID)
	input.QuotedSourceName = cloneString(input.QuotedSourceName)
	input.Title = cloneString(input.Title)
	input.PublishedAt = cloneTime(input.PublishedAt)
	if input.PublishedAt != nil {
		*input.PublishedAt = normalizePostgresTime(*input.PublishedAt)
	}
	input.CollectedAt = normalizePostgresTime(input.CollectedAt)
	input.Keywords = append([]string(nil), input.Keywords...)
	return input
}

func cloneEvidence(input Evidence) Evidence {
	input.SourceWho = cloneString(input.SourceWho)
	input.SourceWhen = cloneTime(input.SourceWhen)
	input.SourceWhenRaw = cloneString(input.SourceWhenRaw)
	input.SourceWhere = cloneString(input.SourceWhere)
	input.SourceWhy = cloneString(input.SourceWhy)
	input.SourceHow = cloneString(input.SourceHow)
	input.SourceWhoCore = cloneString(input.SourceWhoCore)
	input.SourceWhatCore = cloneString(input.SourceWhatCore)
	input.SourceWhenCore = cloneTime(input.SourceWhenCore)
	if input.SourceWhen != nil {
		*input.SourceWhen = normalizePostgresTime(*input.SourceWhen)
	}
	if input.SourceWhenCore != nil {
		*input.SourceWhenCore = normalizePostgresTime(*input.SourceWhenCore)
	}
	input.SourceWhenRawCore = cloneString(input.SourceWhenRawCore)
	input.SourceWhereCore = cloneString(input.SourceWhereCore)
	input.SourceWhyCore = cloneString(input.SourceWhyCore)
	input.SourceHowCore = cloneString(input.SourceHowCore)
	return input
}

func normalizePostgresTime(value time.Time) time.Time {
	return value.UTC().Truncate(time.Microsecond)
}

func cloneString(input *string) *string {
	if input == nil {
		return nil
	}
	value := *input
	return &value
}

func cloneTime(input *time.Time) *time.Time {
	if input == nil {
		return nil
	}
	value := *input
	return &value
}

func sameRawEvidence(left, right StoredRawEvidence) bool {
	return left.RawEvidenceID == right.RawEvidenceID && left.SourceID == right.SourceID &&
		left.SourceName == right.SourceName && left.SourceLevel == right.SourceLevel &&
		left.SourceURL == right.SourceURL && left.IsOriginal == right.IsOriginal &&
		sameString(left.QuotedSourceID, right.QuotedSourceID) &&
		sameString(left.QuotedSourceName, right.QuotedSourceName) &&
		sameString(left.Title, right.Title) && left.RawText == right.RawText &&
		sameTime(left.PublishedAt, right.PublishedAt) && left.CollectedAt.Equal(right.CollectedAt) &&
		left.ContentHash == right.ContentHash && equalStringSlice(left.Keywords, right.Keywords)
}

func sameEvidenceSet(left, right []StoredEvidence) bool {
	if len(left) != len(right) {
		return false
	}
	leftByOrder := make(map[int]StoredEvidence, len(left))
	for _, record := range left {
		leftByOrder[record.SplitOrder] = record
	}
	for _, record := range right {
		existing, ok := leftByOrder[record.SplitOrder]
		if !ok || !sameEvidence(existing, record) {
			return false
		}
	}
	return true
}

func sameEvidence(left, right StoredEvidence) bool {
	return left.EvidenceID == right.EvidenceID && left.RawEvidenceID == right.RawEvidenceID &&
		left.SplitOrder == right.SplitOrder && left.IsSplit == right.IsSplit && left.LayerType == right.LayerType &&
		sameString(left.SourceWho, right.SourceWho) && left.SourceWhat == right.SourceWhat &&
		sameTime(left.SourceWhen, right.SourceWhen) && sameString(left.SourceWhenRaw, right.SourceWhenRaw) &&
		sameString(left.SourceWhere, right.SourceWhere) && sameString(left.SourceWhy, right.SourceWhy) &&
		sameString(left.SourceHow, right.SourceHow) && sameString(left.SourceWhoCore, right.SourceWhoCore) &&
		sameString(left.SourceWhatCore, right.SourceWhatCore) && sameTime(left.SourceWhenCore, right.SourceWhenCore) &&
		sameString(left.SourceWhenRawCore, right.SourceWhenRawCore) && sameString(left.SourceWhereCore, right.SourceWhereCore) &&
		sameString(left.SourceWhyCore, right.SourceWhyCore) && sameString(left.SourceHowCore, right.SourceHowCore) &&
		left.ExpressionFingerprint == right.ExpressionFingerprint && left.ExpressionKey == right.ExpressionKey &&
		left.FingerprintVersion == right.FingerprintVersion
}

func sameString(left, right *string) bool {
	return left == nil && right == nil || left != nil && right != nil && *left == *right
}

func sameTime(left, right *time.Time) bool {
	return left == nil && right == nil || left != nil && right != nil && left.Equal(*right)
}

func equalStringSlice(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
