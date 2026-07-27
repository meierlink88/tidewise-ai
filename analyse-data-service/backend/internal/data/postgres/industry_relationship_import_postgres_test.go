package postgres

import (
	"strings"
	"testing"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/industryrelationshipimport"
)

func TestExistingChainNodeEndpointsSkipsPackageAdditions(t *testing.T) {
	const (
		additionID  = "11111111-1111-1111-1111-111111111111"
		additionKey = "chain_node:v2_addition"
		existingID  = "22222222-2222-2222-2222-222222222222"
		existingKey = "chain_node:existing"
	)
	pkg := biz.Package{
		ChainNodeAdditions: []biz.ChainNodeAddition{{
			EntityID:  additionID,
			EntityKey: additionKey,
		}},
		Memberships: []biz.Membership{
			{ChainNodeEntityID: additionID, NodeKey: additionKey},
			{ChainNodeEntityID: existingID, NodeKey: existingKey},
		},
		GlobalRelations: []biz.GlobalChainNodeRelation{
			{
				FromChainNodeEntityID: additionID,
				FromNodeKey:           additionKey,
				ToChainNodeEntityID:   existingID,
				ToNodeKey:             existingKey,
			},
			{
				FromChainNodeEntityID: existingID,
				FromNodeKey:           existingKey,
				ToChainNodeEntityID:   additionID,
				ToNodeKey:             additionKey,
			},
		},
	}

	got, err := existingChainNodeEndpoints(pkg)
	if err != nil {
		t.Fatalf("existingChainNodeEndpoints() error = %v", err)
	}
	if len(got) != 1 || got[existingID] != existingKey {
		t.Fatalf("existingChainNodeEndpoints() = %#v, want only existing endpoint", got)
	}
	if _, exists := got[additionID]; exists {
		t.Fatalf("package addition %s must not be resolved before insertion", additionID)
	}
}

func TestExistingChainNodeEndpointsRejectsConflictingKeys(t *testing.T) {
	const existingID = "22222222-2222-2222-2222-222222222222"
	pkg := biz.Package{
		Memberships: []biz.Membership{{
			ChainNodeEntityID: existingID,
			NodeKey:           "chain_node:first",
		}},
		GlobalRelations: []biz.GlobalChainNodeRelation{{
			FromChainNodeEntityID: existingID,
			FromNodeKey:           "chain_node:second",
			ToChainNodeEntityID:   "33333333-3333-3333-3333-333333333333",
			ToNodeKey:             "chain_node:other",
		}},
	}

	_, err := existingChainNodeEndpoints(pkg)
	if err == nil || !strings.Contains(err.Error(), "conflicting keys") {
		t.Fatalf("existingChainNodeEndpoints() error = %v, want conflicting keys", err)
	}
}
