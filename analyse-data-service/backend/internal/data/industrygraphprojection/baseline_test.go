package industrygraphprojection

import (
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	graphbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/industrygraphprojection"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/industryrelationshipimport"
)

func TestLoadFrozenV1CSVBaselineLoadsFrozenIndustryGraph(t *testing.T) {
	t.Parallel()

	pkg, err := industryrelationshipimport.LoadDirectory(
		frozenRelationshipPackageDirectory(t),
		graphbiz.FrozenV1PackageSHA256,
	)
	if err != nil {
		t.Fatalf("LoadDirectory() error = %v", err)
	}

	projection, err := LoadFrozenV1CSVBaseline(pkg)
	if err != nil {
		t.Fatalf("LoadFrozenV1CSVBaseline() error = %v", err)
	}
	if projection.PackageSHA256 != graphbiz.FrozenV1PackageSHA256 {
		t.Fatalf(
			"PackageSHA256 = %q, want %q",
			projection.PackageSHA256,
			graphbiz.FrozenV1PackageSHA256,
		)
	}
	if got, want := len(projection.Nodes), 4449; got != want {
		t.Fatalf("node count = %d, want %d", got, want)
	}
	if got, want := len(projection.Relationships), 7867; got != want {
		t.Fatalf("relationship count = %d, want %d", got, want)
	}

	nodeCounts := make(map[graphbiz.EntityType]int)
	for _, node := range projection.Nodes {
		nodeCounts[node.EntityType]++
		if node.Aliases == nil {
			t.Fatalf("node %q aliases = nil, want an explicit empty or populated list", node.EntityKey)
		}
	}
	wantNodeCounts := map[graphbiz.EntityType]int{
		graphbiz.EntityTypeIndustry:      512,
		graphbiz.EntityTypeConcept:       180,
		graphbiz.EntityTypeIndustryChain: 708,
		graphbiz.EntityTypeChainNode:     3049,
	}
	if !reflect.DeepEqual(nodeCounts, wantNodeCounts) {
		t.Fatalf("node type counts = %#v, want %#v", nodeCounts, wantNodeCounts)
	}

	relationshipCounts := make(map[graphbiz.RelationshipType]int)
	for _, relationship := range projection.Relationships {
		relationshipCounts[relationship.Type]++
		if relationship.Type == graphbiz.RelationshipTypeHasNode {
			if relationship.Position == nil || *relationship.Position <= 0 {
				t.Fatalf(
					"HAS_NODE %q position = %v, want positive integer",
					relationship.RelationKey,
					relationship.Position,
				)
			}
		} else if relationship.Position != nil {
			t.Fatalf(
				"%s %q position = %v, want nil",
				relationship.Type,
				relationship.RelationKey,
				relationship.Position,
			)
		}
	}
	wantRelationshipCounts := map[graphbiz.RelationshipType]int{
		graphbiz.RelationshipTypeMappedToIndustry: 716,
		graphbiz.RelationshipTypeMappedToConcept:  521,
		graphbiz.RelationshipTypeHasNode:          3350,
		graphbiz.RelationshipTypeInputTo:          1537,
		graphbiz.RelationshipTypeIsComponentOf:    704,
		graphbiz.RelationshipTypeDependsOn:        404,
		graphbiz.RelationshipTypeIsSubcategoryOf:  635,
	}
	if !reflect.DeepEqual(relationshipCounts, wantRelationshipCounts) {
		t.Fatalf(
			"relationship type counts = %#v, want %#v",
			relationshipCounts,
			wantRelationshipCounts,
		)
	}
}

func TestLoadFrozenV1CSVBaselineRejectsUnpinnedManifestValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		mutate            func(*industryrelationshipimport.Package)
		wantErrorContains string
	}{
		{
			name: "package SHA",
			mutate: func(pkg *industryrelationshipimport.Package) {
				pkg.Manifest.PackageSHA256 = strings.Repeat("a", 64)
			},
			wantErrorContains: "frozen V1 package SHA-256",
		},
		{
			name: "nodes CSV SHA",
			mutate: func(pkg *industryrelationshipimport.Package) {
				descriptor := pkg.Manifest.ProjectionFiles[nodesProjectionPath]
				descriptor.SHA256 = strings.Repeat("a", 64)
				pkg.Manifest.ProjectionFiles[nodesProjectionPath] = descriptor
			},
			wantErrorContains: "frozen V1 SHA-256",
		},
		{
			name: "relationships CSV SHA",
			mutate: func(pkg *industryrelationshipimport.Package) {
				descriptor := pkg.Manifest.ProjectionFiles[relationshipsProjectionPath]
				descriptor.SHA256 = strings.Repeat("a", 64)
				pkg.Manifest.ProjectionFiles[relationshipsProjectionPath] = descriptor
			},
			wantErrorContains: "frozen V1 SHA-256",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			pkg := loadFrozenRelationshipPackage(t)
			pkg.Manifest.ProjectionFiles = cloneProjectionDescriptors(pkg.Manifest.ProjectionFiles)
			test.mutate(&pkg)

			_, err := LoadFrozenV1CSVBaseline(pkg)
			if err == nil || !strings.Contains(err.Error(), test.wantErrorContains) {
				t.Fatalf(
					"LoadFrozenV1CSVBaseline() error = %v, want error containing %q",
					err,
					test.wantErrorContains,
				)
			}
		})
	}
}

func TestParseNodeCSVRejectsMalformedRows(t *testing.T) {
	t.Parallel()

	const validNodes = `"entity_id:ID","entity_key","canonical_name","aliases",":LABEL"
"chain","industry_chain:test","测试产业链","","IndustryChain"
"node-a","chain_node:a","节点甲","","ChainNode"
`
	tests := []struct {
		name              string
		nodes             string
		wantErrorContains string
	}{
		{
			name: "header",
			nodes: strings.Replace(
				validNodes,
				`"aliases",":LABEL"`,
				`"aliases","label"`,
				1,
			),
			wantErrorContains: "header",
		},
		{
			name: "label",
			nodes: strings.Replace(
				validNodes,
				`"","IndustryChain"`,
				`"","Theme"`,
				1,
			),
			wantErrorContains: "unsupported node label",
		},
		{
			name: "aliases",
			nodes: strings.Replace(
				validNodes,
				`"节点甲","","ChainNode"`,
				`"节点甲","别名甲||别名乙","ChainNode"`,
				1,
			),
			wantErrorContains: "aliases",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseNodeCSV([]byte(test.nodes))
			if err == nil || !strings.Contains(err.Error(), test.wantErrorContains) {
				t.Fatalf(
					"parseNodeCSV() error = %v, want error containing %q",
					err,
					test.wantErrorContains,
				)
			}
		})
	}
}

func TestParseRelationshipCSVRejectsMalformedRows(t *testing.T) {
	t.Parallel()

	const validRelationships = `":START_ID",":END_ID",":TYPE","chain_id","relation_key","contextual_stage","position","mechanism"
"chain","node-a","HAS_NODE","chain","industry_chain:test|has_node|chain_node:a","upstream","1","节点甲属于本链上游"
"node-a","node-b","INPUT_TO","chain","industry_chain:test|chain_node:a|input_to|chain_node:b","","","节点甲进入节点乙"
`
	tests := []struct {
		name              string
		relationships     string
		wantErrorContains string
	}{
		{
			name: "header",
			relationships: strings.Replace(
				validRelationships,
				`"position","mechanism"`,
				`"position","reason"`,
				1,
			),
			wantErrorContains: "header",
		},
		{
			name: "relationship type",
			relationships: strings.Replace(
				validRelationships,
				`"INPUT_TO"`,
				`"RELATED_TO"`,
				1,
			),
			wantErrorContains: "unsupported relationship type",
		},
		{
			name: "has-node position",
			relationships: strings.Replace(
				validRelationships,
				`"upstream","1"`,
				`"upstream","01"`,
				1,
			),
			wantErrorContains: "canonical positive integer",
		},
		{
			name: "non-membership position",
			relationships: strings.Replace(
				validRelationships,
				`input_to|chain_node:b","",""`,
				`input_to|chain_node:b","","1"`,
				1,
			),
			wantErrorContains: "must not contain position",
		},
		{
			name: "non-membership contextual stage",
			relationships: strings.Replace(
				validRelationships,
				`input_to|chain_node:b","",""`,
				`input_to|chain_node:b","upstream",""`,
				1,
			),
			wantErrorContains: "must not contain contextual_stage",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseRelationshipCSV([]byte(test.relationships))
			if err == nil || !strings.Contains(err.Error(), test.wantErrorContains) {
				t.Fatalf(
					"parseRelationshipCSV() error = %v, want error containing %q",
					err,
					test.wantErrorContains,
				)
			}
		})
	}
}

func loadFrozenRelationshipPackage(t *testing.T) industryrelationshipimport.Package {
	t.Helper()

	pkg, err := industryrelationshipimport.LoadDirectory(
		frozenRelationshipPackageDirectory(t),
		graphbiz.FrozenV1PackageSHA256,
	)
	if err != nil {
		t.Fatalf("LoadDirectory() error = %v", err)
	}
	return pkg
}

func frozenRelationshipPackageDirectory(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve baseline test source path")
	}
	return filepath.Clean(filepath.Join(
		filepath.Dir(currentFile),
		"..", "..", "..", "data", "industry_relationships", "2026-07-27-v1",
	))
}

func cloneProjectionDescriptors(
	descriptors map[string]industryrelationshipimport.FileDescriptor,
) map[string]industryrelationshipimport.FileDescriptor {
	cloned := make(map[string]industryrelationshipimport.FileDescriptor, len(descriptors))
	for name, descriptor := range descriptors {
		cloned[name] = descriptor
	}
	return cloned
}
