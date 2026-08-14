package evidence

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"

	evidencebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/evidence"
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (Store, error) {
	if db == nil {
		return Store{}, errors.New("Evidence database is required")
	}
	return Store{db: db}, nil
}

type persistedInvariantError struct {
	resource string
	field    string
	reason   string
}

func (e *persistedInvariantError) Error() string {
	return fmt.Sprintf("persisted %s field %s violates invariants: %s", e.resource, e.field, e.reason)
}

func persistedInvariant(resource, field, reason string) error {
	return &persistedInvariantError{resource: resource, field: field, reason: reason}
}

func validateStoredRawEvidence(record *evidencebiz.StoredRawEvidence, expectedID string) error {
	const resource = "Raw Evidence"
	if record.RawEvidenceID != expectedID {
		return persistedInvariant(resource, "raw_evidence_id", "query identity does not match the stored row")
	}
	for _, field := range []struct {
		name  string
		value string
		max   int
	}{
		{name: "raw_evidence_id", value: record.RawEvidenceID, max: 32},
		{name: "source_id", value: record.SourceID, max: 32},
		{name: "source_name", value: record.SourceName, max: 100},
		{name: "source_url", value: record.SourceURL},
		{name: "raw_text", value: record.RawText},
	} {
		if err := validateStoredRequired(resource, field.name, field.value, field.max); err != nil {
			return err
		}
	}
	if err := validateStoredOptional(resource, "quoted_source_id", record.QuotedSourceID, 32); err != nil {
		return err
	}
	if err := validateStoredOptional(resource, "quoted_source_name", record.QuotedSourceName, 100); err != nil {
		return err
	}
	if err := validateStoredOptional(resource, "title", record.Title, 500); err != nil {
		return err
	}
	switch record.SourceLevel {
	case evidencebiz.SourceLevelOfficial, evidencebiz.SourceLevelWire,
		evidencebiz.SourceLevelMedia, evidencebiz.SourceLevelSocial:
	default:
		return persistedInvariant(resource, "source_level", "value is not a supported source level")
	}
	parsedURL, err := url.Parse(record.SourceURL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return persistedInvariant(resource, "source_url", "value is not an absolute HTTP(S) URL")
	}
	if record.IsOriginal && (record.QuotedSourceID != nil || record.QuotedSourceName != nil) {
		return persistedInvariant(resource, "is_original", "original content declares a quoted source")
	}
	if !record.IsOriginal && (record.QuotedSourceName == nil || strings.TrimSpace(*record.QuotedSourceName) == "") {
		return persistedInvariant(resource, "quoted_source_name", "reposted content has no quoted source name")
	}
	if record.CollectedAt.IsZero() {
		return persistedInvariant(resource, "collected_at", "timestamp is zero")
	}
	record.CollectedAt = record.CollectedAt.UTC()
	if record.PublishedAt != nil {
		value := record.PublishedAt.UTC()
		record.PublishedAt = &value
	}
	digest := sha256.Sum256([]byte(record.RawText))
	if record.ContentHash != hex.EncodeToString(digest[:]) {
		return persistedInvariant(resource, "content_hash", "value does not match raw_text")
	}
	if record.Keywords == nil {
		return persistedInvariant(resource, "keywords", "array is null")
	}
	return nil
}

func validateStoredEvidence(record *evidencebiz.StoredEvidence) error {
	const resource = "Evidence"
	for _, field := range []struct {
		name  string
		value string
		max   int
	}{
		{name: "evidence_id", value: record.EvidenceID, max: 32},
		{name: "raw_evidence_id", value: record.RawEvidenceID, max: 32},
		{name: "source_what", value: record.SourceWhat},
		{name: "expression_fingerprint", value: record.ExpressionFingerprint, max: 200},
		{name: "expression_key", value: record.ExpressionKey, max: 64},
		{name: "fingerprint_version", value: record.FingerprintVersion, max: 64},
	} {
		if err := validateStoredRequired(resource, field.name, field.value, field.max); err != nil {
			return err
		}
	}
	if record.SplitOrder < 0 {
		return persistedInvariant(resource, "split_order", "value is negative")
	}
	switch record.LayerType {
	case evidencebiz.LayerTypeSingle:
		if hasStoredCoreFields(*record) {
			return persistedInvariant(resource, "layer_type", "SINGLE row declares core fields")
		}
	case evidencebiz.LayerTypeDouble:
		if record.SourceWhatCore == nil || strings.TrimSpace(*record.SourceWhatCore) == "" {
			return persistedInvariant(resource, "source_what_core", "DOUBLE row has no core fact")
		}
	default:
		return persistedInvariant(resource, "layer_type", "value is not a supported layer type")
	}
	if record.SourceWhen != nil {
		value := record.SourceWhen.UTC()
		record.SourceWhen = &value
	}
	if record.SourceWhenCore != nil {
		value := record.SourceWhenCore.UTC()
		record.SourceWhenCore = &value
	}
	return nil
}

func validateStoredEvidenceSet(expectedRawEvidenceID string, records []evidencebiz.StoredEvidence) error {
	if len(records) == 0 {
		return nil
	}
	expectedSplit := len(records) > 1
	for position, record := range records {
		if record.RawEvidenceID != expectedRawEvidenceID {
			return persistedInvariant("Evidence", "raw_evidence_id", "query identity does not match the stored row")
		}
		if record.SplitOrder != position {
			return persistedInvariant("Evidence", "split_order", "stored set is not continuous from zero")
		}
		if record.IsSplit != expectedSplit {
			return persistedInvariant("Evidence", "is_split", "value does not match the stored set cardinality")
		}
	}
	return nil
}

func validateStoredEvidenceIdentities(expectedIDs []string, records []evidencebiz.StoredEvidence) error {
	expected := make(map[string]struct{}, len(expectedIDs))
	for _, id := range expectedIDs {
		expected[id] = struct{}{}
	}
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		if _, ok := expected[record.EvidenceID]; !ok {
			return persistedInvariant("Evidence", "evidence_id", "query returned an unrequested identity")
		}
		if _, ok := seen[record.EvidenceID]; ok {
			return persistedInvariant("Evidence", "evidence_id", "query returned a duplicate identity")
		}
		seen[record.EvidenceID] = struct{}{}
	}
	return nil
}

func validateStoredRequired(resource, field, value string, max int) error {
	if strings.TrimSpace(value) == "" {
		return persistedInvariant(resource, field, "value is blank")
	}
	if max > 0 && len([]rune(value)) > max {
		return persistedInvariant(resource, field, "value exceeds the storage contract")
	}
	return nil
}

func validateStoredOptional(resource, field string, value *string, max int) error {
	if value == nil {
		return nil
	}
	return validateStoredRequired(resource, field, *value, max)
}

func hasStoredCoreFields(record evidencebiz.StoredEvidence) bool {
	return record.SourceWhoCore != nil || record.SourceWhatCore != nil || record.SourceWhenCore != nil ||
		record.SourceWhenRawCore != nil || record.SourceWhereCore != nil || record.SourceWhyCore != nil ||
		record.SourceHowCore != nil
}

var _ evidencebiz.Store = Store{}
