package eventsemantic

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	eventbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventsemantic"
)

func TestEventSemanticEligibilityAllowsUnknownOccurredAt(t *testing.T) {
	if strings.Contains(strings.ToLower(eventSemanticInputEligibilitySQL), "event_time is not null") {
		t.Fatal("Event Semantic eligibility still excludes Events with unknown occurred_at")
	}
}

func TestEventSemanticRoutePartitionsApplyStableDatabaseBudget(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectQuery("(?s)SELECT profile.entity_id::text, entity.name.*LIMIT \\$1").
		WithArgs(eventSemanticsRoutePartitionLimit).
		WillReturnRows(sqlmock.NewRows([]string{"entity_id", "name"}).
			AddRow("11111111-1111-4111-8111-111111111111", "Industry"))
	if partitions, _, err := repository.eventSemanticIndustryPartitions(context.Background()); err != nil || len(partitions) != 1 {
		t.Fatalf("industry partitions = %v err = %v", partitions, err)
	}

	mock.ExpectQuery("(?s)SELECT DISTINCT concept_type.*LIMIT \\$1").
		WithArgs(eventSemanticsRoutePartitionLimit).
		WillReturnRows(sqlmock.NewRows([]string{"concept_type"}).AddRow("technology"))
	if partitions, _, err := repository.eventSemanticConceptPartitions(context.Background()); err != nil || len(partitions) != 1 {
		t.Fatalf("concept partitions = %v err = %v", partitions, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEventSemanticManifestFingerprintCoversLeaseExpiry(t *testing.T) {
	manifest := eventbiz.ContextManifest{
		ContextLeaseID: "lease-1", AgentExecutionID: "execution-1", WorkerID: "worker-1",
		LeaseStatus: "active", LeaseExpiresAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		ManifestContractVersion: eventSemanticsManifestVersion,
	}
	fingerprint, err := eventSemanticManifestFingerprint(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifest.ManifestFingerprint = fingerprint
	manifest.LeaseExpiresAt = manifest.LeaseExpiresAt.Add(time.Minute)

	_, err = eventSemanticContextFromManifest(context.Background(), nil, manifest)
	var drift *eventbiz.ContextDriftError
	if !errors.As(err, &drift) {
		t.Fatalf("mutated expiry error = %T %v", err, err)
	}
}

func TestSemanticReviewIdentityAndEvidenceAreBoundToFrozenRun(t *testing.T) {
	identity := semanticReviewIdentity{
		AgentExecutionID:   "execution",
		ReviewerPromptHash: "reviewer-hash", ReviewerModel: "reviewer-model",
		AdjudicatorPromptHash: "adjudicator-hash", AdjudicatorModel: "adjudicator-model",
	}
	if !identity.matches(eventbiz.ReviewSubmission{
		ReviewerExecutionKey: "execution:reviewer",
		PromptHash:           "reviewer-hash", Model: "reviewer-model",
	}) {
		t.Fatal("expected frozen reviewer identity to match")
	}
	if identity.matches(eventbiz.ReviewSubmission{
		ReviewerExecutionKey: "other-execution:reviewer",
		PromptHash:           "reviewer-hash", Model: "reviewer-model",
	}) {
		t.Fatal("review lineage from another execution was accepted")
	}
	if identity.matches(eventbiz.ReviewSubmission{
		ReviewerExecutionKey: "execution:reviewer",
		PromptHash:           "adjudicator-hash", Model: "reviewer-model",
	}) {
		t.Fatal("mismatched prompt hash was accepted")
	}
	if !reviewEvidenceMatchesCandidate([]string{"evidence-1"}, []string{"evidence-1", "evidence-2"}) {
		t.Fatal("candidate Evidence citation should match")
	}
	if reviewEvidenceMatchesCandidate([]string{"invented"}, []string{"evidence-1"}) {
		t.Fatal("invented review Evidence was accepted")
	}
}

func TestNoSemanticCandidatesProduceARealRejectedSubmissionOutcome(t *testing.T) {
	status := summarizeSemanticSubmission(eventbiz.PrecheckResult{})
	if status != eventbiz.StatusRejected {
		t.Fatalf("status = %q, want rejected", status)
	}
}
