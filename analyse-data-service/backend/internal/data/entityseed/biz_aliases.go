package entityseed

import (
	"database/sql"
	"encoding/json"
	"sync"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/entityseed"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"
)

type Entity = biz.Entity
type Profile = biz.Profile
type Relationship = biz.Relationship
type SectorSourceMapping = biz.SectorSourceMapping
type Manifest = biz.Manifest
type Repository = biz.Repository
type WriteAction = biz.WriteAction
type WriteResult = biz.WriteResult
type SectorConvergence = biz.SectorConvergence
type SectorConvergenceManifest = biz.SectorConvergenceManifest
type SectorConvergenceMode = biz.SectorConvergenceMode
type SectorConvergenceReport = biz.SectorConvergenceReport
type AllianceEconomyManifest = biz.AllianceEconomyManifest
type ExternalIdentifierMapping = biz.ExternalIdentifierMapping
type ExternalIdentifierBatchReport = biz.ExternalIdentifierBatchReport
type ExternalIdentifierMappingManifest = biz.ExternalIdentifierMappingManifest
type ChainNodeRelationManifest = biz.ChainNodeRelationManifest
type ChainNodeRelationReport = biz.ChainNodeRelationReport
type ChainNodeRelationDataPreflightReport = biz.ChainNodeRelationDataPreflightReport
type IndustryChainBatch = biz.IndustryChainBatch
type IndustryChainWriteReport = biz.IndustryChainWriteReport
type AllianceEconomyDependencyCount = biz.AllianceEconomyDependencyCount
type AllianceEconomyForeignKey = biz.AllianceEconomyForeignKey
type AllianceEconomyDependencyReport = biz.AllianceEconomyDependencyReport
type AllianceEconomyCleanupResult = biz.AllianceEconomyCleanupResult
type AllianceEconomyRebuildResult = biz.AllianceEconomyRebuildResult
type allianceEconomyRebuildPreflight = biz.AllianceEconomyRebuildPreflight

const (
	WriteCreated                              = biz.WriteCreated
	WriteUpdated                              = biz.WriteUpdated
	WriteUnchanged                            = biz.WriteUnchanged
	SectorConvergenceModeInitial              = biz.SectorConvergenceModeInitial
	SectorConvergenceModeCorrection           = biz.SectorConvergenceModeCorrection
	SectorConvergenceReplaceWithExistingIndex = biz.SectorConvergenceReplaceWithExistingIndex
	ExternalSourceEastmoney                   = biz.ExternalSourceEastmoney
	ExternalTaxonomyConcept                   = biz.ExternalTaxonomyConcept
)

func NewRepository(db *sql.DB) PostgresRepository {
	return NewPostgresRepository(db)
}

var _ biz.Repository = PostgresRepository{}

type MemoryRepository struct {
	*biz.MemoryRepository
	mu                               sync.Mutex
	industryChainProfiles            map[string]model.IndustryChainProfile
	industryChainMemberships         map[string]model.IndustryChainMembership
	industryChainTopologyEdges       map[string]model.IndustryChainTopologyEdge
	industryChainPhysicalConstraints map[string]model.IndustryChainPhysicalConstraint
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		MemoryRepository:                 biz.NewMemoryRepository(),
		industryChainProfiles:            map[string]model.IndustryChainProfile{},
		industryChainMemberships:         map[string]model.IndustryChainMembership{},
		industryChainTopologyEdges:       map[string]model.IndustryChainTopologyEdge{},
		industryChainPhysicalConstraints: map[string]model.IndustryChainPhysicalConstraint{},
	}
}

func normalizeExternalIdentifier(identifier model.EntityExternalIdentifier) model.EntityExternalIdentifier {
	return biz.NormalizeExternalIdentifier(identifier)
}

func validateFirstBatchExternalIdentifier(identifier model.EntityExternalIdentifier) error {
	return biz.ValidateFirstBatchExternalIdentifier(identifier)
}

func externalIdentifierIdentity(sourceSystem, taxonomy, code string) string {
	return biz.ExternalIdentifierIdentity(sourceSystem, taxonomy, code)
}

func externalIdentifierSeedUUID(identity string) string {
	return biz.ExternalIdentifierSeedUUID(identity)
}

func mappingFromIdentifier(item model.EntityExternalIdentifier) ExternalIdentifierMapping {
	return biz.ExternalIdentifierMappingFromIdentifier(item)
}

func externalIdentifierFromMapping(mapping ExternalIdentifierMapping) model.EntityExternalIdentifier {
	return biz.ExternalIdentifierFromMapping(mapping)
}

func normalizeAndValidateExternalIdentifierMappings(mappings []ExternalIdentifierMapping) ([]ExternalIdentifierMapping, error) {
	return biz.NormalizeAndValidateExternalIdentifierMappings(mappings)
}

func LoadExternalIdentifierMappingFile(path string) (ExternalIdentifierMappingManifest, error) {
	return biz.LoadExternalIdentifierMappingFile(path)
}

func ValidateExternalIdentifierMappingFile(path string) (ExternalIdentifierBatchReport, error) {
	return biz.ValidateExternalIdentifierMappingFile(path)
}

func ValidateFrozenFirstBatchExternalIdentifierManifest(path string, mappings []ExternalIdentifierMapping) error {
	return biz.ValidateFrozenFirstBatchExternalIdentifierManifest(path, mappings)
}

func validateIndustryChainBatch(batch IndustryChainBatch) error {
	return biz.ValidateIndustryChainBatch(batch)
}

func buildAllianceEconomyDependencyReport(counts []AllianceEconomyDependencyCount, foreignKeys []AllianceEconomyForeignKey, fingerprints, protected []string) (AllianceEconomyDependencyReport, error) {
	return biz.BuildAllianceEconomyDependencyReport(counts, foreignKeys, fingerprints, protected)
}

func allianceEconomyRebuildPayloads(manifest AllianceEconomyManifest) ([]byte, []byte, []byte, error) {
	return biz.AllianceEconomyRebuildPayloads(manifest)
}

func allianceEconomyFingerprintChecksum(fingerprints []string) string {
	return biz.AllianceEconomyFingerprintChecksum(fingerprints)
}

func isTopologyOnlyBatch(batch IndustryChainBatch) bool {
	return biz.IsTopologyOnlyIndustryChainBatch(batch)
}

func isConstraintOnlyBatch(batch IndustryChainBatch) bool {
	return biz.IsConstraintOnlyIndustryChainBatch(batch)
}

func validateConstraintsAgainstPersistedSubjects(constraints []model.IndustryChainPhysicalConstraint, memberships map[string]model.IndustryChainMembership, topology map[string]model.IndustryChainTopologyEdge) error {
	return biz.ValidateIndustryChainConstraintsAgainstPersistedSubjects(constraints, memberships, topology)
}

func validateTopologyAgainstPersistedMemberships(edges []model.IndustryChainTopologyEdge, memberships map[string]model.IndustryChainMembership) error {
	return biz.ValidateIndustryChainTopologyAgainstPersistedMemberships(edges, memberships)
}

func normalizeSectorSourceMapping(mapping SectorSourceMapping) SectorSourceMapping {
	return biz.NormalizeSectorSourceMapping(mapping)
}

func validateSectorSourceMapping(mapping SectorSourceMapping) error {
	return biz.ValidateSectorSourceMapping(mapping)
}

func sectorSourceMappingIdentity(mapping SectorSourceMapping) string {
	return biz.SectorSourceMappingIdentity(mapping)
}

func validateEntity(entity Entity) error {
	return biz.ValidateEntity(entity)
}

func validateProfileData(entityType model.EntityType, data json.RawMessage) error {
	return biz.ValidateProfileData(entityType, data)
}

func validateRelationshipProvenance(relationship Relationship) error {
	return biz.ValidateRelationshipProvenance(relationship)
}

func validateRelationshipPolicy(relationship Relationship, entities map[string]Entity) error {
	return biz.ValidateRelationshipPolicy(relationship, entities)
}

type convergenceEdgeDisposition string

const convergenceEdgeDeactivate convergenceEdgeDisposition = "deactivate"

func planConvergenceRelationship(relationship Relationship, entities map[string]Entity, legacyKey, targetKey string) (Relationship, convergenceEdgeDisposition, error) {
	planned, deactivate, err := biz.PlanConvergenceRelationship(relationship, entities, legacyKey, targetKey)
	if deactivate {
		return planned, convergenceEdgeDeactivate, err
	}
	return planned, "", err
}
