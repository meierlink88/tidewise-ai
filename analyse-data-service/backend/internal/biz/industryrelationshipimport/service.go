package industryrelationshipimport

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/identity"
)

var (
	ErrPackageConflict = errors.New("relationship package conflicts with persisted data")
	ErrCallerConflict  = errors.New("relationship package was imported by another caller subject")
)

type Result struct {
	ReceiptID      string    `json:"receipt_id,omitempty"`
	PackageSHA256  string    `json:"package_sha256"`
	ManifestSHA256 string    `json:"manifest_sha256"`
	PackageCounts  Counts    `json:"package_counts"`
	ImportedAt     time.Time `json:"imported_at,omitempty"`
	DryRun         bool      `json:"dry_run"`
	Unchanged      bool      `json:"unchanged"`
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{store: store, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) Preflight(ctx context.Context, callerSubject string, pkg Package) (Result, error) {
	if err := s.validateRequest(callerSubject, pkg); err != nil {
		return Result{}, err
	}
	var result Result
	err := s.store.InIndustryRelationshipImportTransaction(ctx, func(tx Transaction) error {
		receipt, err := tx.IndustryRelationshipImportReceipt(ctx, pkg.Manifest.PackageSHA256)
		if err != nil {
			return fmt.Errorf("load Industry relationship receipt: %w", err)
		}
		if receipt != nil {
			if err := validateReplay(*receipt, strings.TrimSpace(callerSubject), pkg); err != nil {
				return err
			}
			if err := tx.VerifyIndustryRelationshipPackage(ctx, pkg); err != nil {
				return fmt.Errorf("verify Industry relationship replay: %w", err)
			}
			result = resultFromReceipt(*receipt, true, true)
			return nil
		}
		if err := tx.PreflightIndustryRelationshipPackage(ctx, pkg); err != nil {
			return fmt.Errorf("preflight Industry relationship package: %w", err)
		}
		result = Result{
			PackageSHA256:  pkg.Manifest.PackageSHA256,
			ManifestSHA256: pkg.ManifestSHA256,
			PackageCounts:  pkg.Counts(),
			DryRun:         true,
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func (s *Service) Import(ctx context.Context, callerSubject string, pkg Package) (Result, error) {
	if err := s.validateRequest(callerSubject, pkg); err != nil {
		return Result{}, err
	}
	callerSubject = strings.TrimSpace(callerSubject)
	var result Result
	err := s.store.InIndustryRelationshipImportTransaction(ctx, func(tx Transaction) error {
		if err := tx.LockIndustryRelationshipPackage(ctx, pkg.Manifest.PackageSHA256); err != nil {
			return fmt.Errorf("lock Industry relationship package: %w", err)
		}
		receipt, err := tx.IndustryRelationshipImportReceipt(ctx, pkg.Manifest.PackageSHA256)
		if err != nil {
			return fmt.Errorf("load Industry relationship receipt: %w", err)
		}
		if receipt != nil {
			if err := validateReplay(*receipt, callerSubject, pkg); err != nil {
				return err
			}
			if err := tx.VerifyIndustryRelationshipPackage(ctx, pkg); err != nil {
				return fmt.Errorf("verify Industry relationship replay: %w", err)
			}
			result = resultFromReceipt(*receipt, true, false)
			return nil
		}
		if err := tx.PreflightIndustryRelationshipPackage(ctx, pkg); err != nil {
			return fmt.Errorf("preflight Industry relationship package: %w", err)
		}
		if err := tx.InsertIndustryRelationshipPackage(ctx, pkg); err != nil {
			return fmt.Errorf("insert Industry relationship package: %w", err)
		}
		if err := tx.VerifyIndustryRelationshipPackage(ctx, pkg); err != nil {
			return fmt.Errorf("verify Industry relationship package: %w", err)
		}
		receipt = &Receipt{
			ID: identity.NormalizeUUID(
				"industry_relationship_import_receipt",
				pkg.Manifest.PackageSHA256,
			),
			PackageSHA256:      pkg.Manifest.PackageSHA256,
			ManifestSHA256:     pkg.ManifestSHA256,
			RelationSpecSHA256: pkg.Manifest.RelationSpec.SHA256,
			ApprovalBasis:      ApprovalBasis,
			PackageCounts:      pkg.Counts(),
			CallerSubject:      callerSubject,
			ImportedAt:         s.now().UTC().Truncate(time.Microsecond),
		}
		if err := tx.InsertIndustryRelationshipImportReceipt(ctx, *receipt); err != nil {
			return fmt.Errorf("insert Industry relationship receipt: %w", err)
		}
		result = resultFromReceipt(*receipt, false, false)
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func (s *Service) validateRequest(callerSubject string, pkg Package) error {
	if s == nil || s.store == nil {
		return errors.New("Industry relationship import store is required")
	}
	callerSubject = strings.TrimSpace(callerSubject)
	if callerSubject == "" || len(callerSubject) > 200 {
		return errors.New("caller subject must contain 1..200 characters")
	}
	return pkg.Validate()
}

func validateReplay(receipt Receipt, callerSubject string, pkg Package) error {
	if receipt.CallerSubject != callerSubject {
		return ErrCallerConflict
	}
	if receipt.PackageSHA256 != pkg.Manifest.PackageSHA256 ||
		receipt.ManifestSHA256 != pkg.ManifestSHA256 ||
		receipt.RelationSpecSHA256 != pkg.Manifest.RelationSpec.SHA256 ||
		receipt.ApprovalBasis != ApprovalBasis ||
		!countsEqual(receipt.PackageCounts, pkg.Counts()) {
		return ErrPackageConflict
	}
	return nil
}

func resultFromReceipt(receipt Receipt, unchanged, dryRun bool) Result {
	return Result{
		ReceiptID:      receipt.ID,
		PackageSHA256:  receipt.PackageSHA256,
		ManifestSHA256: receipt.ManifestSHA256,
		PackageCounts:  receipt.PackageCounts,
		ImportedAt:     receipt.ImportedAt.UTC(),
		DryRun:         dryRun,
		Unchanged:      unchanged,
	}
}
