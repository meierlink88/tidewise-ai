package seed

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

func ValidateProductionManifest(manifest Manifest) error {
	for _, entity := range manifest.Entities {
		if entity.EntityType == domain.EntityTypeSector || entity.EntityType == domain.EntityTypeIndustryChain {
			return fmt.Errorf("retired entity type %q is migration input only", entity.EntityType)
		}
		if entity.EntityType == domain.EntityTypeChainNode || entity.EntityType == domain.EntityTypeTheme {
			if err := validateUnifiedProfile(entity.EntityType, entity.Profile); err != nil {
				return fmt.Errorf("entity %q profile: %w", entity.Key, err)
			}
		}
	}
	for _, profile := range manifest.Profiles {
		if profile.EntityType == domain.EntityTypeChainNode || profile.EntityType == domain.EntityTypeTheme {
			if err := validateUnifiedProfile(profile.EntityType, profile.Data); err != nil {
				return fmt.Errorf("profile %q: %w", profile.EntityKey, err)
			}
		}
	}
	if len(manifest.SectorSourceMappings) > 0 {
		return fmt.Errorf("sector source mappings are migration input only")
	}
	if len(manifest.IndustryChainMemberships) > 0 || len(manifest.IndustryChainTopologyEdges) > 0 || len(manifest.IndustryChainPhysicalConstraints) > 0 {
		return fmt.Errorf("legacy industry chain structures are migration input only")
	}
	for _, relationship := range manifest.Relationships {
		if strings.HasPrefix(relationship.From, "sector:") || strings.HasPrefix(relationship.To, "sector:") ||
			strings.HasPrefix(relationship.From, "industry_chain:") || strings.HasPrefix(relationship.To, "industry_chain:") {
			return fmt.Errorf("relationship %q references a retired industry identity", relationship.Key)
		}
		switch relationship.RelationType {
		case "covers_sector", "tracked_by_benchmark", "mapped_to_sector", "scoped_to_economy", "observed_by_benchmark":
			return fmt.Errorf("relationship %q uses retired industry relation type %q", relationship.Key, relationship.RelationType)
		}
	}
	return nil
}

func validateUnifiedProfile(entityType domain.EntityType, data []byte) error {
	if err := validateProfileData(entityType, data); err != nil {
		return err
	}
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	allowed := map[string]struct{}{"definition": {}, "boundary_note": {}}
	for field := range fields {
		if _, ok := allowed[field]; !ok {
			return fmt.Errorf("legacy field %q is forbidden", field)
		}
	}
	definition, ok := fields["definition"]
	if !ok || definition == nil || strings.TrimSpace(fmt.Sprint(definition)) == "" {
		return fmt.Errorf("definition is required")
	}
	if entityType == domain.EntityTypeChainNode {
		if value, ok := fields["boundary_note"]; ok && value != nil && strings.TrimSpace(fmt.Sprint(value)) == "" {
			return fmt.Errorf("boundary_note must be nonblank when present")
		}
	}
	return nil
}
