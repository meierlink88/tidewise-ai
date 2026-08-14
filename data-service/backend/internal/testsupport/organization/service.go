package organization

import (
	"context"
	"net/http"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	organizationapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/organization"
)

// Service is a neutral Organization HTTP dependency for tests focused on another API.
type Service struct{}

func (Service) Create(context.Context, *organizationapi.CreateRequest) (*v1.Response[organizationapi.Organization], error) {
	return &v1.Response[organizationapi.Organization]{Status: http.StatusNoContent}, nil
}

func (Service) List(context.Context, *organizationapi.ListRequest) (*v1.Response[organizationapi.OrganizationList], error) {
	return &v1.Response[organizationapi.OrganizationList]{Status: http.StatusNoContent}, nil
}

func (Service) Get(context.Context, *organizationapi.GetRequest) (*v1.Response[organizationapi.Organization], error) {
	return &v1.Response[organizationapi.Organization]{Status: http.StatusNoContent}, nil
}

func (Service) Update(context.Context, *organizationapi.UpdateRequest) (*v1.Response[organizationapi.Organization], error) {
	return &v1.Response[organizationapi.Organization]{Status: http.StatusNoContent}, nil
}
func (Service) ReplaceDomainTags(context.Context, *organizationapi.ReplaceDomainTagsRequest) (*v1.Response[organizationapi.Organization], error) {
	return &v1.Response[organizationapi.Organization]{Status: http.StatusNoContent}, nil
}
func (Service) GetCatalog(context.Context, *organizationapi.CatalogRequest) (*v1.Response[organizationapi.Catalog], error) {
	return &v1.Response[organizationapi.Catalog]{Status: http.StatusNoContent}, nil
}
func (Service) ListMembers(context.Context, *organizationapi.ListMembersRequest) (*v1.Response[organizationapi.MemberList], error) {
	return &v1.Response[organizationapi.MemberList]{Status: http.StatusNoContent}, nil
}
func (Service) CreateMember(context.Context, *organizationapi.CreateMemberRequest) (*v1.Response[organizationapi.Member], error) {
	return &v1.Response[organizationapi.Member]{Status: http.StatusNoContent}, nil
}
func (Service) UpdateMember(context.Context, *organizationapi.UpdateMemberRequest) (*v1.Response[organizationapi.Member], error) {
	return &v1.Response[organizationapi.Member]{Status: http.StatusNoContent}, nil
}
func (Service) DeleteMember(context.Context, *organizationapi.DeleteMemberRequest) (*v1.Response[organizationapi.DeleteResult], error) {
	return &v1.Response[organizationapi.DeleteResult]{Status: http.StatusNoContent}, nil
}

var _ organizationapi.Service = Service{}
