package seed

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

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
func DryRunExternalIdentifierMappings(path string) (ExternalIdentifierBatchReport, error) {
	m, e := LoadExternalIdentifierMappingFile(path)
	if e != nil {
		return ExternalIdentifierBatchReport{}, e
	}
	return ExternalIdentifierBatchReport{Created: len(m.Mappings)}, nil
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

func mappingFromIdentifier(item domain.EntityExternalIdentifier) ExternalIdentifierMapping {
	return ExternalIdentifierMapping{ID: item.ID, EntityID: item.EntityID, SourceSystem: item.SourceSystem, SourceTaxonomyType: item.SourceTaxonomyType, ExternalCode: item.ExternalCode, ExternalName: item.ExternalName, Status: item.Status}
}
func (m ExternalIdentifierMapping) identifier() domain.EntityExternalIdentifier {
	return domain.EntityExternalIdentifier{ID: m.ID, EntityID: m.EntityID, SourceSystem: m.SourceSystem, SourceTaxonomyType: m.SourceTaxonomyType, ExternalCode: m.ExternalCode, ExternalName: m.ExternalName, Status: m.Status}
}

func (r PostgresRepository) ApplyExternalIdentifierBatch(ctx context.Context, mappings []ExternalIdentifierMapping) (ExternalIdentifierBatchReport, error) {
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
	type plannedMapping struct {
		item   domain.EntityExternalIdentifier
		action WriteAction
	}
	planned := make([]plannedMapping, 0, len(mappings))
	for _, mapping := range mappings {
		item := mapping.identifier()
		identity := externalIdentifierIdentity(item.SourceSystem, item.SourceTaxonomyType, item.ExternalCode)
		if _, err := tx.ExecContext(ctx, externalIdentifierTransactionLockSQL(), identity); err != nil {
			return report, err
		}
		var id string
		if err := tx.QueryRowContext(ctx, externalIdentifierTargetSQL(), item.EntityID).Scan(&id); err != nil {
			return report, fmt.Errorf("external identifier %q requires an active chain_node target", identity)
		}
		var existing storedExternalIdentifier
		err = tx.QueryRowContext(ctx, externalIdentifierSelectSQL(), item.SourceSystem, item.SourceTaxonomyType, item.ExternalCode).Scan(&existing.ID, &existing.EntityID, &existing.ExternalName, &existing.Status)
		if err == nil {
			if existing.ID != item.ID || existing.EntityID != item.EntityID {
				return report, fmt.Errorf("external identifier %q identity conflict", identity)
			}
			if existing.ExternalName == item.ExternalName && existing.Status == item.Status {
				planned = append(planned, plannedMapping{item: item, action: WriteUnchanged})
			} else {
				planned = append(planned, plannedMapping{item: item, action: WriteUpdated})
			}
			continue
		}
		if err != sql.ErrNoRows {
			return report, err
		}
		if err = tx.QueryRowContext(ctx, externalIdentifierSelectByIDSQL(), item.ID).Scan(&existing.ID, &existing.EntityID, &existing.ExternalName, &existing.Status); err == nil {
			return report, fmt.Errorf("external identifier %q deterministic id conflict", identity)
		}
		if err != sql.ErrNoRows {
			return report, err
		}
		planned = append(planned, plannedMapping{item: item, action: WriteCreated})
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
		default:
			return report, fmt.Errorf("unsupported external identifier mapping action %q", plan.action)
		}
	}
	if err := tx.Commit(); err != nil {
		return report, err
	}
	return report, nil
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
