package evidence

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewUseCaseRejectsMissingStore(t *testing.T) {
	if _, err := NewUseCase(nil); err == nil {
		t.Fatal("NewUseCase(nil) error = nil")
	}
}

func TestPublishRawEvidenceReturnsFormalIdentityAndPreservesKeywordsAcrossRetry(t *testing.T) {
	store := newMemoryStore()
	service := mustNewUseCase(t, store)

	input := validRawEvidence()
	created, err := service.PublishRawEvidence(context.Background(), input)
	if err != nil {
		t.Fatalf("publish Raw Evidence: %v", err)
	}
	if created.RawEvidenceID != input.RawEvidenceID {
		t.Fatalf("Raw Evidence ID = %q, want %q", created.RawEvidenceID, input.RawEvidenceID)
	}
	if got, want := store.raw[input.RawEvidenceID].Keywords, []string{" AI芯片 ", "供应链", "AI芯片"}; !equalStrings(got, want) {
		t.Fatalf("keywords = %#v, want exact publisher order %#v", got, want)
	}
	if store.raw[input.RawEvidenceID].ContentHash != "1b46f625a140463536b92ffb1718d101bbcdfe09a76ef63089af6a0d99b8aa33" {
		t.Fatalf("content hash = %q", store.raw[input.RawEvidenceID].ContentHash)
	}

	reused, err := service.PublishRawEvidence(context.Background(), input)
	if err != nil {
		t.Fatalf("replay Raw Evidence: %v", err)
	}
	if reused.RawEvidenceID != created.RawEvidenceID || len(store.raw) != 1 {
		t.Fatalf("retry result = %#v, stored Raw Evidence = %d", reused, len(store.raw))
	}
}

func TestPublishRawEvidenceRejectsIdentityDriftAndInvalidOrigin(t *testing.T) {
	service := mustNewUseCase(t, newMemoryStore())
	raw := validRawEvidence()
	if _, err := service.PublishRawEvidence(context.Background(), raw); err != nil {
		t.Fatal(err)
	}
	drift := raw
	drift.RawText = "different full article"
	_, err := service.PublishRawEvidence(context.Background(), drift)
	var conflict *ConflictError
	if !errors.As(err, &conflict) || !hasIssueCode(conflict.Issues, IssueRawEvidenceConflict) {
		t.Fatalf("drift error = %#v", err)
	}

	invalidOrigin := validRawEvidence()
	invalidOrigin.QuotedSourceName = stringPointer("Upstream Wire")
	_, err = service.PublishRawEvidence(context.Background(), invalidOrigin)
	var validation *ValidationError
	if !errors.As(err, &validation) || !hasIssueCode(validation.Issues, IssueInvalidOrigin) {
		t.Fatalf("invalid origin error = %#v", err)
	}
}

func TestPublishEvidenceCreatesCompleteSplitSetThenReusesIt(t *testing.T) {
	store := newMemoryStore()
	raw := validRawEvidence()
	store.raw[raw.RawEvidenceID] = StoredRawEvidence{RawEvidence: raw, ContentHash: contentHash(raw.RawText)}
	service := mustNewUseCase(t, store)
	evidences := []Evidence{validEvidence("EVD_example_00000000000000000001", 1), validEvidence("EVD_example_00000000000000000000", 0)}
	created, err := service.PublishEvidence(context.Background(), raw.RawEvidenceID, evidences)
	if err != nil {
		t.Fatalf("publish Evidence: %v", err)
	}
	if got, want := created.EvidenceIDs, []string{"EVD_example_00000000000000000000", "EVD_example_00000000000000000001"}; !equalStrings(got, want) {
		t.Fatalf("Evidence IDs = %#v, want canonical split order %#v", got, want)
	}

	reused, err := service.PublishEvidence(context.Background(), raw.RawEvidenceID, evidences)
	if err != nil {
		t.Fatalf("replay Evidence: %v", err)
	}
	if !equalStrings(reused.EvidenceIDs, created.EvidenceIDs) || len(store.evidences) != 2 {
		t.Fatalf("retry result = %#v, stored Evidences = %d", reused, len(store.evidences))
	}

	drifted := append([]Evidence(nil), evidences...)
	drifted[1].SourceWhat = "Changed fact"
	_, err = service.PublishEvidence(context.Background(), raw.RawEvidenceID, drifted)
	var conflict *ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("drift error = %v, want ConflictError", err)
	}
}

func TestPublishEvidenceSingleIsNotSplitAndRejectsNonContinuousOrder(t *testing.T) {
	store := newMemoryStore()
	raw := validRawEvidence()
	store.raw[raw.RawEvidenceID] = StoredRawEvidence{RawEvidence: raw, ContentHash: contentHash(raw.RawText)}
	service := mustNewUseCase(t, store)

	single := validEvidence("EVD_example_00000000000000000000", 0)
	result, err := service.PublishEvidence(context.Background(), raw.RawEvidenceID, []Evidence{single})
	if err != nil {
		t.Fatalf("publish single Evidence: %v", err)
	}
	if len(result.EvidenceIDs) != 1 || result.EvidenceIDs[0] != single.EvidenceID {
		t.Fatalf("single result = %#v", result.EvidenceIDs)
	}

	otherStore := newMemoryStore()
	otherStore.raw[raw.RawEvidenceID] = StoredRawEvidence{RawEvidence: raw, ContentHash: contentHash(raw.RawText)}
	nonContinuous := []Evidence{validEvidence("EVD_example_00000000000000000001", 0), validEvidence("EVD_example_00000000000000000002", 2)}
	_, err = mustNewUseCase(t, otherStore).PublishEvidence(context.Background(), raw.RawEvidenceID, nonContinuous)
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("non-continuous error = %v, want ValidationError", err)
	}
}

func TestPublishEvidenceRejectsCollectionReferenceLayerAndExpressionFailures(t *testing.T) {
	raw := validRawEvidence()
	store := newMemoryStore()
	store.raw[raw.RawEvidenceID] = StoredRawEvidence{RawEvidence: raw, ContentHash: contentHash(raw.RawText)}
	service := mustNewUseCase(t, store)

	tests := []struct {
		name  string
		items []Evidence
		code  IssueCode
	}{
		{name: "zero", items: nil, code: IssueRequired},
		{name: "duplicate identity", items: []Evidence{
			validEvidence("EVD_example_00000000000000000000", 0),
			validEvidence("EVD_example_00000000000000000000", 1),
		}, code: IssueDuplicate},
		{name: "duplicate split order", items: []Evidence{
			validEvidence("EVD_example_00000000000000000000", 0),
			validEvidence("EVD_example_00000000000000000001", 0),
		}, code: IssueDuplicate},
		{name: "single with core", items: func() []Evidence {
			item := validEvidence("EVD_example_00000000000000000000", 0)
			item.SourceWhatCore = stringPointer("core fact")
			return []Evidence{item}
		}(), code: IssueInvalidLayer},
		{name: "double without core what", items: func() []Evidence {
			item := validEvidence("EVD_example_00000000000000000000", 0)
			item.LayerType = LayerTypeDouble
			return []Evidence{item}
		}(), code: IssueRequired},
		{name: "missing expression identity", items: func() []Evidence {
			item := validEvidence("EVD_example_00000000000000000000", 0)
			item.ExpressionKey = ""
			return []Evidence{item}
		}(), code: IssueRequired},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.PublishEvidence(context.Background(), raw.RawEvidenceID, test.items)
			var validation *ValidationError
			if !errors.As(err, &validation) || !hasIssueCode(validation.Issues, test.code) {
				t.Fatalf("error = %#v, issues = %#v", err, validation)
			}
		})
	}

	_, err := service.PublishEvidence(context.Background(), "RAW_missing_00000000000000000000", []Evidence{
		validEvidence("EVD_example_00000000000000000000", 0),
	})
	var reference *ReferenceError
	if !errors.As(err, &reference) || !hasIssueCode(reference.Issues, IssueRawEvidenceNotFound) {
		t.Fatalf("missing Raw Evidence error = %#v", err)
	}
}

func validEvidence(id string, splitOrder int) Evidence {
	return Evidence{
		EvidenceID:            id,
		SplitOrder:            splitOrder,
		LayerType:             "SINGLE",
		SourceWhat:            "Example Corp expanded production.",
		ExpressionFingerprint: "Example Corp expands production",
		ExpressionKey:         "example-corp-expands-production-v1",
		FingerprintVersion:    "evidence-expression.v1",
	}
}

func validRawEvidence() RawEvidence {
	publishedAt := time.Date(2026, 8, 11, 1, 0, 0, 0, time.UTC)
	return RawEvidence{
		RawEvidenceID: "RAW_example_00000000000000000000",
		SourceID:      "SRC_example_00000000000000000000",
		SourceName:    "Example Wire",
		SourceLevel:   "L2_WIRE",
		SourceURL:     "https://example.test/article/1",
		IsOriginal:    true,
		Title:         stringPointer("Example title"),
		RawText:       "Complete original article.",
		PublishedAt:   &publishedAt,
		CollectedAt:   time.Date(2026, 8, 11, 1, 5, 0, 0, time.UTC),
		Keywords:      []string{" AI芯片 ", "供应链", "AI芯片"},
	}
}

func stringPointer(value string) *string { return &value }

func mustNewUseCase(t *testing.T, store Store) *UseCase {
	t.Helper()
	service, err := NewUseCase(store)
	if err != nil {
		t.Fatalf("construct Evidence Publication service: %v", err)
	}
	return service
}

func equalStrings(left, right []string) bool {
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

func hasIssueCode(issues []Issue, code IssueCode) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

type memoryStore struct {
	raw       map[string]StoredRawEvidence
	evidences map[string]StoredEvidence
}

func newMemoryStore() *memoryStore {
	return &memoryStore{raw: make(map[string]StoredRawEvidence), evidences: make(map[string]StoredEvidence)}
}

func (s *memoryStore) InTransaction(_ context.Context, fn func(Transaction) error) error {
	return fn((*memoryTransaction)(s))
}

type memoryTransaction memoryStore

func (t *memoryTransaction) LockIdentities(context.Context, []string) error { return nil }

func (t *memoryTransaction) RawEvidence(_ context.Context, id string) (*StoredRawEvidence, error) {
	record, ok := t.raw[id]
	if !ok {
		return nil, nil
	}
	return &record, nil
}

func (t *memoryTransaction) InsertRawEvidence(_ context.Context, record StoredRawEvidence) error {
	t.raw[record.RawEvidenceID] = record
	return nil
}

func (t *memoryTransaction) EvidencesByRawEvidence(_ context.Context, rawEvidenceID string) ([]StoredEvidence, error) {
	result := make([]StoredEvidence, 0)
	for _, record := range t.evidences {
		if record.RawEvidenceID == rawEvidenceID {
			result = append(result, record)
		}
	}
	return result, nil
}

func (t *memoryTransaction) EvidencesByIDs(_ context.Context, ids []string) ([]StoredEvidence, error) {
	result := make([]StoredEvidence, 0, len(ids))
	for _, id := range ids {
		if record, ok := t.evidences[id]; ok {
			result = append(result, record)
		}
	}
	return result, nil
}

func (t *memoryTransaction) InsertEvidence(_ context.Context, record StoredEvidence) error {
	t.evidences[record.EvidenceID] = record
	return nil
}
