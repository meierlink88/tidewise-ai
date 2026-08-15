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
	router.GET("/raw-evidences/{raw_evidence_id}", getRawEvidenceHandler(application, executionBudget))
	router.POST("/evidence-publications", evidenceHandler(application, executionBudget))
}

func getRawEvidenceHandler(application Service, executionBudget time.Duration) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &GetRawEvidenceRequest{RawEvidenceID: ctx.Vars().Get("raw_evidence_id")}
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
		"is_original", "raw_text", "collected_at", "keywords",
	}, map[string]*v1.StrictJSONShape{
		"publication_key": stringShape, "source_id": stringShape, "source_name": stringShape,
		"source_level": stringShape, "source_url": stringShape, "is_original": v1.StrictJSONBoolean(),
		"quoted_source_id": nullableStringShape, "quoted_source_name": nullableStringShape,
		"title": nullableStringShape, "raw_text": stringShape, "published_at": nullableStringShape,
		"collected_at": stringShape, "keywords": v1.StrictJSONArray(stringShape),
		"category_ids": v1.StrictJSONArray(stringShape),
	})
	return v1.StrictJSONRequiredObject([]string{"raw_evidence"}, map[string]*v1.StrictJSONShape{"raw_evidence": raw})
}

func evidenceShape() *v1.StrictJSONShape {
	stringShape := v1.StrictJSONString()
	nullableStringShape := v1.StrictJSONNullableString()
	item := v1.StrictJSONRequiredObject([]string{
		"split_order", "layer_type", "source_what",
		"expression_fingerprint", "expression_key", "fingerprint_version",
	}, map[string]*v1.StrictJSONShape{
		"split_order": v1.StrictJSONInteger(), "layer_type": stringShape,
		"source_who": nullableStringShape, "source_what": stringShape, "source_when": nullableStringShape,
		"source_when_raw": nullableStringShape, "source_where": nullableStringShape,
		"source_why": nullableStringShape, "source_how": nullableStringShape,
		"source_who_core": nullableStringShape, "source_what_core": nullableStringShape,
		"source_when_core": nullableStringShape, "source_when_raw_core": nullableStringShape,
		"source_where_core": nullableStringShape, "source_why_core": nullableStringShape,
		"source_how_core": nullableStringShape, "expression_fingerprint": stringShape,
		"expression_key": stringShape, "fingerprint_version": stringShape,
	})
	return v1.StrictJSONRequiredObject([]string{"raw_evidence_id", "evidences"}, map[string]*v1.StrictJSONShape{
		"raw_evidence_id": stringShape, "evidences": v1.StrictJSONArray(item),
	})
}
