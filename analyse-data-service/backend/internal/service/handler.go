package service

import (
	"context"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/adminquery"
	eventpublicationapp "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventpublication"
	publicationdomain "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventpublication"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventsemantics"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventtagcatalog"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchanalysiscontext"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchgraph"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchpublication"
)

const (
	Namespace = v1.APIPrefix

	ScopeResearchRead        = "data.research.read"
	ScopeResearchImport      = "data.research.import"
	ScopeAdminRead           = "data.admin.read"
	ScopeReviewedEventImport = "data.reviewed-events.import"
	ScopeEventTagRead        = "data.event-tags.read"
	ScopeEventSemanticsRead  = "data.event-semantics.read"
	ScopeEventSemanticsWrite = "data.event-semantics.write"
)

type EventPublicationService interface {
	Import(context.Context, string, publicationdomain.Publication) (eventpublicationapp.Result, error)
}

type EventTagCatalogService interface {
	Active(context.Context) (eventtagcatalog.Catalog, error)
}

type EventSemanticsService interface {
	ListEligibleEvents(context.Context, int) ([]eventsemantics.EligibleEvent, error)
	CreateContextLease(context.Context, eventsemantics.ContextLeaseRequest) (eventsemantics.ContextLease, error)
	Context(context.Context, string) (eventsemantics.Context, error)
	Resolve(context.Context, string, []eventsemantics.EntityMention) ([]eventsemantics.EntityResolution, error)
	SearchDirectTargets(context.Context, string, string, []string) ([]eventsemantics.DirectTarget, error)
	CreateSubmission(context.Context, eventsemantics.Submission) (eventsemantics.SubmissionResult, error)
	SubmitReview(context.Context, eventsemantics.ReviewSubmission) (eventsemantics.SubmissionResult, error)
	Get(context.Context, string) (eventsemantics.EventSemanticsResult, error)
}

type ResearchThemeImportService interface {
	Publish(context.Context, string, researchpublication.Aggregate) (researchpublication.Result, error)
}

type ResearchService interface {
	ListThemes(context.Context, research.ResearchListRequest) (research.ResearchThemePage, error)
	GetTheme(context.Context, string, research.ResearchDetailRequest) (research.ResearchThemeDetail, error)
	ListReasoningTrees(context.Context, string) (research.ResearchReasoningTreeList, error)
	GetReasoningTree(context.Context, string, string) (research.ResearchReasoningTreeDetail, error)
}

type ResearchAnalysisContextService interface {
	List(
		context.Context,
		researchanalysiscontext.Request,
	) (researchanalysiscontext.Result, error)
}

type ResearchGraphService interface {
	Search(context.Context, researchgraph.Request) (researchgraph.Result, error)
}

type AdminService interface {
	ListRawDocuments(context.Context, adminquery.RawDocumentListRequest) (adminquery.RawDocumentPage, error)
	ListEvents(context.Context, adminquery.EventListRequest) (adminquery.EventPage, error)
}

type Dependencies struct {
	EventPublications       EventPublicationService
	EventTagCatalog         EventTagCatalogService
	EventSemantics          EventSemanticsService
	ResearchThemeImports    ResearchThemeImportService
	Research                ResearchService
	ResearchAnalysisContext ResearchAnalysisContextService
	ResearchGraph           ResearchGraphService
	Admin                   AdminService
}

type DataService struct {
	dependencies Dependencies
}

func NewDataService(dependencies Dependencies) *DataService {
	return &DataService{dependencies: dependencies}
}

var _ v1.DataHTTPServer = (*DataService)(nil)
