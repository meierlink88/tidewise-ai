package entityseed

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"
)

type ExternalIdentifierMappingPreflightReport struct {
	ManifestRows  int `json:"manifest_rows"`
	ActiveTargets int `json:"active_targets"`
	ExistingRows  int `json:"existing_rows"`
}
type plannedExternalIdentifierMapping struct {
	item   model.EntityExternalIdentifier
	action WriteAction
}

func (r PostgresRepository) PreflightExternalIdentifierMappings(ctx context.Context, mappings []ExternalIdentifierMapping) (ExternalIdentifierMappingPreflightReport, error) {
	mappings, err := normalizeAndValidateExternalIdentifierMappings(mappings)
	if err != nil {
		return ExternalIdentifierMappingPreflightReport{}, err
	}
	report := ExternalIdentifierMappingPreflightReport{ManifestRows: len(mappings)}
	for _, mapping := range mappings {
		item := externalIdentifierFromMapping(mapping)
		var targetID string
		if err := r.root.QueryRowContext(ctx, externalIdentifierTargetSQL(), item.EntityID).Scan(&targetID); err != nil {
			return report, fmt.Errorf("external identifier %q requires an active chain_node target", externalIdentifierIdentity(item.SourceSystem, item.SourceTaxonomyType, item.ExternalCode))
		}
		report.ActiveTargets++
		var existing string
		err := r.root.QueryRowContext(ctx, externalIdentifierSelectSQL(), item.SourceSystem, item.SourceTaxonomyType, item.ExternalCode).Scan(&existing, new(string), new(string), new(model.Status))
		if err == nil {
			report.ExistingRows++
			continue
		}
		if err != sql.ErrNoRows {
			return report, err
		}
	}
	return report, nil
}

func (r PostgresRepository) DryRunExternalIdentifierBatch(ctx context.Context, mappings []ExternalIdentifierMapping) (ExternalIdentifierBatchReport, error) {
	var report ExternalIdentifierBatchReport
	var err error
	if mappings, err = normalizeAndValidateExternalIdentifierMappings(mappings); err != nil {
		return report, err
	}
	tx, err := r.root.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return report, err
	}
	defer tx.Rollback()
	planned, err := planExternalIdentifierMappings(ctx, tx, mappings, true)
	if err != nil {
		return report, err
	}
	for _, plan := range planned {
		switch plan.action {
		case WriteCreated:
			report.Created++
		case WriteUpdated:
			report.Updated++
		case WriteUnchanged:
			report.Unchanged++
		}
	}
	return report, nil
}

func (r PostgresRepository) ApplyExternalIdentifierBatch(ctx context.Context, mappings []ExternalIdentifierMapping) (ExternalIdentifierBatchReport, error) {
	return r.applyExternalIdentifierBatch(ctx, mappings, false)
}

func (r PostgresRepository) ApplyFrozenFirstBatchExternalIdentifiers(ctx context.Context, mappings []ExternalIdentifierMapping) (ExternalIdentifierBatchReport, error) {
	return r.applyExternalIdentifierBatch(ctx, mappings, true)
}

func (r PostgresRepository) applyExternalIdentifierBatch(ctx context.Context, mappings []ExternalIdentifierMapping, requireEmptyTable bool) (ExternalIdentifierBatchReport, error) {
	var report ExternalIdentifierBatchReport
	var err error
	if mappings, err = normalizeAndValidateExternalIdentifierMappings(mappings); err != nil {
		return report, err
	}
	tx, err := r.root.BeginTx(ctx, nil)
	if err != nil {
		return report, err
	}
	defer tx.Rollback()
	if requireEmptyTable {
		var existingRows int
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM entity_external_identifiers").Scan(&existingRows); err != nil {
			return report, err
		}
		if existingRows != 0 {
			return report, fmt.Errorf("frozen first-batch mapping write requires zero existing external identifiers, got %d", existingRows)
		}
	}
	planned, err := planExternalIdentifierMappings(ctx, tx, mappings, false)
	if err != nil {
		return report, err
	}
	for _, plan := range planned {
		item := plan.item
		switch plan.action {
		case WriteUnchanged:
			report.Unchanged++
		case WriteUpdated:
			if _, err := tx.ExecContext(ctx, "UPDATE entity_external_identifiers SET external_name=$1,status=$2,updated_at=now() WHERE id=$3::uuid", item.ExternalName, item.Status, item.ID); err != nil {
				return report, err
			}
			report.Updated++
		case WriteCreated:
			var inserted string
			if err := tx.QueryRowContext(ctx, externalIdentifierInsertSQL(), item.ID, item.EntityID, item.SourceSystem, item.SourceTaxonomyType, item.ExternalCode, item.ExternalName, item.Status).Scan(&inserted); err != nil {
				return report, fmt.Errorf("insert %q: %w", externalIdentifierIdentity(item.SourceSystem, item.SourceTaxonomyType, item.ExternalCode), err)
			}
			report.Created++
		}
	}
	if err := verifyExternalIdentifierBatchPostWrite(ctx, tx, planned, report); err != nil {
		return ExternalIdentifierBatchReport{}, err
	}
	if err := tx.Commit(); err != nil {
		return report, err
	}
	return report, nil
}

func planExternalIdentifierMappings(ctx context.Context, tx *sql.Tx, mappings []ExternalIdentifierMapping, readOnly bool) ([]plannedExternalIdentifierMapping, error) {
	planned := make([]plannedExternalIdentifierMapping, 0, len(mappings))
	for _, mapping := range mappings {
		item := externalIdentifierFromMapping(mapping)
		identity := externalIdentifierIdentity(item.SourceSystem, item.SourceTaxonomyType, item.ExternalCode)
		if !readOnly {
			if _, err := tx.ExecContext(ctx, externalIdentifierTransactionLockSQL(), identity); err != nil {
				return nil, err
			}
		}
		var id string
		targetSQL := externalIdentifierTargetSQL()
		if readOnly {
			targetSQL = externalIdentifierTargetSnapshotSQL()
		}
		if err := tx.QueryRowContext(ctx, targetSQL, item.EntityID).Scan(&id); err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("external identifier %q requires an active chain_node target", identity)
			}
			return nil, fmt.Errorf("query active chain_node target for external identifier %q: %w", identity, err)
		}
		var existing storedExternalIdentifier
		selectSQL := externalIdentifierSelectSQL()
		if readOnly {
			selectSQL = externalIdentifierSnapshotSQL()
		}
		err := tx.QueryRowContext(ctx, selectSQL, item.SourceSystem, item.SourceTaxonomyType, item.ExternalCode).Scan(&existing.ID, &existing.EntityID, &existing.ExternalName, &existing.Status)
		if err == nil {
			if existing.ID != item.ID || existing.EntityID != item.EntityID {
				return nil, fmt.Errorf("external identifier %q identity conflict", identity)
			}
			if existing.ExternalName == item.ExternalName && existing.Status == item.Status {
				planned = append(planned, plannedExternalIdentifierMapping{item: item, action: WriteUnchanged})
			} else {
				planned = append(planned, plannedExternalIdentifierMapping{item: item, action: WriteUpdated})
			}
			continue
		}
		if err != sql.ErrNoRows {
			return nil, err
		}
		byIDSQL := externalIdentifierSelectByIDSQL()
		if readOnly {
			byIDSQL = externalIdentifierSnapshotByIDSQL()
		}
		if err = tx.QueryRowContext(ctx, byIDSQL, item.ID).Scan(&existing.ID, &existing.EntityID, new(string), new(string), new(string), &existing.ExternalName, &existing.Status); err == nil {
			return nil, fmt.Errorf("external identifier %q deterministic id conflict", identity)
		}
		if err != sql.ErrNoRows {
			return nil, err
		}
		planned = append(planned, plannedExternalIdentifierMapping{item: item, action: WriteCreated})
	}
	return planned, nil
}

func verifyExternalIdentifierBatchPostWrite(ctx context.Context, tx *sql.Tx, planned []plannedExternalIdentifierMapping, report ExternalIdentifierBatchReport) error {
	if report.Created+report.Updated+report.Unchanged != len(planned) {
		return fmt.Errorf("external identifier mapping report count mismatch")
	}
	for _, plan := range planned {
		var got storedExternalIdentifier
		var sourceSystem, taxonomy, code string
		if err := tx.QueryRowContext(ctx, externalIdentifierSelectByIDSQL(), plan.item.ID).Scan(&got.ID, &got.EntityID, &sourceSystem, &taxonomy, &code, &got.ExternalName, &got.Status); err != nil {
			return fmt.Errorf("verify external identifier %q: %w", plan.item.ID, err)
		}
		if got.ID != plan.item.ID || got.EntityID != plan.item.EntityID || sourceSystem != plan.item.SourceSystem || taxonomy != plan.item.SourceTaxonomyType || code != plan.item.ExternalCode || got.ExternalName != plan.item.ExternalName || got.Status != plan.item.Status {
			return fmt.Errorf("verify external identifier %q did not match manifest", plan.item.ID)
		}
	}
	return nil
}
