package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

func TestInMemoryEventWriteContractPersistsApprovedFields(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	event := domain.Event{
		ID:          "event-1",
		Title:       "政策利率维持不变",
		FirstSeenAt: time.Date(2026, 7, 16, 9, 0, 0, 0, time.UTC),
		EventStatus: domain.EventStatusCandidate,
		FactStatus:  domain.FactStatusUnverified,
		DedupeKey:   "policy-rate-hold",
		FactPayload: domain.FactPayload{"policy_rate": map[string]any{"value": 3.5}},
	}

	created, err := repo.UpsertEvent(ctx, event)
	if err != nil {
		t.Fatalf("UpsertEvent() error = %v", err)
	}
	if !created.Created || created.Event.FactPayload["policy_rate"] == nil {
		t.Fatalf("UpsertEvent() result = %+v, want created event with payload", created)
	}

	duplicate, err := repo.UpsertEvent(ctx, event)
	if err != nil {
		t.Fatalf("second UpsertEvent() error = %v", err)
	}
	if duplicate.Created || duplicate.DuplicateOf != event.ID {
		t.Fatalf("duplicate result = %+v, want existing event", duplicate)
	}

	confidence := 0.85
	source := domain.EventSource{
		ID:               "event-source-1",
		EventID:          event.ID,
		RawDocumentID:    "raw-1",
		EvidenceHash:     "evidence-hash-1",
		EvidenceRelation: domain.EvidenceRelationSupports,
		SupportsFields:   []string{"policy_rate"},
	}
	createdSource, err := repo.AddEventSource(ctx, source)
	if err != nil {
		t.Fatalf("AddEventSource() error = %v", err)
	}
	if !createdSource.Created || createdSource.Source.EvidenceRelation != domain.EvidenceRelationSupports || len(createdSource.Source.SupportsFields) != 1 {
		t.Fatalf("AddEventSource() result = %+v, want evidence attribution", createdSource)
	}

	duplicateSource, err := repo.AddEventSource(ctx, source)
	if err != nil {
		t.Fatalf("second AddEventSource() error = %v", err)
	}
	if duplicateSource.Created || duplicateSource.DuplicateOf != source.ID {
		t.Fatalf("duplicate source result = %+v, want existing source", duplicateSource)
	}

	repo.SeedEventTagDef(domain.EventTagDef{ID: "tag-1", TagKind: "topic", Code: "policy", Name: "政策"})
	tagMap := domain.EventTagMap{
		ID:               "event-tag-1",
		EventID:          event.ID,
		TagID:            "tag-1",
		AssignSource:     domain.TagAssignSourceAI,
		Confidence:       &confidence,
		AssignmentReason: "模型识别政策主题",
	}
	createdTagMap, err := repo.AssignEventTag(ctx, tagMap)
	if err != nil {
		t.Fatalf("AssignEventTag() error = %v", err)
	}
	if !createdTagMap.Created || createdTagMap.TagMap.AssignmentReason == "" || createdTagMap.TagMap.Confidence == nil {
		t.Fatalf("AssignEventTag() result = %+v, want tag attribution", createdTagMap)
	}

	page, err := repo.ListEvents(ctx, EventListFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].FactPayload == nil {
		t.Fatalf("ListEvents() items = %+v, want payload-preserving repository contract", page.Items)
	}
	pagePayload := page.Items[0].FactPayload["policy_rate"].(map[string]any)
	pagePayload["value"] = 4.0
	stored, err := repo.UpsertEvent(ctx, event)
	if err != nil {
		t.Fatalf("third UpsertEvent() error = %v", err)
	}
	storedPayload := stored.Event.FactPayload["policy_rate"].(map[string]any)
	if storedPayload["value"] != 3.5 {
		t.Fatalf("stored payload value = %#v, want independent deep copy", storedPayload["value"])
	}
}

func TestInMemoryEventTagAssignmentRequiresTidewiseTagDefinition(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	event := domain.Event{
		ID:          "event-1",
		Title:       "政策利率维持不变",
		FirstSeenAt: time.Date(2026, 7, 16, 9, 0, 0, 0, time.UTC),
		EventStatus: domain.EventStatusCandidate,
		FactStatus:  domain.FactStatusUnverified,
		DedupeKey:   "policy-rate-hold",
		FactPayload: domain.FactPayload{},
	}
	if _, err := repo.UpsertEvent(ctx, event); err != nil {
		t.Fatalf("UpsertEvent() error = %v", err)
	}

	_, err := repo.AssignEventTag(ctx, domain.EventTagMap{
		ID:               "event-tag-unknown",
		EventID:          event.ID,
		TagID:            "not-in-tidewise-db",
		AssignSource:     domain.TagAssignSourceRule,
		AssignmentReason: "策略命中",
	})
	if err == nil {
		t.Fatal("AssignEventTag() error = nil, want unknown tag definition rejection")
	}
}
