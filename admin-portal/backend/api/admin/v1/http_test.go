package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
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
		path      string
		operation string
	}{
		{APIPrefix + "/raw-documents", OperationListRawDocuments},
		{APIPrefix + "/events", OperationListEvents},
		{APIPrefix + "/runtime-health", OperationGetRuntimeHealth},
	} {
		response := httptest.NewRecorder()
		server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, request.path, nil))
		if response.Code != http.StatusOK || seen[request.operation] != 1 {
			t.Fatalf("GET %s status=%d middleware=%d", request.path, response.Code, seen[request.operation])
		}
	}
}

func TestRetiredManagementRoutesAreNotRegistered(t *testing.T) {
	server := kratoshttp.NewServer()
	RegisterAdminHTTPServer(server, stubAdminHTTPServer{})
	for _, path := range []string{
		APIPrefix + "/agent-schedules/collector",
		APIPrefix + "/agent-statuses",
		APIPrefix + "/monitoring/summary",
		APIPrefix + "/model-providers",
		APIPrefix + "/connectors",
	} {
		response := httptest.NewRecorder()
		server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("GET %s status=%d, want 404", path, response.Code)
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
func (stubAdminHTTPServer) GetRuntimeHealth(context.Context, *EmptyRequest) (*RuntimeHealth, error) {
	return &RuntimeHealth{}, nil
}
