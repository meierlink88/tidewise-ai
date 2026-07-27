package industrygraphprojection

import "fmt"

const (
	FrozenV1PackageSHA256           = "7c737410ac6af562af19f8b9dad9e8e1c802f8f782625bd360bb2e8f20768608"
	FrozenV1NodeFingerprint         = "4229146e37ee554cd58377843743f93dc753bdfd92bbe7f2c9afac61c2003d63"
	FrozenV1RelationshipFingerprint = "aba6be387c0dad1b93c6fd14a4f9216b77a625d206cae9e7b977854f0cacec94"
	frozenV1NodeCount               = 4449
	frozenV1RelationshipCount       = 7867
)

func ValidateFrozenV1Projection(projection Projection) error {
	if projection.PackageSHA256 != FrozenV1PackageSHA256 {
		return fmt.Errorf(
			"package SHA-256 %q, want frozen V1 package SHA-256 %s",
			projection.PackageSHA256,
			FrozenV1PackageSHA256,
		)
	}
	if err := ValidateProjection(projection); err != nil {
		return err
	}

	summary := SummarizeProjection(projection)
	if summary.NodeCount != frozenV1NodeCount {
		return fmt.Errorf(
			"node count %d, want frozen V1 count %d",
			summary.NodeCount,
			frozenV1NodeCount,
		)
	}
	if summary.RelationshipCount != frozenV1RelationshipCount {
		return fmt.Errorf(
			"relationship count %d, want frozen V1 count %d",
			summary.RelationshipCount,
			frozenV1RelationshipCount,
		)
	}
	if summary.NodeFingerprint != FrozenV1NodeFingerprint {
		return fmt.Errorf(
			"node fingerprint %q, want frozen V1 fingerprint %s",
			summary.NodeFingerprint,
			FrozenV1NodeFingerprint,
		)
	}
	if summary.RelationshipFingerprint != FrozenV1RelationshipFingerprint {
		return fmt.Errorf(
			"relationship fingerprint %q, want frozen V1 fingerprint %s",
			summary.RelationshipFingerprint,
			FrozenV1RelationshipFingerprint,
		)
	}

	expectedNodeCounts := map[EntityType]int{
		EntityTypeIndustry:      512,
		EntityTypeConcept:       180,
		EntityTypeIndustryChain: 708,
		EntityTypeChainNode:     3049,
	}
	for entityType, expected := range expectedNodeCounts {
		if summary.NodeTypeCounts[entityType] != expected {
			return fmt.Errorf(
				"%s node count %d, want frozen V1 count %d",
				entityType,
				summary.NodeTypeCounts[entityType],
				expected,
			)
		}
	}

	expectedRelationshipCounts := map[RelationshipType]int{
		RelationshipTypeMappedToIndustry: 716,
		RelationshipTypeMappedToConcept:  521,
		RelationshipTypeHasNode:          3350,
		RelationshipTypeInputTo:          1537,
		RelationshipTypeIsComponentOf:    704,
		RelationshipTypeDependsOn:        404,
		RelationshipTypeIsSubcategoryOf:  635,
	}
	for relationshipType, expected := range expectedRelationshipCounts {
		if summary.RelationshipTypeCounts[relationshipType] != expected {
			return fmt.Errorf(
				"%s relationship count %d, want frozen V1 count %d",
				relationshipType,
				summary.RelationshipTypeCounts[relationshipType],
				expected,
			)
		}
	}
	return nil
}
