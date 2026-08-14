package evidence

import "context"

type Store interface {
	InTransaction(context.Context, func(Transaction) error) error
}

type Transaction interface {
	LockIdentities(context.Context, []string) error
	RawEvidence(context.Context, string) (*StoredRawEvidence, error)
	CategoriesByIDs(context.Context, []CategoryID) ([]Category, error)
	InsertRawEvidence(context.Context, StoredRawEvidence) error
	InsertRawEvidenceCategoryLinks(context.Context, string, []CategoryID) error
	EvidencesByRawEvidence(context.Context, string) ([]StoredEvidence, error)
	EvidencesByIDs(context.Context, []string) ([]StoredEvidence, error)
	InsertEvidence(context.Context, StoredEvidence) error
}
