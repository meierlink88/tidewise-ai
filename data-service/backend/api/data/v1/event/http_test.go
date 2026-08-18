package event_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	eventapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/event"
)

func TestHTTPExposesOnlyRetainedEventListRoute(t *testing.T) {
	var operation string
	recorder := func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, request any) (any, error) {
			if serverTransport, ok := transport.FromServerContext(ctx); ok {
				operation = serverTransport.Operation()
			}
			return next(ctx, request)
		}
	}
	service := new(capturingService)
	server := kratoshttp.NewServer(kratoshttp.Middleware(recorder))
	eventapi.RegisterHTTPServer(server, service)

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet,
		v1.APIPrefix+"/events?title=Event&modality=FACT&status=ACTIVE&occurred_from=2026-08-18T00%3A00%3A00Z&announced_to=2026-08-19T00%3A00%3A00Z&page=2&page_size=10", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if operation != eventapi.OperationListAdminEvents {
		t.Fatalf("operation = %q", operation)
	}
	if service.request == nil || service.request.Modality != "FACT" || service.request.Status != "ACTIVE" || service.request.Page != "2" {
		t.Fatalf("request = %#v", service.request)
	}

	for _, retired := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: v1.APIPrefix + "/reviewed-event-imports"},
		{method: http.MethodGet, path: v1.APIPrefix + "/event-tags?active=true"},
	} {
		response := httptest.NewRecorder()
		server.ServeHTTP(response, httptest.NewRequest(retired.method, retired.path, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("retired %s %s status = %d", retired.method, retired.path, response.Code)
		}
	}
}

type capturingService struct{ request *eventapi.ListRequest }

func (s *capturingService) ListEvents(_ context.Context, request *eventapi.ListRequest) (*v1.Response[eventapi.Page], error) {
	s.request = request
	return &v1.Response[eventapi.Page]{Status: v1.StatusOK, Result: eventapi.Page{Items: []eventapi.Item{}}}, nil
}
