package industrygraphprojection

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type SnapshotReader interface {
	ReadIndustryGraphSnapshot(context.Context, string) (Projection, error)
}

type ProjectionStore interface {
	InspectIndustryGraph(context.Context, string) (ProjectionState, error)
	ReplaceIndustryGraph(context.Context, string, Projection) (ProjectionState, error)
}

type Service struct {
	source SnapshotReader
	store  ProjectionStore
}

func NewService(source SnapshotReader, store ProjectionStore) *Service {
	return &Service{source: source, store: store}
}

func (s *Service) Project(ctx context.Context, request ProjectRequest) (Result, error) {
	if s == nil || s.source == nil || s.store == nil {
		return Result{}, errors.New("Industry graph projection dependencies are required")
	}
	if strings.TrimSpace(request.Baseline.PackageSHA256) == "" {
		return Result{}, errors.New("Industry graph package SHA-256 is required")
	}
	if err := ValidateProjection(request.Baseline); err != nil {
		return Result{}, fmt.Errorf("validate approved Industry graph baseline: %w", err)
	}

	source, err := s.source.ReadIndustryGraphSnapshot(ctx, request.Baseline.PackageSHA256)
	if err != nil {
		return Result{}, fmt.Errorf("read Industry graph source snapshot: %w", err)
	}
	if err := ValidateProjection(source); err != nil {
		return Result{}, fmt.Errorf("validate PostgreSQL Industry graph snapshot: %w", err)
	}
	if !ProjectionsEqual(request.Baseline, source) {
		return Result{}, errors.New("PostgreSQL Industry graph snapshot differs from the approved projection baseline")
	}

	current, err := s.store.InspectIndustryGraph(ctx, Namespace)
	if err != nil {
		return Result{}, fmt.Errorf("inspect Industry graph projection: %w", err)
	}
	result := resultFor(source)
	result.DryRun = !request.Apply
	result.CurrentNeo4j = SummarizeProjection(current.Projection)
	result.FinalNeo4j = result.CurrentNeo4j
	result.CurrentIntegrityViolationCount = current.IntegrityViolationCount
	result.FinalIntegrityViolationCount = current.IntegrityViolationCount
	if stateMatches(current, source) {
		result.Unchanged = true
		return result, nil
	}
	if !request.Apply {
		return result, nil
	}

	projected, err := s.store.ReplaceIndustryGraph(ctx, Namespace, source)
	if err != nil {
		return Result{}, fmt.Errorf("replace Industry graph projection: %w", err)
	}
	if !stateMatches(projected, source) {
		return Result{}, errors.New("Neo4j Industry graph differs from the approved source after replacement")
	}
	result.FinalNeo4j = SummarizeProjection(projected.Projection)
	result.FinalIntegrityViolationCount = projected.IntegrityViolationCount
	result.Applied = true
	return result, nil
}

func stateMatches(state ProjectionState, expected Projection) bool {
	return state.ContractVersion == ContractVersion &&
		state.PackageSHA256 == expected.PackageSHA256 &&
		state.IntegrityViolationCount == 0 &&
		ProjectionsEqual(state.Projection, expected)
}

func resultFor(projection Projection) Result {
	return Result{
		Namespace:         Namespace,
		ContractVersion:   ContractVersion,
		PackageSHA256:     projection.PackageSHA256,
		NodeCount:         len(projection.Nodes),
		RelationshipCount: len(projection.Relationships),
		Source:            SummarizeProjection(projection),
	}
}
