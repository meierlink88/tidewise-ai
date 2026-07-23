// Package internalapi owns the versioned Data Service HTTP transport.
package internalapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	publicationdomain "github.com/meierlink88/tidewise-ai/backend/services/data/domain/eventpublication"
	researchanchordomainimport "github.com/meierlink88/tidewise-ai/backend/services/data/domain/researchanchorimport"
	researchdomainimport "github.com/meierlink88/tidewise-ai/backend/services/data/domain/researchthemeimport"
	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/adminquery"
	eventpublicationapp "github.com/meierlink88/tidewise-ai/backend/services/data/usecase/eventpublication"
	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/research"
	researchanchorimportapp "github.com/meierlink88/tidewise-ai/backend/services/data/usecase/researchanchorimport"
	researchimportapp "github.com/meierlink88/tidewise-ai/backend/services/data/usecase/researchthemeimport"
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
	Authenticator         *Authenticator
	EventPublications     EventPublicationService
	ResearchThemeImports  ResearchThemeImportService
	ResearchAnchorImports ResearchAnchorImportService
	Research              ResearchService
	Admin                 AdminService
	NewRequestID          func() string
}

type operation func(http.ResponseWriter, *http.Request, Principal, string)

type businessRoute struct {
	method    string
	path      string
	scope     string
	operation operation
}

func NewHandler(dependencies Dependencies) http.Handler {
	if dependencies.NewRequestID == nil {
		dependencies.NewRequestID = func() string { return fmt.Sprintf("data-%d", time.Now().UTC().UnixNano()) }
	}
	mux := http.NewServeMux()
	for _, route := range dependencies.businessRoutes() {
		mux.Handle(route.method+" "+route.path, dependencies.authorize(route.scope, route.operation))
	}
	return mux
}

func (d Dependencies) businessRoutes() []businessRoute {
	return []businessRoute{
		{method: http.MethodPost, path: Namespace + "/reviewed-event-imports", scope: ScopeReviewedEventImport, operation: d.importEventPublication},
		{method: http.MethodPost, path: Namespace + "/research-theme-imports", scope: ScopeResearchImport, operation: d.importResearchThemes},
		{method: http.MethodPost, path: Namespace + "/research-anchor-imports", scope: ScopeResearchImport, operation: d.importResearchAnchors},
		{method: http.MethodGet, path: Namespace + "/research/themes", scope: ScopeResearchRead, operation: d.listResearchThemes},
		{method: http.MethodGet, path: Namespace + "/research/themes/{theme_id}", scope: ScopeResearchRead, operation: d.getResearchTheme},
		{method: http.MethodGet, path: Namespace + "/research/themes/{theme_id}/reasoning-trees", scope: ScopeResearchRead, operation: d.listResearchThemeReasoningTrees},
		{method: http.MethodGet, path: Namespace + "/research/themes/{theme_id}/reasoning-trees/{anchor_id}", scope: ScopeResearchRead, operation: d.getResearchThemeReasoningTree},
		{method: http.MethodGet, path: Namespace + "/raw-documents", scope: ScopeAdminRead, operation: d.listAdminRawDocuments},
		{method: http.MethodGet, path: Namespace + "/events", scope: ScopeAdminRead, operation: d.listAdminEvents},
	}
}
