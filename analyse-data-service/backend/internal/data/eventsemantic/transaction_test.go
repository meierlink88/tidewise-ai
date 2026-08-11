package eventsemantic

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	eventbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventsemantic"
	eventsemanticfixture "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/testsupport/eventsemantic"
)

func TestPostgresEventSemanticReplayAndReviewSerializeOnSubmission(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	eventsemanticfixture.SeedScenario(t, db, true)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := eventbiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	const executionID = "semantic-concurrent-replay-review"
	lease, err := useCase.CreateContextLease(context.Background(), eventbiz.ContextLeaseRequest{
		EventID: eventsemanticfixture.EventID, AgentExecutionID: executionID,
		WorkerID: "concurrency-fixture", Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	input := eventsemanticfixture.Submission(lease.ID, executionID, "")
	submission, err := useCase.CreateSubmission(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	replayErr := make(chan error, 1)
	reviewErr := make(chan error, 1)
	go func() {
		<-start
		replayed, err := useCase.CreateSubmission(context.Background(), input)
		if err == nil && !replayed.Replayed {
			err = errors.New("concurrent replay did not replay Submission")
		}
		replayErr <- err
	}()
	go func() {
		<-start
		_, err := useCase.SubmitReview(context.Background(), eventbiz.ReviewSubmission{
			SubmissionID: submission.SubmissionID, ReviewerExecutionKey: executionID + ":reviewer",
			PromptHash: strings.Repeat("b", 64), Model: "fixture-reviewer",
			Items: eventsemanticfixture.ReviewItems(eventbiz.ReviewDecisionPass),
		})
		reviewErr <- err
	}()
	close(start)
	if err := <-replayErr; err != nil {
		t.Fatalf("concurrent replay: %v", err)
	}
	if err := <-reviewErr; err != nil {
		t.Fatalf("concurrent review: %v", err)
	}
}

func TestPostgresEventSemanticExpiredLeaseTransitionPrecedesReplacement(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	eventsemanticfixture.SeedScenario(t, db, true)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := eventbiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	expired, err := useCase.CreateContextLease(context.Background(), eventbiz.ContextLeaseRequest{
		EventID: eventsemanticfixture.EventID, AgentExecutionID: "semantic-expired-lease",
		WorkerID: "expiry-fixture", Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE event_semantic_context_leases SET lease_expires_at = now() - interval '1 minute' WHERE id = $1`, expired.ID); err != nil {
		t.Fatal(err)
	}
	replacement, err := useCase.CreateContextLease(context.Background(), eventbiz.ContextLeaseRequest{
		EventID: eventsemanticfixture.EventID, AgentExecutionID: "semantic-replacement-lease",
		WorkerID: "expiry-fixture", Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	var expiredStatus, replacementStatus string
	if err := db.QueryRow(`SELECT status FROM event_semantic_context_leases WHERE id = $1`, expired.ID).Scan(&expiredStatus); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT status FROM event_semantic_context_leases WHERE id = $1`, replacement.ID).Scan(&replacementStatus); err != nil {
		t.Fatal(err)
	}
	if expiredStatus != "expired" || replacementStatus != "active" {
		t.Fatalf("lease statuses = expired:%q replacement:%q", expiredStatus, replacementStatus)
	}
}

func TestPostgresEventSemanticSubmissionRejectsExpiredLeaseInBiz(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	eventsemanticfixture.SeedScenario(t, db, true)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := eventbiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	lease, err := useCase.CreateContextLease(context.Background(), eventbiz.ContextLeaseRequest{
		EventID: eventsemanticfixture.EventID, AgentExecutionID: "semantic-expired-submission-lease",
		WorkerID: "expiry-fixture", Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE event_semantic_context_leases SET lease_expires_at = now() - interval '1 minute' WHERE id = $1`, lease.ID); err != nil {
		t.Fatal(err)
	}
	_, err = useCase.CreateSubmission(context.Background(), eventsemanticfixture.Submission(
		lease.ID, "semantic-expired-submission-lease", "",
	))
	var notFound *eventbiz.NotFoundError
	if !errors.As(err, &notFound) || notFound.Resource != "Event Semantic Context Lease" {
		t.Fatalf("expired lease error = %v", err)
	}
}

func TestPostgresEventSemanticReplayWaitsForSubmissionLock(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	eventsemanticfixture.SeedScenario(t, db, true)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := eventbiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	const executionID = "semantic-locked-replay"
	lease, err := useCase.CreateContextLease(context.Background(), eventbiz.ContextLeaseRequest{
		EventID: eventsemanticfixture.EventID, AgentExecutionID: executionID,
		WorkerID: "lock-fixture", Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	input := eventsemanticfixture.Submission(lease.ID, executionID, "")
	created, err := useCase.CreateSubmission(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	lockHolder, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lockHolder.Exec(`SELECT id FROM event_semantic_submissions WHERE id = $1 FOR UPDATE`, created.SubmissionID); err != nil {
		_ = lockHolder.Rollback()
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	_, err = useCase.CreateSubmission(ctx, input)
	if rollbackErr := lockHolder.Rollback(); rollbackErr != nil {
		t.Fatal(rollbackErr)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("locked replay error = %T %v", err, err)
	}
	replayed, err := useCase.CreateSubmission(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Replayed {
		t.Fatalf("replay result = %#v", replayed)
	}
}

func TestPostgresEventSemanticExhaustedRetryRemainsValidTerminalState(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	eventsemanticfixture.SeedScenario(t, db, true)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := eventbiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	const executionID = "semantic-exhausted-retry"
	lease, err := useCase.CreateContextLease(context.Background(), eventbiz.ContextLeaseRequest{
		EventID: eventsemanticfixture.EventID, AgentExecutionID: executionID,
		WorkerID: "retry-fixture", Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	submission, err := useCase.CreateSubmission(context.Background(), eventsemanticfixture.Submission(
		lease.ID, executionID, "",
	))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE event_semantic_acceptance_policies SET retry_budget = 0`); err != nil {
		t.Fatal(err)
	}
	quarantined, err := useCase.SubmitReview(context.Background(), eventbiz.ReviewSubmission{
		SubmissionID: submission.SubmissionID, ReviewerExecutionKey: executionID + ":reviewer",
		PromptHash: strings.Repeat("b", 64), Model: "fixture-reviewer",
		Items: eventsemanticfixture.ReviewItems(eventbiz.ReviewDecisionIndeterminate),
	})
	if err != nil {
		t.Fatal(err)
	}
	if quarantined.Status != eventbiz.StatusQuarantined {
		t.Fatalf("status = %q, want quarantined", quarantined.Status)
	}
	_, err = useCase.SubmitReview(context.Background(), eventbiz.ReviewSubmission{
		SubmissionID: submission.SubmissionID, ReviewerExecutionKey: executionID + ":adjudicator",
		PromptHash: strings.Repeat("b", 64), Model: "fixture-reviewer",
		Items: eventsemanticfixture.ReviewItems(eventbiz.ReviewDecisionPass),
	})
	var conflict *eventbiz.ConflictError
	if !errors.As(err, &conflict) || conflict.Reason != "Event Semantic Submission is not reviewable" {
		t.Fatalf("terminal review error = %v", err)
	}
}

func TestPostgresEventSemanticTransactionCancellationRollsBack(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	eventsemanticfixture.SeedScenario(t, db, true)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := eventbiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	lockHolder, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lockHolder.Exec(`SELECT id FROM events WHERE id = $1 FOR UPDATE`, eventsemanticfixture.EventID); err != nil {
		_ = lockHolder.Rollback()
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	_, err = useCase.CreateContextLease(ctx, eventbiz.ContextLeaseRequest{
		EventID: eventsemanticfixture.EventID, AgentExecutionID: "semantic-cancellation",
		WorkerID: "cancellation-fixture", Lease: 15 * time.Minute,
	})
	if rollbackErr := lockHolder.Rollback(); rollbackErr != nil {
		t.Fatal(rollbackErr)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cancellation error = %T %v", err, err)
	}
	var count int
	if err := db.QueryRow(`
		SELECT count(*) FROM event_semantic_context_leases WHERE agent_execution_id = 'semantic-cancellation'
	`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("canceled Context Lease rows = %d", count)
	}
}

func TestPostgresEventSemanticForcedCandidateFailureRollsBackPublication(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	eventsemanticfixture.SeedScenario(t, db, true)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := eventbiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	const executionID = "semantic-forced-rollback"
	lease, err := useCase.CreateContextLease(context.Background(), eventbiz.ContextLeaseRequest{
		EventID: eventsemanticfixture.EventID, AgentExecutionID: executionID,
		WorkerID: "rollback-fixture", Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		CREATE FUNCTION reject_semantic_variable_signal() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
		  RAISE EXCEPTION 'forced variable signal failure';
		END
		$$;
		CREATE TRIGGER reject_semantic_variable_signal
		BEFORE INSERT ON variable_signals
		FOR EACH ROW EXECUTE FUNCTION reject_semantic_variable_signal()
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := useCase.CreateSubmission(context.Background(), eventsemanticfixture.Submission(
		lease.ID, executionID, "",
	)); err == nil {
		t.Fatal("forced candidate failure unexpectedly committed")
	}
	var submissions, snapshots, links int
	if err := db.QueryRow(`SELECT count(*) FROM event_semantic_submissions WHERE agent_execution_id = $1`, executionID).Scan(&submissions); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM event_semantic_candidate_snapshots`).Scan(&snapshots); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM event_entity_links WHERE semantic_submission_id IS NOT NULL`).Scan(&links); err != nil {
		t.Fatal(err)
	}
	var leaseStatus string
	if err := db.QueryRow(`SELECT status FROM event_semantic_context_leases WHERE id = $1`, lease.ID).Scan(&leaseStatus); err != nil {
		t.Fatal(err)
	}
	if submissions != 0 || snapshots != 0 || links != 0 || leaseStatus != "active" {
		t.Fatalf("rollback state submissions=%d snapshots=%d links=%d lease=%q", submissions, snapshots, links, leaseStatus)
	}
}
