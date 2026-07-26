package entityseed

import (
	"fmt"

	bizidentity "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/identity"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"
)

func entitySeedUUID(key string) string {
	return bizidentity.NormalizeUUID("entity", key)
}

func profileTableName(entityType model.EntityType) (string, error) {
	tables := map[model.EntityType]string{
		model.EntityTypeAllianceOrg:   "alliance_org_profiles",
		model.EntityTypeEconomy:       "economy_profiles",
		model.EntityTypePolicyBody:    "policy_body_profiles",
		model.EntityTypeMarket:        "market_profiles",
		model.EntityTypeIndex:         "index_profiles",
		model.EntityTypeBenchmark:     "benchmark_profiles",
		model.EntityTypeSector:        "sector_profiles",
		model.EntityTypeIndustryChain: "industry_chain_profiles",
		model.EntityTypeChainNode:     "chain_node_profiles",
		model.EntityTypeTheme:         "theme_profiles",
		model.EntityTypeCompany:       "company_profiles",
		model.EntityTypeSecurity:      "security_profiles",
		model.EntityTypeInstrument:    "instrument_profiles",
		model.EntityTypeMetric:        "metric_profiles",
		model.EntityTypeCommodity:     "commodity_profiles",
		model.EntityTypePerson:        "person_profiles",
	}
	table, ok := tables[entityType]
	if !ok {
		return "", fmt.Errorf("unsupported profile entity type %q", entityType)
	}
	return table, nil
}
