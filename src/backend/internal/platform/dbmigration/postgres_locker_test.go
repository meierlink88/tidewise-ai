package dbmigration

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
)

func TestPostgresAdvisoryLockerPinsAcquireAndReleaseToOneConnection(t *testing.T) {
	connection := &fakeAdvisoryConnection{unlockResult: true}
	locker := newPostgresAdvisoryLockerWithConnector("migration", func(context.Context) (advisoryConnection, error) {
		return connection, nil
	})

	if err := locker.Lock(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := locker.Unlock(context.Background()); err != nil {
		t.Fatal(err)
	}
	if connection.acquireCalls != 1 || connection.unlockCalls != 1 || connection.closeCalls != 1 || connection.discardCalls != 0 {
		t.Fatalf("connection lifecycle = %+v", connection)
	}
}

func TestPostgresAdvisoryLockerRejectsInvalidOwnershipTransitions(t *testing.T) {
	connection := &fakeAdvisoryConnection{unlockResult: true}
	connectCalls := 0
	locker := newPostgresAdvisoryLockerWithConnector("migration", func(context.Context) (advisoryConnection, error) {
		connectCalls++
		return connection, nil
	})

	if err := locker.Unlock(context.Background()); err == nil || !strings.Contains(err.Error(), "not held") {
		t.Fatalf("Unlock() error = %v", err)
	}
	if err := locker.Lock(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := locker.Lock(context.Background()); err == nil || !strings.Contains(err.Error(), "already held") {
		t.Fatalf("second Lock() error = %v", err)
	}
	if connectCalls != 1 || connection.acquireCalls != 1 {
		t.Fatalf("connect calls = %d, connection = %+v", connectCalls, connection)
	}
	if err := locker.Unlock(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresAdvisoryLockerDiscardsConnectionWhenAcquireMayHaveFailedAfterLock(t *testing.T) {
	acquireErr := errors.New("network failed after acquire")
	connection := &fakeAdvisoryConnection{acquireErr: acquireErr}
	locker := newPostgresAdvisoryLockerWithConnector("migration", func(context.Context) (advisoryConnection, error) {
		return connection, nil
	})

	if err := locker.Lock(context.Background()); !errors.Is(err, acquireErr) {
		t.Fatalf("Lock() error = %v", err)
	}
	if connection.discardCalls != 1 || connection.closeCalls != 1 {
		t.Fatalf("failed acquire cleanup = %+v", connection)
	}
	if err := locker.Unlock(context.Background()); err == nil || !strings.Contains(err.Error(), "not held") {
		t.Fatalf("Unlock() error = %v", err)
	}
}

func TestPostgresAdvisoryLockerDoesNotIgnoreUnlockFailure(t *testing.T) {
	unlockErr := errors.New("unlock query failed")
	connection := &fakeAdvisoryConnection{unlockErr: unlockErr}
	locker := newPostgresAdvisoryLockerWithConnector("migration", func(context.Context) (advisoryConnection, error) {
		return connection, nil
	})

	if err := locker.Lock(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := locker.Unlock(context.Background()); !errors.Is(err, unlockErr) {
		t.Fatalf("Unlock() error = %v", err)
	}
	if connection.discardCalls != 1 || connection.closeCalls != 1 {
		t.Fatalf("failed unlock cleanup = %+v", connection)
	}
}

func TestPostgresAdvisoryLockerCloseDiscardsHeldSession(t *testing.T) {
	connection := &fakeAdvisoryConnection{}
	locker := newPostgresAdvisoryLockerWithConnector("migration", func(context.Context) (advisoryConnection, error) {
		return connection, nil
	})

	if err := locker.Lock(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := locker.Close(); err != nil {
		t.Fatal(err)
	}
	if connection.discardCalls != 1 || connection.closeCalls != 1 {
		t.Fatalf("close cleanup = %+v", connection)
	}
	if err := locker.Unlock(context.Background()); err == nil || !strings.Contains(err.Error(), "not held") {
		t.Fatalf("Unlock() error = %v", err)
	}
}

func TestPostgresAdvisoryLockerReportsConnectionCloseFailure(t *testing.T) {
	closeErr := errors.New("close failed")
	connection := &fakeAdvisoryConnection{unlockResult: true, closeErr: closeErr}
	locker := newPostgresAdvisoryLockerWithConnector("migration", func(context.Context) (advisoryConnection, error) {
		return connection, nil
	})

	if err := locker.Lock(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := locker.Unlock(context.Background()); !errors.Is(err, closeErr) {
		t.Fatalf("Unlock() error = %v", err)
	}
}

func TestPostgresAdvisoryLockerPropagatesConnectionContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	locker := newPostgresAdvisoryLockerWithConnector("migration", func(ctx context.Context) (advisoryConnection, error) {
		return nil, ctx.Err()
	})

	if err := locker.Lock(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Lock() error = %v", err)
	}
}

type fakeAdvisoryConnection struct {
	acquireCalls int
	unlockCalls  int
	discardCalls int
	closeCalls   int
	acquireErr   error
	unlockErr    error
	unlockResult bool
	closeErr     error
}

func (c *fakeAdvisoryConnection) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	c.acquireCalls++
	return nil, c.acquireErr
}

func (c *fakeAdvisoryConnection) QueryRowContext(context.Context, string, ...any) advisoryRow {
	c.unlockCalls++
	return fakeAdvisoryRow{value: c.unlockResult, err: c.unlockErr}
}

func (c *fakeAdvisoryConnection) Discard() error {
	c.discardCalls++
	return nil
}

func (c *fakeAdvisoryConnection) Close() error {
	c.closeCalls++
	return c.closeErr
}

type fakeAdvisoryRow struct {
	value bool
	err   error
}

func (r fakeAdvisoryRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*(dest[0].(*bool)) = r.value
	return nil
}
