package event_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	chainnodeapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/chainnode"
	conceptapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/concept"
	countryapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/country"
	industryapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/industry"
	industrychainapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/industrychain"
	organizationapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/organization"
	eventapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/event"
	evidenceapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/evidence"
	rawdocumentapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/rawdocument"
	runtimehealthapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/runtimehealth"
	eventbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/event"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	eventdata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/event"
	serverpkg "github.com/meierlink88/tidewise-ai/data-service/backend/internal/server"
	eventservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/event"
	eventfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/event"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
	researchfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/research"
)

type eventHTTPOrganizationStub struct{}

func (eventHTTPOrganizationStub) Create(context.Context, *organizationapi.CreateRequest) (*v1.Response[organizationapi.Organization], error) {
	return &v1.Response[organizationapi.Organization]{Status: http.StatusNoContent}, nil
}
func (eventHTTPOrganizationStub) List(context.Context, *organizationapi.ListRequest) (*v1.Response[organizationapi.OrganizationList], error) {
	return &v1.Response[organizationapi.OrganizationList]{Status: http.StatusNoContent}, nil
}
func (eventHTTPOrganizationStub) Get(context.Context, *organizationapi.GetRequest) (*v1.Response[organizationapi.Organization], error) {
	return &v1.Response[organizationapi.Organization]{Status: http.StatusNoContent}, nil
}
func (eventHTTPOrganizationStub) Update(context.Context, *organizationapi.UpdateRequest) (*v1.Response[organizationapi.Organization], error) {
	return &v1.Response[organizationapi.Organization]{Status: http.StatusNoContent}, nil
}
func (eventHTTPOrganizationStub) ReplaceDomainTags(context.Context, *organizationapi.ReplaceDomainTagsRequest) (*v1.Response[organizationapi.Organization], error) {
	return &v1.Response[organizationapi.Organization]{Status: http.StatusNoContent}, nil
}
func (eventHTTPOrganizationStub) GetCatalog(context.Context, *organizationapi.CatalogRequest) (*v1.Response[organizationapi.Catalog], error) {
	return &v1.Response[organizationapi.Catalog]{Status: http.StatusNoContent}, nil
}
func (eventHTTPOrganizationStub) ListMembers(context.Context, *organizationapi.ListMembersRequest) (*v1.Response[organizationapi.MemberList], error) {
	return &v1.Response[organizationapi.MemberList]{Status: http.StatusNoContent}, nil
}
func (eventHTTPOrganizationStub) CreateMember(context.Context, *organizationapi.CreateMemberRequest) (*v1.Response[organizationapi.Member], error) {
	return &v1.Response[organizationapi.Member]{Status: http.StatusNoContent}, nil
}
func (eventHTTPOrganizationStub) UpdateMember(context.Context, *organizationapi.UpdateMemberRequest) (*v1.Response[organizationapi.Member], error) {
	return &v1.Response[organizationapi.Member]{Status: http.StatusNoContent}, nil
}
func (eventHTTPOrganizationStub) DeleteMember(context.Context, *organizationapi.DeleteMemberRequest) (*v1.Response[organizationapi.DeleteResult], error) {
	return &v1.Response[organizationapi.DeleteResult]{Status: http.StatusNoContent}, nil
}

func TestHTTPRoutesPreserveEventContract(t *testing.T) {
	for _, test := range []struct {
		method, path, body, operation string
	}{
		{http.MethodPost, v1.APIPrefix + "/reviewed-event-imports", `{}`, eventapi.OperationPublishReviewedEvents},
		{http.MethodGet, v1.APIPrefix + "/event-tags?active=true", "", eventapi.OperationListActiveEventTags},
		{http.MethodGet, v1.APIPrefix + "/events", "", eventapi.OperationListAdminEvents},
	} {
		t.Run(test.operation, func(t *testing.T) {
			var operation string
			recorder := func(next middleware.Handler) middleware.Handler {
				return func(ctx context.Context, request any) (any, error) {
					if serverTransport, ok := transport.FromServerContext(ctx); ok {
						operation = serverTransport.Operation()
					}
					return next(ctx, request)
				}
			}
			server := kratoshttp.NewServer(kratoshttp.Middleware(recorder))
			eventapi.RegisterHTTPServer(server, testService{})
			response := httptest.NewRecorder()
			server.ServeHTTP(response, httptest.NewRequest(test.method, test.path, strings.NewReader(test.body)))
			if response.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want 204: %s", response.Code, response.Body.String())
			}
			if operation != test.operation {
				t.Fatalf("operation = %q, want %q", operation, test.operation)
			}
		})
	}
}

func TestHTTPEventTagCatalogReturnsOnlyCurrentTags(t *testing.T) {
	server := kratoshttp.NewServer()
	eventapi.RegisterHTTPServer(server, currentTagCatalogService{testService: testService{}})
	response := httptest.NewRecorder()
	server.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, v1.APIPrefix+"/event-tags?active=true", nil),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var result map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if _, exists := result["catalog_revision"]; exists {
		t.Fatalf("response contains catalog_revision: %s", response.Body.String())
	}
	if _, exists := result["catalog_hash"]; exists {
		t.Fatalf("response contains catalog_hash: %s", response.Body.String())
	}
	if _, exists := result["tags"]; !exists || len(result) != 1 {
		t.Fatalf("result = %s", response.Body.String())
	}
}

func TestPublicationBindingAllowsArbitraryFactPayloadButRejectsUnknownFields(t *testing.T) {
	server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(func(response http.ResponseWriter, _ *http.Request, err error) {
		if public, ok := err.(*v1.PublicError); ok {
			response.WriteHeader(public.Status)
			return
		}
		response.WriteHeader(http.StatusInternalServerError)
	}))
	eventapi.RegisterHTTPServer(server, testService{})
	valid := `{"package_id":"package","provenance":{"extractor_execution_id":"execution","extractor_agent_version":"v1","collector_executions":[]},"raw_documents":[],"events":[{"dedupe_key":"event","title":"title","factual_summary":"summary","fact_payload":{"nested":{"value":[1,true,null]}},"evidence":[],"tags":[],"review":{"review_id":"review","evidence_grade":"A","reasons":[]}}]}`
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodPost, v1.APIPrefix+"/reviewed-event-imports", strings.NewReader(valid)))
	if response.Code != http.StatusNoContent {
		t.Fatalf("valid status = %d: %s", response.Code, response.Body.String())
	}

	unknown := httptest.NewRecorder()
	server.ServeHTTP(unknown, httptest.NewRequest(http.MethodPost, v1.APIPrefix+"/reviewed-event-imports", strings.NewReader(`{"package_id":"package","unexpected":true}`)))
	if unknown.Code != http.StatusBadRequest {
		t.Fatalf("unknown-field status = %d, want 400", unknown.Code)
	}
	nullable := httptest.NewRecorder()
	server.ServeHTTP(nullable, httptest.NewRequest(http.MethodPost, v1.APIPrefix+"/reviewed-event-imports", strings.NewReader(`{"package_id":null}`)))
	if nullable.Code != http.StatusNoContent {
		t.Fatalf("legacy null scalar status = %d, want 204", nullable.Code)
	}
}

type testService struct{}

type currentTagCatalogService struct{ testService }

func (currentTagCatalogService) ListActiveEventTags(
	context.Context,
	*eventapi.TagCatalogRequest,
) (*v1.Response[eventapi.TagCatalog], error) {
	return &v1.Response[eventapi.TagCatalog]{
		Status: http.StatusOK,
		Result: eventapi.TagCatalog{Tags: []eventapi.TagCatalogItem{{
			ID: "11111111-1111-4111-8111-111111111111", TagKind: "news_category",
			Code: "technology_industry", Name: "科技产业", IsActive: true,
		}}},
	}, nil
}

func (testService) PublishReviewedEvents(context.Context, *eventapi.PublicationRequest) (*v1.Response[eventapi.PublicationResult], error) {
	return testResponse[eventapi.PublicationResult]()
}

func (testService) ListActiveEventTags(context.Context, *eventapi.TagCatalogRequest) (*v1.Response[eventapi.TagCatalog], error) {
	return testResponse[eventapi.TagCatalog]()
}

func (testService) ListEvents(context.Context, *eventapi.ListRequest) (*v1.Response[eventapi.Page], error) {
	return testResponse[eventapi.Page]()
}

func testResponse[T any]() (*v1.Response[T], error) {
	return &v1.Response[T]{Status: http.StatusNoContent}, nil
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
	          "tag_id": "ETD22a5afc5-20ed-55ce-bf77-54c26bbcc6ea",
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

func TestPostgresEventPublicationContractScenarios(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	handler, service := newEventPublicationTestHandler(t, db)
	t.Run("repeated Event accumulates equal Evidence across publication batches", func(t *testing.T) {
		first := eventfixture.Publication("multi-source-event")
		firstResponse := postEventPublication(t, handler, marshalPublication(t, first))
		if firstResponse.StatusCode != http.StatusCreated {
			t.Fatalf("first status = %d, body = %s", firstResponse.StatusCode, firstResponse.Body)
		}

		second := eventfixture.Publication("multi-source-event")
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
		publication := eventfixture.Publication("relationships")
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

		secondEvent := eventfixture.ClonePublicationEvent(publication.Events[0], "relationships-2")
		thirdEvent := eventfixture.ClonePublicationEvent(publication.Events[0], "relationships-3")
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
		unreferenced := eventfixture.Publication("unreferenced")
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

		unknown := eventfixture.Publication("unknown-artifact")
		unknown.Events[0].Evidence[0].ArtifactID = "artifact-not-declared"
		response = postEventPublication(t, handler, marshalPublication(t, unknown))
		if response.StatusCode != http.StatusUnprocessableEntity || response.Error.Code != "EVENT_PUBLICATION_INVALID" {
			t.Fatalf("unknown status = %d, body = %s", response.StatusCode, response.Body)
		}
		assertPublicationDBCounts(t, db, before)
	})

	t.Run("artifact conflict rolls back the whole batch", func(t *testing.T) {
		base := eventfixture.Publication("artifact-conflict")
		first := postEventPublication(t, handler, marshalPublication(t, base))
		if first.StatusCode != http.StatusCreated {
			t.Fatalf("first status = %d, body = %s", first.StatusCode, first.Body)
		}
		before := readPublicationDBCounts(t, db)
		conflict := eventfixture.Publication("artifact-conflict-second")
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
		base := eventfixture.Publication("event-conflict")
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
		base := eventfixture.Publication("large-number-conflict")
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
		base := eventfixture.Publication("supports-fields-order")
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
				publication := eventfixture.Publication("tag-" + test.name)
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

		invalidReview := eventfixture.Publication("invalid-review")
		invalidReview.Events[0].Review.Reasons = nil
		response := postEventPublication(t, handler, marshalPublication(t, invalidReview))
		if response.StatusCode != http.StatusBadRequest || response.Error.Code != "INVALID_REQUEST" {
			t.Fatalf("review status = %d, body = %s", response.StatusCode, response.Body)
		}
		assertPublicationDBCounts(t, db, before)

		tooManyNewsTags := eventfixture.Publication("too-many-news-tags")
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
		publication := eventfixture.Publication("transport")
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

		invalid := eventfixture.Publication("sorted-errors")
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

		response := postEventPublication(t, handler, marshalPublication(t, eventfixture.Publication("receipt-failure")))
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
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_event_publication", migrationDir, version)
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
	application, err := eventservice.NewService(service)
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

func newEventHTTPHandler(t *testing.T, application *eventservice.Service, credentials map[string]v1.Principal) http.Handler {
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
	}, testRuntimeHealthService{}, researchfixture.Service{}, application, testEvidenceService{}, testRawDocumentService{}, testCountryService{}, testIndustryService{}, testConceptService{}, testChainNodeService{}, testIndustryChainService{}, eventHTTPOrganizationStub{}, authenticator, nil)
	if err != nil {
		t.Fatal(err)
	}
	return httpServer.Server.Handler
}

type testRuntimeHealthService struct{}

type testCountryService struct{}

type testIndustryService struct{}

type testConceptService struct{}

type testChainNodeService struct{}

type testIndustryChainService struct{}

func (testCountryService) Create(context.Context, *countryapi.CreateRequest) (*v1.Response[countryapi.Country], error) {
	return &v1.Response[countryapi.Country]{Status: http.StatusNoContent}, nil
}

func (testCountryService) List(context.Context, *countryapi.ListRequest) (*v1.Response[countryapi.CountryList], error) {
	return &v1.Response[countryapi.CountryList]{Status: http.StatusNoContent}, nil
}

func (testCountryService) Get(context.Context, *countryapi.GetRequest) (*v1.Response[countryapi.Country], error) {
	return &v1.Response[countryapi.Country]{Status: http.StatusNoContent}, nil
}

func (testCountryService) Update(context.Context, *countryapi.UpdateRequest) (*v1.Response[countryapi.Country], error) {
	return &v1.Response[countryapi.Country]{Status: http.StatusNoContent}, nil
}

func (testCountryService) ReplaceRegions(context.Context, *countryapi.ReplaceRegionsRequest) (*v1.Response[countryapi.Country], error) {
	return &v1.Response[countryapi.Country]{Status: http.StatusNoContent}, nil
}

func (testIndustryService) Create(context.Context, *industryapi.CreateRequest) (*v1.Response[industryapi.Industry], error) {
	return &v1.Response[industryapi.Industry]{Status: http.StatusNoContent}, nil
}
func (testIndustryService) List(context.Context, *industryapi.ListRequest) (*v1.Response[industryapi.IndustryList], error) {
	return &v1.Response[industryapi.IndustryList]{Status: http.StatusNoContent}, nil
}
func (testIndustryService) Get(context.Context, *industryapi.GetRequest) (*v1.Response[industryapi.Industry], error) {
	return &v1.Response[industryapi.Industry]{Status: http.StatusNoContent}, nil
}
func (testIndustryService) Update(context.Context, *industryapi.UpdateRequest) (*v1.Response[industryapi.Industry], error) {
	return &v1.Response[industryapi.Industry]{Status: http.StatusNoContent}, nil
}

func (testConceptService) Create(context.Context, *conceptapi.CreateRequest) (*v1.Response[conceptapi.Concept], error) {
	return &v1.Response[conceptapi.Concept]{Status: http.StatusNoContent}, nil
}
func (testConceptService) List(context.Context, *conceptapi.ListRequest) (*v1.Response[conceptapi.ConceptList], error) {
	return &v1.Response[conceptapi.ConceptList]{Status: http.StatusNoContent}, nil
}
func (testConceptService) Get(context.Context, *conceptapi.GetRequest) (*v1.Response[conceptapi.Concept], error) {
	return &v1.Response[conceptapi.Concept]{Status: http.StatusNoContent}, nil
}
func (testConceptService) Update(context.Context, *conceptapi.UpdateRequest) (*v1.Response[conceptapi.Concept], error) {
	return &v1.Response[conceptapi.Concept]{Status: http.StatusNoContent}, nil
}

func (testChainNodeService) Create(context.Context, *chainnodeapi.CreateRequest) (*v1.Response[chainnodeapi.ChainNode], error) {
	return &v1.Response[chainnodeapi.ChainNode]{Status: http.StatusNoContent}, nil
}
func (testChainNodeService) List(context.Context, *chainnodeapi.ListRequest) (*v1.Response[chainnodeapi.ChainNodeList], error) {
	return &v1.Response[chainnodeapi.ChainNodeList]{Status: http.StatusNoContent}, nil
}
func (testChainNodeService) Get(context.Context, *chainnodeapi.GetRequest) (*v1.Response[chainnodeapi.ChainNode], error) {
	return &v1.Response[chainnodeapi.ChainNode]{Status: http.StatusNoContent}, nil
}
func (testChainNodeService) Update(context.Context, *chainnodeapi.UpdateRequest) (*v1.Response[chainnodeapi.ChainNode], error) {
	return &v1.Response[chainnodeapi.ChainNode]{Status: http.StatusNoContent}, nil
}

func (testIndustryChainService) Create(context.Context, *industrychainapi.CreateRequest) (*v1.Response[industrychainapi.IndustryChain], error) {
	return &v1.Response[industrychainapi.IndustryChain]{Status: http.StatusNoContent}, nil
}
func (testIndustryChainService) List(context.Context, *industrychainapi.ListRequest) (*v1.Response[industrychainapi.IndustryChainList], error) {
	return &v1.Response[industrychainapi.IndustryChainList]{Status: http.StatusNoContent}, nil
}
func (testIndustryChainService) Get(context.Context, *industrychainapi.GetRequest) (*v1.Response[industrychainapi.IndustryChain], error) {
	return &v1.Response[industrychainapi.IndustryChain]{Status: http.StatusNoContent}, nil
}
func (testIndustryChainService) Update(context.Context, *industrychainapi.UpdateRequest) (*v1.Response[industrychainapi.IndustryChain], error) {
	return &v1.Response[industrychainapi.IndustryChain]{Status: http.StatusNoContent}, nil
}

func (testRuntimeHealthService) GetRuntimeHealth(context.Context, *runtimehealthapi.Request) (*v1.Response[runtimehealthapi.Result], error) {
	return &v1.Response[runtimehealthapi.Result]{Status: http.StatusNoContent}, nil
}

type testEvidenceService struct{}

type testRawDocumentService struct{}

func (testRawDocumentService) List(context.Context, *rawdocumentapi.ListRequest) (*v1.Response[rawdocumentapi.Page], error) {
	return &v1.Response[rawdocumentapi.Page]{Status: http.StatusNoContent}, nil
}

func (testEvidenceService) PublishRawEvidence(context.Context, *evidenceapi.RawEvidencePublicationRequest) (*v1.Response[evidenceapi.RawEvidencePublicationResult], error) {
	return &v1.Response[evidenceapi.RawEvidencePublicationResult]{Status: http.StatusNoContent}, nil
}

func (testEvidenceService) GetRawEvidence(context.Context, *evidenceapi.GetRawEvidenceRequest) (*v1.Response[evidenceapi.RawEvidenceReadResult], error) {
	return &v1.Response[evidenceapi.RawEvidenceReadResult]{Status: http.StatusNoContent}, nil
}

func (testEvidenceService) PublishEvidence(context.Context, *evidenceapi.EvidencePublicationRequest) (*v1.Response[evidenceapi.EvidencePublicationResult], error) {
	return &v1.Response[evidenceapi.EvidencePublicationResult]{Status: http.StatusNoContent}, nil
}

func (testEvidenceService) ListEvidenceCategories(context.Context) (*v1.Response[evidenceapi.EvidenceCategoryCatalog], error) {
	return &v1.Response[evidenceapi.EvidenceCategoryCatalog]{Status: http.StatusNoContent}, nil
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
