package evidence

import (
	"context"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const ExecutionBudget = 3 * time.Second

func RegisterHTTPServer(server *kratoshttp.Server, application Service) {
	registerHTTPServer(server, application, ExecutionBudget)
}

func registerHTTPServer(server *kratoshttp.Server, application Service, executionBudget time.Duration) {
	router := server.Route(v1.APIPrefix)
	router.POST("/raw-evidence-publications", rawEvidenceHandler(application, executionBudget))
	router.GET("/raw-evidences/{id}", getRawEvidenceHandler(application, executionBudget))
	router.POST("/evidence-publications", evidenceHandler(application, executionBudget))
	router.GET("/evidence-categories", evidenceCategoryCatalogHandler(application, executionBudget))
	router.GET("/evidences", listEvidenceHandler(application, executionBudget))
}

func listEvidenceHandler(application Service, executionBudget time.Duration) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		for name := range ctx.Request().URL.Query() {
			if !allowedEvidenceListParameter(name) {
				return v1.NewPublicError(v1.StatusBadRequest, ErrorInvalidRequest, "unsupported Evidence query parameter", nil)
			}
		}
		request := &ListRequest{
			Title: ctx.Query().Get("title"), Summary: ctx.Query().Get("summary"),
			CategoryID: ctx.Query().Get("category_id"), SourceID: ctx.Query().Get("source_id"), SourceName: ctx.Query().Get("source_name"),
			SourceLevel: ctx.Query().Get("source_level"), IsSplit: ctx.Query().Get("is_split"),
			PublishedFrom: ctx.Query().Get("published_from"), PublishedTo: ctx.Query().Get("published_to"),
			CollectedFrom: ctx.Query().Get("collected_from"), CollectedTo: ctx.Query().Get("collected_to"),
			Page: ctx.Query().Get("page"), PageSize: ctx.Query().Get("page_size"),
		}
		return v1.Call(ctx, OperationListAdminEvidence, request, func(callContext context.Context) (*v1.Response[Page], error) {
			deadlineContext, cancel := context.WithTimeout(callContext, executionBudget)
			defer cancel()
			return application.ListEvidence(deadlineContext, request)
		})
	}
}

func allowedEvidenceListParameter(name string) bool {
	switch name {
	case "title", "summary", "category_id", "source_id", "source_name", "source_level", "is_split",
		"published_from", "published_to", "collected_from", "collected_to", "page", "page_size":
		return true
	default:
		return false
	}
}

func evidenceCategoryCatalogHandler(application Service, executionBudget time.Duration) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		if ctx.Request().URL.RawQuery != "" {
			return v1.NewPublicError(v1.StatusBadRequest, ErrorInvalidRequest, "Evidence Category Catalog does not accept query parameters", nil)
		}
		return v1.Call(ctx, OperationListEvidenceCategories, nil, func(callContext context.Context) (*v1.Response[EvidenceCategoryCatalog], error) {
			deadlineContext, cancel := context.WithTimeout(callContext, executionBudget)
			defer cancel()
			return application.ListEvidenceCategories(deadlineContext)
		})
	}
}

func getRawEvidenceHandler(application Service, executionBudget time.Duration) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &GetRawEvidenceRequest{ID: ctx.Vars().Get("id")}
		return v1.Call(ctx, OperationGetRawEvidence, request, func(callContext context.Context) (*v1.Response[RawEvidenceReadResult], error) {
			deadlineContext, cancel := context.WithTimeout(callContext, executionBudget)
			defer cancel()
			return application.GetRawEvidence(deadlineContext, request)
		})
	}
}

func rawEvidenceHandler(application Service, executionBudget time.Duration) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		payload, err := v1.ReadImportPayload(ctx)
		if err != nil {
			return err
		}
		request, err := decodeRawEvidence(payload)
		if err != nil {
			return err
		}
		return v1.Call(ctx, OperationPublishRawEvidence, request, func(callContext context.Context) (*v1.Response[RawEvidencePublicationResult], error) {
			deadlineContext, cancel := context.WithTimeout(callContext, executionBudget)
			defer cancel()
			return application.PublishRawEvidence(deadlineContext, request)
		})
	}
}

func evidenceHandler(application Service, executionBudget time.Duration) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		payload, err := v1.ReadImportPayload(ctx)
		if err != nil {
			return err
		}
		request, err := decodeEvidence(payload)
		if err != nil {
			return err
		}
		return v1.Call(ctx, OperationPublishEvidence, request, func(callContext context.Context) (*v1.Response[EvidencePublicationResult], error) {
			deadlineContext, cancel := context.WithTimeout(callContext, executionBudget)
			defer cancel()
			return application.PublishEvidence(deadlineContext, request)
		})
	}
}

func decodeRawEvidence(payload []byte) (*RawEvidencePublicationRequest, error) {
	request := new(RawEvidencePublicationRequest)
	if err := v1.DecodeStrictJSON(payload, rawEvidenceShape(), request); err != nil {
		return nil, v1.NewPublicError(v1.StatusBadRequest, ErrorInvalidRequest, "request body is not valid for the Raw Evidence Publication contract", nil)
	}
	return request, nil
}

func decodeEvidence(payload []byte) (*EvidencePublicationRequest, error) {
	request := new(EvidencePublicationRequest)
	if err := v1.DecodeStrictJSON(payload, evidenceShape(), request); err != nil {
		return nil, v1.NewPublicError(v1.StatusBadRequest, ErrorInvalidRequest, "request body is not valid for the Evidence Publication contract", nil)
	}
	return request, nil
}

func rawEvidenceShape() *v1.StrictJSONShape {
	stringShape := v1.StrictJSONString()
	nullableStringShape := v1.StrictJSONNullableString()
	raw := v1.StrictJSONRequiredObject([]string{
		"publication_key", "source_id", "source_name", "source_level", "source_url",
		"is_original", "raw_text", "collected_at",
	}, map[string]*v1.StrictJSONShape{
		"publication_key": stringShape, "source_id": stringShape, "source_name": stringShape,
		"source_level": stringShape, "source_url": stringShape, "is_original": v1.StrictJSONBoolean(),
		"quoted_source_id": nullableStringShape, "quoted_source_name": nullableStringShape,
		"title": nullableStringShape, "raw_text": stringShape, "published_at": nullableStringShape,
		"collected_at": stringShape,
		"category_ids": v1.StrictJSONArray(stringShape),
	})
	return v1.StrictJSONRequiredObject([]string{"raw_evidence"}, map[string]*v1.StrictJSONShape{"raw_evidence": raw})
}

func evidenceShape() *v1.StrictJSONShape {
	stringShape := v1.StrictJSONString()
	nullableStringShape := v1.StrictJSONNullableString()
	timeShape := v1.StrictJSONRequiredObject([]string{"raw", "start_at", "end_at", "precision"}, map[string]*v1.StrictJSONShape{
		"raw": nullableStringShape, "start_at": nullableStringShape, "end_at": nullableStringShape,
		"precision": stringShape,
	})
	metricShape := v1.StrictJSONRequiredObject([]string{"name", "value", "unit", "change", "period"}, map[string]*v1.StrictJSONShape{
		"name": stringShape, "value": nullableStringShape, "unit": nullableStringShape,
		"change": nullableStringShape, "period": nullableStringShape,
	})
	attributionShape := v1.StrictJSONRequiredObject([]string{"reported_by", "claimed_by"}, map[string]*v1.StrictJSONShape{
		"reported_by": nullableStringShape, "claimed_by": nullableStringShape,
	})
	semantic := v1.StrictJSONRequiredObject([]string{
		"actors", "action", "objects", "stage", "modality", "time", "jurisdictions", "reason", "method", "metrics", "attribution",
	}, map[string]*v1.StrictJSONShape{
		"actors": v1.StrictJSONArray(stringShape), "action": stringShape, "objects": v1.StrictJSONArray(stringShape),
		"stage": stringShape, "modality": stringShape, "time": timeShape,
		"jurisdictions": v1.StrictJSONArray(stringShape), "reason": nullableStringShape, "method": nullableStringShape,
		"metrics": v1.StrictJSONArray(metricShape), "attribution": attributionShape,
	})
	item := v1.StrictJSONRequiredObject([]string{"summary", "keywords", "semantic"}, map[string]*v1.StrictJSONShape{
		"summary": stringShape, "keywords": v1.StrictJSONArray(stringShape), "semantic": semantic,
	})
	return v1.StrictJSONRequiredObject([]string{"raw_evidence_id", "evidences"}, map[string]*v1.StrictJSONShape{
		"raw_evidence_id": stringShape, "evidences": v1.StrictJSONArray(item),
	})
}
