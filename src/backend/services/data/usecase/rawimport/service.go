// Package rawimport implements the Data Service raw-document batch contract.
package rawimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
	domainimport "github.com/meierlink88/tidewise-ai/backend/services/data/domain/eventimport"
	"github.com/meierlink88/tidewise-ai/backend/services/data/repositories"
)

const (
	CanonicalVersion = "raw-document-import-v1"
	MaxBatchItems    = 100

	CodeInvalidRequest      = "RAW_DOCUMENT_IMPORT_INVALID"
	CodeIdempotencyConflict = "RAW_DOCUMENT_IMPORT_IDEMPOTENCY_CONFLICT"
	CodeIdentityConflict    = "RAW_DOCUMENT_IDENTITY_CONFLICT"
	CodeBatchCollision      = "RAW_DOCUMENT_BATCH_COLLISION"
	CodeReceiptCorrupt      = "RAW_DOCUMENT_RECEIPT_CORRUPT"

	StatusCompleted = "completed"
	StatusUnknown   = "unknown"

	DispositionCreated = "created"
	DispositionReused  = "reused"
)

var (
	ErrSourceNotFound = errors.New("raw import source not found")
	lowercaseSHA256   = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type Candidate struct {
	SourceID         string     `json:"source_id"`
	SourceExternalID string     `json:"source_external_id"`
	IngestChannel    string     `json:"ingest_channel"`
	SourceType       string     `json:"source_type"`
	SourceName       string     `json:"source_name"`
	SourceURL        string     `json:"source_url"`
	Title            string     `json:"title"`
	ContentText      string     `json:"content_text"`
	ContentLevel     string     `json:"content_level"`
	RawObjectURI     string     `json:"raw_object_uri"`
	RawMIMEType      string     `json:"raw_mime_type"`
	Language         string     `json:"language"`
	PublishedAt      *time.Time `json:"published_at"`
	CollectedAt      time.Time  `json:"collected_at"`
	ContentHash      string     `json:"content_hash"`
	rawDocumentID    string
}

type Batch struct {
	Items []Candidate `json:"items"`
}

type Plan struct {
	Version        string
	CallerIdentity string
	IdempotencyKey string
	ReceiptID      string
	PayloadHash    string
	Candidates     []Candidate
}

type ItemResult struct {
	RawDocumentID string `json:"raw_document_id"`
	Disposition   string `json:"disposition"`
}

type Result struct {
	ReceiptID      string       `json:"receipt_id"`
	PayloadHash    string       `json:"payload_hash"`
	RawDocumentIDs []string     `json:"raw_document_ids"`
	Items          []ItemResult `json:"items"`
	ImportedAt     time.Time    `json:"imported_at"`
	Replayed       bool         `json:"-"`
}

type Receipt struct {
	ID             string
	CallerIdentity string
	IdempotencyKey string
	PayloadHash    string
	RawDocumentIDs []string
	Result         Result
	ImportedAt     time.Time
}

type ImportStatus struct {
	State  string  `json:"status"`
	Result *Result `json:"result,omitempty"`
}

type Store interface {
	InRawImportTransaction(context.Context, func(Transaction) error) error
	RawImportReceipt(context.Context, string, string) (*Receipt, error)
}

type Transaction interface {
	LockReceipt(context.Context, string, string, string) (*Receipt, error)
	Source(context.Context, string) (domain.SourceCatalog, error)
	LockRawIdentities(context.Context, []string) error
	RawDocumentByExternalID(context.Context, string, string) (string, error)
	RawDocumentByContentHash(context.Context, string, string) (string, error)
	InsertRawDocument(context.Context, domain.RawDocument) (bool, error)
	InsertReceipt(context.Context, Receipt) error
}

type ContractError struct {
	Code    string
	Message string
	Cause   error
}

func (e *ContractError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *ContractError) Unwrap() error { return e.Cause }

func ErrorCode(err error) string {
	var target *ContractError
	if errors.As(err, &target) {
		return target.Code
	}
	return ""
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store, now func() time.Time) *Service {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Service{store: store, now: now}
}

func (s *Service) Plan(callerIdentity, idempotencyKey string, batch Batch) (Plan, error) {
	caller, err := normalizeBoundedIdentity("caller identity", callerIdentity)
	if err != nil {
		return Plan{}, invalid(err)
	}
	key, err := normalizeBoundedIdentity("idempotency key", idempotencyKey)
	if err != nil {
		return Plan{}, invalid(err)
	}
	if len(batch.Items) < 1 || len(batch.Items) > MaxBatchItems {
		return Plan{}, invalid(fmt.Errorf("items must contain between 1 and %d candidates", MaxBatchItems))
	}

	normalized := make([]Candidate, len(batch.Items))
	for index, candidate := range batch.Items {
		normalized[index], err = normalizeCandidate(candidate)
		if err != nil {
			return Plan{}, invalid(fmt.Errorf("items[%d]: %w", index, err))
		}
	}
	payload, err := json.Marshal(struct {
		Version string      `json:"version"`
		Items   []Candidate `json:"items"`
	}{Version: CanonicalVersion, Items: normalized})
	if err != nil {
		return Plan{}, invalid(fmt.Errorf("encode canonical batch: %w", err))
	}
	payloadHash, err := domainimport.CanonicalHash(payload)
	if err != nil {
		return Plan{}, invalid(fmt.Errorf("hash canonical batch: %w", err))
	}
	return Plan{
		Version: CanonicalVersion, CallerIdentity: caller, IdempotencyKey: key,
		ReceiptID:   repositories.NormalizeUUID("raw_document_import_receipt", caller, key),
		PayloadHash: payloadHash, Candidates: normalized,
	}, nil
}

func (s *Service) Import(ctx context.Context, callerIdentity, idempotencyKey string, batch Batch) (Result, error) {
	if s == nil || s.store == nil {
		return Result{}, fmt.Errorf("raw import store is required")
	}
	plan, err := s.Plan(callerIdentity, idempotencyKey, batch)
	if err != nil {
		return Result{}, err
	}

	var result Result
	err = s.store.InRawImportTransaction(ctx, func(tx Transaction) error {
		existing, err := tx.LockReceipt(ctx, receiptLockText(plan.CallerIdentity, plan.IdempotencyKey), plan.CallerIdentity, plan.IdempotencyKey)
		if err != nil {
			return fmt.Errorf("lock raw import receipt: %w", err)
		}
		if existing != nil {
			if existing.PayloadHash != plan.PayloadHash {
				return contractError(CodeIdempotencyConflict, "idempotency key is already committed with a different payload", nil)
			}
			if err := validateReceipt(*existing, plan.CallerIdentity, plan.IdempotencyKey); err != nil {
				return err
			}
			result = cloneResult(existing.Result)
			result.Replayed = true
			return nil
		}

		sources, err := validateMissBatch(ctx, tx, plan.Candidates)
		if err != nil {
			return err
		}
		locks := rawIdentityLockTexts(plan.Candidates)
		if err := tx.LockRawIdentities(ctx, locks); err != nil {
			return fmt.Errorf("lock raw document identities: %w", err)
		}

		ids := make([]string, 0, len(plan.Candidates))
		items := make([]ItemResult, 0, len(plan.Candidates))
		for index, candidate := range plan.Candidates {
			document := rawDocument(candidate, sources[candidate.SourceID])
			id, disposition, err := resolveRawDocument(ctx, tx, document)
			if err != nil {
				return fmt.Errorf("resolve items[%d]: %w", index, err)
			}
			ids = append(ids, id)
			items = append(items, ItemResult{RawDocumentID: id, Disposition: disposition})
		}
		if err := validateResolvedMembership(ids, len(plan.Candidates)); err != nil {
			return err
		}

		importedAt := s.now().UTC()
		result = Result{
			ReceiptID: plan.ReceiptID, PayloadHash: plan.PayloadHash,
			RawDocumentIDs: append([]string(nil), ids...), Items: append([]ItemResult(nil), items...),
			ImportedAt: importedAt,
		}
		receipt := Receipt{
			ID: plan.ReceiptID, CallerIdentity: plan.CallerIdentity, IdempotencyKey: plan.IdempotencyKey,
			PayloadHash: plan.PayloadHash, RawDocumentIDs: append([]string(nil), ids...),
			Result: cloneResult(result), ImportedAt: importedAt,
		}
		if err := tx.InsertReceipt(ctx, receipt); err != nil {
			return fmt.Errorf("insert raw import receipt: %w", err)
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func (s *Service) Status(ctx context.Context, callerIdentity, idempotencyKey string) (ImportStatus, error) {
	if s == nil || s.store == nil {
		return ImportStatus{}, fmt.Errorf("raw import store is required")
	}
	caller, err := normalizeBoundedIdentity("caller identity", callerIdentity)
	if err != nil {
		return ImportStatus{}, invalid(err)
	}
	key, err := normalizeBoundedIdentity("idempotency key", idempotencyKey)
	if err != nil {
		return ImportStatus{}, invalid(err)
	}
	receipt, err := s.store.RawImportReceipt(ctx, caller, key)
	if err != nil {
		return ImportStatus{}, fmt.Errorf("read raw import receipt: %w", err)
	}
	if receipt == nil {
		return ImportStatus{State: StatusUnknown}, nil
	}
	if err := validateReceipt(*receipt, caller, key); err != nil {
		return ImportStatus{}, err
	}
	result := cloneResult(receipt.Result)
	return ImportStatus{State: StatusCompleted, Result: &result}, nil
}

func normalizeBoundedIdentity(label, value string) (string, error) {
	normalized := strings.TrimSpace(value)
	if normalized == "" || utf8.RuneCountInString(normalized) > 200 {
		return "", fmt.Errorf("%s must contain 1..200 characters", label)
	}
	if strings.ContainsRune(normalized, '\x00') {
		return "", fmt.Errorf("%s must not contain NUL", label)
	}
	return normalized, nil
}

func normalizeCandidate(candidate Candidate) (Candidate, error) {
	candidate.SourceID = strings.ToLower(strings.TrimSpace(candidate.SourceID))
	candidate.SourceExternalID = strings.TrimSpace(candidate.SourceExternalID)
	candidate.IngestChannel = strings.TrimSpace(candidate.IngestChannel)
	candidate.SourceType = strings.TrimSpace(candidate.SourceType)
	candidate.SourceName = strings.TrimSpace(candidate.SourceName)
	candidate.SourceURL = strings.TrimSpace(candidate.SourceURL)
	candidate.Title = strings.TrimSpace(candidate.Title)
	candidate.ContentLevel = strings.TrimSpace(candidate.ContentLevel)
	candidate.RawObjectURI = strings.TrimSpace(candidate.RawObjectURI)
	candidate.RawMIMEType = strings.TrimSpace(candidate.RawMIMEType)
	candidate.Language = strings.TrimSpace(candidate.Language)
	candidate.ContentHash = strings.TrimSpace(candidate.ContentHash)
	if !repositories.IsUUID(candidate.SourceID) {
		return Candidate{}, fmt.Errorf("source_id must be a UUID")
	}
	if candidate.IngestChannel == "" || candidate.SourceType == "" || candidate.SourceName == "" || candidate.SourceURL == "" || candidate.Title == "" || strings.TrimSpace(candidate.ContentText) == "" || candidate.Language == "" {
		return Candidate{}, fmt.Errorf("source attribution, title, content_text and language are required")
	}
	if candidate.CollectedAt.IsZero() {
		return Candidate{}, fmt.Errorf("collected_at is required")
	}
	if !lowercaseSHA256.MatchString(candidate.ContentHash) {
		return Candidate{}, fmt.Errorf("content_hash must be a lowercase 64-character SHA-256")
	}
	candidate.rawDocumentID = repositories.RawDocumentUUID(candidate.SourceID, "", candidate.SourceExternalID, candidate.ContentHash)
	for _, value := range []string{candidate.SourceExternalID, candidate.ContentHash} {
		if strings.ContainsRune(value, '\x00') {
			return Candidate{}, fmt.Errorf("raw identity must not contain NUL")
		}
	}
	candidate.CollectedAt = candidate.CollectedAt.UTC()
	if candidate.PublishedAt != nil {
		published := candidate.PublishedAt.UTC()
		candidate.PublishedAt = &published
	}
	return candidate, nil
}

func validateMissBatch(ctx context.Context, tx Transaction, candidates []Candidate) (map[string]domain.SourceCatalog, error) {
	sources := make(map[string]domain.SourceCatalog)
	rawIDs := make(map[string]struct{}, len(candidates))
	externalIDs := make(map[string]struct{}, len(candidates))
	hashes := make(map[string]struct{}, len(candidates))
	for index, candidate := range candidates {
		if _, duplicate := rawIDs[candidate.rawDocumentID]; duplicate {
			return nil, invalid(fmt.Errorf("items[%d] duplicates raw_document_id", index))
		}
		rawIDs[candidate.rawDocumentID] = struct{}{}
		if candidate.SourceExternalID != "" {
			identity := candidate.SourceID + "\x00" + candidate.SourceExternalID
			if _, duplicate := externalIDs[identity]; duplicate {
				return nil, invalid(fmt.Errorf("items[%d] duplicates source external identity", index))
			}
			externalIDs[identity] = struct{}{}
		}
		hashIdentity := candidate.SourceID + "\x00" + candidate.ContentHash
		if _, duplicate := hashes[hashIdentity]; duplicate {
			return nil, invalid(fmt.Errorf("items[%d] duplicates source content hash", index))
		}
		hashes[hashIdentity] = struct{}{}

		source, loaded := sources[candidate.SourceID]
		if !loaded {
			var err error
			source, err = tx.Source(ctx, candidate.SourceID)
			if err != nil {
				if errors.Is(err, ErrSourceNotFound) {
					return nil, invalid(fmt.Errorf("source %q is unavailable", candidate.SourceID))
				}
				return nil, fmt.Errorf("resolve raw import source: %w", err)
			}
			sources[candidate.SourceID] = source
		}
		if source.ID != candidate.SourceID || source.Status != domain.SourceCatalogStatusActive || source.IngestChannel != candidate.IngestChannel || source.SourceType != candidate.SourceType {
			return nil, invalid(fmt.Errorf("source %q is inactive or attribution does not match", candidate.SourceID))
		}
	}
	return sources, nil
}

func rawIdentityLockTexts(candidates []Candidate) []string {
	set := make(map[string]struct{}, len(candidates)*2)
	for _, candidate := range candidates {
		if candidate.SourceExternalID != "" {
			set[fmt.Sprintf("raw-external:v1|%s|%d|%s", candidate.SourceID, len([]byte(candidate.SourceExternalID)), candidate.SourceExternalID)] = struct{}{}
		}
		set[fmt.Sprintf("raw-hash:v1|%s|%s", candidate.SourceID, candidate.ContentHash)] = struct{}{}
	}
	locks := make([]string, 0, len(set))
	for lock := range set {
		locks = append(locks, lock)
	}
	sort.Strings(locks)
	return locks
}

func receiptLockText(caller, key string) string {
	return fmt.Sprintf("raw-receipt:v1|%d|%s|%s", len([]byte(caller)), caller, key)
}

func rawDocument(candidate Candidate, source domain.SourceCatalog) domain.RawDocument {
	return domain.RawDocument{
		ID: candidate.rawDocumentID, SourceID: candidate.SourceID,
		IngestChannel: candidate.IngestChannel, SourceType: candidate.SourceType,
		SourceName: candidate.SourceName, SourceURL: candidate.SourceURL,
		SourceExternalID: candidate.SourceExternalID, Title: candidate.Title,
		ContentText: candidate.ContentText, ContentLevel: candidate.ContentLevel,
		RawObjectURI: candidate.RawObjectURI, RawMIMEType: candidate.RawMIMEType,
		Language: candidate.Language, PublishedAt: candidate.PublishedAt,
		CollectedAt: candidate.CollectedAt, ContentHash: candidate.ContentHash,
		IngestStatus: domain.IngestStatusCollected,
	}
}

func resolveRawDocument(ctx context.Context, tx Transaction, document domain.RawDocument) (string, string, error) {
	resolve := func() (string, error) {
		var externalID string
		var err error
		if document.SourceExternalID != "" {
			externalID, err = tx.RawDocumentByExternalID(ctx, document.SourceID, document.SourceExternalID)
			if err != nil {
				return "", err
			}
		}
		hashID, err := tx.RawDocumentByContentHash(ctx, document.SourceID, document.ContentHash)
		if err != nil {
			return "", err
		}
		if externalID != "" && hashID != "" && externalID != hashID {
			return "", contractError(CodeIdentityConflict, "source external ID and content hash resolve to different raw documents", nil)
		}
		if externalID != "" {
			return externalID, nil
		}
		return hashID, nil
	}

	existingID, err := resolve()
	if err != nil {
		return "", "", err
	}
	if existingID != "" {
		return existingID, DispositionReused, nil
	}
	created, err := tx.InsertRawDocument(ctx, document)
	if err != nil {
		return "", "", err
	}
	winnerID, err := resolve()
	if err != nil {
		return "", "", err
	}
	if winnerID == "" {
		return "", "", fmt.Errorf("raw document winner is missing after conflict-safe insert")
	}
	if created && winnerID != document.ID {
		return "", "", contractError(CodeIdentityConflict, "inserted raw document did not win its identities", nil)
	}
	if created {
		return winnerID, DispositionCreated, nil
	}
	return winnerID, DispositionReused, nil
}

func validateResolvedMembership(ids []string, candidates int) error {
	if len(ids) != candidates || len(ids) == 0 {
		return contractError(CodeBatchCollision, "resolved raw document membership does not match the batch", nil)
	}
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if !repositories.IsUUID(id) {
			return contractError(CodeBatchCollision, "resolved raw document membership contains an invalid ID", nil)
		}
		if _, duplicate := seen[id]; duplicate {
			return contractError(CodeBatchCollision, "multiple candidates collapsed to one raw document", nil)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func validateReceipt(receipt Receipt, caller, key string) error {
	wantID := repositories.NormalizeUUID("raw_document_import_receipt", caller, key)
	if receipt.ID != wantID || receipt.CallerIdentity != caller || receipt.IdempotencyKey != key || !lowercaseSHA256.MatchString(receipt.PayloadHash) {
		return contractError(CodeReceiptCorrupt, "stored raw import receipt identity is inconsistent", nil)
	}
	result := receipt.Result
	if result.ReceiptID != receipt.ID || result.PayloadHash != receipt.PayloadHash || !result.ImportedAt.Equal(receipt.ImportedAt) || !sameStrings(result.RawDocumentIDs, receipt.RawDocumentIDs) || len(result.Items) != len(receipt.RawDocumentIDs) {
		return contractError(CodeReceiptCorrupt, "stored raw import result does not match receipt columns", nil)
	}
	if err := validateResolvedMembership(receipt.RawDocumentIDs, len(receipt.RawDocumentIDs)); err != nil {
		return contractError(CodeReceiptCorrupt, "stored raw import receipt has invalid membership", err)
	}
	for index, item := range result.Items {
		if item.RawDocumentID != result.RawDocumentIDs[index] || (item.Disposition != DispositionCreated && item.Disposition != DispositionReused) {
			return contractError(CodeReceiptCorrupt, "stored raw import result item is inconsistent", nil)
		}
	}
	return nil
}

func cloneResult(result Result) Result {
	result.RawDocumentIDs = append([]string(nil), result.RawDocumentIDs...)
	result.Items = append([]ItemResult(nil), result.Items...)
	return result
}

func sameStrings(left, right []string) bool {
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

func invalid(err error) error {
	return contractError(CodeInvalidRequest, err.Error(), err)
}

func contractError(code, message string, cause error) error {
	return &ContractError{Code: code, Message: message, Cause: cause}
}
