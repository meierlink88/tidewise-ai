package data

import (
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/biz"
)

type rawDocumentPageWire struct {
	Items    []rawDocumentWire `json:"items"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

func (w rawDocumentPageWire) toBiz() (biz.RawDocumentPage, error) {
	if w.Total < 0 || w.Page < 1 || w.PageSize < 1 {
		return biz.RawDocumentPage{}, &Error{Kind: ErrorKindDecode}
	}
	items := make([]biz.RawDocument, 0, len(w.Items))
	for _, item := range w.Items {
		mapped, err := item.toBiz()
		if err != nil {
			return biz.RawDocumentPage{}, err
		}
		items = append(items, mapped)
	}
	return biz.RawDocumentPage{Items: items, Total: w.Total, Page: w.Page, PageSize: w.PageSize}, nil
}

type rawDocumentWire struct {
	ID               string           `json:"id"`
	ContractVersion  int              `json:"contract_version"`
	ArtifactID       string           `json:"artifact_id,omitempty"`
	SourceRef        string           `json:"source_ref,omitempty"`
	IngestChannel    string           `json:"ingest_channel"`
	SourceType       string           `json:"source_type"`
	SourceName       string           `json:"source_name"`
	SourceURL        string           `json:"source_url"`
	SourceExternalID string           `json:"source_external_id,omitempty"`
	Title            string           `json:"title"`
	ContentText      string           `json:"content_text"`
	ContentLevel     string           `json:"content_level"`
	RawObjectURI     string           `json:"raw_object_uri"`
	RawMIMEType      string           `json:"raw_mime_type"`
	Language         string           `json:"language"`
	PublishedAt      *time.Time       `json:"published_at,omitempty"`
	CollectedAt      time.Time        `json:"collected_at"`
	IngestStatus     biz.IngestStatus `json:"ingest_status"`
	ContentSHA256    string           `json:"content_sha256"`
}

func (w rawDocumentWire) toBiz() (biz.RawDocument, error) {
	if strings.TrimSpace(w.ID) == "" || w.ContractVersion < 1 || !validIngestStatus(w.IngestStatus) {
		return biz.RawDocument{}, &Error{Kind: ErrorKindDecode}
	}
	return biz.RawDocument{
		ID: w.ID, ContractVersion: w.ContractVersion, ArtifactID: w.ArtifactID,
		SourceRef: w.SourceRef, IngestChannel: w.IngestChannel, SourceType: w.SourceType,
		SourceName: w.SourceName, SourceURL: w.SourceURL, SourceExternalID: w.SourceExternalID,
		Title: w.Title, ContentText: w.ContentText, ContentLevel: w.ContentLevel,
		RawObjectURI: w.RawObjectURI, RawMIMEType: w.RawMIMEType, Language: w.Language,
		PublishedAt: w.PublishedAt, CollectedAt: w.CollectedAt, IngestStatus: w.IngestStatus,
		ContentSHA256: w.ContentSHA256,
	}, nil
}

func validIngestStatus(status biz.IngestStatus) bool {
	switch status {
	case biz.IngestStatusCollected, biz.IngestStatusDuplicate, biz.IngestStatusFailed, biz.IngestStatusPendingExtract:
		return true
	default:
		return false
	}
}

type eventPageWire struct {
	Items    []eventWire `json:"items"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

func (w eventPageWire) toBiz() (biz.EventPage, error) {
	if w.Total < 0 || w.Page < 1 || w.PageSize < 1 {
		return biz.EventPage{}, &Error{Kind: ErrorKindDecode}
	}
	items := make([]biz.Event, 0, len(w.Items))
	for _, item := range w.Items {
		mapped, err := item.toBiz()
		if err != nil {
			return biz.EventPage{}, err
		}
		items = append(items, mapped)
	}
	return biz.EventPage{Items: items, Total: w.Total, Page: w.Page, PageSize: w.PageSize}, nil
}

type eventWire struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Summary     string          `json:"summary"`
	EventTime   *time.Time      `json:"event_time,omitempty"`
	FirstSeenAt time.Time       `json:"first_seen_at"`
	KnowableAt  *time.Time      `json:"knowable_at,omitempty"`
	EventStatus biz.EventStatus `json:"event_status"`
	FactStatus  biz.FactStatus  `json:"fact_status"`
	DedupeKey   string          `json:"dedupe_key"`
}

func (w eventWire) toBiz() (biz.Event, error) {
	if strings.TrimSpace(w.ID) == "" || !validEventStatus(w.EventStatus) || !validFactStatus(w.FactStatus) {
		return biz.Event{}, &Error{Kind: ErrorKindDecode}
	}
	return biz.Event{
		ID: w.ID, Title: w.Title, Summary: w.Summary, EventTime: w.EventTime,
		FirstSeenAt: w.FirstSeenAt, KnowableAt: w.KnowableAt, EventStatus: w.EventStatus,
		FactStatus: w.FactStatus, DedupeKey: w.DedupeKey,
	}, nil
}

func validEventStatus(status biz.EventStatus) bool {
	return status == biz.EventStatusCandidate || status == biz.EventStatusConfirmed || status == biz.EventStatusRejected
}

func validFactStatus(status biz.FactStatus) bool {
	return status == biz.FactStatusUnverified || status == biz.FactStatusVerified || status == biz.FactStatusDisputed
}
