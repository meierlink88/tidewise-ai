package entityseed

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	repoids "github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) PostgresRepository {
	return PostgresRepository{db: db}
}

func (r PostgresRepository) UpsertEntity(ctx context.Context, entity Entity) (WriteResult, error) {
	if entity.Status == "" {
		entity.Status = domain.StatusActive
	}
	if err := validateEntity(entity); err != nil {
		return WriteResult{}, err
	}

	statement := `
WITH upsert AS (
    INSERT INTO entity_nodes (
        id, entity_type, layer_code, name, canonical_name, aliases, status
    ) VALUES (
        $1, $2, $3, $4, $5, $6, $7
    )
    ON CONFLICT (id) DO UPDATE SET
        entity_type = EXCLUDED.entity_type,
        layer_code = EXCLUDED.layer_code,
        name = EXCLUDED.name,
        canonical_name = EXCLUDED.canonical_name,
        aliases = EXCLUDED.aliases,
        status = EXCLUDED.status,
        updated_at = now()
    WHERE entity_nodes.entity_type IS DISTINCT FROM EXCLUDED.entity_type
       OR entity_nodes.layer_code IS DISTINCT FROM EXCLUDED.layer_code
       OR entity_nodes.name IS DISTINCT FROM EXCLUDED.name
       OR entity_nodes.canonical_name IS DISTINCT FROM EXCLUDED.canonical_name
       OR entity_nodes.aliases IS DISTINCT FROM EXCLUDED.aliases
       OR entity_nodes.status IS DISTINCT FROM EXCLUDED.status
    RETURNING xmax = 0 AS inserted
)
SELECT COALESCE((SELECT CASE WHEN inserted THEN 'created' ELSE 'updated' END FROM upsert), 'unchanged')
`
	action, err := r.queryWriteAction(ctx, statement,
		entitySeedUUID(entity.Key),
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
	if relationship.Status != domain.StatusActive && relationship.Status != domain.StatusInactive {
		return WriteResult{}, fmt.Errorf("relationship %q unsupported status %q", relationship.Key, relationship.Status)
	}

	statement := `
WITH upsert AS (
    INSERT INTO entity_edges (
        id, from_entity_id, to_entity_id, relation_type, evidence_note, status
    ) VALUES (
        $1, $2, $3, $4, $5, $6
    )
    ON CONFLICT (id) DO UPDATE SET
        from_entity_id = EXCLUDED.from_entity_id,
        to_entity_id = EXCLUDED.to_entity_id,
        relation_type = EXCLUDED.relation_type,
        evidence_note = EXCLUDED.evidence_note,
        status = EXCLUDED.status,
        updated_at = now()
    WHERE entity_edges.from_entity_id IS DISTINCT FROM EXCLUDED.from_entity_id
       OR entity_edges.to_entity_id IS DISTINCT FROM EXCLUDED.to_entity_id
       OR entity_edges.relation_type IS DISTINCT FROM EXCLUDED.relation_type
       OR entity_edges.evidence_note IS DISTINCT FROM EXCLUDED.evidence_note
       OR entity_edges.status IS DISTINCT FROM EXCLUDED.status
    RETURNING xmax = 0 AS inserted
)
SELECT COALESCE((SELECT CASE WHEN inserted THEN 'created' ELSE 'updated' END FROM upsert), 'unchanged')
`
	action, err := r.queryWriteAction(ctx, statement,
		relationshipSeedUUID(relationship.Key),
		entitySeedUUID(relationship.From),
		entitySeedUUID(relationship.To),
		relationship.RelationType,
		relationship.EvidenceNote,
		relationship.Status,
	)
	if err != nil {
		return WriteResult{}, fmt.Errorf("upsert relationship %q: %w", relationship.Key, err)
	}
	return WriteResult{Key: relationship.Key, Action: action}, nil
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
	case domain.EntityTypeSector:
		return []profileField{
			{"sector_system", text("sector_system")},
			{"sector_code", text("sector_code")},
			{"sector_type", text("sector_type")},
			{"exchange_scope", text("exchange_scope")},
			{"constituent_count", number("constituent_count")},
			{"list_date", date("list_date")},
			{"parent_sector_entity_id", ref("parent_sector_entity_id")},
			{"rank_snapshot", number("rank_snapshot")},
			{"snapshot_date", date("snapshot_date")},
		}, nil
	case domain.EntityTypeChainNode:
		return []profileField{{"chain_position", text("chain_position")}}, nil
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
	case domain.EntityTypeSector:
		return "sector_profiles", nil
	case domain.EntityTypeChainNode:
		return "chain_node_profiles", nil
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
