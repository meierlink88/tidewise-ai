package eventsemantic

import (
	"context"
	"strconv"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

func RegisterHTTPServer(server *kratoshttp.Server, application Service) {
	router := server.Route(v1.APIPrefix)
	router.GET("/event-semantics/eligible-events", listEligibleEventSemanticEventsHandler(application))
	router.POST("/event-semantics/context-leases", createEventSemanticContextLeaseHandler(application))
	router.GET("/event-semantics/context-leases/{context_lease_id}/context", getEventSemanticContextHandler(application))
	router.POST("/event-semantics/submissions", createEventSemanticSubmissionHandler(application))
	router.POST("/event-semantics/submissions/{submission_id}/reviews", submitEventSemanticReviewHandler(application))
	router.GET("/events/{event_id}/semantics", getEventSemanticsHandler(application))
}

func listEligibleEventSemanticEventsHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		limit := 20
		var limitErr error
		if raw := ctx.Query().Get("limit"); raw != "" {
			limit, limitErr = strconv.Atoi(raw)
		}
		pagination := ctx.Query().Get("pagination")
		request := &EligibleEventSemanticEventsRequest{
			Limit:      limit,
			Cursor:     ctx.Query().Get("cursor"),
			Pagination: pagination,
		}
		return v1.Call(ctx, OperationListEligibleEvents, request, func(callContext context.Context) (*v1.Response[EligibleEventSemanticEvents], error) {
			if limitErr != nil {
				return nil, v1.NewPublicError(
					v1.StatusBadRequest, ErrorInvalidRequest, "limit is invalid", nil,
				)
			}
			if pagination != "" && pagination != "cursor" {
				return nil, v1.NewPublicError(
					v1.StatusBadRequest, ErrorInvalidRequest, "pagination is invalid", nil,
				)
			}
			return application.ListEligibleEventSemanticEvents(callContext, request)
		})
	}
}

func createEventSemanticContextLeaseHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request, err := v1.DecodeStrictJSONBody[EventSemanticContextLeaseRequest](ctx)
		if err != nil {
			return err
		}
		return v1.Call(ctx, OperationCreateContextLease, request, func(callContext context.Context) (*v1.Response[EventSemanticContextLease], error) {
			return application.CreateEventSemanticContextLease(callContext, request)
		})
	}
}

func getEventSemanticContextHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &EventSemanticContextRequest{ContextLeaseID: ctx.Vars().Get("context_lease_id")}
		return v1.Call(ctx, OperationGetContext, request, func(callContext context.Context) (*v1.Response[EventSemanticContext], error) {
			return application.GetEventSemanticContext(callContext, request)
		})
	}
}

func createEventSemanticSubmissionHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request, err := v1.DecodeStrictJSONBody[EventSemanticSubmissionRequest](ctx)
		if err != nil {
			return err
		}
		return v1.Call(ctx, OperationCreateSubmission, request, func(callContext context.Context) (*v1.Response[EventSemanticSubmissionResult], error) {
			return application.CreateEventSemanticSubmission(callContext, request)
		})
	}
}

func submitEventSemanticReviewHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request, err := v1.DecodeStrictJSONBody[EventSemanticReviewRequest](ctx)
		if err != nil {
			return err
		}
		request.SubmissionID = ctx.Vars().Get("submission_id")
		return v1.Call(ctx, OperationSubmitReview, request, func(callContext context.Context) (*v1.Response[EventSemanticSubmissionResult], error) {
			return application.SubmitEventSemanticReview(callContext, request)
		})
	}
}

func getEventSemanticsHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &GetEventSemanticsRequest{EventID: ctx.Vars().Get("event_id")}
		return v1.Call(ctx, OperationGetSemantics, request, func(callContext context.Context) (*v1.Response[EventSemanticsResult], error) {
			return application.GetEventSemantics(callContext, request)
		})
	}
}
