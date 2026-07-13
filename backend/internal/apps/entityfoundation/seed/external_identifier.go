package seed

import (
	"fmt"
	"strings"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	repoids "github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

const (
	ExternalSourceEastmoney  = "eastmoney"
	ExternalSourceTHS        = "ths"
	ExternalTaxonomyIndustry = "industry_sector"
	ExternalTaxonomyConcept  = "concept_sector"
	ExternalTaxonomyIndex    = "index_sector"
)

func normalizeExternalIdentifier(identifier domain.EntityExternalIdentifier) domain.EntityExternalIdentifier {
	identifier.ID = strings.ToLower(strings.TrimSpace(identifier.ID))
	identifier.EntityID = strings.ToLower(strings.TrimSpace(identifier.EntityID))
	identifier.SourceSystem = strings.ToLower(strings.TrimSpace(identifier.SourceSystem))
	identifier.SourceTaxonomyType = strings.ToLower(strings.TrimSpace(identifier.SourceTaxonomyType))
	identifier.ExternalCode = strings.TrimSpace(identifier.ExternalCode)
	identifier.ExternalName = strings.TrimSpace(identifier.ExternalName)
	if identifier.Status == "" {
		identifier.Status = domain.StatusActive
	}
	return identifier
}

func validateFirstBatchExternalIdentifier(identifier domain.EntityExternalIdentifier) error {
	identifier = normalizeExternalIdentifier(identifier)
	if err := identifier.Validate(); err != nil {
		return err
	}
	switch identifier.SourceSystem {
	case ExternalSourceEastmoney, ExternalSourceTHS:
	default:
		return fmt.Errorf("unsupported first-batch source system %q", identifier.SourceSystem)
	}
	switch identifier.SourceTaxonomyType {
	case ExternalTaxonomyIndustry, ExternalTaxonomyConcept, ExternalTaxonomyIndex:
	default:
		return fmt.Errorf("unresolved or unsupported first-batch taxonomy %q", identifier.SourceTaxonomyType)
	}
	identity := externalIdentifierIdentity(identifier.SourceSystem, identifier.SourceTaxonomyType, identifier.ExternalCode)
	if identifier.ID != externalIdentifierSeedUUID(identity) {
		return fmt.Errorf("external identifier id does not match its stable identity")
	}
	if !repoids.IsUUID(identifier.EntityID) {
		return fmt.Errorf("external identifier entity id must be a UUID")
	}
	return nil
}

func externalIdentifierIdentity(sourceSystem, taxonomy, code string) string {
	return strings.Join([]string{
		strings.ToLower(strings.TrimSpace(sourceSystem)),
		strings.ToLower(strings.TrimSpace(taxonomy)),
		strings.TrimSpace(code),
	}, "|")
}

func externalIdentifierSeedUUID(identity string) string {
	return repoids.NormalizeUUID("entity_external_identifier", strings.TrimSpace(identity))
}
