package organization

import (
	"context"
	"strconv"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const ExecutionBudget = 5 * time.Second

func RegisterHTTPServer(server *kratoshttp.Server, application Service) {
	router := server.Route(v1.APIPrefix)
	router.POST("/entities/organizations", createHandler(application))
	router.GET("/entities/organizations", listHandler(application))
	router.GET("/entities/organizations/{organization_id}", getHandler(application))
	router.PUT("/entities/organizations/{organization_id}", updateHandler(application))
	router.PUT("/entities/organizations/{organization_id}/domain-tags", replaceDomainTagsHandler(application))
	router.GET("/organization-catalog", catalogHandler(application))
	router.GET("/entities/organizations/{organization_id}/members", listMembersHandler(application))
	router.POST("/entities/organizations/{organization_id}/members", createMemberHandler(application))
	router.PUT("/entities/organizations/{organization_id}/members/{member_id}", updateMemberHandler(application))
	router.DELETE("/entities/organizations/{organization_id}/members/{member_id}", deleteMemberHandler(application))
}

func updateHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request, err := v1.DecodeStrictJSONBody[UpdateRequest](ctx)
		if err != nil {
			return err
		}
		request.OrganizationID = ctx.Vars().Get("organization_id")
		return call(ctx, OperationUpdate, request, func(callContext context.Context) (*v1.Response[Organization], error) {
			return application.Update(callContext, request)
		})
	}
}

func replaceDomainTagsHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request, err := v1.DecodeStrictJSONBody[ReplaceDomainTagsRequest](ctx)
		if err != nil {
			return err
		}
		request.OrganizationID = ctx.Vars().Get("organization_id")
		return call(ctx, OperationReplaceDomainTags, request, func(callContext context.Context) (*v1.Response[Organization], error) {
			return application.ReplaceDomainTags(callContext, request)
		})
	}
}

func catalogHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &CatalogRequest{}
		return call(ctx, OperationGetCatalog, request, func(callContext context.Context) (*v1.Response[Catalog], error) {
			return application.GetCatalog(callContext, request)
		})
	}
}

func listMembersHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ListMembersRequest{OrganizationID: ctx.Vars().Get("organization_id"), AsOf: ctx.Query().Get("as_of")}
		return call(ctx, OperationListMembers, request, func(callContext context.Context) (*v1.Response[MemberList], error) {
			return application.ListMembers(callContext, request)
		})
	}
}

func createMemberHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request, err := v1.DecodeStrictJSONBody[CreateMemberRequest](ctx)
		if err != nil {
			return err
		}
		request.OrganizationID = ctx.Vars().Get("organization_id")
		return call(ctx, OperationCreateMember, request, func(callContext context.Context) (*v1.Response[Member], error) {
			return application.CreateMember(callContext, request)
		})
	}
}

func updateMemberHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request, err := v1.DecodeStrictJSONBody[UpdateMemberRequest](ctx)
		if err != nil {
			return err
		}
		request.OrganizationID = ctx.Vars().Get("organization_id")
		request.MemberID, err = strconv.ParseInt(ctx.Vars().Get("member_id"), 10, 64)
		if err != nil {
			return v1.NewPublicError(v1.StatusUnprocessableEntity, "ORGANIZATION_MEMBER_INVALID", "Organization member ID is invalid", nil)
		}
		return call(ctx, OperationUpdateMember, request, func(callContext context.Context) (*v1.Response[Member], error) {
			return application.UpdateMember(callContext, request)
		})
	}
}

func deleteMemberHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		memberID, err := strconv.ParseInt(ctx.Vars().Get("member_id"), 10, 64)
		if err != nil {
			return v1.NewPublicError(v1.StatusUnprocessableEntity, "ORGANIZATION_MEMBER_INVALID", "Organization member ID is invalid", nil)
		}
		request := &DeleteMemberRequest{OrganizationID: ctx.Vars().Get("organization_id"), MemberID: memberID}
		return call(ctx, OperationDeleteMember, request, func(callContext context.Context) (*v1.Response[DeleteResult], error) {
			return application.DeleteMember(callContext, request)
		})
	}
}

func createHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request, err := v1.DecodeStrictJSONBody[CreateRequest](ctx)
		if err != nil {
			return err
		}
		return call(ctx, OperationCreate, request, func(callContext context.Context) (*v1.Response[Organization], error) {
			return application.Create(callContext, request)
		})
	}
}

func listHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ListRequest{
			CategoryCode: ctx.Query().Get("category_code"), FunctionCode: ctx.Query().Get("function_code"),
			RegionID: ctx.Query().Get("region_id"), CountryID: ctx.Query().Get("country_id"), AsOf: ctx.Query().Get("as_of"),
		}
		return call(ctx, OperationList, request, func(callContext context.Context) (*v1.Response[OrganizationList], error) {
			return application.List(callContext, request)
		})
	}
}

func getHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &GetRequest{OrganizationID: ctx.Vars().Get("organization_id")}
		return call(ctx, OperationGet, request, func(callContext context.Context) (*v1.Response[Organization], error) {
			return application.Get(callContext, request)
		})
	}
}

func call[T any](ctx kratoshttp.Context, operation string, request any, invoke func(context.Context) (*v1.Response[T], error)) error {
	return v1.Call(ctx, operation, request, func(callContext context.Context) (*v1.Response[T], error) {
		deadlineContext, cancel := context.WithTimeout(callContext, ExecutionBudget)
		defer cancel()
		return invoke(deadlineContext)
	})
}
