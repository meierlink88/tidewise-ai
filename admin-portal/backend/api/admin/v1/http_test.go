package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

func TestEveryAdminEndpointExecutesKratosMiddleware(t *testing.T) {
	seen := map[string]int{}
	server := kratoshttp.NewServer(kratoshttp.Middleware(func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, request any) (any, error) {
			serverTransport, ok := transport.FromServerContext(ctx)
			if !ok {
				t.Fatal("middleware context is missing Kratos server transport")
			}
			seen[serverTransport.Operation()]++
			return handler(ctx, request)
		}
	}))
	RegisterAdminHTTPServer(server, stubAdminHTTPServer{})

	for _, request := range []struct {
		method    string
		path      string
		body      string
		operation string
	}{
		{http.MethodGet, APIPrefix + "/raw-documents", "", OperationListRawDocuments},
		{http.MethodGet, APIPrefix + "/events", "", OperationListEvents},
		{http.MethodGet, APIPrefix + "/agent-schedules/collector", "", OperationGetAgentSchedule},
		{http.MethodPut, APIPrefix + "/agent-schedules/collector", `{"agent_version":"collector.v1","schedule_type":"daily","daily_times":["08:00"],"input":{}}`, OperationSaveAgentSchedule},
		{http.MethodPatch, APIPrefix + "/agent-schedules/collector", `{"enabled":true}`, OperationSetScheduleEnabled},
		{http.MethodGet, APIPrefix + "/agent-statuses", "", OperationListAgentStatuses},
		{http.MethodGet, APIPrefix + "/monitoring/summary", "", OperationGetMonitoringSummary},
		{http.MethodGet, APIPrefix + "/monitoring/collector-executions", "", OperationListCollectorMonitoring},
		{http.MethodGet, APIPrefix + "/monitoring/artifact-extractions", "", OperationListArtifactMonitoring},
		{http.MethodGet, APIPrefix + "/monitoring/semantic-work-items", "", OperationListSemanticMonitoring},
		{http.MethodGet, APIPrefix + "/model-providers", "", OperationListModelProviders},
		{http.MethodGet, APIPrefix + "/model-providers/deepseek", "", OperationGetModelProvider},
		{http.MethodPatch, APIPrefix + "/model-providers/deepseek", `{"model":"deepseek-chat"}`, OperationPatchModelProvider},
		{http.MethodGet, APIPrefix + "/connectors", "", OperationListConnectors},
		{http.MethodGet, APIPrefix + "/connectors/tavily", "", OperationGetConnector},
		{http.MethodPatch, APIPrefix + "/connectors/tavily", `{"base_url":"https://api.tavily.com"}`, OperationPatchConnector},
		{http.MethodGet, APIPrefix + "/runtime-health", "", OperationGetRuntimeHealth},
	} {
		response := httptest.NewRecorder()
		httpRequest := httptest.NewRequest(request.method, request.path, strings.NewReader(request.body))
		if request.body != "" {
			httpRequest.Header.Set("Content-Type", "application/json")
		}
		server.ServeHTTP(response, httpRequest)
		if response.Code != http.StatusOK {
			t.Fatalf("%s %s status = %d, want %d; body=%s", request.method, request.path, response.Code, http.StatusOK, response.Body.String())
		}
		if seen[request.operation] != 1 {
			t.Fatalf("%s %s middleware calls for %q = %d, want 1", request.method, request.path, request.operation, seen[request.operation])
		}
	}
}

type monitoringAdminHTTPStub struct {
	AdminHTTPServer
	request *MonitoringListRequest
}

func (stub *monitoringAdminHTTPStub) ListCollectorMonitoring(_ context.Context, request *MonitoringListRequest) (*CollectorMonitoringPage, error) {
	stub.request = request
	return &CollectorMonitoringPage{}, nil
}

func TestMonitoringEndpointForwardsOnlyFrozenQueryParameters(t *testing.T) {
	stub := &monitoringAdminHTTPStub{}
	server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(func(response http.ResponseWriter, _ *http.Request, err error) {
		if public, ok := PublicError(err); ok {
			response.WriteHeader(public.Status())
			return
		}
		response.WriteHeader(http.StatusInternalServerError)
	}))
	RegisterAdminHTTPServer(server, stub)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet,
		APIPrefix+"/monitoring/collector-executions?window=6h&state=running&page=2&page_size=25",
		nil,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	if stub.request == nil || stub.request.Window != "6h" || stub.request.State != "running" ||
		stub.request.Page != 2 || stub.request.PageSize != 25 {
		t.Fatalf("request = %+v", stub.request)
	}

	for _, query := range []string{"window=2h", "window=1h&state=cancelled", "window=1h&page=0"} {
		response = httptest.NewRecorder()
		server.ServeHTTP(response, httptest.NewRequest(
			http.MethodGet,
			APIPrefix+"/monitoring/collector-executions?"+query,
			nil,
		))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("query %q status = %d, want 400", query, response.Code)
		}
	}
}

type stubAdminHTTPServer struct{}

func (stubAdminHTTPServer) ListRawDocuments(context.Context, *ListRawDocumentsRequest) (*RawDocumentListResponse, error) {
	return &RawDocumentListResponse{}, nil
}
func (stubAdminHTTPServer) ListEvents(context.Context, *ListEventsRequest) (*EventListResponse, error) {
	return &EventListResponse{}, nil
}
func (stubAdminHTTPServer) GetAgentSchedule(context.Context, *AgentKeyRequest) (*AgentSchedule, error) {
	return &AgentSchedule{}, nil
}
func (stubAdminHTTPServer) SaveAgentSchedule(context.Context, *SaveAgentScheduleRequest) (*AgentSchedule, error) {
	return &AgentSchedule{}, nil
}
func (stubAdminHTTPServer) SetAgentScheduleEnabled(context.Context, *SetAgentScheduleEnabledRequest) (*AgentSchedule, error) {
	return &AgentSchedule{}, nil
}
func (stubAdminHTTPServer) ListAgentStatuses(context.Context, *EmptyRequest) (*AgentStatusListResponse, error) {
	return &AgentStatusListResponse{}, nil
}
func (stubAdminHTTPServer) GetMonitoringSummary(context.Context, *MonitoringSummaryRequest) (*MonitoringSummary, error) {
	return &MonitoringSummary{}, nil
}
func (stubAdminHTTPServer) ListCollectorMonitoring(context.Context, *MonitoringListRequest) (*CollectorMonitoringPage, error) {
	return &CollectorMonitoringPage{}, nil
}
func (stubAdminHTTPServer) ListArtifactMonitoring(context.Context, *MonitoringListRequest) (*ArtifactMonitoringPage, error) {
	return &ArtifactMonitoringPage{}, nil
}
func (stubAdminHTTPServer) ListSemanticMonitoring(context.Context, *MonitoringListRequest) (*SemanticMonitoringPage, error) {
	return &SemanticMonitoringPage{}, nil
}
func (stubAdminHTTPServer) ListModelProviders(context.Context, *EmptyRequest) (*ModelProviderListResponse, error) {
	return &ModelProviderListResponse{}, nil
}
func (stubAdminHTTPServer) GetModelProvider(context.Context, *ProviderKeyRequest) (*ModelProviderConfiguration, error) {
	return &ModelProviderConfiguration{}, nil
}
func (stubAdminHTTPServer) PatchModelProvider(context.Context, *PatchModelProviderRequest) (*ModelProviderConfiguration, error) {
	return &ModelProviderConfiguration{}, nil
}
func (stubAdminHTTPServer) ListConnectors(context.Context, *EmptyRequest) (*ConnectorListResponse, error) {
	return &ConnectorListResponse{}, nil
}
func (stubAdminHTTPServer) GetConnector(context.Context, *ConnectorKeyRequest) (*ConnectorConfiguration, error) {
	return &ConnectorConfiguration{}, nil
}
func (stubAdminHTTPServer) PatchConnector(context.Context, *PatchConnectorRequest) (*ConnectorConfiguration, error) {
	return &ConnectorConfiguration{}, nil
}
func (stubAdminHTTPServer) GetRuntimeHealth(context.Context, *EmptyRequest) (*RuntimeHealth, error) {
	return &RuntimeHealth{}, nil
}
