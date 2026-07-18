package seed

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

const frozenFirstBatchExternalIdentifierManifestSHA256 = "05539cd9f940cfcc5ec67cde5c395563b672ffa52d56090da0a83bd0d5997658"

type ExternalIdentifierMapping struct {
	ID                 string        `json:"id"`
	EntityID           string        `json:"entity_id"`
	SourceSystem       string        `json:"source_system"`
	SourceTaxonomyType string        `json:"source_taxonomy_type"`
	ExternalCode       string        `json:"external_code"`
	ExternalName       string        `json:"external_name"`
	Status             domain.Status `json:"status"`
}
type ExternalIdentifierBatchReport struct {
	Created   int `json:"created"`
	Updated   int `json:"updated"`
	Unchanged int `json:"unchanged"`
}
type ExternalIdentifierMappingPreflightReport struct {
	ManifestRows  int `json:"manifest_rows"`
	ActiveTargets int `json:"active_targets"`
	ExistingRows  int `json:"existing_rows"`
}
type ExternalIdentifierMappingManifest struct {
	Mappings []ExternalIdentifierMapping `json:"mappings"`
}
type plannedExternalIdentifierMapping struct {
	item   domain.EntityExternalIdentifier
	action WriteAction
}

func LoadExternalIdentifierMappingFile(path string) (ExternalIdentifierMappingManifest, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return ExternalIdentifierMappingManifest{}, e
	}
	var m ExternalIdentifierMappingManifest
	if e = json.Unmarshal(b, &m); e != nil {
		return m, e
	}
	if len(m.Mappings) == 0 {
		return m, fmt.Errorf("external identifier mapping manifest is empty")
	}
	if m.Mappings, e = normalizeAndValidateExternalIdentifierMappings(m.Mappings); e != nil {
		return m, e
	}
	return m, nil
}
func ValidateExternalIdentifierMappingFile(path string) (ExternalIdentifierBatchReport, error) {
	m, e := LoadExternalIdentifierMappingFile(path)
	if e != nil {
		return ExternalIdentifierBatchReport{}, e
	}
	return ExternalIdentifierBatchReport{Created: len(m.Mappings)}, nil
}

func ValidateFrozenFirstBatchExternalIdentifierManifest(path string, mappings []ExternalIdentifierMapping) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if fmt.Sprintf("%x", sha256.Sum256(content)) != frozenFirstBatchExternalIdentifierManifestSHA256 {
		return fmt.Errorf("external identifier mapping manifest hash does not match approved first batch")
	}
	if len(mappings) != 1169 {
		return fmt.Errorf("external identifier mapping manifest rows = %d, want 1169", len(mappings))
	}
	providers, entities := map[string]int{}, map[string]map[string]struct{}{}
	for _, mapping := range mappings {
		providers[mapping.SourceSystem]++
		if entities[mapping.EntityID] == nil {
			entities[mapping.EntityID] = map[string]struct{}{}
		}
		entities[mapping.EntityID][mapping.SourceSystem] = struct{}{}
	}
	if providers["eastmoney"] != 818 || providers["ths"] != 351 {
		return fmt.Errorf("external identifier provider counts = eastmoney %d, ths %d; want 818/351", providers["eastmoney"], providers["ths"])
	}
	dualSource, multiTaxonomy := 0, 0
	for _, systems := range entities {
		if len(systems) == 2 {
			dualSource++
		}
	}
	byCode := map[string]int{}
	for _, mapping := range mappings {
		byCode[mapping.SourceSystem+"\x00"+mapping.ExternalCode]++
	}
	for _, count := range byCode {
		if count == 2 {
			multiTaxonomy++
		}
	}
	if dualSource != 241 || multiTaxonomy != 13 {
		return fmt.Errorf("external identifier dual-source/multi-taxonomy = %d/%d, want 241/13", dualSource, multiTaxonomy)
	}
	return nil
}

func (r PostgresRepository) PreflightExternalIdentifierMappings(ctx context.Context, mappings []ExternalIdentifierMapping) (ExternalIdentifierMappingPreflightReport, error) {
	mappings, err := normalizeAndValidateExternalIdentifierMappings(mappings)
	if err != nil {
		return ExternalIdentifierMappingPreflightReport{}, err
	}
	report := ExternalIdentifierMappingPreflightReport{ManifestRows: len(mappings)}
	for _, mapping := range mappings {
		item := mapping.identifier()
		var targetID string
		if err := r.root.QueryRowContext(ctx, externalIdentifierTargetSQL(), item.EntityID).Scan(&targetID); err != nil {
			return report, fmt.Errorf("external identifier %q requires an active chain_node target", externalIdentifierIdentity(item.SourceSystem, item.SourceTaxonomyType, item.ExternalCode))
		}
		report.ActiveTargets++
		var existing string
		err := r.root.QueryRowContext(ctx, externalIdentifierSelectSQL(), item.SourceSystem, item.SourceTaxonomyType, item.ExternalCode).Scan(&existing, new(string), new(string), new(domain.Status))
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

func mappingFromIdentifier(item domain.EntityExternalIdentifier) ExternalIdentifierMapping {
	return ExternalIdentifierMapping{ID: item.ID, EntityID: item.EntityID, SourceSystem: item.SourceSystem, SourceTaxonomyType: item.SourceTaxonomyType, ExternalCode: item.ExternalCode, ExternalName: item.ExternalName, Status: item.Status}
}
func (m ExternalIdentifierMapping) identifier() domain.EntityExternalIdentifier {
	return domain.EntityExternalIdentifier{ID: m.ID, EntityID: m.EntityID, SourceSystem: m.SourceSystem, SourceTaxonomyType: m.SourceTaxonomyType, ExternalCode: m.ExternalCode, ExternalName: m.ExternalName, Status: m.Status}
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
		item := mapping.identifier()
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

func normalizeAndValidateExternalIdentifierMappings(mappings []ExternalIdentifierMapping) ([]ExternalIdentifierMapping, error) {
	if len(mappings) == 0 {
		return nil, fmt.Errorf("external identifier mapping batch is empty")
	}
	seenIdentity := make(map[string]struct{}, len(mappings))
	seenID := make(map[string]struct{}, len(mappings))
	normalized := make([]ExternalIdentifierMapping, 0, len(mappings))
	for _, mapping := range mappings {
		item := normalizeExternalIdentifier(mapping.identifier())
		if err := validateFirstBatchExternalIdentifier(item); err != nil {
			return nil, err
		}
		identity := externalIdentifierIdentity(item.SourceSystem, item.SourceTaxonomyType, item.ExternalCode)
		if _, exists := seenIdentity[identity]; exists {
			return nil, fmt.Errorf("duplicate external identifier identity %q in mapping manifest", identity)
		}
		if _, exists := seenID[item.ID]; exists {
			return nil, fmt.Errorf("duplicate external identifier id %q in mapping manifest", item.ID)
		}
		seenIdentity[identity], seenID[item.ID] = struct{}{}, struct{}{}
		normalized = append(normalized, mappingFromIdentifier(item))
	}
	sort.Slice(normalized, func(i, j int) bool {
		return externalIdentifierIdentity(normalized[i].SourceSystem, normalized[i].SourceTaxonomyType, normalized[i].ExternalCode) < externalIdentifierIdentity(normalized[j].SourceSystem, normalized[j].SourceTaxonomyType, normalized[j].ExternalCode)
	})
	return normalized, nil
}
