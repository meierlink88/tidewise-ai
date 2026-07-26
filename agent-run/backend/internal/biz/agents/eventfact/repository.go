package eventfact

import (
	"context"
	"time"
)

type Repository interface {
	DispatchPendingSignals(context.Context, string, time.Time) (int, error)
	EnqueueWork(context.Context, []string, string, time.Time) (WorkItem, bool, error)
	NextUnplannedWork(context.Context) (WorkItem, bool, error)
	InitializeArtifactUnits(context.Context, WorkItem, []ArtifactSummary, time.Time) error
	RejectUnplannedWork(context.Context, WorkItem, string, time.Time) error
	ClaimNextWork(context.Context, ExtractionSnapshot, time.Time) (ExecutionAttempt, bool, error)
	SetAwaitingTagCatalog(context.Context, ExecutionAttempt, Result, string, time.Time) error
	RetryExtraction(context.Context, ExecutionAttempt, Result, string, time.Time) error
	SetExecutionCatalog(context.Context, string, string, string, time.Time) error
	CompleteExtraction(context.Context, ExecutionAttempt, Result, []JournalEntry, time.Time) error
	CompleteWithoutPublication(context.Context, ExecutionAttempt, Result, WorkStatus, time.Time) error
	ListDeliverableJournals(context.Context, time.Time) ([]JournalEntry, error)
	MarkJournalSending(context.Context, JournalEntry, time.Time) (bool, error)
	MarkJournalRetry(context.Context, JournalEntry, string, string, time.Time) error
	MarkJournalBlocked(context.Context, JournalEntry, string, string, time.Time) error
	AcknowledgeJournal(context.Context, JournalEntry, string, []CanonicalEvent, time.Time) error
}

type ArtifactReader interface {
	Read(context.Context, []string) ([]Artifact, error)
}

type CanonicalReader interface {
	FindCanonicalEvents(context.Context, []string) ([]CanonicalEvent, error)
}

type DataClient interface {
	ActiveEventTags(context.Context) (TagCatalog, error)
	PublishReviewedEvents(context.Context, []byte) (string, error)
}
