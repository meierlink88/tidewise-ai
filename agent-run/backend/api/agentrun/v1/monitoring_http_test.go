package v1

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

type monitoringHTTPStub struct {
	AgentRunHTTPServer
	requests chan *MonitoringListRequest
}

func (stub *monitoringHTTPStub) GetMonitoringSummary(_ context.Context, request *MonitoringSummaryRequest) (*MonitoringSummary, error) {
	return &MonitoringSummary{Window: request.Window}, nil
}
func (stub *monitoringHTTPStub) ListCollectorMonitoring(_ context.Context, request *MonitoringListRequest) (*CollectorMonitoringPage, error) {
	stub.requests <- request
	return &CollectorMonitoringPage{}, nil
}
func (stub *monitoringHTTPStub) ListArtifactMonitoring(_ context.Context, request *MonitoringListRequest) (*ArtifactMonitoringPage, error) {
	stub.requests <- request
	return &ArtifactMonitoringPage{}, nil
}
func (stub *monitoringHTTPStub) ListSemanticMonitoring(_ context.Context, request *MonitoringListRequest) (*SemanticMonitoringPage, error) {
	stub.requests <- request
	return &SemanticMonitoringPage{}, nil
}

func TestMonitoringHTTPRoutesValidateAndForwardFrozenFilters(t *testing.T) {
	stub := &monitoringHTTPStub{requests: make(chan *MonitoringListRequest, 3)}
	server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(func(response http.ResponseWriter, _ *http.Request, err error) {
		var public *PublicError
		if errors.As(err, &public) {
			response.WriteHeader(public.Status)
			return
		}
		response.WriteHeader(http.StatusInternalServerError)
	}))
	RegisterAgentRunHTTPServer(server, stub)

	for _, path := range []string{
		"/api/admin/v1/monitoring/collector-executions",
		"/api/admin/v1/monitoring/artifact-extractions",
		"/api/admin/v1/monitoring/semantic-work-items",
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path+"?window=12h&state=failure&page=2&page_size=25", nil)
		server.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, body=%s", path, response.Code, response.Body.String())
		}
		got := <-stub.requests
		if got.Window != "12h" || got.State != "failure" || got.Page != 2 || got.PageSize != 25 {
			t.Fatalf("GET %s request = %+v", path, got)
		}
	}

	for _, path := range []string{
		"/api/admin/v1/monitoring/summary?window=2h",
		"/api/admin/v1/monitoring/collector-executions?window=1h&state=cancelled",
	} {
		response := httptest.NewRecorder()
		server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("GET %s status = %d, want 400", path, response.Code)
		}
	}
}
