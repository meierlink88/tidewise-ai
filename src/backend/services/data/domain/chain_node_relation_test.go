package domain

import (
	"strings"
	"testing"
	"time"
)

func TestChainNodeRelationValidation(t *testing.T) {
	valid := ChainNodeRelation{ID: "r", FromChainNodeEntityID: "a", ToChainNodeEntityID: "b", RelationType: ChainNodeRelationInputTo, Mechanism: "material input", EvidenceNote: "reviewed evidence", Provenance: "review", Status: StatusActive, VerifiedAt: time.Now()}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, value := range []ChainNodeRelation{
		func() ChainNodeRelation { v := valid; v.RelationType = "contains"; return v }(),
		func() ChainNodeRelation { v := valid; v.ToChainNodeEntityID = "a"; return v }(),
		func() ChainNodeRelation { v := valid; v.Mechanism = ""; return v }(),
		func() ChainNodeRelation { v := valid; v.ConditionNote = "  "; return v }(),
		func() ChainNodeRelation { v := valid; v.Provenance = ""; return v }(),
	} {
		if err := value.Validate(); err == nil {
			t.Fatalf("Validate(%+v) error = nil", value)
		}
	}
	duplicate := valid
	duplicate.ID = "r2"
	duplicate.RelationType = ChainNodeRelationDependsOn
	duplicate.Mechanism = " MATERIAL INPUT "
	if err := ValidateChainNodeRelationBatch([]ChainNodeRelation{valid, duplicate}); err == nil || !strings.Contains(err.Error(), "input_to") {
		t.Fatalf("batch error = %v", err)
	}
}

func TestChainNodePhysicalConstraintRequiresOneNewSubjectAndHardType(t *testing.T) {
	valid := ChainNodePhysicalConstraint{ID: "c", ChainNodeEntityID: "node", ConstraintType: ChainNodeConstraintProductionCapacity, Description: "扩产受设备与建设周期限制", EvidenceNote: "需要产能与扩建来源", Provenance: "review", VerifiedAt: time.Now(), Status: StatusActive}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, value := range []ChainNodePhysicalConstraint{
		func() ChainNodePhysicalConstraint { v := valid; v.ChainNodeRelationID = "relation"; return v }(),
		func() ChainNodePhysicalConstraint {
			v := valid
			v.ChainNodeEntityID = ""
			v.ConstraintType = "price"
			return v
		}(),
		func() ChainNodePhysicalConstraint { v := valid; v.Description = "政策支持"; return v }(),
	} {
		if err := value.Validate(); err == nil {
			t.Fatalf("Validate(%+v) error = nil", value)
		}
	}
}
