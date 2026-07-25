package v1

import (
	"context"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

const APIPrefix = "/api/data/v1"

type DataHTTPServer interface {
	ImportReviewedEvents(kratoshttp.Context) error
	ImportResearchThemes(kratoshttp.Context) error
	ImportResearchAnchors(kratoshttp.Context) error
	ListResearchThemes(kratoshttp.Context) error
	GetResearchTheme(kratoshttp.Context) error
	ListResearchReasoningTrees(kratoshttp.Context) error
	GetResearchReasoningTree(kratoshttp.Context) error
	ListRawDocuments(kratoshttp.Context) error
	ListEvents(kratoshttp.Context) error
}

func RegisterDataHTTPServer(server *kratoshttp.Server, application DataHTTPServer) {
	router := server.Route(APIPrefix)
	router.POST("/reviewed-event-imports", call("data.v1.publishReviewedEvents", application.ImportReviewedEvents))
	router.POST("/research-theme-imports", call("data.v1.importResearchThemes", application.ImportResearchThemes))
	router.POST("/research-anchor-imports", call("data.v1.importResearchAnchors", application.ImportResearchAnchors))
	router.GET("/research/themes", call("data.v1.listResearchThemes", application.ListResearchThemes))
	router.GET("/research/themes/{theme_id}", call("data.v1.getResearchTheme", application.GetResearchTheme))
	router.GET("/research/themes/{theme_id}/reasoning-trees", call("data.v1.listResearchThemeReasoningTrees", application.ListResearchReasoningTrees))
	router.GET("/research/themes/{theme_id}/reasoning-trees/{anchor_id}", call("data.v1.getResearchThemeReasoningTree", application.GetResearchReasoningTree))
	router.GET("/raw-documents", call("data.v1.listAdminRawDocuments", application.ListRawDocuments))
	router.GET("/events", call("data.v1.listAdminEvents", application.ListEvents))
}

func call(operation string, invoke func(kratoshttp.Context) error) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		kratoshttp.SetOperation(ctx, operation)
		handler := ctx.Middleware(func(context.Context, any) (any, error) {
			return nil, invoke(ctx)
		})
		_, err := handler(ctx, nil)
		return err
	}
}
