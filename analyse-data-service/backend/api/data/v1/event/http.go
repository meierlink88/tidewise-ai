package event

import (
	"context"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
)

func RegisterHTTPServer(server *kratoshttp.Server, application Service) {
	router := server.Route(v1.APIPrefix)
	router.POST("/reviewed-event-imports", publicationHandler(application))
	router.GET("/event-tags", tagCatalogHandler(application))
	router.GET("/events", listHandler(application))
}

func publicationHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		payload, err := v1.ReadImportPayload(ctx)
		if err != nil {
			return err
		}
		request := new(PublicationRequest)
		if err := v1.DecodeStrictJSON(payload, publicationShape(), request); err != nil {
			return v1.NewPublicError(v1.StatusBadRequest, ErrorInvalidRequest, "request body is not valid for this contract", nil)
		}
		return v1.Call(ctx, OperationPublishReviewedEvents, request, func(callContext context.Context) (*v1.Response[PublicationResult], error) {
			return application.PublishReviewedEvents(callContext, request)
		})
	}
}

func tagCatalogHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		query := ctx.Request().URL.Query()
		values, exists := query["active"]
		if !exists || len(query) != 1 || len(values) != 1 || values[0] != "true" {
			return v1.NewPublicError(v1.StatusBadRequest, ErrorInvalidRequest, "active must be exactly true", nil)
		}
		request := &TagCatalogRequest{Active: true}
		return v1.Call(ctx, OperationListActiveEventTags, request, func(callContext context.Context) (*v1.Response[TagCatalog], error) {
			return application.ListActiveEventTags(callContext, request)
		})
	}
}

func listHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		query := ctx.Query()
		request := &ListRequest{
			Title: query.Get("title"), EventStatus: query.Get("event_status"), FactStatus: query.Get("fact_status"),
			EventTimeFrom: query.Get("event_time_from"), EventTimeTo: query.Get("event_time_to"),
			FirstSeenFrom: query.Get("first_seen_from"), FirstSeenTo: query.Get("first_seen_to"),
			Page: query.Get("page"), PageSize: query.Get("page_size"),
		}
		return v1.Call(ctx, OperationListAdminEvents, request, func(callContext context.Context) (*v1.Response[Page], error) {
			return application.ListEvents(callContext, request)
		})
	}
}

func publicationShape() *v1.StrictJSONShape {
	scalarShape := v1.StrictJSONScalar()
	anyShape := v1.StrictJSONAny()
	collector := v1.StrictJSONRequiredObject(nil, map[string]*v1.StrictJSONShape{
		"artifact_id": scalarShape, "collector_execution_id": scalarShape,
	})
	provenance := v1.StrictJSONRequiredObject(nil, map[string]*v1.StrictJSONShape{
		"extractor_execution_id": scalarShape, "extractor_agent_version": scalarShape,
		"collector_executions": v1.StrictJSONArray(collector),
	})
	rawDocument := v1.StrictJSONRequiredObject(nil, map[string]*v1.StrictJSONShape{
		"artifact_id": scalarShape, "content_sha256": scalarShape, "source_ref": scalarShape,
		"source_name": scalarShape, "source_type": scalarShape, "source_url": scalarShape,
		"title": scalarShape, "published_at": scalarShape, "collected_at": scalarShape,
		"language": scalarShape, "mime_type": scalarShape,
	})
	evidence := v1.StrictJSONRequiredObject(nil, map[string]*v1.StrictJSONShape{
		"artifact_id": scalarShape, "evidence_relation": scalarShape, "evidence_statement": scalarShape,
		"supports_fields": v1.StrictJSONArray(scalarShape), "source_level": scalarShape,
	})
	tag := v1.StrictJSONRequiredObject(nil, map[string]*v1.StrictJSONShape{
		"tag_id": scalarShape, "tag_kind": scalarShape, "tag_code": scalarShape, "confidence": scalarShape,
		"assignment_reason": scalarShape, "assign_source": scalarShape,
	})
	review := v1.StrictJSONRequiredObject(nil, map[string]*v1.StrictJSONShape{
		"review_id": scalarShape, "evidence_grade": scalarShape, "reasons": v1.StrictJSONArray(scalarShape),
	})
	event := v1.StrictJSONRequiredObject(nil, map[string]*v1.StrictJSONShape{
		"dedupe_key": scalarShape, "title": scalarShape, "factual_summary": scalarShape,
		"occurred_at": scalarShape, "fact_payload": anyShape, "evidence": v1.StrictJSONArray(evidence),
		"tags": v1.StrictJSONArray(tag), "review": review,
	})
	return v1.StrictJSONRequiredObject(nil, map[string]*v1.StrictJSONShape{
		"package_id": scalarShape, "provenance": provenance,
		"raw_documents": v1.StrictJSONArray(rawDocument), "events": v1.StrictJSONArray(event),
	})
}
