package event

import (
	"context"
	"time"
)

type TransactionStore interface {
	InTransaction(context.Context, func(Transaction) error) error
}

type Transaction interface {
	LockIdentities(context.Context, []string) error
	StoredEventEvidenceRecord(context.Context, string) (*StoredEventEvidenceRecord, error)
	InsertStoredEventEvidenceRecord(context.Context, StoredEventEvidenceRecord) error
	StoredEvent(context.Context, string) (*StoredEvent, error)
	InsertStoredEvent(context.Context, StoredEvent) error
	AdvanceStoredEventObservationTimes(context.Context, string, time.Time, time.Time) error
	StoredEventEvidenceLink(context.Context, string, string) (*StoredEventEvidenceLink, error)
	InsertStoredEventEvidenceLink(context.Context, StoredEventEvidenceLink) error
	PublicationTag(context.Context, string) (*EventTag, error)
	StoredEventTagAssignment(context.Context, string, string) (*StoredEventTagAssignment, error)
	InsertStoredEventTagAssignment(context.Context, StoredEventTagAssignment) error
	InsertEventPublicationReceipt(context.Context, EventPublicationReceipt) error
}

type StoredEventEvidenceRecord struct {
	ID, ArtifactID, ContentSHA256, SourceRef, SourceName, SourceType, SourceURL, Title string
	PublishedAt                                                                        *time.Time
	CollectedAt                                                                        time.Time
	Language, MIMEType                                                                 string
}

type StoredEvent struct {
	ID, DedupeKey, Title, FactualSummary string
	OccurredAt                           *time.Time
	FactPayload                          FactPayload
	FirstSeenAt, KnowableAt              time.Time
	EventStatus                          EventStatus
	FactStatus                           FactStatus
}

type StoredEventEvidenceLink struct {
	ID, EventID, RawDocumentID, SourceLevel, EvidenceStatement, EvidenceHash string
	EvidenceRelation                                                         EvidenceRelation
	SupportsFields                                                           []string
}

type StoredEventTagAssignment struct {
	ID, EventID, TagID, AssignSource string
	ReviewStatus                     ReviewStatus
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
