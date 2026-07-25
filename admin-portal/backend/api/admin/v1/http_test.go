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
		{http.MethodGet, APIPrefix + "/agent-executions", "", OperationListAgentExecutions},
		{http.MethodGet, APIPrefix + "/model-providers", "", OperationListModelProviders},
		{http.MethodGet, APIPrefix + "/model-providers/deepseek", "", OperationGetModelProvider},
		{http.MethodPatch, APIPrefix + "/model-providers/deepseek", `{"model":"deepseek-chat"}`, OperationPatchModelProvider},
		{http.MethodGet, APIPrefix + "/connectors", "", OperationListConnectors},
		{http.MethodGet, APIPrefix + "/connectors/tavily", "", OperationGetConnector},
		{http.MethodPatch, APIPrefix + "/connectors/tavily", `{"base_url":"https://api.tavily.com"}`, OperationPatchConnector},
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
func (stubAdminHTTPServer) ListAgentExecutions(context.Context, *ListAgentExecutionsRequest) (*AgentExecutionPage, error) {
	return &AgentExecutionPage{}, nil
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
