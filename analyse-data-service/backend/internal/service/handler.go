// Package internalapi owns the versioned Data Service HTTP transport.
package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/adminquery"
	eventpublicationapp "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventpublication"
	publicationdomain "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventpublication"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
	researchanchordomainimport "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchanchorimport"
	researchanchorimportapp "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchanchorimport"
	researchdomainimport "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
	researchimportapp "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
)

const (
	Namespace           = "/api/data/v1"
	MaxRequestBodyBytes = 1_048_576

	ScopeResearchRead        = "data.research.read"
	ScopeResearchImport      = "data.research.import"
	ScopeAdminRead           = "data.admin.read"
	ScopeReviewedEventImport = "data.reviewed-events.import"
)

type EventPublicationService interface {
	Import(context.Context, string, publicationdomain.Publication) (eventpublicationapp.Result, error)
}

type ResearchThemeImportService interface {
	Import(context.Context, string, researchdomainimport.Batch) (researchimportapp.Result, error)
}

type ResearchAnchorImportService interface {
	Import(context.Context, string, researchanchordomainimport.Publication) (researchanchorimportapp.Result, error)
}

type ResearchService interface {
	ListThemes(context.Context, research.ResearchListRequest) (research.ResearchThemePage, error)
	GetTheme(context.Context, string, research.ResearchDetailRequest) (research.ResearchThemeDetail, error)
	ListReasoningTrees(context.Context, string) (research.ResearchReasoningTreeList, error)
	GetReasoningTree(context.Context, string, string) (research.ResearchReasoningTreeDetail, error)
}

type AdminService interface {
	ListRawDocuments(context.Context, adminquery.RawDocumentListRequest) (adminquery.RawDocumentPage, error)
	ListEvents(context.Context, adminquery.EventListRequest) (adminquery.EventPage, error)
}

type Dependencies struct {
	EventPublications     EventPublicationService
	ResearchThemeImports  ResearchThemeImportService
	ResearchAnchorImports ResearchAnchorImportService
	Research              ResearchService
	Admin                 AdminService
	NewRequestID          func() string
}

type operation func(http.ResponseWriter, *http.Request, Principal, string)

type DataService struct {
	dependencies Dependencies
}

func NewDataService(dependencies Dependencies) *DataService {
	if dependencies.NewRequestID == nil {
		dependencies.NewRequestID = func() string { return fmt.Sprintf("data-%d", time.Now().UTC().UnixNano()) }
	}
	return &DataService{dependencies: dependencies}
}

func (s *DataService) invoke(ctx kratoshttp.Context, operation operation) error {
	request := ctx.Request()
	for key, values := range ctx.Vars() {
		if len(values) > 0 {
			request.SetPathValue(key, values[0])
		}
	}
	requestID := request.Header.Get("X-Request-ID")
	if requestID == "" || len(requestID) > 128 {
		requestID = s.dependencies.NewRequestID()
		request.Header.Set("X-Request-ID", requestID)
		ctx.Response().Header().Set("X-Request-ID", requestID)
	}
	principal, _ := PrincipalFromContext(request.Context())
	operation(ctx.Response(), request, principal, requestID)
	return nil
}

func (s *DataService) ImportReviewedEvents(ctx kratoshttp.Context) error {
	return s.invoke(ctx, s.dependencies.importEventPublication)
}

func (s *DataService) ImportResearchThemes(ctx kratoshttp.Context) error {
	return s.invoke(ctx, s.dependencies.importResearchThemes)
}

func (s *DataService) ImportResearchAnchors(ctx kratoshttp.Context) error {
	return s.invoke(ctx, s.dependencies.importResearchAnchors)
}

func (s *DataService) ListResearchThemes(ctx kratoshttp.Context) error {
	return s.invoke(ctx, s.dependencies.listResearchThemes)
}

func (s *DataService) GetResearchTheme(ctx kratoshttp.Context) error {
	return s.invoke(ctx, s.dependencies.getResearchTheme)
}

func (s *DataService) ListResearchReasoningTrees(ctx kratoshttp.Context) error {
	return s.invoke(ctx, s.dependencies.listResearchThemeReasoningTrees)
}

func (s *DataService) GetResearchReasoningTree(ctx kratoshttp.Context) error {
	return s.invoke(ctx, s.dependencies.getResearchThemeReasoningTree)
}

func (s *DataService) ListRawDocuments(ctx kratoshttp.Context) error {
	return s.invoke(ctx, s.dependencies.listAdminRawDocuments)
}

func (s *DataService) ListEvents(ctx kratoshttp.Context) error {
	return s.invoke(ctx, s.dependencies.listAdminEvents)
}
