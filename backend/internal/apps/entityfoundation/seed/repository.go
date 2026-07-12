package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type WriteAction string

const (
	WriteCreated   WriteAction = "created"
	WriteUpdated   WriteAction = "updated"
	WriteUnchanged WriteAction = "unchanged"
)

type WriteResult struct {
	Key    string
	Action WriteAction
}

type Repository interface {
	UpsertEntity(context.Context, Entity) (WriteResult, error)
	UpsertProfile(context.Context, Profile) (WriteResult, error)
	UpsertSectorSourceMapping(context.Context, SectorSourceMapping) (WriteResult, error)
	UpsertRelationship(context.Context, Relationship) (WriteResult, error)
	HasActiveLegacySectors(context.Context) (bool, error)
}

type MemoryRepository struct {
	mu                           sync.Mutex
	entities                     map[string]Entity
	profiles                     map[string]Profile
	sectorSourceMappings         map[string]SectorSourceMapping
	relationships                map[string]Relationship
	convergenceManifests         map[int64]string
	convergenceReviews           map[int64]string
	convergenceAudits            map[string]SectorConvergence
	convergenceRelationshipMoves map[string][]string
	convergenceOwnedAliases      map[string]map[string]struct{}
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		entities:                     map[string]Entity{},
		profiles:                     map[string]Profile{},
		sectorSourceMappings:         map[string]SectorSourceMapping{},
		relationships:                map[string]Relationship{},
		convergenceManifests:         map[int64]string{},
		convergenceReviews:           map[int64]string{},
		convergenceAudits:            map[string]SectorConvergence{},
		convergenceRelationshipMoves: map[string][]string{},
		convergenceOwnedAliases:      map[string]map[string]struct{}{},
	}
}

func (r *MemoryRepository) HasActiveLegacySectors(_ context.Context) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, entity := range r.entities {
		if strings.HasPrefix(key, "sector:ths_") && entity.Status != domain.StatusInactive {
			return true, nil
		}
	}
	return false, nil
}

func (r *MemoryRepository) ApplySectorConvergence(_ context.Context, seedManifest Manifest, manifest SectorConvergenceManifest, mode SectorConvergenceMode) (SectorConvergenceReport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.convergenceManifests[manifest.ManifestVersion]; ok {
		if existing != manifest.ManifestChecksum {
			return SectorConvergenceReport{}, fmt.Errorf("same manifest version has different payload")
		}
		return SectorConvergenceReport{ManifestVersion: manifest.ManifestVersion, AuditUnchanged: len(manifest.Convergences)}, nil
	}
	if mode == SectorConvergenceModeInitial && len(r.convergenceManifests) > 0 {
		return SectorConvergenceReport{}, fmt.Errorf("initial convergence already applied")
	}
	current := int64(0)
	for version := range r.convergenceManifests {
		if version > current {
			current = version
		}
	}
	if mode == SectorConvergenceModeCorrection {
		if manifest.PreviousManifestVersion == nil || *manifest.PreviousManifestVersion != current || manifest.ManifestVersion <= current {
			return SectorConvergenceReport{}, fmt.Errorf("invalid correction manifest version")
		}
		if manifest.ReviewSourceURL+manifest.ReviewedAt.String() == r.convergenceReviews[current] {
			return SectorConvergenceReport{}, fmt.Errorf("correction requires a new human Review")
		}
	}
	entities := make(map[string]Entity, len(r.entities))
	for key, value := range r.entities {
		entities[key] = value
	}
	profiles := make(map[string]Profile, len(r.profiles))
	for key, value := range r.profiles {
		profiles[key] = value
	}
	mappings := make(map[string]SectorSourceMapping, len(r.sectorSourceMappings)+29)
	for key, value := range r.sectorSourceMappings {
		mappings[key] = value
	}
	relationships := make(map[string]Relationship, len(r.relationships))
	for key, value := range r.relationships {
		relationships[key] = value
	}
	for _, entity := range seedManifest.Entities {
		if err := validateEntity(entity); err != nil {
			return SectorConvergenceReport{}, err
		}
		entities[entity.Key] = entity
		if len(entity.Profile) > 0 {
			profiles[entity.Key] = Profile{EntityKey: entity.Key, EntityType: entity.EntityType, Data: entity.Profile}
		}
	}
	for _, profile := range seedManifest.Profiles {
		profiles[profile.EntityKey] = profile
	}
	for _, mapping := range seedManifest.SectorSourceMappings {
		mapping = normalizeSectorSourceMapping(mapping)
		if err := validateSectorSourceMapping(mapping); err != nil {
			return SectorConvergenceReport{}, err
		}
		mappings[sectorSourceMappingIdentity(mapping)] = mapping
	}
	for _, relationship := range seedManifest.Relationships {
		relationships[relationship.Key] = relationship
	}
	audits := make(map[string]SectorConvergence, len(r.convergenceAudits)+len(manifest.Convergences))
	for key, value := range r.convergenceAudits {
		audits[key] = value
	}
	relationshipMoves := cloneStringSliceMap(r.convergenceRelationshipMoves)
	ownedAliases := cloneStringSetMap(r.convergenceOwnedAliases)
	report := SectorConvergenceReport{ManifestVersion: manifest.ManifestVersion, RetiredLegacy: len(manifest.Convergences), AuditCreated: len(manifest.Convergences)}
	for _, item := range manifest.Convergences {
		auditKey := item.LegacyEntityKey + "|" + fmt.Sprint(manifest.ManifestVersion)
		legacy, ok := entities[item.LegacyEntityKey]
		if !ok || legacy.EntityType != domain.EntityTypeSector {
			return SectorConvergenceReport{}, fmt.Errorf("unknown legacy sector %q", item.LegacyEntityKey)
		}
		if item.TargetEntityKey != "" {
			target, ok := entities[item.TargetEntityKey]
			if !ok || target.EntityType != item.TargetEntityType {
				return SectorConvergenceReport{}, fmt.Errorf("invalid convergence target %q", item.TargetEntityKey)
			}
		}
		if item.TargetEntityType != domain.EntityTypeSector {
			for _, mapping := range mappings {
				if mapping.SectorEntityKey == item.LegacyEntityKey {
					return SectorConvergenceReport{ManifestVersion: manifest.ManifestVersion, BlockedReferences: 1}, fmt.Errorf("sector-only source mapping blocks convergence of %q to %q", item.LegacyEntityKey, item.TargetEntityType)
				}
			}
			for _, profile := range profiles {
				if profileReferencesEntity(profile.Data, "parent_sector_entity_id", item.LegacyEntityKey) {
					return SectorConvergenceReport{ManifestVersion: manifest.ManifestVersion, BlockedReferences: 1}, fmt.Errorf("sector-only parent reference blocks convergence of %q to %q", item.LegacyEntityKey, item.TargetEntityType)
				}
			}
		}
		if mode == SectorConvergenceModeCorrection && legacy.Status != domain.StatusInactive {
			return SectorConvergenceReport{}, fmt.Errorf("correction drift for legacy sector %q", item.LegacyEntityKey)
		}
		if mode == SectorConvergenceModeCorrection {
			previousKey := item.LegacyEntityKey + "|" + fmt.Sprint(current)
			previous, ok := audits[previousKey]
			if !ok {
				return SectorConvergenceReport{}, fmt.Errorf("missing previous convergence audit for %q", item.LegacyEntityKey)
			}
			if previous.TargetEntityType == domain.EntityTypeSector && previous.TargetEntityKey != item.TargetEntityKey {
				if owned := ownedAliases[previous.TargetEntityKey]; owned != nil {
					delete(owned, previous.LegacyName)
					oldTarget := entities[previous.TargetEntityKey]
					oldTarget.Aliases = removeAlias(oldTarget.Aliases, previous.LegacyName)
					entities[previous.TargetEntityKey] = oldTarget
				}
			}
			sourceKey := previous.TargetEntityKey
			if sourceKey == "" {
				sourceKey = item.LegacyEntityKey
			}
			for _, key := range relationshipMoves[previousKey] {
				relationship, ok := relationships[key]
				if !ok {
					return SectorConvergenceReport{}, fmt.Errorf("recorded relationship %q is missing", key)
				}
				expectedStatus := domain.StatusActive
				if previous.TargetEntityKey == "" {
					expectedStatus = domain.StatusInactive
				}
				if relationship.Status != expectedStatus || (relationship.From != sourceKey && relationship.To != sourceKey) {
					return SectorConvergenceReport{}, fmt.Errorf("recorded relationship %q drifted", key)
				}
				planned, _, err := planConvergenceRelationship(relationship, entities, sourceKey, item.TargetEntityKey)
				if err != nil {
					return SectorConvergenceReport{ManifestVersion: manifest.ManifestVersion, BlockedReferences: 1}, fmt.Errorf("recorded relationship %q is incompatible: %w", key, err)
				}
				relationships[key] = planned
				relationshipMoves[auditKey] = append(relationshipMoves[auditKey], key)
				report.ReferencesMoved++
			}
		}
		legacy.Status = domain.StatusInactive
		entities[item.LegacyEntityKey] = legacy
		if item.TargetEntityType == domain.EntityTypeSector {
			if ownedAliases[item.TargetEntityKey] == nil {
				ownedAliases[item.TargetEntityKey] = map[string]struct{}{}
			}
			ownedAliases[item.TargetEntityKey][item.LegacyName] = struct{}{}
			target := entities[item.TargetEntityKey]
			found := false
			for _, alias := range target.Aliases {
				if alias == item.LegacyName {
					found = true
					break
				}
			}
			if !found {
				target.Aliases = append(target.Aliases, item.LegacyName)
				entities[item.TargetEntityKey] = target
				report.AliasesChanged++
			}
			mapping := normalizeSectorSourceMapping(SectorSourceMapping{SectorEntityKey: item.TargetEntityKey, SourceSystem: "ths", SourceTaxonomyType: item.LegacyTaxonomy, SourceSectorName: item.LegacyName, SourceMarketScope: "cn_a_share", SourceURL: manifest.ReviewSourceURL, MappingStatus: "merged", ReviewNote: item.Reason})
			identity := sectorSourceMappingIdentity(mapping)
			if existing, ok := mappings[identity]; !ok || !reflect.DeepEqual(existing, mapping) {
				report.MappingsChanged++
			}
			mappings[identity] = mapping
			for key, relationship := range relationships {
				if mode == SectorConvergenceModeCorrection {
					continue
				}
				moved := false
				if relationship.From == item.LegacyEntityKey {
					relationship.From = item.TargetEntityKey
					report.ReferencesMoved++
					moved = true
				}
				if relationship.To == item.LegacyEntityKey {
					relationship.To = item.TargetEntityKey
					report.ReferencesMoved++
					moved = true
				}
				relationships[key] = relationship
				if moved {
					relationshipMoves[auditKey] = append(relationshipMoves[auditKey], key)
				}
			}
		} else if item.TargetEntityType == domain.EntityTypeIndex {
			for key, relationship := range relationships {
				if mode == SectorConvergenceModeCorrection {
					continue
				}
				if relationship.From != item.LegacyEntityKey && relationship.To != item.LegacyEntityKey {
					continue
				}
				planned, _, err := planConvergenceRelationship(relationship, entities, item.LegacyEntityKey, item.TargetEntityKey)
				if err != nil {
					return SectorConvergenceReport{ManifestVersion: manifest.ManifestVersion, BlockedReferences: 1}, fmt.Errorf("index target relationship %q is incompatible: %w", key, err)
				}
				if relationship.From == item.LegacyEntityKey {
					report.ReferencesMoved++
				}
				if relationship.To == item.LegacyEntityKey {
					report.ReferencesMoved++
				}
				relationships[key] = planned
				relationshipMoves[auditKey] = append(relationshipMoves[auditKey], key)
			}
		} else {
			for key, relationship := range relationships {
				if mode == SectorConvergenceModeCorrection {
					continue
				}
				if relationship.From == item.LegacyEntityKey || relationship.To == item.LegacyEntityKey {
					planned, _, _ := planConvergenceRelationship(relationship, entities, item.LegacyEntityKey, "")
					if relationship.Status != domain.StatusInactive {
						report.ReferencesMoved++
					}
					relationships[key] = planned
					relationshipMoves[auditKey] = append(relationshipMoves[auditKey], key)
				}
			}
		}
		audits[item.LegacyEntityKey+"|"+fmt.Sprint(manifest.ManifestVersion)] = item
	}
	resolveMemoryRelationshipConflicts(relationships)
	r.entities = entities
	r.profiles = profiles
	r.sectorSourceMappings = mappings
	r.relationships = relationships
	r.convergenceAudits = audits
	r.convergenceRelationshipMoves = relationshipMoves
	r.convergenceOwnedAliases = ownedAliases
	r.convergenceManifests[manifest.ManifestVersion] = manifest.ManifestChecksum
	r.convergenceReviews[manifest.ManifestVersion] = manifest.ReviewSourceURL + manifest.ReviewedAt.String()
	return report, nil
}

func cloneStringSetMap(input map[string]map[string]struct{}) map[string]map[string]struct{} {
	output := make(map[string]map[string]struct{}, len(input))
	for key, values := range input {
		cloned := make(map[string]struct{}, len(values))
		for value := range values {
			cloned[value] = struct{}{}
		}
		output[key] = cloned
	}
	return output
}

func removeAlias(values []string, target string) []string {
	output := make([]string, 0, len(values))
	for _, value := range values {
		if value != target {
			output = append(output, value)
		}
	}
	return output
}

func cloneStringSliceMap(input map[string][]string) map[string][]string {
	output := make(map[string][]string, len(input))
	for key, values := range input {
		output[key] = append([]string(nil), values...)
	}
	return output
}

func profileReferencesEntity(data []byte, field, entityKey string) bool {
	var values map[string]any
	if json.Unmarshal(data, &values) != nil {
		return false
	}
	return strings.TrimSpace(fmt.Sprint(values[field])) == entityKey
}

func resolveMemoryRelationshipConflicts(relationships map[string]Relationship) {
	keys := make([]string, 0, len(relationships))
	for key := range relationships {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	seen := map[string]struct{}{}
	for _, key := range keys {
		relationship := relationships[key]
		if relationship.Status == domain.StatusInactive {
			continue
		}
		identity := relationship.From + "|" + relationship.To + "|" + relationship.RelationType
		if _, exists := seen[identity]; exists {
			relationship.Status = domain.StatusInactive
			relationships[key] = relationship
			continue
		}
		seen[identity] = struct{}{}
	}
}

func (r *MemoryRepository) ConvergenceAuditCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.convergenceAudits)
}

func (r *MemoryRepository) UpsertSectorSourceMapping(_ context.Context, mapping SectorSourceMapping) (WriteResult, error) {
	mapping = normalizeSectorSourceMapping(mapping)
	if err := validateSectorSourceMapping(mapping); err != nil {
		return WriteResult{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	entity, ok := r.entities[mapping.SectorEntityKey]
	if !ok || entity.EntityType != domain.EntityTypeSector {
		return WriteResult{}, fmt.Errorf("unknown sector source mapping entity key %q", mapping.SectorEntityKey)
	}
	identity := sectorSourceMappingIdentity(mapping)
	existing, ok := r.sectorSourceMappings[identity]
	if !ok {
		r.sectorSourceMappings[identity] = mapping
		return WriteResult{Key: identity, Action: WriteCreated}, nil
	}
	if existing.SnapshotDate != "" && (mapping.SnapshotDate == "" || mapping.SnapshotDate < existing.SnapshotDate) {
		mapping.RankSnapshot = existing.RankSnapshot
		mapping.SnapshotDate = existing.SnapshotDate
		mapping.SourceURL = existing.SourceURL
	}
	if reflect.DeepEqual(existing, mapping) {
		return WriteResult{Key: identity, Action: WriteUnchanged}, nil
	}
	r.sectorSourceMappings[identity] = mapping
	return WriteResult{Key: identity, Action: WriteUpdated}, nil
}

func (r *MemoryRepository) UpsertEntity(_ context.Context, entity Entity) (WriteResult, error) {
	if err := validateEntity(entity); err != nil {
		return WriteResult{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	owned := make([]string, 0, len(r.convergenceOwnedAliases[entity.Key]))
	for alias := range r.convergenceOwnedAliases[entity.Key] {
		owned = append(owned, alias)
	}
	entity.Aliases = mergeAliasSets(entity.Aliases, owned)

	existing, ok := r.entities[entity.Key]
	if !ok {
		r.entities[entity.Key] = entity
		return WriteResult{Key: entity.Key, Action: WriteCreated}, nil
	}
	if reflect.DeepEqual(existing, entity) {
		return WriteResult{Key: entity.Key, Action: WriteUnchanged}, nil
	}
	r.entities[entity.Key] = entity
	return WriteResult{Key: entity.Key, Action: WriteUpdated}, nil
}

func mergeAliasSets(groups ...[]string) []string {
	seen := map[string]struct{}{}
	for _, group := range groups {
		for _, alias := range group {
			if alias != "" {
				seen[alias] = struct{}{}
			}
		}
	}
	result := make([]string, 0, len(seen))
	for alias := range seen {
		result = append(result, alias)
	}
	sort.Strings(result)
	return result
}

func (r *MemoryRepository) UpsertProfile(_ context.Context, profile Profile) (WriteResult, error) {
	if profile.EntityKey == "" {
		return WriteResult{}, fmt.Errorf("profile entity key is required")
	}
	if err := validateProfileData(profile.EntityType, profile.Data); err != nil {
		return WriteResult{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	entity, ok := r.entities[profile.EntityKey]
	if !ok {
		return WriteResult{}, fmt.Errorf("unknown profile entity key %q", profile.EntityKey)
	}
	if profile.EntityType == "" {
		profile.EntityType = entity.EntityType
	}
	if profile.EntityType != entity.EntityType {
		return WriteResult{}, fmt.Errorf("profile entity type %q does not match entity %q type %q", profile.EntityType, profile.EntityKey, entity.EntityType)
	}

	existing, ok := r.profiles[profile.EntityKey]
	if !ok {
		r.profiles[profile.EntityKey] = profile
		return WriteResult{Key: profile.EntityKey, Action: WriteCreated}, nil
	}
	if reflect.DeepEqual(existing, profile) {
		return WriteResult{Key: profile.EntityKey, Action: WriteUnchanged}, nil
	}
	r.profiles[profile.EntityKey] = profile
	return WriteResult{Key: profile.EntityKey, Action: WriteUpdated}, nil
}

func (r *MemoryRepository) UpsertRelationship(_ context.Context, relationship Relationship) (WriteResult, error) {
	if relationship.Key == "" {
		return WriteResult{}, fmt.Errorf("relationship key is required")
	}
	if relationship.From == "" {
		return WriteResult{}, fmt.Errorf("relationship %q source is required", relationship.Key)
	}
	if relationship.To == "" {
		return WriteResult{}, fmt.Errorf("relationship %q target is required", relationship.Key)
	}
	if relationship.RelationType == "" {
		return WriteResult{}, fmt.Errorf("relationship %q relation type is required", relationship.Key)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.entities[relationship.From]; !ok {
		return WriteResult{}, fmt.Errorf("unknown relationship source %q", relationship.From)
	}
	if _, ok := r.entities[relationship.To]; !ok {
		return WriteResult{}, fmt.Errorf("unknown relationship target %q", relationship.To)
	}

	existing, ok := r.relationships[relationship.Key]
	if !ok {
		r.relationships[relationship.Key] = relationship
		return WriteResult{Key: relationship.Key, Action: WriteCreated}, nil
	}
	if reflect.DeepEqual(existing, relationship) {
		return WriteResult{Key: relationship.Key, Action: WriteUnchanged}, nil
	}
	r.relationships[relationship.Key] = relationship
	return WriteResult{Key: relationship.Key, Action: WriteUpdated}, nil
}

func (r *MemoryRepository) EntityCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.entities)
}

func (r *MemoryRepository) SectorSourceMappingCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.sectorSourceMappings)
}
