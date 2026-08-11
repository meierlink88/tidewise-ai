package rawdocument

import (
	"context"
	"fmt"
	"time"
)

type IngestStatus string

const (
	IngestStatusCollected      IngestStatus = "collected"
	IngestStatusDuplicate      IngestStatus = "duplicate"
	IngestStatusFailed         IngestStatus = "failed"
	IngestStatusPendingExtract IngestStatus = "pending_extract"
)

type Document struct {
	ID, ArtifactID, SourceRef, IngestChannel, SourceType, SourceName, SourceURL             string
	SourceExternalID, Title, ContentText, ContentLevel, RawObjectURI, RawMIMEType, Language string
	ContractVersion                                                                         int
	PublishedAt                                                                             *time.Time
	CollectedAt                                                                             time.Time
	ContentHash                                                                             string
	IngestStatus                                                                            IngestStatus
}

func (d Document) Validate() error {
	if d.ID == "" {
		return fmt.Errorf("raw document id is required")
	}
	if d.ContractVersion == 2 && (d.ArtifactID == "" || d.SourceRef == "") {
		return fmt.Errorf("artifact id and source ref are required for contract v2")
	}
	if d.ContractVersion != 2 && d.IngestChannel == "" {
		return fmt.Errorf("ingest channel is required")
	}
	if d.SourceType == "" || d.SourceName == "" || d.Title == "" || d.ContentHash == "" || d.CollectedAt.IsZero() {
		return fmt.Errorf("raw document source, title, content hash, and collected time are required")
	}
	if d.ContractVersion != 2 && !validIngestStatus(d.IngestStatus) {
		return fmt.Errorf("unsupported ingest status %q", d.IngestStatus)
	}
	return nil
}

func validIngestStatus(value IngestStatus) bool {
	return value == IngestStatusCollected || value == IngestStatusDuplicate || value == IngestStatusFailed || value == IngestStatusPendingExtract
}

type ListFilter struct {
	Title, SourceRef string
	IngestStatus     IngestStatus
	Page, PageSize   int
}
type StorePage struct {
	Items                 []Document
	Total, Page, PageSize int
}
type Page = StorePage

type Store interface {
	List(context.Context, ListFilter) (StorePage, error)
}

type UseCase struct{ store Store }

func NewUseCase(store Store) (*UseCase, error) {
	if store == nil {
		return nil, fmt.Errorf("RawDocument store is required")
	}
	return &UseCase{store: store}, nil
}

func (u *UseCase) List(ctx context.Context, filter ListFilter) (Page, error) {
	return u.store.List(ctx, filter)
}
