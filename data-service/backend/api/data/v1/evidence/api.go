package evidence

import (
	"context"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const (
	OperationPublishRawEvidence     = "data.v1.publishRawEvidence"
	OperationGetRawEvidence         = "data.v1.getRawEvidence"
	OperationPublishEvidence        = "data.v1.publishEvidence"
	OperationListEvidenceCategories = "data.v1.listEvidenceCategories"
	OperationListAdminEvidence      = "data.v1.listAdminEvidence"

	ErrorInvalidRequest                      = "INVALID_REQUEST"
	ErrorDataServiceNotReady                 = "DATA_SERVICE_NOT_READY"
	ErrorEvidencePublicationInvalid          = "EVIDENCE_PUBLICATION_INVALID"
	ErrorEvidencePublicationReferenceInvalid = "EVIDENCE_PUBLICATION_REFERENCE_INVALID"
	ErrorEvidencePublicationConflict         = "EVIDENCE_PUBLICATION_CONFLICT"
	ErrorEvidencePublicationTimeout          = "EVIDENCE_PUBLICATION_TIMEOUT"
	ErrorEvidencePublicationFailed           = "EVIDENCE_PUBLICATION_FAILED"
	ErrorRawEvidenceNotFound                 = "RAW_EVIDENCE_NOT_FOUND"
	ErrorRawEvidenceReadTimeout              = "RAW_EVIDENCE_READ_TIMEOUT"
	ErrorRawEvidenceReadFailed               = "RAW_EVIDENCE_READ_FAILED"
	ErrorEvidenceCategoryCatalogFailed       = "EVIDENCE_CATEGORY_CATALOG_FAILED"
	ErrorEvidenceCategoryCatalogTimeout      = "EVIDENCE_CATEGORY_CATALOG_TIMEOUT"
	ErrorEvidenceListFailed                  = "EVIDENCE_LIST_FAILED"
	ErrorEvidenceListTimeout                 = "EVIDENCE_LIST_TIMEOUT"
)

func BusinessOperations() []string {
	return []string{
		OperationPublishRawEvidence, OperationGetRawEvidence, OperationPublishEvidence,
		OperationListEvidenceCategories, OperationListAdminEvidence,
	}
}

type Service interface {
	PublishRawEvidence(context.Context, *RawEvidencePublicationRequest) (*v1.Response[RawEvidencePublicationResult], error)
	GetRawEvidence(context.Context, *GetRawEvidenceRequest) (*v1.Response[RawEvidenceReadResult], error)
	PublishEvidence(context.Context, *EvidencePublicationRequest) (*v1.Response[EvidencePublicationResult], error)
	ListEvidenceCategories(context.Context) (*v1.Response[EvidenceCategoryCatalog], error)
	ListEvidence(context.Context, *ListRequest) (*v1.Response[Page], error)
}

type ListRequest struct {
	Title         string
	Summary       string
	CategoryID    string
	SourceName    string
	SourceLevel   string
	IsSplit       string
	PublishedFrom string
	PublishedTo   string
	CollectedFrom string
	CollectedTo   string
	Page          string
	PageSize      string
}

type ListItem struct {
	ID               string             `json:"id"`
	RawEvidenceID    string             `json:"raw_evidence_id"`
	Title            *string            `json:"title"`
	Summary          string             `json:"summary"`
	Semantic         EvidenceSemantic   `json:"semantic"`
	Categories       []EvidenceCategory `json:"categories"`
	SourceName       string             `json:"source_name"`
	SourceLevel      string             `json:"source_level"`
	SourceURL        string             `json:"source_url"`
	IsOriginal       bool               `json:"is_original"`
	QuotedSourceName *string            `json:"quoted_source_name"`
	Keywords         []string           `json:"keywords"`
	IsSplit          bool               `json:"is_split"`
	PublishedAt      *string            `json:"published_at"`
	CollectedAt      string             `json:"collected_at"`
}

type Page struct {
	Items    []ListItem `json:"items"`
	Total    int        `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
}

type RawEvidencePublicationRequest struct {
	RawEvidence RawEvidence `json:"raw_evidence"`
}

type RawEvidence struct {
	PublicationKey   string     `json:"publication_key"`
	SourceID         string     `json:"source_id"`
	SourceName       string     `json:"source_name"`
	SourceLevel      string     `json:"source_level"`
	SourceURL        string     `json:"source_url"`
	IsOriginal       bool       `json:"is_original"`
	QuotedSourceID   *string    `json:"quoted_source_id"`
	QuotedSourceName *string    `json:"quoted_source_name"`
	Title            *string    `json:"title"`
	RawText          string     `json:"raw_text"`
	PublishedAt      *time.Time `json:"published_at"`
	CollectedAt      time.Time  `json:"collected_at"`
	Keywords         []string   `json:"keywords"`
	CategoryIDs      []string   `json:"category_ids,omitempty"`
}

type GetRawEvidenceRequest struct {
	ID string `json:"-"`
}

type EvidenceCategory struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type EvidenceCategoryCatalog struct {
	Categories []EvidenceCategory `json:"categories"`
}

type RawEvidenceRead struct {
	ID               string             `json:"id"`
	SourceID         string             `json:"source_id"`
	SourceName       string             `json:"source_name"`
	SourceLevel      string             `json:"source_level"`
	SourceURL        string             `json:"source_url"`
	IsOriginal       bool               `json:"is_original"`
	QuotedSourceID   *string            `json:"quoted_source_id"`
	QuotedSourceName *string            `json:"quoted_source_name"`
	Title            *string            `json:"title"`
	RawText          string             `json:"raw_text"`
	PublishedAt      *time.Time         `json:"published_at"`
	CollectedAt      time.Time          `json:"collected_at"`
	Keywords         []string           `json:"keywords"`
	Categories       []EvidenceCategory `json:"categories"`
}

type RawEvidenceReadResult struct {
	RawEvidence RawEvidenceRead `json:"raw_evidence"`
}

type EvidencePublicationRequest struct {
	RawEvidenceID string           `json:"raw_evidence_id"`
	Evidences     []AtomicEvidence `json:"evidences"`
}

type AtomicEvidence struct {
	Summary  string           `json:"summary"`
	Semantic EvidenceSemantic `json:"semantic"`
}

type EvidenceSemantic struct {
	Who   *string `json:"who"`
	What  string  `json:"what"`
	When  *string `json:"when"`
	Where *string `json:"where"`
	Why   *string `json:"why"`
	How   *string `json:"how"`
}

type RawEvidencePublicationResult struct {
	ID string `json:"id"`
}

type EvidencePublicationResult struct {
	RawEvidenceID string   `json:"raw_evidence_id"`
	IDs           []string `json:"ids"`
}
