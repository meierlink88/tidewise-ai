package researchseed

import (
	"context"
	"fmt"
	"time"
)

type Report struct {
	AnalysisBatchID string    `json:"analysis_batch_id"`
	ThemeCount      int       `json:"theme_count"`
	ChainNodeCount  int       `json:"chain_node_count"`
	EventCount      int       `json:"event_count"`
	PublishedAt     time.Time `json:"published_at"`
}

type Store interface {
	Apply(context.Context, Manifest, time.Time) (Report, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Apply(ctx context.Context, manifest Manifest, publishedAt time.Time) (Report, error) {
	if err := manifest.Validate(); err != nil {
		return Report{}, fmt.Errorf("validate research theme manifest: %w", err)
	}
	if s == nil || s.store == nil {
		return Report{}, fmt.Errorf("research theme seed store is required")
	}
	if publishedAt.IsZero() {
		publishedAt = time.Now().UTC()
	} else {
		publishedAt = publishedAt.UTC()
	}
	report, err := s.store.Apply(ctx, manifest, publishedAt)
	if err != nil {
		return Report{}, fmt.Errorf("apply research theme manifest: %w", err)
	}
	return report, nil
}
