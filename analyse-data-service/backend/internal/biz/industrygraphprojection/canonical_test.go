package industrygraphprojection

import (
	"testing"
)

func TestProjectionSummaryIsIndependentOfInputOrder(t *testing.T) {
	left := validTestProjection()
	right := cloneProjection(left)
	reverseNodes(right.Nodes)
	reverseRelationships(right.Relationships)

	leftSummary := SummarizeProjection(left)
	rightSummary := SummarizeProjection(right)
	if leftSummary.NodeFingerprint != rightSummary.NodeFingerprint {
		t.Fatalf("node fingerprints differ: %s != %s", leftSummary.NodeFingerprint, rightSummary.NodeFingerprint)
	}
	if leftSummary.RelationshipFingerprint != rightSummary.RelationshipFingerprint {
		t.Fatalf(
			"relationship fingerprints differ: %s != %s",
			leftSummary.RelationshipFingerprint,
			rightSummary.RelationshipFingerprint,
		)
	}
	if !ProjectionsEqual(left, right) {
		t.Fatal("ProjectionsEqual() = false for reordered semantic sets")
	}
}

func TestProjectionSummaryReportsTypeCountsAndSemanticChanges(t *testing.T) {
	original := validTestProjection()
	changed := original
	changed.Relationships = append([]Relationship(nil), original.Relationships...)
	changed.Relationships[0].Mechanism = "changed mechanism"

	summary := SummarizeProjection(original)
	if summary.NodeCount != 5 || summary.RelationshipCount != 5 {
		t.Fatalf("summary counts = %d/%d, want 5/5", summary.NodeCount, summary.RelationshipCount)
	}
	if summary.NodeTypeCounts[EntityTypeIndustryChain] != 1 ||
		summary.NodeTypeCounts[EntityTypeChainNode] != 2 {
		t.Fatalf("node type counts = %#v", summary.NodeTypeCounts)
	}
	if summary.RelationshipTypeCounts[RelationshipTypeHasNode] != 2 ||
		summary.RelationshipTypeCounts[RelationshipTypeInputTo] != 1 {
		t.Fatalf("relationship type counts = %#v", summary.RelationshipTypeCounts)
	}
	if ProjectionsEqual(original, changed) {
		t.Fatal("ProjectionsEqual() = true after a semantic relationship change")
	}
	if summary.RelationshipFingerprint == SummarizeProjection(changed).RelationshipFingerprint {
		t.Fatal("relationship fingerprint did not change")
	}
}
