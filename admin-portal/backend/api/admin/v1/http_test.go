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
		{APIPrefix + "/events", OperationListEvents},
		{APIPrefix + "/evidences", OperationListEvidences},
		{APIPrefix + "/raw-evidences/RAW00000000-0000-5000-8000-000000000001/collection-document", OperationGetCollectionDocument},
		{APIPrefix + "/evidence-categories", OperationListEvidenceCategories},
		{APIPrefix + "/sources", OperationListSources},
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
		APIPrefix + "/raw-documents",
	} {
		response := httptest.NewRecorder()
		server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("GET %s status=%d, want 404", path, response.Code)
		}
	}
}

func TestEventListRejectsRetiredOrUnknownFilters(t *testing.T) {
	server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(func(writer http.ResponseWriter, _ *http.Request, err error) {
		if public, ok := PublicError(err); ok {
			writer.WriteHeader(public.Status())
			return
		}
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	RegisterAdminHTTPServer(server, stubAdminHTTPServer{})
	for _, query := range []string{"event_status=confirmed", "fact_status=verified", "first_seen_from=2026-08-18T00%3A00%3A00Z", "tag=macro"} {
		response := httptest.NewRecorder()
		server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, APIPrefix+"/events?"+query, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("query %q status=%d, want 400", query, response.Code)
		}
	}
}

type stubAdminHTTPServer struct{}

func (stubAdminHTTPServer) ListEvents(context.Context, *ListEventsRequest) (*EventListResponse, error) {
	return &EventListResponse{}, nil
}
func (stubAdminHTTPServer) ListEvidences(context.Context, *ListEvidencesRequest) (*EvidenceListResponse, error) {
	return &EvidenceListResponse{}, nil
}
func (stubAdminHTTPServer) GetCollectionDocument(context.Context, *GetCollectionDocumentRequest) (*CollectionDocumentResponse, error) {
	return &CollectionDocumentResponse{}, nil
}
func (stubAdminHTTPServer) ListEvidenceCategories(context.Context, *EmptyRequest) (*EvidenceCategoryListResponse, error) {
	return &EvidenceCategoryListResponse{}, nil
}
func (stubAdminHTTPServer) ListSources(context.Context, *ListSourcesRequest) (*SourceListResponse, error) {
	return &SourceListResponse{}, nil
}
func (stubAdminHTTPServer) GetRuntimeHealth(context.Context, *EmptyRequest) (*RuntimeHealth, error) {
	return &RuntimeHealth{}, nil
}
