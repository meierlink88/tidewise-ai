package seed

import (
	"context"
	"path/filepath"
	"reflect"
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

func TestMemoryIndexTargetRedirectsPolicyCompatibleEdgeAndNoTargetDeactivates(t *testing.T) {
	repo := NewMemoryRepository()
	seedMemoryConvergenceEntities(t, repo)
	market := Entity{Key: "market:test", EntityType: domain.EntityTypeMarket, LayerCode: "market", Name: "测试市场", CanonicalName: "测试市场", Status: domain.StatusActive}
	if _, err := repo.UpsertEntity(context.Background(), market); err != nil {
		t.Fatal(err)
	}
	manifest := reviewedConvergenceFixture(t)
	indexLegacy, noTargetLegacy := "", ""
	for _, item := range manifest.Convergences {
		if item.Action == SectorConvergenceReplaceWithExistingIndex && indexLegacy == "" {
			indexLegacy = item.LegacyEntityKey
		}
		if item.Action == SectorConvergenceRetireWithoutTarget && noTargetLegacy == "" {
			noTargetLegacy = item.LegacyEntityKey
		}
	}
	verifiedAt := manifest.ReviewedAt
	repo.relationships["relationship:index_redirect"] = Relationship{Key: "relationship:index_redirect", From: "market:test", To: indexLegacy, RelationType: "tracks_index", SourceName: "review", SourceURL: manifest.ReviewSourceURL, VerifiedAt: verifiedAt, Status: domain.StatusActive}
	repo.relationships["relationship:no_target"] = Relationship{Key: "relationship:no_target", From: "market:test", To: noTargetLegacy, RelationType: "tracks_index", SourceName: "review", SourceURL: manifest.ReviewSourceURL, VerifiedAt: verifiedAt, Status: domain.StatusActive}
	if _, err := NewService(repo).ApplySectorConvergence(context.Background(), Manifest{}, manifest, SectorConvergenceModeInitial); err != nil {
		t.Fatal(err)
	}
	redirected := repo.relationships["relationship:index_redirect"]
	if redirected.To == indexLegacy || redirected.Status != domain.StatusActive {
		t.Fatalf("index relationship = %+v", redirected)
	}
	if target := repo.entities[redirected.To]; target.EntityType != domain.EntityTypeIndex {
		t.Fatalf("redirect target = %+v", target)
	}
	if got := repo.relationships["relationship:no_target"].Status; got != domain.StatusInactive {
		t.Fatalf("no-target edge status = %q", got)
	}
}

func TestMemoryIndexTargetBlocksSectorOnlyMapping(t *testing.T) {
	repo := NewMemoryRepository()
	seedMemoryConvergenceEntities(t, repo)
	manifest := reviewedConvergenceFixture(t)
	var legacy string
	for _, item := range manifest.Convergences {
		if item.Action == SectorConvergenceReplaceWithExistingIndex {
			legacy = item.LegacyEntityKey
			break
		}
	}
	mapping := normalizeSectorSourceMapping(SectorSourceMapping{SectorEntityKey: legacy, SourceSystem: "legacy", SourceTaxonomyType: "index_sector", SourceSectorName: "legacy index", SourceMarketScope: "global", MappingStatus: "approved"})
	repo.sectorSourceMappings[sectorSourceMappingIdentity(mapping)] = mapping
	before := repo.ConvergenceAuditCount()
	report, err := NewService(repo).ApplySectorConvergence(context.Background(), Manifest{}, manifest, SectorConvergenceModeInitial)
	if err == nil || !strings.Contains(err.Error(), "sector-only") {
		t.Fatalf("sector-only error = %v", err)
	}
	if report.BlockedReferences != 1 {
		t.Fatalf("blocked references = %d", report.BlockedReferences)
	}
	if repo.ConvergenceAuditCount() != before || repo.entities[legacy].Status != domain.StatusActive {
		t.Fatal("blocked convergence was not atomic")
	}
}

func TestPostgresConvergenceRelationshipPlanRedirectsIndexAndDeactivatesNoTarget(t *testing.T) {
	verifiedAt := reviewedConvergenceFixture(t).ReviewedAt
	relationship := Relationship{Key: "relationship:test", From: "market:test", To: "sector:legacy", RelationType: "tracks_index", SourceName: "review", SourceURL: "https://example.com/review", VerifiedAt: verifiedAt, Status: domain.StatusActive}
	entities := map[string]Entity{"market:test": {Key: "market:test", EntityType: domain.EntityTypeMarket}, "sector:legacy": {Key: "sector:legacy", EntityType: domain.EntityTypeSector}, "index:csi300": {Key: "index:csi300", EntityType: domain.EntityTypeIndex}}
	redirected, disposition, err := planConvergenceRelationship(relationship, entities, "sector:legacy", "index:csi300")
	if err != nil {
		t.Fatal(err)
	}
	if disposition != convergenceEdgeRedirect || redirected.To != "index:csi300" || redirected.Status != domain.StatusActive {
		t.Fatalf("redirect plan = %+v %q", redirected, disposition)
	}
	retired, disposition, err := planConvergenceRelationship(relationship, entities, "sector:legacy", "")
	if err != nil {
		t.Fatal(err)
	}
	if disposition != convergenceEdgeDeactivate || retired.Status != domain.StatusInactive {
		t.Fatalf("retire plan = %+v %q", retired, disposition)
	}
}

func TestConvergenceRelationshipTransitionMatrix(t *testing.T) {
	verifiedAt := reviewedConvergenceFixture(t).ReviewedAt
	type transition struct {
		name, relation, fromTarget, toTarget string
		fromType, toType                     domain.EntityType
		initialStatus                        domain.Status
		wantDisposition                      convergenceEdgeDisposition
	}
	cases := []transition{
		{name: "sector-to-sector", relation: "covers_sector", fromTarget: "sector:old", toTarget: "sector:new", fromType: domain.EntityTypeSector, toType: domain.EntityTypeSector, initialStatus: domain.StatusActive, wantDisposition: convergenceEdgeRedirect},
		{name: "sector-to-index", relation: "tracks_index", fromTarget: "sector:old", toTarget: "index:new", fromType: domain.EntityTypeSector, toType: domain.EntityTypeIndex, initialStatus: domain.StatusActive, wantDisposition: convergenceEdgeRedirect},
		{name: "index-to-index", relation: "tracks_index", fromTarget: "index:old", toTarget: "index:new", fromType: domain.EntityTypeIndex, toType: domain.EntityTypeIndex, initialStatus: domain.StatusActive, wantDisposition: convergenceEdgeRedirect},
		{name: "index-to-sector", relation: "covers_sector", fromTarget: "index:old", toTarget: "sector:new", fromType: domain.EntityTypeIndex, toType: domain.EntityTypeSector, initialStatus: domain.StatusActive, wantDisposition: convergenceEdgeRedirect},
		{name: "target-to-no-target", relation: "covers_sector", fromTarget: "sector:old", toTarget: "", fromType: domain.EntityTypeSector, initialStatus: domain.StatusActive, wantDisposition: convergenceEdgeDeactivate},
		{name: "no-target-to-sector", relation: "covers_sector", fromTarget: "sector:legacy", toTarget: "sector:new", fromType: domain.EntityTypeSector, toType: domain.EntityTypeSector, initialStatus: domain.StatusInactive, wantDisposition: convergenceEdgeRedirect},
		{name: "no-target-to-index", relation: "tracks_index", fromTarget: "sector:legacy", toTarget: "index:new", fromType: domain.EntityTypeSector, toType: domain.EntityTypeIndex, initialStatus: domain.StatusInactive, wantDisposition: convergenceEdgeRedirect},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rel := Relationship{Key: "relationship:test", From: "market:test", To: tc.fromTarget, RelationType: tc.relation, SourceName: "review", SourceURL: "https://example.com/review", VerifiedAt: verifiedAt, Status: tc.initialStatus}
			entities := map[string]Entity{"market:test": {Key: "market:test", EntityType: domain.EntityTypeMarket}, tc.fromTarget: {Key: tc.fromTarget, EntityType: tc.fromType}}
			if tc.toTarget != "" {
				entities[tc.toTarget] = Entity{Key: tc.toTarget, EntityType: tc.toType}
			}
			planned, disposition, err := planConvergenceRelationship(rel, entities, tc.fromTarget, tc.toTarget)
			if err != nil {
				t.Fatal(err)
			}
			if disposition != tc.wantDisposition {
				t.Fatalf("disposition = %q", disposition)
			}
			if tc.toTarget == "" {
				if planned.Status != domain.StatusInactive {
					t.Fatalf("status = %q", planned.Status)
				}
			} else if planned.To != tc.toTarget || planned.Status != domain.StatusActive {
				t.Fatalf("planned = %+v", planned)
			}
		})
	}
}

func TestConvergenceRelationshipTransitionRejectsIncompatiblePolicy(t *testing.T) {
	verifiedAt := reviewedConvergenceFixture(t).ReviewedAt
	rel := Relationship{Key: "relationship:test", From: "market:test", To: "sector:old", RelationType: "covers_sector", SourceName: "review", SourceURL: "https://example.com/review", VerifiedAt: verifiedAt, Status: domain.StatusActive}
	entities := map[string]Entity{"market:test": {Key: "market:test", EntityType: domain.EntityTypeMarket}, "sector:old": {Key: "sector:old", EntityType: domain.EntityTypeSector}, "index:new": {Key: "index:new", EntityType: domain.EntityTypeIndex}}
	if _, _, err := planConvergenceRelationship(rel, entities, "sector:old", "index:new"); err == nil {
		t.Fatal("incompatible transition error = nil")
	}
}

func TestMemoryCorrectionUsesRecordedRowsForNoTargetAndIndexSwap(t *testing.T) {
	repo := NewMemoryRepository()
	seedMemoryConvergenceEntities(t, repo)
	manifest := reviewedConvergenceFixture(t)
	market := Entity{Key: "market:test", EntityType: domain.EntityTypeMarket, LayerCode: "market", Name: "测试市场", CanonicalName: "测试市场", Status: domain.StatusActive}
	if _, err := repo.UpsertEntity(context.Background(), market); err != nil {
		t.Fatal(err)
	}
	noTargetIndex, indexTarget := -1, -1
	for i, item := range manifest.Convergences {
		if item.Action == SectorConvergenceRetireWithoutTarget && noTargetIndex < 0 {
			noTargetIndex = i
		}
		if item.Action == SectorConvergenceReplaceWithExistingIndex && indexTarget < 0 {
			indexTarget = i
		}
	}
	noTargetLegacy, indexLegacy := manifest.Convergences[noTargetIndex].LegacyEntityKey, manifest.Convergences[indexTarget].LegacyEntityKey
	indexKey := ""
	for i, item := range manifest.Convergences {
		if i != indexTarget && item.Action == SectorConvergenceReplaceWithExistingIndex {
			indexKey = item.TargetEntityKey
			break
		}
	}
	for key, to := range map[string]string{"relationship:no_target": noTargetLegacy, "relationship:index": indexLegacy} {
		repo.relationships[key] = Relationship{Key: key, From: "market:test", To: to, RelationType: "tracks_index", SourceName: "review", SourceURL: manifest.ReviewSourceURL, VerifiedAt: manifest.ReviewedAt, Status: domain.StatusActive}
	}
	service := NewService(repo)
	if _, err := service.ApplySectorConvergence(context.Background(), Manifest{}, manifest, SectorConvergenceModeInitial); err != nil {
		t.Fatal(err)
	}
	if repo.relationships["relationship:no_target"].Status != domain.StatusInactive {
		t.Fatal("no-target initial edge not inactive")
	}
	previous := int64(1)
	correction := mutateConvergenceManifest(manifest, func(m *SectorConvergenceManifest) {
		m.ManifestVersion = 2
		m.PreviousManifestVersion = &previous
		m.ReviewSourceURL += "?review=swap"
		m.ReviewedAt = m.ReviewedAt.AddDate(0, 0, 1)
		m.Convergences[noTargetIndex].Action = SectorConvergenceReplaceWithExistingIndex
		m.Convergences[noTargetIndex].TargetEntityKey = indexKey
		m.Convergences[noTargetIndex].TargetEntityType = domain.EntityTypeIndex
		m.Convergences[indexTarget].Action = SectorConvergenceRetireWithoutTarget
		m.Convergences[indexTarget].TargetEntityKey = ""
		m.Convergences[indexTarget].TargetEntityType = ""
		m.ManifestChecksum = sectorConvergenceChecksum(m.Convergences)
	})
	report, err := service.ApplySectorConvergence(context.Background(), Manifest{}, correction, SectorConvergenceModeCorrection)
	if err != nil {
		t.Fatal(err)
	}
	restored := repo.relationships["relationship:no_target"]
	if restored.Status != domain.StatusActive || restored.To != indexKey {
		t.Fatalf("restored edge = %+v", restored)
	}
	if repo.relationships["relationship:index"].Status != domain.StatusInactive {
		t.Fatalf("retired edge = %+v", repo.relationships["relationship:index"])
	}
	if report.ReferencesMoved != 2 {
		t.Fatalf("references moved = %d", report.ReferencesMoved)
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
	if err == nil || !strings.Contains(err.Error(), "migration input only") {
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
	seedManifest, err := LoadFiles(legacyFixturePaths(root)...)
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

func TestMemoryOrdinarySeedPreservesCurrentConvergenceAliasesOnly(t *testing.T) {
	repo := NewMemoryRepository()
	manifest := reviewedConvergenceFixture(t)
	for _, item := range manifest.Convergences {
		legacy := Entity{Key: item.LegacyEntityKey, EntityType: domain.EntityTypeSector, LayerCode: "sector", Name: item.LegacyName, CanonicalName: item.LegacyName, Status: domain.StatusActive, Profile: []byte(`{"sector_system":"ths","sector_type":"concept"}`)}
		if _, err := repo.UpsertEntity(context.Background(), legacy); err != nil {
			t.Fatal(err)
		}
	}
	root := filepath.Join("..", "..", "..", "..", "data", "entity_foundation")
	seedManifest, err := LoadFiles(legacyFixturePaths(root)...)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewService(repo).ApplySectorConvergence(context.Background(), seedManifest, manifest, SectorConvergenceModeInitial); err != nil {
		t.Fatal(err)
	}
	var targetKey string
	for _, item := range manifest.Convergences {
		if item.TargetEntityType == domain.EntityTypeSector {
			count := 0
			for _, candidate := range manifest.Convergences {
				if candidate.TargetEntityKey == item.TargetEntityKey {
					count++
				}
			}
			if count > 1 {
				targetKey = item.TargetEntityKey
				break
			}
		}
	}
	target := repo.entities[targetKey]
	target.Aliases = append(target.Aliases, "temporary ordinary alias")
	repo.entities[targetKey] = target
	var seedEntity Entity
	for _, entity := range seedManifest.Entities {
		if entity.Key == targetKey {
			seedEntity = entity
			break
		}
	}
	result, err := repo.UpsertEntity(context.Background(), seedEntity)
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != WriteUpdated {
		t.Fatalf("action = %q", result.Action)
	}
	aliases := repo.entities[targetKey].Aliases
	expected := append([]string(nil), seedEntity.Aliases...)
	for _, item := range manifest.Convergences {
		if item.TargetEntityKey == targetKey && !containsString(expected, item.LegacyName) {
			expected = append(expected, item.LegacyName)
		}
	}
	if !reflect.DeepEqual(aliases, expected) {
		t.Fatalf("aliases = %v, want %v", aliases, expected)
	}
	if containsString(aliases, "temporary ordinary alias") {
		t.Fatalf("ordinary alias was retained: %v", aliases)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestMergeSeedAndOwnedAliasesPreservesOrderAndOwnership(t *testing.T) {
	cases := []struct {
		name              string
		seed, owned, want []string
	}{
		{name: "nil", want: []string{}},
		{name: "seed order", seed: []string{"中文", "English"}, want: []string{"中文", "English"}},
		{name: "deduplicate", seed: []string{"A", "A", "B"}, owned: []string{"B", "C", "C"}, want: []string{"A", "B", "C"}},
		{name: "owned provenance order", seed: []string{"Formal"}, owned: []string{"Legacy 2", "Legacy 1"}, want: []string{"Formal", "Legacy 2", "Legacy 1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mergeSeedAndOwnedAliases(tc.seed, tc.owned); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
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
