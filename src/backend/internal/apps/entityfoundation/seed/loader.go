package seed

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"golang.org/x/text/unicode/norm"
)

type Manifest struct {
	Entities                         []Entity                              `json:"entities"`
	Profiles                         []Profile                             `json:"profiles,omitempty"`
	SectorSourceMappings             []SectorSourceMapping                 `json:"sector_source_mappings,omitempty"`
	Relationships                    []Relationship                        `json:"relationships,omitempty"`
	IndustryChainMemberships         []IndustryChainMembershipSeed         `json:"industry_chain_memberships,omitempty"`
	IndustryChainTopologyEdges       []IndustryChainTopologySeed           `json:"industry_chain_topology_edges,omitempty"`
	IndustryChainPhysicalConstraints []IndustryChainPhysicalConstraintSeed `json:"industry_chain_physical_constraints,omitempty"`
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

type SectorSourceMapping struct {
	SectorEntityKey            string `json:"sector_entity_key"`
	SourceSystem               string `json:"source_system"`
	SourceTaxonomyType         string `json:"source_taxonomy_type"`
	SourceSectorCode           string `json:"source_sector_code,omitempty"`
	SourceSectorName           string `json:"source_sector_name"`
	SourceSectorNameNormalized string `json:"source_sector_name_normalized,omitempty"`
	SourceMarketScope          string `json:"source_market_scope"`
	SourceURL                  string `json:"source_url,omitempty"`
	RankSnapshot               int    `json:"rank_snapshot,omitempty"`
	SnapshotDate               string `json:"snapshot_date,omitempty"`
	MappingStatus              string `json:"mapping_status,omitempty"`
	ReviewNote                 string `json:"review_note,omitempty"`
}

type Relationship struct {
	Key          string        `json:"key"`
	From         string        `json:"from"`
	To           string        `json:"to"`
	RelationType string        `json:"relation_type"`
	EvidenceNote string        `json:"evidence_note,omitempty"`
	SourceName   string        `json:"source_name"`
	SourceURL    string        `json:"source_url"`
	VerifiedAt   time.Time     `json:"verified_at"`
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
		merged.SectorSourceMappings = append(merged.SectorSourceMappings, manifest.SectorSourceMappings...)
		merged.Relationships = append(merged.Relationships, manifest.Relationships...)
		merged.IndustryChainMemberships = append(merged.IndustryChainMemberships, manifest.IndustryChainMemberships...)
		merged.IndustryChainTopologyEdges = append(merged.IndustryChainTopologyEdges, manifest.IndustryChainTopologyEdges...)
		merged.IndustryChainPhysicalConstraints = append(merged.IndustryChainPhysicalConstraints, manifest.IndustryChainPhysicalConstraints...)
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
	for _, entity := range manifest.Entities {
		if entity.EntityType == domain.EntityTypeSector {
			if err := validateSectorProfileReferences(entity.Profile, entityKeys); err != nil {
				return fmt.Errorf("entity %q profile: %w", entity.Key, err)
			}
		}
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
		if entityType == domain.EntityTypeSector {
			if err := validateSectorProfileReferences(profile.Data, entityKeys); err != nil {
				return fmt.Errorf("profile %q: %w", profile.EntityKey, err)
			}
		}
	}

	mappingIdentities := make(map[string]struct{}, len(manifest.SectorSourceMappings))
	for _, mapping := range manifest.SectorSourceMappings {
		normalized := normalizeSectorSourceMapping(mapping)
		entity, ok := entityKeys[normalized.SectorEntityKey]
		if !ok || entity.EntityType != domain.EntityTypeSector {
			return fmt.Errorf("sector source mapping references unknown sector %q", normalized.SectorEntityKey)
		}
		if err := validateSectorSourceMapping(normalized); err != nil {
			return err
		}
		identity := sectorSourceMappingIdentity(normalized)
		if _, exists := mappingIdentities[identity]; exists {
			return fmt.Errorf("duplicate sector source mapping identity %q", identity)
		}
		mappingIdentities[identity] = struct{}{}
	}

	relationshipKeys := make(map[string]struct{}, len(manifest.Relationships))
	relationshipTuples := make(map[string]struct{}, len(manifest.Relationships))
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
		if err := validateRelationshipPolicy(relationship, entityKeys); err != nil {
			return fmt.Errorf("relationship %q: %w", relationship.Key, err)
		}
		tuple := strings.Join([]string{relationship.From, strings.ToLower(relationship.RelationType), relationship.To}, "|")
		if _, ok := relationshipTuples[tuple]; ok {
			return fmt.Errorf("duplicate relationship tuple %q", tuple)
		}
		if relationship.Status != "" && relationship.Status != domain.StatusActive && relationship.Status != domain.StatusInactive {
			return fmt.Errorf("relationship %q unsupported status %q", relationship.Key, relationship.Status)
		}
		relationshipKeys[relationship.Key] = struct{}{}
		relationshipTuples[tuple] = struct{}{}
	}
	if err := validateIndustryChainManifest(manifest, entityKeys); err != nil {
		return err
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
	if err := node.Validate(); err != nil {
		return err
	}
	if entity.EntityType == domain.EntityTypeBenchmark {
		return validateBenchmarkSearchNames(entity)
	}
	if entity.EntityType == domain.EntityTypeSector && !strings.HasPrefix(entity.Key, "sector:ths_") {
		return validateSectorSearchNames(entity)
	}
	return nil
}

func validateSectorSearchNames(entity Entity) error {
	if !containsUnicodeScript(entity.Name, unicode.Han) || !containsUnicodeScript(entity.CanonicalName, unicode.Han) {
		return fmt.Errorf("sector name and canonical name must use Chinese primary names")
	}
	if !aliasesContainUnicodeScript(entity.Aliases, unicode.Latin) {
		return fmt.Errorf("sector with Chinese primary name requires an English alias")
	}
	return nil
}

func validateBenchmarkSearchNames(entity Entity) error {
	nameHasHan := containsUnicodeScript(entity.Name, unicode.Han)
	canonicalHasHan := containsUnicodeScript(entity.CanonicalName, unicode.Han)
	if nameHasHan || canonicalHasHan {
		if !nameHasHan || !canonicalHasHan {
			return fmt.Errorf("benchmark name and canonical name must use the same Chinese primary-name policy")
		}
		if !aliasesContainUnicodeScript(entity.Aliases, unicode.Latin) {
			return fmt.Errorf("benchmark with Chinese primary name requires an English alias")
		}
		return nil
	}
	if containsUnicodeScript(entity.Name, unicode.Latin) && containsUnicodeScript(entity.CanonicalName, unicode.Latin) {
		if !aliasesContainUnicodeScript(entity.Aliases, unicode.Han) {
			return fmt.Errorf("benchmark with English primary name requires a Chinese alias")
		}
		return nil
	}
	return fmt.Errorf("benchmark primary name must be Chinese or English")
}

func containsUnicodeScript(value string, script *unicode.RangeTable) bool {
	return strings.IndexFunc(value, func(r rune) bool { return unicode.Is(script, r) }) >= 0
}

func aliasesContainUnicodeScript(aliases []string, script *unicode.RangeTable) bool {
	for _, alias := range aliases {
		if containsUnicodeScript(alias, script) {
			return true
		}
	}
	return false
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
	if entityType == domain.EntityTypeSector {
		if err := validateSectorProfileFields(fields); err != nil {
			return err
		}
	}
	if entityType == domain.EntityTypeIndustryChain {
		if err := validateIndustryChainProfileFields(fields); err != nil {
			return err
		}
	}
	return nil
}

func validateSectorProfileFields(fields map[string]json.RawMessage) error {
	if raw, ok := fields["classification_code"]; ok && string(raw) != "null" {
		var value domain.SectorClassification
		if err := json.Unmarshal(raw, &value); err != nil || !validSectorClassification(value) {
			return fmt.Errorf("unsupported sector classification %q", value)
		}
	}
	if raw, ok := fields["review_status"]; ok && string(raw) != "null" {
		var value domain.SectorReviewStatus
		if err := json.Unmarshal(raw, &value); err != nil || !validSectorReviewStatus(value) {
			return fmt.Errorf("unsupported sector review status %q", value)
		}
	}
	return nil
}

func validateSectorProfileReferences(data json.RawMessage, entityKeys map[string]Entity) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for _, reference := range []struct {
		field      string
		entityType domain.EntityType
	}{
		{"primary_market_entity_id", domain.EntityTypeMarket},
		{"primary_economy_entity_id", domain.EntityTypeEconomy},
	} {
		raw, ok := fields[reference.field]
		if !ok || string(raw) == "null" {
			continue
		}
		var key string
		if err := json.Unmarshal(raw, &key); err != nil || strings.TrimSpace(key) == "" {
			continue
		}
		entity, exists := entityKeys[key]
		if !exists || entity.EntityType != reference.entityType {
			return fmt.Errorf("%s must reference %s", reference.field, reference.entityType)
		}
	}
	return nil
}

func validSectorClassification(value domain.SectorClassification) bool {
	switch value {
	case domain.SectorClassificationIndustry, domain.SectorClassificationTheme, domain.SectorClassificationMarket, domain.SectorClassificationStyle, domain.SectorClassificationRegion:
		return true
	default:
		return false
	}
}

func validSectorReviewStatus(value domain.SectorReviewStatus) bool {
	switch value {
	case domain.SectorReviewCandidate, domain.SectorReviewApproved, domain.SectorReviewRejected:
		return true
	default:
		return false
	}
}

func normalizeSectorSourceMapping(mapping SectorSourceMapping) SectorSourceMapping {
	mapping.SectorEntityKey = strings.TrimSpace(mapping.SectorEntityKey)
	mapping.SourceSystem = strings.ToLower(strings.TrimSpace(mapping.SourceSystem))
	mapping.SourceTaxonomyType = strings.ToLower(strings.TrimSpace(mapping.SourceTaxonomyType))
	mapping.SourceSectorCode = strings.TrimSpace(mapping.SourceSectorCode)
	mapping.SourceSectorName = strings.TrimSpace(mapping.SourceSectorName)
	mapping.SourceSectorNameNormalized = normalizeSectorSourceName(mapping.SourceSectorName)
	mapping.SourceMarketScope = strings.ToLower(strings.TrimSpace(mapping.SourceMarketScope))
	mapping.SourceURL = strings.TrimSpace(mapping.SourceURL)
	mapping.MappingStatus = strings.ToLower(strings.TrimSpace(mapping.MappingStatus))
	if mapping.MappingStatus == "" {
		mapping.MappingStatus = string(domain.SectorSourceMappingCandidate)
	}
	mapping.ReviewNote = strings.TrimSpace(mapping.ReviewNote)
	return mapping
}

func normalizeSectorSourceName(value string) string {
	normalized := norm.NFKC.String(strings.TrimSpace(value))
	return strings.ToLower(strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			return -1
		}
		return r
	}, normalized))
}

func validateSectorSourceMapping(mapping SectorSourceMapping) error {
	if mapping.SectorEntityKey == "" || mapping.SourceSystem == "" || mapping.SourceSectorName == "" || mapping.SourceSectorNameNormalized == "" {
		return fmt.Errorf("sector source mapping identity fields are required")
	}
	switch domain.SectorSourceTaxonomyType(mapping.SourceTaxonomyType) {
	case domain.SectorSourceTaxonomyConcept, domain.SectorSourceTaxonomyIndustry, domain.SectorSourceTaxonomyIndexSector:
	default:
		return fmt.Errorf("unsupported source taxonomy type %q", mapping.SourceTaxonomyType)
	}
	switch domain.SectorSourceMappingStatus(mapping.MappingStatus) {
	case domain.SectorSourceMappingCandidate, domain.SectorSourceMappingApproved, domain.SectorSourceMappingRejected, domain.SectorSourceMappingMerged:
	default:
		return fmt.Errorf("unsupported sector source mapping status %q", mapping.MappingStatus)
	}
	if mapping.SnapshotDate != "" {
		if _, err := time.Parse("2006-01-02", mapping.SnapshotDate); err != nil {
			return fmt.Errorf("invalid sector source mapping snapshot date %q", mapping.SnapshotDate)
		}
	}
	return nil
}

func sectorSourceMappingIdentity(mapping SectorSourceMapping) string {
	if mapping.SourceSectorCode != "" {
		return strings.Join([]string{mapping.SourceSystem, mapping.SourceTaxonomyType, mapping.SourceSectorCode}, "|")
	}
	return strings.Join([]string{mapping.SourceSystem, mapping.SourceTaxonomyType, mapping.SourceSectorNameNormalized, mapping.SourceMarketScope}, "|")
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
	case domain.EntityTypeBenchmark:
		return []string{"benchmark_type", "provider", "currency_code", "unit", "frequency", "source_url"}
	case domain.EntityTypeSector:
		return []string{"sector_system", "sector_type"}
	case domain.EntityTypeIndustryChain:
		return []string{"chain_code", "definition", "scope_type", "review_status", "source_name", "source_url", "verified_at"}
	case domain.EntityTypeChainNode:
		return nil
	case domain.EntityTypeTheme:
		return []string{"definition", "boundary_note"}
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
	for i := range manifest.SectorSourceMappings {
		manifest.SectorSourceMappings[i] = normalizeSectorSourceMapping(manifest.SectorSourceMappings[i])
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
		"selection_score",
		"runtime_tier",
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
