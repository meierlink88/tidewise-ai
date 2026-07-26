package artifacts

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector"
	collectorusecase "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector/usecase"
	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

// Store is the filesystem adapter for Collector Artifacts.
type Store struct {
	Root             string
	Publications     PublicationRepository
	Now              func() time.Time
	AfterPublication func()
}

func (s Store) Ready(_ context.Context) error {
	if s.Root == "" {
		return errors.New("Artifact root is required")
	}
	if err := os.MkdirAll(s.Root, 0o755); err != nil {
		return errors.New("Artifact root is unavailable")
	}
	probe, err := os.CreateTemp(s.Root, ".ready-*")
	if err != nil {
		return errors.New("Artifact root is unavailable")
	}
	name := probe.Name()
	if err := probe.Close(); err != nil {
		_ = os.Remove(name)
		return errors.New("Artifact root is unavailable")
	}
	if err := os.Remove(name); err != nil {
		return errors.New("Artifact root is unavailable")
	}
	return nil
}

func (s Store) Materializer(nearDuplicateRadius int) collector.Materializer {
	return File{
		Root:                s.Root,
		NearDuplicateRadius: nearDuplicateRadius,
		Publications:        s.Publications,
		Now:                 s.Now,
		AfterPublication:    s.AfterPublication,
	}
}

func (s Store) ReconcilePreparedPublications(ctx context.Context) error {
	if err := ReconcilePreparedPublications(ctx, s.Root, s.Publications); err != nil {
		return err
	}
	if s.AfterPublication != nil {
		s.AfterPublication()
	}
	return nil
}

func (s Store) WriteTerminalAudit(execution agentrun.Execution) (map[string]string, error) {
	return WriteTerminalAudit(s.Root, execution)
}

// ErrPublicationPending remains available to existing adapter-level callers
// while sharing one Biz-owned failure classification with the Collector Use Case.
var ErrPublicationPending = collectorusecase.ErrPublicationPending
