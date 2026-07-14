package seed

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	repoids "github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

type PostgresRepository struct {
	db   postgresExecutor
	root *sql.DB
}

type postgresExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func NewPostgresRepository(db *sql.DB) PostgresRepository {
	return PostgresRepository{db: db, root: db}
}

func (r PostgresRepository) HasRetiredIndustryEntities(ctx context.Context) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
SELECT EXISTS (
    SELECT 1 FROM entity_nodes
    WHERE entity_type IN ('sector', 'industry_chain')
)`).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check retired industry entities: %w", err)
	}
	return exists, nil
}

func (r PostgresRepository) UpsertEntity(ctx context.Context, entity Entity) (WriteResult, error) {
	if entity.Status == "" {
		entity.Status = domain.StatusActive
	}
	entity.Aliases = normalizeEntityAliases(entity.Aliases)
	if err := validateEntity(entity); err != nil {
		return WriteResult{}, err
	}

	statement := buildEntityUpsert()
	action, err := r.queryWriteAction(ctx, statement,
		entitySeedUUID(entity.Key),
		entity.Key,
		entity.EntityType,
		entity.LayerCode,
		entity.Name,
		entity.CanonicalName,
		entity.Aliases,
		entity.Status,
	)
	if err != nil {
		return WriteResult{}, fmt.Errorf("upsert entity %q: %w", entity.Key, err)
	}
	return WriteResult{Key: entity.Key, Action: action}, nil
}

func buildEntityUpsert() string {
	return `WITH incoming AS (
    SELECT $1::uuid AS id, $2::text AS entity_key, $3::text AS entity_type,
           $4::text AS layer_code, $5::text AS name, $6::text AS canonical_name,
           $7::text[] AS aliases,
           $8::text AS status
), upsert AS (
    INSERT INTO entity_nodes (
        id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status
    ) SELECT id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status FROM incoming
    ON CONFLICT (id) DO UPDATE SET
        entity_key = EXCLUDED.entity_key,
        entity_type = EXCLUDED.entity_type,
        layer_code = EXCLUDED.layer_code,
        name = EXCLUDED.name,
        canonical_name = EXCLUDED.canonical_name,
        aliases = EXCLUDED.aliases,
        status = EXCLUDED.status,
        updated_at = now()
    WHERE entity_nodes.entity_key IS DISTINCT FROM EXCLUDED.entity_key
       OR entity_nodes.entity_type IS DISTINCT FROM EXCLUDED.entity_type
       OR entity_nodes.layer_code IS DISTINCT FROM EXCLUDED.layer_code
       OR entity_nodes.name IS DISTINCT FROM EXCLUDED.name
       OR entity_nodes.canonical_name IS DISTINCT FROM EXCLUDED.canonical_name
       OR entity_nodes.aliases IS DISTINCT FROM EXCLUDED.aliases
       OR entity_nodes.status IS DISTINCT FROM EXCLUDED.status
    RETURNING xmax = 0 AS inserted
)
SELECT COALESCE((SELECT CASE WHEN inserted THEN 'created' ELSE 'updated' END FROM upsert), 'unchanged')`
}

func (r PostgresRepository) UpsertProfile(ctx context.Context, profile Profile) (WriteResult, error) {
	if profile.EntityKey == "" {
		return WriteResult{}, fmt.Errorf("profile entity key is required")
	}
	if err := validateProfileData(profile.EntityType, profile.Data); err != nil {
		return WriteResult{}, err
	}

	statement, args, err := buildProfileUpsert(profile.EntityKey, profile.EntityType, profile.Data)
	if err != nil {
		return WriteResult{}, err
	}
	action, err := r.queryWriteAction(ctx, statement, args...)
	if err != nil {
		return WriteResult{}, fmt.Errorf("upsert profile %q: %w", profile.EntityKey, err)
	}
	return WriteResult{Key: profile.EntityKey, Action: action}, nil
}

func (r PostgresRepository) UpsertExternalIdentifier(ctx context.Context, identifier domain.EntityExternalIdentifier) (WriteResult, error) {
	identifier = normalizeExternalIdentifier(identifier)
	if err := validateFirstBatchExternalIdentifier(identifier); err != nil {
		return WriteResult{}, err
	}
	identity := externalIdentifierIdentity(identifier.SourceSystem, identifier.SourceTaxonomyType, identifier.ExternalCode)
	tx, err := r.root.BeginTx(ctx, nil)
	if err != nil {
		return WriteResult{}, fmt.Errorf("begin external identifier transaction %q: %w", identity, err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if _, err := tx.ExecContext(ctx, externalIdentifierTransactionLockSQL(), identity); err != nil {
		return WriteResult{}, fmt.Errorf("lock external identifier %q: %w", identity, err)
	}
	var targetID string
	if err := tx.QueryRowContext(ctx, externalIdentifierTargetSQL(), identifier.EntityID).Scan(&targetID); err != nil {
		if err == sql.ErrNoRows {
			return WriteResult{}, fmt.Errorf("external identifier %q requires an active chain_node target", identity)
		}
		return WriteResult{}, fmt.Errorf("validate external identifier target %q: %w", identity, err)
	}

	existing, found, err := selectExternalIdentifierForUpdate(ctx, tx, identifier)
	if err != nil {
		return WriteResult{}, fmt.Errorf("read external identifier %q: %w", identity, err)
	}
	if found {
		action, err := reconcileExternalIdentifier(ctx, tx, identifier, existing)
		if err != nil {
			return WriteResult{}, err
		}
		return commitExternalIdentifier(tx, identity, action)
	}

	var insertedID string
	err = tx.QueryRowContext(
		ctx,
		externalIdentifierInsertSQL(),
		identifier.ID,
		identifier.EntityID,
		identifier.SourceSystem,
		identifier.SourceTaxonomyType,
		identifier.ExternalCode,
		identifier.ExternalName,
		identifier.Status,
	).Scan(&insertedID)
	if err == nil {
		return commitExternalIdentifier(tx, identity, WriteCreated)
	}
	if err != sql.ErrNoRows {
		return WriteResult{}, fmt.Errorf("insert external identifier %q: %w", identity, err)
	}

	existing, found, err = selectExternalIdentifierForUpdate(ctx, tx, identifier)
	if err != nil {
		return WriteResult{}, fmt.Errorf("re-read concurrent external identifier %q: %w", identity, err)
	}
	if !found {
		return WriteResult{}, fmt.Errorf("external identifier %q conflict winner was not visible after insert", identity)
	}
	action, err := reconcileExternalIdentifier(ctx, tx, identifier, existing)
	if err != nil {
		return WriteResult{}, err
	}
	return commitExternalIdentifier(tx, identity, action)
}

func externalIdentifierTransactionLockSQL() string {
	return "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))"
}

func externalIdentifierTargetSQL() string {
	return `
SELECT id FROM entity_nodes
WHERE id = $1::uuid
  AND entity_type = 'chain_node'
  AND status = 'active'
FOR SHARE`
}

func externalIdentifierSelectSQL() string {
	return `
SELECT id, entity_id, external_name, status FROM entity_external_identifiers
WHERE source_system = $1
  AND source_taxonomy_type = $2
  AND external_code = $3
FOR UPDATE`
}

func externalIdentifierInsertSQL() string {
	return `
INSERT INTO entity_external_identifiers (
    id, entity_id, source_system, source_taxonomy_type, external_code, external_name, status
) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7)
ON CONFLICT (source_system, source_taxonomy_type, external_code) DO NOTHING
RETURNING id`
}

type storedExternalIdentifier struct {
	ID           string
	EntityID     string
	ExternalName string
	Status       domain.Status
}

func selectExternalIdentifierForUpdate(ctx context.Context, tx *sql.Tx, identifier domain.EntityExternalIdentifier) (storedExternalIdentifier, bool, error) {
	var existing storedExternalIdentifier
	err := tx.QueryRowContext(
		ctx,
		externalIdentifierSelectSQL(),
		identifier.SourceSystem,
		identifier.SourceTaxonomyType,
		identifier.ExternalCode,
	).Scan(&existing.ID, &existing.EntityID, &existing.ExternalName, &existing.Status)
	if err == sql.ErrNoRows {
		return storedExternalIdentifier{}, false, nil
	}
	return existing, err == nil, err
}

func reconcileExternalIdentifier(ctx context.Context, tx *sql.Tx, wanted domain.EntityExternalIdentifier, existing storedExternalIdentifier) (WriteAction, error) {
	identity := externalIdentifierIdentity(wanted.SourceSystem, wanted.SourceTaxonomyType, wanted.ExternalCode)
	if existing.ID != wanted.ID || existing.EntityID != wanted.EntityID {
		return "", fmt.Errorf("external identifier %q identity conflict", identity)
	}
	if existing.ExternalName == wanted.ExternalName && existing.Status == wanted.Status {
		return WriteUnchanged, nil
	}
	result, err := tx.ExecContext(ctx, `
UPDATE entity_external_identifiers
SET external_name = $1, status = $2, updated_at = now()
WHERE id = $3::uuid`, wanted.ExternalName, wanted.Status, wanted.ID)
	if err != nil {
		return "", fmt.Errorf("update external identifier %q: %w", identity, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return "", fmt.Errorf("read external identifier %q update count: %w", identity, err)
	}
	if rows != 1 {
		return "", fmt.Errorf("external identifier %q update affected %d rows, want 1", identity, rows)
	}
	return WriteUpdated, nil
}

func commitExternalIdentifier(tx *sql.Tx, identity string, action WriteAction) (WriteResult, error) {
	if err := tx.Commit(); err != nil {
		return WriteResult{}, fmt.Errorf("commit external identifier %q: %w", identity, err)
	}
	return WriteResult{Key: identity, Action: action}, nil
}

func (r PostgresRepository) UpsertSectorSourceMapping(ctx context.Context, mapping SectorSourceMapping) (WriteResult, error) {
	statement, args, err := buildSectorSourceMappingUpsert(mapping)
	if err != nil {
		return WriteResult{}, err
	}
	action, err := r.queryWriteAction(ctx, statement, args...)
	if err != nil {
		return WriteResult{}, fmt.Errorf("upsert sector source mapping %q: %w", sectorSourceMappingIdentity(mapping), err)
	}
	return WriteResult{Key: sectorSourceMappingIdentity(mapping), Action: action}, nil
}

func buildSectorSourceMappingUpsert(mapping SectorSourceMapping) (string, []any, error) {
	mapping = normalizeSectorSourceMapping(mapping)
	if err := validateSectorSourceMapping(mapping); err != nil {
		return "", nil, err
	}
	var snapshotDate any
	if mapping.SnapshotDate != "" {
		parsed, err := time.Parse("2006-01-02", mapping.SnapshotDate)
		if err != nil {
			return "", nil, fmt.Errorf("invalid sector source mapping snapshot date %q", mapping.SnapshotDate)
		}
		snapshotDate = parsed
	}
	identity := sectorSourceMappingIdentity(mapping)
	statement := `
WITH upsert AS (
    INSERT INTO sector_source_mappings (
        id, sector_entity_id, source_system, source_taxonomy_type, source_sector_code,
        source_sector_name, source_sector_name_normalized, source_market_scope, source_url,
        rank_snapshot, snapshot_date, mapping_status, review_note
    ) VALUES (
        $1, $2, $3, $4, $5,
        $6, $7, $8, $9,
        $10, $11, $12, $13
    )
    ON CONFLICT (id) DO UPDATE SET
        sector_entity_id = EXCLUDED.sector_entity_id,
        source_sector_name = EXCLUDED.source_sector_name,
        source_sector_name_normalized = EXCLUDED.source_sector_name_normalized,
        source_market_scope = EXCLUDED.source_market_scope,
        source_url = CASE
            WHEN sector_source_mappings.snapshot_date IS NULL
              OR (EXCLUDED.snapshot_date IS NOT NULL AND EXCLUDED.snapshot_date >= sector_source_mappings.snapshot_date)
            THEN EXCLUDED.source_url
            ELSE sector_source_mappings.source_url
        END,
        rank_snapshot = CASE
            WHEN sector_source_mappings.snapshot_date IS NULL
              OR (EXCLUDED.snapshot_date IS NOT NULL AND EXCLUDED.snapshot_date >= sector_source_mappings.snapshot_date)
            THEN EXCLUDED.rank_snapshot
            ELSE sector_source_mappings.rank_snapshot
        END,
        snapshot_date = CASE
            WHEN sector_source_mappings.snapshot_date IS NULL
              OR (EXCLUDED.snapshot_date IS NOT NULL AND EXCLUDED.snapshot_date >= sector_source_mappings.snapshot_date)
            THEN EXCLUDED.snapshot_date
            ELSE sector_source_mappings.snapshot_date
        END,
        mapping_status = EXCLUDED.mapping_status,
        review_note = EXCLUDED.review_note,
        updated_at = NOW()
    WHERE (sector_source_mappings.sector_entity_id IS DISTINCT FROM EXCLUDED.sector_entity_id
       OR sector_source_mappings.source_sector_name IS DISTINCT FROM EXCLUDED.source_sector_name
       OR sector_source_mappings.source_sector_name_normalized IS DISTINCT FROM EXCLUDED.source_sector_name_normalized
       OR sector_source_mappings.source_market_scope IS DISTINCT FROM EXCLUDED.source_market_scope
       OR sector_source_mappings.mapping_status IS DISTINCT FROM EXCLUDED.mapping_status
       OR sector_source_mappings.review_note IS DISTINCT FROM EXCLUDED.review_note
       OR ((sector_source_mappings.snapshot_date IS NULL
            OR (EXCLUDED.snapshot_date IS NOT NULL
                AND EXCLUDED.snapshot_date >= sector_source_mappings.snapshot_date))
           AND (sector_source_mappings.source_url IS DISTINCT FROM EXCLUDED.source_url
                OR sector_source_mappings.rank_snapshot IS DISTINCT FROM EXCLUDED.rank_snapshot
                OR sector_source_mappings.snapshot_date IS DISTINCT FROM EXCLUDED.snapshot_date)))
    RETURNING xmax = 0 AS inserted
)
SELECT COALESCE((SELECT CASE WHEN inserted THEN 'created' ELSE 'updated' END FROM upsert), 'unchanged')
`
	args := []any{
		sectorSourceMappingSeedUUID(identity), entitySeedUUID(mapping.SectorEntityKey), mapping.SourceSystem,
		mapping.SourceTaxonomyType, mapping.SourceSectorCode, mapping.SourceSectorName,
		mapping.SourceSectorNameNormalized, mapping.SourceMarketScope, mapping.SourceURL,
		mapping.RankSnapshot, snapshotDate, mapping.MappingStatus, mapping.ReviewNote,
	}
	return statement, args, nil
}

func (r PostgresRepository) UpsertRelationship(ctx context.Context, relationship Relationship) (WriteResult, error) {
	if relationship.Status == "" {
		relationship.Status = domain.StatusActive
	}
	if relationship.Key == "" {
		return WriteResult{}, fmt.Errorf("relationship key is required")
	}
	if relationship.From == "" {
		return WriteResult{}, fmt.Errorf("relationship %q source is required", relationship.Key)
	}
	if relationship.To == "" {
		return WriteResult{}, fmt.Errorf("relationship %q target is required", relationship.Key)
	}
	if relationship.RelationType == "" {
		return WriteResult{}, fmt.Errorf("relationship %q relation type is required", relationship.Key)
	}
	if err := validateRelationshipProvenance(relationship); err != nil {
		return WriteResult{}, fmt.Errorf("relationship %q: %w", relationship.Key, err)
	}
	if relationship.Status != domain.StatusActive && relationship.Status != domain.StatusInactive {
		return WriteResult{}, fmt.Errorf("relationship %q unsupported status %q", relationship.Key, relationship.Status)
	}

	statement, args := buildRelationshipUpsert(relationship)
	action, err := r.queryWriteAction(ctx, statement, args...)
	if err != nil {
		return WriteResult{}, fmt.Errorf("upsert relationship %q: %w", relationship.Key, err)
	}
	return WriteResult{Key: relationship.Key, Action: action}, nil
}

func (r PostgresRepository) UpsertRelationshipBatch(ctx context.Context, relationships []Relationship) ([]WriteResult, error) {
	if r.root == nil {
		return nil, fmt.Errorf("atomic relationship batch requires database root")
	}
	tx, err := r.root.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	txRepo := PostgresRepository{db: tx, root: r.root}
	entities, err := txRepo.lockActiveRelationshipEndpoints(ctx, relationships)
	if err != nil {
		return nil, err
	}
	results := make([]WriteResult, 0, len(relationships))
	for _, relationship := range relationships {
		if err := validateRelationshipPolicy(relationship, entities); err != nil {
			return nil, err
		}
		result, err := txRepo.UpsertRelationship(ctx, relationship)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return results, nil
}

func (r PostgresRepository) lockActiveRelationshipEndpoints(ctx context.Context, relationships []Relationship) (map[string]Entity, error) {
	keys := make(map[string]struct{}, len(relationships)*2)
	for _, relationship := range relationships {
		keys[relationship.From] = struct{}{}
		keys[relationship.To] = struct{}{}
	}
	ordered := make([]string, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Strings(ordered)
	entities := make(map[string]Entity, len(ordered))
	for _, key := range ordered {
		var entity Entity
		if err := r.db.QueryRowContext(ctx, `SELECT entity_key, entity_type, status FROM entity_nodes WHERE entity_key=$1 FOR SHARE`, key).Scan(&entity.Key, &entity.EntityType, &entity.Status); err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("relationship endpoint %q is missing", key)
			}
			return nil, err
		}
		if entity.Status != domain.StatusActive {
			return nil, fmt.Errorf("relationship endpoint %q is not active", key)
		}
		entities[key] = entity
	}
	return entities, nil
}

func buildRelationshipUpsert(relationship Relationship) (string, []any) {
	statement := `
WITH existing AS (
    SELECT from_entity_id,to_entity_id,relation_type FROM entity_edges WHERE id=$1 FOR UPDATE
), upsert AS (
    INSERT INTO entity_edges (
        id, from_entity_id, to_entity_id, relation_type, evidence_note, status,
        source_name, source_url, verified_at
    ) VALUES (
        $1, $2, $3, $4, $5, $6,
        $7, $8, $9
    )
    ON CONFLICT (id) DO UPDATE SET
        evidence_note = EXCLUDED.evidence_note,
        status = EXCLUDED.status,
		source_name = EXCLUDED.source_name,
		source_url = EXCLUDED.source_url,
		verified_at = EXCLUDED.verified_at,
        updated_at = now()
    WHERE entity_edges.from_entity_id = EXCLUDED.from_entity_id
      AND entity_edges.to_entity_id = EXCLUDED.to_entity_id
      AND entity_edges.relation_type = EXCLUDED.relation_type
      AND (entity_edges.evidence_note IS DISTINCT FROM EXCLUDED.evidence_note
       OR entity_edges.status IS DISTINCT FROM EXCLUDED.status
	   OR entity_edges.source_name IS DISTINCT FROM EXCLUDED.source_name
	   OR entity_edges.source_url IS DISTINCT FROM EXCLUDED.source_url
	   OR entity_edges.verified_at IS DISTINCT FROM EXCLUDED.verified_at)
    RETURNING xmax = 0 AS inserted
)
SELECT CASE
    WHEN EXISTS (SELECT 1 FROM existing WHERE from_entity_id IS DISTINCT FROM $2 OR to_entity_id IS DISTINCT FROM $3 OR relation_type IS DISTINCT FROM $4) THEN 'identity_conflict'
    ELSE COALESCE((SELECT CASE WHEN inserted THEN 'created' ELSE 'updated' END FROM upsert), 'unchanged')
END
`
	args := []any{
		relationshipSeedUUID(relationship.Key),
		entitySeedUUID(relationship.From),
		entitySeedUUID(relationship.To),
		relationship.RelationType,
		relationship.EvidenceNote,
		relationship.Status,
		relationship.SourceName,
		relationship.SourceURL,
		relationship.VerifiedAt,
	}
	return statement, args
}

func (r PostgresRepository) queryWriteAction(ctx context.Context, statement string, args ...any) (WriteAction, error) {
	var action string
	if err := r.db.QueryRowContext(ctx, statement, args...).Scan(&action); err != nil {
		return "", err
	}
	switch WriteAction(action) {
	case WriteCreated, WriteUpdated, WriteUnchanged:
		return WriteAction(action), nil
	default:
		return "", fmt.Errorf("unsupported write action %q", action)
	}
}

func buildProfileUpsert(entityKey string, entityType domain.EntityType, data []byte) (string, []any, error) {
	table, err := profileTableName(entityType)
	if err != nil {
		return "", nil, err
	}
	fields, err := profileFields(entityType, data)
	if err != nil {
		return "", nil, err
	}

	columns := []string{"entity_id"}
	args := []any{entitySeedUUID(entityKey)}
	for _, field := range fields {
		columns = append(columns, field.column)
		args = append(args, field.value)
	}

	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = "$" + strconv.Itoa(i+1)
	}

	assignments := make([]string, 0, len(fields))
	distinctChecks := make([]string, 0, len(fields))
	for _, field := range fields {
		assignments = append(assignments, fmt.Sprintf("%s = EXCLUDED.%s", field.column, field.column))
		distinctChecks = append(distinctChecks, fmt.Sprintf("%s.%s IS DISTINCT FROM EXCLUDED.%s", table, field.column, field.column))
	}

	statement := fmt.Sprintf(`
WITH upsert AS (
    INSERT INTO %s (
        %s
    ) VALUES (
        %s
    )
    ON CONFLICT (entity_id) DO UPDATE SET
        %s
    WHERE %s
    RETURNING xmax = 0 AS inserted
)
SELECT COALESCE((SELECT CASE WHEN inserted THEN 'created' ELSE 'updated' END FROM upsert), 'unchanged')
`, table, strings.Join(columns, ", "), strings.Join(placeholders, ", "), strings.Join(assignments, ", "), strings.Join(distinctChecks, " OR "))

	return statement, args, nil
}

type profileField struct {
	column string
	value  any
}

func profileFields(entityType domain.EntityType, data []byte) ([]profileField, error) {
	var profile map[string]any
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("decode profile: %w", err)
	}

	text := func(name string) any { return stringProfileValue(profile, name) }
	ref := func(name string) any { return entityRefProfileValue(profile, name) }
	date := func(name string) any { return dateProfileValue(profile, name) }
	number := func(name string) any { return intProfileValue(profile, name) }

	switch entityType {
	case domain.EntityTypeAllianceOrg:
		return []profileField{
			{"org_code", text("org_code")},
			{"org_type", text("org_type")},
			{"primary_domain", text("primary_domain")},
			{"scope_region", text("scope_region")},
			{"official_url", text("official_url")},
		}, nil
	case domain.EntityTypeEconomy:
		return []profileField{
			{"country_code", text("country_code")},
			{"currency_code", text("currency_code")},
			{"region", text("region")},
		}, nil
	case domain.EntityTypePolicyBody:
		return []profileField{
			{"body_type", text("body_type")},
			{"jurisdiction", text("jurisdiction")},
			{"policy_domain", text("policy_domain")},
		}, nil
	case domain.EntityTypeMarket:
		return []profileField{
			{"market_type", text("market_type")},
			{"economy_entity_id", ref("economy_entity_id")},
			{"currency_code", text("currency_code")},
			{"timezone", text("timezone")},
		}, nil
	case domain.EntityTypeIndex:
		return []profileField{
			{"index_code", text("index_code")},
			{"index_type", text("index_type")},
			{"market_entity_id", ref("market_entity_id")},
			{"provider", text("provider")},
			{"currency_code", text("currency_code")},
			{"list_date", date("list_date")},
		}, nil
	case domain.EntityTypeBenchmark:
		return []profileField{
			{"benchmark_type", text("benchmark_type")},
			{"official_series_code", text("official_series_code")},
			{"provider", text("provider")},
			{"tenor", text("tenor")},
			{"underlying_symbol", text("underlying_symbol")},
			{"currency_code", text("currency_code")},
			{"unit", text("unit")},
			{"frequency", text("frequency")},
			{"source_url", text("source_url")},
		}, nil
	case domain.EntityTypeSector:
		fields := []profileField{
			{"sector_system", text("sector_system")},
			{"sector_code", text("sector_code")},
			{"sector_type", text("sector_type")},
			{"exchange_scope", text("exchange_scope")},
			{"constituent_count", number("constituent_count")},
			{"list_date", date("list_date")},
			{"parent_sector_entity_id", ref("parent_sector_entity_id")},
			{"rank_snapshot", number("rank_snapshot")},
			{"snapshot_date", date("snapshot_date")},
		}
		for _, field := range []struct {
			name  string
			value func(string) any
		}{
			{"classification_code", text},
			{"primary_market_entity_id", ref},
			{"primary_economy_entity_id", ref},
			{"methodology_url", text},
			{"review_status", text},
		} {
			if _, ok := profile[field.name]; ok {
				fields = append(fields, profileField{field.name, field.value(field.name)})
			}
		}
		return fields, nil
	case domain.EntityTypeIndustryChain:
		return []profileField{
			{"chain_code", text("chain_code")},
			{"definition", text("definition")},
			{"boundary_note", text("boundary_note")},
			{"scope_type", text("scope_type")},
			{"primary_economy_entity_id", ref("primary_economy_entity_id")},
			{"version", number("version")},
			{"review_status", text("review_status")},
			{"source_name", text("source_name")},
			{"source_url", text("source_url")},
			{"verified_at", text("verified_at")},
		}, nil
	case domain.EntityTypeChainNode:
		return []profileField{{"definition", text("definition")}, {"boundary_note", nullableTextProfileValue(profile, "boundary_note")}}, nil
	case domain.EntityTypeTheme:
		return []profileField{{"definition", text("definition")}, {"boundary_note", text("boundary_note")}}, nil
	case domain.EntityTypeCompany:
		return []profileField{
			{"registration_economy_entity_id", ref("registration_economy_entity_id")},
			{"area", text("area")},
			{"industry_name", text("industry_name")},
			{"controller_name", text("controller_name")},
			{"controller_type", text("controller_type")},
		}, nil
	case domain.EntityTypeSecurity:
		return []profileField{
			{"ticker", text("ticker")},
			{"symbol", text("symbol")},
			{"exchange", text("exchange")},
			{"market_board", text("market_board")},
			{"security_type", text("security_type")},
			{"issuer_company_entity_id", ref("issuer_company_entity_id")},
			{"list_date", date("list_date")},
			{"delist_date", date("delist_date")},
			{"list_status", text("list_status")},
			{"currency_code", text("currency_code")},
		}, nil
	case domain.EntityTypeInstrument:
		return []profileField{
			{"instrument_type", text("instrument_type")},
			{"underlying_entity_id", ref("underlying_entity_id")},
			{"exchange", text("exchange")},
			{"currency_code", text("currency_code")},
		}, nil
	case domain.EntityTypeMetric:
		return []profileField{
			{"metric_type", text("metric_type")},
			{"unit", text("unit")},
			{"frequency", text("frequency")},
		}, nil
	case domain.EntityTypeCommodity:
		return []profileField{{"commodity_type", text("commodity_type")}}, nil
	case domain.EntityTypePerson:
		return []profileField{
			{"role_title", text("role_title")},
			{"organization_entity_id", ref("organization_entity_id")},
			{"economy_entity_id", ref("economy_entity_id")},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported profile entity type %q", entityType)
	}
}

func profileTableName(entityType domain.EntityType) (string, error) {
	switch entityType {
	case domain.EntityTypeAllianceOrg:
		return "alliance_org_profiles", nil
	case domain.EntityTypeEconomy:
		return "economy_profiles", nil
	case domain.EntityTypePolicyBody:
		return "policy_body_profiles", nil
	case domain.EntityTypeMarket:
		return "market_profiles", nil
	case domain.EntityTypeIndex:
		return "index_profiles", nil
	case domain.EntityTypeBenchmark:
		return "benchmark_profiles", nil
	case domain.EntityTypeSector:
		return "sector_profiles", nil
	case domain.EntityTypeIndustryChain:
		return "industry_chain_profiles", nil
	case domain.EntityTypeChainNode:
		return "chain_node_profiles", nil
	case domain.EntityTypeTheme:
		return "theme_profiles", nil
	case domain.EntityTypeCompany:
		return "company_profiles", nil
	case domain.EntityTypeSecurity:
		return "security_profiles", nil
	case domain.EntityTypeInstrument:
		return "instrument_profiles", nil
	case domain.EntityTypeMetric:
		return "metric_profiles", nil
	case domain.EntityTypeCommodity:
		return "commodity_profiles", nil
	case domain.EntityTypePerson:
		return "person_profiles", nil
	default:
		return "", fmt.Errorf("unsupported profile entity type %q", entityType)
	}
}

func stringProfileValue(profile map[string]any, key string) string {
	value, ok := profile[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func nullableTextProfileValue(profile map[string]any, key string) any {
	value, ok := profile[key]
	if !ok || value == nil {
		return nil
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func entityRefProfileValue(profile map[string]any, key string) any {
	value := stringProfileValue(profile, key)
	if value == "" {
		return nil
	}
	return entitySeedUUID(value)
}

func intProfileValue(profile map[string]any, key string) int {
	value, ok := profile[key]
	if !ok || value == nil {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return 0
}

func dateProfileValue(profile map[string]any, key string) any {
	value := stringProfileValue(profile, key)
	if value == "" {
		return nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil
	}
	return parsed
}

func entitySeedUUID(key string) string {
	return repoids.NormalizeUUID("entity", key)
}

func relationshipSeedUUID(key string) string {
	return repoids.NormalizeUUID("entity_relationship", key)
}

func sectorSourceMappingSeedUUID(identity string) string {
	return repoids.NormalizeUUID("sector_source_mapping", identity)
}

func normalizeEntityAliases(aliases []string) []string {
	if aliases == nil {
		return []string{}
	}
	return aliases
}
