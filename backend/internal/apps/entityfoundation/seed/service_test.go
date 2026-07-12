package seed

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestServiceAppliesManifestInOrderAndReportsStats(t *testing.T) {
	repo := &recordingRepository{}
	service := NewService(repo)
	manifest := Manifest{
		Entities: []Entity{
			testSeedEntity("economy:cn", domain.EntityTypeEconomy, "economy", `{"country_code":"CN","currency_code":"CNY"}`),
			testSeedEntity("alliance_org:g20", domain.EntityTypeAllianceOrg, "alliance", `{"org_code":"G20","org_type":"economic_forum"}`),
		},
		Relationships: []Relationship{
			{Key: "relationship:cn_member_of_g20", From: "economy:cn", To: "alliance_org:g20", RelationType: "member_of", Status: domain.StatusActive},
		},
	}

	report, err := service.Apply(context.Background(), manifest, ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	wantCalls := []string{
		"entity:economy:cn",
		"entity:alliance_org:g20",
		"profile:economy:cn",
		"profile:alliance_org:g20",
		"relationship:relationship:cn_member_of_g20",
	}
	if !reflect.DeepEqual(repo.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", repo.calls, wantCalls)
	}
	if report.TotalEntities != 2 {
		t.Fatalf("TotalEntities = %d, want 2", report.TotalEntities)
	}
	if report.ByEntityType[domain.EntityTypeEconomy] != 1 || report.ByLayerCode["alliance"] != 1 {
		t.Fatalf("unexpected entity distribution: %#v %#v", report.ByEntityType, report.ByLayerCode)
	}
	if report.ProfileCounts["economy_profiles"] != 1 || report.EdgeCounts["member_of"] != 1 {
		t.Fatalf("unexpected profile/edge counts: %#v %#v", report.ProfileCounts, report.EdgeCounts)
	}
	if report.Created != 5 || report.Updated != 0 || report.Unchanged != 0 || report.Failed != 0 {
		t.Fatalf("unexpected write stats: %#v", report)
	}
}

func TestServiceStopsOnRepositoryError(t *testing.T) {
	repo := &recordingRepository{failOnCall: "profile:economy:cn"}
	service := NewService(repo)
	manifest := Manifest{
		Entities: []Entity{
			testSeedEntity("economy:cn", domain.EntityTypeEconomy, "economy", `{"country_code":"CN","currency_code":"CNY"}`),
		},
		Relationships: []Relationship{
			{Key: "relationship:unused", From: "economy:cn", To: "economy:cn", RelationType: "self"},
		},
	}

	report, err := service.Apply(context.Background(), manifest, ApplyOptions{})
	if err == nil {
		t.Fatal("Apply() error = nil, want repository error")
	}
	if report.Failed != 1 {
		t.Fatalf("Failed = %d, want 1", report.Failed)
	}
	if got, want := repo.calls, []string{"entity:economy:cn", "profile:economy:cn"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %v, want %v", got, want)
	}
}

func TestServiceReportsBenchmarkProfileCounts(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo)
	manifest := Manifest{
		Entities: []Entity{
			testSeedEntity("benchmark:us_10y_treasury_yield", domain.EntityTypeBenchmark, "benchmark", `{
			  "benchmark_type":"government_bond_yield",
			  "official_series_code":null,
			  "provider":"us_treasury",
			  "tenor":"10Y",
			  "currency_code":"USD",
			  "unit":"percent",
			  "frequency":"daily",
			  "source_url":"https://home.treasury.gov/"
			}`),
		},
	}

	first, err := service.Apply(context.Background(), manifest, ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply(first) error = %v", err)
	}
	second, err := service.Apply(context.Background(), manifest, ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply(second) error = %v", err)
	}

	if first.ByEntityType[domain.EntityTypeBenchmark] != 1 || first.ProfileCounts["benchmark_profiles"] != 1 {
		t.Fatalf("first benchmark counts = %#v %#v", first.ByEntityType, first.ProfileCounts)
	}
	if second.Unchanged != 2 || second.Created != 0 || second.ProfileCounts["benchmark_profiles"] != 1 {
		t.Fatalf("second report = %#v, want idempotent unchanged entity and profile", second)
	}
}

func TestServiceSkipsInactiveEntitiesByDefault(t *testing.T) {
	repo := &recordingRepository{}
	service := NewService(repo)
	inactive := testSeedEntity("economy:inactive", domain.EntityTypeEconomy, "economy", `{"country_code":"ZZ","currency_code":"ZZZ"}`)
	inactive.Status = domain.StatusInactive
	manifest := Manifest{
		Entities: []Entity{
			testSeedEntity("economy:cn", domain.EntityTypeEconomy, "economy", `{"country_code":"CN","currency_code":"CNY"}`),
			inactive,
		},
		Relationships: []Relationship{
			{Key: "relationship:skip_inactive", From: "economy:cn", To: "economy:inactive", RelationType: "related_to", Status: domain.StatusActive},
		},
	}

	report, err := service.Apply(context.Background(), manifest, ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if got, want := repo.calls, []string{"entity:economy:cn", "profile:economy:cn"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %v, want %v", got, want)
	}
	if report.Skipped != 2 {
		t.Fatalf("Skipped = %d, want 2", report.Skipped)
	}
}

type recordingRepository struct {
	calls      []string
	failOnCall string
}

func (r *recordingRepository) UpsertEntity(_ context.Context, entity Entity) (WriteResult, error) {
	return r.record("entity:"+entity.Key, entity.Key)
}

func (r *recordingRepository) UpsertProfile(_ context.Context, profile Profile) (WriteResult, error) {
	return r.record("profile:"+profile.EntityKey, profile.EntityKey)
}

func (r *recordingRepository) UpsertRelationship(_ context.Context, relationship Relationship) (WriteResult, error) {
	return r.record("relationship:"+relationship.Key, relationship.Key)
}

func (r *recordingRepository) record(call string, key string) (WriteResult, error) {
	r.calls = append(r.calls, call)
	if call == r.failOnCall {
		return WriteResult{}, fmt.Errorf("forced failure on %s", call)
	}
	return WriteResult{Key: key, Action: WriteCreated}, nil
}

func testSeedEntity(key string, entityType domain.EntityType, layerCode string, profile string) Entity {
	name := key
	aliases := []string(nil)
	if entityType == domain.EntityTypeBenchmark {
		name = "测试基准"
		aliases = []string{"Test Benchmark"}
	}
	return Entity{
		Key:           key,
		EntityType:    entityType,
		LayerCode:     layerCode,
		Name:          name,
		CanonicalName: name,
		Aliases:       aliases,
		Status:        domain.StatusActive,
		Profile:       []byte(profile),
	}
}
