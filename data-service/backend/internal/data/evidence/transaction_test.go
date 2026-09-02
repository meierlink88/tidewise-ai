package evidence

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	evidencebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/evidence"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

func TestEvidencePublicationTransactionRollsBackOnPanic(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	mock.ExpectRollback()

	deferred := false
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	func() {
		defer func() {
			if recovered := recover(); recovered != "test panic" {
				t.Fatalf("recovered panic = %#v", recovered)
			}
			deferred = true
		}()
		_ = store.InTransaction(context.Background(), func(evidencebiz.Transaction) error {
			panic("test panic")
		})
	}()
	if !deferred {
		t.Fatal("transaction panic did not reach caller")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEvidenceTransactionRollsBackWhenExecutionBudgetExpires(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	mock.ExpectRollback()
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = store.InTransaction(ctx, func(evidencebiz.Transaction) error {
		return context.DeadlineExceeded
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("InTransaction() error = %v, want deadline exceeded", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresEvidencePublicationTransactions(t *testing.T) {
	db := openEvidencePublicationTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	publication, err := evidencebiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	assertConcurrentRawEvidenceConvergence(t, publication, db)
	assertConcurrentRawEvidenceConflict(t, publication, db)
	assertConcurrentEvidenceConvergenceAndConflict(t, publication, db)
	assertEvidenceTransactionRollback(t, publication, store, db)
	assertEvidenceDeadlineCancelsQueryAndRollsBack(t, store, db)
}

func assertConcurrentRawEvidenceConflict(t *testing.T, publication *evidencebiz.UseCase, db *sql.DB) {
	t.Helper()
	left := postgresEvidenceRaw("RAWb47847ee-e053-5012-b313-1dc77c9c4f15")
	rawID := postgresRawEvidenceID(left)
	right := left
	right.RawText = "A different concurrent article."
	start := make(chan struct{})
	errorsChannel := make(chan error, 2)
	for _, input := range []evidencebiz.RawEvidence{left, right} {
		go func(raw evidencebiz.RawEvidence) {
			<-start
			_, err := publication.PublishRawEvidence(context.Background(), raw)
			errorsChannel <- err
		}(input)
	}
	close(start)
	succeeded, conflicted := 0, 0
	for count := 0; count < 2; count++ {
		err := <-errorsChannel
		if err == nil {
			succeeded++
			continue
		}
		var conflict *evidencebiz.ConflictError
		if errors.As(err, &conflict) {
			conflicted++
			continue
		}
		t.Fatalf("concurrent Raw drift error = %v", err)
	}
	var rows int
	if err := db.QueryRow(`SELECT count(*) FROM raw_evidences WHERE id = $1`, rawID).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if succeeded != 1 || conflicted != 1 || rows != 1 {
		t.Fatalf("Raw drift convergence succeeded=%d conflicted=%d rows=%d", succeeded, conflicted, rows)
	}
}

func assertConcurrentEvidenceConvergenceAndConflict(t *testing.T, publication *evidencebiz.UseCase, db *sql.DB) {
	t.Helper()
	for _, test := range []struct {
		rawID         string
		drift         bool
		wantSucceeded int
		wantConflicts int
	}{
		{rawID: "RAW31761f37-d5ea-5df4-849e-a2e0871cca83", wantSucceeded: 2},
		{rawID: "RAW18c5ad2d-579d-5b0d-9c38-790dfd021100", drift: true, wantSucceeded: 1, wantConflicts: 1},
	} {
		raw := postgresEvidenceRaw(test.rawID)
		rawResult, err := publication.PublishRawEvidence(context.Background(), raw)
		if err != nil {
			t.Fatal(err)
		}
		rawID := rawResult.ID
		left := []evidencebiz.Evidence{postgresEvidence(0)}
		right := []evidencebiz.Evidence{postgresEvidence(0)}
		if test.drift {
			right[0].Semantic.Action = "Concurrent semantic drift."
		}
		start := make(chan struct{})
		type outcome struct {
			result evidencebiz.EvidenceResult
			err    error
		}
		outcomes := make(chan outcome, 2)
		for _, input := range [][]evidencebiz.Evidence{left, right} {
			go func(items []evidencebiz.Evidence) {
				<-start
				result, err := publication.PublishEvidence(context.Background(), rawID, items)
				outcomes <- outcome{result: result, err: err}
			}(input)
		}
		close(start)
		succeeded, conflicted := 0, 0
		var expectedEvidenceID string
		for count := 0; count < 2; count++ {
			completed := <-outcomes
			if completed.err == nil {
				if len(completed.result.IDs) != 1 {
					t.Fatalf("concurrent Evidence result = %#v", completed.result)
				}
				if expectedEvidenceID == "" {
					expectedEvidenceID = completed.result.IDs[0]
				} else if completed.result.IDs[0] != expectedEvidenceID {
					t.Fatalf("concurrent Evidence ID = %q, want %q", completed.result.IDs[0], expectedEvidenceID)
				}
				succeeded++
				continue
			}
			var conflict *evidencebiz.ConflictError
			if errors.As(completed.err, &conflict) {
				conflicted++
				continue
			}
			t.Fatalf("concurrent Evidence error = %v", completed.err)
		}
		var rows int
		if err := db.QueryRow(`SELECT count(*) FROM evidences WHERE raw_evidence_id = $1`, rawID).Scan(&rows); err != nil {
			t.Fatal(err)
		}
		if succeeded != test.wantSucceeded || conflicted != test.wantConflicts || rows != 1 {
			t.Fatalf("Evidence convergence succeeded=%d conflicts=%d rows=%d", succeeded, conflicted, rows)
		}
	}
}

func assertConcurrentRawEvidenceConvergence(t *testing.T, publication *evidencebiz.UseCase, db *sql.DB) {
	t.Helper()
	raw := postgresEvidenceRaw("RAW6c7e6640-423e-5a22-834f-631e5a61a5b2")
	rawID := postgresRawEvidenceID(raw)
	start := make(chan struct{})
	results := make(chan evidencebiz.RawEvidenceResult, 2)
	errorsChannel := make(chan error, 2)
	var workers sync.WaitGroup
	for index := 0; index < 2; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			result, err := publication.PublishRawEvidence(context.Background(), raw)
			results <- result
			errorsChannel <- err
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	close(errorsChannel)
	for err := range errorsChannel {
		if err != nil {
			t.Fatalf("concurrent publication: %v", err)
		}
	}
	for result := range results {
		if result.ID != rawID {
			t.Fatalf("concurrent publication result = %#v", result)
		}
	}
	var rows int
	if err := db.QueryRow(`SELECT count(*) FROM raw_evidences WHERE id = $1`, rawID).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 1 {
		t.Fatalf("convergence rows=%d", rows)
	}
}

func assertEvidenceTransactionRollback(t *testing.T, publication *evidencebiz.UseCase, store evidencebiz.Store, db *sql.DB) {
	t.Helper()
	forcedFailure := errors.New("force transaction rollback")
	raw := postgresEvidenceRaw("RAWb24ba990-71e7-522d-b192-3e5c82790aa6")
	raw.ID = postgresRawEvidenceID(raw)
	err := store.InTransaction(context.Background(), func(tx evidencebiz.Transaction) error {
		if err := tx.InsertRawEvidence(context.Background(), evidencebiz.StoredRawEvidence{
			RawEvidence: raw, ContentHash: "unused-generated-column-value",
		}); err != nil {
			return err
		}
		return forcedFailure
	})
	if !errors.Is(err, forcedFailure) {
		t.Fatalf("Raw Evidence forced rollback error = %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM raw_evidences WHERE id = $1`, raw.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("rolled-back Raw Evidence count = %d", count)
	}

	evidenceRaw := postgresEvidenceRaw("RAWe87e4f45-47e8-54f8-9dea-7b54348f0963")
	evidenceRawResult, err := publication.PublishRawEvidence(context.Background(), evidenceRaw)
	if err != nil {
		t.Fatal(err)
	}
	evidenceRaw.ID = evidenceRawResult.ID
	evidence := postgresEvidence(0)
	evidence.ID, err = coreid.New(coreid.Evidence)
	if err != nil {
		t.Fatal(err)
	}
	err = store.InTransaction(context.Background(), func(tx evidencebiz.Transaction) error {
		if err := tx.InsertEvidence(context.Background(), evidencebiz.StoredEvidence{
			Evidence: evidence, RawEvidenceID: evidenceRaw.ID, IsSplit: false,
		}); err != nil {
			return err
		}
		return forcedFailure
	})
	if !errors.Is(err, forcedFailure) {
		t.Fatalf("Evidence forced rollback error = %v", err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM evidences WHERE raw_evidence_id = $1`, evidenceRaw.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("rolled-back Evidence count = %d", count)
	}
}

func assertEvidenceDeadlineCancelsQueryAndRollsBack(t *testing.T, store Store, db *sql.DB) {
	t.Helper()
	lockKey := "raw-evidence:deadline-rollback"
	lockHolder, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := lockHolder.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			t.Errorf("release Evidence deadline test lock: %v", err)
		}
	}()
	if _, err := lockHolder.Exec(`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey); err != nil {
		t.Fatal(err)
	}

	raw := postgresEvidenceRaw("RAW939071e3-a016-5296-8008-377876cbb972")
	raw.ID = postgresRawEvidenceID(raw)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err = store.InTransaction(ctx, func(tx evidencebiz.Transaction) error {
		if err := tx.InsertRawEvidence(ctx, evidencebiz.StoredRawEvidence{RawEvidence: raw}); err != nil {
			return err
		}
		return tx.LockIdentities(ctx, []string{lockKey})
	})
	if err == nil || !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("deadline transaction error=%v context=%v", err, ctx.Err())
	}
	var rawCount int
	if err := db.QueryRow(`SELECT count(*) FROM raw_evidences WHERE id = $1`, raw.ID).Scan(&rawCount); err != nil {
		t.Fatal(err)
	}
	if rawCount != 0 {
		t.Fatalf("deadline rollback left raw=%d", rawCount)
	}
}
