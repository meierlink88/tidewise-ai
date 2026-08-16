package evidence

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

func TestNewUseCaseRejectsMissingStore(t *testing.T) {
	if _, err := NewUseCase(nil); err == nil {
		t.Fatal("NewUseCase(nil) error = nil")
	}
}

func TestListCategoriesReturnsCompleteStableCatalog(t *testing.T) {
	store := newMemoryStore()
	store.categories = []Category{
		{ID: "EVC5b12ffce-178d-56ed-a54f-c01696c486f4", Code: "IN_DEPTH_REPORT", Name: "专题/深度报道", Description: "围绕一个专题进行长篇、多角度调查、梳理或深度报道。"},
		{ID: "EVCc18ddddb-14bc-5496-99ea-963ee2c25597", Code: "EVENT_BRIEF", Name: "事件快讯", Description: "简短报告已经发生或正在发生的事件，核心目的是说明发生了什么。"},
	}
	catalog, err := mustNewUseCase(t, store).ListCategories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Categories) != 2 || catalog.Categories[0].Code != "EVENT_BRIEF" || catalog.Categories[1].Code != "IN_DEPTH_REPORT" {
		t.Fatalf("catalog = %#v", catalog)
	}
	if catalog.Categories[0].Description != store.categories[1].Description {
		t.Fatalf("description = %q, want exact stored value %q", catalog.Categories[0].Description, store.categories[1].Description)
	}
}

func TestListCategoriesFailsClosedForInvalidCatalog(t *testing.T) {
	valid := Category{ID: "EVCc18ddddb-14bc-5496-99ea-963ee2c25597", Code: "EVENT_BRIEF", Name: "事件快讯", Description: "事件快讯定义"}
	tests := []struct {
		name       string
		categories []Category
		err        error
	}{
		{name: "empty"},
		{name: "invalid ID", categories: []Category{{ID: "EVC_001", Code: valid.Code, Name: valid.Name, Description: valid.Description}}},
		{name: "invalid code", categories: []Category{{ID: valid.ID, Code: "event-brief", Name: valid.Name, Description: valid.Description}}},
		{name: "missing content", categories: []Category{{ID: valid.ID, Code: valid.Code, Name: " ", Description: valid.Description}}},
		{name: "duplicate ID", categories: []Category{valid, {ID: valid.ID, Code: "SECOND_CODE", Name: "第二分类", Description: "第二分类定义"}}},
		{name: "duplicate code", categories: []Category{valid, {ID: "EVC5b12ffce-178d-56ed-a54f-c01696c486f4", Code: valid.Code, Name: "第二分类", Description: "第二分类定义"}}},
		{name: "repository failure", err: errors.New("database unavailable")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := newMemoryStore()
			store.categories = test.categories
			store.categoryErr = test.err
			catalog, err := mustNewUseCase(t, store).ListCategories(context.Background())
			if err == nil || len(catalog.Categories) != 0 {
				t.Fatalf("catalog=%#v error=%v, want empty failure", catalog, err)
			}
		})
	}
}

func TestPublishRawEvidenceReturnsFormalIdentityAndPreservesKeywordsAcrossRetry(t *testing.T) {
	store := newMemoryStore()
	service := mustNewUseCase(t, store)

	input := validRawEvidence()
	input.CategoryIDs = []CategoryID{"EVCc18ddddb-14bc-5496-99ea-963ee2c25597"}
	created, err := service.PublishRawEvidence(context.Background(), input)
	if err != nil {
		t.Fatalf("publish Raw Evidence: %v", err)
	}
	if !strings.HasPrefix(created.ID, "RAW") {
		t.Fatalf("Raw Evidence ID = %q", created.ID)
	}
	if got, want := store.raw[created.ID].Keywords, []string{" AI芯片 ", "供应链", "AI芯片"}; !equalStrings(got, want) {
		t.Fatalf("keywords = %#v, want exact publisher order %#v", got, want)
	}
	if store.raw[created.ID].ContentHash != "1b46f625a140463536b92ffb1718d101bbcdfe09a76ef63089af6a0d99b8aa33" {
		t.Fatalf("content hash = %q", store.raw[created.ID].ContentHash)
	}
	if len(store.links) != 1 || store.links[0].ID != "RCLa0c8d966-dbde-56e0-ab89-d0b67b1b7794" {
		t.Fatalf("Raw Evidence Category Links = %#v", store.links)
	}

	reused, err := service.PublishRawEvidence(context.Background(), input)
	if err != nil {
		t.Fatalf("replay Raw Evidence: %v", err)
	}
	if reused.ID != created.ID || len(store.raw) != 1 {
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
	raw := publishedRawEvidence(t)
	store.raw[raw.ID] = StoredRawEvidence{RawEvidence: raw, ContentHash: contentHash(raw.RawText)}
	service := mustNewUseCase(t, store)
	evidences := []Evidence{validEvidence("EVD888d6be0-6378-5f06-bfa1-6e6294f43dca", 1), validEvidence("EVD5cb71bef-5b1d-5995-add0-7408eaa2be15", 0)}
	created, err := service.PublishEvidence(context.Background(), raw.ID, evidences)
	if err != nil {
		t.Fatalf("publish Evidence: %v", err)
	}
	if len(created.IDs) != 2 || !strings.HasPrefix(created.IDs[0], "EVD") || !strings.HasPrefix(created.IDs[1], "EVD") {
		t.Fatalf("Evidence IDs = %#v", created.IDs)
	}

	reused, err := service.PublishEvidence(context.Background(), raw.ID, evidences)
	if err != nil {
		t.Fatalf("replay Evidence: %v", err)
	}
	if !equalStrings(reused.IDs, created.IDs) || len(store.evidences) != 2 {
		t.Fatalf("retry result = %#v, stored Evidences = %d", reused, len(store.evidences))
	}

	drifted := append([]Evidence(nil), evidences...)
	drifted[1].SourceWhat = "Changed fact"
	_, err = service.PublishEvidence(context.Background(), raw.ID, drifted)
	var conflict *ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("drift error = %v, want ConflictError", err)
	}
}

func TestPublishEvidenceCollisionDetailsUseUnifiedIDName(t *testing.T) {
	store := newMemoryStore()
	raw := publishedRawEvidence(t)
	store.raw[raw.ID] = StoredRawEvidence{RawEvidence: raw, ContentHash: contentHash(raw.RawText)}
	collisionID, err := coreid.Derive(coreid.Evidence, "atomic-evidence", raw.ID, "0")
	if err != nil {
		t.Fatal(err)
	}
	store.evidences[collisionID] = StoredEvidence{
		Evidence:      Evidence{ID: collisionID},
		RawEvidenceID: "RAW11111111-1111-4111-8111-111111111111",
	}
	_, err = mustNewUseCase(t, store).PublishEvidence(context.Background(), raw.ID, []Evidence{validEvidence("", 0)})
	var conflict *ConflictError
	if !errors.As(err, &conflict) || len(conflict.Issues) != 1 ||
		strings.Contains(conflict.Issues[0].Message, "evidence_id") || !strings.Contains(conflict.Issues[0].Message, "id") {
		t.Fatalf("collision error = %#v", err)
	}
}

func TestPublishEvidenceSingleIsNotSplitAndRejectsNonContinuousOrder(t *testing.T) {
	store := newMemoryStore()
	raw := publishedRawEvidence(t)
	store.raw[raw.ID] = StoredRawEvidence{RawEvidence: raw, ContentHash: contentHash(raw.RawText)}
	service := mustNewUseCase(t, store)

	single := validEvidence("EVD5cb71bef-5b1d-5995-add0-7408eaa2be15", 0)
	result, err := service.PublishEvidence(context.Background(), raw.ID, []Evidence{single})
	if err != nil {
		t.Fatalf("publish single Evidence: %v", err)
	}
	if len(result.IDs) != 1 || !strings.HasPrefix(result.IDs[0], "EVD") {
		t.Fatalf("single result = %#v", result.IDs)
	}

	otherStore := newMemoryStore()
	otherStore.raw[raw.ID] = StoredRawEvidence{RawEvidence: raw, ContentHash: contentHash(raw.RawText)}
	nonContinuous := []Evidence{validEvidence("EVD888d6be0-6378-5f06-bfa1-6e6294f43dca", 0), validEvidence("EVDe81576cf-65e5-5790-9e39-854386939e72", 2)}
	_, err = mustNewUseCase(t, otherStore).PublishEvidence(context.Background(), raw.ID, nonContinuous)
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("non-continuous error = %v, want ValidationError", err)
	}
}

func TestPublishEvidenceRejectsCollectionReferenceLayerAndExpressionFailures(t *testing.T) {
	raw := publishedRawEvidence(t)
	store := newMemoryStore()
	store.raw[raw.ID] = StoredRawEvidence{RawEvidence: raw, ContentHash: contentHash(raw.RawText)}
	service := mustNewUseCase(t, store)

	tests := []struct {
		name  string
		items []Evidence
		code  IssueCode
	}{
		{name: "zero", items: nil, code: IssueRequired},
		{name: "duplicate split order", items: []Evidence{
			validEvidence("EVD5cb71bef-5b1d-5995-add0-7408eaa2be15", 0),
			validEvidence("EVD888d6be0-6378-5f06-bfa1-6e6294f43dca", 0),
		}, code: IssueDuplicate},
		{name: "single with core", items: func() []Evidence {
			item := validEvidence("EVD5cb71bef-5b1d-5995-add0-7408eaa2be15", 0)
			item.SourceWhatCore = stringPointer("core fact")
			return []Evidence{item}
		}(), code: IssueInvalidLayer},
		{name: "double without core what", items: func() []Evidence {
			item := validEvidence("EVD5cb71bef-5b1d-5995-add0-7408eaa2be15", 0)
			item.LayerType = LayerTypeDouble
			return []Evidence{item}
		}(), code: IssueRequired},
		{name: "missing expression identity", items: func() []Evidence {
			item := validEvidence("EVD5cb71bef-5b1d-5995-add0-7408eaa2be15", 0)
			item.ExpressionKey = ""
			return []Evidence{item}
		}(), code: IssueRequired},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.PublishEvidence(context.Background(), raw.ID, test.items)
			var validation *ValidationError
			if !errors.As(err, &validation) || !hasIssueCode(validation.Issues, test.code) {
				t.Fatalf("error = %#v, issues = %#v", err, validation)
			}
		})
	}

	_, err := service.PublishEvidence(context.Background(), "RAW6a2f6777-6aa6-5f07-9b95-ced31a3d8e59", []Evidence{
		validEvidence("EVD5cb71bef-5b1d-5995-add0-7408eaa2be15", 0),
	})
	var reference *ReferenceError
	if !errors.As(err, &reference) || !hasIssueCode(reference.Issues, IssueRawEvidenceNotFound) {
		t.Fatalf("missing Raw Evidence error = %#v", err)
	}
}

func validEvidence(id string, splitOrder int) Evidence {
	return Evidence{
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
		PublicationKey: "example-publication-1",
		SourceID:       "SRC_example_00000000000000000000",
		SourceName:     "Example Wire",
		SourceLevel:    "L2_WIRE",
		SourceURL:      "https://example.test/article/1",
		IsOriginal:     true,
		Title:          stringPointer("Example title"),
		RawText:        "Complete original article.",
		PublishedAt:    &publishedAt,
		CollectedAt:    time.Date(2026, 8, 11, 1, 5, 0, 0, time.UTC),
		Keywords:       []string{" AI芯片 ", "供应链", "AI芯片"},
	}
}

func publishedRawEvidence(t *testing.T) RawEvidence {
	t.Helper()
	raw := validRawEvidence()
	id, err := coreid.Derive(coreid.RawEvidence, "raw-evidence-publication", raw.PublicationKey)
	if err != nil {
		t.Fatal(err)
	}
	raw.ID = id
	return raw
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
	raw         map[string]StoredRawEvidence
	evidences   map[string]StoredEvidence
	links       []RawEvidenceCategoryLink
	categories  []Category
	categoryErr error
}

func newMemoryStore() *memoryStore {
	return &memoryStore{raw: make(map[string]StoredRawEvidence), evidences: make(map[string]StoredEvidence)}
}

func (s *memoryStore) ListCategories(context.Context) ([]Category, error) {
	return cloneCategories(s.categories), s.categoryErr
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

func (*memoryTransaction) CategoriesByIDs(_ context.Context, ids []CategoryID) ([]Category, error) {
	result := make([]Category, len(ids))
	for index, id := range ids {
		result[index] = Category{ID: id, Code: "TEST", Name: "测试", Description: "测试分类"}
	}
	return result, nil
}

func (t *memoryTransaction) InsertRawEvidence(_ context.Context, record StoredRawEvidence) error {
	t.raw[record.ID] = record
	return nil
}

func (t *memoryTransaction) InsertRawEvidenceCategoryLinks(_ context.Context, _ string, links []RawEvidenceCategoryLink) error {
	t.links = append([]RawEvidenceCategoryLink(nil), links...)
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
	t.evidences[record.ID] = record
	return nil
}
