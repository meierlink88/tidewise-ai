package service

import (
	"context"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
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
	Namespace = v1.APIPrefix

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
}

type DataService struct {
	dependencies Dependencies
}

func NewDataService(dependencies Dependencies) *DataService {
	return &DataService{dependencies: dependencies}
}

var _ v1.DataHTTPServer = (*DataService)(nil)
