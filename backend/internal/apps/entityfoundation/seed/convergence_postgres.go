package seed

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	repoids "github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

type SectorReferenceRule struct {
	Table      string
	Column     string
	SectorOnly bool
	Archival   bool
}

type SectorReferenceRegistry struct {
	rules map[string]SectorReferenceRule
}

func NewSectorReferenceRegistry() SectorReferenceRegistry {
	rules := []SectorReferenceRule{
		{Table: "entity_edges", Column: "from_entity_id"},
		{Table: "entity_edges", Column: "to_entity_id"},
		{Table: "sector_source_mappings", Column: "sector_entity_id", SectorOnly: true},
		{Table: "sector_profiles", Column: "parent_sector_entity_id", SectorOnly: true},
		{Table: "entity_convergences", Column: "legacy_entity_id", Archival: true},
		{Table: "entity_convergences", Column: "target_entity_id", Archival: true},
		{Table: "entity_convergence_reference_moves", Column: "from_entity_id", Archival: true},
		{Table: "entity_convergence_reference_moves", Column: "to_entity_id", Archival: true},
		{Table: "entity_convergence_alias_moves", Column: "from_entity_id", Archival: true},
		{Table: "entity_convergence_alias_moves", Column: "to_entity_id", Archival: true},
	}
	registry := SectorReferenceRegistry{rules: map[string]SectorReferenceRule{}}
	for _, rule := range rules {
		registry.rules[rule.Table+"."+rule.Column] = rule
	}
	return registry
}

func (r SectorReferenceRegistry) Rule(table, column string) (SectorReferenceRule, bool) {
	rule, ok := r.rules[table+"."+column]
	return rule, ok
}

func (r PostgresRepository) ApplySectorConvergence(ctx context.Context, entities Manifest, manifest SectorConvergenceManifest, mode SectorConvergenceMode) (SectorConvergenceReport, error) {
	if r.root == nil {
		return SectorConvergenceReport{}, fmt.Errorf("sector convergence requires a transactional PostgreSQL repository")
	}
	tx, err := r.root.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return SectorConvergenceReport{}, fmt.Errorf("begin sector convergence: %w", err)
	}
	defer tx.Rollback()
	txRepo := PostgresRepository{db: tx}

	currentVersion, currentChecksum, currentReview, err := currentConvergenceManifest(ctx, tx)
	if err != nil {
		return SectorConvergenceReport{}, err
	}
	if currentVersion == manifest.ManifestVersion {
		if currentChecksum != manifest.ManifestChecksum {
			return SectorConvergenceReport{}, fmt.Errorf("same manifest version has different payload")
		}
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM entity_convergences WHERE manifest_version = $1`, manifest.ManifestVersion).Scan(&count); err != nil {
			return SectorConvergenceReport{}, err
		}
		return SectorConvergenceReport{ManifestVersion: manifest.ManifestVersion, AuditUnchanged: count}, nil
	}
	if mode == SectorConvergenceModeInitial && currentVersion != 0 {
		return SectorConvergenceReport{}, fmt.Errorf("initial convergence already applied")
	}
	if mode == SectorConvergenceModeCorrection && (manifest.PreviousManifestVersion == nil || *manifest.PreviousManifestVersion != currentVersion || manifest.ManifestVersion <= currentVersion) {
		return SectorConvergenceReport{}, fmt.Errorf("invalid correction manifest version")
	}
	if mode == SectorConvergenceModeCorrection && currentReview == manifest.ReviewSourceURL+manifest.ReviewedAt.String() {
		return SectorConvergenceReport{}, fmt.Errorf("correction requires a new human Review")
	}
	if mode == SectorConvergenceModeInitial {
		if err := applyManifestInTransaction(ctx, txRepo, entities); err != nil {
			return SectorConvergenceReport{}, err
		}
	}
	if err := preflightSectorReferences(ctx, tx, manifest, NewSectorReferenceRegistry()); err != nil {
		return SectorConvergenceReport{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO entity_convergence_manifests (manifest_version, previous_manifest_version, manifest_checksum, review_source_url, reviewed_at, applied_mode) VALUES ($1,$2,$3,$4,$5,$6)`, manifest.ManifestVersion, manifest.PreviousManifestVersion, manifest.ManifestChecksum, manifest.ReviewSourceURL, manifest.ReviewedAt, mode); err != nil {
		return SectorConvergenceReport{}, fmt.Errorf("insert convergence manifest: %w", err)
	}
	report := SectorConvergenceReport{ManifestVersion: manifest.ManifestVersion}
	for _, item := range manifest.Convergences {
		legacyID := entitySeedUUID(item.LegacyEntityKey)
		var targetID any
		if item.TargetEntityKey != "" {
			targetID = entitySeedUUID(item.TargetEntityKey)
			var targetType domain.EntityType
			if err := tx.QueryRowContext(ctx, `SELECT entity_type FROM entity_nodes WHERE id=$1 FOR UPDATE`, targetID).Scan(&targetType); err != nil || targetType != item.TargetEntityType {
				return SectorConvergenceReport{}, fmt.Errorf("invalid convergence target %q", item.TargetEntityKey)
			}
		}
		var status domain.Status
		if err := tx.QueryRowContext(ctx, `SELECT status FROM entity_nodes WHERE id=$1 AND entity_type='sector' FOR UPDATE`, legacyID).Scan(&status); err != nil {
			return SectorConvergenceReport{}, fmt.Errorf("lock legacy sector %q: %w", item.LegacyEntityKey, err)
		}
		if mode == SectorConvergenceModeCorrection && status != domain.StatusInactive {
			return SectorConvergenceReport{}, fmt.Errorf("correction drift for legacy sector %q", item.LegacyEntityKey)
		}
		auditID := repoids.NormalizeUUID("entity_convergence", fmt.Sprintf("%s|%d", item.LegacyEntityKey, manifest.ManifestVersion))
		if _, err := tx.ExecContext(ctx, `INSERT INTO entity_convergences (id, legacy_entity_id, target_entity_id, manifest_version, action, legacy_taxonomy, mutation_provenance) VALUES ($1,$2,$3,$4,$5,$6,jsonb_build_object('manifest_checksum',$7,'mode',$8))`, auditID, legacyID, targetID, manifest.ManifestVersion, item.Action, item.LegacyTaxonomy, manifest.ManifestChecksum, mode); err != nil {
			return SectorConvergenceReport{}, fmt.Errorf("insert convergence audit %q: %w", item.LegacyEntityKey, err)
		}
		if mode == SectorConvergenceModeCorrection {
			previousID := repoids.NormalizeUUID("entity_convergence", fmt.Sprintf("%s|%d", item.LegacyEntityKey, *manifest.PreviousManifestVersion))
			if err := applyRecordedCorrectionMoves(ctx, tx, previousID, auditID, item.TargetEntityType, targetID); err != nil {
				return SectorConvergenceReport{}, fmt.Errorf("correction drift for %q: %w", item.LegacyEntityKey, err)
			}
		}
		if item.TargetEntityType == domain.EntityTypeSector {
			mapping := SectorSourceMapping{
				SectorEntityKey: item.TargetEntityKey, SourceSystem: "ths", SourceTaxonomyType: item.LegacyTaxonomy,
				SourceSectorName: item.LegacyName, SourceMarketScope: "cn_a_share", SourceURL: manifest.ReviewSourceURL,
				MappingStatus: "merged", ReviewNote: item.Reason,
			}
			mappingResult, err := txRepo.UpsertSectorSourceMapping(ctx, mapping)
			if err != nil {
				return SectorConvergenceReport{}, fmt.Errorf("preserve legacy source mapping %q: %w", item.LegacyEntityKey, err)
			}
			if mappingResult.Action != WriteUnchanged {
				report.MappingsChanged++
				mappingID := sectorSourceMappingSeedUUID(sectorSourceMappingIdentity(normalizeSectorSourceMapping(mapping)))
				if err := insertReferenceMove(ctx, tx, auditID, "sector_source_mappings", "sector_entity_id", mappingID, legacyID, targetID.(string), "legacy_mapping_projection"); err != nil {
					return SectorConvergenceReport{}, err
				}
			}
			moves, err := moveSectorReferences(ctx, tx, auditID, legacyID, targetID.(string), item.LegacyName)
			if err != nil {
				return SectorConvergenceReport{}, err
			}
			report.ReferencesMoved += moves.references
			report.MappingsChanged += moves.mappings
			report.AliasesChanged += moves.aliases
		} else {
			moves, err := deactivateLegacyEdges(ctx, tx, auditID, legacyID)
			if err != nil {
				return SectorConvergenceReport{}, err
			}
			report.ReferencesMoved += moves
		}
		if _, err := tx.ExecContext(ctx, `UPDATE entity_nodes SET status='inactive', updated_at=NOW() WHERE id=$1`, legacyID); err != nil {
			return SectorConvergenceReport{}, err
		}
		report.RetiredLegacy++
		report.AuditCreated++
	}
	if err := resolveConvergenceEdgeConflicts(ctx, tx, manifest.ManifestVersion); err != nil {
		return SectorConvergenceReport{}, err
	}
	if err := tx.Commit(); err != nil {
		return SectorConvergenceReport{}, fmt.Errorf("commit sector convergence: %w", err)
	}
	return report, nil
}

func currentConvergenceManifest(ctx context.Context, tx *sql.Tx) (int64, string, string, error) {
	var version int64
	var checksum, reviewURL string
	var reviewedAt time.Time
	err := tx.QueryRowContext(ctx, `SELECT manifest_version, manifest_checksum, review_source_url, reviewed_at FROM entity_convergence_manifests ORDER BY manifest_version DESC LIMIT 1`).Scan(&version, &checksum, &reviewURL, &reviewedAt)
	if err == sql.ErrNoRows {
		return 0, "", "", nil
	}
	if err != nil {
		return 0, "", "", fmt.Errorf("read current convergence manifest: %w", err)
	}
	return version, checksum, reviewURL + reviewedAt.String(), nil
}

func applyManifestInTransaction(ctx context.Context, repo PostgresRepository, manifest Manifest) error {
	for _, entity := range manifest.Entities {
		if _, err := repo.UpsertEntity(ctx, entity); err != nil {
			return err
		}
	}
	for _, entity := range manifest.Entities {
		if len(entity.Profile) > 0 {
			if _, err := repo.UpsertProfile(ctx, Profile{EntityKey: entity.Key, EntityType: entity.EntityType, Data: entity.Profile}); err != nil {
				return err
			}
		}
	}
	for _, profile := range manifest.Profiles {
		if _, err := repo.UpsertProfile(ctx, profile); err != nil {
			return err
		}
	}
	for _, mapping := range manifest.SectorSourceMappings {
		if _, err := repo.UpsertSectorSourceMapping(ctx, mapping); err != nil {
			return err
		}
	}
	for _, relationship := range manifest.Relationships {
		if _, err := repo.UpsertRelationship(ctx, relationship); err != nil {
			return err
		}
	}
	return nil
}

var safeSQLIdentifier = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func preflightSectorReferences(ctx context.Context, tx *sql.Tx, manifest SectorConvergenceManifest, registry SectorReferenceRegistry) error {
	rows, err := tx.QueryContext(ctx, `SELECT tc.table_name, kcu.column_name FROM information_schema.table_constraints tc JOIN information_schema.key_column_usage kcu ON tc.constraint_name=kcu.constraint_name AND tc.constraint_schema=kcu.constraint_schema JOIN information_schema.constraint_column_usage ccu ON ccu.constraint_name=tc.constraint_name AND ccu.constraint_schema=tc.constraint_schema WHERE tc.constraint_type='FOREIGN KEY' AND ccu.table_name='entity_nodes' AND ccu.column_name='id' AND tc.table_schema=current_schema()`)
	if err != nil {
		return fmt.Errorf("inspect entity reference catalog: %w", err)
	}
	defer rows.Close()
	type ref struct{ table, column string }
	var refs []ref
	for rows.Next() {
		var item ref
		if err := rows.Scan(&item.table, &item.column); err != nil {
			return err
		}
		refs = append(refs, item)
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].table+refs[i].column < refs[j].table+refs[j].column })
	for _, item := range manifest.Convergences {
		legacyID := entitySeedUUID(item.LegacyEntityKey)
		for _, reference := range refs {
			if reference.column == "entity_id" && regexp.MustCompile(`_profiles$`).MatchString(reference.table) {
				continue
			}
			if !safeSQLIdentifier.MatchString(reference.table) || !safeSQLIdentifier.MatchString(reference.column) {
				return fmt.Errorf("unsafe entity reference catalog entry")
			}
			query := fmt.Sprintf(`SELECT EXISTS (SELECT 1 FROM "%s" WHERE "%s"=$1)`, reference.table, reference.column)
			var exists bool
			if err := tx.QueryRowContext(ctx, query, legacyID).Scan(&exists); err != nil {
				return err
			}
			if rule, known := registry.Rule(reference.table, reference.column); known {
				if rule.Archival {
					continue
				}
				if exists && item.TargetEntityType != domain.EntityTypeSector && rule.SectorOnly {
					return fmt.Errorf("sector-only reference %s.%s cannot target %s", reference.table, reference.column, item.TargetEntityType)
				}
				continue
			}
			if exists {
				return fmt.Errorf("unknown FK reference %s.%s for %s", reference.table, reference.column, item.LegacyEntityKey)
			}
		}
	}
	return nil
}

type sectorMoveCounts struct{ references, mappings, aliases int }

func moveSectorReferences(ctx context.Context, tx *sql.Tx, convergenceID, legacyID, targetID, alias string) (sectorMoveCounts, error) {
	var counts sectorMoveCounts
	for _, column := range []string{"from_entity_id", "to_entity_id"} {
		rows, err := tx.QueryContext(ctx, fmt.Sprintf(`UPDATE entity_edges SET %s=$1, updated_at=NOW() WHERE %s=$2 RETURNING id`, column, column), targetID, legacyID)
		if err != nil {
			return counts, fmt.Errorf("move entity edge reference: %w", err)
		}
		for rows.Next() {
			var rowID string
			if err := rows.Scan(&rowID); err != nil {
				rows.Close()
				return counts, err
			}
			if err := insertReferenceMove(ctx, tx, convergenceID, "entity_edges", column, rowID, legacyID, targetID, "redirect"); err != nil {
				rows.Close()
				return counts, err
			}
			counts.references++
		}
		if err := rows.Close(); err != nil {
			return counts, err
		}
	}
	rows, err := tx.QueryContext(ctx, `UPDATE sector_source_mappings SET sector_entity_id=$1, updated_at=NOW() WHERE sector_entity_id=$2 RETURNING id`, targetID, legacyID)
	if err != nil {
		return counts, err
	}
	for rows.Next() {
		var rowID string
		if err := rows.Scan(&rowID); err != nil {
			rows.Close()
			return counts, err
		}
		if err := insertReferenceMove(ctx, tx, convergenceID, "sector_source_mappings", "sector_entity_id", rowID, legacyID, targetID, "redirect"); err != nil {
			rows.Close()
			return counts, err
		}
		counts.mappings++
	}
	if err := rows.Close(); err != nil {
		return counts, err
	}
	rows, err = tx.QueryContext(ctx, `UPDATE sector_profiles SET parent_sector_entity_id=$1 WHERE parent_sector_entity_id=$2 RETURNING entity_id`, targetID, legacyID)
	if err != nil {
		return counts, err
	}
	for rows.Next() {
		var rowID string
		if err := rows.Scan(&rowID); err != nil {
			rows.Close()
			return counts, err
		}
		if err := insertReferenceMove(ctx, tx, convergenceID, "sector_profiles", "parent_sector_entity_id", rowID, legacyID, targetID, "redirect"); err != nil {
			rows.Close()
			return counts, err
		}
		counts.references++
	}
	if err := rows.Close(); err != nil {
		return counts, err
	}
	var aliasEntityID string
	err = tx.QueryRowContext(ctx, `UPDATE entity_nodes SET aliases=array_append(aliases,$1), updated_at=NOW() WHERE id=$2 AND NOT ($1=ANY(aliases)) RETURNING id`, alias, targetID).Scan(&aliasEntityID)
	if err == sql.ErrNoRows {
		return counts, nil
	}
	if err != nil {
		return counts, err
	}
	aliasMoveID := repoids.NormalizeUUID("entity_convergence_alias_move", convergenceID+"|"+alias)
	if _, err := tx.ExecContext(ctx, `INSERT INTO entity_convergence_alias_moves (id,convergence_id,alias,from_entity_id,to_entity_id,mutation_provenance) VALUES ($1,$2,$3,$4,$5,jsonb_build_object('operation','append'))`, aliasMoveID, convergenceID, alias, legacyID, targetID); err != nil {
		return counts, err
	}
	counts.aliases = 1
	return counts, nil
}

func deactivateLegacyEdges(ctx context.Context, tx *sql.Tx, convergenceID, legacyID string) (int, error) {
	rows, err := tx.QueryContext(ctx, `UPDATE entity_edges SET status='inactive',updated_at=NOW() WHERE status='active' AND (from_entity_id=$1 OR to_entity_id=$1) RETURNING id`, legacyID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var rowID string
		if err := rows.Scan(&rowID); err != nil {
			return count, err
		}
		if err := insertReferenceMove(ctx, tx, convergenceID, "entity_edges", "status", rowID, legacyID, nil, "deactivate_incompatible_reference"); err != nil {
			return count, err
		}
		count++
	}
	return count, rows.Err()
}

func insertReferenceMove(ctx context.Context, tx *sql.Tx, convergenceID, table, column, rowID, fromID string, toID any, operation string) error {
	id := repoids.NormalizeUUID("entity_convergence_reference_move", convergenceID+"|"+table+"|"+column+"|"+rowID)
	_, err := tx.ExecContext(ctx, `INSERT INTO entity_convergence_reference_moves (id,convergence_id,reference_table,reference_column,reference_row_id,from_entity_id,to_entity_id,mutation_provenance) VALUES ($1,$2,$3,$4,$5,$6,$7,jsonb_build_object('operation',$8))`, id, convergenceID, table, column, rowID, fromID, toID, operation)
	return err
}

func applyRecordedCorrectionMoves(ctx context.Context, tx *sql.Tx, previousID, currentID string, targetType domain.EntityType, targetID any) error {
	rows, err := tx.QueryContext(ctx, `SELECT reference_table,reference_column,reference_row_id,from_entity_id,to_entity_id FROM entity_convergence_reference_moves WHERE convergence_id=$1 ORDER BY reference_table,reference_column,reference_row_id`, previousID)
	if err != nil {
		return err
	}
	defer rows.Close()
	type move struct {
		table, column, rowID, fromID string
		toID                         sql.NullString
	}
	var moves []move
	for rows.Next() {
		var m move
		if err := rows.Scan(&m.table, &m.column, &m.rowID, &m.fromID, &m.toID); err != nil {
			return err
		}
		moves = append(moves, m)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, m := range moves {
		if !m.toID.Valid {
			if m.table == "entity_edges" && m.column == "status" {
				var status string
				if err := tx.QueryRowContext(ctx, `SELECT status FROM entity_edges WHERE id=$1`, m.rowID).Scan(&status); err != nil {
					return err
				}
				if status != "inactive" {
					return fmt.Errorf("recorded deactivated edge %s drifted", m.rowID)
				}
				if err := insertReferenceMove(ctx, tx, currentID, m.table, m.column, m.rowID, m.fromID, nil, "forward_correction_verified"); err != nil {
					return err
				}
			}
			continue
		}
		var result sql.Result
		switch m.table + "." + m.column {
		case "entity_edges.from_entity_id", "entity_edges.to_entity_id":
			if targetType == domain.EntityTypeSector {
				result, err = tx.ExecContext(ctx, fmt.Sprintf(`UPDATE entity_edges SET %s=$1,updated_at=NOW() WHERE id=$2 AND %s=$3`, m.column, m.column), targetID, m.rowID, m.toID.String)
			} else {
				result, err = tx.ExecContext(ctx, `UPDATE entity_edges SET status='inactive',updated_at=NOW() WHERE id=$1 AND status='active'`, m.rowID)
			}
		case "sector_source_mappings.sector_entity_id":
			if targetType == domain.EntityTypeSector {
				result, err = tx.ExecContext(ctx, `UPDATE sector_source_mappings SET sector_entity_id=$1,updated_at=NOW() WHERE id=$2 AND sector_entity_id=$3`, targetID, m.rowID, m.toID.String)
			} else {
				result, err = tx.ExecContext(ctx, `UPDATE sector_source_mappings SET mapping_status='rejected',updated_at=NOW() WHERE id=$1 AND sector_entity_id=$2`, m.rowID, m.toID.String)
			}
		case "sector_profiles.parent_sector_entity_id":
			if targetType != domain.EntityTypeSector {
				return fmt.Errorf("sector-specific parent reference cannot target %s", targetType)
			}
			result, err = tx.ExecContext(ctx, `UPDATE sector_profiles SET parent_sector_entity_id=$1 WHERE entity_id=$2 AND parent_sector_entity_id=$3`, targetID, m.rowID, m.toID.String)
		default:
			continue
		}
		if err != nil {
			return err
		}
		affected, _ := result.RowsAffected()
		if affected != 1 {
			return fmt.Errorf("recorded reference %s.%s row %s drifted", m.table, m.column, m.rowID)
		}
		if err := insertReferenceMove(ctx, tx, currentID, m.table, m.column, m.rowID, m.toID.String, targetID, "forward_correction"); err != nil {
			return err
		}
	}
	aliasRows, err := tx.QueryContext(ctx, `SELECT alias,to_entity_id FROM entity_convergence_alias_moves WHERE convergence_id=$1 ORDER BY alias`, previousID)
	if err != nil {
		return err
	}
	defer aliasRows.Close()
	for aliasRows.Next() {
		var alias, oldTarget string
		if err := aliasRows.Scan(&alias, &oldTarget); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `UPDATE entity_nodes SET aliases=array_remove(aliases,$1),updated_at=NOW() WHERE id=$2 AND $1=ANY(aliases)`, alias, oldTarget)
		if err != nil {
			return err
		}
		affected, _ := result.RowsAffected()
		if affected != 1 {
			return fmt.Errorf("recorded alias %q drifted", alias)
		}
	}
	return aliasRows.Err()
}

func resolveConvergenceEdgeConflicts(ctx context.Context, tx *sql.Tx, version int64) error {
	_, err := tx.ExecContext(ctx, `WITH touched AS (
    SELECT DISTINCT e.from_entity_id,e.to_entity_id,e.relation_type
    FROM entity_edges e
    JOIN entity_convergence_reference_moves m ON m.reference_row_id=e.id
    JOIN entity_convergences c ON c.id=m.convergence_id
    WHERE c.manifest_version=$1
), ranked AS (
    SELECT e.id,row_number() OVER (PARTITION BY e.from_entity_id,e.to_entity_id,e.relation_type ORDER BY e.id) AS rn
    FROM entity_edges e JOIN touched t USING (from_entity_id,to_entity_id,relation_type)
    WHERE e.status='active'
)
UPDATE entity_edges e SET status='inactive',updated_at=NOW() FROM ranked r WHERE e.id=r.id AND r.rn>1`, version)
	return err
}
