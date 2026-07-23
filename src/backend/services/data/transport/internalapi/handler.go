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
	Namespace                 = "/internal/data/v1"
	EventPublicationNamespace = "/internal/data/v2"
	MaxRequestBodyBytes       = 1_048_576

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

func NewHandler(dependencies Dependencies) http.Handler {
	if dependencies.NewRequestID == nil {
		dependencies.NewRequestID = func() string { return fmt.Sprintf("data-%d", time.Now().UTC().UnixNano()) }
	}
	mux := http.NewServeMux()
	mux.Handle("POST "+Namespace+"/raw-document-imports", dependencies.authorize(ScopeReviewedEventImport, dependencies.retiredEventImport))
	mux.Handle("GET "+Namespace+"/raw-document-imports/{idempotency_key}", dependencies.authorize(ScopeReviewedEventImport, dependencies.retiredEventImport))
	mux.Handle("POST "+Namespace+"/reviewed-event-imports", dependencies.authorize(ScopeReviewedEventImport, dependencies.retiredEventImport))
	mux.Handle("POST "+EventPublicationNamespace+"/reviewed-event-imports", dependencies.authorize(ScopeReviewedEventImport, dependencies.importEventPublication))
	mux.Handle("POST "+Namespace+"/research-theme-imports", dependencies.authorize(ScopeResearchImport, dependencies.importResearchThemes))
	mux.Handle("POST "+Namespace+"/research-anchor-imports", dependencies.authorize(ScopeResearchImport, dependencies.importResearchAnchors))
	mux.Handle("GET "+Namespace+"/research/themes", dependencies.authorize(ScopeResearchRead, dependencies.listResearchThemes))
	mux.Handle("GET "+Namespace+"/research/themes/{theme_id}", dependencies.authorize(ScopeResearchRead, dependencies.getResearchTheme))
	mux.Handle("GET "+Namespace+"/research/themes/{theme_id}/reasoning-trees", dependencies.authorize(ScopeResearchRead, dependencies.listResearchThemeReasoningTrees))
	mux.Handle("GET "+Namespace+"/research/themes/{theme_id}/reasoning-trees/{anchor_id}", dependencies.authorize(ScopeResearchRead, dependencies.getResearchThemeReasoningTree))
	mux.Handle("GET "+Namespace+"/admin/raw-documents", dependencies.authorize(ScopeAdminRead, dependencies.listAdminRawDocuments))
	mux.Handle("GET "+Namespace+"/admin/events", dependencies.authorize(ScopeAdminRead, dependencies.listAdminEvents))
	return mux
}

func (d Dependencies) retiredEventImport(response http.ResponseWriter, _ *http.Request, _ Principal, requestID string) {
	writeError(
		response,
		requestID,
		http.StatusGone,
		"EVENT_IMPORT_CONTRACT_RETIRED",
		"this import contract has retired; use POST /internal/data/v2/reviewed-event-imports",
	)
}
