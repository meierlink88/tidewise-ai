package domain

import (
	"fmt"
	"time"
)

type SourceCatalogStatus string

const (
	SourceCatalogStatusActive   SourceCatalogStatus = "active"
	SourceCatalogStatusInactive SourceCatalogStatus = "inactive"
	SourceCatalogStatusDisabled SourceCatalogStatus = "disabled"
)

type SourceCatalog struct {
	ID              string
	IngestChannel   string
	ProviderKey     string
	ConnectorKey    string
	ParserKey       string
	SourceType      string
	SourceName      string
	SourceURL       string
	SourceLevel     string
	TopicHint       string
	RouteTemplate   string
	CodeStyle       string
	AuthRequired    bool
	AuthType        string
	CredentialRef   string
	SourceConfig    map[string]any
	RateLimitPolicy map[string]any
	UsagePolicy     string
	Status          SourceCatalogStatus
}

type IngestStatus string

const (
	IngestStatusCollected      IngestStatus = "collected"
	IngestStatusDuplicate      IngestStatus = "duplicate"
	IngestStatusFailed         IngestStatus = "failed"
	IngestStatusPendingExtract IngestStatus = "pending_extract"
)

type RawDocument struct {
	ID               string
	SourceID         string
	IngestChannel    string
	SourceType       string
	SourceName       string
	SourceURL        string
	SourceExternalID string
	Title            string
	ContentText      string
	ContentLevel     string
	RawObjectURI     string
	RawMIMEType      string
	Language         string
	PublishedAt      *time.Time
	CollectedAt      time.Time
	ContentHash      string
	IngestStatus     IngestStatus
}

func (d RawDocument) Validate() error {
	if d.ID == "" {
		return fmt.Errorf("raw document id is required")
	}
	if d.SourceID == "" {
		return fmt.Errorf("source id is required")
	}
	if d.IngestChannel == "" {
		return fmt.Errorf("ingest channel is required")
	}
	if d.SourceType == "" {
		return fmt.Errorf("source type is required")
	}
	if d.SourceName == "" {
		return fmt.Errorf("source name is required")
	}
	if d.Title == "" {
		return fmt.Errorf("title is required")
	}
	if d.ContentHash == "" {
		return fmt.Errorf("content hash is required")
	}
	if d.CollectedAt.IsZero() {
		return fmt.Errorf("collected at is required")
	}
	if !validStatus(d.IngestStatus, IngestStatusCollected, IngestStatusDuplicate, IngestStatusFailed, IngestStatusPendingExtract) {
		return fmt.Errorf("unsupported ingest status %q", d.IngestStatus)
	}
	return nil
}
