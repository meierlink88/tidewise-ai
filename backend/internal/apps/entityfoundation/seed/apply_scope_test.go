package seed

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestIndustryChainMasterScopeSelectsOnlyPilotMasterData(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "data", "entity_foundation")
	manifest, err := LoadFiles(DefaultSeedPaths(root)...)
	if err != nil {
		t.Fatalf("LoadFiles() error = %v", err)
	}
	repo := &masterScopeRepository{existing: map[string]struct{}{
		"chain_node:power_grid": {}, "chain_node:data_center": {}, "chain_node:gpu": {},
		"chain_node:eda": {}, "chain_node:lithography_machine": {},
	}}

	report, err := NewService(repo).Apply(context.Background(), manifest, ApplyOptions{Scope: ApplyScopeIndustryChainMaster})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if repo.industryBatchCalled {
		t.Fatal("master-only scope called UpsertIndustryChainBatch")
	}
	if len(repo.entities) != 28 {
		t.Fatalf("entities = %d, want 28", len(repo.entities))
	}
	byType := map[domain.EntityType]int{}
	for _, entity := range repo.entities {
		byType[entity.EntityType]++
		if entity.EntityType != domain.EntityTypeIndustryChain && entity.EntityType != domain.EntityTypeChainNode {
			t.Fatalf("master-only entity %q has unrelated type %q", entity.Key, entity.EntityType)
		}
	}
	if byType[domain.EntityTypeIndustryChain] != 2 || byType[domain.EntityTypeChainNode] != 26 {
		t.Fatalf("entity types = %#v, want 2 chains and 26 nodes", byType)
	}
	if len(repo.profileKeys) != 33 {
		t.Fatalf("profile writes = %d, want 33 including 5 reviewed overrides", len(repo.profileKeys))
	}
	if len(repo.relationships) != 0 || len(repo.sectorMappings) != 0 {
		t.Fatalf("master-only wrote unrelated data: relationships=%d mappings=%d", len(repo.relationships), len(repo.sectorMappings))
	}
	if report.Scope != string(ApplyScopeIndustryChainMaster) {
		t.Fatalf("Scope = %q", report.Scope)
	}
	if report.Created != 46 || report.Updated != 15 || report.Unchanged != 0 {
		t.Fatalf("operation stats = created:%d updated:%d unchanged:%d, want 46/15/0", report.Created, report.Updated, report.Unchanged)
	}
	wantImpact := map[string]WriteStats{
		"entity_nodes":            {Created: 23, Updated: 5},
		"industry_chain_profiles": {Created: 2},
		"chain_node_profiles":     {Created: 21, Updated: 5},
	}
	if !reflect.DeepEqual(report.FinalTableImpact, wantImpact) {
		t.Fatalf("FinalTableImpact = %#v, want %#v", report.FinalTableImpact, wantImpact)
	}
}

func TestDefaultScopeStillAppliesIndustryChainBatch(t *testing.T) {
	manifest := validIndustryChainManifest()
	repo := &masterScopeRepository{}
	if _, err := NewService(repo).Apply(context.Background(), manifest, ApplyOptions{}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !repo.industryBatchCalled {
		t.Fatal("default scope did not call UpsertIndustryChainBatch")
	}
}

func TestIndustryChainMembershipScopeWritesOnlyMembershipBatch(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "data", "entity_foundation")
	manifest, err := LoadFiles(DefaultSeedPaths(root)...)
	if err != nil {
		t.Fatalf("LoadFiles() error = %v", err)
	}
	repo := &masterScopeRepository{}

	report, err := NewService(repo).Apply(context.Background(), manifest, ApplyOptions{Scope: ApplyScopeIndustryChainMembership})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if len(repo.entities) != 0 || len(repo.profileKeys) != 0 || len(repo.relationships) != 0 || len(repo.sectorMappings) != 0 {
		t.Fatalf("membership scope wrote unrelated families: entities=%d profiles=%d relationships=%d mappings=%d", len(repo.entities), len(repo.profileKeys), len(repo.relationships), len(repo.sectorMappings))
	}
	if !repo.industryBatchCalled {
		t.Fatal("membership scope did not call UpsertIndustryChainBatch")
	}
	if len(repo.industryBatch.Memberships) != 27 || len(repo.industryBatch.TopologyEdges) != 0 || len(repo.industryBatch.PhysicalConstraints) != 0 {
		t.Fatalf("industry batch = memberships:%d topology:%d constraints:%d", len(repo.industryBatch.Memberships), len(repo.industryBatch.TopologyEdges), len(repo.industryBatch.PhysicalConstraints))
	}
	if report.Scope != string(ApplyScopeIndustryChainMembership) || report.IndustryChainCounts["membership"] != 27 {
		t.Fatalf("report scope/counts = %q %#v", report.Scope, report.IndustryChainCounts)
	}
	if report.Created != 27 || report.Updated != 0 || report.Unchanged != 0 || report.OperationCounts.Created != 27 {
		t.Fatalf("operation report = %#v", report)
	}
	wantImpact := map[string]WriteStats{"industry_chain_memberships": {Created: 27}}
	if !reflect.DeepEqual(report.FinalTableImpact, wantImpact) {
		t.Fatalf("FinalTableImpact = %#v, want %#v", report.FinalTableImpact, wantImpact)
	}
}

func TestParseApplyScopeRejectsUnknownValues(t *testing.T) {
	for _, value := range []string{"", "industry-chain-master", "industry-chain-membership"} {
		if _, err := ParseApplyScope(value); err != nil {
			t.Fatalf("ParseApplyScope(%q) error = %v", value, err)
		}
	}
	if _, err := ParseApplyScope("membership-only"); err == nil {
		t.Fatal("ParseApplyScope(unknown) error = nil")
	}
}

type masterScopeRepository struct {
	existing            map[string]struct{}
	entities            []Entity
	profileKeys         []string
	sectorMappings      []SectorSourceMapping
	relationships       []Relationship
	industryBatchCalled bool
	industryBatch       IndustryChainBatch
}

func (r *masterScopeRepository) HasActiveLegacySectors(context.Context) (bool, error) {
	return false, nil
}

func (r *masterScopeRepository) UpsertEntity(_ context.Context, entity Entity) (WriteResult, error) {
	r.entities = append(r.entities, entity)
	return WriteResult{Key: entity.Key, Action: r.action(entity.Key)}, nil
}

func (r *masterScopeRepository) UpsertProfile(_ context.Context, profile Profile) (WriteResult, error) {
	r.profileKeys = append(r.profileKeys, profile.EntityKey)
	return WriteResult{Key: profile.EntityKey, Action: r.action(profile.EntityKey)}, nil
}

func (r *masterScopeRepository) UpsertSectorSourceMapping(_ context.Context, mapping SectorSourceMapping) (WriteResult, error) {
	r.sectorMappings = append(r.sectorMappings, mapping)
	return WriteResult{Action: WriteCreated}, nil
}

func (r *masterScopeRepository) UpsertRelationship(_ context.Context, relationship Relationship) (WriteResult, error) {
	r.relationships = append(r.relationships, relationship)
	return WriteResult{Action: WriteCreated}, nil
}

func (r *masterScopeRepository) UpsertIndustryChainBatch(_ context.Context, batch IndustryChainBatch) (IndustryChainWriteReport, error) {
	r.industryBatchCalled = true
	r.industryBatch = batch
	return IndustryChainWriteReport{Created: len(batch.Memberships) + len(batch.TopologyEdges) + len(batch.PhysicalConstraints)}, nil
}

func (r *masterScopeRepository) action(key string) WriteAction {
	if _, ok := r.existing[key]; ok {
		return WriteUpdated
	}
	return WriteCreated
}
