package entityseed

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/model"
)

type ChainNodeRelationSnapshotRepository interface {
	IsActiveChainNode(context.Context, string) (bool, error)
	FindChainNodeRelationByID(context.Context, string) (model.ChainNodeRelation, bool, error)
	FindChainNodeRelationByTuple(context.Context, model.ChainNodeRelation) (model.ChainNodeRelation, bool, error)
}

func LoadChainNodeRelationManifest(path string) (ChainNodeRelationManifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return ChainNodeRelationManifest{}, err
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	decoder.DisallowUnknownFields()
	var manifest ChainNodeRelationManifest
	if err := decoder.Decode(&manifest); err != nil {
		return manifest, fmt.Errorf("decode chain node relation manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return manifest, fmt.Errorf("chain node relation manifest must contain a single JSON document")
		}
		return manifest, fmt.Errorf("chain node relation manifest must contain a single JSON document: %w", err)
	}
	if err := model.ValidateChainNodeRelationBatch(manifest.Relations); err != nil {
		return manifest, err
	}
	for _, constraint := range manifest.PhysicalConstraints {
		if err := constraint.Validate(); err != nil {
			return manifest, err
		}
	}
	return manifest, nil
}

func DryRunChainNodeRelations(ctx context.Context, repo ChainNodeRelationSnapshotRepository, relations []model.ChainNodeRelation) (ChainNodeRelationReport, error) {
	report := ChainNodeRelationReport{ByRelationType: map[model.ChainNodeRelationType]int{}}
	if err := model.ValidateChainNodeRelationBatch(relations); err != nil {
		return report, err
	}
	for _, relation := range relations {
		for _, endpoint := range []string{relation.FromChainNodeEntityID, relation.ToChainNodeEntityID} {
			active, err := repo.IsActiveChainNode(ctx, endpoint)
			if err != nil {
				return report, err
			}
			if !active {
				return report, fmt.Errorf("relation %q requires active chain_node endpoint %q", relation.ID, endpoint)
			}
		}
		existing, found, err := repo.FindChainNodeRelationByID(ctx, relation.ID)
		if err != nil {
			return report, err
		}
		if !found {
			if tuple, tupleFound, err := repo.FindChainNodeRelationByTuple(ctx, relation); err != nil {
				return report, err
			} else if tupleFound && tuple.ID != relation.ID {
				return report, fmt.Errorf("chain node relation %q tuple conflict with %q", relation.ID, tuple.ID)
			}
			report.Created++
		} else if existing.FromChainNodeEntityID != relation.FromChainNodeEntityID || existing.ToChainNodeEntityID != relation.ToChainNodeEntityID || existing.RelationType != relation.RelationType {
			return report, fmt.Errorf("chain node relation %q identity conflict", relation.ID)
		} else if reflect.DeepEqual(existing, relation) {
			report.Unchanged++
		} else {
			report.Updated++
		}
		report.ByRelationType[relation.RelationType]++
	}
	return report, nil
}

func ValidateChainNodeRelationDryRunManifest(manifest ChainNodeRelationManifest) error {
	if len(manifest.PhysicalConstraints) != 0 {
		return fmt.Errorf("relation dry-run rejects physical_constraints until its repository and dry-run contract are implemented")
	}
	return nil
}
