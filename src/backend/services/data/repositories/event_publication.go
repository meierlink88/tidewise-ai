package repositories

import (
	"context"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

type EventPublicationStore interface {
	InEventPublicationTransaction(context.Context, func(EventPublicationTransaction) error) error
}

type EventPublicationTransaction interface {
	LockEventPublicationIdentities(context.Context, []string) error
	PublicationRawDocument(context.Context, string) (*PublicationRawDocument, error)
	InsertPublicationRawDocument(context.Context, PublicationRawDocument) error
	PublicationEvent(context.Context, string) (*PublicationEvent, error)
	InsertPublicationEvent(context.Context, PublicationEvent) error
	AdvancePublicationEventObservationTimes(context.Context, string, time.Time, time.Time) error
	PublicationEventSource(context.Context, string, string) (*PublicationEventSource, error)
	InsertPublicationEventSource(context.Context, PublicationEventSource) error
	SetPublicationEventPrimarySource(context.Context, string, string) error
	PublicationTag(context.Context, string) (*domain.EventTagDef, error)
	PublicationEventTag(context.Context, string, string) (*PublicationEventTag, error)
	InsertPublicationEventTag(context.Context, PublicationEventTag) error
	InsertEventPublicationReceipt(context.Context, EventPublicationReceipt) error
}

type PublicationRawDocument struct {
	ID            string
	ArtifactID    string
	ContentSHA256 string
	SourceRef     string
	SourceName    string
	SourceType    string
	SourceURL     string
	Title         string
	PublishedAt   *time.Time
	CollectedAt   time.Time
	Language      string
	MIMEType      string
}

type PublicationEvent struct {
	ID              string
	DedupeKey       string
	Title           string
	FactualSummary  string
	OccurredAt      *time.Time
	FactPayload     domain.FactPayload
	FirstSeenAt     time.Time
	KnowableAt      time.Time
	EventStatus     domain.EventStatus
	FactStatus      domain.FactStatus
	PrimarySourceID string
}

type PublicationEventSource struct {
	ID               string
	EventID          string
	RawDocumentID    string
	SourceLevel      string
	EvidenceExcerpt  string
	EvidenceHash     string
	EvidenceRelation domain.EvidenceRelation
	SupportsFields   []string
	IsPrimary        bool
}

type PublicationEventTag struct {
	ID               string
	EventID          string
	TagID            string
	AssignSource     string
	ReviewStatus     domain.ReviewStatus
	Confidence       string
	AssignmentReason string
}

type PublicationCollectorExecution struct {
	ArtifactID           string `json:"artifact_id"`
	CollectorExecutionID string `json:"collector_execution_id"`
}

type PublicationReviewMetadata struct {
	DedupeKey     string   `json:"dedupe_key"`
	ReviewID      string   `json:"review_id"`
	EvidenceGrade string   `json:"evidence_grade"`
	Reasons       []string `json:"reasons"`
}

type PublicationWriteCounts struct {
	EventsCreated       int `json:"events_created"`
	EventsReused        int `json:"events_reused"`
	RawDocumentsCreated int `json:"raw_documents_created"`
	RawDocumentsReused  int `json:"raw_documents_reused"`
	EventSourcesCreated int `json:"event_sources_created"`
	EventSourcesReused  int `json:"event_sources_reused"`
	EventTagsCreated    int `json:"event_tags_created"`
	EventTagsReused     int `json:"event_tags_reused"`
}

type EventPublicationReceipt struct {
	ID                    string
	PackageID             string
	CallerSubject         string
	ExtractorExecutionID  string
	ExtractorAgentVersion string
	CollectorExecutions   []PublicationCollectorExecution
	EventIDs              []string
	RawDocumentIDs        []string
	EventSourceIDs        []string
	EventTagMapIDs        []string
	ReviewMetadata        []PublicationReviewMetadata
	WriteCounts           PublicationWriteCounts
	ImportedAt            time.Time
}
