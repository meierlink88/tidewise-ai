package event

import "context"

// PublicationStore exposes storage mechanics while keeping replay, conflict,
// reference, and write decisions inside Biz.
type PublicationStore interface {
	InEventPublicationTransaction(context.Context, func(PublicationTransaction) error) error
}

type PublicationTransaction interface {
	Lock(context.Context, string) error
	Receipt(context.Context, string, string) (*PublicationReceipt, error)
	ExistingEvidenceIDs(context.Context, []string) ([]string, error)
	InsertAggregate(context.Context, Aggregate) error
	InsertReceipt(context.Context, PublicationReceipt) error
}

type ReferenceError struct {
	Field   string
	Message string
}

func (e *ReferenceError) Error() string { return e.Field + ": " + e.Message }

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

func invalidEvent(message string) error { return &ValidationError{Message: message} }
