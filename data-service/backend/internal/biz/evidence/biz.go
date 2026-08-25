package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

type SourceLevel string
type IssueCode string
type CategoryID string

const (
	RawEvidenceIDPrefix = coreid.RawEvidence
	EvidenceIDPrefix    = coreid.Evidence
	CategoryIDPrefix    = coreid.EvidenceCategory
)

const (
	SourceLevelOfficial SourceLevel = "L1_OFFICIAL"
	SourceLevelWire     SourceLevel = "L2_WIRE"
	SourceLevelMedia    SourceLevel = "L3_MEDIA"
	SourceLevelSocial   SourceLevel = "L4_SOCIAL"

	IssueRequired            IssueCode = "REQUIRED"
	IssueTooLong             IssueCode = "TOO_LONG"
	IssueInvalidEnum         IssueCode = "INVALID_ENUM"
	IssueInvalidURL          IssueCode = "INVALID_URL"
	IssueInvalidOrigin       IssueCode = "INVALID_ORIGIN"
	IssueInvalidTimestamp    IssueCode = "INVALID_TIMESTAMP"
	IssueInvalidFormat       IssueCode = "INVALID_FORMAT"
	IssueDuplicate           IssueCode = "DUPLICATE"
	IssueRawEvidenceConflict IssueCode = "RAW_EVIDENCE_CONFLICT"
	IssueRawEvidenceNotFound IssueCode = "RAW_EVIDENCE_NOT_FOUND"
	IssueCategoryNotFound    IssueCode = "EVIDENCE_CATEGORY_NOT_FOUND"
	IssueEvidenceIDConflict  IssueCode = "EVIDENCE_ID_CONFLICT"
	IssueEvidenceSetConflict IssueCode = "EVIDENCE_SET_CONFLICT"
)

type RawEvidence struct {
	ID               string
	PublicationKey   string
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
	CategoryIDs      []CategoryID
}

type StoredRawEvidence struct {
	RawEvidence
	ContentHash string
	Categories  []Category
}

type Category struct {
	ID          CategoryID
	Code        string
	Name        string
	Description string
	CreatedAt   time.Time
}

type CategoryCatalog struct {
	Categories []Category
}

type Store interface {
	TransactionStore
	ListCategories(context.Context) ([]Category, error)
	ListEvidence(context.Context, EvidenceListFilter) (EvidencePage, error)
}

type Evidence struct {
	ID       string
	Summary  string
	Semantic Semantic
}

type Semantic struct {
	Who   *string `json:"who"`
	What  string  `json:"what"`
	When  *string `json:"when"`
	Where *string `json:"where"`
	Why   *string `json:"why"`
	How   *string `json:"how"`
}

type StoredEvidence struct {
	Evidence
	RawEvidenceID string
	IsSplit       bool
}

type EvidenceListFilter struct {
	Title         string
	Summary       string
	CategoryID    CategoryID
	SourceID      string
	SourceName    string
	SourceLevel   SourceLevel
	IsSplit       *bool
	PublishedFrom *time.Time
	PublishedTo   *time.Time
	CollectedFrom *time.Time
	CollectedTo   *time.Time
	Page          int
	PageSize      int
}

type EvidenceListItem struct {
	ID               string
	RawEvidenceID    string
	Title            *string
	Summary          string
	Semantic         Semantic
	Categories       []Category
	SourceID         string
	SourceName       string
	SourceLevel      SourceLevel
	SourceURL        string
	IsOriginal       bool
	QuotedSourceName *string
	Keywords         []string
	IsSplit          bool
	PublishedAt      *time.Time
	CollectedAt      time.Time
}

type EvidencePage struct {
	Items    []EvidenceListItem
	Total    int
	Page     int
	PageSize int
}

type RawEvidenceResult struct {
	ID string
}

type EvidenceResult struct {
	RawEvidenceID string
	IDs           []string
	Items         []EvidenceResultItem
}

type EvidenceResultItem struct {
	InputIndex int
	ID         string
}

type RawEvidenceCategoryLink struct {
	ID         string
	CategoryID CategoryID
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

var ErrRawEvidenceNotFound = errors.New("Raw Evidence was not found")

var categoryCodePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

var allowedSourceLevels = map[SourceLevel]struct{}{
	SourceLevelOfficial: {},
	SourceLevelWire:     {},
	SourceLevelMedia:    {},
	SourceLevelSocial:   {},
}

func (id CategoryID) IsValid() bool {
	return coreid.Is(string(id), CategoryIDPrefix)
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

func (s *UseCase) ListCategories(ctx context.Context) (CategoryCatalog, error) {
	if s == nil || s.store == nil {
		return CategoryCatalog{}, errors.New("Evidence Category Catalog store is required")
	}
	categories, err := s.store.ListCategories(ctx)
	if err != nil {
		return CategoryCatalog{}, fmt.Errorf("list Evidence Categories: %w", err)
	}
	if len(categories) == 0 {
		return CategoryCatalog{}, errors.New("Evidence Category Catalog is empty")
	}
	categories = cloneCategories(categories)
	seenIDs := make(map[CategoryID]struct{}, len(categories))
	seenCodes := make(map[string]struct{}, len(categories))
	for _, category := range categories {
		if !category.ID.IsValid() {
			return CategoryCatalog{}, errors.New("Evidence Category Catalog contains an invalid category ID")
		}
		if !categoryCodePattern.MatchString(category.Code) || len(category.Code) > 50 {
			return CategoryCatalog{}, errors.New("Evidence Category Catalog contains an invalid category code")
		}
		if strings.TrimSpace(category.Name) == "" || utf8.RuneCountInString(category.Name) > 50 || strings.TrimSpace(category.Description) == "" {
			return CategoryCatalog{}, errors.New("Evidence Category Catalog contains incomplete category content")
		}
		if _, duplicate := seenIDs[category.ID]; duplicate {
			return CategoryCatalog{}, errors.New("Evidence Category Catalog contains a duplicate category ID")
		}
		if _, duplicate := seenCodes[category.Code]; duplicate {
			return CategoryCatalog{}, errors.New("Evidence Category Catalog contains a duplicate category code")
		}
		seenIDs[category.ID] = struct{}{}
		seenCodes[category.Code] = struct{}{}
	}
	sort.Slice(categories, func(left, right int) bool {
		if categories[left].Code != categories[right].Code {
			return categories[left].Code < categories[right].Code
		}
		return categories[left].ID < categories[right].ID
	})
	return CategoryCatalog{Categories: categories}, nil
}

func (s *UseCase) ListEvidence(ctx context.Context, filter EvidenceListFilter) (EvidencePage, error) {
	if s == nil || s.store == nil {
		return EvidencePage{}, errors.New("Evidence store is required")
	}
	if issue := validateEvidenceListFilter(filter); issue != nil {
		return EvidencePage{}, &ValidationError{Issues: []Issue{*issue}}
	}
	page, err := s.store.ListEvidence(ctx, filter)
	if err != nil {
		return EvidencePage{}, fmt.Errorf("list Evidence: %w", err)
	}
	return page, nil
}

func validateEvidenceListFilter(filter EvidenceListFilter) *Issue {
	for _, text := range []struct {
		path  string
		value string
		max   int
	}{
		{path: "title", value: filter.Title, max: 500},
		{path: "summary", value: filter.Summary, max: 200},
		{path: "source_id", value: filter.SourceID, max: 32},
		{path: "source_name", value: filter.SourceName, max: 100},
	} {
		if utf8.RuneCountInString(text.value) > text.max {
			return &Issue{Path: text.path, Code: IssueTooLong, Message: "query is too long"}
		}
	}
	if filter.CategoryID != "" && !filter.CategoryID.IsValid() {
		return &Issue{Path: "category_id", Code: IssueInvalidFormat, Message: "must be a stable Evidence Category ID"}
	}
	if filter.SourceLevel != "" {
		if _, ok := allowedSourceLevels[filter.SourceLevel]; !ok {
			return &Issue{Path: "source_level", Code: IssueInvalidEnum, Message: "is not supported"}
		}
	}
	for _, value := range []*time.Time{filter.PublishedFrom, filter.PublishedTo, filter.CollectedFrom, filter.CollectedTo} {
		if value != nil && !isUTC(*value) {
			return &Issue{Path: "time", Code: IssueInvalidTimestamp, Message: "must use UTC"}
		}
	}
	if filter.PublishedFrom != nil && filter.PublishedTo != nil && filter.PublishedFrom.After(*filter.PublishedTo) {
		return &Issue{Path: "published_from", Code: IssueInvalidTimestamp, Message: "must not be after published_to"}
	}
	if filter.CollectedFrom != nil && filter.CollectedTo != nil && filter.CollectedFrom.After(*filter.CollectedTo) {
		return &Issue{Path: "collected_from", Code: IssueInvalidTimestamp, Message: "must not be after collected_to"}
	}
	if filter.Page < 1 || filter.Page > 1_000_000 || filter.PageSize < 1 || filter.PageSize > 100 {
		return &Issue{Path: "page", Code: IssueInvalidFormat, Message: "pagination is outside the supported range"}
	}
	return nil
}

func (s *UseCase) PublishRawEvidence(ctx context.Context, input RawEvidence) (RawEvidenceResult, error) {
	if s == nil || s.store == nil {
		return RawEvidenceResult{}, errors.New("Evidence Publication store is required")
	}
	if strings.TrimSpace(input.ID) != "" {
		return RawEvidenceResult{}, &ValidationError{Issues: []Issue{{Path: "raw_evidence.id", Code: IssueInvalidFormat, Message: "must be omitted because Data generates Raw Evidence IDs"}}}
	}
	if strings.TrimSpace(input.PublicationKey) == "" {
		return RawEvidenceResult{}, &ValidationError{Issues: []Issue{{Path: "raw_evidence.publication_key", Code: IssueRequired, Message: "value is required"}}}
	}
	rawEvidenceID, err := coreid.Derive(coreid.RawEvidence, "raw-evidence-publication", input.PublicationKey)
	if err != nil {
		return RawEvidenceResult{}, fmt.Errorf("generate Raw Evidence ID: %w", err)
	}
	input.ID = rawEvidenceID
	if err := validateRawEvidence(input); err != nil {
		return RawEvidenceResult{}, err
	}

	record := StoredRawEvidence{RawEvidence: cloneRawEvidence(input), ContentHash: contentHash(input.RawText)}
	var result RawEvidenceResult
	err = s.store.InTransaction(ctx, func(tx Transaction) error {
		if err := tx.LockIdentities(ctx, []string{"raw-evidence:" + input.ID}); err != nil {
			return err
		}
		categories, err := tx.CategoriesByIDs(ctx, record.CategoryIDs)
		if err != nil {
			return err
		}
		if issue := missingCategoryIssue(input.CategoryIDs, categories); issue != nil {
			return &ReferenceError{Issues: []Issue{*issue}}
		}
		record.Categories = cloneCategories(categories)
		existing, err := tx.RawEvidence(ctx, input.ID)
		if err != nil {
			return err
		}
		if existing != nil {
			if !sameRawEvidence(*existing, record) {
				return &ConflictError{Issues: []Issue{{
					Path: "raw_evidence.id", Code: IssueRawEvidenceConflict,
					Message: "id conflicts with stored content",
				}}}
			}
		} else {
			if err := tx.InsertRawEvidence(ctx, record); err != nil {
				return err
			}
			links, err := rawEvidenceCategoryLinks(record.ID, record.CategoryIDs)
			if err != nil {
				return err
			}
			if err := tx.InsertRawEvidenceCategoryLinks(ctx, record.ID, links); err != nil {
				return err
			}
		}
		result = RawEvidenceResult{ID: record.ID}
		return nil
	})
	if err != nil {
		return RawEvidenceResult{}, err
	}
	return result, nil
}

func (s *UseCase) GetRawEvidence(ctx context.Context, rawEvidenceID string) (StoredRawEvidence, error) {
	if s == nil || s.store == nil {
		return StoredRawEvidence{}, errors.New("Evidence store is required")
	}
	var issues []Issue
	requiredDomainID(&issues, "id", rawEvidenceID, RawEvidenceIDPrefix)
	if len(issues) > 0 {
		return StoredRawEvidence{}, &ValidationError{Issues: issues}
	}
	var result StoredRawEvidence
	err := s.store.InTransaction(ctx, func(tx Transaction) error {
		record, err := tx.RawEvidence(ctx, rawEvidenceID)
		if err != nil {
			return err
		}
		if record == nil {
			return ErrRawEvidenceNotFound
		}
		result = cloneStoredRawEvidence(*record)
		return nil
	})
	if err != nil {
		return StoredRawEvidence{}, err
	}
	return result, nil
}

func rawEvidenceCategoryLinks(rawEvidenceID string, categoryIDs []CategoryID) ([]RawEvidenceCategoryLink, error) {
	links := make([]RawEvidenceCategoryLink, len(categoryIDs))
	for index, categoryID := range categoryIDs {
		id, err := coreid.Derive(coreid.RawEvidenceCategoryLink, "raw-evidence-category-link", rawEvidenceID, string(categoryID))
		if err != nil {
			return nil, fmt.Errorf("generate Raw Evidence Category Link ID: %w", err)
		}
		links[index] = RawEvidenceCategoryLink{ID: id, CategoryID: categoryID}
	}
	return links, nil
}

func (s *UseCase) PublishEvidence(ctx context.Context, rawEvidenceID string, input []Evidence) (EvidenceResult, error) {
	if s == nil || s.store == nil {
		return EvidenceResult{}, errors.New("Evidence Publication store is required")
	}
	input = append([]Evidence(nil), input...)
	for index := range input {
		if strings.TrimSpace(input[index].ID) != "" {
			return EvidenceResult{}, &ValidationError{Issues: []Issue{{Path: fmt.Sprintf("evidences[%d].id", index), Code: IssueInvalidFormat, Message: "must be omitted because Data generates Evidence IDs"}}}
		}
		seed, err := evidenceIdentitySeed(input[index])
		if err != nil {
			return EvidenceResult{}, fmt.Errorf("encode Evidence identity: %w", err)
		}
		id, err := coreid.Derive(coreid.Evidence, "atomic-evidence", rawEvidenceID, seed)
		if err != nil {
			return EvidenceResult{}, fmt.Errorf("generate Evidence ID: %w", err)
		}
		input[index].ID = id
	}
	if err := validateEvidencePublication(rawEvidenceID, input); err != nil {
		return EvidenceResult{}, err
	}
	items := make([]EvidenceResultItem, len(input))
	for index, item := range input {
		items[index] = EvidenceResultItem{InputIndex: index, ID: item.ID}
	}

	isSplit := len(input) > 1
	records := make([]StoredEvidence, len(input))
	for index, item := range input {
		records[index] = StoredEvidence{Evidence: cloneEvidence(item), RawEvidenceID: rawEvidenceID, IsSplit: isSplit}
	}
	sort.Slice(records, func(i, j int) bool { return records[i].ID < records[j].ID })
	ids := make([]string, len(records))
	for index, record := range records {
		ids[index] = record.ID
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
					Message: "an id is already assigned to different content",
				}}}
			}
			for _, record := range records {
				if err := tx.InsertEvidence(ctx, record); err != nil {
					return err
				}
			}
		}

		result = EvidenceResult{
			RawEvidenceID: rawEvidenceID,
			IDs:           append([]string(nil), ids...),
			Items:         append([]EvidenceResultItem(nil), items...),
		}
		return nil
	})
	if err != nil {
		return EvidenceResult{}, err
	}
	return result, nil
}

func validateEvidencePublication(rawEvidenceID string, input []Evidence) error {
	var issues []Issue
	requiredDomainID(&issues, "raw_evidence_id", rawEvidenceID, RawEvidenceIDPrefix)
	if len(input) == 0 {
		issues = append(issues, Issue{Path: "evidences", Code: IssueRequired, Message: "at least one Evidence is required"})
	}
	seenIDs := make(map[string]struct{}, len(input))
	for index, item := range input {
		prefix := fmt.Sprintf("evidences[%d]", index)
		requiredDomainID(&issues, prefix+".id", item.ID, EvidenceIDPrefix)
		if _, ok := seenIDs[item.ID]; ok {
			issues = append(issues, Issue{Path: prefix + ".id", Code: IssueDuplicate, Message: "id must be unique within the publication"})
		}
		seenIDs[item.ID] = struct{}{}
		required(&issues, prefix+".summary", item.Summary, 200)
		required(&issues, prefix+".semantic.what", item.Semantic.What, 0)
		optional(&issues, prefix+".semantic.who", item.Semantic.Who, 0)
		optional(&issues, prefix+".semantic.when", item.Semantic.When, 0)
		optional(&issues, prefix+".semantic.where", item.Semantic.Where, 0)
		optional(&issues, prefix+".semantic.why", item.Semantic.Why, 0)
		optional(&issues, prefix+".semantic.how", item.Semantic.How, 0)
	}
	if len(issues) == 0 {
		return nil
	}
	sortIssues(issues)
	return &ValidationError{Issues: issues}
}

func evidenceSetConflict() error {
	return &ConflictError{Issues: []Issue{{
		Path: "evidences", Code: IssueEvidenceSetConflict,
		Message: "Raw Evidence already has a different immutable Evidence set",
	}}}
}

func validateRawEvidence(input RawEvidence) error {
	var issues []Issue
	requiredDomainID(&issues, "raw_evidence.id", input.ID, RawEvidenceIDPrefix)
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
	seenCategories := make(map[CategoryID]struct{}, len(input.CategoryIDs))
	for index, categoryID := range input.CategoryIDs {
		path := fmt.Sprintf("raw_evidence.category_ids[%d]", index)
		requiredDomainID(&issues, path, string(categoryID), CategoryIDPrefix)
		if categoryID != "" && !categoryID.IsValid() {
			issues = append(issues, Issue{Path: path, Code: IssueInvalidFormat, Message: "category_id must use EVC immediately followed by a canonical lowercase UUID"})
		}
		if _, exists := seenCategories[categoryID]; exists {
			issues = append(issues, Issue{Path: path, Code: IssueDuplicate, Message: "category_id must be unique within the Raw Evidence"})
		}
		seenCategories[categoryID] = struct{}{}
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

func requiredDomainID(issues *[]Issue, path, value string, prefix coreid.Kind) {
	if strings.TrimSpace(value) == "" {
		*issues = append(*issues, Issue{Path: path, Code: IssueRequired, Message: "value is required"})
		return
	}
	if !coreid.Is(value, prefix) {
		*issues = append(*issues, Issue{
			Path: path, Code: IssueInvalidFormat,
			Message: coreid.Prefix(prefix) + " must be immediately followed by a canonical lowercase UUID",
		})
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
	input.CategoryIDs = append([]CategoryID(nil), input.CategoryIDs...)
	sort.Slice(input.CategoryIDs, func(i, j int) bool { return input.CategoryIDs[i] < input.CategoryIDs[j] })
	return input
}

func cloneStoredRawEvidence(input StoredRawEvidence) StoredRawEvidence {
	input.RawEvidence = cloneRawEvidence(input.RawEvidence)
	input.Categories = cloneCategories(input.Categories)
	return input
}

func cloneCategories(input []Category) []Category {
	result := append([]Category(nil), input...)
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func missingCategoryIssue(requested []CategoryID, categories []Category) *Issue {
	found := make(map[CategoryID]struct{}, len(categories))
	for _, category := range categories {
		found[category.ID] = struct{}{}
	}
	for index, categoryID := range requested {
		if _, exists := found[categoryID]; !exists {
			return &Issue{
				Path: fmt.Sprintf("raw_evidence.category_ids[%d]", index), Code: IssueCategoryNotFound,
				Message: "category_id does not reference an Evidence Category",
			}
		}
	}
	return nil
}

func cloneEvidence(input Evidence) Evidence {
	input.Semantic.Who = cloneString(input.Semantic.Who)
	input.Semantic.When = cloneString(input.Semantic.When)
	input.Semantic.Where = cloneString(input.Semantic.Where)
	input.Semantic.Why = cloneString(input.Semantic.Why)
	input.Semantic.How = cloneString(input.Semantic.How)
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
	return left.ID == right.ID && left.SourceID == right.SourceID &&
		left.SourceName == right.SourceName && left.SourceLevel == right.SourceLevel &&
		left.SourceURL == right.SourceURL && left.IsOriginal == right.IsOriginal &&
		sameString(left.QuotedSourceID, right.QuotedSourceID) &&
		sameString(left.QuotedSourceName, right.QuotedSourceName) &&
		sameString(left.Title, right.Title) && left.RawText == right.RawText &&
		sameTime(left.PublishedAt, right.PublishedAt) && left.CollectedAt.Equal(right.CollectedAt) &&
		left.ContentHash == right.ContentHash && equalStringSlice(left.Keywords, right.Keywords) &&
		equalCategoryIDs(left.CategoryIDs, right.CategoryIDs)
}

func sameEvidenceSet(left, right []StoredEvidence) bool {
	if len(left) != len(right) {
		return false
	}
	leftByID := make(map[string]StoredEvidence, len(left))
	for _, record := range left {
		leftByID[record.ID] = record
	}
	for _, record := range right {
		existing, ok := leftByID[record.ID]
		if !ok || !sameEvidence(existing, record) {
			return false
		}
	}
	return true
}

func sameEvidence(left, right StoredEvidence) bool {
	return left.ID == right.ID && left.RawEvidenceID == right.RawEvidenceID &&
		left.IsSplit == right.IsSplit && left.Summary == right.Summary &&
		sameSemantic(left.Semantic, right.Semantic)
}

func sameSemantic(left, right Semantic) bool {
	return sameString(left.Who, right.Who) && left.What == right.What && sameString(left.When, right.When) &&
		sameString(left.Where, right.Where) && sameString(left.Why, right.Why) && sameString(left.How, right.How)
}

func evidenceIdentitySeed(input Evidence) (string, error) {
	value, err := json.Marshal(struct {
		Summary  string   `json:"summary"`
		Semantic Semantic `json:"semantic"`
	}{Summary: input.Summary, Semantic: input.Semantic})
	if err != nil {
		return "", err
	}
	return contentHash(string(value)), nil
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

func equalCategoryIDs(left, right []CategoryID) bool {
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
