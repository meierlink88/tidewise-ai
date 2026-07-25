package entityseed

import (
	"fmt"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"
)

type ChainNodeRelationWriteSnapshot struct {
	ActiveEndpoints int
	ExistingByID    *model.ChainNodeRelation
	ExistingByTuple *model.ChainNodeRelation
}

func ClassifyChainNodeRelationWrite(wanted model.ChainNodeRelation, snapshot ChainNodeRelationWriteSnapshot) (WriteAction, error) {
	if snapshot.ActiveEndpoints != 2 {
		return "", fmt.Errorf("chain node relation %q requires two active profiled chain_node endpoints", wanted.ID)
	}
	if snapshot.ExistingByID != nil {
		existing := *snapshot.ExistingByID
		if existing.FromChainNodeEntityID != wanted.FromChainNodeEntityID ||
			existing.ToChainNodeEntityID != wanted.ToChainNodeEntityID ||
			existing.RelationType != wanted.RelationType {
			return "", fmt.Errorf("chain node relation %q identity conflict", wanted.ID)
		}
		if ChainNodeRelationsEqual(existing, wanted) {
			return WriteUnchanged, nil
		}
		return WriteUpdated, nil
	}
	if snapshot.ExistingByTuple != nil && snapshot.ExistingByTuple.ID != wanted.ID {
		return "", fmt.Errorf("chain node relation %q tuple conflict with %q", wanted.ID, snapshot.ExistingByTuple.ID)
	}
	return WriteCreated, nil
}

func ChainNodeRelationsEqual(left, right model.ChainNodeRelation) bool {
	return left.ID == right.ID &&
		left.FromChainNodeEntityID == right.FromChainNodeEntityID &&
		left.ToChainNodeEntityID == right.ToChainNodeEntityID &&
		left.RelationType == right.RelationType &&
		left.Mechanism == right.Mechanism &&
		left.ConditionNote == right.ConditionNote &&
		left.EvidenceNote == right.EvidenceNote &&
		left.Provenance == right.Provenance &&
		left.Status == right.Status &&
		equalOptionalInstant(left.VerifiedAt, right.VerifiedAt)
}

func equalOptionalInstant(left, right time.Time) bool {
	if left.IsZero() || right.IsZero() {
		return left.IsZero() && right.IsZero()
	}
	return left.Equal(right)
}

func ValidateFrozenChainNodeRelationBaseline(baseline FrozenChainNodeRelationBaseline) error {
	beforeWrite := baseline.ExistingRelations == 100 &&
		baseline.SubcategoryRelations == 95 &&
		baseline.ComponentRelations == 1 &&
		baseline.InputRelations == 3 &&
		baseline.DependsRelations == 1
	afterWrite := baseline.ExistingRelations == 212 &&
		baseline.SubcategoryRelations == 108 &&
		baseline.ComponentRelations == 3 &&
		baseline.InputRelations == 93 &&
		baseline.DependsRelations == 8
	if !beforeWrite && !afterWrite {
		return fmt.Errorf(
			"frozen relation dry-run requires 100=95/1/3/1 or 212=108/3/93/8 relations, got %d=%d/%d/%d/%d",
			baseline.ExistingRelations,
			baseline.SubcategoryRelations,
			baseline.ComponentRelations,
			baseline.InputRelations,
			baseline.DependsRelations,
		)
	}
	return nil
}

func ValidateFrozenChainNodeRelationPlan(baseline FrozenChainNodeRelationBaseline, report ChainNodeRelationReport) error {
	if err := ValidateFrozenChainNodeRelationBaseline(baseline); err != nil {
		return err
	}
	wantTypes := map[model.ChainNodeRelationType]int{
		model.ChainNodeRelationSubcategoryOf: 108,
		model.ChainNodeRelationComponentOf:   3,
		model.ChainNodeRelationInputTo:       93,
		model.ChainNodeRelationDependsOn:     8,
	}
	if len(report.ByRelationType) != len(wantTypes) {
		return fmt.Errorf("frozen relation plan type set drifted: %+v", report.ByRelationType)
	}
	for relationType, count := range wantTypes {
		if report.ByRelationType[relationType] != count {
			return fmt.Errorf("frozen relation plan requires types 108/3/93/8, got %+v", report.ByRelationType)
		}
	}
	if baseline.ExistingRelations == 100 && (report.Created != 112 || report.Updated != 0 || report.Unchanged != 100) {
		return fmt.Errorf("frozen relation pre-write plan requires 112 created, 0 updated and 100 unchanged, got %+v", report)
	}
	if baseline.ExistingRelations == 212 && (report.Created != 0 || report.Updated != 0 || report.Unchanged != 212) {
		return fmt.Errorf("frozen relation post-write plan requires 0 created, 0 updated and 212 unchanged, got %+v", report)
	}
	if report.Created+report.Updated+report.Unchanged != 212 {
		return fmt.Errorf("frozen relation plan requires 212 total rows, got %+v", report)
	}
	return nil
}

func ValidateFrozenChainNodeRelationActions(baseline FrozenChainNodeRelationBaseline, actions []WriteAction) error {
	if len(actions) != 212 {
		return fmt.Errorf("frozen relation plan requires 212 ordered rows, got %d", len(actions))
	}
	for index, action := range actions {
		want := WriteUnchanged
		if index >= 100 && baseline.ExistingRelations == 100 {
			want = WriteCreated
		}
		if action != want {
			return fmt.Errorf("frozen relation plan row %d requires %s, got %s", index, want, action)
		}
	}
	return nil
}

func ValidateFrozenChainNodeRelationAggregate(aggregate FrozenChainNodeRelationAggregate) error {
	if aggregate.Total != 212 ||
		aggregate.Subcategory != 108 ||
		aggregate.Component != 3 ||
		aggregate.Input != 93 ||
		aggregate.Depends != 8 ||
		aggregate.Incomplete != 0 ||
		aggregate.SelfLoops != 0 ||
		aggregate.Duplicates != 0 ||
		aggregate.Orphans != 0 {
		return fmt.Errorf(
			"frozen relation post-write aggregate mismatch: total=%d types=%d/%d/%d/%d incomplete=%d self=%d duplicate=%d orphan=%d",
			aggregate.Total,
			aggregate.Subcategory,
			aggregate.Component,
			aggregate.Input,
			aggregate.Depends,
			aggregate.Incomplete,
			aggregate.SelfLoops,
			aggregate.Duplicates,
			aggregate.Orphans,
		)
	}
	return nil
}
