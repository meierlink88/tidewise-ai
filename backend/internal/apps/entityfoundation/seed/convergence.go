package seed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type SectorConvergenceAction string

const (
	SectorConvergenceReplace                  SectorConvergenceAction = "replace"
	SectorConvergenceMerge                    SectorConvergenceAction = "merge"
	SectorConvergenceRetireWithoutCanonical   SectorConvergenceAction = "retire_without_canonical"
	SectorConvergenceReplaceWithExistingIndex SectorConvergenceAction = "replace_with_existing_index"
	SectorConvergenceRetireWithoutTarget      SectorConvergenceAction = "retire_without_target"
)

type SectorConvergenceMode string

const (
	SectorConvergenceModeInitial    SectorConvergenceMode = "initial"
	SectorConvergenceModeCorrection SectorConvergenceMode = "correction"
)

type SectorConvergenceManifest struct {
	ManifestVersion         int64               `json:"manifest_version"`
	PreviousManifestVersion *int64              `json:"previous_manifest_version"`
	ManifestChecksum        string              `json:"manifest_checksum"`
	ReviewSourceURL         string              `json:"review_source_url"`
	ReviewedAt              time.Time           `json:"reviewed_at"`
	Convergences            []SectorConvergence `json:"convergences"`
}

type SectorConvergence struct {
	LegacyEntityKey     string                  `json:"legacy_entity_key"`
	LegacyName          string                  `json:"legacy_name"`
	LegacyTaxonomy      string                  `json:"legacy_taxonomy"`
	Action              SectorConvergenceAction `json:"action"`
	TargetEntityKey     string                  `json:"target_entity_key,omitempty"`
	TargetEntityType    domain.EntityType       `json:"target_entity_type,omitempty"`
	UUIDPolicy          string                  `json:"uuid_policy"`
	AliasPolicy         string                  `json:"alias_policy"`
	SourceMappingPolicy string                  `json:"source_mapping_policy"`
	ReferencePolicy     string                  `json:"reference_policy"`
	Reason              string                  `json:"reason"`
}

type SectorConvergenceReport struct {
	ManifestVersion   int64 `json:"manifest_version"`
	RetiredLegacy     int   `json:"retired_legacy"`
	AuditCreated      int   `json:"audit_created"`
	AuditUnchanged    int   `json:"audit_unchanged"`
	ReferencesMoved   int   `json:"references_moved"`
	MappingsChanged   int   `json:"mappings_changed"`
	AliasesChanged    int   `json:"aliases_changed"`
	BlockedReferences int   `json:"blocked_references"`
}

type sectorConvergenceRepository interface {
	ApplySectorConvergence(context.Context, Manifest, SectorConvergenceManifest, SectorConvergenceMode) (SectorConvergenceReport, error)
}

type convergenceEdgeDisposition string

const (
	convergenceEdgeRedirect   convergenceEdgeDisposition = "redirect"
	convergenceEdgeDeactivate convergenceEdgeDisposition = "deactivate"
)

func planConvergenceRelationship(relationship Relationship, entities map[string]Entity, legacyKey, targetKey string) (Relationship, convergenceEdgeDisposition, error) {
	if targetKey == "" {
		relationship.Status = domain.StatusInactive
		return relationship, convergenceEdgeDeactivate, nil
	}
	if relationship.From == legacyKey {
		relationship.From = targetKey
	}
	if relationship.To == legacyKey {
		relationship.To = targetKey
	}
	relationship.Status = domain.StatusActive
	if err := validateRelationshipPolicy(relationship, entities); err != nil {
		return Relationship{}, "", err
	}
	return relationship, convergenceEdgeRedirect, nil
}

func LoadSectorConvergenceFile(path string) (SectorConvergenceManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SectorConvergenceManifest{}, fmt.Errorf("read sector convergence manifest: %w", err)
	}
	if err := rejectForbiddenReasoningFields(data); err != nil {
		return SectorConvergenceManifest{}, fmt.Errorf("sector convergence manifest: %w", err)
	}
	var manifest SectorConvergenceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return SectorConvergenceManifest{}, fmt.Errorf("decode sector convergence manifest: %w", err)
	}
	if err := ValidateSectorConvergenceManifest(manifest); err != nil {
		return SectorConvergenceManifest{}, err
	}
	return manifest, nil
}

func ValidateSectorConvergenceManifest(manifest SectorConvergenceManifest) error {
	if manifest.ManifestVersion <= 0 {
		return fmt.Errorf("manifest version must be positive")
	}
	if manifest.PreviousManifestVersion != nil && *manifest.PreviousManifestVersion >= manifest.ManifestVersion {
		return fmt.Errorf("previous manifest version must be lower")
	}
	if strings.TrimSpace(manifest.ManifestChecksum) == "" || strings.TrimSpace(manifest.ReviewSourceURL) == "" || manifest.ReviewedAt.IsZero() {
		return fmt.Errorf("manifest checksum and Review metadata are required")
	}
	if len(manifest.Convergences) != 60 {
		return fmt.Errorf("sector convergence manifest must contain 60 items")
	}
	if got := sectorConvergenceChecksum(manifest.Convergences); got != manifest.ManifestChecksum {
		return fmt.Errorf("manifest checksum mismatch")
	}
	legacyKeys := map[string]struct{}{}
	sectorTargets := 0
	indexTargets := 0
	withoutTarget := 0
	for _, item := range manifest.Convergences {
		if !strings.HasPrefix(item.LegacyEntityKey, "sector:ths_") || item.LegacyName == "" || item.LegacyTaxonomy == "" || item.Reason == "" {
			return fmt.Errorf("legacy convergence identity fields are required")
		}
		if _, exists := legacyKeys[item.LegacyEntityKey]; exists {
			return fmt.Errorf("duplicate legacy entity key %q", item.LegacyEntityKey)
		}
		legacyKeys[item.LegacyEntityKey] = struct{}{}
		if item.UUIDPolicy == "" || item.AliasPolicy == "" || item.SourceMappingPolicy == "" || item.ReferencePolicy == "" {
			return fmt.Errorf("convergence policies are required")
		}
		switch item.Action {
		case SectorConvergenceReplace, SectorConvergenceMerge:
			if item.TargetEntityType != domain.EntityTypeSector || !strings.HasPrefix(item.TargetEntityKey, "sector:") {
				return fmt.Errorf("%s requires a sector target", item.Action)
			}
			sectorTargets++
		case SectorConvergenceReplaceWithExistingIndex:
			if item.TargetEntityType != domain.EntityTypeIndex || !strings.HasPrefix(item.TargetEntityKey, "index:") {
				return fmt.Errorf("replace_with_existing_index requires an index target")
			}
			indexTargets++
		case SectorConvergenceRetireWithoutCanonical, SectorConvergenceRetireWithoutTarget:
			if item.TargetEntityKey != "" || item.TargetEntityType != "" {
				return fmt.Errorf("%s must not define a target", item.Action)
			}
			withoutTarget++
		default:
			return fmt.Errorf("unsupported convergence action %q", item.Action)
		}
	}
	if sectorTargets != 29 || indexTargets != 15 || withoutTarget != 16 {
		return fmt.Errorf("convergence target distribution must be 29 sector, 15 index, and 16 without target")
	}
	return nil
}

func sectorConvergenceChecksum(items []SectorConvergence) string {
	data, _ := json.Marshal(items)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (s Service) ApplySectorConvergence(ctx context.Context, entities Manifest, manifest SectorConvergenceManifest, mode SectorConvergenceMode) (SectorConvergenceReport, error) {
	if err := ValidateSectorConvergenceManifest(manifest); err != nil {
		return SectorConvergenceReport{}, err
	}
	repository, ok := s.repository.(sectorConvergenceRepository)
	if !ok {
		return SectorConvergenceReport{}, fmt.Errorf("repository does not support sector convergence")
	}
	return repository.ApplySectorConvergence(ctx, entities, manifest, mode)
}
