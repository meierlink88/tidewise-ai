package industryrelationshipimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/identity"
)

func TestServicePreflightApplyAndReplay(t *testing.T) {
	pkg := validTestPackage(t)
	store := &fakeStore{tx: &fakeTx{}}
	service := NewService(store)
	now := time.Date(2026, 7, 27, 12, 0, 0, 123456789, time.UTC)
	service.now = func() time.Time { return now }

	preflight, err := service.Preflight(context.Background(), "test-caller", pkg)
	if err != nil {
		t.Fatal(err)
	}
	if !preflight.DryRun || preflight.Unchanged || store.tx.applied || store.tx.receipt != nil {
		t.Fatalf("preflight = %#v, state = %#v", preflight, store.tx)
	}

	first, err := service.Import(context.Background(), "test-caller", pkg)
	if err != nil {
		t.Fatal(err)
	}
	if first.DryRun || first.Unchanged || first.ReceiptID == "" || !store.tx.applied ||
		store.tx.receipt == nil {
		t.Fatalf("first result/state = %#v/%#v", first, store.tx)
	}
	if first.ImportedAt != now.Truncate(time.Microsecond) {
		t.Fatalf("imported_at = %s", first.ImportedAt)
	}

	second, err := service.Import(context.Background(), "test-caller", pkg)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Unchanged || second.DryRun || second.ReceiptID != first.ReceiptID {
		t.Fatalf("replay = %#v", second)
	}
	if store.tx.insertCalls != 1 || store.tx.verifyCalls != 2 {
		t.Fatalf("insert/verify calls = %d/%d", store.tx.insertCalls, store.tx.verifyCalls)
	}
}

func TestServiceRejectsCallerConflictAndRollsBackFailedInsert(t *testing.T) {
	pkg := validTestPackage(t)
	store := &fakeStore{tx: &fakeTx{}}
	service := NewService(store)
	if _, err := service.Import(context.Background(), "first", pkg); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Import(context.Background(), "other", pkg); !errors.Is(err, ErrCallerConflict) {
		t.Fatalf("caller conflict = %v", err)
	}

	failing := &fakeStore{tx: &fakeTx{insertError: errors.New("synthetic insert failure")}}
	if _, err := NewService(failing).Import(context.Background(), "first", pkg); err == nil {
		t.Fatal("failed insert was accepted")
	}
	if failing.tx.applied || failing.tx.receipt != nil || failing.commits != 0 || failing.rollbacks != 1 {
		t.Fatalf("failed state = %#v, commits/rollbacks=%d/%d", failing.tx, failing.commits, failing.rollbacks)
	}
}

type fakeStore struct {
	tx        *fakeTx
	commits   int
	rollbacks int
}

func (s *fakeStore) InIndustryRelationshipImportTransaction(
	_ context.Context,
	fn func(Transaction) error,
) error {
	beforeApplied, beforeReceipt := s.tx.applied, s.tx.receipt
	if err := fn(s.tx); err != nil {
		s.tx.applied, s.tx.receipt = beforeApplied, beforeReceipt
		s.rollbacks++
		return err
	}
	s.commits++
	return nil
}

type fakeTx struct {
	receipt     *Receipt
	applied     bool
	insertError error
	insertCalls int
	verifyCalls int
}

func (f *fakeTx) LockIndustryRelationshipPackage(context.Context, string) error { return nil }
func (f *fakeTx) IndustryRelationshipImportReceipt(context.Context, string) (*Receipt, error) {
	return f.receipt, nil
}
func (f *fakeTx) PreflightIndustryRelationshipPackage(context.Context, Package) error { return nil }
func (f *fakeTx) InsertIndustryRelationshipPackage(context.Context, Package) error {
	f.insertCalls++
	if f.insertError != nil {
		return f.insertError
	}
	f.applied = true
	return nil
}
func (f *fakeTx) VerifyIndustryRelationshipPackage(context.Context, Package) error {
	f.verifyCalls++
	if !f.applied {
		return errors.New("package is not persisted")
	}
	return nil
}
func (f *fakeTx) InsertIndustryRelationshipImportReceipt(_ context.Context, receipt Receipt) error {
	copy := receipt
	f.receipt = &copy
	return nil
}

func validTestPackage(t *testing.T) Package {
	t.Helper()
	now := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	nodeAKey := "chain_node:test_a"
	nodeBKey := "chain_node:test_b"
	nodeAID := identity.NormalizeUUID("entity", nodeAKey)
	nodeBID := identity.NormalizeUUID("entity", nodeBKey)
	industryKey := "industry:test"
	industryID := identity.NormalizeUUID("entity", industryKey)
	pkg := Package{
		Manifest: Manifest{
			SchemaVersion: ManifestSchemaVersion, PackageVersion: "test-v1",
			PackageStatus: PackageStatusApproved, ApprovalBasis: ApprovalBasis,
			PackageSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			PackageCounts: make(map[string]int),
			RelationSpec: RelationSpecDescriptor{
				Version: RelationSpecVersion,
				SHA256:  "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			},
		},
		ManifestSHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		Evidence:       []json.RawMessage{json.RawMessage(`{"evidence_id":"test"}`)},
		ValidationReport: ValidationReport{
			SchemaVersion: "industry_relationship_validation_report_v1",
			Status:        "passed", ApprovalBasis: ApprovalBasis, VerifiedAt: now,
			PackageCounts: make(map[string]int),
			HardGates: map[string]any{
				"chain_topology_decision_coverage":                  "708/708",
				"chain_mapping_coverage":                            "708/708",
				"chain_topologies_weakly_connected":                 true,
				"chain_topologies_acyclic":                          true,
				"topology_semantic_audit_status":                    "passed",
				"topology_semantic_error_count":                     0,
				"topology_semantic_unreviewed_warning_count":        0,
				"topology_global_closure_status":                    "passed",
				"topology_global_closure_supplement_relation_count": 586,
				"orphan_membership_count":                           0,
				"frozen_chain_node_without_formal_relation_count":   0,
				"orphan_industry_count":                             0,
				"projected_concept_without_mapping_count":           0,
				"neo4j_projected_orphan_count":                      0,
				"unresolved_endpoint_count":                         0,
				"unresolved_evidence_count":                         0,
				"duplicate_semantic_relation_count":                 0,
				"candidate_relation_count":                          0,
				"unmapped_relation_candidate_count":                 0,
			},
			ClosedWorldNote: "closed for the test fixture",
		},
	}
	for index := 0; index < 194; index++ {
		pkg.ConceptDispositions = append(pkg.ConceptDispositions, mustRawJSON(t, conceptDisposition{
			ConceptKey:   fmt.Sprintf("concept:test_%03d", index),
			Disposition:  "needs_chain_expansion",
			ReviewStatus: "approved",
			Status:       "active",
		}))
	}
	pkg.NodeDispositions = append(
		pkg.NodeDispositions,
		mustRawJSON(t, nodeDisposition{
			NodeKey: nodeAKey, Disposition: "hierarchy_parent",
			ReviewStatus: "approved", Status: "active",
		}),
		mustRawJSON(t, nodeDisposition{
			NodeKey: nodeBKey, Disposition: "connected_by_discovery",
			ReviewStatus: "approved", Status: "active",
		}),
	)
	for index := 0; index < 586; index++ {
		nodeKey := fmt.Sprintf("chain_node:test_disposition_%03d", index)
		nodeID := identity.NormalizeUUID("entity", nodeKey)
		tuple := fmt.Sprintf("%s|is_subcategory_of|%s", nodeID, nodeAID)
		relationID := identity.NormalizeUUID("chain_node_relation", tuple)
		pkg.NodeDispositions = append(pkg.NodeDispositions, mustRawJSON(t, nodeDisposition{
			NodeKey: nodeKey, Disposition: "hierarchy_child",
			ReviewStatus: "approved", Status: "active",
		}))
		pkg.GlobalRelations = append(pkg.GlobalRelations, GlobalChainNodeRelation{
			ID: relationID, RelationKey: "chain_node_relation:" + relationID,
			FromChainNodeEntityID: nodeID, FromNodeKey: nodeKey,
			FromName:            fmt.Sprintf("test disposition %03d", index),
			RelationType:        "is_subcategory_of",
			ToChainNodeEntityID: nodeAID, ToNodeKey: nodeAKey, ToName: "test A",
			Mechanism: "strict test subset", EvidenceNote: "test definition evidence",
			Provenance: "artifact://test/evidence", Confidence: "high",
			ReviewStatus: "approved", Status: "active", VerifiedAt: now,
		})
	}
	for index := 0; index < 708; index++ {
		chainKey := fmt.Sprintf("industry_chain:test_%04d", index)
		chainID := identity.NormalizeUUID("entity", chainKey)
		pkg.IndustryChains = append(pkg.IndustryChains, IndustryChain{
			EntityID: chainID, EntityKey: chainKey, EntityType: "industry_chain",
			LayerCode: "industry_chain", Name: chainKey, CanonicalName: chainKey,
			Status: "active", Scope: "scope", TargetOutput: "output", EndUse: "end use",
			Geography: "global", ObservableVariables: []string{"volume"},
			AsOfDate: "2026-07-27", ReviewStatus: "approved", ReviewNote: "reviewed",
			RelationshipApprovalBasis: ApprovalBasis,
		})
		relationKey := fmt.Sprintf("%s|mapped_to_industry|%s", chainKey, industryKey)
		pkg.IndustryMappings = append(pkg.IndustryMappings, EntityMapping{
			RelationID:  identity.NormalizeUUID("entity_relationship", relationKey),
			RelationKey: relationKey, FromKey: chainKey, FromEntityID: chainID,
			RelationType: "mapped_to_industry", ToKey: industryKey, ToEntityID: industryID,
			MappingReason: "direct scope", EvidenceIDs: []string{"test"},
			EvidenceNote: "direct scope evidence=test", ReviewStatus: "approved",
			Status: "active", VerifiedAt: now,
		})
		for position, node := range []struct{ key, id, stage string }{
			{nodeAKey, nodeAID, "upstream"},
			{nodeBKey, nodeBID, "downstream"},
		} {
			pkg.Memberships = append(pkg.Memberships, Membership{
				RelationKey:           fmt.Sprintf("%s|has_node|%s", chainKey, node.key),
				IndustryChainEntityID: chainID, ChainKey: chainKey,
				ChainNodeEntityID: node.id, NodeKey: node.key,
				ContextualStage: node.stage, Position: position + 1,
				InclusionReason: "direct member", EvidenceIDs: []string{"test"},
				SourceName: "test", SourceURL: "artifact://test/evidence", VerifiedAt: now,
				ReviewStatus: "approved", Status: "active",
			})
		}
		edgeKey := fmt.Sprintf("%s|%s|input_to|%s", chainKey, nodeAKey, nodeBKey)
		pkg.GraphEdges = append(pkg.GraphEdges, GraphEdge{
			ID:          identity.NormalizeUUID("industry_chain_graph_edge", edgeKey),
			RelationKey: edgeKey, IndustryChainEntityID: chainID, ChainKey: chainKey,
			FromChainNodeEntityID: nodeAID, FromNodeKey: nodeAKey, RelationType: "input_to",
			ToChainNodeEntityID: nodeBID, ToNodeKey: nodeBKey, Mechanism: "direct input",
			SegmentKind: "direct_candidate", EvidenceIDs: []string{"test"},
			SourceName: "test", SourceURL: "artifact://test/evidence", VerifiedAt: now,
			ReviewStatus: "approved", Status: "active",
		})
	}
	counts := countsMap(pkg.Counts())
	pkg.Manifest.PackageCounts = counts
	pkg.ValidationReport.PackageCounts = counts
	return pkg
}

func mustRawJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}
