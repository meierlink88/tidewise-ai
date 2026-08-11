package rawdocument

import (
	"context"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
)

const OperationList = "data.v1.listAdminRawDocuments"

func BusinessOperations() []string { return []string{OperationList} }

type Service interface {
	List(context.Context, *ListRequest) (*v1.Response[Page], error)
}

type ListRequest struct {
	Title        string
	SourceRef    string
	IngestStatus string
	Page         string
	PageSize     string
}

type Document struct {
	ID               string  `json:"id"`
	ContractVersion  int     `json:"contract_version"`
	ArtifactID       string  `json:"artifact_id,omitempty"`
	SourceRef        string  `json:"source_ref,omitempty"`
	IngestChannel    string  `json:"ingest_channel"`
	SourceType       string  `json:"source_type"`
	SourceName       string  `json:"source_name"`
	SourceURL        string  `json:"source_url"`
	SourceExternalID string  `json:"source_external_id,omitempty"`
	Title            string  `json:"title"`
	ContentText      string  `json:"content_text"`
	ContentLevel     string  `json:"content_level"`
	RawObjectURI     string  `json:"raw_object_uri"`
	RawMIMEType      string  `json:"raw_mime_type"`
	Language         string  `json:"language"`
	PublishedAt      *string `json:"published_at"`
	CollectedAt      string  `json:"collected_at"`
	IngestStatus     string  `json:"ingest_status"`
	ContentSHA256    string  `json:"content_sha256"`
}

type Page struct {
	Items    []Document `json:"items"`
	Total    int        `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
}
