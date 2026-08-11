package event

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	eventapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1/event"
	eventsemanticapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1/eventsemantic"
	evidenceapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1/evidence"
	eventbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/event"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/conf"
	eventdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/event"
	serverpkg "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/server"
	parentservice "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/service"
	postgresfixture "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/testsupport/postgres"
)

type fakeUseCase struct {
	catalog eventbiz.EventTagCatalog
	err     error
}

func (f fakeUseCase) Import(context.Context, string, eventbiz.PublicationBatch) (eventbiz.Result, error) {
	return eventbiz.Result{}, f.err
}

func (f fakeUseCase) ActiveTags(context.Context) (eventbiz.EventTagCatalog, error) {
	return f.catalog, f.err
}

func (f fakeUseCase) ListEvents(context.Context, eventbiz.EventListRequest) (eventbiz.EventPage, error) {
	return eventbiz.EventPage{}, f.err
}

func TestCancellationPreservesLegacyPublicErrorMapping(t *testing.T) {
	service, err := NewService(fakeUseCase{err: context.Canceled})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, code string
		call       func() error
	}{
		{name: "publication", code: eventapi.ErrorEventPublicationFailed, call: func() error {
			_, callErr := service.PublishReviewedEvents(v1.WithPrincipal(context.Background(), v1.Principal{Identity: "caller"}), &eventapi.PublicationRequest{})
			return callErr
		}},
		{name: "catalog", code: eventapi.ErrorEventTagCatalogFailed, call: func() error {
			_, callErr := service.ListActiveEventTags(context.Background(), &eventapi.TagCatalogRequest{Active: true})
			return callErr
		}},
		{name: "list", code: eventapi.ErrorDataRepositoryFailure, call: func() error {
			_, callErr := service.ListEvents(context.Background(), &eventapi.ListRequest{})
			return callErr
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var public *v1.PublicError
			if err := test.call(); !errors.As(err, &public) || public.Code != test.code {
				t.Fatalf("error = %#v, want public code %s", err, test.code)
			}
		})
	}
}

func TestListActiveEventTagsMapsCatalogSnapshot(t *testing.T) {
	const catalogHash = "1a7035195312cdf7652880308d9fffcc6aea180f7c09b5c07f6678514a1298eb"
	service, err := NewService(fakeUseCase{catalog: eventbiz.EventTagCatalog{
		Revision: "event-tags:" + catalogHash,
		Hash:     catalogHash,
		Tags: []eventbiz.EventTag{{
			ID: "22a5afc5-20ed-55ce-bf77-54c26bbcc6ea", Kind: "news_category",
			Code: "technology_industry", Name: "科技产业", Active: true,
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.ListActiveEventTags(context.Background(), &eventapi.TagCatalogRequest{Active: true})
	if err != nil {
		t.Fatal(err)
	}
	if response.Result.CatalogHash != catalogHash || response.Result.CatalogRevision != "event-tags:"+catalogHash ||
		len(response.Result.Tags) != 1 || !response.Result.Tags[0].IsActive {
		t.Fatalf("catalog = %#v", response.Result)
	}
}

func TestPostgresEventPublicationResponseLossReplayReusesFactsAndPreservesLineage(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	handler, service := newEventPublicationTestHandler(t, db)

	body := []byte(`{
	  "package_id": "agentrun:event-publication:20260723:001",
	  "provenance": {
	    "extractor_execution_id": "extractor-exec-001",
	    "extractor_agent_version": "event-extractor-v2.0.0",
	    "collector_executions": [
	      {
	        "artifact_id": "artifact-001",
	        "collector_execution_id": "collector-exec-101"
	      }
	    ]
	  },
	  "raw_documents": [
	    {
	      "artifact_id": "artifact-001",
	      "content_sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	      "source_ref": "source:reuters:world",
	      "source_name": "Reuters",
	      "source_type": "news",
	      "source_url": "https://example.test/article/1",
	      "title": "Example source title",
	      "published_at": "2026-07-23T01:00:00Z",
	      "collected_at": "2026-07-23T01:05:00Z",
	      "language": "en",
	      "mime_type": "text/markdown"
	    }
	  ],
	  "events": [
	    {
	      "dedupe_key": "event:example:20260723:001",
	      "title": "Example event",
	      "factual_summary": "A verifiable state change occurred.",
	      "occurred_at": "2026-07-23T00:30:00Z",
	      "fact_payload": {
	        "metric": "example"
	      },
	      "evidence": [
	        {
	          "artifact_id": "artifact-001",
	          "evidence_relation": "supports",
	          "evidence_statement": "A short excerpt supporting the event.",
	          "supports_fields": [
	            "title",
	            "factual_summary"
	          ],
	          "source_level": "primary"
	        }
	      ],
	      "tags": [
	        {
	          "tag_id": "22a5afc5-20ed-55ce-bf77-54c26bbcc6ea",
	          "tag_kind": "news_category",
	          "tag_code": "technology_industry",
	          "confidence": 0.94,
	          "assignment_reason": "The event concerns technology industry supply.",
	          "assign_source": "ai"
	        }
	      ],
	      "review": {
	        "review_id": "review-001",
	        "evidence_grade": "A",
	        "reasons": [
	          "The source and event facts are internally consistent."
	        ]
	      }
	    }
	  ]
	}`)

	first := postEventPublication(t, handler, body)
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("first status = %d, body = %s", first.StatusCode, first.Body)
	}
	if first.Result.Counts.EventsCreated != 1 ||
		first.Result.Counts.RawDocumentsCreated != 1 ||
		first.Result.Counts.EventSourcesCreated != 1 ||
		first.Result.Counts.EventTagsCreated != 1 {
		t.Fatalf("first counts = %#v, want all created", first.Result.Counts)
	}

	second := postEventPublication(t, handler, body)
	if second.StatusCode != http.StatusCreated {
		t.Fatalf("second status = %d, body = %s, service error = %v", second.StatusCode, second.Body, service.lastError)
	}
	if second.Result.ReceiptID == first.Result.ReceiptID {
		t.Fatalf("receipt ID was reused: %q", second.Result.ReceiptID)
	}
	if second.Result.Counts.EventsReused != 1 ||
		second.Result.Counts.RawDocumentsReused != 1 ||
		second.Result.Counts.EventSourcesReused != 1 ||
		second.Result.Counts.EventTagsReused != 1 {
		t.Fatalf("second counts = %#v, want all reused", second.Result.Counts)
	}
	if second.Result.Events[0].EventID != first.Result.Events[0].EventID ||
		second.Result.RawDocuments[0].RawDocumentID != first.Result.RawDocuments[0].RawDocumentID {
		t.Fatalf("natural identities changed between successful publications")
	}
	assertPublicationDBCounts(t, db, publicationDBCounts{
		RawDocuments: 1,
		Events:       1,
		EventSources: 1,
		EventTags:    1,
		Receipts:     2,
	})
	var receiptCount, extractorExecutions, extractorVersions, collectorLineages int
	if err := db.QueryRow(`
SELECT
  count(*),
  count(DISTINCT extractor_execution_id),
  count(DISTINCT extractor_agent_version),
  count(DISTINCT collector_executions::text)
FROM event_publication_receipts
WHERE package_id = 'agentrun:event-publication:20260723:001'`).Scan(
		&receiptCount,
		&extractorExecutions,
		&extractorVersions,
		&collectorLineages,
	); err != nil {
		t.Fatal(err)
	}
	if receiptCount != 2 || extractorExecutions != 1 ||
		extractorVersions != 1 || collectorLineages != 1 {
		t.Fatalf(
			"replayed receipt lineage count=%d extractorExecutions=%d extractorVersions=%d collectors=%d",
			receiptCount, extractorExecutions, extractorVersions, collectorLineages,
		)
	}
}

func TestPostgresEventAdapterRejectsCorruptedEvidenceHash(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	handler, _ := newEventPublicationTestHandler(t, db)
	response := postEventPublication(t, handler, marshalPublication(t, eventPublicationFixture("corrupted-hash")))
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("publication status = %d: %s", response.StatusCode, response.Body)
	}
	eventID := response.Result.Events[0].EventID
	rawDocumentID := response.Result.RawDocuments[0].RawDocumentID
	if _, err := db.Exec(`UPDATE event_sources SET evidence_hash = $1 WHERE event_id = $2 AND raw_document_id = $3`, strings.Repeat("f", 64), eventID, rawDocumentID); err != nil {
		t.Fatal(err)
	}
	store, err := eventdata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	err = store.InTransaction(context.Background(), func(tx eventbiz.Transaction) error {
		_, readErr := tx.StoredEventEvidenceLink(context.Background(), eventID, rawDocumentID)
		return readErr
	})
	if err == nil || !strings.Contains(err.Error(), "hash does not match") {
		t.Fatalf("read corrupted Evidence Link error = %v", err)
	}
}

func TestPostgresEventPublicationContractScenarios(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	handler, service := newEventPublicationTestHandler(t, db)
	t.Run("repeated Event accumulates equal Evidence across publication batches", func(t *testing.T) {
		first := eventPublicationFixture("multi-source-event")
		firstResponse := postEventPublication(t, handler, marshalPublication(t, first))
		if firstResponse.StatusCode != http.StatusCreated {
			t.Fatalf("first status = %d, body = %s", firstResponse.StatusCode, firstResponse.Body)
		}

		second := eventPublicationFixture("multi-source-event")
		second.PackageID += ":second-source"
		second.RawDocuments[0].ArtifactID += ":second"
		second.RawDocuments[0].ContentSHA256 = strings.Repeat("d", 64)
		second.RawDocuments[0].SourceRef += ":second"
		second.RawDocuments[0].SourceURL += "/second"
		second.RawDocuments[0].Title = "Independent second source"
		second.Provenance.ExtractorExecutionID += ":second"
		second.Provenance.CollectorExecutions[0].ArtifactID = second.RawDocuments[0].ArtifactID
		second.Provenance.CollectorExecutions[0].CollectorExecutionID += ":second"
		second.Events[0].Evidence[0].ArtifactID = second.RawDocuments[0].ArtifactID
		second.Events[0].Evidence[0].EvidenceStatement = "An independent source supports the same Event."
		secondResponse := postEventPublication(t, handler, marshalPublication(t, second))
		if secondResponse.StatusCode != http.StatusCreated {
			t.Fatalf("second status = %d, body = %s", secondResponse.StatusCode, secondResponse.Body)
		}
		if secondResponse.Result.Counts.EventsReused != 1 ||
			secondResponse.Result.Counts.EventSourcesCreated != 1 ||
			secondResponse.Result.Counts.RawDocumentsCreated != 1 {
			t.Fatalf("second counts = %#v", secondResponse.Result.Counts)
		}
	})

	t.Run("one artifact supports multiple Events and one Event uses multiple artifacts", func(t *testing.T) {
		before := readPublicationDBCounts(t, db)
		publication := eventPublicationFixture("relationships")
		secondDocument := publication.RawDocuments[0]
		secondDocument.ArtifactID = "artifact-relationships-2"
		secondDocument.ContentSHA256 = fmt.Sprintf("%064x", 2)
		secondDocument.SourceRef = "source:relationships:2"
		secondDocument.SourceURL = "https://example.test/relationships/2"
		secondDocument.Title = "Second evidence document"
		publication.RawDocuments = append(publication.RawDocuments, secondDocument)
		publication.Provenance.CollectorExecutions = append(
			publication.Provenance.CollectorExecutions,
			eventbiz.CollectorExecution{
				ArtifactID:           secondDocument.ArtifactID,
				CollectorExecutionID: "collector-relationships-2",
			},
		)

		secondEvent := clonePublicationEvent(publication.Events[0], "relationships-2")
		thirdEvent := clonePublicationEvent(publication.Events[0], "relationships-3")
		thirdEvent.Evidence = append(thirdEvent.Evidence, eventbiz.EventEvidenceLinkInput{
			ArtifactID:        secondDocument.ArtifactID,
			EvidenceRelation:  "context",
			EvidenceStatement: "A second document provides relevant context.",
			SupportsFields:    []string{},
			SourceLevel:       "secondary",
		})
		publication.Events = append(publication.Events, secondEvent, thirdEvent)

		response := postEventPublication(t, handler, marshalPublication(t, publication))
		if response.StatusCode != http.StatusCreated {
			t.Fatalf("status = %d, body = %s, service error = %v", response.StatusCode, response.Body, service.lastError)
		}
		counts := response.Result.Counts
		if counts.EventsCreated != 3 || counts.RawDocumentsCreated != 2 ||
			counts.EventSourcesCreated != 4 || counts.EventTagsCreated != 3 {
			t.Fatalf("counts = %#v", counts)
		}
		var sourceLinks int
		if err := db.QueryRow(`
SELECT count(*) FROM event_sources WHERE contract_version = 3`).Scan(&sourceLinks); err != nil {
			t.Fatal(err)
		}
		if sourceLinks-before.EventSources != 4 {
			t.Fatalf("new source links = %d, want 4", sourceLinks-before.EventSources)
		}
		var lightweightDocuments int
		if err := db.QueryRow(`
SELECT count(*)
FROM raw_documents
WHERE contract_version = 2
  AND content_text = ''
  AND raw_object_uri = ''
  AND ingest_channel = ''
  AND source_external_id IS NULL`).Scan(&lightweightDocuments); err != nil {
			t.Fatal(err)
		}
		if lightweightDocuments-before.RawDocuments != 2 {
			t.Fatalf("new lightweight publication documents = %d, want 2", lightweightDocuments-before.RawDocuments)
		}
	})

	t.Run("unreferenced and unknown artifacts are rejected without writes", func(t *testing.T) {
		before := readPublicationDBCounts(t, db)
		unreferenced := eventPublicationFixture("unreferenced")
		extra := unreferenced.RawDocuments[0]
		extra.ArtifactID = "artifact-unreferenced-extra"
		extra.ContentSHA256 = fmt.Sprintf("%064x", 3)
		extra.SourceRef = "source:unreferenced:extra"
		unreferenced.RawDocuments = append(unreferenced.RawDocuments, extra)
		unreferenced.Provenance.CollectorExecutions = append(
			unreferenced.Provenance.CollectorExecutions,
			eventbiz.CollectorExecution{
				ArtifactID: extra.ArtifactID, CollectorExecutionID: "collector-unreferenced-extra",
			},
		)
		response := postEventPublication(t, handler, marshalPublication(t, unreferenced))
		if response.StatusCode != http.StatusUnprocessableEntity || response.Error.Code != "EVENT_PUBLICATION_INVALID" {
			t.Fatalf("unreferenced status = %d, body = %s", response.StatusCode, response.Body)
		}
		assertPublicationDBCounts(t, db, before)

		unknown := eventPublicationFixture("unknown-artifact")
		unknown.Events[0].Evidence[0].ArtifactID = "artifact-not-declared"
		response = postEventPublication(t, handler, marshalPublication(t, unknown))
		if response.StatusCode != http.StatusUnprocessableEntity || response.Error.Code != "EVENT_PUBLICATION_INVALID" {
			t.Fatalf("unknown status = %d, body = %s", response.StatusCode, response.Body)
		}
		assertPublicationDBCounts(t, db, before)
	})

	t.Run("artifact conflict rolls back the whole batch", func(t *testing.T) {
		base := eventPublicationFixture("artifact-conflict")
		first := postEventPublication(t, handler, marshalPublication(t, base))
		if first.StatusCode != http.StatusCreated {
			t.Fatalf("first status = %d, body = %s", first.StatusCode, first.Body)
		}
		before := readPublicationDBCounts(t, db)
		conflict := eventPublicationFixture("artifact-conflict-second")
		conflict.RawDocuments[0] = base.RawDocuments[0]
		conflict.RawDocuments[0].ContentSHA256 = fmt.Sprintf("%064x", 9)
		conflict.Provenance.CollectorExecutions[0].ArtifactID = base.RawDocuments[0].ArtifactID
		conflict.Events[0].Evidence[0].ArtifactID = base.RawDocuments[0].ArtifactID
		response := postEventPublication(t, handler, marshalPublication(t, conflict))
		if response.StatusCode != http.StatusConflict || response.Error.Code != "EVENT_PUBLICATION_CONFLICT" {
			t.Fatalf("status = %d, body = %s", response.StatusCode, response.Body)
		}
		assertPublicationDBCounts(t, db, before)
	})

	t.Run("Event conflict rolls back the whole batch", func(t *testing.T) {
		base := eventPublicationFixture("event-conflict")
		first := postEventPublication(t, handler, marshalPublication(t, base))
		if first.StatusCode != http.StatusCreated {
			t.Fatalf("first status = %d, body = %s", first.StatusCode, first.Body)
		}
		before := readPublicationDBCounts(t, db)
		conflict := base
		conflict.PackageID = "package-event-conflict-second"
		conflict.Events = append([]eventbiz.PublicationEvent(nil), base.Events...)
		conflict.Events[0].Title = "A different immutable title"
		response := postEventPublication(t, handler, marshalPublication(t, conflict))
		if response.StatusCode != http.StatusConflict || response.Error.Code != "EVENT_PUBLICATION_CONFLICT" {
			t.Fatalf("status = %d, body = %s", response.StatusCode, response.Body)
		}
		assertPublicationDBCounts(t, db, before)
	})

	t.Run("large fact payload integers remain conflict-safe", func(t *testing.T) {
		base := eventPublicationFixture("large-number-conflict")
		base.Events[0].FactPayload = map[string]any{
			"count": json.Number("9007199254740992"),
		}
		first := postEventPublication(t, handler, marshalPublication(t, base))
		if first.StatusCode != http.StatusCreated {
			t.Fatalf("first status = %d, body = %s", first.StatusCode, first.Body)
		}
		before := readPublicationDBCounts(t, db)

		conflict := base
		conflict.PackageID = "package-large-number-conflict-second"
		conflict.Events = append([]eventbiz.PublicationEvent(nil), base.Events...)
		conflict.Events[0].FactPayload = map[string]any{
			"count": json.Number("9007199254740993"),
		}
		response := postEventPublication(t, handler, marshalPublication(t, conflict))
		if response.StatusCode != http.StatusConflict || response.Error.Code != "EVENT_PUBLICATION_CONFLICT" {
			t.Fatalf("status = %d, body = %s", response.StatusCode, response.Body)
		}
		assertPublicationDBCounts(t, db, before)
	})

	t.Run("supports fields are an order-insensitive set", func(t *testing.T) {
		base := eventPublicationFixture("supports-fields-order")
		base.Events[0].Evidence[0].SupportsFields = []string{"title", "fact_payload"}
		first := postEventPublication(t, handler, marshalPublication(t, base))
		if first.StatusCode != http.StatusCreated {
			t.Fatalf("first status = %d, body = %s", first.StatusCode, first.Body)
		}

		reordered := base
		reordered.PackageID = "package-supports-fields-order-second"
		reordered.Events = append([]eventbiz.PublicationEvent(nil), base.Events...)
		reordered.Events[0].Evidence = append(
			[]eventbiz.EventEvidenceLinkInput(nil),
			base.Events[0].Evidence...,
		)
		reordered.Events[0].Evidence[0].SupportsFields = []string{"fact_payload", "title"}
		response := postEventPublication(t, handler, marshalPublication(t, reordered))
		if response.StatusCode != http.StatusCreated {
			t.Fatalf("status = %d, body = %s, service error = %v", response.StatusCode, response.Body, service.lastError)
		}
		if response.Result.Counts.EventSourcesReused != 1 {
			t.Fatalf("counts = %#v, want existing source association reused", response.Result.Counts)
		}
	})

	t.Run("unknown inactive and mismatched Tags are rejected atomically", func(t *testing.T) {
		tests := []struct {
			name    string
			prepare func(*eventbiz.PublicationBatch)
			cleanup func()
		}{
			{
				name: "unknown",
				prepare: func(publication *eventbiz.PublicationBatch) {
					publication.Events[0].Tags[0].TagID = "11111111-1111-4111-8111-111111111111"
				},
			},
			{
				name: "inactive",
				prepare: func(_ *eventbiz.PublicationBatch) {
					if _, err := db.Exec(`UPDATE event_tag_defs SET is_active = false WHERE id = '22a5afc5-20ed-55ce-bf77-54c26bbcc6ea'`); err != nil {
						t.Fatal(err)
					}
				},
				cleanup: func() {
					if _, err := db.Exec(`UPDATE event_tag_defs SET is_active = true WHERE id = '22a5afc5-20ed-55ce-bf77-54c26bbcc6ea'`); err != nil {
						t.Fatal(err)
					}
				},
			},
			{
				name: "identity mismatch",
				prepare: func(publication *eventbiz.PublicationBatch) {
					publication.Events[0].Tags[0].TagCode = "macroeconomy"
				},
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				before := readPublicationDBCounts(t, db)
				publication := eventPublicationFixture("tag-" + test.name)
				test.prepare(&publication)
				response := postEventPublication(t, handler, marshalPublication(t, publication))
				if test.cleanup != nil {
					test.cleanup()
				}
				if response.StatusCode != http.StatusUnprocessableEntity || response.Error.Code != "EVENT_PUBLICATION_INVALID" {
					t.Fatalf("status = %d, body = %s", response.StatusCode, response.Body)
				}
				assertPublicationDBCounts(t, db, before)
			})
		}
	})

	t.Run("Evidence Review and Tag dimension failures leave no partial writes", func(t *testing.T) {
		before := readPublicationDBCounts(t, db)

		invalidReview := eventPublicationFixture("invalid-review")
		invalidReview.Events[0].Review.Reasons = nil
		response := postEventPublication(t, handler, marshalPublication(t, invalidReview))
		if response.StatusCode != http.StatusBadRequest || response.Error.Code != "INVALID_REQUEST" {
			t.Fatalf("review status = %d, body = %s", response.StatusCode, response.Body)
		}
		assertPublicationDBCounts(t, db, before)

		tooManyNewsTags := eventPublicationFixture("too-many-news-tags")
		tooManyNewsTags.Events[0].Tags = append(
			tooManyNewsTags.Events[0].Tags,
			eventbiz.EventTagInput{
				TagID: "b0fe1994-0db2-526c-a57f-97fa73c1b595", TagKind: "news_category",
				TagCode: "geopolitics", Confidence: json.Number("0.8"),
				AssignmentReason: "Geopolitical context", AssignSource: "ai",
			},
			eventbiz.EventTagInput{
				TagID: "b1a5438f-6e81-55e7-8ecb-33230b9ae965", TagKind: "news_category",
				TagCode: "macroeconomy", Confidence: json.Number("0.8"),
				AssignmentReason: "Macroeconomic context", AssignSource: "rule",
			},
		)
		response = postEventPublication(t, handler, marshalPublication(t, tooManyNewsTags))
		if response.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("Tag dimension status = %d, body = %s", response.StatusCode, response.Body)
		}
		assertPublicationDBCounts(t, db, before)
	})

	t.Run("strict decoding deterministic validation and authorization", func(t *testing.T) {
		before := readPublicationDBCounts(t, db)
		publication := eventPublicationFixture("transport")
		raw := map[string]any{}
		if err := json.Unmarshal(marshalPublication(t, publication), &raw); err != nil {
			t.Fatal(err)
		}
		documents := raw["raw_documents"].([]any)
		documents[0].(map[string]any)["content_text"] = "forbidden"
		response := postEventPublication(t, handler, mustJSON(t, raw))
		if response.StatusCode != http.StatusBadRequest || response.Error.Code != "INVALID_REQUEST" {
			t.Fatalf("unknown-field status = %d, body = %s", response.StatusCode, response.Body)
		}

		invalid := eventPublicationFixture("sorted-errors")
		invalid.PackageID = ""
		invalid.Events[0].Title = ""
		invalid.RawDocuments[0].Title = ""
		response = postEventPublication(t, handler, marshalPublication(t, invalid))
		if response.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("validation status = %d, body = %s", response.StatusCode, response.Body)
		}
		issues := response.Error.ValidationIssues(t)
		wantPaths := []string{"events[0].title", "package_id", "raw_documents[0].title"}
		if len(issues) != len(wantPaths) {
			t.Fatalf("issues = %#v", issues)
		}
		for index, path := range wantPaths {
			if issues[index].Path != path {
				t.Fatalf("issue paths = %#v, want %v", issues, wantPaths)
			}
		}

		response = postEventPublicationAs(t, handler, marshalPublication(t, publication), "")
		if response.StatusCode != http.StatusUnauthorized {
			t.Fatalf("unauthenticated status = %d, body = %s", response.StatusCode, response.Body)
		}
		response = postEventPublicationAs(t, handler, marshalPublication(t, publication), "read-only-token")
		if response.StatusCode != http.StatusForbidden {
			t.Fatalf("wrong-scope status = %d, body = %s", response.StatusCode, response.Body)
		}
		assertPublicationDBCounts(t, db, before)
	})

	t.Run("receipt failure rolls back Event evidence and Tag writes", func(t *testing.T) {
		before := readPublicationDBCounts(t, db)
		if _, err := db.Exec(`
CREATE FUNCTION fail_event_publication_receipt_insert()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'forced receipt insert failure';
END;
$$`); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`
CREATE TRIGGER trg_fail_event_publication_receipt_insert
BEFORE INSERT ON event_publication_receipts
FOR EACH STATEMENT
EXECUTE FUNCTION fail_event_publication_receipt_insert()`); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_, _ = db.Exec(`DROP TRIGGER IF EXISTS trg_fail_event_publication_receipt_insert ON event_publication_receipts`)
			_, _ = db.Exec(`DROP FUNCTION IF EXISTS fail_event_publication_receipt_insert()`)
		})

		response := postEventPublication(t, handler, marshalPublication(t, eventPublicationFixture("receipt-failure")))
		if response.StatusCode != http.StatusInternalServerError || response.Error.Code != "EVENT_PUBLICATION_FAILED" {
			t.Fatalf("status = %d, body = %s", response.StatusCode, response.Body)
		}
		assertPublicationDBCounts(t, db, before)
		if _, err := db.Exec(`DROP TRIGGER trg_fail_event_publication_receipt_insert ON event_publication_receipts`); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`DROP FUNCTION fail_event_publication_receipt_insert()`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestEventPublicationMigrationPreservesHistoricalEvidenceContent(t *testing.T) {
	db := openEventPublicationTestDatabaseAt(t, 28)
	const (
		rawID          = "11111111-1111-4111-8111-111111111111"
		eventID        = "22222222-2222-4222-8222-222222222222"
		sourceID       = "33333333-3333-4333-8333-333333333333"
		themeReceiptID = "44444444-4444-4444-8444-444444444444"
		themeID        = "55555555-5555-4555-8555-555555555555"
		anchorReceipt  = "66666666-6666-4666-8666-666666666666"
		anchorID       = "77777777-7777-4777-8777-777777777777"
		chainNodeOne   = "88888888-8888-4888-8888-888888888888"
		chainNodeTwo   = "99999999-9999-4999-8999-999999999999"
	)
	if _, err := db.Exec(`
INSERT INTO raw_documents (
  id, source_id, ingest_channel, source_type, source_name, source_url, title,
  content_text, raw_mime_type, language, collected_at, content_hash, ingest_status
) VALUES (
  $1, 'cd209afe-2ea9-54b8-bdd7-db64eebf0d71', 'legacy', 'news', 'Legacy Source',
  'https://example.test/legacy', 'Legacy document', 'historical full content',
  'text/markdown', 'en', '2026-07-22T00:00:00Z', $2, 'collected'
)`, rawID, fmt.Sprintf("%064x", 7)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO events (
  id, title, summary, first_seen_at, event_status, fact_status, dedupe_key, fact_payload
) VALUES (
  $1, 'Historical Event', 'Historical summary', '2026-07-22T00:00:00Z',
  'confirmed', 'verified', 'event:historical', '{"legacy":true}'::jsonb
)`, eventID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO event_sources (
  id, event_id, raw_document_id, source_level, evidence_excerpt, evidence_hash,
  evidence_relation, supports_fields
) VALUES (
  $1, $2, $3, 'primary', 'Historical excerpt', $4, 'supports', ARRAY['title']
)`, sourceID, eventID, rawID, fmt.Sprintf("%064x", 8)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE events SET primary_source_id = $2 WHERE id = $1`, eventID, sourceID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO entity_nodes (
  id, entity_key, entity_type, layer_code, name, canonical_name, status
) VALUES
  ($1, 'chain_node:migration-preservation-one', 'chain_node', 'chain_node',
   'Migration Preservation One', 'Migration Preservation One', 'active'),
  ($2, 'chain_node:migration-preservation-two', 'chain_node', 'chain_node',
   'Migration Preservation Two', 'Migration Preservation Two', 'active')`,
		chainNodeOne, chainNodeTwo,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO chain_node_profiles (entity_id, definition, boundary_note, review_status)
VALUES
  ($1, 'First preservation node', 'First preservation boundary', 'approved'),
  ($2, 'Second preservation node', 'Second preservation boundary', 'approved')`,
		chainNodeOne, chainNodeTwo,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO research_theme_import_receipts (
  id, analysis_batch_id, publisher_subject, payload_hash, theme_ids_by_key,
  write_counts, published_at, imported_at
) VALUES (
  $1, 'migration-preservation-batch', 'migration-test', $2,
  jsonb_build_object('migration-preservation-theme', $3::text),
  '{"themes":1,"chain_node_associations":1,"event_associations":1,"receipts":1}'::jsonb,
  '2026-07-22T01:00:00Z', '2026-07-22T01:00:01Z'
)`, themeReceiptID, fmt.Sprintf("%064x", 9), themeID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO research_themes (
  id, analysis_batch_id, theme_key, import_receipt_id, name, one_line_conclusion,
  impact_level, transmission_path, trading_direction, transmission_stage,
  next_checkpoint, market_confirmation_summary, window_start, window_end, published_at
) VALUES (
  $1, 'migration-preservation-batch', 'migration-preservation-theme', $2,
  'Preserved Theme', 'Preserved theme conclusion', 'high',
  'Event to node transmission', 'Track the preserved direction', 'validation',
  'Verify the next checkpoint', 'Market evidence remains preserved',
  '2026-07-21T00:00:00Z', '2026-07-22T00:00:00Z', '2026-07-22T01:00:00Z'
)`, themeID, themeReceiptID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO research_theme_chain_nodes (
  theme_id, chain_node_entity_id, relation_role, impact_summary
) VALUES ($1, $2, 'driver', 'Preserved Theme node association')`,
		themeID, chainNodeOne,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO research_theme_events (theme_id, event_id, evidence_role, supported_claim)
VALUES ($1, $2, 'driver', 'Preserved Theme Event claim')`, themeID, eventID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO research_anchor_import_receipts (
  id, theme_id, publisher_subject, payload_hash, anchor_ids_by_center_chain_node_id,
  write_counts, published_at, imported_at
) VALUES (
  $1, $2, 'migration-test', $3,
  jsonb_build_object($4::text, $5::text),
  '{"anchors":1,"event_associations":1,"path_nodes":2,"receipts":1}'::jsonb,
  '2026-07-22T01:00:00Z', '2026-07-22T01:00:01Z'
)`, anchorReceipt, themeID, fmt.Sprintf("%064x", 10), chainNodeOne, anchorID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO research_anchors (
  id, theme_id, center_chain_node_entity_id, import_receipt_id,
  one_line_conclusion, fact_summary, net_direction_summary, trading_direction,
  next_checkpoint, support_summary, counter_summary
) VALUES (
  $1, $2, $3, $4, 'Preserved Anchor conclusion', 'Preserved facts',
  'Preserved net direction', 'Preserved trading direction',
  'Preserved checkpoint', 'Preserved support summary', 'Preserved counter summary'
)`, anchorID, themeID, chainNodeOne, anchorReceipt); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO research_anchor_chain_nodes (
  anchor_id, position, chain_node_entity_id, change_direction,
  change_summary, impact_summary, incoming_transmission_mechanism
) VALUES
  ($1, 1, $2, 'increase', 'First preserved change', 'First preserved impact', NULL),
  ($1, 2, $3, 'increase', 'Second preserved change', 'Second preserved impact',
   'Preserved transmission mechanism')`,
		anchorID, chainNodeOne, chainNodeTwo,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO research_anchor_events (anchor_id, event_id, evidence_role, evidence_summary)
VALUES ($1, $2, 'driver', 'Preserved Anchor Event summary')`, anchorID, eventID); err != nil {
		t.Fatal(err)
	}

	applyEventPublicationMigration(t, db, 29)

	var content string
	var contractVersion int
	var artifactID *string
	if err := db.QueryRow(`
SELECT content_text, contract_version, artifact_id
FROM raw_documents WHERE id = $1`, rawID).Scan(&content, &contractVersion, &artifactID); err != nil {
		t.Fatal(err)
	}
	if content != "historical full content" || contractVersion != 1 || artifactID != nil {
		t.Fatalf("historical raw document = content %q, version %d, artifact %v", content, contractVersion, artifactID)
	}
	var linked int
	if err := db.QueryRow(`
SELECT count(*)
FROM events e
JOIN event_sources es ON es.id = e.primary_source_id AND es.event_id = e.id
JOIN raw_documents rd ON rd.id = es.raw_document_id
WHERE e.id = $1 AND rd.id = $2`, eventID, rawID).Scan(&linked); err != nil {
		t.Fatal(err)
	}
	if linked != 1 {
		t.Fatalf("historical Event evidence link count = %d, want 1", linked)
	}
	var preservedResearchRows int
	if err := db.QueryRow(`
SELECT
  (SELECT count(*) FROM research_themes
    WHERE id = $1 AND name = 'Preserved Theme')
  + (SELECT count(*) FROM research_theme_chain_nodes
    WHERE theme_id = $1 AND chain_node_entity_id = $3)
  + (SELECT count(*) FROM research_theme_events
    WHERE theme_id = $1 AND event_id = $2 AND supported_claim = 'Preserved Theme Event claim')
  + (SELECT count(*) FROM research_anchors
    WHERE id = $4 AND theme_id = $1 AND one_line_conclusion = 'Preserved Anchor conclusion')
  + (SELECT count(*) FROM research_anchor_chain_nodes
    WHERE anchor_id = $4 AND position IN (1, 2))
  + (SELECT count(*) FROM research_anchor_events
    WHERE anchor_id = $4 AND event_id = $2 AND evidence_summary = 'Preserved Anchor Event summary')`,
		themeID, eventID, chainNodeOne, anchorID,
	).Scan(&preservedResearchRows); err != nil {
		t.Fatal(err)
	}
	if preservedResearchRows != 7 {
		t.Fatalf("preserved Research row count = %d, want 7", preservedResearchRows)
	}
	for _, table := range []string{
		"events", "event_sources", "event_tag_defs", "event_tag_maps", "research_themes", "research_anchors",
	} {
		if !relationExists(t, db, table) {
			t.Fatalf("preserved table %s does not exist", table)
		}
	}
	for _, table := range []string{
		"source_catalogs", "ingestion_run_sources", "ingestion_runs",
		"ingestion_scheduler_configs", "raw_document_import_receipts", "event_import_receipts",
	} {
		if relationExists(t, db, table) {
			t.Fatalf("retired table %s still exists", table)
		}
	}
}

func TestEventPublicationV3MigrationPreservesEvidenceAndRemovesPrimarySemantics(t *testing.T) {
	db := openEventPublicationTestDatabaseAt(t, 37)
	const (
		rawID    = "a1000000-0000-4000-8000-000000000001"
		eventID  = "a1000000-0000-4000-8000-000000000002"
		sourceID = "a1000000-0000-4000-8000-000000000003"
	)
	if _, err := db.Exec(`
INSERT INTO raw_documents (
  id, contract_version, artifact_id, source_ref, source_type,
  source_name, source_url, title, content_text, raw_mime_type, language,
  collected_at, content_hash, ingest_status
) VALUES (
  $1, 2, 'artifact:migration-v3', 'source:migration-v3', 'news',
  'Migration Source', 'https://example.test/migration-v3', 'Migration V3 source',
  '', 'text/markdown', 'en', '2026-08-02T00:00:00Z', $2, 'collected'
)`, rawID, strings.Repeat("a", 64)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO events (
  id, title, summary, first_seen_at, event_status, fact_status, dedupe_key, fact_payload
) VALUES (
  $1, 'Migration V3 Event', 'Migration V3 summary', '2026-08-02T00:00:00Z',
  'confirmed', 'verified', 'event:migration-v3', '{}'::jsonb
)`, eventID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO event_sources (
  id, contract_version, event_id, raw_document_id, source_level,
  evidence_excerpt, evidence_hash, evidence_relation, supports_fields, is_primary
) VALUES (
  $1, 2, $2, $3, 'primary', 'Preserved evidence statement', $4,
  'supports', ARRAY['title'], true
)`, sourceID, eventID, rawID, strings.Repeat("b", 64)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`UPDATE events SET primary_source_id = $2 WHERE id = $1`, eventID, sourceID,
	); err != nil {
		t.Fatal(err)
	}

	applyEventPublicationMigration(t, db, 38)

	var statement string
	var contractVersion int
	if err := db.QueryRow(`
SELECT evidence_statement, contract_version FROM event_sources WHERE id = $1
`, sourceID).Scan(&statement, &contractVersion); err != nil {
		t.Fatal(err)
	}
	if statement != "Preserved evidence statement" || contractVersion != 3 {
		t.Fatalf("migrated evidence = %q, version = %d", statement, contractVersion)
	}
	var retiredColumns int
	if err := db.QueryRow(`
SELECT count(*) FROM information_schema.columns
WHERE table_schema = current_schema()
  AND ((table_name = 'events' AND column_name = 'primary_source_id')
    OR (table_name = 'event_sources' AND column_name = 'is_primary'))
`).Scan(&retiredColumns); err != nil {
		t.Fatal(err)
	}
	if retiredColumns != 0 {
		t.Fatalf("retired primary Evidence columns remaining = %d", retiredColumns)
	}
}

type capturingEventPublicationService struct {
	delegate  *eventbiz.UseCase
	lastError error
}

func (s *capturingEventPublicationService) Import(
	ctx context.Context,
	callerSubject string,
	publication eventbiz.PublicationBatch,
) (eventbiz.Result, error) {
	result, err := s.delegate.Import(ctx, callerSubject, publication)
	s.lastError = err
	return result, err
}

func (s *capturingEventPublicationService) ActiveTags(ctx context.Context) (eventbiz.EventTagCatalog, error) {
	return s.delegate.ActiveTags(ctx)
}

func (s *capturingEventPublicationService) ListEvents(ctx context.Context, request eventbiz.EventListRequest) (eventbiz.EventPage, error) {
	return s.delegate.ListEvents(ctx, request)
}

type eventPublicationHTTPResult struct {
	StatusCode int
	Body       string
	Error      eventPublicationHTTPError `json:"error"`
	Result     struct {
		ReceiptID string `json:"receipt_id"`
		Events    []struct {
			DedupeKey   string `json:"dedupe_key"`
			EventID     string `json:"event_id"`
			Disposition string `json:"disposition"`
		} `json:"events"`
		RawDocuments []struct {
			ArtifactID    string `json:"artifact_id"`
			RawDocumentID string `json:"raw_document_id"`
			Disposition   string `json:"disposition"`
		} `json:"raw_documents"`
		Counts struct {
			EventsCreated       int `json:"events_created"`
			EventsReused        int `json:"events_reused"`
			RawDocumentsCreated int `json:"raw_documents_created"`
			RawDocumentsReused  int `json:"raw_documents_reused"`
			EventSourcesCreated int `json:"event_sources_created"`
			EventSourcesReused  int `json:"event_sources_reused"`
			EventTagsCreated    int `json:"event_tags_created"`
			EventTagsReused     int `json:"event_tags_reused"`
		} `json:"counts"`
	} `json:"result"`
}

type eventPublicationHTTPError struct {
	Code    string          `json:"code"`
	Details json.RawMessage `json:"details"`
}

type eventPublicationValidationIssue struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e eventPublicationHTTPError) ValidationIssues(t *testing.T) []eventPublicationValidationIssue {
	t.Helper()
	var details struct {
		Issues []eventPublicationValidationIssue `json:"issues"`
	}
	if err := json.Unmarshal(e.Details, &details); err != nil {
		t.Fatalf("decode validation issues: %v\n%s", err, e.Details)
	}
	return details.Issues
}

func postEventPublication(t *testing.T, handler http.Handler, body []byte) eventPublicationHTTPResult {
	return postEventPublicationAs(t, handler, body, "agent-token")
}

func postEventPublicationAs(t *testing.T, handler http.Handler, body []byte, token string) eventPublicationHTTPResult {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, v1.APIPrefix+"/reviewed-event-imports", bytes.NewReader(body))
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	result := eventPublicationHTTPResult{StatusCode: response.Code, Body: response.Body.String()}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v\n%s", err, response.Body.String())
	}
	return result
}

func openEventPublicationTestDatabase(t *testing.T) *sql.DB {
	return openEventPublicationTestDatabaseAt(t, 0)
}

func openEventPublicationTestDatabaseAt(t *testing.T, version int64) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_event_publication", migrationDir, version)
}

func applyEventPublicationMigration(t *testing.T, db *sql.DB, version int64) {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	postgresfixture.ApplyMigration(t, db, migrationDir, version)
}

func newEventPublicationTestHandler(t *testing.T, db *sql.DB) (http.Handler, *capturingEventPublicationService) {
	t.Helper()
	store, err := eventdata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := eventbiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	service := &capturingEventPublicationService{delegate: useCase}
	application, err := NewService(service)
	if err != nil {
		t.Fatal(err)
	}
	credentials := map[string]v1.Principal{
		"agent-token": {
			Identity: "agent-run",
			Scopes:   []string{serverpkg.ScopeReviewedEventImport},
		},
		"read-only-token": {
			Identity: "read-only-client",
			Scopes:   []string{serverpkg.ScopeResearchRead},
		},
	}
	return newEventHTTPHandler(t, application, credentials), service
}

func newEventHTTPHandler(t *testing.T, application *Service, credentials map[string]v1.Principal) http.Handler {
	t.Helper()
	configured := make([]serverpkg.Credential, 0, len(credentials))
	for secret, principal := range credentials {
		configured = append(configured, serverpkg.Credential{Secret: secret, Principal: principal})
	}
	authenticator, err := serverpkg.NewAuthenticator(configured)
	if err != nil {
		t.Fatal(err)
	}
	httpServer, err := serverpkg.NewHTTPServer(conf.Config{
		App:    conf.AppConfig{Env: conf.EnvLocal},
		Server: conf.ServerConfig{Host: "127.0.0.1", Port: 18081, ReadTimeoutSeconds: 5, WriteTimeoutSeconds: 10},
	}, parentservice.NewDataService(parentservice.Dependencies{}), application, testEventSemanticService{}, testEvidenceService{}, authenticator, nil)
	if err != nil {
		t.Fatal(err)
	}
	return httpServer.Server.Handler
}

type testEvidenceService struct{}

type testEventSemanticService struct{}

func (testEventSemanticService) ListEligibleEventSemanticEvents(context.Context, *eventsemanticapi.EligibleEventSemanticEventsRequest) (*v1.Response[eventsemanticapi.EligibleEventSemanticEvents], error) {
	return &v1.Response[eventsemanticapi.EligibleEventSemanticEvents]{Status: http.StatusNoContent}, nil
}
func (testEventSemanticService) CreateEventSemanticContextLease(context.Context, *eventsemanticapi.EventSemanticContextLeaseRequest) (*v1.Response[eventsemanticapi.EventSemanticContextLease], error) {
	return &v1.Response[eventsemanticapi.EventSemanticContextLease]{Status: http.StatusNoContent}, nil
}
func (testEventSemanticService) GetEventSemanticContext(context.Context, *eventsemanticapi.EventSemanticContextRequest) (*v1.Response[eventsemanticapi.EventSemanticContext], error) {
	return &v1.Response[eventsemanticapi.EventSemanticContext]{Status: http.StatusNoContent}, nil
}
func (testEventSemanticService) CreateEventSemanticSubmission(context.Context, *eventsemanticapi.EventSemanticSubmissionRequest) (*v1.Response[eventsemanticapi.EventSemanticSubmissionResult], error) {
	return &v1.Response[eventsemanticapi.EventSemanticSubmissionResult]{Status: http.StatusNoContent}, nil
}
func (testEventSemanticService) SubmitEventSemanticReview(context.Context, *eventsemanticapi.EventSemanticReviewRequest) (*v1.Response[eventsemanticapi.EventSemanticSubmissionResult], error) {
	return &v1.Response[eventsemanticapi.EventSemanticSubmissionResult]{Status: http.StatusNoContent}, nil
}
func (testEventSemanticService) GetEventSemantics(context.Context, *eventsemanticapi.GetEventSemanticsRequest) (*v1.Response[eventsemanticapi.EventSemanticsResult], error) {
	return &v1.Response[eventsemanticapi.EventSemanticsResult]{Status: http.StatusNoContent}, nil
}

func (testEvidenceService) PublishRawEvidence(context.Context, *evidenceapi.RawEvidencePublicationRequest) (*v1.Response[evidenceapi.RawEvidencePublicationResult], error) {
	return &v1.Response[evidenceapi.RawEvidencePublicationResult]{Status: http.StatusNoContent}, nil
}

func (testEvidenceService) PublishEvidence(context.Context, *evidenceapi.EvidencePublicationRequest) (*v1.Response[evidenceapi.EvidencePublicationResult], error) {
	return &v1.Response[evidenceapi.EvidencePublicationResult]{Status: http.StatusNoContent}, nil
}

func eventPublicationFixture(suffix string) eventbiz.PublicationBatch {
	publishedAt := time.Date(2026, 7, 23, 1, 0, 0, 0, time.UTC)
	collectedAt := time.Date(2026, 7, 23, 1, 5, 0, 0, time.UTC)
	occurredAt := time.Date(2026, 7, 23, 0, 30, 0, 0, time.UTC)
	artifactID := "artifact-" + suffix
	return eventbiz.PublicationBatch{
		PackageID: "package-" + suffix,
		Provenance: eventbiz.Provenance{
			ExtractorExecutionID:  "extractor-" + suffix,
			ExtractorAgentVersion: "event-extractor-v2",
			CollectorExecutions: []eventbiz.CollectorExecution{{
				ArtifactID: artifactID, CollectorExecutionID: "collector-" + suffix,
			}},
		},
		RawDocuments: []eventbiz.EventEvidenceRecord{{
			ArtifactID: artifactID, ContentSHA256: fmt.Sprintf("%064x", len(suffix)+10),
			SourceRef: "source:" + suffix, SourceName: "Source " + suffix, SourceType: "news",
			SourceURL: "https://example.test/" + url.PathEscape(suffix), Title: "Source " + suffix,
			PublishedAt: &publishedAt, CollectedAt: collectedAt, Language: "en", MIMEType: "text/markdown",
		}},
		Events: []eventbiz.PublicationEvent{{
			DedupeKey: "event:" + suffix + ":1", Title: "Event " + suffix,
			FactualSummary: "A verifiable state change occurred for " + suffix + ".",
			OccurredAt:     &occurredAt,
			FactPayload:    map[string]any{"fixture": suffix},
			Evidence: []eventbiz.EventEvidenceLinkInput{{
				ArtifactID: artifactID, EvidenceRelation: "supports",
				EvidenceStatement: "Evidence for " + suffix,
				SupportsFields:    []string{"title", "factual_summary"},
				SourceLevel:       "primary",
			}},
			Tags: []eventbiz.EventTagInput{{
				TagID:   "22a5afc5-20ed-55ce-bf77-54c26bbcc6ea",
				TagKind: "news_category", TagCode: "technology_industry",
				Confidence: json.Number("0.94"), AssignmentReason: "Technology event",
				AssignSource: "ai",
			}},
			Review: eventbiz.Review{
				ReviewID: "review-" + suffix, EvidenceGrade: "A", Reasons: []string{"Reviewed"},
			},
		}},
	}
}

func clonePublicationEvent(input eventbiz.PublicationEvent, suffix string) eventbiz.PublicationEvent {
	cloned := input
	cloned.DedupeKey = "event:" + suffix + ":1"
	cloned.Title = "Event " + suffix
	cloned.FactualSummary = "A verifiable state change occurred for " + suffix + "."
	cloned.FactPayload = map[string]any{"fixture": suffix}
	cloned.Evidence = append([]eventbiz.EventEvidenceLinkInput(nil), input.Evidence...)
	cloned.Tags = append([]eventbiz.EventTagInput(nil), input.Tags...)
	cloned.Review = eventbiz.Review{
		ReviewID: "review-" + suffix, EvidenceGrade: "A", Reasons: []string{"Reviewed"},
	}
	return cloned
}

func marshalPublication(t *testing.T, publication eventbiz.PublicationBatch) []byte {
	t.Helper()
	return mustJSON(t, publication)
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

type publicationDBCounts struct {
	RawDocuments int
	Events       int
	EventSources int
	EventTags    int
	Receipts     int
}

func readPublicationDBCounts(t *testing.T, db *sql.DB) publicationDBCounts {
	t.Helper()
	var counts publicationDBCounts
	if err := db.QueryRow(`
SELECT
  (SELECT count(*) FROM raw_documents WHERE contract_version = 2),
  (SELECT count(*) FROM events),
  (SELECT count(*) FROM event_sources WHERE contract_version = 3),
  (SELECT count(*) FROM event_tag_maps),
  (SELECT count(*) FROM event_publication_receipts)`).Scan(
		&counts.RawDocuments,
		&counts.Events,
		&counts.EventSources,
		&counts.EventTags,
		&counts.Receipts,
	); err != nil {
		t.Fatal(err)
	}
	return counts
}

func assertPublicationDBCounts(t *testing.T, db *sql.DB, want publicationDBCounts) {
	t.Helper()
	if got := readPublicationDBCounts(t, db); got != want {
		t.Fatalf("database counts = %#v, want %#v", got, want)
	}
}

func relationExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var exists bool
	if err := db.QueryRow(`SELECT to_regclass($1) IS NOT NULL`, name).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	return exists
}
