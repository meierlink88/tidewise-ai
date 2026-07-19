// Package internalapi owns the versioned Data Service HTTP transport.
package internalapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
	domainimport "github.com/meierlink88/tidewise-ai/backend/services/data/domain/eventimport"
	researchdomainimport "github.com/meierlink88/tidewise-ai/backend/services/data/domain/researchthemeimport"
	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/adminquery"
	eventapp "github.com/meierlink88/tidewise-ai/backend/services/data/usecase/eventimport"
	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/rawimport"
	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/research"
	researchimportapp "github.com/meierlink88/tidewise-ai/backend/services/data/usecase/researchthemeimport"
	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/sourcemetadata"
)

const (
	Namespace           = "/internal/data/v1"
	MaxRequestBodyBytes = 1_048_576

	ScopeResearchRead        = "data.research.read"
	ScopeResearchImport      = "data.research.import"
	ScopeAdminRead           = "data.admin.read"
	ScopeSourceMetadataRead  = "data.source-metadata.read"
	ScopeRawImport           = "data.raw-documents.import"
	ScopeReviewedEventImport = "data.reviewed-events.import"
)

type RawImportService interface {
	Import(context.Context, string, string, rawimport.Batch) (rawimport.Result, error)
	Status(context.Context, string, string) (rawimport.ImportStatus, error)
}

type ReviewedEventService interface {
	Import(context.Context, domainimport.Package) (eventapp.Result, error)
}

type ResearchThemeImportService interface {
	Import(context.Context, string, researchdomainimport.Batch) (researchimportapp.Result, error)
}

type ResearchService interface {
	ListThemes(context.Context, research.ResearchListRequest) (research.ResearchThemePage, error)
	GetTheme(context.Context, string, research.ResearchDetailRequest) (research.ResearchThemeDetail, error)
	ListAnchors(context.Context, research.ResearchListRequest) (research.ResearchAnchorPage, error)
	GetAnchor(context.Context, string, research.ResearchDetailRequest) (research.ResearchAnchorDetail, error)
}

type AdminService interface {
	ListRawDocuments(context.Context, adminquery.RawDocumentListRequest) (adminquery.RawDocumentPage, error)
	ListEvents(context.Context, adminquery.EventListRequest) (adminquery.EventPage, error)
	ListSourceCatalogs(context.Context, adminquery.SourceCatalogListRequest) ([]domain.SourceCatalog, error)
}

type SourceMetadataService interface {
	List(context.Context, sourcemetadata.ListRequest) (sourcemetadata.Page, error)
}

type Dependencies struct {
	Authenticator        *Authenticator
	RawImports           RawImportService
	ReviewedEvents       ReviewedEventService
	ResearchThemeImports ResearchThemeImportService
	Research             ResearchService
	Admin                AdminService
	SourceMetadata       SourceMetadataService
	NewRequestID         func() string
}

type operation func(http.ResponseWriter, *http.Request, Principal, string)

func NewHandler(dependencies Dependencies) http.Handler {
	if dependencies.NewRequestID == nil {
		dependencies.NewRequestID = func() string { return fmt.Sprintf("data-%d", time.Now().UTC().UnixNano()) }
	}
	mux := http.NewServeMux()
	mux.Handle("POST "+Namespace+"/raw-document-imports", dependencies.authorize(ScopeRawImport, dependencies.importRawDocuments))
	mux.Handle("GET "+Namespace+"/raw-document-imports/{idempotency_key}", dependencies.authorize(ScopeRawImport, dependencies.rawImportStatus))
	mux.Handle("POST "+Namespace+"/reviewed-event-imports", dependencies.authorize(ScopeReviewedEventImport, dependencies.importReviewedEvent))
	mux.Handle("POST "+Namespace+"/research-theme-imports", dependencies.authorize(ScopeResearchImport, dependencies.importResearchThemes))
	mux.Handle("GET "+Namespace+"/research/themes", dependencies.authorize(ScopeResearchRead, dependencies.listResearchThemes))
	mux.Handle("GET "+Namespace+"/research/themes/{theme_id}", dependencies.authorize(ScopeResearchRead, dependencies.getResearchTheme))
	mux.Handle("GET "+Namespace+"/research/anchors", dependencies.authorize(ScopeResearchRead, dependencies.listResearchAnchors))
	mux.Handle("GET "+Namespace+"/research/anchors/{anchor_id}", dependencies.authorize(ScopeResearchRead, dependencies.getResearchAnchor))
	mux.Handle("GET "+Namespace+"/admin/raw-documents", dependencies.authorize(ScopeAdminRead, dependencies.listAdminRawDocuments))
	mux.Handle("GET "+Namespace+"/admin/events", dependencies.authorize(ScopeAdminRead, dependencies.listAdminEvents))
	mux.Handle("GET "+Namespace+"/admin/source-catalogs", dependencies.authorize(ScopeAdminRead, dependencies.listAdminSources))
	mux.Handle("GET "+Namespace+"/agent-run/source-metadata", dependencies.authorize(ScopeSourceMetadataRead, dependencies.listSourceMetadata))
	return mux
}
