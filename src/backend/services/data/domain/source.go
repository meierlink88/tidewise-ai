package domain

import (
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

type RawDocument struct {
	ID               string
	ContractVersion  int
	ArtifactID       string
	SourceRef        string
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
	if d.ContractVersion == 2 {
		if d.ArtifactID == "" {
			return fmt.Errorf("artifact id is required")
		}
		if d.SourceRef == "" {
			return fmt.Errorf("source ref is required")
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
		return nil
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
