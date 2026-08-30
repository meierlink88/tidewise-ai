package event_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestHTTPRejectsRetiredOrUnknownEventFilters(t *testing.T) {
	service := new(capturingService)
	server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(func(writer http.ResponseWriter, _ *http.Request, err error) {
		var public *v1.PublicError
		if errors.As(err, &public) {
			writer.WriteHeader(public.Status)
			return
		}
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	eventapi.RegisterHTTPServer(server, service)

	for _, query := range []string{"event_status=confirmed", "fact_status=verified", "event_time_from=2026-08-18T00%3A00%3A00Z", "tag=macro"} {
		response := httptest.NewRecorder()
		server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, v1.APIPrefix+"/events?"+query, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("query %q status = %d, body = %s", query, response.Code, response.Body.String())
		}
	}
	if service.request != nil {
		t.Fatalf("rejected query reached Event service: %#v", service.request)
	}
}

func TestHTTPAcceptsStrictEventPublicationContract(t *testing.T) {
	service := new(capturingService)
	server := kratoshttp.NewServer()
	eventapi.RegisterHTTPServer(server, service)
	body := `{"publication_key":"submission-1","event":{"title":"US expands HBM controls","summary":"The US announced expanded controls.","semantic":{"actors":["US government"],"action":"expands export controls","objects":["HBM"],"stage":"ANNOUNCED","modality":"FACT","time":{"occurred_at":null,"announced_at":"2026-08-25T00:00:00Z","effective_at":null,"precision":"DAY"},"jurisdictions":["China"],"reason":null,"method":"rule update","metrics":[]}},"evidence_ids":["EVD11111111-1111-4111-8111-111111111111"]}`
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodPost, v1.APIPrefix+"/events", strings.NewReader(body)))
	if response.Code != http.StatusCreated || service.publication == nil || service.publication.PublicationKey != "submission-1" {
		t.Fatalf("status = %d, request = %#v, body = %s", response.Code, service.publication, response.Body.String())
	}
}

func TestHTTPRejectsIncompleteOrExpandedEventSemantic(t *testing.T) {
	const valid = `{"publication_key":"submission-1","event":{"title":"US expands HBM controls","summary":"The US announced expanded controls.","semantic":{"actors":["US government"],"action":"expands export controls","objects":["HBM"],"stage":"ANNOUNCED","modality":"FACT","time":{"occurred_at":null,"announced_at":"2026-08-25T00:00:00Z","effective_at":null,"precision":"DAY"},"jurisdictions":["China"],"reason":null,"method":"rule update","metrics":[]}},"evidence_ids":["EVD11111111-1111-4111-8111-111111111111"]}`
	for _, test := range []struct {
		name string
		body string
	}{
		{name: "missing semantic field", body: strings.Replace(valid, `,"metrics":[]`, "", 1)},
		{name: "extra semantic field", body: strings.Replace(valid, `,"metrics":[]`, `,"metrics":[],"attribution":null`, 1)},
		{name: "missing time field", body: strings.Replace(valid, `"occurred_at":null,`, "", 1)},
		{name: "extra metric field", body: strings.Replace(valid, `"metrics":[]`, `"metrics":[{"name":"capacity","value":"10","unit":"units","change":null,"period":null,"extra":null}]`, 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := new(capturingService)
			server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(func(writer http.ResponseWriter, _ *http.Request, err error) {
				var public *v1.PublicError
				if errors.As(err, &public) {
					writer.WriteHeader(public.Status)
					return
				}
				writer.WriteHeader(http.StatusInternalServerError)
			}))
			eventapi.RegisterHTTPServer(server, service)
			response := httptest.NewRecorder()
			server.ServeHTTP(response, httptest.NewRequest(http.MethodPost, v1.APIPrefix+"/events", strings.NewReader(test.body)))
			if response.Code != http.StatusBadRequest || service.publication != nil {
				t.Fatalf("status = %d, request = %#v, body = %s", response.Code, service.publication, response.Body.String())
			}
		})
	}
}

type capturingService struct {
	request     *eventapi.ListRequest
	publication *eventapi.PublicationRequest
}

func (s *capturingService) ListEvents(_ context.Context, request *eventapi.ListRequest) (*v1.Response[eventapi.Page], error) {
	s.request = request
	return &v1.Response[eventapi.Page]{Status: v1.StatusOK, Result: eventapi.Page{Items: []eventapi.Item{}}}, nil
}

func (s *capturingService) PublishEvent(_ context.Context, request *eventapi.PublicationRequest) (*v1.Response[eventapi.PublicationResult], error) {
	s.publication = request
	return &v1.Response[eventapi.PublicationResult]{Status: v1.StatusCreated}, nil
}
