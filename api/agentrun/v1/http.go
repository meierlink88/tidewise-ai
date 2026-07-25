package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

func RegisterAgentRunHTTPServer(server *kratoshttp.Server, service AgentRunHTTPServer) {
	collector := server.Route(CollectorRunsPath)
	collector.POST("", createCollectorRunHandler(service))
	collector.GET("/{execution_id}", getCollectorRunHandler(service))

	admin := server.Route(AdminAPIPrefix)
	admin.GET("/model-providers", listModelProvidersHandler(service))
	admin.GET("/model-providers/{provider_key}", getModelProviderHandler(service))
	admin.PATCH("/model-providers/{provider_key}", patchModelProviderHandler(service))
	admin.GET("/connectors", listConnectorsHandler(service))
	admin.GET("/connectors/{connector_key}", getConnectorHandler(service))
	admin.PATCH("/connectors/{connector_key}", patchConnectorHandler(service))
	admin.GET("/agent-schedules", listAgentSchedulesHandler(service))
	admin.GET("/agent-schedules/{agent_key}", getAgentScheduleHandler(service))
	admin.PUT("/agent-schedules/{agent_key}", putAgentScheduleHandler(service))
	admin.PATCH("/agent-schedules/{agent_key}", patchAgentScheduleHandler(service))
	admin.GET("/agent-executions", listAgentExecutionsHandler(service))
}

func createCollectorRunHandler(service AgentRunHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &CreateCollectorRunRequest{
			IdempotencyKey: strings.TrimSpace(ctx.Request().Header.Get("Idempotency-Key")),
		}
		return call(ctx, OperationCreateCollectorRun, request, http.StatusAccepted, func(callContext context.Context) (any, error) {
			if request.IdempotencyKey == "" {
				return nil, InvalidRequest("IDEMPOTENCY_KEY_REQUIRED", "Idempotency-Key is required")
			}
			if err := decodeStrict(ctx.Request(), request, MaxRequestBody); err != nil {
				if errors.Is(err, errBodyTooLarge) {
					return nil, NewPublicError(http.StatusRequestEntityTooLarge, "PROMPT_TOO_LARGE", "Prompt exceeds 64 KiB", nil)
				}
				return nil, InvalidRequest("INVALID_REQUEST", "request body is invalid")
			}
			return service.CreateCollectorRun(callContext, request)
		})
	}
}

func getCollectorRunHandler(service AgentRunHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &GetCollectorRunRequest{ExecutionID: strings.TrimSpace(ctx.Vars().Get("execution_id"))}
		return call(ctx, OperationGetCollectorRun, request, http.StatusOK, func(callContext context.Context) (any, error) {
			return service.GetCollectorRun(callContext, request)
		})
	}
}

func listModelProvidersHandler(service AgentRunHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ListModelProvidersRequest{}
		return call(ctx, OperationListModelProviders, request, http.StatusOK, func(callContext context.Context) (any, error) {
			return service.ListModelProviders(callContext, request)
		})
	}
}

func getModelProviderHandler(service AgentRunHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &GetModelProviderRequest{ProviderKey: ctx.Vars().Get("provider_key")}
		return call(ctx, OperationGetModelProvider, request, http.StatusOK, func(callContext context.Context) (any, error) {
			return service.GetModelProvider(callContext, request)
		})
	}
}

func patchModelProviderHandler(service AgentRunHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &PatchModelProviderRequest{ProviderKey: ctx.Vars().Get("provider_key")}
		return call(ctx, OperationPatchModelProvider, request, http.StatusOK, func(callContext context.Context) (any, error) {
			if err := decodeStrict(ctx.Request(), request, MaxAdminRequestBody); err != nil {
				return nil, InvalidRequest("INVALID_REQUEST", "request body is invalid")
			}
			if !request.BaseURL.Set && !request.Model.Set && !request.APIKey.Set {
				return nil, InvalidRequest("INVALID_REQUEST", "request body is invalid")
			}
			return service.PatchModelProvider(callContext, request)
		})
	}
}

func listConnectorsHandler(service AgentRunHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ListConnectorsRequest{}
		return call(ctx, OperationListConnectors, request, http.StatusOK, func(callContext context.Context) (any, error) {
			return service.ListConnectors(callContext, request)
		})
	}
}

func getConnectorHandler(service AgentRunHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &GetConnectorRequest{ConnectorKey: ctx.Vars().Get("connector_key")}
		return call(ctx, OperationGetConnector, request, http.StatusOK, func(callContext context.Context) (any, error) {
			return service.GetConnector(callContext, request)
		})
	}
}

func patchConnectorHandler(service AgentRunHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &PatchConnectorRequest{ConnectorKey: ctx.Vars().Get("connector_key")}
		return call(ctx, OperationPatchConnector, request, http.StatusOK, func(callContext context.Context) (any, error) {
			if err := decodeStrict(ctx.Request(), request, MaxAdminRequestBody); err != nil {
				return nil, InvalidRequest("INVALID_REQUEST", "request body is invalid")
			}
			if !request.BaseURL.Set && !request.APIKey.Set {
				return nil, InvalidRequest("INVALID_REQUEST", "request body is invalid")
			}
			return service.PatchConnector(callContext, request)
		})
	}
}

func listAgentSchedulesHandler(service AgentRunHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ListAgentSchedulesRequest{}
		return call(ctx, OperationListAgentSchedules, request, http.StatusOK, func(callContext context.Context) (any, error) {
			return service.ListAgentSchedules(callContext, request)
		})
	}
}

func getAgentScheduleHandler(service AgentRunHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &GetAgentScheduleRequest{AgentKey: ctx.Vars().Get("agent_key")}
		return call(ctx, OperationGetAgentSchedule, request, http.StatusOK, func(callContext context.Context) (any, error) {
			return service.GetAgentSchedule(callContext, request)
		})
	}
}

func putAgentScheduleHandler(service AgentRunHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &PutAgentScheduleRequest{AgentKey: ctx.Vars().Get("agent_key")}
		return call(ctx, OperationPutAgentSchedule, request, http.StatusOK, func(callContext context.Context) (any, error) {
			if err := decodeStrict(ctx.Request(), request, MaxAdminRequestBody); err != nil || request.Enabled == nil {
				return nil, InvalidRequest("INVALID_REQUEST", "request body is invalid")
			}
			return service.PutAgentSchedule(callContext, request)
		})
	}
}

func patchAgentScheduleHandler(service AgentRunHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &PatchAgentScheduleRequest{AgentKey: ctx.Vars().Get("agent_key")}
		return call(ctx, OperationPatchAgentSchedule, request, http.StatusOK, func(callContext context.Context) (any, error) {
			if err := decodeStrict(ctx.Request(), request, MaxAdminRequestBody); err != nil {
				return nil, InvalidRequest("INVALID_REQUEST", "request body is invalid")
			}
			if request.AgentVersion == nil && request.ScheduleType == nil && request.CronExpression == nil &&
				request.DailyTimes == nil && request.Input == nil && request.Enabled == nil {
				return nil, InvalidRequest("INVALID_REQUEST", "request body is invalid")
			}
			return service.PatchAgentSchedule(callContext, request)
		})
	}
}

func listAgentExecutionsHandler(service AgentRunHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ListAgentExecutionsRequest{}
		return call(ctx, OperationListAgentExecutions, request, http.StatusOK, func(callContext context.Context) (any, error) {
			page, err := positiveInt(ctx.Query().Get("page"), 1)
			if err != nil {
				return nil, InvalidRequest("INVALID_PAGINATION", "Execution pagination is invalid")
			}
			pageSize, err := positiveInt(ctx.Query().Get("page_size"), 20)
			if err != nil || pageSize > 100 {
				return nil, InvalidRequest("INVALID_PAGINATION", "Execution pagination is invalid")
			}
			sortOrder := strings.TrimSpace(ctx.Query().Get("sort_order"))
			if sortOrder == "" {
				sortOrder = "desc"
			}
			if sortOrder != "asc" && sortOrder != "desc" {
				return nil, InvalidRequest("INVALID_SORT_ORDER", "Execution sort order is invalid")
			}
			request.AgentKey = strings.TrimSpace(ctx.Query().Get("agent_key"))
			request.Page = page
			request.PageSize = pageSize
			request.SortOrder = sortOrder
			return service.ListAgentExecutions(callContext, request)
		})
	}
}

func call(
	ctx kratoshttp.Context,
	operation string,
	request any,
	status int,
	invoke func(context.Context) (any, error),
) error {
	kratoshttp.SetOperation(ctx, operation)
	handler := ctx.Middleware(func(callContext context.Context, _ any) (any, error) {
		return invoke(callContext)
	})
	response, err := handler(ctx, request)
	if err != nil {
		return err
	}
	return ctx.Result(status, response)
}

var errBodyTooLarge = errors.New("request body is too large")

func decodeStrict(request *http.Request, target any, limit int64) error {
	body, err := io.ReadAll(io.LimitReader(request.Body, limit+1))
	if err != nil || len(body) == 0 {
		return ErrInvalidRequest
	}
	if int64(len(body)) > limit {
		return errBodyTooLarge
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return ErrInvalidRequest
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return ErrInvalidRequest
	}
	return nil
}

func positiveInt(value string, fallback int) (int, error) {
	if strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, ErrInvalidRequest
	}
	return parsed, nil
}
