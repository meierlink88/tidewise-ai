package seed

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
	repoids "github.com/meierlink88/tidewise-ai/backend/services/data/repositories"
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
		return SectorConvergenceReport{ManifestVersion: manifest.ManifestVersion, BlockedReferences: 1}, err
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
		statement, args := buildConvergenceAuditInsert(auditID, legacyID, targetID, manifest, item, mode)
		if _, err := tx.ExecContext(ctx, statement, args...); err != nil {
			return SectorConvergenceReport{}, fmt.Errorf("insert convergence audit %q: %w", item.LegacyEntityKey, err)
		}
		if mode == SectorConvergenceModeCorrection {
			previousID := repoids.NormalizeUUID("entity_convergence", fmt.Sprintf("%s|%d", item.LegacyEntityKey, *manifest.PreviousManifestVersion))
			moves, err := applyRecordedCorrectionMoves(ctx, tx, previousID, auditID, item.TargetEntityType, targetID, item.TargetEntityKey)
			if err != nil {
				return SectorConvergenceReport{}, fmt.Errorf("correction drift for %q: %w", item.LegacyEntityKey, err)
			}
			report.ReferencesMoved += moves.references
			report.MappingsChanged += moves.mappings
			report.AliasesChanged += moves.aliases
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
		} else if item.TargetEntityType == domain.EntityTypeIndex {
			moves, err := redirectLegacyEdgesToIndex(ctx, tx, auditID, legacyID, item.LegacyEntityKey, targetID.(string), item.TargetEntityKey)
			if err != nil {
				return SectorConvergenceReport{ManifestVersion: manifest.ManifestVersion, BlockedReferences: 1}, err
			}
			report.ReferencesMoved += moves
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

func buildConvergenceAuditInsert(auditID, legacyID string, targetID any, manifest SectorConvergenceManifest, item SectorConvergence, mode SectorConvergenceMode) (string, []any) {
	var targetType any
	if item.TargetEntityType != "" {
		targetType = item.TargetEntityType
	}
	statement := `INSERT INTO entity_convergences (id, legacy_entity_id, target_entity_id, target_entity_type, manifest_version, action, legacy_taxonomy, reason, mutation_provenance) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,` + convergenceAuditMutationExpression(9, 10) + `)`
	return statement, []any{auditID, legacyID, targetID, targetType, manifest.ManifestVersion, item.Action, item.LegacyTaxonomy, item.Reason, manifest.ManifestChecksum, mode}
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
	err = tx.QueryRowContext(ctx, `UPDATE entity_nodes SET aliases=ARRAY(SELECT DISTINCT value FROM unnest(aliases || ARRAY[$1::text]) AS value ORDER BY value), updated_at=NOW() WHERE id=$2 AND NOT ($1=ANY(aliases)) RETURNING id`, alias, targetID).Scan(&aliasEntityID)
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

func redirectLegacyEdgesToIndex(ctx context.Context, tx *sql.Tx, convergenceID, legacyID, legacyKey, targetID, targetKey string) (int, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT e.id, fn.entity_key, fn.entity_type, tn.entity_key, tn.entity_type,
       e.relation_type, e.evidence_note, e.source_name, e.source_url, e.verified_at, e.status
FROM entity_edges e
JOIN entity_nodes fn ON fn.id=e.from_entity_id
JOIN entity_nodes tn ON tn.id=e.to_entity_id
WHERE e.status='active' AND (e.from_entity_id=$1 OR e.to_entity_id=$1)
ORDER BY e.id FOR UPDATE OF e`, legacyID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	type edge struct {
		id, fromKey, toKey, relationType, evidence, sourceName, sourceURL string
		fromType, toType                                                  domain.EntityType
		verifiedAt                                                        time.Time
		status                                                            domain.Status
	}
	var edges []edge
	for rows.Next() {
		var item edge
		if err := rows.Scan(&item.id, &item.fromKey, &item.fromType, &item.toKey, &item.toType, &item.relationType, &item.evidence, &item.sourceName, &item.sourceURL, &item.verifiedAt, &item.status); err != nil {
			return 0, err
		}
		edges = append(edges, item)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	moved := 0
	for _, item := range edges {
		relationship := Relationship{Key: item.id, From: item.fromKey, To: item.toKey, RelationType: item.relationType, EvidenceNote: item.evidence, SourceName: item.sourceName, SourceURL: item.sourceURL, VerifiedAt: item.verifiedAt, Status: item.status}
		entities := map[string]Entity{item.fromKey: {Key: item.fromKey, EntityType: item.fromType}, item.toKey: {Key: item.toKey, EntityType: item.toType}, targetKey: {Key: targetKey, EntityType: domain.EntityTypeIndex}}
		fromMoved := relationship.From == legacyKey
		toMoved := relationship.To == legacyKey
		planned, _, err := planConvergenceRelationship(relationship, entities, legacyKey, targetKey)
		if err != nil {
			return moved, fmt.Errorf("index target relationship %q is incompatible: %w", item.id, err)
		}
		relationship = planned
		if fromMoved {
			if _, err := tx.ExecContext(ctx, `UPDATE entity_edges SET from_entity_id=$1,updated_at=NOW() WHERE id=$2 AND from_entity_id=$3`, targetID, item.id, legacyID); err != nil {
				return moved, err
			}
			if err := insertReferenceMove(ctx, tx, convergenceID, "entity_edges", "from_entity_id", item.id, legacyID, targetID, "redirect_to_index"); err != nil {
				return moved, err
			}
			moved++
		}
		if toMoved {
			if _, err := tx.ExecContext(ctx, `UPDATE entity_edges SET to_entity_id=$1,updated_at=NOW() WHERE id=$2 AND to_entity_id=$3`, targetID, item.id, legacyID); err != nil {
				return moved, err
			}
			if err := insertReferenceMove(ctx, tx, convergenceID, "entity_edges", "to_entity_id", item.id, legacyID, targetID, "redirect_to_index"); err != nil {
				return moved, err
			}
			moved++
		}
	}
	return moved, nil
}

func insertReferenceMove(ctx context.Context, tx *sql.Tx, convergenceID, table, column, rowID, fromID string, toID any, operation string) error {
	id := repoids.NormalizeUUID("entity_convergence_reference_move", convergenceID+"|"+table+"|"+column+"|"+rowID)
	statement := `INSERT INTO entity_convergence_reference_moves (id,convergence_id,reference_table,reference_column,reference_row_id,from_entity_id,to_entity_id,mutation_provenance) VALUES ($1,$2,$3,$4,$5,$6,$7,` + convergenceOperationMutationExpression(8) + `)`
	_, err := tx.ExecContext(ctx, statement, id, convergenceID, table, column, rowID, fromID, toID, operation)
	return err
}

func convergenceAuditMutationExpression(checksumParameter, modeParameter int) string {
	return fmt.Sprintf("jsonb_build_object('manifest_checksum',$%d::text,'mode',$%d::text)", checksumParameter, modeParameter)
}

func convergenceOperationMutationExpression(operationParameter int) string {
	return fmt.Sprintf("jsonb_build_object('operation',$%d::text)", operationParameter)
}

func currentConvergenceOwnedAliasesQuery() string {
	return `SELECT am.to_entity_id::text,am.alias,am.alias=ANY(n.aliases) AS present FROM entity_convergence_alias_moves am JOIN entity_convergences c ON c.id=am.convergence_id JOIN entity_nodes n ON n.id=am.to_entity_id WHERE c.manifest_version=(SELECT MAX(manifest_version) FROM entity_convergence_manifests) ORDER BY am.to_entity_id,am.alias`
}

func applyRecordedCorrectionMoves(ctx context.Context, tx *sql.Tx, previousID, currentID string, targetType domain.EntityType, targetID any, targetKey string) (sectorMoveCounts, error) {
	var counts sectorMoveCounts
	var previousTargetID, previousTargetType sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT target_entity_id, target_entity_type FROM entity_convergences WHERE id=$1`, previousID).Scan(&previousTargetID, &previousTargetType); err != nil {
		return counts, err
	}
	if previousTargetID.Valid != previousTargetType.Valid || (previousTargetType.Valid && previousTargetType.String != string(domain.EntityTypeSector) && previousTargetType.String != string(domain.EntityTypeIndex)) {
		return counts, fmt.Errorf("previous convergence target snapshot is invalid")
	}
	rows, err := tx.QueryContext(ctx, `SELECT reference_table,reference_column,reference_row_id,from_entity_id,to_entity_id FROM entity_convergence_reference_moves WHERE convergence_id=$1 ORDER BY reference_table,reference_column,reference_row_id`, previousID)
	if err != nil {
		return counts, err
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
			return counts, err
		}
		moves = append(moves, m)
	}
	if err := rows.Err(); err != nil {
		return counts, err
	}
	for _, m := range moves {
		if m.table == "entity_edges" && (m.column == "from_entity_id" || m.column == "to_entity_id" || m.column == "status") {
			if err := applyRecordedEdgeTransition(ctx, tx, currentID, m.table, m.column, m.rowID, m.fromID, m.toID, previousTargetID, targetType, targetID, targetKey); err != nil {
				return counts, err
			}
			counts.references++
			continue
		}
		if m.table == "sector_source_mappings" && m.column == "sector_entity_id" {
			sourceID := m.fromID
			expectedStatus := "rejected"
			operation := "forward_correction_mapping_reactivate"
			if m.toID.Valid {
				sourceID = m.toID.String
				expectedStatus = "merged"
				operation = "forward_correction_mapping_redirect"
			}
			var result sql.Result
			if targetType == domain.EntityTypeSector {
				result, err = tx.ExecContext(ctx, `UPDATE sector_source_mappings SET sector_entity_id=$1,mapping_status='merged',updated_at=NOW() WHERE id=$2 AND sector_entity_id=$3 AND mapping_status=$4`, targetID, m.rowID, sourceID, expectedStatus)
				if err == nil {
					err = insertReferenceMove(ctx, tx, currentID, m.table, m.column, m.rowID, sourceID, targetID, operation)
				}
			} else {
				result, err = tx.ExecContext(ctx, `UPDATE sector_source_mappings SET mapping_status='rejected',updated_at=NOW() WHERE id=$1 AND sector_entity_id=$2 AND mapping_status=$3`, m.rowID, sourceID, expectedStatus)
				if err == nil {
					err = insertReferenceMove(ctx, tx, currentID, m.table, m.column, m.rowID, sourceID, nil, "forward_correction_mapping_reject")
				}
			}
			if err != nil {
				return counts, err
			}
			affected, _ := result.RowsAffected()
			if affected != 1 {
				return counts, fmt.Errorf("recorded mapping row %s drifted", m.rowID)
			}
			counts.mappings++
			continue
		}
		if !m.toID.Valid {
			continue
		}
		var result sql.Result
		switch m.table + "." + m.column {
		case "sector_profiles.parent_sector_entity_id":
			if targetType != domain.EntityTypeSector {
				return counts, fmt.Errorf("sector-specific parent reference cannot target %s", targetType)
			}
			result, err = tx.ExecContext(ctx, `UPDATE sector_profiles SET parent_sector_entity_id=$1 WHERE entity_id=$2 AND parent_sector_entity_id=$3`, targetID, m.rowID, m.toID.String)
		default:
			continue
		}
		if err != nil {
			return counts, err
		}
		affected, _ := result.RowsAffected()
		if affected != 1 {
			return counts, fmt.Errorf("recorded reference %s.%s row %s drifted", m.table, m.column, m.rowID)
		}
		if err := insertReferenceMove(ctx, tx, currentID, m.table, m.column, m.rowID, m.toID.String, targetID, "forward_correction"); err != nil {
			return counts, err
		}
		if m.table == "sector_source_mappings" {
			counts.mappings++
		} else {
			counts.references++
		}
	}
	aliasRows, err := tx.QueryContext(ctx, `SELECT alias,to_entity_id FROM entity_convergence_alias_moves WHERE convergence_id=$1 ORDER BY alias`, previousID)
	if err != nil {
		return counts, err
	}
	defer aliasRows.Close()
	for aliasRows.Next() {
		var alias, oldTarget string
		if err := aliasRows.Scan(&alias, &oldTarget); err != nil {
			return counts, err
		}
		result, err := tx.ExecContext(ctx, `UPDATE entity_nodes SET aliases=array_remove(aliases,$1),updated_at=NOW() WHERE id=$2 AND $1=ANY(aliases)`, alias, oldTarget)
		if err != nil {
			return counts, err
		}
		affected, _ := result.RowsAffected()
		if affected != 1 {
			return counts, fmt.Errorf("recorded alias %q drifted", alias)
		}
		counts.aliases++
	}
	return counts, aliasRows.Err()
}

type recordedConvergenceEdge struct {
	id, fromID, fromKey, toID, toKey, relationType, evidence, sourceName, sourceURL string
	fromType, toType                                                                domain.EntityType
	verifiedAt                                                                      time.Time
	status                                                                          domain.Status
}

func applyRecordedEdgeTransition(ctx context.Context, tx *sql.Tx, currentID, table, column, rowID, legacyID string, recordedToID, previousTargetID sql.NullString, targetType domain.EntityType, targetID any, targetKey string) error {
	if table != "entity_edges" {
		return fmt.Errorf("unsupported recorded edge table %q", table)
	}
	if column == "status" {
		if previousTargetID.Valid || recordedToID.Valid {
			return fmt.Errorf("recorded no-target edge %s provenance drifted", rowID)
		}
	} else if !previousTargetID.Valid || !recordedToID.Valid || previousTargetID.String != recordedToID.String {
		return fmt.Errorf("recorded edge %s target provenance drifted", rowID)
	}
	edge, err := loadRecordedConvergenceEdge(ctx, tx, rowID)
	if err != nil {
		return err
	}
	sourceID := legacyID
	expectedStatus := domain.StatusInactive
	if column != "status" {
		sourceID = recordedToID.String
		expectedStatus = domain.StatusActive
	}
	sourceKey := ""
	if edge.fromID == sourceID {
		sourceKey = edge.fromKey
	} else if edge.toID == sourceID {
		sourceKey = edge.toKey
	} else {
		return fmt.Errorf("recorded edge %s endpoint drifted", rowID)
	}
	if edge.status != expectedStatus {
		return fmt.Errorf("recorded edge %s status drifted", rowID)
	}
	relationship := Relationship{Key: edge.id, From: edge.fromKey, To: edge.toKey, RelationType: edge.relationType, EvidenceNote: edge.evidence, SourceName: edge.sourceName, SourceURL: edge.sourceURL, VerifiedAt: edge.verifiedAt, Status: edge.status}
	entities := map[string]Entity{edge.fromKey: {Key: edge.fromKey, EntityType: edge.fromType}, edge.toKey: {Key: edge.toKey, EntityType: edge.toType}}
	if targetKey != "" {
		entities[targetKey] = Entity{Key: targetKey, EntityType: targetType}
	}
	planned, disposition, err := planConvergenceRelationship(relationship, entities, sourceKey, targetKey)
	if err != nil {
		return fmt.Errorf("recorded edge %q is incompatible: %w", rowID, err)
	}
	if disposition == convergenceEdgeDeactivate {
		result, err := tx.ExecContext(ctx, `UPDATE entity_edges SET status='inactive',updated_at=NOW() WHERE id=$1 AND status=$2`, rowID, expectedStatus)
		if err != nil {
			return err
		}
		affected, _ := result.RowsAffected()
		if affected != 1 {
			return fmt.Errorf("recorded edge %s drifted", rowID)
		}
		return insertReferenceMove(ctx, tx, currentID, "entity_edges", "status", rowID, sourceID, nil, "forward_correction_deactivate")
	}
	endpointColumn := "to_entity_id"
	currentEndpointID := edge.toID
	if planned.From != relationship.From {
		endpointColumn = "from_entity_id"
		currentEndpointID = edge.fromID
	}
	if currentEndpointID != sourceID {
		return fmt.Errorf("recorded edge %s planned endpoint drifted", rowID)
	}
	statement := fmt.Sprintf(`UPDATE entity_edges SET %s=$1,status='active',updated_at=NOW() WHERE id=$2 AND %s=$3 AND status=$4`, endpointColumn, endpointColumn)
	result, err := tx.ExecContext(ctx, statement, targetID, rowID, sourceID, expectedStatus)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return fmt.Errorf("recorded edge %s drifted", rowID)
	}
	operation := "forward_correction_redirect"
	if expectedStatus == domain.StatusInactive {
		operation = "forward_correction_reactivate_redirect"
	}
	return insertReferenceMove(ctx, tx, currentID, "entity_edges", endpointColumn, rowID, sourceID, targetID, operation)
}

func loadRecordedConvergenceEdge(ctx context.Context, tx *sql.Tx, rowID string) (recordedConvergenceEdge, error) {
	var edge recordedConvergenceEdge
	err := tx.QueryRowContext(ctx, `SELECT e.id, fn.id, fn.entity_key, fn.entity_type, tn.id, tn.entity_key, tn.entity_type, e.relation_type, e.evidence_note, e.source_name, e.source_url, e.verified_at, e.status FROM entity_edges e JOIN entity_nodes fn ON fn.id=e.from_entity_id JOIN entity_nodes tn ON tn.id=e.to_entity_id WHERE e.id=$1 FOR UPDATE OF e`, rowID).Scan(&edge.id, &edge.fromID, &edge.fromKey, &edge.fromType, &edge.toID, &edge.toKey, &edge.toType, &edge.relationType, &edge.evidence, &edge.sourceName, &edge.sourceURL, &edge.verifiedAt, &edge.status)
	return edge, err
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
