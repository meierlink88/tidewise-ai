package industrygraphprojection

import (
	"context"
	"strings"
	"testing"
)

func TestProjectReturnsUnchangedWhenStoredGraphMatchesSnapshot(t *testing.T) {
	t.Parallel()

	baseline := validTestProjection()
	source := &fakeSnapshotReader{projection: baseline}
	store := &fakeProjectionStore{
		state: ProjectionState{
			Projection:      baseline,
			ContractVersion: ContractVersion,
			PackageSHA256:   baseline.PackageSHA256,
		},
	}
	service := NewService(source, store)

	result, err := service.Project(context.Background(), ProjectRequest{
		Baseline: baseline,
		Apply:    true,
	})
	if err != nil {
		t.Fatalf("Project() error = %v", err)
	}
	if !result.Unchanged || result.Applied || result.DryRun {
		t.Fatalf("Project() flags = %+v, want unchanged apply replay", result)
	}
	if store.replaced {
		t.Fatal("Project() replaced an already matching graph")
	}
	if source.expectedPackageSHA != baseline.PackageSHA256 {
		t.Fatalf("snapshot package SHA = %q, want %q", source.expectedPackageSHA, baseline.PackageSHA256)
	}
	if result.Source.NodeFingerprint == "" ||
		result.Source.NodeFingerprint != result.CurrentNeo4j.NodeFingerprint ||
		result.Source.RelationshipFingerprint != result.FinalNeo4j.RelationshipFingerprint {
		t.Fatalf("Project() summaries = %+v, want matching source/current/final fingerprints", result)
	}
}

func TestProjectReportsDryRunWhenStoredGraphAlreadyMatches(t *testing.T) {
	t.Parallel()

	baseline := validTestProjection()
	service := NewService(
		&fakeSnapshotReader{projection: baseline},
		&fakeProjectionStore{state: ProjectionState{
			Projection:      baseline,
			ContractVersion: ContractVersion,
			PackageSHA256:   baseline.PackageSHA256,
		}},
	)

	result, err := service.Project(context.Background(), ProjectRequest{Baseline: baseline})
	if err != nil {
		t.Fatalf("Project() error = %v", err)
	}
	if !result.DryRun || !result.Unchanged || result.Applied {
		t.Fatalf("Project() flags = %+v, want unchanged dry-run", result)
	}
}

func TestProjectComparesProjectionAsCanonicalSets(t *testing.T) {
	t.Parallel()

	baseline := validTestProjection()
	reordered := cloneProjection(baseline)
	reverseNodes(reordered.Nodes)
	reverseRelationships(reordered.Relationships)
	service := NewService(
		&fakeSnapshotReader{projection: reordered},
		&fakeProjectionStore{},
	)

	result, err := service.Project(context.Background(), ProjectRequest{
		Baseline: baseline,
	})
	if err != nil {
		t.Fatalf("Project() error = %v", err)
	}
	if !result.DryRun || result.Applied || result.Unchanged {
		t.Fatalf("Project() flags = %+v, want dry-run with a missing projection", result)
	}
}

func TestProjectReplacesSemanticallyMatchingGraphWithIntegrityViolations(t *testing.T) {
	t.Parallel()

	baseline := validTestProjection()
	source := &fakeSnapshotReader{projection: baseline}
	store := &fakeProjectionStore{
		state: ProjectionState{
			Projection:              baseline,
			ContractVersion:         ContractVersion,
			PackageSHA256:           baseline.PackageSHA256,
			IntegrityViolationCount: 1,
		},
		replacementState: ProjectionState{
			Projection:      baseline,
			ContractVersion: ContractVersion,
			PackageSHA256:   baseline.PackageSHA256,
		},
	}

	result, err := NewService(source, store).Project(context.Background(), ProjectRequest{
		Baseline: baseline,
		Apply:    true,
	})
	if err != nil {
		t.Fatalf("Project() error = %v", err)
	}
	if !result.Applied || result.Unchanged || !store.replaced {
		t.Fatalf("Project() result/store = %+v/%+v, want integrity repair", result, store)
	}
	if result.CurrentIntegrityViolationCount != 1 || result.FinalIntegrityViolationCount != 0 {
		t.Fatalf("Project() integrity counts = %d/%d, want 1/0",
			result.CurrentIntegrityViolationCount,
			result.FinalIntegrityViolationCount,
		)
	}
}

func TestProjectRejectsChainScopedRelationshipWithoutChainID(t *testing.T) {
	t.Parallel()

	invalid := validTestProjection()
	invalid.Relationships[len(invalid.Relationships)-1].ChainID = ""
	store := &fakeProjectionStore{}
	service := NewService(&fakeSnapshotReader{projection: invalid}, store)

	_, err := service.Project(context.Background(), ProjectRequest{Baseline: invalid})
	if err == nil || !strings.Contains(err.Error(), "chain_id") {
		t.Fatalf("Project() error = %v, want missing chain_id rejection", err)
	}
	if store.inspected {
		t.Fatal("Project() inspected Neo4j before validating the source graph")
	}
}

func TestProjectRejectsRelationshipWithMissingEndpoint(t *testing.T) {
	t.Parallel()

	invalid := validTestProjection()
	invalid.Relationships[0].ToEntityID = "missing-industry"
	store := &fakeProjectionStore{}
	service := NewService(&fakeSnapshotReader{projection: invalid}, store)

	_, err := service.Project(context.Background(), ProjectRequest{Baseline: invalid})
	if err == nil || !strings.Contains(err.Error(), "missing endpoint") {
		t.Fatalf("Project() error = %v, want missing endpoint rejection", err)
	}
	if store.inspected {
		t.Fatal("Project() inspected Neo4j before validating relationship endpoints")
	}
}

func TestProjectRejectsChainEdgeOutsideMembershipClosure(t *testing.T) {
	t.Parallel()

	invalid := validTestProjection()
	invalid.Relationships[len(invalid.Relationships)-1].ChainID = "another-chain"
	store := &fakeProjectionStore{}
	service := NewService(&fakeSnapshotReader{projection: invalid}, store)

	_, err := service.Project(context.Background(), ProjectRequest{Baseline: invalid})
	if err == nil || !strings.Contains(err.Error(), "active memberships") {
		t.Fatalf("Project() error = %v, want membership closure rejection", err)
	}
	if store.inspected {
		t.Fatal("Project() inspected Neo4j before validating the chain membership closure")
	}
}

func TestProjectRejectsHierarchyCycle(t *testing.T) {
	t.Parallel()

	invalid := validTestProjection()
	invalid.Nodes = append(invalid.Nodes, Node{
		EntityID: "industry-parent", EntityKey: "industry:parent",
		EntityType: EntityTypeIndustry, CanonicalName: "父行业",
	})
	invalid.Relationships = append(invalid.Relationships,
		Relationship{
			FromEntityID: "industry", ToEntityID: "industry-parent",
			Type:        RelationshipTypeIsSubcategoryOf,
			RelationKey: "industry:test|is_subcategory_of|industry:parent",
			Mechanism:   "行业分类层级",
		},
		Relationship{
			FromEntityID: "industry-parent", ToEntityID: "industry",
			Type:        RelationshipTypeIsSubcategoryOf,
			RelationKey: "industry:parent|is_subcategory_of|industry:test",
			Mechanism:   "反向循环",
		},
	)
	store := &fakeProjectionStore{}
	service := NewService(&fakeSnapshotReader{projection: invalid}, store)

	_, err := service.Project(context.Background(), ProjectRequest{Baseline: invalid})
	if err == nil || !strings.Contains(err.Error(), "hierarchy") || !strings.Contains(err.Error(), "acyclic") {
		t.Fatalf("Project() error = %v, want hierarchy cycle rejection", err)
	}
	if store.inspected {
		t.Fatal("Project() inspected Neo4j before validating hierarchy acyclicity")
	}
}

type fakeSnapshotReader struct {
	projection         Projection
	err                error
	expectedPackageSHA string
}

func (f *fakeSnapshotReader) ReadIndustryGraphSnapshot(_ context.Context, expectedPackageSHA string) (Projection, error) {
	f.expectedPackageSHA = expectedPackageSHA
	return f.projection, f.err
}

type fakeProjectionStore struct {
	state            ProjectionState
	replacementState ProjectionState
	err              error
	inspected        bool
	replaced         bool
}

func (f *fakeProjectionStore) InspectIndustryGraph(context.Context, string) (ProjectionState, error) {
	f.inspected = true
	return f.state, f.err
}

func (f *fakeProjectionStore) ReplaceIndustryGraph(context.Context, string, Projection) (ProjectionState, error) {
	f.replaced = true
	if f.replacementState.Projection.PackageSHA256 != "" {
		return f.replacementState, f.err
	}
	return f.state, f.err
}

func validTestProjection() Projection {
	positionOne := 1
	positionTwo := 2
	return Projection{
		PackageSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Nodes: []Node{
			{EntityID: "chain", EntityKey: "industry_chain:test", EntityType: EntityTypeIndustryChain, CanonicalName: "测试产业链"},
			{EntityID: "concept", EntityKey: "concept:test", EntityType: EntityTypeConcept, CanonicalName: "测试概念"},
			{EntityID: "industry", EntityKey: "industry:test", EntityType: EntityTypeIndustry, CanonicalName: "测试行业"},
			{EntityID: "node-a", EntityKey: "chain_node:a", EntityType: EntityTypeChainNode, CanonicalName: "节点甲"},
			{EntityID: "node-b", EntityKey: "chain_node:b", EntityType: EntityTypeChainNode, CanonicalName: "节点乙"},
		},
		Relationships: []Relationship{
			{
				FromEntityID: "chain", ToEntityID: "industry",
				Type: RelationshipTypeMappedToIndustry, ChainID: "chain",
				RelationKey: "industry_chain:test|mapped_to_industry|industry:test",
				Mechanism:   "产业链边界与行业定义相符",
			},
			{
				FromEntityID: "chain", ToEntityID: "concept",
				Type: RelationshipTypeMappedToConcept, ChainID: "chain",
				RelationKey: "industry_chain:test|mapped_to_concept|concept:test",
				Mechanism:   "产业链范围与概念边界相符",
			},
			{
				FromEntityID: "chain", ToEntityID: "node-a",
				Type: RelationshipTypeHasNode, ChainID: "chain",
				RelationKey:     "industry_chain:test|has_node|chain_node:a",
				ContextualStage: "upstream", Position: &positionOne,
				Mechanism: "节点甲属于本链上游",
			},
			{
				FromEntityID: "chain", ToEntityID: "node-b",
				Type: RelationshipTypeHasNode, ChainID: "chain",
				RelationKey:     "industry_chain:test|has_node|chain_node:b",
				ContextualStage: "downstream", Position: &positionTwo,
				Mechanism: "节点乙属于本链下游",
			},
			{
				FromEntityID: "node-a", ToEntityID: "node-b",
				Type: RelationshipTypeInputTo, ChainID: "chain",
				RelationKey: "industry_chain:test|chain_node:a|input_to|chain_node:b",
				Mechanism:   "节点甲的产出进入节点乙",
			},
		},
	}
}

func cloneProjection(value Projection) Projection {
	cloned := value
	cloned.Nodes = append([]Node(nil), value.Nodes...)
	for index := range cloned.Nodes {
		cloned.Nodes[index].Aliases = append([]string(nil), value.Nodes[index].Aliases...)
	}
	cloned.Relationships = append([]Relationship(nil), value.Relationships...)
	for index := range cloned.Relationships {
		if value.Relationships[index].Position != nil {
			position := *value.Relationships[index].Position
			cloned.Relationships[index].Position = &position
		}
	}
	return cloned
}

func reverseNodes(values []Node) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}

func reverseRelationships(values []Relationship) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}
