package rawdocument

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
)

func TestHTTPBindingPreservesOperationAndQuery(t *testing.T) {
	application := &capturingService{}
	var operation string
	recorder := func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, request any) (any, error) {
			if serverTransport, ok := transport.FromServerContext(ctx); ok {
				operation = serverTransport.Operation()
			}
			return next(ctx, request)
		}
	}
	server := kratoshttp.NewServer(kratoshttp.Middleware(recorder))
	RegisterHTTPServer(server, application)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, v1.APIPrefix+"/raw-documents?title=chip&source_ref=feed&ingest_status=collected&page=2&page_size=10", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d", response.Code)
	}
	if operation != OperationList {
		t.Fatalf("operation = %q", operation)
	}
	if application.request == nil || application.request.Title != "chip" || application.request.SourceRef != "feed" || application.request.IngestStatus != "collected" || application.request.Page != "2" || application.request.PageSize != "10" {
		t.Fatalf("request = %#v", application.request)
	}
}

type capturingService struct{ request *ListRequest }

func (s *capturingService) List(_ context.Context, request *ListRequest) (*v1.Response[Page], error) {
	s.request = request
	return &v1.Response[Page]{Status: http.StatusNoContent}, nil
}
