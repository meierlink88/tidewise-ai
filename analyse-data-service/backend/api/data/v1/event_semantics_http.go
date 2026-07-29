package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strconv"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

func listEligibleEventSemanticEventsHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		limit := 20
		if raw := ctx.Query().Get("limit"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				return NewPublicError(StatusBadRequest, "INVALID_REQUEST", "limit is invalid", nil)
			}
			limit = parsed
		}
		request := &EligibleEventSemanticEventsRequest{Limit: limit}
		return call(ctx, OperationListEligibleEventSemanticEvents, request, func(callContext context.Context) (*Response[EligibleEventSemanticEvents], error) {
			return application.ListEligibleEventSemanticEvents(callContext, request)
		})
	}
}

func createEventSemanticContextLeaseHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request, err := decodeEventSemanticJSON[EventSemanticContextLeaseRequest](ctx)
		if err != nil {
			return err
		}
		return call(ctx, OperationCreateEventSemanticContextLease, request, func(callContext context.Context) (*Response[EventSemanticContextLease], error) {
			return application.CreateEventSemanticContextLease(callContext, request)
		})
	}
}

func getEventSemanticContextHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &EventSemanticContextRequest{ContextLeaseID: ctx.Vars().Get("context_lease_id")}
		return call(ctx, OperationGetEventSemanticContext, request, func(callContext context.Context) (*Response[EventSemanticContext], error) {
			return application.GetEventSemanticContext(callContext, request)
		})
	}
}

func resolveEventSemanticEntitiesHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request, err := decodeEventSemanticJSON[EventSemanticEntityResolutionRequest](ctx)
		if err != nil {
			return err
		}
		return call(ctx, OperationResolveEventSemanticEntities, request, func(callContext context.Context) (*Response[EventSemanticEntityResolutionResult], error) {
			return application.ResolveEventSemanticEntities(callContext, request)
		})
	}
}

func searchEventSemanticDirectTargetsHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request, err := decodeEventSemanticJSON[EventSemanticDirectTargetSearchRequest](ctx)
		if err != nil {
			return err
		}
		return call(ctx, OperationSearchEventSemanticDirectTargets, request, func(callContext context.Context) (*Response[EventSemanticDirectTargetSearchResult], error) {
			return application.SearchEventSemanticDirectTargets(callContext, request)
		})
	}
}

func createEventSemanticSubmissionHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request, err := decodeEventSemanticJSON[EventSemanticSubmissionRequest](ctx)
		if err != nil {
			return err
		}
		return call(ctx, OperationCreateEventSemanticSubmission, request, func(callContext context.Context) (*Response[EventSemanticSubmissionResult], error) {
			return application.CreateEventSemanticSubmission(callContext, request)
		})
	}
}

func submitEventSemanticReviewHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request, err := decodeEventSemanticJSON[EventSemanticReviewRequest](ctx)
		if err != nil {
			return err
		}
		request.SubmissionID = ctx.Vars().Get("submission_id")
		return call(ctx, OperationSubmitEventSemanticReview, request, func(callContext context.Context) (*Response[EventSemanticSubmissionResult], error) {
			return application.SubmitEventSemanticReview(callContext, request)
		})
	}
}

func getEventSemanticsHandler(application DataHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &GetEventSemanticsRequest{EventID: ctx.Vars().Get("event_id")}
		return call(ctx, OperationGetEventSemantics, request, func(callContext context.Context) (*Response[EventSemanticsResult], error) {
			return application.GetEventSemantics(callContext, request)
		})
	}
}

func decodeEventSemanticJSON[T any](ctx kratoshttp.Context) (*T, error) {
	payload, err := io.ReadAll(io.LimitReader(ctx.Request().Body, MaxRequestBodySize+1))
	if err != nil {
		return nil, NewPublicError(StatusBadRequest, "INVALID_REQUEST", "request body is not valid for this contract", nil)
	}
	if len(payload) > MaxRequestBodySize {
		return nil, NewPublicError(StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body exceeds 1048576 bytes", nil)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var request T
	if err := decoder.Decode(&request); err != nil {
		return nil, NewPublicError(StatusBadRequest, "INVALID_REQUEST", "request body is not valid for this contract", nil)
	}
	if err := ensureEventSemanticJSONEOF(decoder); err != nil {
		return nil, NewPublicError(StatusBadRequest, "INVALID_REQUEST", "request body is not valid for this contract", nil)
	}
	return &request, nil
}

func ensureEventSemanticJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return io.ErrUnexpectedEOF
		}
		return err
	}
	return nil
}
