package seed

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type relationshipTypePolicy struct {
	from []domain.EntityType
	to   []domain.EntityType
}

var relationshipPolicies = map[string]relationshipTypePolicy{
	"member_of":             {from: []domain.EntityType{domain.EntityTypeEconomy}, to: []domain.EntityType{domain.EntityTypeAllianceOrg}},
	"has_market":            {from: []domain.EntityType{domain.EntityTypeEconomy}, to: []domain.EntityType{domain.EntityTypeMarket}},
	"tracks_index":          {from: []domain.EntityType{domain.EntityTypeMarket}, to: []domain.EntityType{domain.EntityTypeIndex}},
	"observes_benchmark":    {from: []domain.EntityType{domain.EntityTypeMarket}, to: []domain.EntityType{domain.EntityTypeBenchmark}},
	"covers_sector":         {from: []domain.EntityType{domain.EntityTypeMarket}, to: []domain.EntityType{domain.EntityTypeSector}},
	"tracked_by_benchmark":  {from: []domain.EntityType{domain.EntityTypeSector}, to: []domain.EntityType{domain.EntityTypeBenchmark}},
	"measures":              {from: []domain.EntityType{domain.EntityTypeBenchmark}, to: []domain.EntityType{domain.EntityTypeMetric}},
	"references":            {from: []domain.EntityType{domain.EntityTypeBenchmark}, to: []domain.EntityType{domain.EntityTypeCommodity, domain.EntityTypeInstrument}},
	"issues":                {from: []domain.EntityType{domain.EntityTypeCompany}, to: []domain.EntityType{domain.EntityTypeSecurity}},
	"participates_in":       {from: []domain.EntityType{domain.EntityTypeCompany}, to: []domain.EntityType{domain.EntityTypeChainNode}},
	"affiliated_with":       {from: []domain.EntityType{domain.EntityTypePerson}, to: []domain.EntityType{domain.EntityTypePolicyBody, domain.EntityTypeCompany}},
	"applies_to":            {from: []domain.EntityType{domain.EntityTypeMetric}, to: []domain.EntityType{domain.EntityTypeInstrument, domain.EntityTypeCommodity, domain.EntityTypeChainNode}},
	"scoped_to_economy":     {from: []domain.EntityType{domain.EntityTypeIndustryChain}, to: []domain.EntityType{domain.EntityTypeEconomy}},
	"uses_commodity":        {from: []domain.EntityType{domain.EntityTypeChainNode}, to: []domain.EntityType{domain.EntityTypeCommodity}},
	"produces_commodity":    {from: []domain.EntityType{domain.EntityTypeChainNode}, to: []domain.EntityType{domain.EntityTypeCommodity}},
	"observed_by_benchmark": {from: []domain.EntityType{domain.EntityTypeIndustryChain, domain.EntityTypeChainNode}, to: []domain.EntityType{domain.EntityTypeBenchmark}},
	"mapped_to_sector":      {from: []domain.EntityType{domain.EntityTypeIndustryChain, domain.EntityTypeChainNode}, to: []domain.EntityType{domain.EntityTypeSector}},
}

func validateRelationshipPolicy(relationship Relationship, entities map[string]Entity) error {
	if relationship.From == relationship.To {
		return fmt.Errorf("self relationship is not allowed")
	}
	policy, ok := relationshipPolicies[strings.ToLower(strings.TrimSpace(relationship.RelationType))]
	if !ok {
		return fmt.Errorf("unsupported relationship type %q", relationship.RelationType)
	}
	from, ok := entities[relationship.From]
	if !ok {
		return fmt.Errorf("unknown relationship source %q", relationship.From)
	}
	to, ok := entities[relationship.To]
	if !ok {
		return fmt.Errorf("unknown relationship target %q", relationship.To)
	}
	if !containsEntityType(policy.from, from.EntityType) || !containsEntityType(policy.to, to.EntityType) {
		return fmt.Errorf("relationship type %q does not allow %q -> %q", relationship.RelationType, from.EntityType, to.EntityType)
	}
	if strings.EqualFold(relationship.RelationType, "covers_sector") && overseasMarketCoversChinaSector(from, to) {
		return fmt.Errorf("overseas market cannot cover China sector")
	}
	if err := validateRelationshipProvenance(relationship); err != nil {
		return err
	}
	if containsForbiddenRelationshipText(relationship.EvidenceNote) {
		return fmt.Errorf("relationship evidence contains forbidden reasoning text")
	}
	return nil
}

func overseasMarketCoversChinaSector(from, to Entity) bool {
	var marketProfile, sectorProfile map[string]any
	if json.Unmarshal(from.Profile, &marketProfile) != nil || json.Unmarshal(to.Profile, &sectorProfile) != nil {
		return false
	}
	marketEconomy, _ := marketProfile["economy_entity_id"].(string)
	sectorEconomy, _ := sectorProfile["primary_economy_entity_id"].(string)
	return sectorEconomy == "economy:cn" && marketEconomy != "" && marketEconomy != "economy:cn"
}

func validateRelationshipProvenance(relationship Relationship) error {
	if strings.TrimSpace(relationship.SourceName) == "" {
		return fmt.Errorf("source name is required")
	}
	parsed, err := url.ParseRequestURI(strings.TrimSpace(relationship.SourceURL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("valid source URL is required")
	}
	if relationship.VerifiedAt.IsZero() {
		return fmt.Errorf("verified at is required")
	}
	return nil
}

func containsEntityType(values []domain.EntityType, target domain.EntityType) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsForbiddenRelationshipText(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	for _, forbidden := range []string{
		"bullish", "bearish", "benefit", "pressure", "prediction", "investment advice",
		"利好", "利空", "受益", "承压", "预测", "投资建议", "传导强度", "事件评分",
	} {
		if strings.Contains(normalized, strings.ToLower(forbidden)) {
			return true
		}
	}
	return false
}
