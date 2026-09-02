package report

import "context"

// PublicationStore exposes the atomic publication boundary. Replay and
// conflict decisions remain owned by the Report use case.
type PublicationStore interface {
	InPublicationTransaction(context.Context, func(PublicationTransaction) error) error
}

type PublicationTransaction interface {
	Lock(context.Context, string) error
	ReportByPublisherID(context.Context, string) (*Record, error)
	ExistingEvidenceIDs(context.Context, []string) ([]string, error)
	InsertReport(context.Context, Record) error
	InsertEvidenceLinks(context.Context, []EvidenceLink) error
}
