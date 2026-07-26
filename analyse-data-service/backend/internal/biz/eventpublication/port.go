package eventpublication

import (
	"context"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"
)

type Store interface {
	InEventPublicationTransaction(context.Context, func(Transaction) error) error
}

type Transaction interface {
	LockEventPublicationIdentities(context.Context, []string) error
	PublicationRawDocument(context.Context, string) (*PublicationRawDocument, error)
	InsertPublicationRawDocument(context.Context, PublicationRawDocument) error
	PublicationEvent(context.Context, string) (*PublicationEvent, error)
	InsertPublicationEvent(context.Context, PublicationEvent) error
	AdvancePublicationEventObservationTimes(context.Context, string, time.Time, time.Time) error
	PublicationEventSource(context.Context, string, string) (*PublicationEventSource, error)
	InsertPublicationEventSource(context.Context, PublicationEventSource) error
	SetPublicationEventPrimarySource(context.Context, string, string) error
	PublicationTag(context.Context, string) (*model.EventTagDef, error)
	PublicationEventTag(context.Context, string, string) (*PublicationEventTag, error)
	InsertPublicationEventTag(context.Context, PublicationEventTag) error
	InsertEventPublicationReceipt(context.Context, EventPublicationReceipt) error
}

type PublicationRawDocument struct {
	ID, ArtifactID, ContentSHA256, SourceRef, SourceName, SourceType, SourceURL, Title string
	PublishedAt                                                                        *time.Time
	CollectedAt                                                                        time.Time
	Language, MIMEType                                                                 string
}

type PublicationEvent struct {
	ID, DedupeKey, Title, FactualSummary string
	OccurredAt                           *time.Time
	FactPayload                          model.FactPayload
	FirstSeenAt, KnowableAt              time.Time
	EventStatus                          model.EventStatus
	FactStatus                           model.FactStatus
	PrimarySourceID                      string
}

type PublicationEventSource struct {
	ID, EventID, RawDocumentID, SourceLevel, EvidenceExcerpt, EvidenceHash string
	EvidenceRelation                                                       model.EvidenceRelation
	SupportsFields                                                         []string
	IsPrimary                                                              bool
}

type PublicationEventTag struct {
	ID, EventID, TagID, AssignSource string
	ReviewStatus                     model.ReviewStatus
	Confidence, AssignmentReason     string
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
	ID, PackageID, CallerSubject, ExtractorExecutionID, ExtractorAgentVersion string
	CollectorExecutions                                                       []PublicationCollectorExecution
	EventIDs, RawDocumentIDs, EventSourceIDs, EventTagMapIDs                  []string
	ReviewMetadata                                                            []PublicationReviewMetadata
	WriteCounts                                                               PublicationWriteCounts
	ImportedAt                                                                time.Time
}
