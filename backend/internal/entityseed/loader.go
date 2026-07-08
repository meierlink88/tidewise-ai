package entityseed

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type Manifest struct {
	Entities      []Entity       `json:"entities"`
	Profiles      []Profile      `json:"profiles,omitempty"`
	Relationships []Relationship `json:"relationships,omitempty"`
}

type Entity struct {
	Key           string            `json:"key"`
	EntityType    domain.EntityType `json:"entity_type"`
	LayerCode     string            `json:"layer_code"`
	Name          string            `json:"name"`
	CanonicalName string            `json:"canonical_name"`
	Aliases       []string          `json:"aliases,omitempty"`
	Status        domain.Status     `json:"status,omitempty"`
	Profile       json.RawMessage   `json:"profile"`
}

type Profile struct {
	EntityKey  string            `json:"entity_key"`
	EntityType domain.EntityType `json:"entity_type"`
	Data       json.RawMessage   `json:"data"`
}

type Relationship struct {
	Key          string        `json:"key"`
	From         string        `json:"from"`
	To           string        `json:"to"`
	RelationType string        `json:"relation_type"`
	EvidenceNote string        `json:"evidence_note,omitempty"`
	Status       domain.Status `json:"status,omitempty"`
}

func LoadFile(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read entity seed manifest: %w", err)
	}
	return Load(data)
}

func LoadFiles(paths ...string) (Manifest, error) {
	var merged Manifest
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return Manifest{}, fmt.Errorf("read entity seed manifest %q: %w", path, err)
		}
		if err := rejectForbiddenReasoningFields(data); err != nil {
			return Manifest{}, fmt.Errorf("%s: %w", path, err)
		}
		var manifest Manifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			return Manifest{}, fmt.Errorf("decode entity seed manifest %q: %w", path, err)
		}
		merged.Entities = append(merged.Entities, manifest.Entities...)
		merged.Profiles = append(merged.Profiles, manifest.Profiles...)
		merged.Relationships = append(merged.Relationships, manifest.Relationships...)
	}
	if err := Validate(merged); err != nil {
		return Manifest{}, err
	}
	normalizeDefaults(&merged)
	return merged, nil
}

func Load(data []byte) (Manifest, error) {
	if err := rejectForbiddenReasoningFields(data); err != nil {
		return Manifest{}, err
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode entity seed manifest: %w", err)
	}
	if err := Validate(manifest); err != nil {
		return Manifest{}, err
	}
	normalizeDefaults(&manifest)
	return manifest, nil
}

func Validate(manifest Manifest) error {
	entityKeys := make(map[string]Entity, len(manifest.Entities))
	for _, entity := range manifest.Entities {
		if entity.Key == "" {
			return fmt.Errorf("entity key is required")
		}
		if _, ok := entityKeys[entity.Key]; ok {
			return fmt.Errorf("duplicate entity key %q", entity.Key)
		}
		if err := validateEntity(entity); err != nil {
			return fmt.Errorf("entity %q: %w", entity.Key, err)
		}
		if err := validateProfileData(entity.EntityType, entity.Profile); err != nil {
			return fmt.Errorf("entity %q profile: %w", entity.Key, err)
		}
		entityKeys[entity.Key] = entity
	}

	for _, profile := range manifest.Profiles {
		if profile.EntityKey == "" {
			return fmt.Errorf("profile entity key is required")
		}
		entity, ok := entityKeys[profile.EntityKey]
		if !ok {
			return fmt.Errorf("unknown profile entity key %q", profile.EntityKey)
		}
		entityType := profile.EntityType
		if entityType == "" {
			entityType = entity.EntityType
		}
		if entityType != entity.EntityType {
			return fmt.Errorf("profile entity type %q does not match entity %q type %q", entityType, profile.EntityKey, entity.EntityType)
		}
		if err := validateProfileData(entityType, profile.Data); err != nil {
			return fmt.Errorf("profile %q: %w", profile.EntityKey, err)
		}
	}

	relationshipKeys := make(map[string]struct{}, len(manifest.Relationships))
	for _, relationship := range manifest.Relationships {
		if relationship.Key == "" {
			return fmt.Errorf("relationship key is required")
		}
		if _, ok := relationshipKeys[relationship.Key]; ok {
			return fmt.Errorf("duplicate relationship key %q", relationship.Key)
		}
		if relationship.From == "" {
			return fmt.Errorf("relationship %q source is required", relationship.Key)
		}
		if _, ok := entityKeys[relationship.From]; !ok {
			return fmt.Errorf("unknown relationship source %q", relationship.From)
		}
		if relationship.To == "" {
			return fmt.Errorf("relationship %q target is required", relationship.Key)
		}
		if _, ok := entityKeys[relationship.To]; !ok {
			return fmt.Errorf("unknown relationship target %q", relationship.To)
		}
		if relationship.RelationType == "" {
			return fmt.Errorf("relationship %q relation type is required", relationship.Key)
		}
		if relationship.Status != "" && relationship.Status != domain.StatusActive && relationship.Status != domain.StatusInactive {
			return fmt.Errorf("relationship %q unsupported status %q", relationship.Key, relationship.Status)
		}
		relationshipKeys[relationship.Key] = struct{}{}
	}

	return nil
}

func validateEntity(entity Entity) error {
	status := entity.Status
	if status == "" {
		status = domain.StatusActive
	}
	node := domain.EntityNode{
		ID:            entity.Key,
		EntityType:    entity.EntityType,
		LayerCode:     entity.LayerCode,
		Name:          entity.Name,
		CanonicalName: entity.CanonicalName,
		Aliases:       entity.Aliases,
		Status:        status,
	}
	return node.Validate()
}

func validateProfileData(entityType domain.EntityType, data json.RawMessage) error {
	if len(data) == 0 || string(data) == "null" {
		return fmt.Errorf("profile is required")
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("profile must be an object: %w", err)
	}
	for _, field := range requiredProfileFields(entityType) {
		raw, ok := fields[field]
		if !ok || string(raw) == "null" {
			return fmt.Errorf("%s is required", field)
		}
		var value string
		if err := json.Unmarshal(raw, &value); err != nil || strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	return nil
}

func requiredProfileFields(entityType domain.EntityType) []string {
	switch entityType {
	case domain.EntityTypeAllianceOrg:
		return []string{"org_code", "org_type"}
	case domain.EntityTypeEconomy:
		return []string{"country_code", "currency_code"}
	case domain.EntityTypePolicyBody:
		return []string{"body_type"}
	case domain.EntityTypeMarket:
		return []string{"market_type"}
	case domain.EntityTypeIndex:
		return []string{"index_code", "index_type"}
	case domain.EntityTypeSector:
		return []string{"sector_system", "sector_code", "sector_type"}
	case domain.EntityTypeChainNode:
		return []string{"chain_position"}
	case domain.EntityTypeSecurity:
		return []string{"ticker", "exchange", "security_type"}
	case domain.EntityTypeInstrument:
		return []string{"instrument_type"}
	case domain.EntityTypeMetric:
		return []string{"metric_type"}
	case domain.EntityTypeCommodity:
		return []string{"commodity_type"}
	case domain.EntityTypePerson:
		return []string{"role_title"}
	default:
		return nil
	}
}

func normalizeDefaults(manifest *Manifest) {
	for i := range manifest.Entities {
		if manifest.Entities[i].Status == "" {
			manifest.Entities[i].Status = domain.StatusActive
		}
	}
	for i := range manifest.Relationships {
		if manifest.Relationships[i].Status == "" {
			manifest.Relationships[i].Status = domain.StatusActive
		}
	}
}

func rejectForbiddenReasoningFields(data []byte) error {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("decode entity seed manifest: %w", err)
	}
	return scanForbiddenReasoningFields(value)
}

func scanForbiddenReasoningFields(value any) error {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if forbiddenReasoningField(key) {
				return fmt.Errorf("forbidden reasoning field %q", key)
			}
			if err := scanForbiddenReasoningFields(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range typed {
			if err := scanForbiddenReasoningFields(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func forbiddenReasoningField(field string) bool {
	normalized := strings.ToLower(strings.TrimSpace(field))
	for _, forbidden := range []string{
		"event_score",
		"impact_strength",
		"transmission_strength",
		"prediction",
		"investment_advice",
		"bullish",
		"bearish",
		"利好",
		"利空",
		"预测结论",
		"传导强度",
		"事件评分",
		"投资建议",
	} {
		if normalized == strings.ToLower(forbidden) {
			return true
		}
	}
	return false
}
