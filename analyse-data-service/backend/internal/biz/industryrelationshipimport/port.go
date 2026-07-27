package industryrelationshipimport

import (
	"context"
	"time"
)

type Store interface {
	InIndustryRelationshipImportTransaction(context.Context, func(Transaction) error) error
}

type Transaction interface {
	LockIndustryRelationshipPackage(context.Context, string) error
	IndustryRelationshipImportReceipt(context.Context, string) (*Receipt, error)
	PreflightIndustryRelationshipPackage(context.Context, Package) error
	InsertIndustryRelationshipPackage(context.Context, Package) error
	VerifyIndustryRelationshipPackage(context.Context, Package) error
	InsertIndustryRelationshipImportReceipt(context.Context, Receipt) error
}

type Receipt struct {
	ID                 string
	PackageSHA256      string
	ManifestSHA256     string
	RelationSpecSHA256 string
	ApprovalBasis      string
	PackageCounts      Counts
	CallerSubject      string
	ImportedAt         time.Time
}
