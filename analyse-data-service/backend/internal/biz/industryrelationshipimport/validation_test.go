package industryrelationshipimport

import (
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/identity"
)

func TestPackageValidateRejectsUnconnectedFrozenNodeDisposition(t *testing.T) {
	pkg := validTestPackage(t)
	pkg.NodeDispositions[0] = mustRawJSON(t, nodeDisposition{
		NodeKey:      "chain_node:orphan",
		Disposition:  "connected_by_discovery",
		ReviewStatus: "approved",
		Status:       "active",
	})

	err := pkg.Validate()
	if err == nil || !strings.Contains(err.Error(), "has no membership/hierarchy") {
		t.Fatalf("Package.Validate() error = %v, want frozen-node closure failure", err)
	}
}

func TestPackageValidateRejectsMappedConceptWithoutRelation(t *testing.T) {
	pkg := validTestPackage(t)
	pkg.ConceptDispositions[0] = mustRawJSON(t, conceptDisposition{
		ConceptKey:   "concept:test_000",
		Disposition:  "mapped",
		ReviewStatus: "approved",
		Status:       "active",
	})

	err := pkg.Validate()
	if err == nil || !strings.Contains(err.Error(), "has no formal relation") {
		t.Fatalf("Package.Validate() error = %v, want Concept projection closure failure", err)
	}
}

func TestPackageValidateRejectsMembershipRequiredNodeWithHierarchyOnly(t *testing.T) {
	pkg := validTestPackage(t)
	pkg.NodeDispositions[2] = mustRawJSON(t, nodeDisposition{
		NodeKey:      "chain_node:test_disposition_000",
		Disposition:  "membership_required",
		ReviewStatus: "approved",
		Status:       "active",
	})

	err := pkg.Validate()
	if err == nil || !strings.Contains(err.Error(), "has no M2 membership") {
		t.Fatalf("Package.Validate() error = %v, want mandatory membership failure", err)
	}
}

func TestValidateGlobalRelationsRejectsHierarchyCycle(t *testing.T) {
	now := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	firstKey, secondKey := "chain_node:first", "chain_node:second"
	firstID := identity.NormalizeUUID("entity", firstKey)
	secondID := identity.NormalizeUUID("entity", secondKey)
	relation := func(fromID, fromKey, fromName, toID, toKey, toName string) GlobalChainNodeRelation {
		tuple := fromID + "|is_subcategory_of|" + toID
		id := identity.NormalizeUUID("chain_node_relation", tuple)
		return GlobalChainNodeRelation{
			ID: id, RelationKey: "chain_node_relation:" + id,
			FromChainNodeEntityID: fromID, FromNodeKey: fromKey, FromName: fromName,
			RelationType:        "is_subcategory_of",
			ToChainNodeEntityID: toID, ToNodeKey: toKey, ToName: toName,
			Mechanism: "strict subset", EvidenceNote: "definition evidence",
			Provenance: "artifact://test/evidence", Confidence: "high",
			ReviewStatus: "approved", Status: "active", VerifiedAt: now,
		}
	}
	items := []GlobalChainNodeRelation{
		relation(firstID, firstKey, "first", secondID, secondKey, "second"),
		relation(secondID, secondKey, "second", firstID, firstKey, "first"),
	}

	err := validateGlobalRelations(items)
	if err == nil || !strings.Contains(err.Error(), "contains a cycle") {
		t.Fatalf("validateGlobalRelations() error = %v, want cycle rejection", err)
	}
}
