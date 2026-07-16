package repositories

import (
	"context"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type EventImportStore interface {
	InTransaction(context.Context, func(EventImportTransaction) error) error
}

type EventImportTransaction interface {
	LockReceipt(context.Context, string) (*EventImportReceipt, error)
	Source(context.Context, string) (domain.SourceCatalog, error)
	UpsertRawDocument(context.Context, domain.RawDocument) (string, error)
	UpsertEvent(context.Context, domain.Event) (string, error)
	AddEventSource(context.Context, domain.EventSource) (string, error)
	Tag(context.Context, string, string, string) (domain.EventTagDef, error)
	AssignEventTag(context.Context, domain.EventTagMap) (string, error)
	InsertReceipt(context.Context, EventImportReceipt) error
}

type EventImportReceipt struct {
	ID             string
	IdempotencyKey string
	PackageID      string
	ReviewID       string
	ReviewDecision string
	PayloadHash    string
	EventID        string
	RawDocumentIDs []string
	EventSourceIDs []string
	EventTagMapIDs []string
	ReviewMetadata map[string]any
	ImportedAt     time.Time
}
