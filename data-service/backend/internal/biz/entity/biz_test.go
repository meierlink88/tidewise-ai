package entity

import (
	"testing"
)

func TestDomainObjectIdentityRequiresPrefixAndUUID(t *testing.T) {
	for value, want := range map[string]bool{
		"ORG550e8400-e29b-41d4-a716-446655440000":  true,
		"ORG_550e8400-e29b-41d4-a716-446655440000": false,
		"ORG550E8400-E29B-41D4-A716-446655440000":  false,
		"6f845f9f-10e2-44dd-b08a-e482e32d3558":     false,
	} {
		if got := IsOrganizationID(value); got != want {
			t.Errorf("IsOrganizationID(%q) = %v, want %v", value, got, want)
		}
	}
}

func TestObjectIdentityRecognizesIndependentDomainPrefixes(t *testing.T) {
	suffix := "550e8400-e29b-41d4-a716-446655440000"
	for objectType, prefix := range map[string]string{
		ObjectTypeIndustry:      "IND",
		ObjectTypeConcept:       "CON",
		ObjectTypeChainNode:     "CND",
		ObjectTypeIndustryChain: "ICH",
	} {
		value := prefix + suffix
		if !IsObjectID(value) {
			t.Errorf("IsObjectID(%q) = false", value)
		}
		if !ObjectTypeMatchesID(objectType, value) {
			t.Errorf("ObjectTypeMatchesID(%q, %q) = false", objectType, value)
		}
		if ObjectTypeMatchesID(objectType, "ENT"+suffix) {
			t.Errorf("ObjectTypeMatchesID(%q, legacy ENT ID) = true", objectType)
		}
	}
}

func TestIndustryChainTopologyTypesValidateRetainedVocabulary(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{ Validate() error }
		wantErr bool
	}{
		{
			name: "membership",
			value: IndustryChainNodeMembership{
				IndustryChainID: "chain", ChainNodeID: "node", Position: 1,
				ContextualStage: IndustryChainContextualStageUpstream,
			},
		},
		{
			name: "direct graph edge",
			value: IndustryChainGraphEdge{
				ID: "edge", IndustryChainID: "chain", FromChainNodeID: "a", ToChainNodeID: "b",
				RelationType: IndustryChainGraphRelationInputTo,
			},
		},
		{
			name: "legacy stage is rejected",
			value: IndustryChainNodeMembership{
				IndustryChainID: "chain", ChainNodeID: "node", Position: 1,
				ContextualStage: IndustryChainContextualStage("infrastructure"),
			},
			wantErr: true,
		},
		{
			name: "self edge is rejected",
			value: IndustryChainGraphEdge{
				ID: "edge", IndustryChainID: "chain", FromChainNodeID: "a", ToChainNodeID: "a",
				RelationType: IndustryChainGraphRelationDependsOn,
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.value.Validate()
			if (err != nil) != testCase.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func TestRetiredGenericEntityIdentityIsNotAResearchObject(t *testing.T) {
	id := "ENT550e8400-e29b-41d4-a716-446655440000"
	if IsObjectID(id) || ObjectTypeMatchesID("security", id) {
		t.Fatal("retired generic Entity accepted")
	}
}
