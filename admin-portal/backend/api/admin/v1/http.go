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

const maxRequestBodyBytes = 128 * 1024

func RegisterAdminHTTPServer(server *kratoshttp.Server, service AdminHTTPServer) {
	router := server.Route(APIPrefix)
	router.GET("/raw-documents", listRawDocumentsHandler(service))
	router.GET("/events", listEventsHandler(service))
	router.GET("/agent-schedules/{agent_key}", getAgentScheduleHandler(service))
	router.PUT("/agent-schedules/{agent_key}", saveAgentScheduleHandler(service))
	router.PATCH("/agent-schedules/{agent_key}", setAgentScheduleEnabledHandler(service))
	router.GET("/agent-executions", listAgentExecutionsHandler(service))
	router.GET("/agent-statuses", listAgentStatusesHandler(service))
	router.GET("/monitoring/summary", getMonitoringSummaryHandler(service))
	router.GET("/monitoring/collector-executions", listCollectorMonitoringHandler(service))
	router.GET("/monitoring/artifact-extractions", listArtifactMonitoringHandler(service))
	router.GET("/monitoring/semantic-work-items", listSemanticMonitoringHandler(service))
	router.GET("/model-providers", listModelProvidersHandler(service))
	router.GET("/model-providers/{provider_key}", getModelProviderHandler(service))
	router.PATCH("/model-providers/{provider_key}", patchModelProviderHandler(service))
	router.GET("/connectors", listConnectorsHandler(service))
	router.GET("/connectors/{connector_key}", getConnectorHandler(service))
	router.PATCH("/connectors/{connector_key}", patchConnectorHandler(service))
}

func listRawDocumentsHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		page, pageSize, err := parsePage(ctx, 50)
		if err != nil {
			return err
		}
		request := &ListRawDocumentsRequest{
			Title: ctx.Query().Get("title"), SourceRef: ctx.Query().Get("source_ref"),
			Page: page, PageSize: pageSize,
		}
		return call(ctx, OperationListRawDocuments, request, func(callContext context.Context) (any, error) {
			return service.ListRawDocuments(callContext, request)
		})
	}
}

func listEventsHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		page, pageSize, err := parsePage(ctx, 50)
		if err != nil {
			return err
		}
		request := &ListEventsRequest{
			Title: ctx.Query().Get("title"), EventStatus: ctx.Query().Get("event_status"),
			FactStatus: ctx.Query().Get("fact_status"), EventTimeFrom: ctx.Query().Get("event_time_from"),
			EventTimeTo: ctx.Query().Get("event_time_to"), FirstSeenFrom: ctx.Query().Get("first_seen_from"),
			FirstSeenTo: ctx.Query().Get("first_seen_to"), Page: page, PageSize: pageSize,
		}
		return call(ctx, OperationListEvents, request, func(callContext context.Context) (any, error) {
			return service.ListEvents(callContext, request)
		})
	}
}

func getAgentScheduleHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &AgentKeyRequest{AgentKey: ctx.Vars().Get("agent_key")}
		return call(ctx, OperationGetAgentSchedule, request, func(callContext context.Context) (any, error) {
			return service.GetAgentSchedule(callContext, request)
		})
	}
}

func saveAgentScheduleHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &SaveAgentScheduleRequest{AgentKey: ctx.Vars().Get("agent_key")}
		if err := decodeStrictJSON(ctx.Request(), request); err != nil {
			return NewHTTPError(http.StatusBadRequest, "INVALID_REQUEST", "request body is invalid")
		}
		return call(ctx, OperationSaveAgentSchedule, request, func(callContext context.Context) (any, error) {
			return service.SaveAgentSchedule(callContext, request)
		})
	}
}

func setAgentScheduleEnabledHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &SetAgentScheduleEnabledRequest{AgentKey: ctx.Vars().Get("agent_key")}
		if err := decodeStrictJSON(ctx.Request(), request); err != nil || request.Enabled == nil {
			return NewHTTPError(http.StatusBadRequest, "INVALID_REQUEST", "enabled is required")
		}
		return call(ctx, OperationSetScheduleEnabled, request, func(callContext context.Context) (any, error) {
			return service.SetAgentScheduleEnabled(callContext, request)
		})
	}
}

func listAgentExecutionsHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		page, err := parsePositiveInt(ctx.Query().Get("page"), 1, "page must be positive")
		if err != nil {
			return err
		}
		request := &ListAgentExecutionsRequest{Page: page}
		return call(ctx, OperationListAgentExecutions, request, func(callContext context.Context) (any, error) {
			return service.ListAgentExecutions(callContext, request)
		})
	}
}

func listAgentStatusesHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &EmptyRequest{}
		return call(ctx, OperationListAgentStatuses, request, func(callContext context.Context) (any, error) {
			return service.ListAgentStatuses(callContext, request)
		})
	}
}

func getMonitoringSummaryHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		window, err := parseMonitoringWindow(ctx.Query().Get("window"))
		if err != nil {
			return err
		}
		request := &MonitoringSummaryRequest{Window: window}
		return call(ctx, OperationGetMonitoringSummary, request, func(callContext context.Context) (any, error) {
			return service.GetMonitoringSummary(callContext, request)
		})
	}
}

func listCollectorMonitoringHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return monitoringListHandler(OperationListCollectorMonitoring, func(ctx context.Context, request *MonitoringListRequest) (any, error) {
		return service.ListCollectorMonitoring(ctx, request)
	})
}
func listArtifactMonitoringHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return monitoringListHandler(OperationListArtifactMonitoring, func(ctx context.Context, request *MonitoringListRequest) (any, error) {
		return service.ListArtifactMonitoring(ctx, request)
	})
}
func listSemanticMonitoringHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return monitoringListHandler(OperationListSemanticMonitoring, func(ctx context.Context, request *MonitoringListRequest) (any, error) {
		return service.ListSemanticMonitoring(ctx, request)
	})
}

func monitoringListHandler(operation string, list func(context.Context, *MonitoringListRequest) (any, error)) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		window, err := parseMonitoringWindow(ctx.Query().Get("window"))
		if err != nil {
			return err
		}
		state := strings.TrimSpace(ctx.Query().Get("state"))
		if state == "" {
			state = "all"
		}
		if state != "all" && state != "success" && state != "running" && state != "failure" {
			return NewHTTPError(http.StatusBadRequest, "INVALID_MONITORING_FILTER", "monitoring filter is invalid")
		}
		page, pageSize, err := parsePage(ctx, 20)
		if err != nil || pageSize > 100 {
			return NewHTTPError(http.StatusBadRequest, "INVALID_PAGINATION", "monitoring pagination is invalid")
		}
		request := &MonitoringListRequest{Window: window, State: state, Page: page, PageSize: pageSize}
		return call(ctx, operation, request, func(callContext context.Context) (any, error) { return list(callContext, request) })
	}
}

func parseMonitoringWindow(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "1h", nil
	}
	if value == "1h" || value == "6h" || value == "12h" || value == "24h" {
		return value, nil
	}
	return "", NewHTTPError(http.StatusBadRequest, "INVALID_MONITORING_WINDOW", "monitoring window is invalid")
}

func listModelProvidersHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &EmptyRequest{}
		return call(ctx, OperationListModelProviders, request, func(callContext context.Context) (any, error) {
			return service.ListModelProviders(callContext, request)
		})
	}
}

func getModelProviderHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ProviderKeyRequest{ProviderKey: ctx.Vars().Get("provider_key")}
		return call(ctx, OperationGetModelProvider, request, func(callContext context.Context) (any, error) {
			return service.GetModelProvider(callContext, request)
		})
	}
}

func patchModelProviderHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &PatchModelProviderRequest{ProviderKey: ctx.Vars().Get("provider_key")}
		if err := decodeStrictJSON(ctx.Request(), request); err != nil ||
			(request.BaseURL == nil && request.Model == nil && request.APIKey == nil) {
			return NewHTTPError(http.StatusBadRequest, "INVALID_REQUEST", "request body is invalid")
		}
		return call(ctx, OperationPatchModelProvider, request, func(callContext context.Context) (any, error) {
			return service.PatchModelProvider(callContext, request)
		})
	}
}

func listConnectorsHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &EmptyRequest{}
		return call(ctx, OperationListConnectors, request, func(callContext context.Context) (any, error) {
			return service.ListConnectors(callContext, request)
		})
	}
}

func getConnectorHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ConnectorKeyRequest{ConnectorKey: ctx.Vars().Get("connector_key")}
		return call(ctx, OperationGetConnector, request, func(callContext context.Context) (any, error) {
			return service.GetConnector(callContext, request)
		})
	}
}

func patchConnectorHandler(service AdminHTTPServer) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &PatchConnectorRequest{ConnectorKey: ctx.Vars().Get("connector_key")}
		if err := decodeStrictJSON(ctx.Request(), request); err != nil ||
			(request.BaseURL == nil && request.APIKey == nil) {
			return NewHTTPError(http.StatusBadRequest, "INVALID_REQUEST", "request body is invalid")
		}
		return call(ctx, OperationPatchConnector, request, func(callContext context.Context) (any, error) {
			return service.PatchConnector(callContext, request)
		})
	}
}

func call(
	ctx kratoshttp.Context,
	operation string,
	request any,
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
	return ctx.Result(http.StatusOK, response)
}

func decodeStrictJSON(request *http.Request, target any) error {
	body, err := io.ReadAll(io.LimitReader(request.Body, maxRequestBodyBytes+1))
	if err != nil || len(body) == 0 || len(body) > maxRequestBodyBytes {
		return errors.New("invalid request body")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("trailing request content")
	}
	return nil
}

func parsePage(ctx kratoshttp.Context, defaultPageSize int) (int, int, error) {
	page, err := parsePositiveInt(ctx.Query().Get("page"), 1, "page must be positive")
	if err != nil {
		return 0, 0, err
	}
	pageSize, err := parsePositiveInt(ctx.Query().Get("page_size"), defaultPageSize, "page_size must be positive")
	if err != nil {
		return 0, 0, err
	}
	return page, pageSize, nil
}

func parsePositiveInt(value string, fallback int, message string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, NewHTTPError(http.StatusBadRequest, "INVALID_REQUEST", message)
	}
	return parsed, nil
}
