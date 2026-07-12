package seed

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestReviewedSectorConvergenceManifestIsComplete(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "data", "entity_foundation")
	manifest, err := LoadSectorConvergenceFile(filepath.Join(root, "sector_convergences.json"))
	if err != nil {
		t.Fatalf("LoadSectorConvergenceFile() error = %v", err)
	}
	if manifest.ManifestVersion != 1 || manifest.PreviousManifestVersion != nil || manifest.ManifestChecksum == "" || manifest.ReviewSourceURL == "" || manifest.ReviewedAt.IsZero() {
		t.Fatalf("manifest metadata = %+v", manifest)
	}
	if got := len(manifest.Convergences); got != 60 {
		t.Fatalf("convergences = %d, want 60", got)
	}
	counts := map[SectorConvergenceAction]int{}
	legacy := map[string]struct{}{}
	for _, item := range manifest.Convergences {
		counts[item.Action]++
		if _, ok := legacy[item.LegacyEntityKey]; ok {
			t.Fatalf("duplicate legacy key %q", item.LegacyEntityKey)
		}
		legacy[item.LegacyEntityKey] = struct{}{}
	}
	want := map[SectorConvergenceAction]int{
		SectorConvergenceReplace: 10, SectorConvergenceMerge: 19,
		SectorConvergenceRetireWithoutCanonical:   11,
		SectorConvergenceReplaceWithExistingIndex: 15,
		SectorConvergenceRetireWithoutTarget:      5,
	}
	for action, count := range want {
		if counts[action] != count {
			t.Errorf("%s count = %d, want %d", action, counts[action], count)
		}
	}
}

func TestSectorConvergenceManifestRejectsInvalidTargetsAndVersions(t *testing.T) {
	valid := reviewedConvergenceFixture(t)
	cases := map[string]SectorConvergenceManifest{
		"non-positive version": mutateConvergenceManifest(valid, func(m *SectorConvergenceManifest) { m.ManifestVersion = 0 }),
		"missing checksum":     mutateConvergenceManifest(valid, func(m *SectorConvergenceManifest) { m.ManifestChecksum = "" }),
		"incomplete":           mutateConvergenceManifest(valid, func(m *SectorConvergenceManifest) { m.Convergences = m.Convergences[:59] }),
		"sector action with index": mutateConvergenceManifest(valid, func(m *SectorConvergenceManifest) {
			m.Convergences[0].TargetEntityKey = "index:csi300"
			m.Convergences[0].TargetEntityType = domain.EntityTypeIndex
		}),
		"retire with target": mutateConvergenceManifest(valid, func(m *SectorConvergenceManifest) {
			for i := range m.Convergences {
				if m.Convergences[i].Action == SectorConvergenceRetireWithoutCanonical {
					m.Convergences[i].TargetEntityKey = "sector:theme_artificial_intelligence"
					m.Convergences[i].TargetEntityType = domain.EntityTypeSector
					break
				}
			}
		}),
	}
	for name, manifest := range cases {
		t.Run(name, func(t *testing.T) {
			if err := ValidateSectorConvergenceManifest(manifest); err == nil {
				t.Fatal("ValidateSectorConvergenceManifest() error = nil")
			}
		})
	}
}

func TestMemoryConvergenceIsAtomicVersionedAndIdempotent(t *testing.T) {
	repo := NewMemoryRepository()
	seedMemoryConvergenceEntities(t, repo)
	service := NewService(repo)
	manifest := reviewedConvergenceFixture(t)

	result, err := service.ApplySectorConvergence(context.Background(), Manifest{}, manifest, SectorConvergenceModeInitial)
	if err != nil {
		t.Fatalf("ApplySectorConvergence(initial) error = %v", err)
	}
	if result.RetiredLegacy != 60 || result.AuditCreated != 60 {
		t.Fatalf("initial report = %+v", result)
	}
	repeated, err := service.ApplySectorConvergence(context.Background(), Manifest{}, manifest, SectorConvergenceModeInitial)
	if err != nil {
		t.Fatalf("ApplySectorConvergence(repeat) error = %v", err)
	}
	if repeated.AuditUnchanged != 60 || repo.ConvergenceAuditCount() != 60 {
		t.Fatalf("repeat report = %+v audit count=%d", repeated, repo.ConvergenceAuditCount())
	}

	changed := mutateConvergenceManifest(manifest, func(m *SectorConvergenceManifest) {
		m.Convergences[0].Reason = "changed payload"
		m.ManifestChecksum = sectorConvergenceChecksum(m.Convergences)
	})
	if _, err := service.ApplySectorConvergence(context.Background(), Manifest{}, changed, SectorConvergenceModeInitial); err == nil || !strings.Contains(err.Error(), "same manifest version") {
		t.Fatalf("changed same-version error = %v", err)
	}
	if repo.ConvergenceAuditCount() != 60 {
		t.Fatalf("failed correction changed audit count = %d", repo.ConvergenceAuditCount())
	}
}

func TestMemoryConvergenceAppendsReviewedCorrectionAndRollsBackInvalidVersion(t *testing.T) {
	repo := NewMemoryRepository()
	seedMemoryConvergenceEntities(t, repo)
	service := NewService(repo)
	initial := reviewedConvergenceFixture(t)
	if _, err := service.ApplySectorConvergence(context.Background(), Manifest{}, initial, SectorConvergenceModeInitial); err != nil {
		t.Fatal(err)
	}
	previous := int64(1)
	correction := mutateConvergenceManifest(initial, func(m *SectorConvergenceManifest) {
		m.ManifestVersion = 2
		m.PreviousManifestVersion = &previous
		m.ReviewSourceURL += "?review=correction"
		m.ReviewedAt = m.ReviewedAt.AddDate(0, 0, 1)
	})
	result, err := service.ApplySectorConvergence(context.Background(), Manifest{}, correction, SectorConvergenceModeCorrection)
	if err != nil {
		t.Fatalf("ApplySectorConvergence(correction) error = %v", err)
	}
	if result.AuditCreated != 60 || repo.ConvergenceAuditCount() != 120 {
		t.Fatalf("correction report=%+v audits=%d", result, repo.ConvergenceAuditCount())
	}

	badPrevious := int64(0)
	invalid := mutateConvergenceManifest(correction, func(m *SectorConvergenceManifest) {
		m.ManifestVersion = 3
		m.PreviousManifestVersion = &badPrevious
	})
	if _, err := service.ApplySectorConvergence(context.Background(), Manifest{}, invalid, SectorConvergenceModeCorrection); err == nil {
		t.Fatal("invalid correction version error = nil")
	}
	if repo.ConvergenceAuditCount() != 120 {
		t.Fatalf("invalid correction changed audit count to %d", repo.ConvergenceAuditCount())
	}
}

func TestSectorReferenceRegistryIsTypeSafeAndUnknownReferencesFailClosed(t *testing.T) {
	registry := NewSectorReferenceRegistry()
	if rule, ok := registry.Rule("sector_source_mappings", "sector_entity_id"); !ok || !rule.SectorOnly {
		t.Fatalf("sector source mapping rule = %+v, %v", rule, ok)
	}
	if _, ok := registry.Rule("future_sector_facts", "sector_id"); ok {
		t.Fatal("unknown FK was registered implicitly")
	}
}

func TestMemoryCorrectionDriftRollsBackAndEdgeConflictsResolveDeterministically(t *testing.T) {
	relationships := map[string]Relationship{
		"relationship:z": {Key: "relationship:z", From: "market:a_share", To: "sector:target", RelationType: "covers_sector", Status: domain.StatusActive},
		"relationship:a": {Key: "relationship:a", From: "market:a_share", To: "sector:target", RelationType: "covers_sector", Status: domain.StatusActive},
	}
	resolveMemoryRelationshipConflicts(relationships)
	if relationships["relationship:a"].Status != domain.StatusActive || relationships["relationship:z"].Status != domain.StatusInactive {
		t.Fatalf("conflict resolution = %+v", relationships)
	}

	repo := NewMemoryRepository()
	seedMemoryConvergenceEntities(t, repo)
	service := NewService(repo)
	initial := reviewedConvergenceFixture(t)
	if _, err := service.ApplySectorConvergence(context.Background(), Manifest{}, initial, SectorConvergenceModeInitial); err != nil {
		t.Fatal(err)
	}
	first := initial.Convergences[0].LegacyEntityKey
	drifted := repo.entities[first]
	drifted.Status = domain.StatusActive
	repo.entities[first] = drifted
	previous := int64(1)
	correction := mutateConvergenceManifest(initial, func(m *SectorConvergenceManifest) {
		m.ManifestVersion = 2
		m.PreviousManifestVersion = &previous
		m.ReviewSourceURL += "?review=drift"
		m.ReviewedAt = m.ReviewedAt.AddDate(0, 0, 1)
	})
	if _, err := service.ApplySectorConvergence(context.Background(), Manifest{}, correction, SectorConvergenceModeCorrection); err == nil || !strings.Contains(err.Error(), "drift") {
		t.Fatalf("drift error = %v", err)
	}
	if repo.ConvergenceAuditCount() != 60 {
		t.Fatalf("drift changed audit count to %d", repo.ConvergenceAuditCount())
	}
}

func TestNormalSeedFailsClosedWithActiveLegacySectors(t *testing.T) {
	repo := NewMemoryRepository()
	legacy := Entity{Key: "sector:ths_concept_ai", EntityType: domain.EntityTypeSector, LayerCode: "sector", Name: "人工智能", CanonicalName: "人工智能", Status: domain.StatusActive, Profile: []byte(`{"sector_system":"ths","sector_type":"concept"}`)}
	if _, err := repo.UpsertEntity(context.Background(), legacy); err != nil {
		t.Fatal(err)
	}
	service := NewService(repo)
	_, err := service.Apply(context.Background(), Manifest{Entities: []Entity{{Key: "sector:theme_artificial_intelligence", EntityType: domain.EntityTypeSector, LayerCode: "sector", Name: "人工智能", CanonicalName: "人工智能", Aliases: []string{"Artificial Intelligence"}, Profile: []byte(`{"sector_system":"canonical","sector_type":"theme"}`)}}}, ApplyOptions{})
	if err == nil || !strings.Contains(err.Error(), "active legacy sector") {
		t.Fatalf("Apply() error = %v", err)
	}
	if repo.EntityCount() != 1 {
		t.Fatalf("normal seed wrote before guard, entity count = %d", repo.EntityCount())
	}
}

func TestReviewedConvergenceClosesCanonicalSeedCountsInMemory(t *testing.T) {
	repo := NewMemoryRepository()
	manifest := reviewedConvergenceFixture(t)
	for _, item := range manifest.Convergences {
		legacy := Entity{Key: item.LegacyEntityKey, EntityType: domain.EntityTypeSector, LayerCode: "sector", Name: item.LegacyName, CanonicalName: item.LegacyName, Status: domain.StatusActive, Profile: []byte(`{"sector_system":"ths","sector_type":"concept"}`)}
		if _, err := repo.UpsertEntity(context.Background(), legacy); err != nil {
			t.Fatal(err)
		}
	}
	root := filepath.Join("..", "..", "..", "..", "data", "entity_foundation")
	seedManifest, err := LoadFiles(DefaultSeedPaths(root)...)
	if err != nil {
		t.Fatal(err)
	}
	result, err := NewService(repo).ApplySectorConvergence(context.Background(), seedManifest, manifest, SectorConvergenceModeInitial)
	if err != nil {
		t.Fatal(err)
	}
	activeCanonical, inactiveLegacy := 0, 0
	for key, entity := range repo.entities {
		if entity.EntityType != domain.EntityTypeSector {
			continue
		}
		if strings.HasPrefix(key, "sector:ths_") && entity.Status == domain.StatusInactive {
			inactiveLegacy++
		}
		if !strings.HasPrefix(key, "sector:ths_") && entity.Status == domain.StatusActive {
			activeCanonical++
		}
	}
	if activeCanonical != 52 || inactiveLegacy != 60 || len(repo.convergenceAudits) != 60 {
		t.Fatalf("sector closure active=%d inactiveLegacy=%d audits=%d", activeCanonical, inactiveLegacy, len(repo.convergenceAudits))
	}
	if len(repo.sectorSourceMappings) != 89 || result.MappingsChanged != 29 {
		t.Fatalf("mapping closure count=%d report=%+v", len(repo.sectorSourceMappings), result)
	}
	covers, tracked := 0, 0
	for _, relationship := range repo.relationships {
		if relationship.RelationType == "covers_sector" {
			covers++
		}
		if relationship.RelationType == "tracked_by_benchmark" {
			tracked++
		}
	}
	if covers != 52 || tracked != 0 {
		t.Fatalf("relationship closure covers=%d tracked=%d", covers, tracked)
	}
}

func reviewedConvergenceFixture(t *testing.T) SectorConvergenceManifest {
	t.Helper()
	manifest, err := LoadSectorConvergenceFile(filepath.Join("..", "..", "..", "..", "data", "entity_foundation", "sector_convergences.json"))
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

func mutateConvergenceManifest(input SectorConvergenceManifest, mutate func(*SectorConvergenceManifest)) SectorConvergenceManifest {
	clone := input
	clone.Convergences = append([]SectorConvergence(nil), input.Convergences...)
	mutate(&clone)
	return clone
}

func seedMemoryConvergenceEntities(t *testing.T, repo *MemoryRepository) {
	t.Helper()
	manifest := reviewedConvergenceFixture(t)
	for _, item := range manifest.Convergences {
		entity := Entity{Key: item.LegacyEntityKey, EntityType: domain.EntityTypeSector, LayerCode: "sector", Name: item.LegacyName, CanonicalName: item.LegacyName, Status: domain.StatusActive, Profile: []byte(`{"sector_system":"ths","sector_type":"concept"}`)}
		if _, err := repo.UpsertEntity(context.Background(), entity); err != nil {
			t.Fatal(err)
		}
		if item.TargetEntityKey != "" {
			target := Entity{Key: item.TargetEntityKey, EntityType: item.TargetEntityType, LayerCode: string(item.TargetEntityType), Name: "目标实体", CanonicalName: "目标实体", Status: domain.StatusActive}
			if item.TargetEntityType == domain.EntityTypeSector {
				target.LayerCode = "sector"
				target.Aliases = []string{"Target Entity"}
				target.Profile = []byte(`{"sector_system":"canonical","sector_type":"theme"}`)
			}
			if _, err := repo.UpsertEntity(context.Background(), target); err != nil {
				t.Fatal(err)
			}
		}
	}
}
