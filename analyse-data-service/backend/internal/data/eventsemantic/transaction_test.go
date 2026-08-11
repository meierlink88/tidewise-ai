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
	submission, err := useCase.CreateSubmission(context.Background(), eventsemanticfixture.Submission(
		lease.ID, executionID, "",
	))
	if err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	replayErr := make(chan error, 1)
	reviewErr := make(chan error, 1)
	go func() {
		<-start
		_, found, err := store.ReplaySubmission(context.Background(), executionID, submission.CanonicalPayloadHash)
		if err == nil && !found {
			err = errors.New("concurrent replay did not find Submission")
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

func TestPostgresEventSemanticTransactionCancellationRollsBack(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	eventsemanticfixture.SeedScenario(t, db, true)
	store, err := NewStore(db)
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
	_, err = store.CreateContextLease(ctx, eventbiz.ContextLeaseRequest{
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
