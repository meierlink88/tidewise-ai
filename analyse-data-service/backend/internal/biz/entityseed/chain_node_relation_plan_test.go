package entityseed

import (
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"
)

func TestClassifyChainNodeRelationWriteKeepsIdentityAndScalarRulesInBiz(t *testing.T) {
	wanted := model.ChainNodeRelation{
		ID: "relation-1", FromChainNodeEntityID: "node-a", ToChainNodeEntityID: "node-b",
		RelationType: model.ChainNodeRelationInputTo, Mechanism: "mechanism", EvidenceNote: "evidence",
		Provenance: "provenance", VerifiedAt: time.Date(2026, 7, 14, 6, 11, 6, 0, time.UTC),
		Status: model.StatusActive,
	}
	sameInstant := wanted
	sameInstant.VerifiedAt = wanted.VerifiedAt.In(time.FixedZone("database-session", 8*60*60))
	action, err := ClassifyChainNodeRelationWrite(wanted, ChainNodeRelationWriteSnapshot{
		ActiveEndpoints: 2,
		ExistingByID:    &sameInstant,
	})
	if err != nil || action != WriteUnchanged {
		t.Fatalf("same instant classification = %q, %v", action, err)
	}

	drifted := sameInstant
	drifted.EvidenceNote += " changed"
	action, err = ClassifyChainNodeRelationWrite(wanted, ChainNodeRelationWriteSnapshot{
		ActiveEndpoints: 2,
		ExistingByID:    &drifted,
	})
	if err != nil || action != WriteUpdated {
		t.Fatalf("scalar drift classification = %q, %v", action, err)
	}

	identityConflict := sameInstant
	identityConflict.ToChainNodeEntityID = "node-c"
	if _, err := ClassifyChainNodeRelationWrite(wanted, ChainNodeRelationWriteSnapshot{
		ActiveEndpoints: 2,
		ExistingByID:    &identityConflict,
	}); err == nil {
		t.Fatal("identity conflict was accepted")
	}
}

func TestValidateFrozenChainNodeRelationBaselineAcceptsOnlyFrozenStates(t *testing.T) {
	tests := []struct {
		name     string
		baseline FrozenChainNodeRelationBaseline
		wantErr  bool
	}{
		{
			name: "before write",
			baseline: FrozenChainNodeRelationBaseline{
				ExistingRelations:    100,
				SubcategoryRelations: 95,
				ComponentRelations:   1,
				InputRelations:       3,
				DependsRelations:     1,
			},
		},
		{
			name: "after write",
			baseline: FrozenChainNodeRelationBaseline{
				ExistingRelations:    212,
				SubcategoryRelations: 108,
				ComponentRelations:   3,
				InputRelations:       93,
				DependsRelations:     8,
			},
		},
		{
			name: "retired historical baseline",
			baseline: FrozenChainNodeRelationBaseline{
				ExistingRelations:    96,
				SubcategoryRelations: 95,
				ComponentRelations:   1,
			},
			wantErr: true,
		},
		{
			name: "before write type drift",
			baseline: FrozenChainNodeRelationBaseline{
				ExistingRelations:    100,
				SubcategoryRelations: 95,
				ComponentRelations:   1,
				InputRelations:       4,
			},
			wantErr: true,
		},
		{
			name: "after write type drift",
			baseline: FrozenChainNodeRelationBaseline{
				ExistingRelations:    212,
				SubcategoryRelations: 108,
				ComponentRelations:   3,
				InputRelations:       94,
				DependsRelations:     7,
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateFrozenChainNodeRelationBaseline(test.baseline)
			if (err != nil) != test.wantErr {
				t.Fatalf("error=%v wantErr=%v", err, test.wantErr)
			}
		})
	}
}

func TestValidateFrozenChainNodeRelationPlanProtectsAcceptedHundredAndFinalCounts(t *testing.T) {
	typeCounts := map[model.ChainNodeRelationType]int{
		model.ChainNodeRelationSubcategoryOf: 108,
		model.ChainNodeRelationComponentOf:   3,
		model.ChainNodeRelationInputTo:       93,
		model.ChainNodeRelationDependsOn:     8,
	}
	before := FrozenChainNodeRelationBaseline{ExistingRelations: 100, SubcategoryRelations: 95, ComponentRelations: 1, InputRelations: 3, DependsRelations: 1}
	after := FrozenChainNodeRelationBaseline{ExistingRelations: 212, SubcategoryRelations: 108, ComponentRelations: 3, InputRelations: 93, DependsRelations: 8}
	if err := ValidateFrozenChainNodeRelationPlan(before, ChainNodeRelationReport{Created: 112, Unchanged: 100, ByRelationType: typeCounts}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFrozenChainNodeRelationPlan(after, ChainNodeRelationReport{Unchanged: 212, ByRelationType: typeCounts}); err != nil {
		t.Fatal(err)
	}
	tests := []ChainNodeRelationReport{
		{Created: 111, Unchanged: 101, ByRelationType: typeCounts},
		{Created: 112, Updated: 1, Unchanged: 99, ByRelationType: typeCounts},
		{Created: 112, Unchanged: 100, ByRelationType: map[model.ChainNodeRelationType]int{
			model.ChainNodeRelationSubcategoryOf: 107,
			model.ChainNodeRelationComponentOf:   3,
			model.ChainNodeRelationInputTo:       94,
			model.ChainNodeRelationDependsOn:     8,
		}},
	}
	for _, report := range tests {
		if err := ValidateFrozenChainNodeRelationPlan(before, report); err == nil {
			t.Fatalf("plan drift accepted: %+v", report)
		}
	}
}

func TestValidateFrozenChainNodeRelationActionsRejectsBalancedAcceptedBaselineDrift(t *testing.T) {
	baseline := FrozenChainNodeRelationBaseline{ExistingRelations: 100, SubcategoryRelations: 95, ComponentRelations: 1, InputRelations: 3, DependsRelations: 1}
	actions := frozenChainNodeRelationActionsForTest(WriteUnchanged, WriteCreated)
	if err := ValidateFrozenChainNodeRelationActions(baseline, actions); err != nil {
		t.Fatal(err)
	}
	actions[0] = WriteCreated
	actions[100] = WriteUnchanged
	if err := ValidateFrozenChainNodeRelationActions(baseline, actions); err == nil {
		t.Fatal("accepted baseline drift passed because aggregate counts balanced")
	}
}

func TestValidateFrozenChainNodeRelationAggregateRejectsPostWriteDrift(t *testing.T) {
	valid := FrozenChainNodeRelationAggregate{
		Total: 212, Subcategory: 108, Component: 3, Input: 93, Depends: 8,
	}
	if err := ValidateFrozenChainNodeRelationAggregate(valid); err != nil {
		t.Fatal(err)
	}
	drifted := valid
	drifted.Orphans = 1
	if err := ValidateFrozenChainNodeRelationAggregate(drifted); err == nil {
		t.Fatal("orphaned post-write aggregate was accepted")
	}
}

func frozenChainNodeRelationActionsForTest(acceptedAction, additiveAction WriteAction) []WriteAction {
	actions := make([]WriteAction, 0, 212)
	for range 100 {
		actions = append(actions, acceptedAction)
	}
	for range 112 {
		actions = append(actions, additiveAction)
	}
	return actions
}
