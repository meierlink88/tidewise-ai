package domain

func validStatus[T comparable](value T, allowed ...T) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func validEntityType(value EntityType) bool {
	return validStatus(
		value,
		EntityTypeAllianceOrg,
		EntityTypeEconomy,
		EntityTypePolicyBody,
		EntityTypeMarket,
		EntityTypeIndex,
		EntityTypeBenchmark,
		EntityTypeSector,
		EntityTypeIndustryChain,
		EntityTypeChainNode,
		EntityTypeTheme,
		EntityTypeCompany,
		EntityTypeSecurity,
		EntityTypeInstrument,
		EntityTypeMetric,
		EntityTypeCommodity,
		EntityTypePerson,
	)
}
