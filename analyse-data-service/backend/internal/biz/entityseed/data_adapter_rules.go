package entityseed

import (
	"encoding/json"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"
)

// The exported functions in this file are the business-rule seam used by the
// PostgreSQL adapter. They keep normalization and validation in Biz without
// exposing database concepts to this package.

func NormalizeExternalIdentifier(identifier model.EntityExternalIdentifier) model.EntityExternalIdentifier {
	return normalizeExternalIdentifier(identifier)
}

func ValidateFirstBatchExternalIdentifier(identifier model.EntityExternalIdentifier) error {
	return validateFirstBatchExternalIdentifier(identifier)
}

func ExternalIdentifierIdentity(sourceSystem, taxonomy, code string) string {
	return externalIdentifierIdentity(sourceSystem, taxonomy, code)
}

func ExternalIdentifierSeedUUID(identity string) string {
	return externalIdentifierSeedUUID(identity)
}

func NormalizeSectorSourceMapping(mapping SectorSourceMapping) SectorSourceMapping {
	return normalizeSectorSourceMapping(mapping)
}

func ValidateSectorSourceMapping(mapping SectorSourceMapping) error {
	return validateSectorSourceMapping(mapping)
}

func SectorSourceMappingIdentity(mapping SectorSourceMapping) string {
	return sectorSourceMappingIdentity(mapping)
}

func ValidateEntity(entity Entity) error {
	return validateEntity(entity)
}

func ValidateProfileData(entityType model.EntityType, data json.RawMessage) error {
	return validateProfileData(entityType, data)
}

func ValidateRelationshipProvenance(relationship Relationship) error {
	return validateRelationshipProvenance(relationship)
}

func ValidateRelationshipPolicy(relationship Relationship, entities map[string]Entity) error {
	return validateRelationshipPolicy(relationship, entities)
}

func PlanConvergenceRelationship(relationship Relationship, entities map[string]Entity, legacyKey, targetKey string) (Relationship, bool, error) {
	planned, disposition, err := planConvergenceRelationship(relationship, entities, legacyKey, targetKey)
	return planned, disposition == convergenceEdgeDeactivate, err
}
