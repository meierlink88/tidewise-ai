package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// This key is intentionally stable across AgentRun releases. Normal Event
// Semantic cycles take a shared lock; historical maintenance takes the
// exclusive form of the same session-level advisory lock.
const eventSemanticMaintenanceLockKey int64 = 781593201536236368

func (s *Store) WithEventSemanticProcessingPermit(
	ctx context.Context,
	operation func() error,
) error {
	return s.withEventSemanticMaintenanceLock(ctx, true, operation)
}

func (s *Store) WithHistoricalEventSemanticMaintenance(
	ctx context.Context,
	operation func() error,
) error {
	return s.withEventSemanticMaintenanceLock(ctx, false, operation)
}

func (s *Store) withEventSemanticMaintenanceLock(
	ctx context.Context,
	shared bool,
	operation func() error,
) (resultErr error) {
	if operation == nil {
		return errors.New("Event Semantic locked operation is required")
	}
	connection, err := s.database.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire Event Semantic maintenance lock connection: %w", err)
	}
	released := false
	defer func() {
		if !released {
			connection.Release()
		}
	}()

	lockQuery := `SELECT pg_advisory_lock($1)`
	unlockQuery := `SELECT pg_advisory_unlock($1)`
	if shared {
		lockQuery = `SELECT pg_advisory_lock_shared($1)`
		unlockQuery = `SELECT pg_advisory_unlock_shared($1)`
	}
	if _, err := connection.Exec(ctx, lockQuery, eventSemanticMaintenanceLockKey); err != nil {
		return fmt.Errorf("acquire Event Semantic maintenance lock: %w", err)
	}

	resultErr = operation()

	unlockContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var unlocked bool
	unlockErr := connection.QueryRow(
		unlockContext, unlockQuery, eventSemanticMaintenanceLockKey,
	).Scan(&unlocked)
	if unlockErr != nil || !unlocked {
		rawConnection := connection.Hijack()
		released = true
		closeErr := rawConnection.Close(unlockContext)
		if unlockErr == nil {
			unlockErr = errors.New("Event Semantic maintenance lock was not held")
		}
		return errors.Join(
			resultErr,
			fmt.Errorf("release Event Semantic maintenance lock: %w", unlockErr),
			closeErr,
		)
	}
	connection.Release()
	released = true
	return resultErr
}
