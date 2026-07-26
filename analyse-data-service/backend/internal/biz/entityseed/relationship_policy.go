package entityseed

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"
)

type relationshipTypePolicy struct {
	from []model.EntityType
	to   []model.EntityType
}

var relationshipPolicies = map[string]relationshipTypePolicy{
	"member_of":             {from: []model.EntityType{model.EntityTypeEconomy}, to: []model.EntityType{model.EntityTypeAllianceOrg}},
	"has_market":            {from: []model.EntityType{model.EntityTypeEconomy}, to: []model.EntityType{model.EntityTypeMarket}},
	"tracks_index":          {from: []model.EntityType{model.EntityTypeMarket}, to: []model.EntityType{model.EntityTypeIndex}},
	"observes_benchmark":    {from: []model.EntityType{model.EntityTypeMarket}, to: []model.EntityType{model.EntityTypeBenchmark}},
	"covers_sector":         {from: []model.EntityType{model.EntityTypeMarket}, to: []model.EntityType{model.EntityTypeSector}},
	"tracked_by_benchmark":  {from: []model.EntityType{model.EntityTypeSector}, to: []model.EntityType{model.EntityTypeBenchmark}},
	"measures":              {from: []model.EntityType{model.EntityTypeBenchmark}, to: []model.EntityType{model.EntityTypeMetric}},
	"references":            {from: []model.EntityType{model.EntityTypeBenchmark}, to: []model.EntityType{model.EntityTypeCommodity, model.EntityTypeInstrument}},
	"issues":                {from: []model.EntityType{model.EntityTypeCompany}, to: []model.EntityType{model.EntityTypeSecurity}},
	"participates_in":       {from: []model.EntityType{model.EntityTypeCompany}, to: []model.EntityType{model.EntityTypeChainNode}},
	"affiliated_with":       {from: []model.EntityType{model.EntityTypePerson}, to: []model.EntityType{model.EntityTypePolicyBody, model.EntityTypeCompany}},
	"applies_to":            {from: []model.EntityType{model.EntityTypeMetric}, to: []model.EntityType{model.EntityTypeInstrument, model.EntityTypeCommodity, model.EntityTypeChainNode}},
	"scoped_to_economy":     {from: []model.EntityType{model.EntityTypeIndustryChain}, to: []model.EntityType{model.EntityTypeEconomy}},
	"uses_commodity":        {from: []model.EntityType{model.EntityTypeChainNode}, to: []model.EntityType{model.EntityTypeCommodity}},
	"produces_commodity":    {from: []model.EntityType{model.EntityTypeChainNode}, to: []model.EntityType{model.EntityTypeCommodity}},
	"observed_by_benchmark": {from: []model.EntityType{model.EntityTypeIndustryChain, model.EntityTypeChainNode}, to: []model.EntityType{model.EntityTypeBenchmark}},
	"mapped_to_sector":      {from: []model.EntityType{model.EntityTypeIndustryChain, model.EntityTypeChainNode}, to: []model.EntityType{model.EntityTypeSector}},
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

func containsEntityType(values []model.EntityType, target model.EntityType) bool {
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
