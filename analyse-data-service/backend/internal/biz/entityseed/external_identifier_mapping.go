package entityseed

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"
)

const frozenFirstBatchExternalIdentifierManifestSHA256 = "05539cd9f940cfcc5ec67cde5c395563b672ffa52d56090da0a83bd0d5997658"

type ExternalIdentifierMapping struct {
	ID                 string       `json:"id"`
	EntityID           string       `json:"entity_id"`
	SourceSystem       string       `json:"source_system"`
	SourceTaxonomyType string       `json:"source_taxonomy_type"`
	ExternalCode       string       `json:"external_code"`
	ExternalName       string       `json:"external_name"`
	Status             model.Status `json:"status"`
}

type ExternalIdentifierBatchReport struct {
	Created   int `json:"created"`
	Updated   int `json:"updated"`
	Unchanged int `json:"unchanged"`
}

type ExternalIdentifierMappingManifest struct {
	Mappings []ExternalIdentifierMapping `json:"mappings"`
}

func LoadExternalIdentifierMappingFile(path string) (ExternalIdentifierMappingManifest, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return ExternalIdentifierMappingManifest{}, err
	}
	var manifest ExternalIdentifierMappingManifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return manifest, err
	}
	if len(manifest.Mappings) == 0 {
		return manifest, fmt.Errorf("external identifier mapping manifest is empty")
	}
	manifest.Mappings, err = NormalizeAndValidateExternalIdentifierMappings(manifest.Mappings)
	return manifest, err
}

func ValidateExternalIdentifierMappingFile(path string) (ExternalIdentifierBatchReport, error) {
	manifest, err := LoadExternalIdentifierMappingFile(path)
	if err != nil {
		return ExternalIdentifierBatchReport{}, err
	}
	return ExternalIdentifierBatchReport{Created: len(manifest.Mappings)}, nil
}

func ValidateFrozenFirstBatchExternalIdentifierManifest(path string, mappings []ExternalIdentifierMapping) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if fmt.Sprintf("%x", sha256.Sum256(content)) != frozenFirstBatchExternalIdentifierManifestSHA256 {
		return fmt.Errorf("external identifier mapping manifest hash does not match approved first batch")
	}
	if len(mappings) != 1169 {
		return fmt.Errorf("external identifier mapping manifest rows = %d, want 1169", len(mappings))
	}
	providers, entities := map[string]int{}, map[string]map[string]struct{}{}
	for _, mapping := range mappings {
		providers[mapping.SourceSystem]++
		if entities[mapping.EntityID] == nil {
			entities[mapping.EntityID] = map[string]struct{}{}
		}
		entities[mapping.EntityID][mapping.SourceSystem] = struct{}{}
	}
	if providers["eastmoney"] != 818 || providers["ths"] != 351 {
		return fmt.Errorf("external identifier provider counts = eastmoney %d, ths %d; want 818/351", providers["eastmoney"], providers["ths"])
	}
	dualSource, multiTaxonomy := 0, 0
	for _, systems := range entities {
		if len(systems) == 2 {
			dualSource++
		}
	}
	byCode := map[string]int{}
	for _, mapping := range mappings {
		byCode[mapping.SourceSystem+"\x00"+mapping.ExternalCode]++
	}
	for _, count := range byCode {
		if count == 2 {
			multiTaxonomy++
		}
	}
	if dualSource != 241 || multiTaxonomy != 13 {
		return fmt.Errorf("external identifier dual-source/multi-taxonomy = %d/%d, want 241/13", dualSource, multiTaxonomy)
	}
	return nil
}

func NormalizeAndValidateExternalIdentifierMappings(mappings []ExternalIdentifierMapping) ([]ExternalIdentifierMapping, error) {
	if len(mappings) == 0 {
		return nil, fmt.Errorf("external identifier mapping batch is empty")
	}
	seenIdentity := make(map[string]struct{}, len(mappings))
	seenID := make(map[string]struct{}, len(mappings))
	normalized := make([]ExternalIdentifierMapping, 0, len(mappings))
	for _, mapping := range mappings {
		item := normalizeExternalIdentifier(ExternalIdentifierFromMapping(mapping))
		if err := validateFirstBatchExternalIdentifier(item); err != nil {
			return nil, err
		}
		identity := externalIdentifierIdentity(item.SourceSystem, item.SourceTaxonomyType, item.ExternalCode)
		if _, exists := seenIdentity[identity]; exists {
			return nil, fmt.Errorf("duplicate external identifier identity %q in mapping manifest", identity)
		}
		if _, exists := seenID[item.ID]; exists {
			return nil, fmt.Errorf("duplicate external identifier id %q in mapping manifest", item.ID)
		}
		seenIdentity[identity], seenID[item.ID] = struct{}{}, struct{}{}
		normalized = append(normalized, ExternalIdentifierMappingFromIdentifier(item))
	}
	sort.Slice(normalized, func(i, j int) bool {
		return externalIdentifierIdentity(normalized[i].SourceSystem, normalized[i].SourceTaxonomyType, normalized[i].ExternalCode) <
			externalIdentifierIdentity(normalized[j].SourceSystem, normalized[j].SourceTaxonomyType, normalized[j].ExternalCode)
	})
	return normalized, nil
}

func ExternalIdentifierMappingFromIdentifier(item model.EntityExternalIdentifier) ExternalIdentifierMapping {
	return ExternalIdentifierMapping{
		ID: item.ID, EntityID: item.EntityID, SourceSystem: item.SourceSystem,
		SourceTaxonomyType: item.SourceTaxonomyType, ExternalCode: item.ExternalCode,
		ExternalName: item.ExternalName, Status: item.Status,
	}
}

func ExternalIdentifierFromMapping(mapping ExternalIdentifierMapping) model.EntityExternalIdentifier {
	return model.EntityExternalIdentifier{
		ID: mapping.ID, EntityID: mapping.EntityID, SourceSystem: mapping.SourceSystem,
		SourceTaxonomyType: mapping.SourceTaxonomyType, ExternalCode: mapping.ExternalCode,
		ExternalName: mapping.ExternalName, Status: mapping.Status,
	}
}
