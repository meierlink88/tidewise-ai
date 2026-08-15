package organization_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	organizationapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/organization"
)

type httpServiceStub struct {
	operation string
	request   any
}

func (s *httpServiceStub) Create(_ context.Context, request *organizationapi.CreateRequest) (*v1.Response[organizationapi.Organization], error) {
	s.operation, s.request = organizationapi.OperationCreate, request
	return noContent[organizationapi.Organization](), nil
}
func (s *httpServiceStub) List(_ context.Context, request *organizationapi.ListRequest) (*v1.Response[organizationapi.OrganizationList], error) {
	s.operation, s.request = organizationapi.OperationList, request
	return noContent[organizationapi.OrganizationList](), nil
}
func (s *httpServiceStub) Get(_ context.Context, request *organizationapi.GetRequest) (*v1.Response[organizationapi.Organization], error) {
	s.operation, s.request = organizationapi.OperationGet, request
	return noContent[organizationapi.Organization](), nil
}
func (s *httpServiceStub) Update(_ context.Context, request *organizationapi.UpdateRequest) (*v1.Response[organizationapi.Organization], error) {
	s.operation, s.request = organizationapi.OperationUpdate, request
	return noContent[organizationapi.Organization](), nil
}
func (s *httpServiceStub) ReplaceDomainTags(_ context.Context, request *organizationapi.ReplaceDomainTagsRequest) (*v1.Response[organizationapi.Organization], error) {
	s.operation, s.request = organizationapi.OperationReplaceDomainTags, request
	return noContent[organizationapi.Organization](), nil
}
func (s *httpServiceStub) GetCatalog(_ context.Context, request *organizationapi.CatalogRequest) (*v1.Response[organizationapi.Catalog], error) {
	s.operation, s.request = organizationapi.OperationGetCatalog, request
	return noContent[organizationapi.Catalog](), nil
}
func (s *httpServiceStub) ListMembers(_ context.Context, request *organizationapi.ListMembersRequest) (*v1.Response[organizationapi.MemberList], error) {
	s.operation, s.request = organizationapi.OperationListMembers, request
	return noContent[organizationapi.MemberList](), nil
}
func (s *httpServiceStub) CreateMember(_ context.Context, request *organizationapi.CreateMemberRequest) (*v1.Response[organizationapi.Member], error) {
	s.operation, s.request = organizationapi.OperationCreateMember, request
	return noContent[organizationapi.Member](), nil
}
func (s *httpServiceStub) UpdateMember(_ context.Context, request *organizationapi.UpdateMemberRequest) (*v1.Response[organizationapi.Member], error) {
	s.operation, s.request = organizationapi.OperationUpdateMember, request
	return noContent[organizationapi.Member](), nil
}
func (s *httpServiceStub) DeleteMember(_ context.Context, request *organizationapi.DeleteMemberRequest) (*v1.Response[organizationapi.DeleteResult], error) {
	s.operation, s.request = organizationapi.OperationDeleteMember, request
	return noContent[organizationapi.DeleteResult](), nil
}

func TestHTTPRoutesBindOrganizationOperationsAndRequests(t *testing.T) {
	for _, test := range []struct {
		name, method, path, body, operation string
		assertRequest                       func(*testing.T, any)
	}{
		{name: "create", method: http.MethodPost, path: "/entities/organizations", body: `{}`, operation: organizationapi.OperationCreate},
		{name: "list", method: http.MethodGet, path: "/entities/organizations?category_code=TRADE_BLOC&function_code=TRADE&region_id=REG88d53cc8-1c75-57e6-a02c-56f9a4bc13c4&country_id=COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b&as_of=2026-08-15", operation: organizationapi.OperationList, assertRequest: func(t *testing.T, input any) {
			request := input.(*organizationapi.ListRequest)
			if request.CategoryCode != "TRADE_BLOC" || request.FunctionCode != "TRADE" || request.RegionID != "REG88d53cc8-1c75-57e6-a02c-56f9a4bc13c4" || request.CountryID != "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b" || request.AsOf != "2026-08-15" {
				t.Fatalf("list request = %#v", request)
			}
		}},
		{name: "get", method: http.MethodGet, path: "/entities/organizations/ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", operation: organizationapi.OperationGet, assertRequest: assertOrganizationID},
		{name: "update", method: http.MethodPut, path: "/entities/organizations/ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", body: `{}`, operation: organizationapi.OperationUpdate, assertRequest: assertOrganizationID},
		{name: "replace tags", method: http.MethodPut, path: "/entities/organizations/ORG3fb9e7ff-2222-57fa-b306-c223ce3af549/domain-tags", body: `{"domain_tag_codes":["REGIONAL_SECURITY_DIALOGUE"]}`, operation: organizationapi.OperationReplaceDomainTags, assertRequest: assertOrganizationID},
		{name: "catalog", method: http.MethodGet, path: "/organization-catalog", operation: organizationapi.OperationGetCatalog},
		{name: "list members", method: http.MethodGet, path: "/entities/organizations/ORG3fb9e7ff-2222-57fa-b306-c223ce3af549/members?as_of=2020-06-01", operation: organizationapi.OperationListMembers, assertRequest: func(t *testing.T, input any) {
			request := input.(*organizationapi.ListMembersRequest)
			if request.OrganizationID != "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549" || request.AsOf != "2020-06-01" {
				t.Fatalf("list members request = %#v", request)
			}
		}},
		{name: "create member", method: http.MethodPost, path: "/entities/organizations/ORG3fb9e7ff-2222-57fa-b306-c223ce3af549/members", body: `{}`, operation: organizationapi.OperationCreateMember, assertRequest: assertOrganizationID},
		{name: "update member", method: http.MethodPut, path: "/entities/organizations/ORG3fb9e7ff-2222-57fa-b306-c223ce3af549/members/OMB77777777-7777-4777-8777-777777777777", body: `{}`, operation: organizationapi.OperationUpdateMember, assertRequest: assertOrganizationMemberID},
		{name: "delete member", method: http.MethodDelete, path: "/entities/organizations/ORG3fb9e7ff-2222-57fa-b306-c223ce3af549/members/OMB77777777-7777-4777-8777-777777777777", operation: organizationapi.OperationDeleteMember, assertRequest: assertOrganizationMemberID},
	} {
		t.Run(test.name, func(t *testing.T) {
			stub := &httpServiceStub{}
			var transportOperation string
			server := newOrganizationHTTPServer(stub, func(next middleware.Handler) middleware.Handler {
				return func(ctx context.Context, request any) (any, error) {
					if serverTransport, ok := transport.FromServerContext(ctx); ok {
						transportOperation = serverTransport.Operation()
					}
					return next(ctx, request)
				}
			})
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(test.method, v1.APIPrefix+test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			server.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want 204; body=%s", recorder.Code, recorder.Body.String())
			}
			if stub.operation != test.operation || transportOperation != test.operation {
				t.Fatalf("operations: service=%q transport=%q want=%q", stub.operation, transportOperation, test.operation)
			}
			if test.assertRequest != nil {
				test.assertRequest(t, stub.request)
			}
		})
	}
}

func TestHTTPRejectsUnknownFields(t *testing.T) {
	server := newOrganizationHTTPServer(&httpServiceStub{}, nil)
	for _, test := range []struct {
		name, method, path, body string
		wantStatus               int
	}{
		{name: "unknown create field", method: http.MethodPost, path: "/entities/organizations", body: `{"unknown":true}`, wantStatus: http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(test.method, v1.APIPrefix+test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			server.ServeHTTP(recorder, request)
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, test.wantStatus, recorder.Body.String())
			}
		})
	}
}

func newOrganizationHTTPServer(application organizationapi.Service, middlewares ...middleware.Middleware) *kratoshttp.Server {
	server := kratoshttp.NewServer(
		kratoshttp.Middleware(middlewares...),
		kratoshttp.ErrorEncoder(func(response http.ResponseWriter, _ *http.Request, err error) {
			if public, ok := err.(*v1.PublicError); ok {
				response.WriteHeader(public.Status)
				_ = json.NewEncoder(response).Encode(public)
				return
			}
			response.WriteHeader(http.StatusInternalServerError)
		}),
	)
	organizationapi.RegisterHTTPServer(server, application)
	return server
}

func noContent[T any]() *v1.Response[T] {
	return &v1.Response[T]{Status: http.StatusNoContent}
}

func assertOrganizationID(t *testing.T, input any) {
	t.Helper()
	var id string
	switch request := input.(type) {
	case *organizationapi.GetRequest:
		id = request.OrganizationID
	case *organizationapi.UpdateRequest:
		id = request.OrganizationID
	case *organizationapi.ReplaceDomainTagsRequest:
		id = request.OrganizationID
	case *organizationapi.CreateMemberRequest:
		id = request.OrganizationID
	default:
		t.Fatalf("request type = %T", input)
	}
	if id != "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549" {
		t.Fatalf("organization ID = %q", id)
	}
}

func assertOrganizationMemberID(t *testing.T, input any) {
	t.Helper()
	var organizationID string
	var memberID string
	switch request := input.(type) {
	case *organizationapi.UpdateMemberRequest:
		organizationID, memberID = request.OrganizationID, request.MemberID
	case *organizationapi.DeleteMemberRequest:
		organizationID, memberID = request.OrganizationID, request.MemberID
	default:
		t.Fatalf("request type = %T", input)
	}
	if organizationID != "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549" || memberID != "OMB77777777-7777-4777-8777-777777777777" {
		t.Fatalf("member identity = %q/%s", organizationID, memberID)
	}
}
